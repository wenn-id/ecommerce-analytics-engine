package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/metrics"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

// ErrSyncInProgress is returned when a sync is triggered while another one is
// still running (scheduler overlap or duplicate manual POST /sync). The HTTP
// layer maps it to 409 Conflict (#58).
var ErrSyncInProgress = errors.New("sync already in progress")

type SyncService interface {
	SyncAll(ctx context.Context, start, end time.Time) error
	SyncChannel(ctx context.Context, code string, start, end time.Time) error
}

type syncService struct {
	repo       store.Repository
	connectors map[string]connector.PlatformConnector
	// inFlight guards against concurrent SyncAll runs (#58).
	inFlight atomic.Bool
}

func NewSyncService(repo store.Repository, conns []connector.PlatformConnector) SyncService {
	connMap := make(map[string]connector.PlatformConnector)
	for _, c := range conns {
		connMap[c.GetChannelCode()] = c
	}
	return &syncService{
		repo:       repo,
		connectors: connMap,
	}
}

func (s *syncService) SyncAll(ctx context.Context, start, end time.Time) error {
	if !s.inFlight.CompareAndSwap(false, true) {
		slog.Warn("rejecting sync trigger: another sync is still in progress")
		return ErrSyncInProgress
	}
	defer s.inFlight.Store(false)

	var wg sync.WaitGroup
	errChan := make(chan error, len(s.connectors))

	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)

	for _, conn := range s.connectors {
		wg.Add(1)
		sem <- struct{}{}
		go func(c connector.PlatformConnector) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.syncSingle(ctx, c, start, end); err != nil {
				slog.Error("sync failed for channel", "channel", c.GetChannelCode(), "error", err)
				errChan <- err
			}
		}(conn)
	}

	wg.Wait()
	close(errChan)

	var errList []string
	for err := range errChan {
		errList = append(errList, err.Error())
	}

	if len(errList) > 0 {
		return fmt.Errorf("sync failed on %d channels: %s", len(errList), strings.Join(errList, "; "))
	}
	return nil
}

func (s *syncService) SyncChannel(ctx context.Context, code string, start, end time.Time) error {
	conn, ok := s.connectors[code]
	if !ok {
		return fmt.Errorf("unknown channel code: %q", code)
	}
	return s.syncSingle(ctx, conn, start, end)
}

func (s *syncService) syncSingle(ctx context.Context, conn connector.PlatformConnector, start, end time.Time) error {
	now := time.Now()
	channelCode := conn.GetChannelCode()

	chID, err := s.repo.UpsertChannel(ctx, model.Channel{
		Code:         channelCode,
		Name:         conn.GetChannelName(),
		Status:       "active",
		LastSyncedAt: &now,
	})
	if err != nil {
		slog.Error("error upserting channel", "channel", channelCode, "error", err)
		return fmt.Errorf("failed to upsert channel %s: %w", channelCode, err)
	}

	var syncErrors []string
	recordsCount := 0

	// 1. Fetch & Upsert Campaigns
	campaigns, err := conn.FetchCampaigns(ctx)
	if err != nil {
		slog.Error("error fetching campaigns", "channel", channelCode, "error", err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch campaigns: %v", err))
		campaigns = nil
	} else {
		for i := range campaigns {
			campaigns[i].ChannelID = chID
		}
		if err := s.repo.UpsertCampaigns(ctx, campaigns); err != nil {
			slog.Error("error upserting campaigns", "channel", channelCode, "error", err)
			syncErrors = append(syncErrors, fmt.Sprintf("upsert campaigns: %v", err))
		}
	}

	// Resolve campaign IDs by external_id so ad metrics are attributed to the
	// correct campaign regardless of ordering, additions or deletions (#59).
	// This reads the full channel campaign set; no Limit ceiling is applied.
	campaignIDByExternalID, err := s.repo.GetCampaignIDByExternalID(ctx, chID)
	if err != nil {
		slog.Error("error resolving campaign IDs", "channel", channelCode, "error", err)
		syncErrors = append(syncErrors, fmt.Sprintf("resolve campaign ids: %v", err))
	}

	// 2. Fetch & Upsert Ad Metrics
	adMetrics, err := conn.FetchDailyAdMetrics(ctx, start, end)
	if err != nil {
		slog.Error("error fetching ad metrics", "channel", channelCode, "error", err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch ad metrics: %v", err))
	} else if len(campaignIDByExternalID) > 0 {
		dbMetrics := make([]model.DailyAdMetric, 0, len(adMetrics))
		unmapped := 0
		for _, m := range adMetrics {
			dbID, ok := campaignIDByExternalID[m.CampaignExternalID]
			if !ok {
				// The connector reported a campaign we did not fetch/upsert in
				// this run; drop the metric rather than misattribute it (#59).
				unmapped++
				continue
			}
			dbMetrics = append(dbMetrics, model.DailyAdMetric{
				CampaignID:        dbID,
				Date:              m.Date,
				Impressions:       int(m.Impressions),
				Clicks:            int(m.Clicks),
				Spend:             m.Spend,
				Conversions:       int(m.Conversions),
				AttributedRevenue: m.AttributedRevenue,
			})
		}
		if unmapped > 0 {
			slog.Warn("dropped ad metrics with unknown campaign external_id",
				"channel", channelCode, "dropped", unmapped)
		}
		if err := s.repo.UpsertAdMetrics(ctx, dbMetrics); err != nil {
			slog.Error("error upserting ad metrics", "channel", channelCode, "error", err)
			syncErrors = append(syncErrors, fmt.Sprintf("upsert ad metrics: %v", err))
		} else {
			recordsCount += len(dbMetrics)
		}
	}

	// 3. Fetch & Upsert Sales Metrics
	sales, err := conn.FetchDailySales(ctx, start, end)
	if err != nil {
		slog.Error("error fetching sales metrics", "channel", channelCode, "error", err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch sales: %v", err))
	} else {
		for i := range sales {
			sales[i].ChannelID = chID
		}
		if err := s.repo.UpsertSalesMetrics(ctx, sales); err != nil {
			slog.Error("error upserting sales metrics", "channel", channelCode, "error", err)
			syncErrors = append(syncErrors, fmt.Sprintf("upsert sales: %v", err))
		} else {
			recordsCount += len(sales)
		}
	}

	// 4. Record Sync Log
	syncStatus := "SUCCESS"
	var errorMsg string
	if len(syncErrors) > 0 {
		errorMsg = strings.Join(syncErrors, "; ")
		if recordsCount > 0 {
			syncStatus = "PARTIAL_FAILURE"
		} else {
			syncStatus = "FAILED"
		}
	}
	metrics.Default.RecordSyncRun(channelCode, syncStatus)

	if err := s.repo.InsertSyncLog(ctx, model.SyncLog{
		ChannelID:        chID,
		SyncedAt:         now,
		Status:           syncStatus,
		RecordsProcessed: recordsCount,
		ErrorMessage:     errorMsg,
	}); err != nil {
		slog.Error("error inserting sync log", "channel", channelCode, "error", err)
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("channel %s sync encountered errors: %s", channelCode, errorMsg)
	}

	return nil
}

package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

type SyncService interface {
	SyncAll(ctx context.Context, start, end time.Time) error
	SyncChannel(ctx context.Context, code string, start, end time.Time) error
}

type syncService struct {
	repo       store.Repository
	connectors map[string]connector.PlatformConnector
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
				log.Printf("Sync failed for channel %s: %v", c.GetChannelCode(), err)
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
		log.Printf("Error upserting channel %s: %v", channelCode, err)
		return fmt.Errorf("failed to upsert channel %s: %w", channelCode, err)
	}

	var syncErrors []string
	recordsCount := 0

	// 1. Fetch & Upsert Campaigns
	campaigns, err := conn.FetchCampaigns(ctx)
	if err != nil {
		log.Printf("Error fetching campaigns for %s: %v", channelCode, err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch campaigns: %v", err))
	} else {
		for i := range campaigns {
			campaigns[i].ChannelID = chID
		}
		if err := s.repo.UpsertCampaigns(ctx, campaigns); err != nil {
			log.Printf("Error upserting campaigns for %s: %v", channelCode, err)
			syncErrors = append(syncErrors, fmt.Sprintf("upsert campaigns: %v", err))
		}
	}

	// Retrieve campaign mapping for this channel to correctly link ad metrics
	dbCampaigns, _, err := s.repo.GetCampaigns(ctx, model.CampaignFilter{ChannelID: chID, Limit: 100})
	campaignIDs := make([]int64, 0, len(dbCampaigns))
	if err == nil {
		for _, c := range dbCampaigns {
			campaignIDs = append(campaignIDs, c.ID)
		}
	}

	// 2. Fetch & Upsert Ad Metrics
	adMetrics, err := conn.FetchDailyAdMetrics(ctx, start, end)
	if err != nil {
		log.Printf("Error fetching ad metrics for %s: %v", channelCode, err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch ad metrics: %v", err))
	} else {
		if len(campaignIDs) > 0 {
			for i := range adMetrics {
				rawID := int(adMetrics[i].CampaignID)
				if rawID > 0 && rawID <= len(campaignIDs) {
					adMetrics[i].CampaignID = campaignIDs[rawID-1]
				} else {
					// Distribute across channel's available campaigns
					mappedIdx := (rawID - 1) % len(campaignIDs)
					if mappedIdx < 0 {
						mappedIdx = 0
					}
					adMetrics[i].CampaignID = campaignIDs[mappedIdx]
				}
			}
		}
		if err := s.repo.UpsertAdMetrics(ctx, adMetrics); err != nil {
			log.Printf("Error upserting ad metrics for %s: %v", channelCode, err)
			syncErrors = append(syncErrors, fmt.Sprintf("upsert ad metrics: %v", err))
		} else {
			recordsCount += len(adMetrics)
		}
	}

	// 3. Fetch & Upsert Sales Metrics
	sales, err := conn.FetchDailySales(ctx, start, end)
	if err != nil {
		log.Printf("Error fetching sales metrics for %s: %v", channelCode, err)
		syncErrors = append(syncErrors, fmt.Sprintf("fetch sales: %v", err))
	} else {
		for i := range sales {
			sales[i].ChannelID = chID
		}
		if err := s.repo.UpsertSalesMetrics(ctx, sales); err != nil {
			log.Printf("Error upserting sales metrics for %s: %v", channelCode, err)
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

	if err := s.repo.InsertSyncLog(ctx, model.SyncLog{
		ChannelID:        chID,
		SyncedAt:         now,
		Status:           syncStatus,
		RecordsProcessed: recordsCount,
		ErrorMessage:     errorMsg,
	}); err != nil {
		log.Printf("Error inserting sync log for channel %s: %v", channelCode, err)
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("channel %s sync encountered errors: %s", channelCode, errorMsg)
	}

	return nil
}

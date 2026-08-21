package service

import (
	"context"
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

	for _, conn := range s.connectors {
		wg.Add(1)
		go func(c connector.PlatformConnector) {
			defer wg.Done()
			if err := s.syncSingle(ctx, c, start, end); err != nil {
				errChan <- err
			}
		}(conn)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

func (s *syncService) SyncChannel(ctx context.Context, code string, start, end time.Time) error {
	conn, ok := s.connectors[code]
	if !ok {
		return nil
	}
	return s.syncSingle(ctx, conn, start, end)
}

func (s *syncService) syncSingle(ctx context.Context, conn connector.PlatformConnector, start, end time.Time) error {
	now := time.Now()
	chID, err := s.repo.UpsertChannel(ctx, model.Channel{
		Code:         conn.GetChannelCode(),
		Name:         conn.GetChannelName(),
		Status:       "active",
		LastSyncedAt: &now,
	})
	if err != nil {
		return err
	}

	campaigns, err := conn.FetchCampaigns(ctx)
	if err == nil {
		for i := range campaigns {
			campaigns[i].ChannelID = chID
		}
		_ = s.repo.UpsertCampaigns(ctx, campaigns)
	}

	adMetrics, err := conn.FetchDailyAdMetrics(ctx, start, end)
	if err == nil {
		for i := range adMetrics {
			adMetrics[i].CampaignID = chID
		}
		_ = s.repo.UpsertAdMetrics(ctx, adMetrics)
	}

	sales, err := conn.FetchDailySales(ctx, start, end)
	if err == nil {
		for i := range sales {
			sales[i].ChannelID = chID
		}
		_ = s.repo.UpsertSalesMetrics(ctx, sales)
	}

	_ = s.repo.InsertSyncLog(ctx, model.SyncLog{
		ChannelID:        chID,
		SyncedAt:         now,
		Status:           "SUCCESS",
		RecordsProcessed: len(adMetrics) + len(sales),
	})

	return nil
}

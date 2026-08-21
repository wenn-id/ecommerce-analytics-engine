package handler_test

import (
	"context"
	"time"

	"ecommerce-analytics/internal/model"
)

type mockAnalyticsService struct {
	overviewFn  func(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error)
	trendFn     func(ctx context.Context, start, end time.Time) ([]model.TrendDataPoint, error)
	breakdownFn func(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error)
}

func (m *mockAnalyticsService) GetOverviewMetrics(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error) {
	if m.overviewFn != nil {
		return m.overviewFn(ctx, start, end)
	}
	return &model.OverviewMetrics{TotalSpend: 1000, TotalGMV: 5000, BlendedROAS: 5.0}, nil
}

func (m *mockAnalyticsService) GetTrendData(ctx context.Context, start, end time.Time) ([]model.TrendDataPoint, error) {
	if m.trendFn != nil {
		return m.trendFn(ctx, start, end)
	}
	return []model.TrendDataPoint{{Date: "2026-08-20", Spend: 100, GMV: 500}}, nil
}

func (m *mockAnalyticsService) GetChannelBreakdown(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error) {
	if m.breakdownFn != nil {
		return m.breakdownFn(ctx, start, end)
	}
	return []model.ChannelSummary{{ChannelCode: "meta_ads", TotalSpend: 100, TotalGMV: 500}}, nil
}

type mockSyncService struct {
	syncAllFn     func(ctx context.Context, start, end time.Time) error
	syncChannelFn func(ctx context.Context, channelCode string, start, end time.Time) error
}

func (m *mockSyncService) SyncAll(ctx context.Context, start, end time.Time) error {
	if m.syncAllFn != nil {
		return m.syncAllFn(ctx, start, end)
	}
	return nil
}

func (m *mockSyncService) SyncChannel(ctx context.Context, channelCode string, start, end time.Time) error {
	if m.syncChannelFn != nil {
		return m.syncChannelFn(ctx, channelCode, start, end)
	}
	return nil
}

package connector

import (
	"context"
	"time"

	"ecommerce-analytics/internal/model"
)

// AdMetric is a daily ad performance record keyed by the platform's own
// campaign identifier. Connectors must return CampaignExternalID exactly as
// reported by FetchCampaigns so the sync service can resolve the internal
// campaign ID by external_id — never by list position (#59).
type AdMetric struct {
	CampaignExternalID string
	Date               time.Time
	Impressions        int64
	Clicks             int64
	Spend              float64
	Conversions        int64
	AttributedRevenue  float64
}

type PlatformConnector interface {
	GetChannelCode() string
	GetChannelName() string
	FetchCampaigns(ctx context.Context) ([]model.Campaign, error)
	FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error)
	FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error)
}

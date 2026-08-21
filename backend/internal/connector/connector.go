package connector

import (
	"context"
	"time"

	"ecommerce-analytics/internal/model"
)

type PlatformConnector interface {
	GetChannelCode() string
	GetChannelName() string
	FetchCampaigns(ctx context.Context) ([]model.Campaign, error)
	FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error)
	FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error)
}

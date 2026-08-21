package connector

import (
	"context"
	"math/rand/v2"
	"time"

	"ecommerce-analytics/internal/model"
)

type tikTokConnector struct{}

func NewTikTokConnector() PlatformConnector {
	return &tikTokConnector{}
}

func (t *tikTokConnector) GetChannelCode() string { return "tiktok_shop" }
func (t *tikTokConnector) GetChannelName() string { return "TikTok Shop" }

func (t *tikTokConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "tt_c_201", Name: "Product GMV Max - Live Shopping", Status: "ACTIVE", DailyBudget: 1000000.0, CreatedAt: time.Now()},
		{ExternalID: "tt_c_202", Name: "Video Shopping Ads - Top Sellers", Status: "ACTIVE", DailyBudget: 800000.0, CreatedAt: time.Now()},
	}, nil
}

func (t *tikTokConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 900000.0 + float64(rand.IntN(300000))
		clicks := 600 + rand.IntN(200)
		conversions := 35 + rand.IntN(20)
		roas := 4.8 + (rand.Float64() * 1.8)

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        2,
			Date:              curr,
			Impressions:       25000 + rand.IntN(8000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: spend * roas,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (t *tikTokConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 45 + rand.IntN(25)
		gmv := float64(orders) * (150000.0 + float64(rand.IntN(30000)))
		cogs := gmv * 0.42
		netSales := gmv * 0.94

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.IntN(4),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

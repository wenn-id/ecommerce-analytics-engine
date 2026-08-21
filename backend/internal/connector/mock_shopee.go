package connector

import (
	"context"
	"math/rand/v2"
	"time"

	"ecommerce-analytics/internal/model"
)

type shopeeConnector struct{}

func NewShopeeConnector() PlatformConnector {
	return &shopeeConnector{}
}

func (s *shopeeConnector) GetChannelCode() string { return "shopee" }
func (s *shopeeConnector) GetChannelName() string { return "Shopee" }

func (s *shopeeConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "sh_c_301", Name: "Shopee Discovery Ads - Flash Deals", Status: "ACTIVE", DailyBudget: 600000.0, CreatedAt: time.Now()},
	}, nil
}

func (s *shopeeConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 500000.0 + float64(rand.IntN(200000))
		clicks := 300 + rand.IntN(100)
		conversions := 20 + rand.IntN(10)
		roas := 5.2 + (rand.Float64() * 2.0)

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        3,
			Date:              curr,
			Impressions:       12000 + rand.IntN(3000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: spend * roas,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (s *shopeeConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 35 + rand.IntN(15)
		gmv := float64(orders) * (140000.0 + float64(rand.IntN(25000)))
		cogs := gmv * 0.44
		netSales := gmv * 0.93

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.IntN(2),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

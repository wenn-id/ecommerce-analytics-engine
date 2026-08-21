package connector

import (
	"context"
	"math/rand"
	"time"

	"ecommerce-analytics/internal/model"
)

type metaConnector struct{}

func NewMetaConnector() PlatformConnector {
	return &metaConnector{}
}

func (m *metaConnector) GetChannelCode() string { return "meta_ads" }
func (m *metaConnector) GetChannelName() string { return "Meta Ads" }

func (m *metaConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "meta_c_101", Name: "Catalog Sales - Retargeting", Status: "ACTIVE", DailyBudget: 750000.0, CreatedAt: time.Now()},
		{ExternalID: "meta_c_102", Name: "Advantage+ Shopping Campaign", Status: "ACTIVE", DailyBudget: 1500000.0, CreatedAt: time.Now()},
	}, nil
}

func (m *metaConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 1200000.0 + float64(rand.Intn(400000))
		clicks := 400 + rand.Intn(150)
		conversions := 25 + rand.Intn(15)
		roas := 4.2 + (rand.Float64() * 1.5)
		attributedRev := spend * roas

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        1,
			Date:              curr,
			Impressions:       18000 + rand.Intn(5000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: attributedRev,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (m *metaConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 30 + rand.Intn(20)
		gmv := float64(orders) * (180000.0 + float64(rand.Intn(40000)))
		cogs := gmv * 0.45
		netSales := gmv * 0.95

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.Intn(3),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

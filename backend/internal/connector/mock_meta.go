package connector

import (
	"context"
	"math/rand/v2"
	"time"

	"ecommerce-analytics/internal/model"
)

type metaConnector struct{}

func NewMetaConnector() PlatformConnector {
	return &metaConnector{}
}

func (m *metaConnector) GetChannelCode() string { return "meta_ads" }
func (m *metaConnector) GetChannelName() string { return "Meta Ads" }

func (m *metaConnector) campaigns() []model.Campaign {
	return []model.Campaign{
		{ExternalID: "meta_c_101", Name: "Catalog Sales - Retargeting", Status: "ACTIVE", DailyBudget: 750000.0, CreatedAt: time.Now()},
		{ExternalID: "meta_c_102", Name: "Advantage+ Shopping Campaign", Status: "ACTIVE", DailyBudget: 1500000.0, CreatedAt: time.Now()},
	}
}

func (m *metaConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return m.campaigns(), nil
}

func (m *metaConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	campaigns := m.campaigns()
	var metrics []AdMetric
	for _, campaign := range campaigns {
		curr := startDate
		for !curr.After(endDate) {
			daySeed := uint64(curr.Unix() / 86400)
			rng := rand.New(rand.NewPCG(daySeed, 101))

			spend := 1200000.0 + float64(rng.IntN(400000))
			clicks := int64(400 + rng.IntN(150))
			conversions := int64(25 + rng.IntN(15))
			roas := 4.2 + (rng.Float64() * 1.5)

			metrics = append(metrics, AdMetric{
				CampaignExternalID: campaign.ExternalID,
				Date:               curr,
				Impressions:        int64(18000 + rng.IntN(5000)),
				Clicks:             clicks,
				Spend:              spend,
				Conversions:        conversions,
				AttributedRevenue:  spend * roas,
			})
			curr = curr.AddDate(0, 0, 1)
		}
	}
	return metrics, nil
}

func (m *metaConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		daySeed := uint64(curr.Unix() / 86400)
		rng := rand.New(rand.NewPCG(daySeed, 102))

		orders := 30 + rng.IntN(20)
		gmv := float64(orders) * (180000.0 + float64(rng.IntN(40000)))
		cogs := gmv * 0.45
		netSales := gmv * 0.95

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rng.IntN(3),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

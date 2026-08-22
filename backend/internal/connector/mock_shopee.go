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

func (s *shopeeConnector) campaigns() []model.Campaign {
	return []model.Campaign{
		{ExternalID: "sh_c_301", Name: "Shopee Discovery Ads - Flash Deals", Status: "ACTIVE", DailyBudget: 600000.0, CreatedAt: time.Now()},
	}
}

func (s *shopeeConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return s.campaigns(), nil
}

func (s *shopeeConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	campaigns := s.campaigns()
	var metrics []AdMetric
	for _, campaign := range campaigns {
		curr := startDate
		for !curr.After(endDate) {
			daySeed := uint64(curr.Unix() / 86400)
			rng := rand.New(rand.NewPCG(daySeed, 301))

			spend := 500000.0 + float64(rng.IntN(200000))
			clicks := int64(300 + rng.IntN(100))
			conversions := int64(20 + rng.IntN(10))
			roas := 5.2 + (rng.Float64() * 2.0)

			metrics = append(metrics, AdMetric{
				CampaignExternalID: campaign.ExternalID,
				Date:               curr,
				Impressions:        int64(12000 + rng.IntN(3000)),
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

func (s *shopeeConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		daySeed := uint64(curr.Unix() / 86400)
		rng := rand.New(rand.NewPCG(daySeed, 302))

		orders := 35 + rng.IntN(15)
		gmv := float64(orders) * (140000.0 + float64(rng.IntN(25000)))
		cogs := gmv * 0.44
		netSales := gmv * 0.93

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rng.IntN(2),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

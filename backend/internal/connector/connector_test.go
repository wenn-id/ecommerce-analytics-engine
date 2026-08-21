package connector_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
)

func TestConnectors(t *testing.T) {
	ctx := context.Background()
	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	for _, conn := range connectors {
		t.Run(conn.GetChannelCode(), func(t *testing.T) {
			campaigns, err := conn.FetchCampaigns(ctx)
			if err != nil || len(campaigns) == 0 {
				t.Fatalf("expected campaigns from %s, got err: %v", conn.GetChannelCode(), err)
			}

			adMetrics, err := conn.FetchDailyAdMetrics(ctx, startDate, endDate)
			if err != nil || len(adMetrics) == 0 {
				t.Fatalf("expected ad metrics from %s, got err: %v", conn.GetChannelCode(), err)
			}

			sales, err := conn.FetchDailySales(ctx, startDate, endDate)
			if err != nil || len(sales) == 0 {
				t.Fatalf("expected sales metrics from %s, got err: %v", conn.GetChannelCode(), err)
			}
		})
	}
}

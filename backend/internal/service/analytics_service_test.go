package service_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestAnalyticsServiceOverview(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	chID, _ := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	today := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	_ = repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{
		{CampaignID: 1, Date: today, Spend: 1000000.0, Conversions: 20, AttributedRevenue: 5000000.0},
	})
	_ = repo.UpsertSalesMetrics(ctx, []model.DailySalesMetric{
		{ChannelID: chID, Date: today, TotalOrders: 25, GMV: 6000000.0, COGS: 2400000.0},
	})

	analyticsSvc := service.NewAnalyticsService(repo)
	overview, err := analyticsSvc.GetOverviewMetrics(ctx, today, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.TotalSpend != 1000000.0 || overview.TotalGMV != 6000000.0 {
		t.Errorf("incorrect totals: %+v", overview)
	}
	if overview.BlendedROAS != 6.0 {
		t.Errorf("expected Blended ROAS 6.0, got %f", overview.BlendedROAS)
	}
	if overview.AvgCPA != 40000.0 {
		t.Errorf("expected Avg CPA 40000.0, got %f", overview.AvgCPA)
	}
}

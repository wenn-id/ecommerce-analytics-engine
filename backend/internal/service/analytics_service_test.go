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

func TestAnalyticsServiceChannelBreakdown(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	ch1, _ := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	ch2, _ := repo.UpsertChannel(ctx, model.Channel{Code: "tiktok_shop", Name: "TikTok Shop", Status: "active"})

	// Campaign 1 belongs to Meta (ch1), Campaign 2 and 3 belong to TikTok (ch2)
	_ = repo.UpsertCampaigns(ctx, []model.Campaign{
		{ChannelID: ch1, ExternalID: "c1", Name: "Meta Camp 1", Status: "ACTIVE", DailyBudget: 500000, CreatedAt: time.Now()},
		{ChannelID: ch2, ExternalID: "c2", Name: "TikTok Camp 2", Status: "ACTIVE", DailyBudget: 500000, CreatedAt: time.Now()},
		{ChannelID: ch2, ExternalID: "c3", Name: "TikTok Camp 3", Status: "ACTIVE", DailyBudget: 500000, CreatedAt: time.Now()},
	})

	campaigns, _, _ := repo.GetCampaigns(ctx, model.CampaignFilter{Limit: 10})
	cMap := make(map[string]int64)
	for _, c := range campaigns {
		cMap[c.ExternalID] = c.ID
	}

	today := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	// Ad metric on Campaign 3 (ID 3, belonging to TikTok ch2)
	_ = repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{
		{CampaignID: cMap["c1"], Date: today, Spend: 1000000.0, Conversions: 10, AttributedRevenue: 4000000.0},
		{CampaignID: cMap["c3"], Date: today, Spend: 2000000.0, Conversions: 20, AttributedRevenue: 10000000.0},
	})

	_ = repo.UpsertSalesMetrics(ctx, []model.DailySalesMetric{
		{ChannelID: ch1, Date: today, TotalOrders: 10, GMV: 4000000.0, COGS: 1600000.0},
		{ChannelID: ch2, Date: today, TotalOrders: 30, GMV: 12000000.0, COGS: 4800000.0},
	})

	analyticsSvc := service.NewAnalyticsService(repo)
	breakdowns, err := analyticsSvc.GetChannelBreakdown(ctx, today, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(breakdowns) != 2 {
		t.Fatalf("expected 2 channel summaries, got %d", len(breakdowns))
	}

	for _, b := range breakdowns {
		if b.ChannelCode == "meta_ads" {
			if b.TotalSpend != 1000000.0 || b.TotalGMV != 4000000.0 || b.ChannelROAS != 4.0 {
				t.Errorf("incorrect meta_ads metrics: %+v", b)
			}
		}
		if b.ChannelCode == "tiktok_shop" {
			if b.TotalSpend != 2000000.0 || b.TotalGMV != 12000000.0 || b.ChannelROAS != 6.0 {
				t.Errorf("incorrect tiktok_shop metrics: %+v", b)
			}
		}
	}
}

func TestAnalyticsServiceTrendData(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	chID, _ := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	d1 := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	_ = repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{
		{CampaignID: 1, Date: d1, Spend: 500000.0, Conversions: 10, AttributedRevenue: 2000000.0},
		{CampaignID: 1, Date: d2, Spend: 1000000.0, Conversions: 20, AttributedRevenue: 5000000.0},
	})
	_ = repo.UpsertSalesMetrics(ctx, []model.DailySalesMetric{
		{ChannelID: chID, Date: d1, TotalOrders: 15, GMV: 3000000.0, COGS: 1200000.0},
		{ChannelID: chID, Date: d2, TotalOrders: 25, GMV: 6000000.0, COGS: 2400000.0},
	})

	analyticsSvc := service.NewAnalyticsService(repo)
	trends, err := analyticsSvc.GetTrendData(ctx, d1, d2)
	if err != nil {
		t.Fatalf("unexpected error fetching trend data: %v", err)
	}

	if len(trends) != 2 {
		t.Fatalf("expected 2 trend points, got %d", len(trends))
	}

	if trends[0].Spend != 500000.0 || trends[0].GMV != 3000000.0 || trends[0].BlendedROAS != 6.0 {
		t.Errorf("incorrect trend 0: %+v", trends[0])
	}
	if trends[1].Spend != 1000000.0 || trends[1].GMV != 6000000.0 || trends[1].BlendedROAS != 6.0 {
		t.Errorf("incorrect trend 1: %+v", trends[1])
	}
}

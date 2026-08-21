package store_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

func TestRepositoryInitAndUpsert(t *testing.T) {
	ctx := context.Background()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open test memory db: %v", err)
	}
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// 1. Insert Channel
	chID, err := repo.UpsertChannel(ctx, model.Channel{
		Code:   "meta_ads",
		Name:   "Meta Ads",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("failed to upsert channel: %v", err)
	}

	// 2. Insert Campaigns
	campaign := model.Campaign{
		ChannelID:   chID,
		ExternalID:  "meta_c_001",
		Name:        "Summer Promo",
		Status:      "ACTIVE",
		DailyBudget: 500000.0,
		CreatedAt:   time.Now(),
	}
	if err := repo.UpsertCampaigns(ctx, []model.Campaign{campaign}); err != nil {
		t.Fatalf("failed to upsert campaign: %v", err)
	}

	// 3. Insert and verify Ad Metrics
	today := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	adMetric := model.DailyAdMetric{
		CampaignID:        1,
		Date:              today,
		Impressions:       5000,
		Clicks:            250,
		Spend:             100000.0,
		Conversions:       10,
		AttributedRevenue: 500000.0,
	}
	if err := repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{adMetric}); err != nil {
		t.Fatalf("failed to upsert ad metric: %v", err)
	}

	metrics, err := repo.QueryAdMetrics(ctx, today, today)
	if err != nil || len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d (err: %v)", len(metrics), err)
	}
	if metrics[0].Spend != 100000.0 {
		t.Errorf("expected spend 100000, got %f", metrics[0].Spend)
	}
}

package store_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

func TestMigrationsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("first InitSchema: %v", err)
	}
	// Re-running must be a no-op, and repeated startup is the normal path.
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("second InitSchema should be a no-op: %v", err)
	}

	// Writing through the migrated schema proves the tables exist and work.
	chID, err := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	if err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if err := repo.UpsertCampaigns(ctx, []model.Campaign{{
		ChannelID: chID, ExternalID: "ext-1", Name: "Campaign One", Status: "ACTIVE", DailyBudget: 10, CreatedAt: time.Now(),
	}}); err != nil {
		t.Fatalf("upsert campaign: %v", err)
	}
}

func TestGetCampaignIDByExternalID(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	chID, err := repo.UpsertChannel(ctx, model.Channel{Code: "shopee", Name: "Shopee", Status: "active"})
	if err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if err := repo.UpsertCampaigns(ctx, []model.Campaign{
		{ChannelID: chID, ExternalID: "sh-1", Name: "First", Status: "ACTIVE", DailyBudget: 1, CreatedAt: time.Now()},
		{ChannelID: chID, ExternalID: "sh-2", Name: "Second", Status: "ACTIVE", DailyBudget: 2, CreatedAt: time.Now()},
	}); err != nil {
		t.Fatalf("upsert campaigns: %v", err)
	}

	mapping, err := repo.GetCampaignIDByExternalID(ctx, chID)
	if err != nil {
		t.Fatalf("get mapping: %v", err)
	}
	if len(mapping) != 2 {
		t.Fatalf("expected 2 mapped campaigns, got %d", len(mapping))
	}
	if mapping["sh-1"] == 0 || mapping["sh-2"] == 0 {
		t.Errorf("expected non-zero db ids, got %v", mapping)
	}
}

func TestCampaignSearchEscapesLikeWildcards(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	chID, _ := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	if err := repo.UpsertCampaigns(ctx, []model.Campaign{
		{ChannelID: chID, ExternalID: "e-1", Name: "50% OFF SALE", Status: "ACTIVE", DailyBudget: 1, CreatedAt: time.Now()},
		{ChannelID: chID, ExternalID: "e-2", Name: "Discount_50_extra", Status: "ACTIVE", DailyBudget: 1, CreatedAt: time.Now()},
		{ChannelID: chID, ExternalID: "e-3", Name: "DiscountA5_wildcard_trap", Status: "ACTIVE", DailyBudget: 1, CreatedAt: time.Now()},
	}); err != nil {
		t.Fatalf("upsert campaigns: %v", err)
	}

	// "%" must match literally: only the first campaign contains "50%".
	rows, total, err := repo.GetCampaigns(ctx, model.CampaignFilter{Search: "50%", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Name != "50% OFF SALE" {
		t.Fatalf("expected only the percent-sale campaign for a literal percent search, got total=%d rows=%+v", total, rows)
	}

	// "_" must match literally: "Discount_5" matches the literal
	// "Discount_50_extra" (prefix) but NOT "DiscountA5_wildcard_trap", which
	// only matches when "_" acts as a single-character wildcard.
	rows, total, err = repo.GetCampaigns(ctx, model.CampaignFilter{Search: "Discount_5", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Name != "Discount_50_extra" {
		t.Fatalf("expected only the literal-underscore campaign, got total=%d rows=%+v", total, rows)
	}
}

func TestPing(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.Ping(ctx); err != nil {
		t.Fatalf("expected ping to succeed on open db, got %v", err)
	}
}

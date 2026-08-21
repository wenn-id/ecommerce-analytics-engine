package service_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestSyncServiceAll(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	syncSvc := service.NewSyncService(repo, connectors)
	start := time.Now().AddDate(0, 0, -3)
	end := time.Now()

	if err := syncSvc.SyncAll(ctx, start, end); err != nil {
		t.Fatalf("sync all failed: %v", err)
	}

	channels, _ := repo.GetChannels(ctx)
	if len(channels) != 3 {
		t.Errorf("expected 3 channels registered, got %d", len(channels))
	}
}

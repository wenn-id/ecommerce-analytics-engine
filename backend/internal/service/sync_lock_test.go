package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

// blockingConnector stalls FetchCampaigns until released, letting tests hold
// a sync in flight.
type blockingConnector struct {
	release chan struct{}
}

func (b *blockingConnector) GetChannelCode() string { return "meta_ads" }
func (b *blockingConnector) GetChannelName() string { return "Meta Ads" }

func (b *blockingConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	<-b.release
	return nil, nil
}

func (b *blockingConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]connector.AdMetric, error) {
	return nil, nil
}

func (b *blockingConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	return nil, nil
}

func TestSyncAllRejectsConcurrentRun(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	blocker := &blockingConnector{release: make(chan struct{})}
	syncSvc := service.NewSyncService(repo, []connector.PlatformConnector{blocker})

	done := make(chan error, 1)
	go func() {
		done <- syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -1), time.Now())
	}()

	// Give the first sync a moment to enter flight, then assert the second is
	// rejected instead of running concurrently (#58).
	time.Sleep(50 * time.Millisecond)
	err := syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -1), time.Now())
	if !errors.Is(err, service.ErrSyncInProgress) {
		t.Fatalf("expected ErrSyncInProgress, got %v", err)
	}

	close(blocker.release)
	if err := <-done; err != nil {
		t.Fatalf("first sync should complete after release, got %v", err)
	}

	// After completion the lock must be released again.
	if err := syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -1), time.Now()); err != nil {
		t.Fatalf("sync after completion should be accepted, got %v", err)
	}
}

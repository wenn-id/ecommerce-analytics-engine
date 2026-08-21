package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"ecommerce-analytics/internal/scheduler"
)

type mockSyncService struct {
	syncCalls int64
}

func (m *mockSyncService) SyncAll(ctx context.Context, start, end time.Time) error {
	atomic.AddInt64(&m.syncCalls, 1)
	return nil
}

func (m *mockSyncService) SyncChannel(ctx context.Context, code string, start, end time.Time) error {
	atomic.AddInt64(&m.syncCalls, 1)
	return nil
}

func TestSchedulerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockSvc := &mockSyncService{}

	// Start scheduler with 1 minute interval
	scheduler.Start(ctx, mockSvc, 1)

	// Cancel context to trigger shutdown of scheduler
	cancel()

	// Give a small pause to ensure goroutine exits cleanly
	time.Sleep(50 * time.Millisecond)
}

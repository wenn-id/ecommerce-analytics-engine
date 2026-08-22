package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ecommerce-analytics/internal/service"
)

// Start runs SyncAll on a fixed interval until ctx is cancelled. A sync that
// overlaps a still-running one is skipped by the sync service itself
// (service.ErrSyncInProgress) rather than queued (#58).
func Start(ctx context.Context, syncSvc service.SyncService, intervalMinutes int) {
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				end := time.Now()
				start := end.AddDate(0, 0, -30)
				if err := syncSvc.SyncAll(ctx, start, end); err != nil {
					if errors.Is(err, service.ErrSyncInProgress) {
						slog.Debug("scheduled sync skipped: previous run still in progress")
						continue
					}
					slog.Error("scheduled sync failed", "error", err)
				}
			}
		}
	}()
}

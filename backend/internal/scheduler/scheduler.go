package scheduler

import (
	"context"
	"time"

	"ecommerce-analytics/internal/service"
)

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
				_ = syncSvc.SyncAll(ctx, start, end)
			}
		}
	}()
}

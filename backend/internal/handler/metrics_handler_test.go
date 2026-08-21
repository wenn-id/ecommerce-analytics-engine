package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestMetricsHandlerOverview(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)
	conns := []connector.PlatformConnector{connector.NewMetaConnector()}
	syncSvc := service.NewSyncService(repo, conns)
	_ = syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -5), time.Now())

	analyticsSvc := service.NewAnalyticsService(repo)
	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)

	req := httptest.NewRequest("GET", "/api/v1/metrics/overview?start_date=2026-08-01&end_date=2026-08-21", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

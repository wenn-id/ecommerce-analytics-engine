package handler_test

import (
	"context"
	"encoding/json"
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

func TestMetricsHandlerCampaignsPagination(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/v1/campaigns?page=1&limit=2", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data       []interface{} `json:"data"`
		Pagination struct {
			CurrentPage  int `json:"current_page"`
			Limit        int `json:"limit"`
			TotalRecords int `json:"total_records"`
			TotalPages   int `json:"total_pages"`
		} `json:"pagination"`
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Pagination.CurrentPage != 1 {
		t.Errorf("expected current_page 1, got %d", resp.Pagination.CurrentPage)
	}
	if resp.Pagination.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Pagination.Limit)
	}
	if resp.Pagination.TotalRecords == 0 {
		t.Errorf("expected total_records > 0, got %d", resp.Pagination.TotalRecords)
	}
}

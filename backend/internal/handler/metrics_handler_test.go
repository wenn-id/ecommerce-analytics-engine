package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/model"
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
	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, nil)

	req := httptest.NewRequest("GET", "/api/v1/metrics/overview?start_date=2026-08-01&end_date=2026-08-21", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected CORS origin http://localhost:3000, got %s", w.Header().Get("Access-Control-Allow-Origin"))
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
	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, nil)

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

func TestMetricsHandlerAuth(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)
	conns := []connector.PlatformConnector{connector.NewMetaConnector()}
	syncSvc := service.NewSyncService(repo, conns)
	analyticsSvc := service.NewAnalyticsService(repo)

	cfg := &config.Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		APIKey:         "secret-key-123",
	}
	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, cfg)

	// Health endpoint should be public
	reqHealth := httptest.NewRequest("GET", "/api/v1/health", nil)
	wHealth := httptest.NewRecorder()
	h.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Errorf("expected health endpoint to be 200, got %d", wHealth.Code)
	}

	// Request without API key should be 401
	reqUnauth := httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	wUnauth := httptest.NewRecorder()
	h.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", wUnauth.Code)
	}

	// Request with valid X-API-Key should succeed
	reqAuth := httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	reqAuth.Header.Set("X-API-Key", "secret-key-123")
	wAuth := httptest.NewRecorder()
	h.ServeHTTP(wAuth, reqAuth)
	if wAuth.Code != http.StatusOK {
		t.Errorf("expected 200 OK with valid API key, got %d", wAuth.Code)
	}
}

func TestMetricsHandlerRateLimiting(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)
	conns := []connector.PlatformConnector{connector.NewMetaConnector()}
	syncSvc := service.NewSyncService(repo, conns)
	analyticsSvc := service.NewAnalyticsService(repo)

	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, nil)

	// Exhaust tokens with rate limiter set to 2 capacity
	limiter := handler.NewRateLimiter(1, 2)
	// Temporarily test limiter
	ip := "192.168.1.100"
	if !limiter.Allow(ip) {
		t.Errorf("first request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Errorf("second request should be allowed")
	}
	if limiter.Allow(ip) {
		t.Errorf("third request exceeding capacity should be blocked")
	}

	// Verify 429 response through handler
	received429 := false
	for i := 0; i < 110; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			received429 = true
			break
		}
	}
	if !received429 {
		t.Fatalf("expected HTTP 429 Too Many Requests from rate limiter after exceeding capacity")
	}
}

func TestMetricsHandlerCSRF(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)
	conns := []connector.PlatformConnector{connector.NewMetaConnector()}
	syncSvc := service.NewSyncService(repo, conns)
	analyticsSvc := service.NewAnalyticsService(repo)

	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, nil)

	// POST /sync without custom header should fail with 403 Forbidden
	reqNoHeader := httptest.NewRequest("POST", "/api/v1/sync", nil)
	wNoHeader := httptest.NewRecorder()
	h.ServeHTTP(wNoHeader, reqNoHeader)
	if wNoHeader.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden without CSRF header, got %d", wNoHeader.Code)
	}

	// POST /sync with X-Requested-With should succeed with 200 OK
	reqWithHeader := httptest.NewRequest("POST", "/api/v1/sync", nil)
	reqWithHeader.Header.Set("X-Requested-With", "XMLHttpRequest")
	wWithHeader := httptest.NewRecorder()
	h.ServeHTTP(wWithHeader, reqWithHeader)
	if wWithHeader.Code != http.StatusOK {
		t.Errorf("expected 200 OK with X-Requested-With header, got %d", wWithHeader.Code)
	}
}

func TestMetricsHandlerWithMocks(t *testing.T) {
	mockAnalytics := &mockAnalyticsService{
		overviewFn: func(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error) {
			return &model.OverviewMetrics{
				TotalSpend:  1234567,
				TotalGMV:    9876543,
				BlendedROAS: 8.0,
			}, nil
		},
	}
	mockSync := &mockSyncService{}
	h := handler.NewMetricsHandler(mockAnalytics, mockSync, nil, nil)

	req := httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with mock service, got %d", w.Code)
	}

	var res model.OverviewMetrics
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if res.BlendedROAS != 8.0 || res.TotalSpend != 1234567 {
		t.Errorf("unexpected mock data: %+v", res)
	}
}

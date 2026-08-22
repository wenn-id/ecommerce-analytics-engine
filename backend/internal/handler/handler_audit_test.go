package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func newTestHandler(t *testing.T, cfg *config.Config) (*handler.MetricsHandler, service.SyncService) {
	t.Helper()
	ctx := context.Background()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	syncSvc := service.NewSyncService(repo, []connector.PlatformConnector{connector.NewMetaConnector()})
	analyticsSvc := service.NewAnalyticsService(repo)
	return handler.NewMetricsHandler(analyticsSvc, syncSvc, repo, cfg), syncSvc
}

func TestMalformedDateRangeReturns400(t *testing.T) {
	h, _ := newTestHandler(t, nil)

	for _, q := range []string{
		"start_date=2026-13-40&end_date=2026-08-21",
		"start_date=2026-08-01&end_date=not-a-date",
		"start_date=2026-08-21&end_date=2026-08-01",
	} {
		req := httptest.NewRequest("GET", "/api/v1/metrics/overview?"+q, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, w.Code)
		}
	}

	// Valid dates still work.
	req := httptest.NewRequest("GET", "/api/v1/metrics/overview?start_date=2026-08-01&end_date=2026-08-21", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("valid range: expected 200, got %d", w.Code)
	}

	// Absent dates keep the last-30-days default.
	req = httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("default range: expected 200, got %d", w.Code)
	}
}

// stubSyncService lets handler tests force specific sync outcomes.
type stubSyncService struct{ err error }

func (s *stubSyncService) SyncAll(ctx context.Context, start, end time.Time) error { return s.err }
func (s *stubSyncService) SyncChannel(ctx context.Context, code string, start, end time.Time) error {
	return s.err
}

func TestSyncConflictReturns409(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()
	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	analyticsSvc := service.NewAnalyticsService(repo)
	h := handler.NewMetricsHandler(analyticsSvc, &stubSyncService{err: service.ErrSyncInProgress}, repo, nil)

	req := httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict while sync in progress, got %d", w.Code)
	}
}

func TestCacheControlPrivateWhenAuthEnabled(t *testing.T) {
	cfg := &config.Config{AllowedOrigins: []string{"http://localhost:3000"}, APIKey: "secret-key-123"}
	h, _ := newTestHandler(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "private, max-age=60" {
		t.Errorf("expected private cache-control with auth, got %q", got)
	}

	// Without auth configured the response stays publicly cacheable.
	h2, _ := newTestHandler(t, nil)
	req2 := httptest.NewRequest("GET", "/api/v1/metrics/overview", nil)
	w2 := httptest.NewRecorder()
	h2.ServeHTTP(w2, req2)
	if got := w2.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Errorf("expected public cache-control without auth, got %q", got)
	}
}

func TestCSRFOriginCheck(t *testing.T) {
	h, _ := newTestHandler(t, nil)

	// Custom header with a disallowed Origin is rejected (#64).
	req := httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disallowed origin, got %d", w.Code)
	}

	// Allowed origin passes.
	req = httptest.NewRequest("POST", "/api/v1/sync", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "http://localhost:3000")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed origin, got %d", w.Code)
	}
}

func TestTrustedProxyResolution(t *testing.T) {
	resolver := handler.NewClientIPResolver([]string{"172.16.0.0/12"})

	// Peer inside the trusted Docker range: forwarded header honored (#57).
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.RemoteAddr = "172.17.0.5:44321"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := resolver.Resolve(req); got != "203.0.113.9" {
		t.Errorf("trusted proxy: expected 203.0.113.9, got %s", got)
	}

	// Untrusted peer: header ignored, peer address used.
	req = httptest.NewRequest("GET", "/api/v1/health", nil)
	req.RemoteAddr = "198.51.100.7:44321"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := resolver.Resolve(req); got != "198.51.100.7" {
		t.Errorf("untrusted peer: expected 198.51.100.7, got %s", got)
	}

	// Loopback peers remain trusted by default (pre-#57 behavior).
	loopback := handler.NewClientIPResolver(nil)
	req = httptest.NewRequest("GET", "/api/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:5000"
	req.Header.Set("X-Real-IP", "203.0.113.9")
	if got := loopback.Resolve(req); got != "203.0.113.9" {
		t.Errorf("loopback default: expected 203.0.113.9, got %s", got)
	}
}

func TestRateLimiterRetryAfter(t *testing.T) {
	if got := handler.NewRateLimiter(50, 100).RetryAfter(); got != 1 {
		t.Errorf("rate 50/s: expected Retry-After 1s, got %d", got)
	}
	if got := handler.NewRateLimiter(0.25, 10).RetryAfter(); got != 4 {
		t.Errorf("rate 0.25/s: expected Retry-After 4s, got %d", got)
	}
}

func TestHealthReportsDatabase(t *testing.T) {
	h, _ := newTestHandler(t, nil)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"database":"up"`) {
		t.Errorf("expected database status in health payload, got %s", body)
	}
}

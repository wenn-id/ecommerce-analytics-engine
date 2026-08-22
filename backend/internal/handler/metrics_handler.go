package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/metrics"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

type MetricsHandler struct {
	handler      http.Handler
	analyticsSvc service.AnalyticsService
	syncSvc      service.SyncService
	repo         store.Repository
	cfg          *config.Config
	limiter      *RateLimiter
	ipResolver   *ClientIPResolver
}

func NewMetricsHandler(analyticsSvc service.AnalyticsService, syncSvc service.SyncService, repo store.Repository, cfg *config.Config) *MetricsHandler {
	if cfg == nil {
		cfg = config.Load()
	}
	mux := http.NewServeMux()
	limiter := NewRateLimiter(50, 100)
	ipResolver := NewClientIPResolver(cfg.TrustedProxies)

	h := &MetricsHandler{
		analyticsSvc: analyticsSvc,
		syncSvc:      syncSvc,
		repo:         repo,
		cfg:          cfg,
		limiter:      limiter,
		ipResolver:   ipResolver,
	}
	h.registerRoutes(mux)

	h.handler = Chain(
		mux,
		RecoveryMiddleware,
		MetricsMiddleware(metrics.Default),
		LoggingMiddleware(ipResolver),
		CORSMiddleware(cfg),
		RateLimitMiddleware(limiter, ipResolver),
		AuthMiddleware(cfg),
	)
	return h
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h *MetricsHandler) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/metrics/overview", h.handleOverview)
	mux.HandleFunc("/api/v1/metrics/trend", h.handleTrend)
	mux.HandleFunc("/api/v1/metrics/channels", h.handleChannels)
	mux.HandleFunc("/api/v1/campaigns", h.handleCampaigns)
	mux.HandleFunc("/api/v1/sync", h.handleSync)
	mux.Handle("/metrics", metrics.Handler(metrics.Default))
}

// parseDateRange validates the start_date/end_date query parameters. Missing
// parameters fall back to the last 30 days; malformed values return an error
// so the caller can answer 400 instead of silently serving a different range
// than requested (#65).
func (h *MetricsHandler) parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	end := time.Now()
	if endStr != "" {
		parsed, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date %q: expected YYYY-MM-DD", endStr)
		}
		end = parsed
	}

	start := end.AddDate(0, 0, -30)
	if startStr != "" {
		parsed, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date %q: expected YYYY-MM-DD", startStr)
		}
		start = parsed
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_date %s is after end_date %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	return start, end, nil
}

// setCacheControl marks responses private when API-key auth is enabled, so
// shared caches can never store authenticated responses (#64).
func (h *MetricsHandler) setCacheControl(w http.ResponseWriter, maxAgeSeconds int) {
	visibility := "public"
	if h.cfg != nil && h.cfg.APIKey != "" {
		visibility = "private"
	}
	w.Header().Set("Cache-Control", fmt.Sprintf("%s, max-age=%d", visibility, maxAgeSeconds))
}

// checkOriginHeader is the anti-CSRF gate for state-mutating endpoints of
// this header/API-key authenticated API (no cookie auth). This is an
// origin/custom-header check, not token-based CSRF validation (#64):
//
//  1. A custom header (X-Requested-With / X-API-Key / X-CSRF-Token) must be
//     present. Browsers never attach custom headers cross-site without a
//     successful CORS preflight, which the CORS middleware only grants to
//     explicitly allowed origins.
//  2. When a browser supplies an Origin header, it must be an allowed
//     origin; non-browser clients without Origin pass on header presence.
func (h *MetricsHandler) checkOriginHeader(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "" &&
		r.Header.Get("X-API-Key") == "" &&
		r.Header.Get("X-CSRF-Token") == "" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if h.cfg == nil {
		return true
	}
	for _, allowed := range h.cfg.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// handleHealth reports service health including database connectivity so an
// unhealthy instance is detectable by orchestrators (#39).
func (h *MetricsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "up"
	if err := h.repo.Ping(pingCtx); err != nil {
		dbStatus = "down"
		slog.Error("health check database ping failed", "error", err)
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"status":    "degraded",
			"database":  dbStatus,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"database":  dbStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *MetricsHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.setCacheControl(w, 60)
	start, end, err := h.parseDateRange(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.analyticsSvc.GetOverviewMetrics(r.Context(), start, end)
	if err != nil {
		slog.Error("error fetching overview metrics", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to fetch overview metrics")
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.setCacheControl(w, 60)
	start, end, err := h.parseDateRange(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.analyticsSvc.GetTrendData(r.Context(), start, end)
	if err != nil {
		slog.Error("error fetching trend data", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to fetch trend data")
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.setCacheControl(w, 60)
	start, end, err := h.parseDateRange(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.analyticsSvc.GetChannelBreakdown(r.Context(), start, end)
	if err != nil {
		slog.Error("error fetching channel breakdown", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to fetch channel breakdown")
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.setCacheControl(w, 30)

	query := r.URL.Query()
	chIDStr := query.Get("channel_id")
	var chID int64
	if chIDStr != "" {
		chID, _ = strconv.ParseInt(chIDStr, 10, 64)
	}

	status := query.Get("status")
	search := query.Get("search")
	if len(search) > 200 {
		search = search[:200]
	}

	page := 1
	if pStr := query.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
	}

	filter := model.CampaignFilter{
		ChannelID: chID,
		Status:    status,
		Search:    search,
		Page:      page,
		Limit:     limit,
	}

	campaigns, totalRecords, err := h.repo.GetCampaigns(r.Context(), filter)
	if err != nil {
		slog.Error("error fetching campaigns", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to fetch campaigns")
		return
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = (totalRecords + limit - 1) / limit
	}

	response := model.PaginatedCampaigns{
		Data: campaigns,
		Pagination: model.PaginationMeta{
			CurrentPage:  page,
			Limit:        limit,
			TotalRecords: totalRecords,
			TotalPages:   totalPages,
		},
	}

	jsonResponse(w, http.StatusOK, response)
}

func (h *MetricsHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !h.checkOriginHeader(r) {
		jsonError(w, http.StatusForbidden, "CSRF protection: missing custom request header or disallowed origin")
		return
	}

	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if err := h.syncSvc.SyncAll(r.Context(), start, end); err != nil {
		if errors.Is(err, service.ErrSyncInProgress) {
			jsonError(w, http.StatusConflict, "sync already in progress, please retry once it completes")
			return
		}
		slog.Error("error triggering sync", "error", err)
		jsonError(w, http.StatusInternalServerError, "failed to trigger multi-channel sync")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "sync_completed"})
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

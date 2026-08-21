package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

type MetricsHandler struct {
	mux          *http.ServeMux
	analyticsSvc service.AnalyticsService
	syncSvc      service.SyncService
	repo         store.Repository
	cfg          *config.Config
	limiter      *RateLimiter
}

func NewMetricsHandler(analyticsSvc service.AnalyticsService, syncSvc service.SyncService, repo store.Repository, cfg *config.Config) *MetricsHandler {
	if cfg == nil {
		cfg = config.Load()
	}
	h := &MetricsHandler{
		mux:          http.NewServeMux(),
		analyticsSvc: analyticsSvc,
		syncSvc:      syncSvc,
		repo:         repo,
		cfg:          cfg,
		limiter:      NewRateLimiter(50, 100),
	}
	h.registerRoutes()
	return h
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		allowed := false
		for _, o := range h.cfg.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
	} else if len(h.cfg.AllowedOrigins) > 0 && h.cfg.AllowedOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Rate limiting check
	clientIP := getClientIP(r)
	if h.limiter != nil && !h.limiter.Allow(clientIP) {
		w.Header().Set("Retry-After", "1")
		jsonError(w, http.StatusTooManyRequests, "rate limit exceeded, please retry later")
		return
	}

	// Optional API key authentication
	if h.cfg.APIKey != "" && r.URL.Path != "/api/v1/health" {
		reqKey := r.Header.Get("X-API-Key")
		if reqKey == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				reqKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if reqKey != h.cfg.APIKey {
			jsonError(w, http.StatusUnauthorized, "unauthorized: invalid or missing API key")
			return
		}
	}

	h.mux.ServeHTTP(w, r)
}

func (h *MetricsHandler) registerRoutes() {
	h.mux.HandleFunc("/api/v1/health", h.handleHealth)
	h.mux.HandleFunc("/api/v1/metrics/overview", h.handleOverview)
	h.mux.HandleFunc("/api/v1/metrics/trend", h.handleTrend)
	h.mux.HandleFunc("/api/v1/metrics/channels", h.handleChannels)
	h.mux.HandleFunc("/api/v1/campaigns", h.handleCampaigns)
	h.mux.HandleFunc("/api/v1/sync", h.handleSync)
}

func (h *MetricsHandler) parseDateRange(r *http.Request) (time.Time, time.Time) {
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		end = time.Now()
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = end.AddDate(0, 0, -30)
	}
	return start, end
}

func (h *MetricsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
}

func (h *MetricsHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetOverviewMetrics(r.Context(), start, end)
	if err != nil {
		log.Printf("Error fetching overview metrics: %v", err)
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
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetTrendData(r.Context(), start, end)
	if err != nil {
		log.Printf("Error fetching trend data: %v", err)
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
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetChannelBreakdown(r.Context(), start, end)
	if err != nil {
		log.Printf("Error fetching channel breakdown: %v", err)
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
		log.Printf("Error fetching campaigns: %v", err)
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
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if err := h.syncSvc.SyncAll(r.Context(), start, end); err != nil {
		log.Printf("Error triggering sync: %v", err)
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

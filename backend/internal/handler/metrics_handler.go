package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

type MetricsHandler struct {
	mux          *http.ServeMux
	analyticsSvc service.AnalyticsService
	syncSvc      service.SyncService
	repo         store.Repository
}

func NewMetricsHandler(analyticsSvc service.AnalyticsService, syncSvc service.SyncService, repo store.Repository) *MetricsHandler {
	h := &MetricsHandler{
		mux:          http.NewServeMux(),
		analyticsSvc: analyticsSvc,
		syncSvc:      syncSvc,
		repo:         repo,
	}
	h.registerRoutes()
	return h
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
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
		jsonError(w, http.StatusInternalServerError, err.Error())
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
		jsonError(w, http.StatusInternalServerError, err.Error())
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
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	chIDStr := r.URL.Query().Get("channel_id")
	var chID int64
	if chIDStr != "" {
		chID, _ = strconv.ParseInt(chIDStr, 10, 64)
	}
	data, err := h.repo.GetCampaigns(r.Context(), chID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if err := h.syncSvc.SyncAll(r.Context(), start, end); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
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

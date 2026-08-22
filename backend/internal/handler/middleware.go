package handler

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/metrics"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered in HTTP handler",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()))
				jsonError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// routeLabel maps a request path to a fixed route label. Unmatched paths
// collapse into a single "unmatched" bucket so arbitrary client-supplied
// paths cannot grow metric label sets without bound.
func routeLabel(path string) string {
	switch path {
	case "/api/v1/health", "/api/v1/metrics/overview", "/api/v1/metrics/trend",
		"/api/v1/metrics/channels", "/api/v1/campaigns", "/api/v1/sync", "/metrics":
		return path
	default:
		return "unmatched"
	}
}

func MetricsMiddleware(reg *metrics.Registry) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg.RequestStarted()
			start := time.Now()
			rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
			// Deferred cleanup: RecoveryMiddleware wraps this middleware, so a
			// downstream panic unwinds through this frame before the inline
			// bookkeeping would run. Record the request (as a 500) and re-raise
			// so Recovery still converts it into a response.
			defer func() {
				if p := recover(); p != nil {
					reg.RequestFinished()
					reg.RecordRequest(routeLabel(r.URL.Path), r.Method, http.StatusInternalServerError, time.Since(start).Seconds())
					panic(p)
				}
				reg.RequestFinished()
				reg.RecordRequest(routeLabel(r.URL.Path), r.Method, rw.statusCode, time.Since(start).Seconds())
			}()
			next.ServeHTTP(rw, r)
		})
	}
}

func LoggingMiddleware(ipResolver *ClientIPResolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"client_ip", ipResolver.Resolve(r),
				"status", rw.statusCode,
				"duration_ms", time.Since(start).Milliseconds())
		})
	}
}

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func CORSMiddleware(cfg *config.Config) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && cfg != nil {
				allowed := false
				for _, o := range cfg.AllowedOrigins {
					if o == "*" || o == origin {
						allowed = true
						break
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			} else if cfg != nil && len(cfg.AllowedOrigins) > 0 && cfg.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Requested-With, X-CSRF-Token")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware enforces two buckets per request when the client IP
// came from a forwarded header (#57): the per-client bucket, and an
// aggregate bucket for the forwarding peer. The peer cap means a caller
// rotating spoofed X-Forwarded-For values through a trusted proxy cannot
// mint a fresh bucket per request and bypass per-client limits.
func RateLimitMiddleware(limiter *RateLimiter, ipResolver *ClientIPResolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter != nil {
				clientIP := ipResolver.Resolve(r)
				if !limiter.Allow(clientIP) {
					w.Header().Set("Retry-After", strconv.Itoa(limiter.RetryAfter()))
					jsonError(w, http.StatusTooManyRequests, "rate limit exceeded, please retry later")
					return
				}
				if peer := ipResolver.Peer(r); peer != clientIP {
					if !limiter.Allow("peer:" + peer) {
						w.Header().Set("Retry-After", strconv.Itoa(limiter.RetryAfter()))
						jsonError(w, http.StatusTooManyRequests, "rate limit exceeded, please retry later")
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware enforces the static API key on every route except the
// health endpoint and the Prometheus /metrics endpoint. The metrics endpoint
// is intentionally unauthenticated because the backend is expected to run on
// a private network (no host port mapping; see docker-compose.yml) and the
// exposition contains no business data.
func AuthMiddleware(cfg *config.Config) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg != nil && cfg.APIKey != "" && !isPublicPath(r.URL.Path) {
				reqKey := r.Header.Get("X-API-Key")
				if reqKey == "" {
					authHeader := r.Header.Get("Authorization")
					if strings.HasPrefix(authHeader, "Bearer ") {
						reqKey = strings.TrimPrefix(authHeader, "Bearer ")
					}
				}
				if reqKey != cfg.APIKey {
					jsonError(w, http.StatusUnauthorized, "unauthorized: invalid or missing API key")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isPublicPath(path string) bool {
	return path == "/api/v1/health" || path == "/metrics"
}

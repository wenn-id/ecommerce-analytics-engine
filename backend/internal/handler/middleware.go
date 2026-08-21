package handler

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"ecommerce-analytics/internal/config"
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
				log.Printf("PANIC recovered in HTTP handler: %v\nStack: %s", err, string(debug.Stack()))
				jsonError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		log.Printf("[%s] %s %s %d (%v)", r.Method, r.URL.Path, getClientIP(r), rw.statusCode, duration)
	})
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

func RateLimitMiddleware(limiter *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter != nil {
				clientIP := getClientIP(r)
				if !limiter.Allow(clientIP) {
					w.Header().Set("Retry-After", "1")
					jsonError(w, http.StatusTooManyRequests, "rate limit exceeded, please retry later")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(cfg *config.Config) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg != nil && cfg.APIKey != "" && r.URL.Path != "/api/v1/health" {
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

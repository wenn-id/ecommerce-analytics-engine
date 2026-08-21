package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientVisitor struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*clientVisitor
	rate     float64 // tokens per second
	capacity float64 // max token capacity
}

func NewRateLimiter(rate float64, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     rate,
		capacity: capacity,
	}
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &clientVisitor{
			tokens:     rl.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(v.lastRefill).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > rl.capacity {
		v.tokens = rl.capacity
	}
	v.lastRefill = now

	if v.tokens >= 1 {
		v.tokens--
		return true
	}
	return false
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

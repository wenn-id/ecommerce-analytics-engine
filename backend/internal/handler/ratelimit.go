package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxRateLimiterEntries = 5000
	visitorExpiry         = 3 * time.Minute
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
	go rl.cleanupLoop(1 * time.Minute)
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// If visitors map reaches capacity, perform proactive cleanup
	if len(rl.visitors) >= maxRateLimiterEntries {
		for k, v := range rl.visitors {
			if now.Sub(v.lastRefill) > visitorExpiry {
				delete(rl.visitors, k)
			}
		}
		// If still full, reset to prevent unbounded memory growth
		if len(rl.visitors) >= maxRateLimiterEntries {
			rl.visitors = make(map[string]*clientVisitor)
		}
	}

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

func (rl *RateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastRefill) > visitorExpiry {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts client IP, only trusting forwarded headers if the immediate peer is a local proxy.
func getClientIP(r *http.Request) string {
	peerHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peerHost = r.RemoteAddr
	}

	// Only trust forwarded headers if request is from a local/trusted proxy (e.g. Next.js BFF proxy on loopback)
	if isLoopbackOrLocal(peerHost) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				clientIP := strings.TrimSpace(parts[0])
				if clientIP != "" && net.ParseIP(clientIP) != nil {
					return clientIP
				}
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			clientIP := strings.TrimSpace(xri)
			if clientIP != "" && net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}

	return peerHost
}

func isLoopbackOrLocal(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return host == "localhost"
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()
}

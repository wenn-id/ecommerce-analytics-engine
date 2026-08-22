package handler

import (
	"log/slog"
	"math"
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

	v, exists := rl.visitors[ip]
	if !exists {
		// Evict oldest visitor entry if capacity reached
		if len(rl.visitors) >= maxRateLimiterEntries {
			var oldestKey string
			var oldestTime time.Time
			for k, vis := range rl.visitors {
				if oldestTime.IsZero() || vis.lastRefill.Before(oldestTime) {
					oldestTime = vis.lastRefill
					oldestKey = k
				}
			}
			if oldestKey != "" {
				delete(rl.visitors, oldestKey)
			}
		}

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

// RetryAfter returns the number of seconds a client should wait before
// retrying, derived from the token refill rate instead of a hardcoded value
// (#65): one token is always available after ceil(1/rate) seconds.
func (rl *RateLimiter) RetryAfter() int {
	if rl.rate <= 0 {
		return 1
	}
	secs := math.Ceil(1 / rl.rate)
	if secs < 1 {
		return 1
	}
	return int(secs)
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

// ClientIPResolver resolves the real client IP behind optional reverse
// proxies. Forwarded headers (X-Forwarded-For / X-Real-IP) are only honored
// when the immediate peer matches a trusted network; otherwise any client
// could spoof its IP and bypass per-client rate limiting (#57).
type ClientIPResolver struct {
	trustedCIDRs []*net.IPNet
}

// NewClientIPResolver builds a resolver from a list of IPs/CIDRs. When the
// list is empty, only loopback peers are trusted (same behavior as before
// #57), which is the safe default for direct exposure.
func NewClientIPResolver(trustedProxies []string) *ClientIPResolver {
	resolver := &ClientIPResolver{}
	for _, entry := range trustedProxies {
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			slog.Warn("ignoring invalid trusted proxy entry", "entry", entry, "error", err)
			continue
		}
		resolver.trustedCIDRs = append(resolver.trustedCIDRs, cidr)
	}
	return resolver
}

// Peer returns the immediate peer address of the request (the TCP remote),
// regardless of forwarded headers.
func (c *ClientIPResolver) Peer(r *http.Request) string {
	peerHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peerHost = r.RemoteAddr
	}
	return peerHost
}

// Resolve returns the best-known client IP for the request: the first valid
// X-Forwarded-For entry (falling back to X-Real-IP) when the immediate peer
// is trusted, or the peer address itself otherwise.
func (c *ClientIPResolver) Resolve(r *http.Request) string {
	peerHost := c.Peer(r)
	peerIP := net.ParseIP(peerHost)
	if peerIP == nil {
		// Non-IP peer (e.g. "localhost" from test servers): treat as loopback.
		if isLoopback(peerHost) {
			peerIP = net.ParseIP("127.0.0.1")
		}
	}

	if peerIP != nil && c.isTrusted(peerIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// The proxy appends the peer it received the request from, so the
			// left-most entry is the original client. Walk right-to-left and
			// skip hops that are themselves trusted proxies, so a spoofed
			// left-most value cannot be used when an untrusted hop is in the
			// chain.
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				if candidate == "" {
					continue
				}
				candidateIP := net.ParseIP(hostOnly(candidate))
				if candidateIP == nil {
					continue
				}
				if i == 0 || !c.isTrusted(candidateIP) {
					return candidateIP.String()
				}
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			if ip := net.ParseIP(hostOnly(xri)); ip != nil {
				return ip.String()
			}
		}
	}

	return peerHost
}

func (c *ClientIPResolver) isTrusted(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	for _, cidr := range c.trustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func hostOnly(host string) string {
	// Strip an optional port from header-supplied values (e.g. "1.2.3.4:5678").
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func isLoopback(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return host == "localhost" || host == "127.0.0.1" || host == "::1"
	}
	return ip.IsLoopback()
}

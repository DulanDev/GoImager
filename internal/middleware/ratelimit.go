package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter is a per-client-IP token bucket limiter. Either per-second
// (rps) or per-minute (rpm) refill is used; rpm overrides rps when > 0.
// A zero rate disables limiting entirely.
type RateLimiter struct {
	rate    float64 // tokens added per second
	burst   float64
	mu      sync.Mutex
	clients map[string]*bucket
}

func NewRateLimiter(rps, rpm int) *RateLimiter {
	var rate, burst float64
	switch {
	case rpm > 0:
		rate = float64(rpm) / 60.0
		burst = float64(rpm)
	case rps > 0:
		rate = float64(rps)
		burst = float64(rps)
	default:
		return &RateLimiter{rate: 0}
	}
	return &RateLimiter{
		rate:    rate,
		burst:   burst,
		clients: make(map[string]*bucket),
	}
}

// Allow reports whether client ip may proceed, refilling the bucket first.
func (rl *RateLimiter) Allow(ip string) bool {
	if rl == nil || rl.rate <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.clients[ip]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.clients[ip] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min64(b.tokens+elapsed*rl.rate, rl.burst)
	b.last = now
	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

// AllowHTTP extracts the client IP and writes 429 when over limit.
func (rl *RateLimiter) AllowHTTP(w http.ResponseWriter, r *http.Request) bool {
	return rl.Allow(clientIP(r))
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !rl.AllowHTTP(w, r) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMITED",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Adapter returns a gorilla/mux-compatible middleware function.
func (rl *RateLimiter) Adapter() func(http.Handler) http.Handler {
	return rl.Middleware
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := indexOfComma(xff); idx >= 0 {
			return trimSpace(xff[:idx])
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexOfComma(s string) int {
	for i, c := range s {
		if c == ',' {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateFromHeaders reports the configured rate in human form for Retry-After
// heuristics; unused for now but kept for future admin endpoints.
func (rl *RateLimiter) RateFromHeaders() string {
	return strconv.FormatFloat(rl.rate, 'f', 2, 64)
}

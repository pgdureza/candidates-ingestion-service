package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters sync.Map // map[ip]*rate.Limiter
	limit    rate.Limit
	burst    int
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		limit: rate.Limit(float64(requestsPerMinute) / 60.0),
		burst: 1,
	}
}

func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For for proxied requests
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	// Fall back to RemoteAddr
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	return host
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	limiter, _ := rl.limiters.LoadOrStore(ip, rate.NewLimiter(rl.limit, rl.burst))
	return limiter.(*rate.Limiter)
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.getClientIP(r)
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Cleanup old limiters periodically (optional, prevents unbounded memory growth)
func (rl *RateLimiter) CleanupOldLimiters() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.limiters.Range(func(key, value interface{}) bool {
			rl.limiters.Delete(key)
			return true
		})
	}
}

package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/candidate-ingestion/service/internal/domain/repo"
)

type RateLimiter struct {
	limiters sync.Map // map[ip]*rate.Limiter
	limit    rate.Limit
	burst    int
	db       repo.DB
}

func NewRateLimiter(requestsPerMinute int, db repo.DB) *RateLimiter {
	return &RateLimiter{
		limit: rate.Limit(float64(requestsPerMinute) / 60.0),
		burst: 1,
		db:    db,
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
		_ = rl.db.Metrics().IncrementMetric(r.Context(), "webhooks_total_request", 1)
		ip := rl.getClientIP(r)
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			_ = rl.db.Metrics().IncrementMetric(r.Context(), "webhooks_rate_limited", 1)
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

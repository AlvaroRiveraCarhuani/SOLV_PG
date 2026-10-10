package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/metrics"
)

type userLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type UserRateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

func NewUserRateLimiter(reqsPerMinute int, burst int) *UserRateLimiter {
	r := rate.Limit(float64(reqsPerMinute) / 60.0)
	limiter := &UserRateLimiter{
		rate:  r,
		burst: burst,
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			now := time.Now()
			limiter.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*userLimiterEntry)
				if now.Sub(entry.lastSeen) > 10*time.Minute {
					limiter.limiters.Delete(key)
				}
				return true
			})
		}
	}()

	return limiter
}

func (l *UserRateLimiter) getLimiter(key string) *userLimiterEntry {
	now := time.Now()
	val, loaded := l.limiters.Load(key)
	if loaded {
		entry := val.(*userLimiterEntry)
		entry.lastSeen = now
		return entry
	}

	newEntry := &userLimiterEntry{
		limiter:  rate.NewLimiter(l.rate, l.burst),
		lastSeen: now,
	}
	actual, _ := l.limiters.LoadOrStore(key, newEntry)
	return actual.(*userLimiterEntry)
}

func RateLimitMiddleware(limiter *UserRateLimiter) func(http.Handler) http.Handler {
	return EndpointRateLimitMiddleware(limiter, "general")
}

func EndpointRateLimitMiddleware(limiter *UserRateLimiter, endpointName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := domain.GetUserID(r.Context())
			if key == "" {
				key = r.Header.Get("X-User-Id")
			}
			if key == "" {
				key = getClientIP(r)
			}

			entry := limiter.getLimiter(key)
			now := time.Now()
			resetUnix := now.Add(60 * time.Second).Unix()

			tokensRemaining := int(entry.limiter.Tokens())
			if tokensRemaining < 0 {
				tokensRemaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.burst))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tokensRemaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))

			if !entry.limiter.Allow() {
				metrics.RateLimitHitsTotal.WithLabelValues(endpointName).Inc()
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate_limited",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

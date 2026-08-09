package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
	"golang.org/x/time/rate"
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

	// Routine periódica de limpieza de clientes inactivos por más de 10 minutos
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

func (l *UserRateLimiter) getLimiter(userID string) *userLimiterEntry {
	now := time.Now()
	val, loaded := l.limiters.Load(userID)
	if loaded {
		entry := val.(*userLimiterEntry)
		entry.lastSeen = now
		return entry
	}

	newEntry := &userLimiterEntry{
		limiter:  rate.NewLimiter(l.rate, l.burst),
		lastSeen: now,
	}
	actual, _ := l.limiters.LoadOrStore(userID, newEntry)
	return actual.(*userLimiterEntry)
}

func RateLimitMiddleware(limiter *UserRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := domain.GetUserID(r.Context())
			if userID == "" {
				// Fallback a IP si no hay userID en contexto
				userID = r.RemoteAddr
			}

			entry := limiter.getLimiter(userID)
			now := time.Now()
			resetUnix := now.Add(60 * time.Second).Unix()

			tokensRemaining := int(entry.limiter.Tokens())
			if tokensRemaining < 0 {
				tokensRemaining = 0
			}

			// Headers informativos obligatorios RFC 6585
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.burst))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tokensRemaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))

			if !entry.limiter.Allow() {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Rate limit exceeded. Please wait before starting another workspace.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

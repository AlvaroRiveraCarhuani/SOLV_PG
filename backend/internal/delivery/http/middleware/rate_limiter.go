package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"solv-backend/internal/core/domain"
)

type RateLimiter struct {
	ipLimiters         map[string]*rate.Limiter
	submissionLimiters map[string]*rate.Limiter
	authFailures       map[string]authFailureRecord
	mu                 sync.RWMutex
}

type authFailureRecord struct {
	count     int
	blockedAt time.Time
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		ipLimiters:         make(map[string]*rate.Limiter),
		submissionLimiters: make(map[string]*rate.Limiter),
		authFailures:       make(map[string]authFailureRecord),
	}

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			rl.mu.Lock()
			now := time.Now()
			for ip, rec := range rl.authFailures {
				if now.Sub(rec.blockedAt) > 15*time.Minute {
					delete(rl.authFailures, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *RateLimiter) GetIPLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.ipLimiters[ip]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(100.0/60.0), 20)
	rl.ipLimiters[ip] = limiter
	return limiter
}

func (rl *RateLimiter) GetSubmissionLimiter(studentID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.submissionLimiters[studentID]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(10.0/60.0), 5)
	rl.submissionLimiters[studentID] = limiter
	return limiter
}

func (rl *RateLimiter) RecordAuthFailure(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rec := rl.authFailures[ip]
	rec.count++
	if rec.count >= 5 {
		rec.blockedAt = time.Now()
	}
	rl.authFailures[ip] = rec
	return rec.count >= 5
}

func (rl *RateLimiter) IsAuthBlocked(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	rec, exists := rl.authFailures[ip]
	if !exists || rec.count < 5 {
		return false
	}

	if time.Since(rec.blockedAt) > 15*time.Minute {
		return false
	}

	return true
}

func getClientIP(r *http.Request) string {
	if forward := r.Header.Get("X-Forwarded-For"); forward != "" {
		return strings.TrimSpace(strings.Split(forward, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func GlobalRateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if rl.IsAuthBlocked(ip) {
				w.Header().Set("Retry-After", "900")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Demasiados intentos fallidos de autenticación. Bloqueado temporalmente.",
				})
				return
			}

			limiter := rl.GetIPLimiter(ip)
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Too Many Requests",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func SubmissionRateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			studentID := domain.GetUserID(r.Context())
			if studentID == "" {
				studentID = r.Header.Get("X-User-Id")
			}
			if studentID == "" {
				studentID = r.RemoteAddr
			}

			limiter := rl.GetSubmissionLimiter(studentID)
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Límite de envíos por minuto excedido (máximo 10/minuto).",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

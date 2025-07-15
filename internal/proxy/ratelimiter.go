package proxy

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	clients map[string]*rate.Limiter
	mu      sync.Mutex
	rate    rate.Limit
	burst   int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*rate.Limiter),
		rate:    r,
		burst:   b,
	}
}

func (rl *RateLimiter) getClient(apiKey string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.clients[apiKey]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.clients[apiKey] = limiter
	}

	return limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		limiter := rl.getClient(apiKey)
		if !limiter.Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

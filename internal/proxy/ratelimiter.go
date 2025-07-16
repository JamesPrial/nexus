package proxy

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// GlobalRateLimiter applies a single rate limit to all incoming requests.
type GlobalRateLimiter struct {
	limiter *rate.Limiter
}

// NewGlobalRateLimiter creates a new global rate limiter.
func NewGlobalRateLimiter(r rate.Limit, b int) *GlobalRateLimiter {
	return &GlobalRateLimiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// Middleware wraps an http.Handler with rate-limiting logic.
func (rl *GlobalRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			// We can add a Retry-After header to be more compliant
			// w.Header().Set("Retry-After", "10") // Example
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// PerClientRateLimiter applies rate limits on a per-API-key basis.
// This was the old behavior of RateLimiter, preserved here for clarity.
type PerClientRateLimiter struct {
	clients map[string]*rate.Limiter
	mu      sync.Mutex
	rate    rate.Limit
	burst   int
}

// NewPerClientRateLimiter creates a new per-client rate limiter.
func NewPerClientRateLimiter(r rate.Limit, b int) *PerClientRateLimiter {
	return &PerClientRateLimiter{
		clients: make(map[string]*rate.Limiter),
		rate:    r,
		burst:   b,
	}
}

func (rl *PerClientRateLimiter) getClient(apiKey string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.clients[apiKey]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.clients[apiKey] = limiter
	}
	return limiter
}

func (rl *PerClientRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			// For per-client limiting, an API key is essential.
			http.Error(w, "Authorization header is required for rate limiting", http.StatusUnauthorized)
			return
		}

		limiter := rl.getClient(apiKey)
		if !limiter.Allow() {
			http.Error(w, "Too many requests for this client", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

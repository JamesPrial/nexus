package proxy

import (
	"net/http"
	"sync"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/utils"
	"golang.org/x/time/rate"
)

// PerClientRateLimiterWithTTL extends PerClientRateLimiter with TTL cleanup
type PerClientRateLimiterWithTTL struct {
	*PerClientRateLimiter
	lastAccess map[string]time.Time
	ttl        time.Duration
	logger     interfaces.Logger
	mu         sync.RWMutex
}

// NewPerClientRateLimiterWithTTL creates a new per-client rate limiter with TTL
func NewPerClientRateLimiterWithTTL(r rate.Limit, b int, ttl time.Duration, logger interfaces.Logger) *PerClientRateLimiterWithTTL {
	return &PerClientRateLimiterWithTTL{
		PerClientRateLimiter: NewPerClientRateLimiter(r, b),
		lastAccess:           make(map[string]time.Time),
		ttl:                  ttl,
		logger:               logger,
	}
}

// Middleware implements the rate limiting middleware with TTL tracking
func (r *PerClientRateLimiterWithTTL) Middleware(next http.Handler) http.Handler {
	originalMiddleware := r.PerClientRateLimiter.Middleware(next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		apiKey := req.Header.Get("Authorization")
		if apiKey != "" {
			r.updateLastAccess(apiKey)
		}

		if r.logger != nil {
			r.logger.Debug("Per-client rate limit check", map[string]any{
				"path":    req.URL.Path,
				"api_key": utils.MaskAPIKey(apiKey),
			})
		}

		originalMiddleware.ServeHTTP(w, req)
	})
}

// updateLastAccess updates the last access time for a client
func (r *PerClientRateLimiterWithTTL) updateLastAccess(apiKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastAccess[apiKey] = time.Now()
}

// getOrCreateLimiter gets or creates a limiter for the given key
func (r *PerClientRateLimiterWithTTL) getOrCreateLimiter(apiKey string) *rate.Limiter {
	r.updateLastAccess(apiKey)
	return r.PerClientRateLimiter.getClient(apiKey)
}

// HasClient checks if a client is currently tracked
func (r *PerClientRateLimiterWithTTL) HasClient(apiKey string) bool {
	r.PerClientRateLimiter.mu.Lock()
	defer r.PerClientRateLimiter.mu.Unlock()
	_, exists := r.PerClientRateLimiter.clients[apiKey]
	return exists
}

// ClientCount returns the number of tracked clients
func (r *PerClientRateLimiterWithTTL) ClientCount() int {
	r.PerClientRateLimiter.mu.Lock()
	defer r.PerClientRateLimiter.mu.Unlock()
	return len(r.PerClientRateLimiter.clients)
}

// StartCleanup starts a goroutine that periodically cleans up expired entries
func (r *PerClientRateLimiterWithTTL) StartCleanup(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-stop:
			return
		}
	}
}

// cleanup removes expired client entries
func (r *PerClientRateLimiterWithTTL) cleanup() {
	r.mu.RLock()
	now := time.Now()
	expired := make([]string, 0)
	
	for apiKey, lastAccess := range r.lastAccess {
		if now.Sub(lastAccess) > r.ttl {
			expired = append(expired, apiKey)
		}
	}
	r.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	// Remove expired entries
	r.mu.Lock()
	r.PerClientRateLimiter.mu.Lock()
	
	for _, apiKey := range expired {
		delete(r.lastAccess, apiKey)
		delete(r.PerClientRateLimiter.clients, apiKey)
	}
	
	r.PerClientRateLimiter.mu.Unlock()
	r.mu.Unlock()

	if r.logger != nil && len(expired) > 0 {
		r.logger.Debug("Cleaned up inactive rate limiter entries", map[string]any{
			"cleaned_count": len(expired),
		})
	}
}

// GetLimit returns remaining requests for the API key
func (r *PerClientRateLimiterWithTTL) GetLimit(apiKey string) (allowed bool, remaining int) {
	limiter := r.PerClientRateLimiter.getClient(apiKey)
	tokens := limiter.Tokens()
	return tokens > 0, int(tokens)
}

// Reset clears the rate limit state for the API key
func (r *PerClientRateLimiterWithTTL) Reset(apiKey string) {
	r.mu.Lock()
	delete(r.lastAccess, apiKey)
	r.mu.Unlock()

	r.PerClientRateLimiter.mu.Lock()
	delete(r.PerClientRateLimiter.clients, apiKey)
	r.PerClientRateLimiter.mu.Unlock()

	if r.logger != nil {
		r.logger.Info("Reset per-client rate limit", map[string]any{
			"api_key": utils.MaskAPIKey(apiKey),
		})
	}
}

// TokenLimiterWithTTL extends the token limiter with TTL cleanup
type TokenLimiterWithTTL struct {
	clients      map[string]*rate.Limiter
	lastAccess   map[string]time.Time
	mu           sync.RWMutex
	tpm          int
	tps          float64
	burst        int
	ttl          time.Duration
	tokenCounter interfaces.TokenCounter
	logger       interfaces.Logger
}

// NewTokenLimiterWithTTL creates a token limiter with TTL cleanup
func NewTokenLimiterWithTTL(tpm, burst int, counter interfaces.TokenCounter, ttl time.Duration, logger interfaces.Logger) *TokenLimiterWithTTL {
	return &TokenLimiterWithTTL{
		tpm:          tpm,
		tps:          float64(tpm) / 60.0,
		burst:        burst,
		clients:      make(map[string]*rate.Limiter),
		lastAccess:   make(map[string]time.Time),
		ttl:          ttl,
		tokenCounter: counter,
		logger:       logger,
	}
}

// Middleware implements the rate limiting middleware
func (t *TokenLimiterWithTTL) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		t.mu.Lock()
		limiter, exists := t.clients[apiKey]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(t.tps), t.burst)
			t.clients[apiKey] = limiter
		}
		t.lastAccess[apiKey] = time.Now()
		t.mu.Unlock()

		// Count tokens for the request
		tokenCount, err := t.tokenCounter.CountTokens(r)
		if err != nil {
			if t.logger != nil {
				t.logger.Error("Token counting failed", map[string]any{
					"error":   err.Error(),
					"api_key": utils.MaskAPIKey(apiKey),
				})
			}
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		if !limiter.AllowN(time.Now(), tokenCount) {
			if t.logger != nil {
				t.logger.Warn("Token limit exceeded", map[string]any{
					"api_key":          utils.MaskAPIKey(apiKey),
					"tokens_needed":    tokenCount,
					"tokens_available": limiter.Tokens(),
				})
			}
			http.Error(w, "Token limit exceeded", http.StatusTooManyRequests)
			return
		}

		if t.logger != nil {
			t.logger.Debug("Token limit check passed", map[string]any{
				"api_key":          utils.MaskAPIKey(apiKey),
				"tokens_used":      tokenCount,
				"tokens_remaining": limiter.Tokens(),
			})
		}

		next.ServeHTTP(w, r)
	})
}

// HasClient checks if a client is currently tracked
func (t *TokenLimiterWithTTL) HasClient(apiKey string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.clients[apiKey]
	return exists
}

// StartCleanup starts a goroutine that periodically cleans up expired entries
func (t *TokenLimiterWithTTL) StartCleanup(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.cleanup()
		case <-stop:
			return
		}
	}
}

// cleanup removes expired client entries
func (t *TokenLimiterWithTTL) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for apiKey, lastAccess := range t.lastAccess {
		if now.Sub(lastAccess) > t.ttl {
			expired = append(expired, apiKey)
		}
	}

	for _, apiKey := range expired {
		delete(t.lastAccess, apiKey)
		delete(t.clients, apiKey)
	}

	if t.logger != nil && len(expired) > 0 {
		t.logger.Debug("Cleaned up inactive token limiter entries", map[string]any{
			"cleaned_count": len(expired),
		})
	}
}

// GetLimit returns token limit information for an API key
func (t *TokenLimiterWithTTL) GetLimit(apiKey string) (allowed bool, remaining int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	limiter, exists := t.clients[apiKey]
	if !exists {
		return true, t.burst
	}

	tokens := limiter.Tokens()
	return tokens > 0, int(tokens)
}

// Reset clears the token limit state for an API key
func (t *TokenLimiterWithTTL) Reset(apiKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.lastAccess, apiKey)
	delete(t.clients, apiKey)

	if t.logger != nil {
		t.logger.Info("Reset token limit for API key", map[string]any{
			"api_key": utils.MaskAPIKey(apiKey),
		})
	}
}
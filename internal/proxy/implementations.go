package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"golang.org/x/time/rate"
)

// DefaultTokenCounter implements interfaces.TokenCounter
type DefaultTokenCounter struct{}

// CountTokens implements the token counting logic
func (d *DefaultTokenCounter) CountTokens(r *http.Request) (int, error) {
	// Read request body without consuming it
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// If no body content, assign minimal token count
	if len(body) == 0 {
		return 1, nil
	}

	// Parse JSON structure (OpenAI-compatible format)
	var payload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Prompt string `json:"prompt"`
	}

	// If JSON parsing fails, count the raw body size
	if err := json.Unmarshal(body, &payload); err != nil {
		tokenCount := len(body) / 4
		if tokenCount < 1 {
			tokenCount = 1
		}
		return tokenCount, nil
	}

	// Calculate token count: 4 characters ≈ 1 token
	tokenCount := 0

	// Count tokens from messages if present
	for _, msg := range payload.Messages {
		tokenCount += len(msg.Content) / 4
	}

	// Also count tokens from prompt if present
	tokenCount += len(payload.Prompt) / 4

	// Add minimum token count for system messages/metadata
	if tokenCount < 5 {
		tokenCount = 5
	}

	return tokenCount, nil
}

// HTTPProxy implements interfaces.Proxy
type HTTPProxy struct {
	ReverseProxy *httputil.ReverseProxy
	Logger       interfaces.Logger
	target       *url.URL
	mu           sync.RWMutex
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Logger != nil {
		h.Logger.Debug("Proxying request", map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"host":   r.Host,
		})
	}
	
	h.ReverseProxy.ServeHTTP(w, r)
}

// SetTarget changes the upstream target URL
func (h *HTTPProxy) SetTarget(targetURL string) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}
	
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.target = target
	h.ReverseProxy = httputil.NewSingleHostReverseProxy(target)
	
	if h.Logger != nil {
		h.Logger.Info("Updated proxy target", map[string]any{
			"target": targetURL,
		})
	}
	
	return nil
}

// NewRateLimiterWithLogger creates a rate limiter with logging support
func NewRateLimiterWithLogger(r rate.Limit, b int, logger interfaces.Logger) interfaces.RateLimiter {
	return &rateLimiterWithLogger{
		RateLimiter: NewRateLimiter(r, b),
		logger:      logger,
	}
}

// rateLimiterWithLogger wraps the existing RateLimiter with logging
type rateLimiterWithLogger struct {
	*RateLimiter
	logger interfaces.Logger
}

// GetLimit returns rate limit information for an API key
func (r *rateLimiterWithLogger) GetLimit(apiKey string) (allowed bool, remaining int) {
	limiter := r.getClient(apiKey)
	tokens := limiter.Tokens()
	
	return tokens > 0, int(tokens)
}

// Reset clears the rate limit state for an API key
func (r *rateLimiterWithLogger) Reset(apiKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.clients, apiKey)
	
	if r.logger != nil {
		r.logger.Info("Reset rate limit for API key", map[string]any{
			"api_key_prefix": apiKey[:min(len(apiKey), 10)],
		})
	}
}

// Middleware wraps the original middleware with logging
func (r *rateLimiterWithLogger) Middleware(next http.Handler) http.Handler {
	originalMiddleware := r.RateLimiter.Middleware(next)
	
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		apiKey := req.Header.Get("Authorization")
		
		if r.logger != nil {
			r.logger.Debug("Rate limit check", map[string]any{
				"api_key_prefix": apiKey[:min(len(apiKey), 10)],
				"method":         req.Method,
				"path":           req.URL.Path,
			})
		}
		
		originalMiddleware.ServeHTTP(w, req)
	})
}

// NewTokenLimiterWithDeps creates a token limiter with dependency injection
func NewTokenLimiterWithDeps(tpm, burst int, counter interfaces.TokenCounter, logger interfaces.Logger) interfaces.RateLimiter {
	return &tokenLimiterWithDeps{
		tpm:          tpm,
		tps:          float64(tpm) / 60.0,
		burst:        burst,
		clients:      make(map[string]*rate.Limiter),
		tokenCounter: counter,
		logger:       logger,
	}
}

// tokenLimiterWithDeps implements interfaces.RateLimiter with dependency injection
type tokenLimiterWithDeps struct {
	clients      map[string]*rate.Limiter
	mu           sync.Mutex
	tpm          int
	tps          float64
	burst        int
	tokenCounter interfaces.TokenCounter
	logger       interfaces.Logger
}

// Middleware implements the rate limiting middleware
func (t *tokenLimiterWithDeps) Middleware(next http.Handler) http.Handler {
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
		t.mu.Unlock()

		// Count tokens for the request
		tokenCount, err := t.tokenCounter.CountTokens(r)
		if err != nil {
			if t.logger != nil {
				t.logger.Error("Token counting failed", map[string]any{
					"error":          err.Error(),
					"api_key_prefix": apiKey[:min(len(apiKey), 10)],
				})
			}
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		if !limiter.AllowN(time.Now(), tokenCount) {
			if t.logger != nil {
				t.logger.Warn("Token limit exceeded", map[string]any{
					"api_key_prefix": apiKey[:min(len(apiKey), 10)],
					"tokens_needed":  tokenCount,
					"tokens_available": limiter.Tokens(),
				})
			}
			http.Error(w, "Token limit exceeded", http.StatusTooManyRequests)
			return
		}

		if t.logger != nil {
			t.logger.Debug("Token limit check passed", map[string]any{
				"api_key_prefix": apiKey[:min(len(apiKey), 10)],
				"tokens_used":    tokenCount,
				"tokens_remaining": limiter.Tokens(),
			})
		}

		next.ServeHTTP(w, r)
	})
}

// GetLimit returns token limit information for an API key
func (t *tokenLimiterWithDeps) GetLimit(apiKey string) (allowed bool, remaining int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	limiter, exists := t.clients[apiKey]
	if !exists {
		return true, t.burst
	}
	
	tokens := limiter.Tokens()
	return tokens > 0, int(tokens)
}

// Reset clears the token limit state for an API key
func (t *tokenLimiterWithDeps) Reset(apiKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	delete(t.clients, apiKey)
	
	if t.logger != nil {
		t.logger.Info("Reset token limit for API key", map[string]any{
			"api_key_prefix": apiKey[:min(len(apiKey), 10)],
		})
	}
}

// Helper function for safe string slicing
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
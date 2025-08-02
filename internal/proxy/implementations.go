package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/utils"
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
	// Validate input
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Errorf("target URL cannot be empty")
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	// Validate parsed URL
	if target.Scheme == "" {
		return fmt.Errorf("target URL must have a scheme")
	}
	if target.Host == "" {
		return fmt.Errorf("target URL must have a host")
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

func NewPerClientRateLimiterWithLogger(r rate.Limit, b int, logger interfaces.Logger) interfaces.RateLimiter {
	return &perClientRateLimiterWithLogger{
		limiter: NewPerClientRateLimiter(r, b),
		logger:  logger,
	}
}

// perClientRateLimiterWithLogger wraps the PerClientRateLimiter with logging.
type perClientRateLimiterWithLogger struct {
	limiter *PerClientRateLimiter
	logger  interfaces.Logger
}

// Middleware wraps the original middleware with logging.
func (r *perClientRateLimiterWithLogger) Middleware(next http.Handler) http.Handler {
	originalMiddleware := r.limiter.Middleware(next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		apiKey := req.Header.Get("Authorization")
		if r.logger != nil {
			r.logger.Debug("Per-client rate limit check", map[string]any{
				"path":    req.URL.Path,
				"api_key": utils.MaskAPIKey(apiKey),
			})
		}
		originalMiddleware.ServeHTTP(w, req)
	})
}

// GetLimit returns remaining requests for the API key.
func (r *perClientRateLimiterWithLogger) GetLimit(apiKey string) (allowed bool, remaining int) {
	limiter := r.limiter.getClient(apiKey)
	tokens := limiter.Tokens()
	return tokens > 0, int(tokens)
}

// Reset clears the rate limit state for the API key.
func (r *perClientRateLimiterWithLogger) Reset(apiKey string) {
	r.limiter.mu.Lock()
	delete(r.limiter.clients, apiKey)
	r.limiter.mu.Unlock()

	if r.logger != nil {
		r.logger.Info("Reset per-client rate limit", map[string]any{
			"api_key": utils.MaskAPIKey(apiKey),
		})
	}
}

// NewGlobalRateLimiterWithLogger creates a global rate limiter with logging support.
func NewGlobalRateLimiterWithLogger(r rate.Limit, b int, logger interfaces.Logger) interfaces.RateLimiter {
	return &globalRateLimiterWithLogger{
		limiter: NewGlobalRateLimiter(r, b),
		logger:  logger,
	}
}

// globalRateLimiterWithLogger wraps the GlobalRateLimiter with logging.
type globalRateLimiterWithLogger struct {
	limiter *GlobalRateLimiter
	logger  interfaces.Logger
}

// Middleware wraps the original middleware with logging.
func (r *globalRateLimiterWithLogger) Middleware(next http.Handler) http.Handler {
	originalMiddleware := r.limiter.Middleware(next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.logger != nil {
			r.logger.Debug("Global rate limit check", map[string]any{
				"path": req.URL.Path,
			})
		}
		originalMiddleware.ServeHTTP(w, req)
	})
}

// GetLimit is not applicable to the global rate limiter.
func (r *globalRateLimiterWithLogger) GetLimit(apiKey string) (allowed bool, remaining int) {
	// This is a global limiter, so per-client state doesn't exist.
	// We could return the state of the global limiter, but it's less meaningful.
	return true, r.limiter.limiter.Burst()
}

// Reset is not applicable to the global rate limiter.
func (r *globalRateLimiterWithLogger) Reset(apiKey string) {
	// Cannot reset the global limiter on a per-key basis.
	if r.logger != nil {
		r.logger.Warn("Attempted to reset global rate limiter for a single key", map[string]any{
			"api_key": utils.MaskAPIKey(apiKey),
		})
	}
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
			"api_key": utils.MaskAPIKey(apiKey),
		})
	}
}


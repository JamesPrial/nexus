package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenLimiter is a middleware that limits the number of tokens per minute.
type TokenLimiter struct {
	clients map[string]*rate.Limiter
	mu      sync.Mutex
	tpm     int     // tokens per minute (as configured)
	tps     float64 // tokens per second (converted for rate.Limiter)
	burst   int
}

// NewTokenLimiter creates a new TokenLimiter.
// tpm: tokens per minute limit
// burst: maximum burst allowance in tokens
func NewTokenLimiter(tpm, burst int) *TokenLimiter {
	// Convert tokens per minute to tokens per second for rate.Limiter
	tps := float64(tpm) / 60.0
	
	return &TokenLimiter{
		clients: make(map[string]*rate.Limiter),
		tpm:     tpm,
		tps:     tps,
		burst:   burst,
	}
}

// countTokens calculates the approximate token count for a request
func countTokens(r *http.Request) (int, error) {
	// Read request body without consuming it
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// If no body content, assign minimal token count
	if len(body) == 0 {
		return 1, nil // Minimal cost for non-body requests
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
		// For non-JSON requests, estimate based on body size
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

// Middleware is the HTTP middleware for the token limiter.
func (tl *TokenLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		tl.mu.Lock()
		limiter, exists := tl.clients[apiKey]
		if !exists {
			// Use converted tokens per second rate, not tokens per minute
			limiter = rate.NewLimiter(rate.Limit(tl.tps), tl.burst)
			tl.clients[apiKey] = limiter
		}
		tl.mu.Unlock()

		// Count tokens for the request
		tokenCount, err := countTokens(r)
		if err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		if !limiter.AllowN(time.Now(), tokenCount) {
			http.Error(w, "Token limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

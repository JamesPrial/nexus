package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tiktoken-go/tokenizer"
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

// getEncodingForModel returns the appropriate tiktoken encoding for a model
func getEncodingForModel(model string) tokenizer.Encoding {
	// Map models to their appropriate encodings
	// Reference: https://github.com/openai/tiktoken/blob/main/tiktoken/model.py
	switch {
	case strings.HasPrefix(model, "gpt-4o"):
		return tokenizer.O200kBase // GPT-4o models use o200k_base
	case strings.HasPrefix(model, "gpt-4"), strings.HasPrefix(model, "gpt-3.5-turbo"):
		return tokenizer.Cl100kBase // GPT-4 and GPT-3.5-turbo use cl100k_base
	case strings.HasPrefix(model, "text-davinci-003"), strings.HasPrefix(model, "text-davinci-002"):
		return tokenizer.P50kBase // Davinci models use p50k_base
	case strings.HasPrefix(model, "code-davinci-002"), strings.HasPrefix(model, "code-cushman-001"):
		return tokenizer.P50kBase // Code models use p50k_base
	case strings.HasPrefix(model, "text-davinci-001"), strings.HasPrefix(model, "text-curie-001"), strings.HasPrefix(model, "text-babbage-001"), strings.HasPrefix(model, "text-ada-001"):
		return tokenizer.R50kBase // Older models use r50k_base
	default:
		// Default to cl100k_base for unknown models (most common)
		return tokenizer.Cl100kBase
	}
}

// countTokens calculates the accurate token count for a request using tiktoken
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
		Prompt      string `json:"prompt"`
		MaxTokens   int    `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
	}

	// If JSON parsing fails, fall back to character-based estimation
	if err := json.Unmarshal(body, &payload); err != nil {
		// For non-JSON requests, estimate based on body size (4 chars ≈ 1 token)
		tokenCount := max(1, len(body)/4)
		return tokenCount, nil
	}

	// Get the appropriate encoding for this model
	encoding := getEncodingForModel(payload.Model)
	enc, err := tokenizer.Get(encoding)
	if err != nil {
		// Fallback to character-based estimation if tokenizer fails
		totalChars := 0
		for _, msg := range payload.Messages {
			totalChars += len(msg.Content)
		}
		totalChars += len(payload.Prompt)
		tokenCount := max(5, totalChars/4)
		return tokenCount, nil
	}

	totalTokens := 0

	// Count tokens from messages (chat completion format)
	if len(payload.Messages) > 0 {
		// Add overhead tokens for chat format
		// Every message follows <|start|>{role/name}\n{content}<|end|>\n
		totalTokens += 3 // Every reply is primed with <|start|>assistant<|message|>

		for _, message := range payload.Messages {
			totalTokens += 3 // Every message has overhead tokens
			
			// Count tokens in role
			roleTokens, _, _ := enc.Encode(message.Role)
			totalTokens += len(roleTokens)
			
			// Count tokens in content
			contentTokens, _, _ := enc.Encode(message.Content)
			totalTokens += len(contentTokens)
		}
	}

	// Count tokens from prompt (completion format)
	if payload.Prompt != "" {
		promptTokens, _, _ := enc.Encode(payload.Prompt)
		totalTokens += len(promptTokens)
	}

	// Add estimated tokens for response if max_tokens is specified
	// This is an estimate of what the response might consume
	// For rate limiting purposes, we might want to include this
	// But for now, we'll only count input tokens

	// Ensure minimum token count
	if totalTokens < 1 {
		totalTokens = 1
	}

	return totalTokens, nil
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

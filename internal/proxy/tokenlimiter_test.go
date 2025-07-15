package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenLimiter(t *testing.T) {
	// Test that token limiter properly converts minutes to seconds
	limiter := NewTokenLimiter(60, 10) // 60 tokens per minute = 1 token per second
	
	// Verify internal rate conversion
	expectedTPS := 60.0 / 60.0 // 1 token per second
	if limiter.tps != expectedTPS {
		t.Errorf("Expected tokens per second %f, got %f", expectedTPS, limiter.tps)
	}
	
	// Verify configuration values are preserved
	if limiter.tpm != 60 {
		t.Errorf("Expected tokens per minute 60, got %d", limiter.tpm)
	}
	if limiter.burst != 10 {
		t.Errorf("Expected burst 10, got %d", limiter.burst)
	}
}

func TestTokenCounting(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		expected    int
	}{
		{
			name:        "Prompt request",
			requestBody: `{"model":"gpt-4","prompt":"Hello, how are you?"}`,
			expected:    5, // len("Hello, how are you?")/4 = 19/4 ≈ 4.75 → min 5
		},
		{
			name:        "Chat request",
			requestBody: `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expected:    5, // len("Hello")/4 = 5/4 ≈ 1.25 → min 5
		},
		{
			name:        "Long prompt",
			requestBody: `{"model":"gpt-4","prompt":"` + strings.Repeat("a", 100) + `"}`,
			expected:    25, // 100/4 = 25
		},
		{
			name:        "Non-JSON request",
			requestBody: "plain text request",
			expected:    4, // len("plain text request")/4 = 18/4 = 4.5 → 4
		},
		{
			name:        "Empty body",
			requestBody: "",
			expected:    1, // Minimal token count
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test body
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.requestBody))

			tokenCount, err := countTokens(req)
			if err != nil {
				t.Fatalf("countTokens failed: %v", err)
			}

			if tokenCount != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, tokenCount)
			}
		})
	}
}

func TestTokenLimiterWithTokenCounting(t *testing.T) {
	limiter := NewTokenLimiter(60, 10) // 60 tokens/min = 1 token/sec, burst 10

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(limiter.Middleware(protectedHandler))
	defer ts.Close()

	client := &http.Client{}

	// First request (5 tokens) - should pass
	req1, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{"prompt":"Short"}`))
	req1.Header.Set("Authorization", "test-key")
	resp1, err := client.Do(req1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Errorf("First request should pass")
	}

	// Second request (10 tokens) - should fail (5+10=15 > 10 burst)
	req2, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{"prompt":"`+strings.Repeat("a", 40)+`"}`))
	req2.Header.Set("Authorization", "test-key")
	resp2, err := client.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status: %d", resp2.StatusCode)
	}

	// Third request with new key (5 tokens) - should pass
	req3, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{"prompt":"Short"}`))
	req3.Header.Set("Authorization", "new-key")
	resp3, err := client.Do(req3)
	if err != nil || resp3.StatusCode != http.StatusOK {
		t.Errorf("New key request should pass")
	}
}

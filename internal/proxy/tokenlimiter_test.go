package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tiktoken-go/tokenizer"
)

func TestGetEncodingForModel(t *testing.T) {
	tests := []struct {
		model            string
		expectedEncoding tokenizer.Encoding
	}{
		{"gpt-4o", tokenizer.O200kBase},
		{"gpt-4o-mini", tokenizer.O200kBase},
		{"gpt-4", tokenizer.Cl100kBase},
		{"gpt-4-turbo", tokenizer.Cl100kBase},
		{"gpt-3.5-turbo", tokenizer.Cl100kBase},
		{"gpt-3.5-turbo-16k", tokenizer.Cl100kBase},
		{"text-davinci-003", tokenizer.P50kBase},
		{"text-davinci-002", tokenizer.P50kBase},
		{"code-davinci-002", tokenizer.P50kBase},
		{"text-davinci-001", tokenizer.R50kBase},
		{"unknown-model", tokenizer.Cl100kBase}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			encoding := getEncodingForModel(tt.model)
			if encoding != tt.expectedEncoding {
				t.Errorf("getEncodingForModel(%s) = %v, want %v", tt.model, encoding, tt.expectedEncoding)
			}
		})
	}
}

func TestCountTokens(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		expectError bool
		minTokens   int
		maxTokens   int
	}{
		{
			name:        "Empty body",
			requestBody: "",
			expectError: false,
			minTokens:   1,
			maxTokens:   1,
		},
		{
			name:        "Invalid JSON",
			requestBody: "invalid json",
			expectError: false,
			minTokens:   1, // Should fallback to character estimation
			maxTokens:   5,
		},
		{
			name: "Simple chat completion",
			requestBody: `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello, world!"}
				]
			}`,
			expectError: false,
			minTokens:   5,  // Should be more accurate than character counting
			maxTokens:   20,
		},
		{
			name: "Multiple messages",
			requestBody: `{
				"model": "gpt-4",
				"messages": [
					{"role": "system", "content": "You are a helpful assistant."},
					{"role": "user", "content": "What is the capital of France?"},
					{"role": "assistant", "content": "The capital of France is Paris."},
					{"role": "user", "content": "What about Germany?"}
				]
			}`,
			expectError: false,
			minTokens:   20, // Multiple messages should have more tokens
			maxTokens:   60,
		},
		{
			name: "Text completion with prompt",
			requestBody: `{
				"model": "text-davinci-003",
				"prompt": "The quick brown fox jumps over the lazy dog."
			}`,
			expectError: false,
			minTokens:   8,  // Should accurately count tokens in the prompt
			maxTokens:   15,
		},
		{
			name: "GPT-4o model",
			requestBody: `{
				"model": "gpt-4o",
				"messages": [
					{"role": "user", "content": "Count tokens using o200k_base encoding"}
				]
			}`,
			expectError: false,
			minTokens:   5,
			maxTokens:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with the test body
			req, err := http.NewRequest("POST", "/v1/chat/completions", strings.NewReader(tt.requestBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Count tokens
			tokenCount, err := countTokens(req)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tokenCount < tt.minTokens || tokenCount > tt.maxTokens {
				t.Errorf("Token count %d not in expected range [%d, %d]", tokenCount, tt.minTokens, tt.maxTokens)
			}

			t.Logf("Request: %s -> %d tokens", tt.name, tokenCount)
		})
	}
}

func TestCountTokensAccuracy(t *testing.T) {
	// Test with a known string to verify accuracy
	testMessage := "The quick brown fox jumps over the lazy dog."
	
	requestBody := `{
		"model": "gpt-3.5-turbo", 
		"messages": [
			{"role": "user", "content": "` + testMessage + `"}
		]
	}`

	req, err := http.NewRequest("POST", "/v1/chat/completions", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	tokenCount, err := countTokens(req)
	if err != nil {
		t.Fatalf("countTokens failed: %v", err)
	}

	// Verify manually with tiktoken
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		t.Fatalf("Failed to get tokenizer: %v", err)
	}
	
	directTokens, _, _ := enc.Encode(testMessage)
	directCount := len(directTokens)

	// Our count should be close to the direct count (accounting for chat format overhead)
	if tokenCount < directCount || tokenCount > directCount+10 {
		t.Errorf("Token count accuracy: got %d, direct encoding gives %d", tokenCount, directCount)
	}

	t.Logf("Message: '%s'", testMessage)
	t.Logf("Direct encoding: %d tokens", directCount)
	t.Logf("Chat format count: %d tokens", tokenCount)
	t.Logf("Overhead: %d tokens", tokenCount-directCount)
}

func TestTokenLimiterIntegration(t *testing.T) {
	// Test the complete token limiter with realistic request
	limiter := NewTokenLimiter(1000, 100) // 1000 tokens per minute, 100 burst

	requestBody := `{
		"model": "gpt-3.5-turbo",
		"messages": [
			{"role": "user", "content": "Write a short poem about programming."}
		],
		"max_tokens": 50
	}`

	req, err := http.NewRequest("POST", "/v1/chat/completions", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")

	// Create a test handler
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Success"))
	}))

	// Test the request
	rr := &testResponseWriter{}
	handler.ServeHTTP(rr, req)

	if rr.statusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d. Body: %s", rr.statusCode, rr.body.String())
	}
}

// Legacy test for backward compatibility
func TestTokenLimiter(t *testing.T) {
	// Test that token limiter properly converts minutes to seconds
	limiter := NewTokenLimiter(60, 10) // 60 tokens per minute = 1 token per second
	
	// Verify internal rate conversion
	expectedTPS := 1.0 // 60 tokens per minute = 1 token per second
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

func TestTokenLimiterWithTokenCounting(t *testing.T) {
	limiter := NewTokenLimiter(1000, 50) // Higher limits for this test

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(limiter.Middleware(protectedHandler))
	defer ts.Close()

	client := &http.Client{}

	// First request with reasonable token count - should pass
	req1, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{
		"model": "gpt-3.5-turbo",
		"messages": [{"role": "user", "content": "Hello"}]
	}`))
	req1.Header.Set("Authorization", "test-key")
	resp1, err := client.Do(req1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Errorf("First request should pass, got status: %d", resp1.StatusCode)
	}

	// Request with different key - should pass (separate limits)
	req2, _ := http.NewRequest("POST", ts.URL, strings.NewReader(`{
		"model": "gpt-3.5-turbo", 
		"messages": [{"role": "user", "content": "Hello from different user"}]
	}`))
	req2.Header.Set("Authorization", "different-key")
	resp2, err := client.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusOK {
		t.Errorf("Different key request should pass, got status: %d", resp2.StatusCode)
	}
}

// Simple test response writer
type testResponseWriter struct {
	body       bytes.Buffer
	statusCode int
	headers    http.Header
}

func (w *testResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
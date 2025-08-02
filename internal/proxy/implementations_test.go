package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// mockLogger for testing
type mockLogger struct {
	logs []logEntry
	mu   sync.Mutex
}

type logEntry struct {
	level   string
	message string
	fields  map[string]any
}

func (m *mockLogger) Debug(msg string, fields map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{"debug", msg, fields})
}

func (m *mockLogger) Info(msg string, fields map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{"info", msg, fields})
}

func (m *mockLogger) Warn(msg string, fields map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{"warn", msg, fields})
}

func (m *mockLogger) Error(msg string, fields map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry{"error", msg, fields})
}

func TestDefaultTokenCounter_CountTokens(t *testing.T) {
	counter := &DefaultTokenCounter{}

	tests := []struct {
		name          string
		requestBody   string
		expectedMin   int
		expectedMax   int
		expectError   bool
	}{
		{
			name:        "empty body",
			requestBody: "",
			expectedMin: 1,
			expectedMax: 1,
		},
		{
			name: "chat completion format",
			requestBody: `{
				"model": "gpt-4",
				"messages": [
					{"role": "system", "content": "You are a helpful assistant."},
					{"role": "user", "content": "Hello, how are you?"}
				]
			}`,
			expectedMin: 10,
			expectedMax: 20,
		},
		{
			name: "completion format",
			requestBody: `{
				"model": "text-davinci-003",
				"prompt": "Once upon a time in a land far far away"
			}`,
			expectedMin: 5,
			expectedMax: 15,
		},
		{
			name: "mixed format",
			requestBody: `{
				"model": "gpt-4",
				"messages": [{"role": "user", "content": "test"}],
				"prompt": "additional prompt"
			}`,
			expectedMin: 5,
			expectedMax: 15,
		},
		{
			name:        "invalid JSON",
			requestBody: `{invalid json`,
			expectedMin: 3,
			expectedMax: 5,
		},
		{
			name:        "plain text",
			requestBody: `This is just plain text, not JSON at all`,
			expectedMin: 8,
			expectedMax: 12,
		},
		{
			name: "empty messages",
			requestBody: `{
				"model": "gpt-4",
				"messages": []
			}`,
			expectedMin: 5,
			expectedMax: 5,
		},
		{
			name: "very long content",
			requestBody: `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "` + strings.Repeat("Hello world. ", 100) + `"}
				]
			}`,
			expectedMin: 250,
			expectedMax: 350,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			tokens, err := counter.CountTokens(req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tokens < tt.expectedMin || tokens > tt.expectedMax {
				t.Errorf("Expected tokens between %d and %d, got %d", 
					tt.expectedMin, tt.expectedMax, tokens)
			}

			// Verify body can still be read
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("Failed to read body after counting: %v", err)
			}
			if string(body) != tt.requestBody {
				t.Error("Body content changed after counting")
			}
		})
	}
}

func TestHTTPProxy_ServeHTTP(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "true")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	logger := &mockLogger{}
	
	proxy := &HTTPProxy{
		ReverseProxy: httputil.NewSingleHostReverseProxy(backendURL),
		Logger:       logger,
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute
	proxy.ServeHTTP(rr, req)

	// Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("X-Backend") != "true" {
		t.Error("Expected X-Backend header from backend")
	}

	// Check logging
	if len(logger.logs) == 0 {
		t.Error("Expected debug log")
	}
}

func TestHTTPProxy_SetTarget(t *testing.T) {
	logger := &mockLogger{}
	proxy := &HTTPProxy{
		Logger: logger,
	}

	tests := []struct {
		name        string
		targetURL   string
		expectError bool
	}{
		{
			name:        "valid HTTP URL",
			targetURL:   "http://example.com",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL",
			targetURL:   "https://example.com",
			expectError: false,
		},
		{
			name:        "invalid URL",
			targetURL:   "://invalid",
			expectError: true,
		},
		{
			name:        "empty URL",
			targetURL:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := proxy.SetTarget(tt.targetURL)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify proxy was updated
				if proxy.ReverseProxy == nil {
					t.Error("Expected ReverseProxy to be set")
				}
				if proxy.target == nil {
					t.Error("Expected target to be set")
				}
				
				// Check logging
				foundLog := false
				for _, log := range logger.logs {
					if log.level == "info" && strings.Contains(log.message, "Updated proxy target") {
						foundLog = true
						break
					}
				}
				if !foundLog {
					t.Error("Expected info log for target update")
				}
			}
		})
	}
}

// TestPerClientRateLimiterWithLogger - replaced by TestPerClientRateLimiterWithLogger_Precise
// which has more precise control over rate limiting behavior

func TestGlobalRateLimiterWithLogger(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewGlobalRateLimiterWithLogger(1, 1, logger)

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// First request should pass
	req1 := httptest.NewRequest("GET", "/test", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request should pass, got status %d", rr1.Code)
	}

	// Second request should be rate limited (global)
	req2 := httptest.NewRequest("GET", "/test", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", rr2.Code)
	}

	// Test GetLimit (always returns burst for global)
	allowed, remaining := limiter.GetLimit("any-key")
	if !allowed || remaining != 1 {
		t.Errorf("GetLimit should return true, %d for global limiter", 1)
	}

	// Test Reset (should warn)
	logger.logs = nil
	limiter.Reset("any-key")
	
	foundWarn := false
	for _, log := range logger.logs {
		if log.level == "warn" {
			foundWarn = true
			break
		}
	}
	if !foundWarn {
		t.Error("Expected warning when trying to reset global limiter")
	}
}

func TestTokenLimiterWithDeps(t *testing.T) {
	logger := &mockLogger{}
	tokenCounter := &DefaultTokenCounter{}
	limiter := NewTokenLimiterWithDeps(60, 10, tokenCounter, logger) // 60 tokens/min, burst 10

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// Request without auth should fail
	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing auth, got %d", rr.Code)
	}

	// Small request should pass
	smallBody := `{"messages": [{"role": "user", "content": "Hi"}]}`
	req2 := httptest.NewRequest("POST", "/test", strings.NewReader(smallBody))
	req2.Header.Set("Authorization", "test-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Small request should pass, got status %d", rr2.Code)
	}

	// Check GetLimit
	allowed, remaining := limiter.GetLimit("test-key")
	if !allowed {
		t.Error("Should still be allowed")
	}
	if remaining <= 0 || remaining > 10 {
		t.Errorf("Unexpected remaining tokens: %d", remaining)
	}

	// Test token counting error
	req3 := httptest.NewRequest("POST", "/test", &errorReader{})
	req3.Header.Set("Authorization", "test-key")
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for token counting error, got %d", rr3.Code)
	}
}

// errorReader always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestTokenLimiterWithDeps_RateLimit(t *testing.T) {
	logger := &mockLogger{}
	tokenCounter := &DefaultTokenCounter{}
	// Very restrictive limit for testing
	limiter := NewTokenLimiterWithDeps(10, 10, tokenCounter, logger) // 10 tokens/min, burst 10

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// Large request should exhaust limit
	largeBody := `{"messages": [{"role": "user", "content": "` + strings.Repeat("word ", 50) + `"}]}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
	req.Header.Set("Authorization", "test-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// First might pass if tokens < 10
	// Second large request should definitely fail
	req2 := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
	req2.Header.Set("Authorization", "test-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit, got status %d", rr2.Code)
	}

	// Check logging
	foundWarn := false
	for _, log := range logger.logs {
		if log.level == "warn" && strings.Contains(log.message, "Token limit exceeded") {
			foundWarn = true
			break
		}
	}
	if !foundWarn {
		t.Error("Expected warning log for token limit exceeded")
	}

	// Test Reset
	limiter.Reset("test-key")
	
	// Should be able to make request again
	req3 := httptest.NewRequest("POST", "/test", strings.NewReader(smallBody))
	req3.Header.Set("Authorization", "test-key")
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("After reset should pass, got status %d", rr3.Code)
	}
}

var smallBody = `{"messages": [{"role": "user", "content": "Hi"}]}`

func TestConcurrentRateLimiting(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithLogger(10, 10, logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// Launch concurrent requests
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "test-key")
			rr := httptest.NewRecorder()
			
			handler.ServeHTTP(rr, req)
			
			if rr.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// With burst=10, we expect around 10 successes
	if successCount < 8 || successCount > 12 {
		t.Errorf("Expected around 10 successful requests, got %d", successCount)
	}
}

func BenchmarkDefaultTokenCounter(b *testing.B) {
	counter := &DefaultTokenCounter{}
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello, how are you?"}
		]
	}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
		_, _ = counter.CountTokens(req)
	}
}

func BenchmarkPerClientRateLimiter(b *testing.B) {
	limiter := NewPerClientRateLimiterWithLogger(1000, 1000, nil)
	
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	handler := limiter.Middleware(next)
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "test-key")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
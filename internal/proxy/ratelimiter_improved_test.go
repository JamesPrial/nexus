package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test rate limiter with more precise control
func TestPerClientRateLimiterWithLogger_Precise(t *testing.T) {
	logger := &mockLogger{}
	// Create limiter with very low rate and burst to ensure we hit the limit
	limiter := NewPerClientRateLimiterWithLogger(0.1, 2, logger) // 0.1 req/sec, burst 2

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// First two requests should pass due to burst
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-key")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should pass (burst), got status %d", i+1, rr.Code)
		}
	}

	// Third request should be rate limited immediately
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", rr.Code)
	}

	// Test GetLimit right after rate limiting
	allowed, remaining := limiter.GetLimit("Bearer test-key")
	// With very low rate, should have close to 0 tokens
	if allowed && remaining > 0 {
		t.Logf("Note: GetLimit shows allowed=%v, remaining=%d (tokens may have regenerated)", allowed, remaining)
	}

	// Test Reset
	limiter.Reset("Bearer test-key")
	
	// After reset, should be able to make request again
	req4 := httptest.NewRequest("GET", "/test", nil)
	req4.Header.Set("Authorization", "Bearer test-key")
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusOK {
		t.Errorf("After reset should pass, got status %d", rr4.Code)
	}

	// Verify logging occurred
	hasDebugLog := false
	hasResetLog := false
	for _, log := range logger.logs {
		if log.level == "debug" && log.message == "Per-client rate limit check" {
			hasDebugLog = true
		}
		if log.level == "info" && log.message == "Reset per-client rate limit" {
			hasResetLog = true
		}
	}
	if !hasDebugLog {
		t.Error("Expected debug logs for rate limit checks")
	}
	if !hasResetLog {
		t.Error("Expected info log for reset")
	}
}

// Test rate limiter behavior with multiple clients
func TestPerClientRateLimiterWithLogger_MultipleClients(t *testing.T) {
	logger := &mockLogger{}
	limiter := NewPerClientRateLimiterWithLogger(1, 1, logger) // 1 req/sec, burst 1

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// Client 1 makes a request
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer client1")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("Client1 first request should pass, got status %d", rr1.Code)
	}

	// Client 1 second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer client1")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Client1 second request should be rate limited, got status %d", rr2.Code)
	}

	// Client 2 first request should pass (different client)
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.Header.Set("Authorization", "Bearer client2")
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("Client2 first request should pass, got status %d", rr3.Code)
	}

	// Client 2 second request should be rate limited
	req4 := httptest.NewRequest("GET", "/test", nil)
	req4.Header.Set("Authorization", "Bearer client2")
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusTooManyRequests {
		t.Errorf("Client2 second request should be rate limited, got status %d", rr4.Code)
	}
}

// Test rate limiter token regeneration
func TestPerClientRateLimiterWithLogger_TokenRegeneration(t *testing.T) {
	logger := &mockLogger{}
	// 10 req/sec means 1 token every 100ms
	limiter := NewPerClientRateLimiterWithLogger(10, 1, logger) // 10 req/sec, burst 1

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(next)

	// First request should pass
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer test-key")
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request should pass, got status %d", rr1.Code)
	}

	// Immediate second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer test-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Immediate second request should be rate limited, got status %d", rr2.Code)
	}

	// Wait for token regeneration (150ms to be safe)
	time.Sleep(150 * time.Millisecond)

	// Third request should pass after waiting
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.Header.Set("Authorization", "Bearer test-key")
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("Request after token regeneration should pass, got status %d", rr3.Code)
	}
}
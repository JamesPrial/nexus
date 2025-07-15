package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	// Create a rate limiter that allows 1 request per second with a burst of 1.
	limiter := NewRateLimiter(rate.Limit(1), 1)

	// Create a test handler that will be protected by the rate limiter.
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create the middleware handler.
	middleware := limiter.Middleware(testHandler)

	// Create a test request with an API key.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "test-key")

	// --- Test 1: First request should be allowed ---
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	// --- Test 2: Second request immediately after should be denied ---
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status TooManyRequests, got %d", rr.Code)
	}

	// --- Test 3: Wait for the rate limiter to allow another request ---
	time.Sleep(1 * time.Second)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK after waiting, got %d", rr.Code)
	}
}

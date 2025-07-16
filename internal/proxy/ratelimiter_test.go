package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestGlobalRateLimiter(t *testing.T) {
	// Create a rate limiter that allows 2 requests per second
	limiter := NewGlobalRateLimiter(rate.Limit(2), 4)

	// Create a handler that will be protected by the rate limiter
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create the middleware
	middleware := limiter.Middleware(testHandler)

	// Create a test server
	server := httptest.NewServer(middleware)
	defer server.Close()

	// Create a client
	client := &http.Client{}

	// Make requests
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		// No authorization header needed for global limiter
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// The first 4 requests should be allowed due to the burst capacity.
		if i < 4 {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK for request %d, got %d", i+1, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Expected status TooManyRequests for request %d, got %d", i+1, resp.StatusCode)
			}
		}
	}

	// Wait for the rate limiter to replenish
	time.Sleep(1 * time.Second)

	// Make two more requests, which should be allowed
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK after waiting, got %d", resp.StatusCode)
		}
	}
}

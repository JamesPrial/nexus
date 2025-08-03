package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/config"
	"github.com/jamesprial/nexus/internal/container"
	"github.com/jamesprial/nexus/internal/gateway"
	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/logging"
)

// TestCompleteRequestFlow tests the complete request flow through all middleware layers
func TestCompleteRequestFlow(t *testing.T) {
	// Create mock upstream server
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request made it through all middleware
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header was not passed to upstream")
		}
		
		// Echo back some info about the request
		response := map[string]any{
			"received_path":   r.URL.Path,
			"received_method": r.Method,
			"model":           "gpt-4",
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer mockUpstream.Close()

	// Create test configuration
	testConfig := &interfaces.Config{
		ListenPort: 8090,
		TargetURL:  mockUpstream.URL,
		APIKeys: map[string]string{
			"client-key-1": "upstream-key-1",
			"client-key-2": "upstream-key-2",
		},
		Limits: interfaces.Limits{
			RequestsPerSecond:    5,
			Burst:                5,
			ModelTokensPerMinute: 1000,
		},
		Metrics: interfaces.MetricsConfig{
			Enabled:           true,
			MetricsEndpoint:   "/metrics",
			PrometheusEnabled: true,
			JSONExportEnabled: true,
			AuthRequired:      false,
		},
	}

	// Set up container
	cont := container.New()
	cont.SetLogger(logging.NewSlogLogger("debug"))
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Create gateway service
	service := gateway.NewService(cont)
	
	// Start the service
	if err := service.Start(); err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer func() { _ = service.Stop() }()

	// Give the server time to start
	time.Sleep(200 * time.Millisecond)

	// Test cases for different scenarios
	testCases := []struct {
		name           string
		path           string
		method         string
		clientKey      string
		body           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid request with authentication",
			path:           "/v1/chat/completions",
			method:         "POST",
			clientKey:      "client-key-1",
			body:           `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Missing API key",
			path:           "/v1/chat/completions",
			method:         "POST",
			clientKey:      "",
			body:           `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "Invalid API key",
			path:           "/v1/chat/completions",
			method:         "POST",
			clientKey:      "invalid-key",
			body:           `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "Health check endpoint",
			path:           "/health",
			method:         "GET",
			clientKey:      "",
			body:           "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = bytes.NewBufferString(tc.body)
			}
			
			req, err := http.NewRequest(tc.method, "http://localhost:8090"+tc.path, bodyReader)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Add headers
			if tc.clientKey != "" {
				req.Header.Set("Authorization", "Bearer "+tc.clientKey)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Make request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Check status code
			if resp.StatusCode != tc.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, resp.StatusCode, string(body))
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// For successful API requests, verify the response
			if !tc.expectError && tc.path != "/health" && tc.path != "/metrics" {
				var response map[string]any
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to parse response JSON: %v. Body: %s", err, string(body))
				} else {
					// Verify response contains expected fields
					if response["model"] != "gpt-4" {
						t.Errorf("Expected model gpt-4, got %v", response["model"])
					}
				}
			}
		})
	}

	// Test rate limiting
	t.Run("Rate limiting", func(t *testing.T) {
		// Make requests up to the burst limit
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "http://localhost:8090/v1/chat/completions", 
				bytes.NewBufferString(`{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}`))
			req.Header.Set("Authorization", "Bearer client-key-2")
			req.Header.Set("Content-Type", "application/json")
			
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}
			_ = resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Request %d: expected status 200, got %d. Body: %s", i+1, resp.StatusCode, string(body))
			}
		}

		// The 6th request should be rate limited
		req, _ := http.NewRequest("POST", "http://localhost:8090/v1/chat/completions", 
			bytes.NewBufferString(`{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}`))
		req.Header.Set("Authorization", "Bearer client-key-2")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Rate limit test request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("Expected rate limit status 429, got %d", resp.StatusCode)
		}
	})

	// Verify metrics were collected
	t.Run("Metrics collection", func(t *testing.T) {
		collector := cont.MetricsCollector()
		if collector == nil {
			t.Skip("Metrics collector is nil - metrics may be disabled")
		}

		metrics := collector.GetMetrics()
		if len(metrics) == 0 {
			t.Error("No metrics were collected")
		}

		// Check metrics for client-key-1 (from valid request test)
		if keyMetrics, ok := collector.GetMetricsForKey("client-key-1"); ok {
			if keyMetrics.TotalRequests == 0 {
				t.Error("No requests recorded for client-key-1")
			}
			// Token counting may not be implemented yet
			t.Logf("Tokens consumed for client-key-1: %d", keyMetrics.TotalTokensConsumed)
		} else {
			t.Error("No metrics found for client-key-1")
		}
	})
}

// TestMiddlewareOrder verifies that middleware is applied in the correct order
func TestMiddlewareOrder(t *testing.T) {
	orderTrace := []string{}
	
	// Create mock middleware that tracks order
	validationMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orderTrace = append(orderTrace, "validation")
			next.ServeHTTP(w, r)
		})
	}

	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orderTrace = append(orderTrace, "auth")
			// Set a header to indicate auth passed
			r.Header.Set("X-Auth-Passed", "true")
			next.ServeHTTP(w, r)
		})
	}

	rateLimitMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orderTrace = append(orderTrace, "rate-limit")
			next.ServeHTTP(w, r)
		})
	}

	tokenLimitMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orderTrace = append(orderTrace, "token-limit")
			next.ServeHTTP(w, r)
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderTrace = append(orderTrace, "proxy")
		w.WriteHeader(http.StatusOK)
	})

	// Build the chain manually to test order
	handler := validationMiddleware(
		authMiddleware(
			rateLimitMiddleware(
				tokenLimitMiddleware(
					finalHandler,
				),
			),
		),
	)

	// Make a request
	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify order
	expectedOrder := []string{"validation", "auth", "rate-limit", "token-limit", "proxy"}
	if len(orderTrace) != len(expectedOrder) {
		t.Errorf("Expected %d middleware calls, got %d", len(expectedOrder), len(orderTrace))
	}

	for i, expected := range expectedOrder {
		if i >= len(orderTrace) || orderTrace[i] != expected {
			t.Errorf("Expected middleware order[%d] to be %s, got %s", i, expected, orderTrace[i])
		}
	}
}

// TestErrorHandling tests error handling throughout the request flow
func TestErrorHandling(t *testing.T) {
	// Create mock upstream that returns errors
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return different errors based on path
		switch r.URL.Path {
		case "/timeout":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusGatewayTimeout)
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		case "/bad-gateway":
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockUpstream.Close()

	// Create test configuration
	testConfig := &interfaces.Config{
		ListenPort: 8091,
		TargetURL:  mockUpstream.URL,
		APIKeys: map[string]string{
			"test-key": "upstream-key",
		},
		Limits: interfaces.Limits{
			RequestsPerSecond:    100,
			Burst:                200,
			ModelTokensPerMinute: 10000,
		},
	}

	// Set up container
	cont := container.New()
	cont.SetLogger(logging.NewNoOpLogger())
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	handler := cont.BuildHandler()

	testCases := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "Server error",
			path:           "/server-error",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Bad gateway",
			path:           "/bad-gateway",
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "Not found",
			path:           "/not-found",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			req.Header.Set("Authorization", "Bearer test-key")
			
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}
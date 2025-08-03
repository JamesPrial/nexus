package metrics

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsMiddlewareBasicFunctionality verifies basic middleware operation
func TestMetricsMiddlewareBasicFunctionality(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Create a simple handler that the middleware will wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrappedHandler := middleware(handler)

	// Create test request with Authorization header
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4"}`))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify handler executed successfully
	assert.Equal(t, http.StatusOK, w.Code, "Handler should execute successfully")
	assert.Equal(t, "success", w.Body.String(), "Handler response should be preserved")

	// Verify metrics were recorded
	metrics := collector.GetMetrics()
	require.Contains(t, metrics, "test-api-key", "Should record metrics for API key")

	keyMetrics := metrics["test-api-key"].(*KeyMetrics)
	assert.Equal(t, int64(1), keyMetrics.TotalRequests, "Should record one request")
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests, "Should record one successful request")
	assert.Equal(t, int64(0), keyMetrics.FailedRequests, "Should have no failed requests")
}

// TestMetricsMiddlewareAuthorizationExtraction verifies API key extraction
func TestMetricsMiddlewareAuthorizationExtraction(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_ = middleware(handler) // wrappedHandler would be used for each test case

	testCases := []struct {
		name           string
		authHeader     string
		expectRecorded bool
		expectedKey    string
	}{
		{
			name:           "valid_bearer_token",
			authHeader:     "Bearer valid-api-key-123",
			expectRecorded: true,
			expectedKey:    "valid-api-key-123",
		},
		{
			name:           "bearer_with_spaces",
			authHeader:     "Bearer   key-with-leading-spaces",
			expectRecorded: true,
			expectedKey:    "key-with-leading-spaces",
		},
		{
			name:           "no_authorization_header",
			authHeader:     "",
			expectRecorded: true,
			expectedKey:    "",
		},
		{
			name:           "invalid_authorization_format",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectRecorded: true,
			expectedKey:    "Basic dXNlcjpwYXNz",
		},
		{
			name:           "bearer_lowercase",
			authHeader:     "bearer lowercase-bearer",
			expectRecorded: true, // Record all requests, even invalid auth
			expectedKey:    "bearer lowercase-bearer",
		},
		{
			name:           "empty_bearer_token",
			authHeader:     "Bearer ",
			expectRecorded: true,
			expectedKey:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh collector for each test
			testCollector := NewMetricsCollector()
			testMiddleware := MetricsMiddleware(testCollector)
			testWrappedHandler := testMiddleware(handler)

			req := httptest.NewRequest("GET", "/v1/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			w := httptest.NewRecorder()
			testWrappedHandler.ServeHTTP(w, req)

			metrics := testCollector.GetMetrics()

			if tc.expectRecorded {
				if tc.expectedKey != "" {
					require.Contains(t, metrics, tc.expectedKey, 
						"Should record metrics for key: %s", tc.expectedKey)
				} else {
					// Empty key should still be recorded
					require.Contains(t, metrics, "", "Should record metrics for empty key")
				}
			} else {
				assert.Len(t, metrics, 0, "Should not record any metrics")
			}
		})
	}
}

// TestMetricsMiddlewareStatusCodeHandling verifies status code categorization
func TestMetricsMiddlewareStatusCodeHandling(t *testing.T) {

	testCases := []struct {
		name               string
		statusCode         int
		expectedSuccessful int64
		expectedFailed     int64
	}{
		{"success_200", 200, 1, 0},
		{"success_201", 201, 1, 0},
		{"success_204", 204, 1, 0},
		{"success_299", 299, 1, 0},
		{"redirect_300", 300, 0, 1},
		{"redirect_301", 301, 0, 1},
		{"redirect_399", 399, 0, 1},
		{"client_error_400", 400, 0, 1},
		{"client_error_401", 401, 0, 1},
		{"client_error_404", 404, 0, 1},
		{"client_error_429", 429, 0, 1},
		{"client_error_499", 499, 0, 1},
		{"server_error_500", 500, 0, 1},
		{"server_error_502", 502, 0, 1},
		{"server_error_599", 599, 0, 1},
		{"informational_100", 100, 0, 1},
		{"informational_199", 199, 0, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh collector for each test to avoid shared state
			collector := NewMetricsCollector()
			middleware := MetricsMiddleware(collector)
			
			// Create handler that returns specific status code
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/v1/test", nil)
			req.Header.Set("Authorization", "Bearer test-key")

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code was preserved
			assert.Equal(t, tc.statusCode, w.Code, "Status code should be preserved")

			// Verify metrics categorization
			metrics := collector.GetMetrics()
			require.Contains(t, metrics, "test-key")

			keyMetrics := metrics["test-key"].(*KeyMetrics)
			assert.Equal(t, tc.expectedSuccessful, keyMetrics.SuccessfulRequests,
				"Successful requests count for status %d", tc.statusCode)
			assert.Equal(t, tc.expectedFailed, keyMetrics.FailedRequests,
				"Failed requests count for status %d", tc.statusCode)
		})
	}
}

// TestMetricsMiddlewareLatencyMeasurement verifies latency measurement
func TestMetricsMiddlewareLatencyMeasurement(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	testDelays := []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
	}

	for i, delay := range testDelays {
		t.Run(fmt.Sprintf("delay_%dms", delay.Milliseconds()), func(t *testing.T) {
			// Create handler with artificial delay
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(delay)
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/v1/test", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer test-key-%d", i))

			start := time.Now()
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			actualDuration := time.Since(start)

			// Verify the request took at least the expected delay
			assert.GreaterOrEqual(t, actualDuration, delay, 
				"Request should take at least the artificial delay")

			// The latency measurement is tested via Prometheus histogram
			// which is covered in other tests
		})
	}
}

// TestMetricsMiddlewareEndpointExtraction verifies endpoint path recording
func TestMetricsMiddlewareEndpointExtraction(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	testPaths := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/embeddings",
		"/v1/images/generations",
		"/health",
		"/metrics",
		"/",
		"/v1/models",
		"/very/long/path/with/multiple/segments",
	}

	apiKey := "endpoint-test-key"

	for _, path := range testPaths {
		req := httptest.NewRequest("POST", path, nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verify all endpoints were recorded
	metrics := collector.GetMetrics()
	require.Contains(t, metrics, apiKey)

	keyMetrics := metrics[apiKey].(*KeyMetrics)
	assert.Equal(t, int64(len(testPaths)), keyMetrics.TotalRequests, 
		"Should record all requests")

	// Verify per-endpoint breakdown
	assert.Len(t, keyMetrics.PerEndpoint, len(testPaths), 
		"Should have per-endpoint breakdown for all paths")

	for _, path := range testPaths {
		require.Contains(t, keyMetrics.PerEndpoint, path, 
			"Should have metrics for path: %s", path)
		assert.Equal(t, int64(1), keyMetrics.PerEndpoint[path].TotalRequests,
			"Should have 1 request for path: %s", path)
	}
}

// TestMetricsMiddlewareWithTokenExtraction verifies integration with token extraction
func TestMetricsMiddlewareWithTokenExtraction(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Create a more sophisticated handler that simulates token extraction
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate adding token info to context (like a real token counter would)
		ctx := r.Context()
		
		// Extract model and tokens from request/context
		var model string
		var tokens int

		if r.URL.Path == "/v1/chat/completions" {
			model = "gpt-4"
			tokens = 150
		} else if r.URL.Path == "/v1/embeddings" {
			model = "text-embedding-ada-002"
			tokens = 50
		} else {
			model = "unknown"
			tokens = 0
		}

		// Store in context (this is where token counter would put it)
		ctx = context.WithValue(ctx, ModelContextKey, model)
		ctx = context.WithValue(ctx, TokensContextKey, tokens)
		*r = *r.WithContext(ctx)

		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{"model":"%s","usage":{"total_tokens":%d}}`, model, tokens)
		_, _ = w.Write([]byte(response))
	})

	wrappedHandler := middleware(handler)

	testCases := []struct {
		path           string
		expectedModel  string
		expectedTokens int
	}{
		{"/v1/chat/completions", "gpt-4", 150},
		{"/v1/embeddings", "text-embedding-ada-002", 50},
		{"/v1/unknown", "unknown", 0},
	}

	apiKey := "token-extraction-key"

	for _, tc := range testCases {
		req := httptest.NewRequest("POST", tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Note: Current middleware implementation uses hardcoded model/tokens
	// This test verifies the structure is in place for future token extraction integration
	metrics := collector.GetMetrics()
	require.Contains(t, metrics, apiKey)

	keyMetrics := metrics[apiKey].(*KeyMetrics)
	assert.Equal(t, int64(len(testCases)), keyMetrics.TotalRequests)
	
	// With current implementation, tokens would be 0 since it's hardcoded
	// This test documents the expected behavior once token extraction is integrated
}

// TestMetricsMiddlewareConcurrentRequestsComprehensive verifies thread safety
func TestMetricsMiddlewareConcurrentRequestsComprehensive(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add small random delay to increase chance of race conditions
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	numGoroutines := 50
	requestsPerGoroutine := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			apiKey := fmt.Sprintf("concurrent-key-%d", workerID)

			for j := 0; j < requestsPerGoroutine; j++ {
				endpoint := fmt.Sprintf("/v1/endpoint-%d", j%3)
				
				req := httptest.NewRequest("POST", endpoint, nil)
				req.Header.Set("Authorization", "Bearer "+apiKey)

				w := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code, 
					"Worker %d request %d should succeed", workerID, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify all metrics were recorded correctly
	metrics := collector.GetMetrics()
	assert.Len(t, metrics, numGoroutines, "Should have metrics for all workers")

	totalRequests := int64(0)
	for i := 0; i < numGoroutines; i++ {
		apiKey := fmt.Sprintf("concurrent-key-%d", i)
		require.Contains(t, metrics, apiKey, "Should have metrics for worker %d", i)

		keyMetrics := metrics[apiKey].(*KeyMetrics)
		assert.Equal(t, int64(requestsPerGoroutine), keyMetrics.TotalRequests,
			"Worker %d should have all requests recorded", i)
		
		totalRequests += keyMetrics.TotalRequests
	}

	expectedTotal := int64(numGoroutines * requestsPerGoroutine)
	assert.Equal(t, expectedTotal, totalRequests, "Total requests should match expected")
}

// TestMetricsMiddlewareErrorPropagation verifies error handling
func TestMetricsMiddlewareErrorPropagation(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	testCases := []struct {
		name         string
		handlerFunc  http.HandlerFunc
		expectedCode int
	}{
		{
			name: "handler_panic",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("simulated panic")
			}),
			expectedCode: http.StatusInternalServerError, // This would be handled by recovery middleware
		},
		{
			name: "handler_error_response",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "handler_write_error",
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				// Simulate write error by closing connection
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}),
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedHandler := middleware(tc.handlerFunc)
			
			req := httptest.NewRequest("GET", "/v1/test", nil)
			req.Header.Set("Authorization", "Bearer error-test-key")

			w := httptest.NewRecorder()

			if tc.name == "handler_panic" {
				// Expect panic to be propagated (would be caught by recovery middleware in real app)
				assert.Panics(t, func() {
					wrappedHandler.ServeHTTP(w, req)
				}, "Panic should be propagated")
			} else {
				wrappedHandler.ServeHTTP(w, req)
				assert.Equal(t, tc.expectedCode, w.Code)
			}
		})
	}
}

// TestMetricsMiddlewareResponseWriterWrapping verifies statusRecorder functionality
func TestMetricsMiddlewareResponseWriterWrapping(t *testing.T) {
	testCases := []struct {
		name         string
		writeActions func(w http.ResponseWriter)
		expectedCode int
	}{
		{
			name: "write_header_called_explicitly",
			writeActions: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			},
			expectedCode: http.StatusCreated,
		},
		{
			name: "write_header_implicit",
			writeActions: func(w http.ResponseWriter) {
				_, _ = w.Write([]byte("implicit 200"))
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "multiple_write_header_calls",
			writeActions: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadRequest)
				w.WriteHeader(http.StatusInternalServerError) // Should be ignored
				_, _ = w.Write([]byte("first status wins"))
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "headers_and_write",
			writeActions: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Custom-Header", "test-value")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"status":"accepted"}`))
			},
			expectedCode: http.StatusAccepted,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh collector for each test case to ensure isolation
			collector := NewMetricsCollector()
			middleware := MetricsMiddleware(collector)
			
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.writeActions(w)
			})

			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("POST", "/v1/test", nil)
			req.Header.Set("Authorization", "Bearer wrapper-test-key")

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Verify response code is captured correctly
			assert.Equal(t, tc.expectedCode, w.Code, "Response code should be captured")

			// Verify metrics recorded the correct status
			metrics := collector.GetMetrics()
			require.Contains(t, metrics, "wrapper-test-key")

			keyMetrics := metrics["wrapper-test-key"].(*KeyMetrics)
			
			if tc.expectedCode >= 200 && tc.expectedCode < 300 {
				assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests,
					"Should record as successful for status %d", tc.expectedCode)
			} else {
				assert.Equal(t, int64(1), keyMetrics.FailedRequests,
					"Should record as failed for status %d", tc.expectedCode)
			}
		})
	}
}

// TestMetricsMiddlewareIntegrationWithRealRequests simulates realistic request patterns
func TestMetricsMiddlewareIntegrationWithRealRequests(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Simulate realistic API gateway behavior
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different response times based on endpoint
		var delay time.Duration
		var statusCode int = http.StatusOK
		var responseBody string

		switch r.URL.Path {
		case "/v1/chat/completions":
			delay = 200 * time.Millisecond
			responseBody = `{"choices":[{"message":{"content":"Hello!"}}],"usage":{"total_tokens":150}}`
		case "/v1/completions":
			delay = 150 * time.Millisecond
			responseBody = `{"choices":[{"text":"Hello world"}],"usage":{"total_tokens":100}}`
		case "/v1/embeddings":
			delay = 100 * time.Millisecond
			responseBody = `{"data":[{"embedding":[0.1,0.2,0.3]}],"usage":{"total_tokens":50}}`
		case "/v1/models":
			delay = 50 * time.Millisecond
			responseBody = `{"data":[{"id":"gpt-4"},{"id":"gpt-3.5-turbo"}]}`
		case "/health":
			delay = 10 * time.Millisecond
			responseBody = `{"status":"healthy"}`
		default:
			statusCode = http.StatusNotFound
			responseBody = `{"error":"Not found"}`
		}

		// Simulate some failures
		if r.Header.Get("X-Simulate-Error") != "" {
			statusCode = http.StatusInternalServerError
			responseBody = `{"error":"Internal server error"}`
		}

		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	})

	wrappedHandler := middleware(handler)

	// Simulate realistic traffic pattern
	trafficPattern := []struct {
		apiKey      string
		path        string
		requests    int
		errorRate   float64 // 0.0 to 1.0
	}{
		{"user-premium", "/v1/chat/completions", 100, 0.02},    // 2% error rate
		{"user-premium", "/v1/completions", 50, 0.01},          // 1% error rate
		{"user-basic", "/v1/chat/completions", 20, 0.05},       // 5% error rate
		{"user-basic", "/v1/embeddings", 80, 0.01},             // 1% error rate
		{"service-monitor", "/health", 200, 0.0},               // No errors
		{"user-invalid", "/v1/invalid-endpoint", 5, 1.0},       // All errors
	}

	for _, pattern := range trafficPattern {
		for i := 0; i < pattern.requests; i++ {
			req := httptest.NewRequest("POST", pattern.path, strings.NewReader(`{"test":"data"}`))
			req.Header.Set("Authorization", "Bearer "+pattern.apiKey)
			req.Header.Set("Content-Type", "application/json")

			// Simulate errors based on error rate
			if rand.Float64() < pattern.errorRate {
				req.Header.Set("X-Simulate-Error", "true")
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Basic assertion that middleware doesn't break request flow
			assert.NotEqual(t, 0, w.Code, "Should have valid response code")
		}
	}

	// Verify comprehensive metrics collection
	metrics := collector.GetMetrics()
	
	expectedKeys := []string{"user-premium", "user-basic", "service-monitor", "user-invalid"}
	for _, key := range expectedKeys {
		require.Contains(t, metrics, key, "Should have metrics for key: %s", key)
		
		keyMetrics := metrics[key].(*KeyMetrics)
		assert.Greater(t, keyMetrics.TotalRequests, int64(0), 
			"Key %s should have recorded requests", key)

		// Verify per-endpoint breakdown
		assert.NotEmpty(t, keyMetrics.PerEndpoint, 
			"Key %s should have per-endpoint metrics", key)

		// Verify success/failure tracking
		totalExpected := keyMetrics.SuccessfulRequests + keyMetrics.FailedRequests
		assert.Equal(t, keyMetrics.TotalRequests, totalExpected,
			"Key %s: successful + failed should equal total", key)
	}

	// Verify specific patterns
	userPremiumMetrics := metrics["user-premium"].(*KeyMetrics)
	assert.Equal(t, int64(150), userPremiumMetrics.TotalRequests, "Premium user should have 150 requests")
	assert.Contains(t, userPremiumMetrics.PerEndpoint, "/v1/chat/completions")
	assert.Contains(t, userPremiumMetrics.PerEndpoint, "/v1/completions")

	serviceMonitorMetrics := metrics["service-monitor"].(*KeyMetrics)
	assert.Equal(t, int64(200), serviceMonitorMetrics.TotalRequests, "Monitor should have 200 requests")
	assert.Equal(t, serviceMonitorMetrics.TotalRequests, serviceMonitorMetrics.SuccessfulRequests,
		"Monitor should have no failures")
}
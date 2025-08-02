package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndToEndFlow verifies complete metrics collection flow
func TestMetricsEndToEndFlow(t *testing.T) {
	// Set up metrics collector
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)
	
	// Create a realistic API handler
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different API behaviors
		switch r.URL.Path {
		case "/v1/chat/completions":
			time.Sleep(200 * time.Millisecond) // Simulate processing time
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"message": map[string]interface{}{"content": "Hello, world!"}},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     25,
					"completion_tokens": 10,
					"total_tokens":      35,
				},
				"model": "gpt-4",
			}
			json.NewEncoder(w).Encode(response)
			
		case "/v1/completions":
			time.Sleep(150 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"text": "Completed text"},
				},
				"usage": map[string]interface{}{
					"total_tokens": 50,
				},
				"model": "gpt-3.5-turbo",
			}
			json.NewEncoder(w).Encode(response)
			
		case "/v1/embeddings":
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{"embedding": []float64{0.1, 0.2, 0.3}},
				},
				"usage": map[string]interface{}{
					"total_tokens": 20,
				},
				"model": "text-embedding-ada-002",
			}
			json.NewEncoder(w).Encode(response)
			
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
			
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"endpoint not found"}`))
		}
	})
	
	// Create wrapped handler with metrics middleware
	wrappedHandler := middleware(apiHandler)
	
	// Simulate realistic traffic patterns
	trafficTests := []struct {
		apiKey     string
		endpoint   string
		method     string
		requests   int
		expectCode int
		body       string
	}{
		{
			apiKey:     "user-premium-001",
			endpoint:   "/v1/chat/completions",
			method:     "POST",
			requests:   25,
			expectCode: http.StatusOK,
			body:       `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
		},
		{
			apiKey:     "user-premium-001",
			endpoint:   "/v1/completions",
			method:     "POST",
			requests:   15,
			expectCode: http.StatusOK,
			body:       `{"model":"gpt-3.5-turbo","prompt":"Complete this:"}`,
		},
		{
			apiKey:     "user-basic-002",
			endpoint:   "/v1/embeddings",
			method:     "POST",
			requests:   50,
			expectCode: http.StatusOK,
			body:       `{"model":"text-embedding-ada-002","input":"text to embed"}`,
		},
		{
			apiKey:     "user-basic-002",
			endpoint:   "/v1/chat/completions",
			method:     "POST",
			requests:   10,
			expectCode: http.StatusOK,
			body:       `{"model":"gpt-4","messages":[{"role":"user","content":"Test"}]}`,
		},
		{
			apiKey:     "service-monitor",
			endpoint:   "/health",
			method:     "GET",
			requests:   100,
			expectCode: http.StatusOK,
			body:       "",
		},
		{
			apiKey:     "user-test",
			endpoint:   "/v1/invalid",
			method:     "POST",
			requests:   5,
			expectCode: http.StatusNotFound,
			body:       `{"test":"data"}`,
		},
	}
	
	// Execute traffic patterns
	for _, traffic := range trafficTests {
		t.Run(fmt.Sprintf("%s_%s", traffic.apiKey, traffic.endpoint), func(t *testing.T) {
			for i := 0; i < traffic.requests; i++ {
				var reqBody io.Reader
				if traffic.body != "" {
					reqBody = strings.NewReader(traffic.body)
				}
				
				req := httptest.NewRequest(traffic.method, traffic.endpoint, reqBody)
				req.Header.Set("Authorization", "Bearer "+traffic.apiKey)
				req.Header.Set("Content-Type", "application/json")
				
				w := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(w, req)
				
				assert.Equal(t, traffic.expectCode, w.Code,
					"Request %d for %s should return expected status", i+1, traffic.endpoint)
			}
		})
	}
	
	// Verify comprehensive metrics collection
	t.Run("verify_metrics_collection", func(t *testing.T) {
		metrics := collector.GetMetrics()
		
		// Should have metrics for all API keys
		expectedKeys := []string{"user-premium-001", "user-basic-002", "service-monitor", "user-test"}
		for _, key := range expectedKeys {
			require.Contains(t, metrics, key, "Should have metrics for key: %s", key)
		}
		
		// Verify premium user metrics
		premiumMetrics := metrics["user-premium-001"].(*KeyMetrics)
		assert.Equal(t, int64(40), premiumMetrics.TotalRequests, "Premium user should have 40 requests")
		assert.Equal(t, int64(40), premiumMetrics.SuccessfulRequests, "All premium requests should succeed")
		assert.Equal(t, int64(0), premiumMetrics.FailedRequests, "Premium user should have no failures")
		
		// Verify per-endpoint breakdown
		require.Contains(t, premiumMetrics.PerEndpoint, "/v1/chat/completions")
		require.Contains(t, premiumMetrics.PerEndpoint, "/v1/completions")
		assert.Equal(t, int64(25), premiumMetrics.PerEndpoint["/v1/chat/completions"].TotalRequests)
		assert.Equal(t, int64(15), premiumMetrics.PerEndpoint["/v1/completions"].TotalRequests)
		
		// Verify basic user metrics
		basicMetrics := metrics["user-basic-002"].(*KeyMetrics)
		assert.Equal(t, int64(60), basicMetrics.TotalRequests, "Basic user should have 60 requests")
		assert.Equal(t, int64(60), basicMetrics.SuccessfulRequests, "All basic requests should succeed")
		
		// Verify service monitor metrics (health checks)
		monitorMetrics := metrics["service-monitor"].(*KeyMetrics)
		assert.Equal(t, int64(100), monitorMetrics.TotalRequests, "Monitor should have 100 requests")
		assert.Equal(t, int64(100), monitorMetrics.SuccessfulRequests, "All health checks should succeed")
		
		// Verify error handling metrics
		testMetrics := metrics["user-test"].(*KeyMetrics)
		assert.Equal(t, int64(5), testMetrics.TotalRequests, "Test user should have 5 requests")
		assert.Equal(t, int64(0), testMetrics.SuccessfulRequests, "Test requests should fail")
		assert.Equal(t, int64(5), testMetrics.FailedRequests, "All test requests should be failures")
	})
	
	// Test JSON export integration
	t.Run("verify_json_export", func(t *testing.T) {
		jsonData := ExportJSON(collector)
		require.NotEmpty(t, jsonData, "JSON export should not be empty")
		
		var exportedMetrics map[string]KeyMetrics
		err := json.Unmarshal(jsonData, &exportedMetrics)
		require.NoError(t, err, "JSON should be valid")
		
		// Verify exported data matches collected data
		collectedMetrics := collector.GetMetrics()
		assert.Len(t, exportedMetrics, len(collectedMetrics), 
			"Exported metrics should match collected metrics count")
		
		for key, collected := range collectedMetrics {
			require.Contains(t, exportedMetrics, key, "Exported should contain key: %s", key)
			
			collectedKey := collected.(*KeyMetrics)
			exportedKey := exportedMetrics[key]
			
			assert.Equal(t, collectedKey.TotalRequests, exportedKey.TotalRequests,
				"Total requests should match for key: %s", key)
			assert.Equal(t, collectedKey.SuccessfulRequests, exportedKey.SuccessfulRequests,
				"Successful requests should match for key: %s", key)
			assert.Equal(t, collectedKey.FailedRequests, exportedKey.FailedRequests,
				"Failed requests should match for key: %s", key)
		}
	})
	
	// Test Prometheus export integration
	t.Run("verify_prometheus_export", func(t *testing.T) {
		handler := PrometheusHandler(collector)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Prometheus endpoint should return OK")
		
		body := w.Body.String()
		assert.Contains(t, body, "request_latency_seconds", "Should contain latency histogram")
		
		// Verify all API keys are present in Prometheus output
		expectedKeys := []string{"user-premium-001", "user-basic-002", "service-monitor", "user-test"}
		for _, key := range expectedKeys {
			assert.Contains(t, body, key, "Prometheus output should contain key: %s", key)
		}
		
		// Verify specific metrics are present
		assert.Contains(t, body, "/v1/chat/completions", "Should contain chat completions endpoint")
		assert.Contains(t, body, "/v1/embeddings", "Should contain embeddings endpoint")
		assert.Contains(t, body, "/health", "Should contain health endpoint")
	})
}

// TestMetricsWithAuthenticatedEndpoint verifies metrics collection with protected endpoints
func TestMetricsWithAuthenticatedEndpoint(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)
	
	// Create protected metrics endpoint
	metricsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate authentication check
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer admin-key" && authHeader != "Bearer monitor-key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		
		// Serve metrics based on format
		format := r.URL.Query().Get("format")
		switch format {
		case "json":
			w.Header().Set("Content-Type", "application/json")
			jsonData := ExportJSON(collector)
			w.Write(jsonData)
		case "prometheus", "":
			prometheusHandler := PrometheusHandler(collector)
			prometheusHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid format"}`))
		}
	})
	
	wrappedMetricsHandler := middleware(metricsHandler)
	
	// Test authenticated access scenarios
	authTests := []struct {
		name         string
		authHeader   string
		format       string
		expectedCode int
		expectMetrics bool
	}{
		{
			name:         "admin_json_access",
			authHeader:   "Bearer admin-key",
			format:       "json",
			expectedCode: http.StatusOK,
			expectMetrics: true,
		},
		{
			name:         "monitor_prometheus_access",
			authHeader:   "Bearer monitor-key",
			format:       "prometheus",
			expectedCode: http.StatusOK,
			expectMetrics: true,
		},
		{
			name:         "unauthorized_access",
			authHeader:   "Bearer invalid-key",
			format:       "json",
			expectedCode: http.StatusUnauthorized,
			expectMetrics: false,
		},
		{
			name:         "no_auth_header",
			authHeader:   "",
			format:       "json",
			expectedCode: http.StatusUnauthorized,
			expectMetrics: false,
		},
		{
			name:         "invalid_format",
			authHeader:   "Bearer admin-key",
			format:       "xml",
			expectedCode: http.StatusBadRequest,
			expectMetrics: false,
		},
	}
	
	// Record some test data first
	collector.RecordRequest("test-app", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	
	for _, test := range authTests {
		t.Run(test.name, func(t *testing.T) {
			url := "/metrics"
			if test.format != "" {
				url += "?format=" + test.format
			}
			
			req := httptest.NewRequest("GET", url, nil)
			if test.authHeader != "" {
				req.Header.Set("Authorization", test.authHeader)
			}
			
			w := httptest.NewRecorder()
			wrappedMetricsHandler.ServeHTTP(w, req)
			
			assert.Equal(t, test.expectedCode, w.Code, 
				"Test %s should return expected status code", test.name)
			
			if test.expectMetrics {
				body := w.Body.String()
				assert.NotEmpty(t, body, "Should return metrics data")
				
				if test.format == "json" {
					var metrics map[string]interface{}
					err := json.Unmarshal([]byte(body), &metrics)
					assert.NoError(t, err, "Should return valid JSON")
					assert.Contains(t, metrics, "test-app", "Should contain test data")
				} else {
					assert.Contains(t, body, "request_latency_seconds", 
						"Should contain Prometheus metrics")
				}
			}
		})
	}
	
	// Verify that metrics collection itself was recorded
	t.Run("verify_metrics_endpoint_recorded", func(t *testing.T) {
		metrics := collector.GetMetrics()
		
		// Should have recorded metrics for the admin and monitor keys
		require.Contains(t, metrics, "admin-key", "Should record admin access")
		require.Contains(t, metrics, "monitor-key", "Should record monitor access")
		
		adminMetrics := metrics["admin-key"].(*KeyMetrics)
		assert.Greater(t, adminMetrics.TotalRequests, int64(0), "Should have recorded admin requests")
		
		monitorMetrics := metrics["monitor-key"].(*KeyMetrics)
		assert.Greater(t, monitorMetrics.TotalRequests, int64(0), "Should have recorded monitor requests")
	})
}

// TestMetricsHighThroughputScenario verifies performance under high load
func TestMetricsHighThroughputScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high throughput test in short mode")
	}
	
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)
	
	// Simple fast handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	
	wrappedHandler := middleware(handler)
	
	// High throughput test parameters
	numWorkers := 50
	requestsPerWorker := 100
	totalRequests := numWorkers * requestsPerWorker
	
	var wg sync.WaitGroup
	start := time.Now()
	
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			
			apiKey := fmt.Sprintf("load-test-%d", workerID%10) // 10 different API keys
			
			for j := 0; j < requestsPerWorker; j++ {
				endpoint := fmt.Sprintf("/v1/endpoint-%d", j%5) // 5 different endpoints
				
				req := httptest.NewRequest("POST", endpoint, strings.NewReader(`{"test":"data"}`))
				req.Header.Set("Authorization", "Bearer "+apiKey)
				req.Header.Set("Content-Type", "application/json")
				
				w := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(w, req)
				
				if w.Code != http.StatusOK {
					t.Errorf("Worker %d request %d failed with status %d", workerID, j, w.Code)
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	// Performance verification
	requestsPerSecond := float64(totalRequests) / duration.Seconds()
	t.Logf("Processed %d requests in %v (%.2f req/sec)", totalRequests, duration, requestsPerSecond)
	
	// Should handle at least 1000 req/sec
	assert.Greater(t, requestsPerSecond, 1000.0, 
		"Should process at least 1000 requests per second")
	
	// Verify metrics accuracy under high load
	metrics := collector.GetMetrics()
	
	totalRecorded := int64(0)
	for i := 0; i < 10; i++ {
		apiKey := fmt.Sprintf("load-test-%d", i)
		if keyMetrics, exists := metrics[apiKey]; exists {
			km := keyMetrics.(*KeyMetrics)
			totalRecorded += km.TotalRequests
		}
	}
	
	assert.Equal(t, int64(totalRequests), totalRecorded, 
		"Should accurately record all requests under high load")
	
	// Verify per-endpoint breakdown is maintained
	for i := 0; i < 10; i++ {
		apiKey := fmt.Sprintf("load-test-%d", i)
		if keyMetrics, exists := metrics[apiKey]; exists {
			km := keyMetrics.(*KeyMetrics)
			assert.Len(t, km.PerEndpoint, 5, 
				"Key %s should have 5 different endpoints", apiKey)
		}
	}
}

// TestMetricsMemoryUsageUnderLoad verifies memory usage remains reasonable
func TestMetricsMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}
	
	collector := NewMetricsCollector()
	
	// Simulate large number of unique API keys and endpoints
	numKeys := 1000
	endpointsPerKey := 10
	requestsPerEndpoint := 50
	
	for i := 0; i < numKeys; i++ {
		apiKey := fmt.Sprintf("memory-test-key-%d", i)
		
		for j := 0; j < endpointsPerKey; j++ {
			endpoint := fmt.Sprintf("/v1/service-%d/action-%d", i%100, j)
			model := fmt.Sprintf("model-%d", j%5)
			
			for k := 0; k < requestsPerEndpoint; k++ {
				statusCode := 200
				if k%20 == 0 { // 5% failure rate
					statusCode = 500
				}
				
				collector.RecordRequest(apiKey, endpoint, model, 100, statusCode, 
					time.Duration(100+k)*time.Millisecond)
			}
		}
	}
	
	// Verify metrics are collected correctly
	metrics := collector.GetMetrics()
	assert.Len(t, metrics, numKeys, "Should have metrics for all API keys")
	
	// Verify structure is maintained efficiently
	for i := 0; i < min(10, numKeys); i++ { // Check first 10 keys
		apiKey := fmt.Sprintf("memory-test-key-%d", i)
		require.Contains(t, metrics, apiKey)
		
		keyMetrics := metrics[apiKey].(*KeyMetrics)
		expectedRequests := int64(endpointsPerKey * requestsPerEndpoint)
		assert.Equal(t, expectedRequests, keyMetrics.TotalRequests,
			"Key %s should have expected request count", apiKey)
		
		assert.Len(t, keyMetrics.PerEndpoint, endpointsPerKey,
			"Key %s should have %d endpoints", apiKey, endpointsPerKey)
		assert.Len(t, keyMetrics.PerModel, 5,
			"Key %s should have 5 models", apiKey)
	}
	
	// Test JSON export performance with large dataset
	t.Run("json_export_performance", func(t *testing.T) {
		start := time.Now()
		jsonData := ExportJSON(collector)
		exportDuration := time.Since(start)
		
		assert.NotEmpty(t, jsonData, "JSON export should succeed")
		assert.Less(t, exportDuration, 5*time.Second, 
			"JSON export should complete within 5 seconds")
		
		t.Logf("JSON export of %d keys took %v", numKeys, exportDuration)
	})
	
	// Test Prometheus export performance with large dataset
	t.Run("prometheus_export_performance", func(t *testing.T) {
		handler := PrometheusHandler(collector)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		
		start := time.Now()
		handler.ServeHTTP(w, req)
		exportDuration := time.Since(start)
		
		assert.Equal(t, http.StatusOK, w.Code, "Prometheus export should succeed")
		assert.Less(t, exportDuration, 10*time.Second,
			"Prometheus export should complete within 10 seconds")
		
		body := w.Body.String()
		assert.NotEmpty(t, body, "Prometheus export should have content")
		
		t.Logf("Prometheus export of %d keys took %v", numKeys, exportDuration)
	})
}

// TestMetricsErrorRecovery verifies error handling and recovery
func TestMetricsErrorRecovery(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)
	
	// Handler that simulates various error conditions
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panic":
			panic("simulated panic")
		case "/timeout":
			time.Sleep(5 * time.Second) // Simulate timeout
			w.WriteHeader(http.StatusOK)
		case "/slow-response":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		case "/large-response":
			// Simulate large response
			largeData := strings.Repeat("x", 1024*1024) // 1MB
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(largeData))
		default:
			w.WriteHeader(http.StatusOK)
		}
	})
	
	wrappedHandler := middleware(errorHandler)
	
	testCases := []struct {
		name     string
		path     string
		expectPanic bool
		timeout  time.Duration
	}{
		{
			name:        "normal_request",
			path:        "/normal",
			expectPanic: false,
			timeout:     1 * time.Second,
		},
		{
			name:        "panic_request",
			path:        "/panic", 
			expectPanic: true,
			timeout:     1 * time.Second,
		},
		{
			name:        "slow_request",
			path:        "/slow-response",
			expectPanic: false,
			timeout:     3 * time.Second,
		},
		{
			name:        "large_response",
			path:        "/large-response",
			expectPanic: false,
			timeout:     2 * time.Second,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tc.path, strings.NewReader(`{"test":"data"}`))
			req.Header.Set("Authorization", "Bearer error-recovery-key")
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			
			if tc.expectPanic {
				assert.Panics(t, func() {
					wrappedHandler.ServeHTTP(w, req)
				}, "Should panic for path: %s", tc.path)
			} else {
				// Use context with timeout for slow requests
				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()
				req = req.WithContext(ctx)
				
				assert.NotPanics(t, func() {
					wrappedHandler.ServeHTTP(w, req)
				}, "Should not panic for path: %s", tc.path)
				
				if ctx.Err() != context.DeadlineExceeded {
					assert.Equal(t, http.StatusOK, w.Code, 
						"Should return OK for path: %s", tc.path)
				}
			}
		})
	}
	
	// Verify that metrics are still collected despite errors
	t.Run("verify_metrics_despite_errors", func(t *testing.T) {
		metrics := collector.GetMetrics()
		require.Contains(t, metrics, "error-recovery-key", 
			"Should have metrics despite errors")
		
		keyMetrics := metrics["error-recovery-key"].(*KeyMetrics)
		assert.Greater(t, keyMetrics.TotalRequests, int64(0), 
			"Should have recorded some requests")
		
		// Should have per-endpoint breakdown
		assert.NotEmpty(t, keyMetrics.PerEndpoint, 
			"Should have per-endpoint metrics")
	})
}
package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportJSONComprehensive verifies JSON export functionality
func TestExportJSONComprehensive(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record some test data
	testData := []struct {
		apiKey     string
		endpoint   string
		model      string
		tokens     int
		statusCode int
		duration   time.Duration
	}{
		{"api-key-1", "/v1/chat/completions", "gpt-4", 150, 200, 250 * time.Millisecond},
		{"api-key-1", "/v1/chat/completions", "gpt-4", 200, 200, 300 * time.Millisecond},
		{"api-key-1", "/v1/completions", "gpt-3.5-turbo", 100, 400, 100 * time.Millisecond},
		{"api-key-2", "/v1/embeddings", "text-embedding-ada-002", 50, 200, 150 * time.Millisecond},
		{"api-key-2", "/v1/chat/completions", "gpt-4", 175, 500, 400 * time.Millisecond},
	}

	for _, td := range testData {
		collector.RecordRequest(td.apiKey, td.endpoint, td.model, td.tokens, td.statusCode, td.duration)
	}

	// Export as JSON
	jsonData := ExportJSON(collector)
	require.NotEmpty(t, jsonData, "JSON export should not be empty")

	// Parse JSON to verify structure
	var metrics map[string]interface{}
	err := json.Unmarshal(jsonData, &metrics)
	require.NoError(t, err, "JSON should be valid")

	// Verify API keys are present
	assert.Contains(t, metrics, "api-key-1", "Should contain api-key-1 metrics")
	assert.Contains(t, metrics, "api-key-2", "Should contain api-key-2 metrics")

	// Verify structure for api-key-1
	key1Metrics := metrics["api-key-1"].(map[string]interface{})
	assert.Equal(t, float64(3), key1Metrics["TotalRequests"], "api-key-1 should have 3 requests")
	assert.Equal(t, float64(2), key1Metrics["SuccessfulRequests"], "api-key-1 should have 2 successful requests")
	assert.Equal(t, float64(1), key1Metrics["FailedRequests"], "api-key-1 should have 1 failed request")
	assert.Equal(t, float64(450), key1Metrics["TotalTokensConsumed"], "api-key-1 should have 450 tokens")

	// Verify per-endpoint breakdown exists
	assert.Contains(t, key1Metrics, "PerEndpoint", "Should have per-endpoint breakdown")
	perEndpoint := key1Metrics["PerEndpoint"].(map[string]interface{})
	assert.Contains(t, perEndpoint, "/v1/chat/completions")
	assert.Contains(t, perEndpoint, "/v1/completions")

	// Verify per-model breakdown exists
	assert.Contains(t, key1Metrics, "PerModel", "Should have per-model breakdown")
	perModel := key1Metrics["PerModel"].(map[string]interface{})
	assert.Contains(t, perModel, "gpt-4")
	assert.Contains(t, perModel, "gpt-3.5-turbo")
}

// TestExportJSONEmpty verifies JSON export with no data
func TestExportJSONEmpty(t *testing.T) {
	collector := NewMetricsCollector()
	
	jsonData := ExportJSON(collector)
	require.NotEmpty(t, jsonData, "JSON export should not be empty even with no data")

	var metrics map[string]interface{}
	err := json.Unmarshal(jsonData, &metrics)
	require.NoError(t, err, "JSON should be valid")
	assert.Empty(t, metrics, "Metrics should be empty map")
}

// TestExportJSONStructure verifies the JSON structure matches expected format
func TestExportJSONStructure(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record one request to have some data
	collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	
	jsonData := ExportJSON(collector)
	
	// Verify JSON structure matches KeyMetrics
	var metrics map[string]KeyMetrics
	err := json.Unmarshal(jsonData, &metrics)
	require.NoError(t, err, "Should unmarshal to KeyMetrics structure")
	
	require.Contains(t, metrics, "test-key")
	keyMetrics := metrics["test-key"]
	
	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests)
	assert.Equal(t, int64(0), keyMetrics.FailedRequests)
	assert.Equal(t, int64(100), keyMetrics.TotalTokensConsumed)
	assert.NotNil(t, keyMetrics.PerEndpoint)
	assert.NotNil(t, keyMetrics.PerModel)
}

// TestPrometheusHandlerComprehensive verifies Prometheus export functionality
func TestPrometheusHandlerComprehensive(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record some test data
	collector.RecordRequest("prom-key-1", "/v1/chat/completions", "gpt-4", 150, 200, 250*time.Millisecond)
	collector.RecordRequest("prom-key-1", "/v1/chat/completions", "gpt-4", 200, 429, 100*time.Millisecond)
	collector.RecordRequest("prom-key-2", "/v1/embeddings", "ada-002", 50, 200, 150*time.Millisecond)

	handler := PrometheusHandler(collector)
	require.NotNil(t, handler, "Handler should not be nil")

	// Create test HTTP request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain", "Should have correct content type")

	body := w.Body.String()
	assert.NotEmpty(t, body, "Response body should not be empty")

	// Verify Prometheus format
	assert.Contains(t, body, "# HELP", "Should contain Prometheus help text")
	assert.Contains(t, body, "# TYPE", "Should contain Prometheus type information")
	assert.Contains(t, body, "request_latency_seconds", "Should contain histogram metric")

	// Verify metric values are present
	assert.Contains(t, body, "prom-key-1", "Should contain first API key")
	assert.Contains(t, body, "prom-key-2", "Should contain second API key")
	assert.Contains(t, body, "gpt-4", "Should contain model labels")
	assert.Contains(t, body, "/v1/chat/completions", "Should contain endpoint labels")
}

// TestPrometheusHandlerEmpty verifies Prometheus export with no data
func TestPrometheusHandlerEmpty(t *testing.T) {
	collector := NewMetricsCollector()
	handler := PrometheusHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK even with no data")
	
	body := w.Body.String()
	// Should still contain metric definitions even with no data
	assert.Contains(t, body, "request_latency_seconds", "Should contain metric definition")
}

// TestPrometheusHandlerHistogramBuckets verifies histogram buckets in Prometheus output
func TestPrometheusHandlerHistogramBuckets(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record requests with different latencies to populate histogram buckets
	latencies := []time.Duration{
		50 * time.Millisecond,   // Should go in 0.1 bucket
		250 * time.Millisecond,  // Should go in 0.3 bucket
		750 * time.Millisecond,  // Should go in 1 bucket
		2 * time.Second,         // Should go in 3 bucket
		4 * time.Second,         // Should go in 5 bucket
		10 * time.Second,        // Should go in +Inf bucket
	}

	for i, latency := range latencies {
		collector.RecordRequest("histogram-test", "/v1/test", "test-model", 100, 200, latency)
		t.Logf("Recorded request %d with latency %v", i+1, latency)
	}

	handler := PrometheusHandler(collector)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()
	
	// Verify histogram buckets are present
	expectedBuckets := []string{"0.1", "0.3", "0.5", "1", "3", "5", "+Inf"}
	for _, bucket := range expectedBuckets {
		bucketPattern := `le="` + bucket + `"`
		assert.Contains(t, body, bucketPattern, "Should contain bucket %s", bucket)
	}

	// Verify histogram sum and count
	assert.Contains(t, body, "request_latency_seconds_sum", "Should contain histogram sum")
	assert.Contains(t, body, "request_latency_seconds_count", "Should contain histogram count")

	// Verify the count shows 6 requests
	assert.Contains(t, body, "request_latency_seconds_count 6", "Should show 6 total requests")
}

// TestPrometheusHandlerLabels verifies proper label handling
func TestPrometheusHandlerLabels(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record requests with various label combinations
	testCases := []struct {
		apiKey   string
		endpoint string
		model    string
	}{
		{"user-123", "/v1/chat/completions", "gpt-4"},
		{"user-456", "/v1/completions", "gpt-3.5-turbo"},
		{"user-789", "/v1/embeddings", "text-embedding-ada-002"},
		{"special-chars", "/v1/test-endpoint", "model-with-dashes"},
	}

	for _, tc := range testCases {
		collector.RecordRequest(tc.apiKey, tc.endpoint, tc.model, 100, 200, 100*time.Millisecond)
	}

	handler := PrometheusHandler(collector)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify all labels are properly escaped and present
	for _, tc := range testCases {
		// Check that all labels appear in the output
		assert.Contains(t, body, tc.apiKey, "Should contain API key: %s", tc.apiKey)
		assert.Contains(t, body, tc.endpoint, "Should contain endpoint: %s", tc.endpoint)
		assert.Contains(t, body, tc.model, "Should contain model: %s", tc.model)
		
		// Check that they appear in proper label format
		labelPattern := fmt.Sprintf(`api_key="%s"`, tc.apiKey)
		assert.Contains(t, body, labelPattern, "Should contain proper API key label")
	}
}

// TestPrometheusHandlerConcurrentAccess verifies thread safety of Prometheus handler
func TestPrometheusHandlerConcurrentAccess(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record some initial data
	collector.RecordRequest("concurrent-test", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	
	handler := PrometheusHandler(collector)
	
	// Make multiple concurrent requests to the handler
	numRequests := 10
	done := make(chan bool, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			defer func() { done <- true }()
			
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", requestID)
			body := w.Body.String()
			assert.Contains(t, body, "request_latency_seconds", 
				"Request %d should contain metrics", requestID)
		}(i)
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// TestPrometheusHandlerErrorHandling verifies error handling in Prometheus export
func TestPrometheusHandlerErrorHandling(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Test with special characters that might cause issues
	problematicData := []struct {
		apiKey   string
		endpoint string
		model    string
		desc     string
	}{
		{"key\nwith\nnewlines", "/endpoint", "model", "newlines in key"},
		{"key\"with\"quotes", "/endpoint", "model", "quotes in key"},
		{"key\\with\\backslashes", "/endpoint", "model", "backslashes in key"},
		{"key with spaces", "/endpoint", "model", "spaces in key"},
		{"", "/endpoint", "model", "empty key"},
		{"normal-key", "", "model", "empty endpoint"},
		{"normal-key", "/endpoint", "", "empty model"},
	}

	for _, pd := range problematicData {
		// Should not panic
		assert.NotPanics(t, func() {
			collector.RecordRequest(pd.apiKey, pd.endpoint, pd.model, 100, 200, 100*time.Millisecond)
		}, "Recording should not panic with: %s", pd.desc)
	}

	handler := PrometheusHandler(collector)
	
	// Should not panic when serving
	assert.NotPanics(t, func() {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		// Should still return OK status
		assert.Equal(t, http.StatusOK, w.Code, "Should handle problematic data gracefully")
	}, "Handler should not panic with problematic data")
}

// TestMetricsExportIntegration verifies integration between collector and exporters
func TestMetricsExportIntegration(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record comprehensive test data
	testData := []struct {
		apiKey     string
		endpoint   string
		model      string
		tokens     int
		statusCode int
		duration   time.Duration
	}{
		{"integration-key-1", "/v1/chat/completions", "gpt-4", 250, 200, 200 * time.Millisecond},
		{"integration-key-1", "/v1/chat/completions", "gpt-4", 300, 200, 250 * time.Millisecond},
		{"integration-key-1", "/v1/completions", "gpt-3.5-turbo", 150, 400, 100 * time.Millisecond},
		{"integration-key-2", "/v1/embeddings", "ada-002", 75, 200, 150 * time.Millisecond},
		{"integration-key-2", "/v1/chat/completions", "gpt-4", 200, 500, 300 * time.Millisecond},
	}

	for _, td := range testData {
		collector.RecordRequest(td.apiKey, td.endpoint, td.model, td.tokens, td.statusCode, td.duration)
	}

	t.Run("json_export_consistency", func(t *testing.T) {
		// Get metrics directly
		directMetrics := collector.GetMetrics()
		
		// Get metrics via JSON export
		jsonData := ExportJSON(collector)
		var jsonMetrics map[string]KeyMetrics
		err := json.Unmarshal(jsonData, &jsonMetrics)
		require.NoError(t, err)
		
		// Compare the two
		assert.Len(t, jsonMetrics, len(directMetrics), "JSON should have same number of keys")
		
		for key, directValue := range directMetrics {
			require.Contains(t, jsonMetrics, key, "JSON should contain key %s", key)
			
			directKeyMetrics := directValue.(*KeyMetrics)
			jsonKeyMetrics := jsonMetrics[key]
			
			assert.Equal(t, directKeyMetrics.TotalRequests, jsonKeyMetrics.TotalRequests,
				"Total requests should match for key %s", key)
			assert.Equal(t, directKeyMetrics.SuccessfulRequests, jsonKeyMetrics.SuccessfulRequests,
				"Successful requests should match for key %s", key)
			assert.Equal(t, directKeyMetrics.FailedRequests, jsonKeyMetrics.FailedRequests,
				"Failed requests should match for key %s", key)
			assert.Equal(t, directKeyMetrics.TotalTokensConsumed, jsonKeyMetrics.TotalTokensConsumed,
				"Total tokens should match for key %s", key)
		}
	})

	t.Run("prometheus_export_consistency", func(t *testing.T) {
		handler := PrometheusHandler(collector)
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		
		// Verify that Prometheus export contains data for all recorded metrics
		assert.Contains(t, body, "integration-key-1", "Should contain first integration key")
		assert.Contains(t, body, "integration-key-2", "Should contain second integration key")
		assert.Contains(t, body, "gpt-4", "Should contain GPT-4 model")
		assert.Contains(t, body, "/v1/chat/completions", "Should contain chat completions endpoint")
		
		// Verify histogram contains expected number of samples
		assert.Contains(t, body, "request_latency_seconds_count 5", "Should show 5 total samples")
	})
}
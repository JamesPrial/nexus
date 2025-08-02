package metrics

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsCollectorInterface verifies the collector implements required interfaces
func TestMetricsCollectorInterface(t *testing.T) {
	collector := NewMetricsCollector()

	// Test that collector implements prometheus.Collector
	var _ prometheus.Collector = collector

	// Test that collector can be used with prometheus registry
	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(collector))

	// Test collection without panics
	desc := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(desc)
		collector.Describe(desc)
	}()

	// Collect all descriptions
	descs := make([]*prometheus.Desc, 0)
	for d := range desc {
		descs = append(descs, d)
	}

	assert.NotEmpty(t, descs, "Collector should provide metric descriptions")
}

// TestMetricsCollectorREDMethod verifies RED method metrics collection
func TestMetricsCollectorREDMethod(t *testing.T) {
	collector := NewMetricsCollector()
	apiKey := "test-key-123"
	endpoint := "/v1/chat/completions"
	model := "gpt-4"

	tests := []struct {
		name       string
		statusCode int
		duration   time.Duration
		tokens     int
		isSuccess  bool
	}{
		{"successful_request", 200, 150 * time.Millisecond, 100, true},
		{"successful_request_2", 201, 200 * time.Millisecond, 150, true},
		{"client_error", 400, 50 * time.Millisecond, 0, false},
		{"server_error", 500, 75 * time.Millisecond, 0, false},
		{"rate_limit_error", 429, 25 * time.Millisecond, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector.RecordRequest(apiKey, endpoint, model, tt.tokens, tt.statusCode, tt.duration)
		})
	}

	// Verify aggregated metrics
	metrics := collector.GetMetrics()
	require.Contains(t, metrics, apiKey)

	keyMetrics := metrics[apiKey].(*KeyMetrics)
	
	// Rate: Total requests
	assert.Equal(t, int64(5), keyMetrics.TotalRequests, "Should track total request rate")
	
	// Errors: Failed vs successful requests
	assert.Equal(t, int64(2), keyMetrics.SuccessfulRequests, "Should track successful requests")
	assert.Equal(t, int64(3), keyMetrics.FailedRequests, "Should track failed requests")
	
	// Duration: Should be tracked in histogram (verified via Prometheus collection)
	// Token consumption
	assert.Equal(t, int64(250), keyMetrics.TotalTokensConsumed, "Should track total tokens")

	// Per-endpoint metrics
	require.Contains(t, keyMetrics.PerEndpoint, endpoint)
	endpointMetrics := keyMetrics.PerEndpoint[endpoint]
	assert.Equal(t, int64(5), endpointMetrics.TotalRequests)
	assert.Equal(t, int64(250), endpointMetrics.TotalTokens)

	// Per-model metrics
	require.Contains(t, keyMetrics.PerModel, model)
	modelMetrics := keyMetrics.PerModel[model]
	assert.Equal(t, int64(5), modelMetrics.TotalRequests)
	assert.Equal(t, int64(250), modelMetrics.TotalTokens)
}

// TestMetricsCollectorHistogramLatency verifies latency histogram functionality
func TestMetricsCollectorHistogramLatency(t *testing.T) {
	collector := NewMetricsCollector()
	apiKey := "histogram-test-key"
	endpoint := "/v1/completions"
	model := "gpt-3.5-turbo"

	// Record requests with different latencies
	latencies := []time.Duration{
		50 * time.Millisecond,   // Bucket: 0.1
		250 * time.Millisecond,  // Bucket: 0.3
		750 * time.Millisecond,  // Bucket: 1
		2 * time.Second,         // Bucket: 3
		4 * time.Second,         // Bucket: 5
		10 * time.Second,        // Above all buckets
	}

	for i, latency := range latencies {
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, latency)
		t.Logf("Recorded request %d with latency %v", i+1, latency)
	}

	// Collect metrics from Prometheus histogram
	metricFamilies, err := collectMetricFamilies(collector)
	require.NoError(t, err)

	// Find the histogram metric
	var histogramMetric *dto.MetricFamily
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "request_latency_seconds") {
			histogramMetric = mf
			break
		}
	}

	require.NotNil(t, histogramMetric, "Should find latency histogram metric")
	require.Len(t, histogramMetric.GetMetric(), 1, "Should have one metric series for our labels")

	histogram := histogramMetric.GetMetric()[0].GetHistogram()
	require.NotNil(t, histogram, "Should be histogram type")

	// Verify histogram buckets contain expected counts
	buckets := histogram.GetBucket()
	assert.True(t, len(buckets) > 0, "Should have histogram buckets")

	// Verify total sample count
	assert.Equal(t, uint64(6), histogram.GetSampleCount(), "Should have 6 samples")

	// Verify sum is reasonable (all latencies added up)
	expectedSum := float64(50+250+750+2000+4000+10000) / 1000.0 // Convert to seconds
	assert.InDelta(t, expectedSum, histogram.GetSampleSum(), 0.1, "Sum should match total latency")
}

// TestMetricsCollectorMultipleKeys verifies per-API-key isolation
func TestMetricsCollectorMultipleKeys(t *testing.T) {
	collector := NewMetricsCollector()

	testCases := []struct {
		apiKey   string
		requests int
		tokens   int
	}{
		{"user1-key", 10, 1000},
		{"user2-key", 5, 500},
		{"user3-key", 20, 2000},
	}

	// Record requests for different API keys
	for _, tc := range testCases {
		for i := 0; i < tc.requests; i++ {
			tokensPerRequest := tc.tokens / tc.requests
			collector.RecordRequest(tc.apiKey, "/v1/chat/completions", "gpt-4", 
				tokensPerRequest, 200, 100*time.Millisecond)
		}
	}

	metrics := collector.GetMetrics()

	// Verify each key has separate metrics
	for _, tc := range testCases {
		require.Contains(t, metrics, tc.apiKey, "Should have metrics for key %s", tc.apiKey)
		
		keyMetrics := metrics[tc.apiKey].(*KeyMetrics)
		assert.Equal(t, int64(tc.requests), keyMetrics.TotalRequests, 
			"Key %s should have %d requests", tc.apiKey, tc.requests)
		assert.Equal(t, int64(tc.tokens), keyMetrics.TotalTokensConsumed,
			"Key %s should have %d tokens", tc.apiKey, tc.tokens)
	}

	// Verify keys don't interfere with each other
	assert.Len(t, metrics, len(testCases), "Should have exactly the expected number of keys")
}

// TestMetricsCollectorMultipleEndpoints verifies per-endpoint breakdown
func TestMetricsCollectorMultipleEndpoints(t *testing.T) {
	collector := NewMetricsCollector()
	apiKey := "multi-endpoint-key"

	endpoints := []struct {
		path     string
		requests int
		tokens   int
	}{
		{"/v1/chat/completions", 15, 1500},
		{"/v1/completions", 10, 800},
		{"/v1/embeddings", 25, 500},
		{"/v1/images/generations", 5, 0},
	}

	// Record requests for different endpoints
	for _, ep := range endpoints {
		for i := 0; i < ep.requests; i++ {
			tokensPerRequest := 0
			if ep.requests > 0 {
				tokensPerRequest = ep.tokens / ep.requests
			}
			collector.RecordRequest(apiKey, ep.path, "gpt-4", tokensPerRequest, 200, 100*time.Millisecond)
		}
	}

	metrics := collector.GetMetrics()
	require.Contains(t, metrics, apiKey)

	keyMetrics := metrics[apiKey].(*KeyMetrics)

	// Verify per-endpoint breakdown
	for _, ep := range endpoints {
		require.Contains(t, keyMetrics.PerEndpoint, ep.path, "Should have metrics for endpoint %s", ep.path)
		
		endpointMetrics := keyMetrics.PerEndpoint[ep.path]
		assert.Equal(t, int64(ep.requests), endpointMetrics.TotalRequests,
			"Endpoint %s should have %d requests", ep.path, ep.requests)
		assert.Equal(t, int64(ep.tokens), endpointMetrics.TotalTokens,
			"Endpoint %s should have %d tokens", ep.path, ep.tokens)
	}

	// Verify total aggregation
	totalRequests := int64(15 + 10 + 25 + 5)
	totalTokens := int64(1500 + 800 + 500 + 0)
	assert.Equal(t, totalRequests, keyMetrics.TotalRequests, "Should aggregate total requests")
	assert.Equal(t, totalTokens, keyMetrics.TotalTokensConsumed, "Should aggregate total tokens")
}

// TestMetricsCollectorMultipleModels verifies per-model breakdown
func TestMetricsCollectorMultipleModels(t *testing.T) {
	collector := NewMetricsCollector()
	apiKey := "multi-model-key"

	models := []struct {
		name     string
		requests int
		tokens   int
	}{
		{"gpt-4", 8, 1200},
		{"gpt-3.5-turbo", 15, 900},
		{"text-embedding-ada-002", 30, 300},
		{"dall-e-3", 3, 0},
	}

	// Record requests for different models
	for _, model := range models {
		for i := 0; i < model.requests; i++ {
			tokensPerRequest := 0
			if model.requests > 0 {
				tokensPerRequest = model.tokens / model.requests
			}
			collector.RecordRequest(apiKey, "/v1/chat/completions", model.name, 
				tokensPerRequest, 200, 100*time.Millisecond)
		}
	}

	metrics := collector.GetMetrics()
	require.Contains(t, metrics, apiKey)

	keyMetrics := metrics[apiKey].(*KeyMetrics)

	// Verify per-model breakdown
	for _, model := range models {
		require.Contains(t, keyMetrics.PerModel, model.name, "Should have metrics for model %s", model.name)
		
		modelMetrics := keyMetrics.PerModel[model.name]
		assert.Equal(t, int64(model.requests), modelMetrics.TotalRequests,
			"Model %s should have %d requests", model.name, model.requests)
		assert.Equal(t, int64(model.tokens), modelMetrics.TotalTokens,
			"Model %s should have %d tokens", model.name, model.tokens)
	}

	// Verify total aggregation
	totalRequests := int64(8 + 15 + 30 + 3)
	totalTokens := int64(1200 + 900 + 300 + 0)
	assert.Equal(t, totalRequests, keyMetrics.TotalRequests, "Should aggregate total requests")
	assert.Equal(t, totalTokens, keyMetrics.TotalTokensConsumed, "Should aggregate total tokens")
}

// TestMetricsCollectorConcurrentAccess verifies thread safety
func TestMetricsCollectorConcurrentAccess(t *testing.T) {
	collector := NewMetricsCollector()
	numGoroutines := 100
	requestsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start concurrent workers
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			
			apiKey := fmt.Sprintf("worker-%d", workerID)
			
			for j := 0; j < requestsPerGoroutine; j++ {
				// Vary the parameters to test different code paths
				endpoint := fmt.Sprintf("/v1/test-%d", j%3)
				model := fmt.Sprintf("model-%d", j%2)
				statusCode := 200
				if j%10 == 0 {
					statusCode = 500 // 10% failure rate
				}
				
				collector.RecordRequest(apiKey, endpoint, model, 100, 
					statusCode, time.Duration(j)*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify no data races occurred and data is consistent
	metrics := collector.GetMetrics()
	assert.Len(t, metrics, numGoroutines, "Should have metrics for all workers")

	totalRequests := int64(0)
	for i := 0; i < numGoroutines; i++ {
		apiKey := fmt.Sprintf("worker-%d", i)
		require.Contains(t, metrics, apiKey, "Should have metrics for worker %d", i)
		
		keyMetrics := metrics[apiKey].(*KeyMetrics)
		assert.Equal(t, int64(requestsPerGoroutine), keyMetrics.TotalRequests,
			"Worker %d should have recorded all requests", i)
		
		totalRequests += keyMetrics.TotalRequests
	}

	expectedTotal := int64(numGoroutines * requestsPerGoroutine)
	assert.Equal(t, expectedTotal, totalRequests, "Total requests should match expected")
}

// TestMetricsCollectorEdgeCases verifies handling of edge cases
func TestMetricsCollectorEdgeCases(t *testing.T) {
	collector := NewMetricsCollector()

	t.Run("empty_strings", func(t *testing.T) {
		// Should handle empty strings gracefully by converting to "unknown"
		collector.RecordRequest("", "", "", 0, 200, 0)
		
		metrics := collector.GetMetrics()
		require.Contains(t, metrics, "unknown")
		
		keyMetrics := metrics["unknown"].(*KeyMetrics)
		assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	})

	t.Run("negative_tokens", func(t *testing.T) {
		// Should handle negative tokens by converting to 0
		collector.RecordRequest("test-key", "/test", "model", -100, 200, 100*time.Millisecond)
		
		metrics := collector.GetMetrics()
		keyMetrics := metrics["test-key"].(*KeyMetrics)
		assert.Equal(t, int64(0), keyMetrics.TotalTokensConsumed)
	})

	t.Run("zero_duration", func(t *testing.T) {
		// Should handle zero duration
		collector.RecordRequest("zero-duration", "/test", "model", 50, 200, 0)
		
		// Should not panic and should record the metric
		metrics := collector.GetMetrics()
		assert.Contains(t, metrics, "zero-duration")
	})

	t.Run("very_long_strings", func(t *testing.T) {
		// Test with very long strings - should be truncated to 255 chars
		longString := strings.Repeat("a", 10000)
		expectedKey := strings.Repeat("a", 255) // Truncated to 255 chars
		collector.RecordRequest(longString, longString, longString, 100, 200, 100*time.Millisecond)
		
		metrics := collector.GetMetrics()
		assert.Contains(t, metrics, expectedKey)
	})

	t.Run("unusual_status_codes", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			shouldBeSuccess bool
		}{
			{100, false}, // 1xx - informational
			{199, false},
			{200, true},  // 2xx - success
			{299, true},
			{300, false}, // 3xx - redirection
			{399, false},
			{400, false}, // 4xx - client error
			{499, false},
			{500, false}, // 5xx - server error
			{599, false},
			{999, false}, // Invalid status code
		}

		for _, tc := range testCases {
			apiKey := fmt.Sprintf("status-%d", tc.statusCode)
			collector.RecordRequest(apiKey, "/test", "model", 100, tc.statusCode, 100*time.Millisecond)
			
			metrics := collector.GetMetrics()
			keyMetrics := metrics[apiKey].(*KeyMetrics)
			
			if tc.shouldBeSuccess {
				assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests, 
					"Status %d should be counted as success", tc.statusCode)
				assert.Equal(t, int64(0), keyMetrics.FailedRequests,
					"Status %d should not be counted as failure", tc.statusCode)
			} else {
				assert.Equal(t, int64(0), keyMetrics.SuccessfulRequests,
					"Status %d should not be counted as success", tc.statusCode)
				assert.Equal(t, int64(1), keyMetrics.FailedRequests,
					"Status %d should be counted as failure", tc.statusCode)
			}
		}
	})
}

// TestMetricsCollectorMemoryUsage verifies reasonable memory usage
func TestMetricsCollectorMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	collector := NewMetricsCollector()
	
	// Simulate realistic usage pattern
	numKeys := 1000
	requestsPerKey := 100
	
	for i := 0; i < numKeys; i++ {
		apiKey := fmt.Sprintf("user-%d", i)
		
		for j := 0; j < requestsPerKey; j++ {
			endpoint := fmt.Sprintf("/v1/endpoint-%d", j%5)
			model := fmt.Sprintf("model-%d", j%3)
			collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
		}
	}

	metrics := collector.GetMetrics()
	assert.Len(t, metrics, numKeys, "Should track all keys")
	
	// Verify each key has reasonable breakdown
	for i := 0; i < min(10, numKeys); i++ { // Check first 10 keys
		apiKey := fmt.Sprintf("user-%d", i)
		keyMetrics := metrics[apiKey].(*KeyMetrics)
		
		assert.Equal(t, int64(requestsPerKey), keyMetrics.TotalRequests)
		assert.Len(t, keyMetrics.PerEndpoint, 5, "Should have 5 different endpoints")
		assert.Len(t, keyMetrics.PerModel, 3, "Should have 3 different models")
	}
}

// Helper function to collect Prometheus metrics
func collectMetricFamilies(collector prometheus.Collector) ([]*dto.MetricFamily, error) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(collector); err != nil {
		return nil, err
	}

	metricFamilies, err := reg.Gather()
	if err != nil {
		return nil, err
	}

	return metricFamilies, nil
}

// Helper function for min (Go 1.21+ has this built-in)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
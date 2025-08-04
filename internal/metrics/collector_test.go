package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecordRequestUpdatesCounters(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		endpoint       string
		model          string
		tokens         int
		statusCode     int
		duration       time.Duration
		expectedTotal  int64
		expectedSucc   int64
		expectedFail   int64
		expectedTokens int64
	}{
		{
			name:           "successful request",
			apiKey:         "key1",
			endpoint:       "/v1/chat",
			model:          "gpt-3.5-turbo",
			tokens:         100,
			statusCode:     200,
			duration:       500 * time.Millisecond,
			expectedTotal:  int64(1),
			expectedSucc:   int64(1),
			expectedFail:   int64(0),
			expectedTokens: int64(100),
		},
		{
			name:           "failed request",
			apiKey:         "key1",
			endpoint:       "/v1/chat",
			model:          "gpt-3.5-turbo",
			tokens:         50,
			statusCode:     429,
			duration:       300 * time.Millisecond,
			expectedTotal:  int64(1),
			expectedSucc:   int64(0),
			expectedFail:   int64(1),
			expectedTokens: int64(50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMetricsCollector()
			c.RecordRequest(tt.apiKey, tt.endpoint, tt.model, tt.tokens, tt.statusCode, tt.duration)

			metrics := c.GetMetrics()
			keyMetrics, ok := metrics[tt.apiKey].(*KeyMetrics)
			assert.True(t, ok)

			assert.Equal(t, tt.expectedTotal, keyMetrics.TotalRequests)
			assert.Equal(t, tt.expectedSucc, keyMetrics.SuccessfulRequests)
			assert.Equal(t, tt.expectedFail, keyMetrics.FailedRequests)
			assert.Equal(t, tt.expectedTokens, keyMetrics.TotalTokensConsumed)

			// Check histogram observation
			// Note: Actual histogram check would require querying Prometheus, but for unit test, we can skip or mock
		})
	}
}

func TestGetMetricsReturnsAggregates(t *testing.T) {
	c := NewMetricsCollector()

	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 200, 200, 600*time.Millisecond)
	c.RecordRequest("key2", "/v1/embeddings", "ada-002", 50, 429, 300*time.Millisecond)

	metrics := c.GetMetrics()
	assert.Len(t, metrics, 2)

	key1, ok := metrics["key1"].(*KeyMetrics)
	assert.True(t, ok)
	assert.Equal(t, int64(2), key1.TotalRequests)
	assert.Equal(t, int64(2), key1.SuccessfulRequests)
	assert.Equal(t, int64(0), key1.FailedRequests)
	assert.Equal(t, int64(300), key1.TotalTokensConsumed)

	key2, ok := metrics["key2"].(*KeyMetrics)
	assert.True(t, ok)
	assert.Equal(t, int64(1), key2.TotalRequests)
	assert.Equal(t, int64(0), key2.SuccessfulRequests)
	assert.Equal(t, int64(1), key2.FailedRequests)
	assert.Equal(t, int64(50), key2.TotalTokensConsumed)
}

func TestConcurrentRecordRequest(t *testing.T) {
	c := NewMetricsCollector()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 10, 200, 100*time.Millisecond)
		}()
	}

	wg.Wait()

	metrics := c.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)
	assert.Equal(t, int64(100), keyMetrics.TotalRequests)
	assert.Equal(t, int64(100), keyMetrics.SuccessfulRequests)
	assert.Equal(t, int64(0), keyMetrics.FailedRequests)
	assert.Equal(t, int64(1000), keyMetrics.TotalTokensConsumed)
}

func TestRecordRequestUpdatesEndpointBreakdown(t *testing.T) {
	c := NewMetricsCollector()
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	c.RecordRequest("key1", "/v1/embeddings", "ada-002", 50, 200, 300*time.Millisecond)

	metrics := c.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)

	assert.Len(t, keyMetrics.PerEndpoint, 2)

	chatMetrics := keyMetrics.PerEndpoint["/v1/chat"]
	assert.Equal(t, int64(1), chatMetrics.TotalRequests)
	assert.Equal(t, int64(100), chatMetrics.TotalTokens)

	embedMetrics := keyMetrics.PerEndpoint["/v1/embeddings"]
	assert.Equal(t, int64(1), embedMetrics.TotalRequests)
	assert.Equal(t, int64(50), embedMetrics.TotalTokens)
}

func TestRecordRequestUpdatesModelBreakdown(t *testing.T) {
	c := NewMetricsCollector()
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	c.RecordRequest("key1", "/v1/chat", "gpt-4", 200, 200, 600*time.Millisecond)

	metrics := c.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)

	assert.Len(t, keyMetrics.PerModel, 2)

	gpt35Metrics := keyMetrics.PerModel["gpt-3.5-turbo"]
	assert.Equal(t, int64(1), gpt35Metrics.TotalRequests)
	assert.Equal(t, int64(100), gpt35Metrics.TotalTokens)

	gpt4Metrics := keyMetrics.PerModel["gpt-4"]
	assert.Equal(t, int64(1), gpt4Metrics.TotalRequests)
	assert.Equal(t, int64(200), gpt4Metrics.TotalTokens)
}

func BenchmarkRecordRequest(b *testing.B) {
	c := NewMetricsCollector()
	for i := 0; i < b.N; i++ {
		c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	}
}

func TestGetMetricsForKey(t *testing.T) {
	c := NewMetricsCollector()

	// Test non-existent key
	metrics, exists := c.GetMetricsForKey("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, metrics)

	// Add some metrics
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 300*time.Millisecond)
	c.RecordRequest("key1", "/v1/completion", "gpt-4", 200, 404, 400*time.Millisecond)
	c.RecordRequest("key2", "/v1/chat", "gpt-3.5-turbo", 50, 200, 200*time.Millisecond)

	// Test existing key
	metrics, exists = c.GetMetricsForKey("key1")
	assert.True(t, exists)
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(2), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.Equal(t, int64(1), metrics.FailedRequests)
	assert.Equal(t, int64(300), metrics.TotalTokensConsumed)

	// Verify it's a copy (modifying returned metrics shouldn't affect original)
	originalTotal := metrics.TotalRequests
	metrics.TotalRequests = 999
	metricsAgain, _ := c.GetMetricsForKey("key1")
	assert.Equal(t, originalTotal, metricsAgain.TotalRequests)

	// Test endpoint metrics
	assert.Equal(t, 2, len(metrics.PerEndpoint))
	assert.Equal(t, int64(1), metrics.PerEndpoint["/v1/chat"].TotalRequests)
	assert.Equal(t, int64(100), metrics.PerEndpoint["/v1/chat"].TotalTokens)

	// Test model metrics
	assert.Equal(t, 2, len(metrics.PerModel))
	assert.Equal(t, int64(1), metrics.PerModel["gpt-3.5-turbo"].TotalRequests)
	assert.Equal(t, int64(100), metrics.PerModel["gpt-3.5-turbo"].TotalTokens)
}

func TestResetMetrics(t *testing.T) {
	c := NewMetricsCollector()

	// Add some metrics
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 300*time.Millisecond)
	c.RecordRequest("key2", "/v1/chat", "gpt-4", 200, 200, 400*time.Millisecond)

	// Verify metrics exist
	allMetrics := c.GetMetrics()
	assert.Equal(t, 2, len(allMetrics))

	// Reset all metrics
	c.ResetMetrics()

	// Verify all metrics are cleared
	allMetrics = c.GetMetrics()
	assert.Equal(t, 0, len(allMetrics))

	// Verify histogram is re-initialized
	assert.NotNil(t, c.RequestLatency)

	// Can still record new metrics
	c.RecordRequest("key3", "/v1/chat", "gpt-3.5-turbo", 50, 200, 100*time.Millisecond)
	metrics, exists := c.GetMetricsForKey("key3")
	assert.True(t, exists)
	assert.Equal(t, int64(1), metrics.TotalRequests)
}

func TestResetMetricsForKey(t *testing.T) {
	c := NewMetricsCollector()

	// Add metrics for multiple keys
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 300*time.Millisecond)
	c.RecordRequest("key2", "/v1/chat", "gpt-4", 200, 200, 400*time.Millisecond)
	c.RecordRequest("key3", "/v1/chat", "gpt-3.5-turbo", 50, 200, 200*time.Millisecond)

	// Reset metrics for key2
	c.ResetMetricsForKey("key2")

	// Verify key2 metrics are gone
	_, exists := c.GetMetricsForKey("key2")
	assert.False(t, exists)

	// Verify other keys still exist
	metrics1, exists1 := c.GetMetricsForKey("key1")
	assert.True(t, exists1)
	assert.Equal(t, int64(1), metrics1.TotalRequests)

	metrics3, exists3 := c.GetMetricsForKey("key3")
	assert.True(t, exists3)
	assert.Equal(t, int64(1), metrics3.TotalRequests)

	// Test empty key protection
	c.ResetMetricsForKey("") // Should not panic or affect other keys
	allMetrics := c.GetMetrics()
	assert.Equal(t, 2, len(allMetrics)) // key1 and key3 should still exist
}

func TestString(t *testing.T) {
	c := NewMetricsCollector()

	// Test string representation
	str := c.String()
	assert.Contains(t, str, "MetricsCollector")
	assert.Contains(t, str, "keys: 0")
	assert.Contains(t, str, "histogram: true") // Should be initialized

	// Add some metrics
	c.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 300*time.Millisecond)
	c.RecordRequest("key2", "/v1/chat", "gpt-4", 200, 200, 400*time.Millisecond)

	// Test string with metrics
	str = c.String()
	assert.Contains(t, str, "keys: 2")
}

func TestCopyKeyMetrics(t *testing.T) {
	c := NewMetricsCollector()

	// Test with nil
	copied := c.copyKeyMetrics(nil)
	assert.Nil(t, copied)

	// Create original metrics
	original := &KeyMetrics{
		TotalRequests:       100,
		SuccessfulRequests:  80,
		FailedRequests:      20,
		TotalTokensConsumed: 1000,
		PerEndpoint: map[string]*EndpointMetrics{
			"/v1/chat": {
				TotalRequests: 50,
				TotalTokens:   500,
			},
		},
		PerModel: map[string]*ModelMetrics{
			"gpt-3.5-turbo": {
				TotalRequests: 60,
				TotalTokens:   600,
			},
		},
	}

	// Copy metrics
	copied = c.copyKeyMetrics(original)

	// Verify deep copy
	assert.Equal(t, original.TotalRequests, copied.TotalRequests)
	assert.Equal(t, original.SuccessfulRequests, copied.SuccessfulRequests)
	assert.Equal(t, original.FailedRequests, copied.FailedRequests)
	assert.Equal(t, original.TotalTokensConsumed, copied.TotalTokensConsumed)

	// Verify maps are separate instances
	assert.NotSame(t, &original.PerEndpoint, &copied.PerEndpoint)
	assert.NotSame(t, &original.PerModel, &copied.PerModel)

	// Verify content is copied
	assert.Equal(t, original.PerEndpoint["/v1/chat"].TotalRequests, copied.PerEndpoint["/v1/chat"].TotalRequests)
	assert.Equal(t, original.PerModel["gpt-3.5-turbo"].TotalRequests, copied.PerModel["gpt-3.5-turbo"].TotalRequests)

	// Modify original to ensure copy is independent
	original.TotalRequests = 200
	original.PerEndpoint["/v1/chat"].TotalRequests = 100
	assert.Equal(t, int64(100), copied.TotalRequests)
	assert.Equal(t, int64(50), copied.PerEndpoint["/v1/chat"].TotalRequests)
}

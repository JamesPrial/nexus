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

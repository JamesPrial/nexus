package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// EndpointMetrics holds aggregated metrics for an endpoint
type EndpointMetrics struct {
	TotalRequests int64
	TotalTokens   int64
}

// ModelMetrics holds aggregated metrics for a model
type ModelMetrics struct {
	TotalRequests int64
	TotalTokens   int64
}

// KeyMetrics holds aggregated metrics for a single API key
type KeyMetrics struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	TotalTokensConsumed int64
	PerEndpoint         map[string]*EndpointMetrics
	PerModel            map[string]*ModelMetrics
}

type MetricsCollector struct {
	metrics        map[string]*KeyMetrics
	mu             sync.Mutex
	RequestLatency *prometheus.HistogramVec
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RequestLatency.Collect(ch)
}

// NewMetricsCollector creates a new MetricsCollector
func NewMetricsCollector() *MetricsCollector {
	c := &MetricsCollector{
		metrics: make(map[string]*KeyMetrics),
		RequestLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_latency_seconds",
				Help:    "Request latency in seconds",
				Buckets: []float64{0.1, 0.3, 0.5, 1, 3, 5},
			},
			[]string{"api_key", "endpoint", "model"},
		),
	}
	return c
}

// RecordRequest records metrics for a completed request
func (c *MetricsCollector) RecordRequest(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	km, ok := c.metrics[apiKey]
	if !ok {
		km = &KeyMetrics{
			PerEndpoint: make(map[string]*EndpointMetrics),
			PerModel:    make(map[string]*ModelMetrics),
		}
		c.metrics[apiKey] = km
	}

	atomic.AddInt64(&km.TotalRequests, 1)
	if statusCode >= 200 && statusCode < 300 {
		atomic.AddInt64(&km.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&km.FailedRequests, 1)
	}
	atomic.AddInt64(&km.TotalTokensConsumed, int64(tokens))

	// Update per-endpoint
	if _, ok := km.PerEndpoint[endpoint]; !ok {
		km.PerEndpoint[endpoint] = &EndpointMetrics{}
	}
	atomic.AddInt64(&km.PerEndpoint[endpoint].TotalRequests, 1)
	atomic.AddInt64(&km.PerEndpoint[endpoint].TotalTokens, int64(tokens))

	// Update per-model
	if _, ok := km.PerModel[model]; !ok {
		km.PerModel[model] = &ModelMetrics{}
	}
	atomic.AddInt64(&km.PerModel[model].TotalRequests, 1)
	atomic.AddInt64(&km.PerModel[model].TotalTokens, int64(tokens))

	// Observe latency
	c.RequestLatency.WithLabelValues(apiKey, endpoint, model).Observe(duration.Seconds())
}

// GetMetrics returns the current aggregated metrics
func (c *MetricsCollector) GetMetrics() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]any, len(c.metrics))
	for k, v := range c.metrics {
		result[k] = v
	}
	return result
}

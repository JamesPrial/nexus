// Package metrics provides comprehensive metrics collection and reporting for the Nexus API gateway.
// It supports Prometheus, JSON, and CSV export formats with security features like API key masking
// and input sanitization.
package metrics

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/prometheus/client_golang/prometheus"
)

// Type aliases for cleaner code - these refer to the canonical types in interfaces package
type (
	EndpointMetrics = interfaces.EndpointMetrics
	ModelMetrics    = interfaces.ModelMetrics
	KeyMetrics      = interfaces.KeyMetrics
)

// MetricsCollector implements interfaces.MetricsCollector for collecting and aggregating
// API request metrics. It provides thread-safe operations and Prometheus integration.
type MetricsCollector struct {
	// metrics holds per-API-key aggregated metrics
	metrics map[string]*KeyMetrics
	// mu protects the metrics map from concurrent access
	mu sync.RWMutex // Use RWMutex for better read performance
	// RequestLatency tracks request duration histograms for Prometheus export
	RequestLatency *prometheus.HistogramVec
	// histogramInit ensures histogram is properly initialized
	histogramInit sync.Once
}

// Describe implements prometheus.Collector interface for metric registration
func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.RequestLatency != nil {
		c.RequestLatency.Describe(ch)
	}
}

// Collect implements prometheus.Collector interface for metric collection
func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.RequestLatency != nil {
		c.RequestLatency.Collect(ch)
	}
}

// NewMetricsCollector creates a new MetricsCollector with proper initialization.
// The collector is thread-safe and ready for concurrent use.
func NewMetricsCollector() *MetricsCollector {
	c := &MetricsCollector{
		metrics: make(map[string]*KeyMetrics),
	}
	c.initializeHistogram()
	return c
}

// initializeHistogram creates and initializes the Prometheus histogram
func (c *MetricsCollector) initializeHistogram() {
	c.histogramInit.Do(func() {
		c.RequestLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nexus_request_latency_seconds",
				Help:    "Request latency distribution in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.5, 1, 3, 5, 10},
			},
			[]string{"api_key", "endpoint", "model"},
		)
	})
}

// RecordRequest records metrics for a completed request.
// This method is thread-safe and handles all metric aggregation including
// per-key, per-endpoint, and per-model breakdowns.
func (c *MetricsCollector) RecordRequest(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration) {
	// Sanitize and validate inputs
	// For API keys, preserve empty strings but sanitize non-empty ones
	if apiKey != "" {
		apiKey = c.sanitizeInput(apiKey, "unknown")
	}
	endpoint = c.sanitizeInput(endpoint, "unknown")
	model = c.sanitizeInput(model, "unknown")
	if tokens < 0 {
		tokens = 0
	}

	// Get or create key metrics
	km := c.getOrCreateKeyMetrics(apiKey)

	// Update aggregate counters atomically
	atomic.AddInt64(&km.TotalRequests, 1)
	if c.isSuccessStatusCode(statusCode) {
		atomic.AddInt64(&km.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&km.FailedRequests, 1)
	}
	atomic.AddInt64(&km.TotalTokensConsumed, int64(tokens))

	// Update breakdown metrics
	c.updateEndpointMetrics(km, endpoint, tokens)
	c.updateModelMetrics(km, model, tokens)

	// Record latency histogram
	c.recordLatency(apiKey, endpoint, model, duration)
}

// getOrCreateKeyMetrics safely retrieves or creates KeyMetrics for an API key
func (c *MetricsCollector) getOrCreateKeyMetrics(apiKey string) *KeyMetrics {
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
	return km
}

// isSuccessStatusCode determines if a status code represents success
func (c *MetricsCollector) isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// updateEndpointMetrics updates per-endpoint metrics breakdown
func (c *MetricsCollector) updateEndpointMetrics(km *KeyMetrics, endpoint string, tokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := km.PerEndpoint[endpoint]; !ok {
		km.PerEndpoint[endpoint] = &EndpointMetrics{}
	}
	atomic.AddInt64(&km.PerEndpoint[endpoint].TotalRequests, 1)
	atomic.AddInt64(&km.PerEndpoint[endpoint].TotalTokens, int64(tokens))
}

// updateModelMetrics updates per-model metrics breakdown
func (c *MetricsCollector) updateModelMetrics(km *KeyMetrics, model string, tokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := km.PerModel[model]; !ok {
		km.PerModel[model] = &ModelMetrics{}
	}
	atomic.AddInt64(&km.PerModel[model].TotalRequests, 1)
	atomic.AddInt64(&km.PerModel[model].TotalTokens, int64(tokens))
}

// recordLatency records request latency in the Prometheus histogram
func (c *MetricsCollector) recordLatency(apiKey, endpoint, model string, duration time.Duration) {
	if c.RequestLatency != nil {
		c.RequestLatency.WithLabelValues(apiKey, endpoint, model).Observe(duration.Seconds())
	}
}

// GetMetrics returns a copy of all current aggregated metrics.
// The returned map is safe for concurrent use and modification.
func (c *MetricsCollector) GetMetrics() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]any, len(c.metrics))
	for k, v := range c.metrics {
		// Return a deep copy to prevent race conditions
		result[k] = c.copyKeyMetrics(v)
	}
	return result
}

// GetMetricsForKey returns a copy of metrics for a specific API key.
// Returns nil, false if the key doesn't exist.
func (c *MetricsCollector) GetMetricsForKey(apiKey string) (*KeyMetrics, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	km, ok := c.metrics[apiKey]
	if !ok {
		return nil, false
	}

	return c.copyKeyMetrics(km), true
}

// copyKeyMetrics creates a deep copy of KeyMetrics to prevent race conditions
func (c *MetricsCollector) copyKeyMetrics(km *KeyMetrics) *KeyMetrics {
	if km == nil {
		return nil
	}

	copy := &KeyMetrics{
		TotalRequests:       atomic.LoadInt64(&km.TotalRequests),
		SuccessfulRequests:  atomic.LoadInt64(&km.SuccessfulRequests),
		FailedRequests:      atomic.LoadInt64(&km.FailedRequests),
		TotalTokensConsumed: atomic.LoadInt64(&km.TotalTokensConsumed),
		PerEndpoint:         make(map[string]*EndpointMetrics, len(km.PerEndpoint)),
		PerModel:            make(map[string]*ModelMetrics, len(km.PerModel)),
	}

	// Copy endpoint metrics
	for k, v := range km.PerEndpoint {
		copy.PerEndpoint[k] = &EndpointMetrics{
			TotalRequests: atomic.LoadInt64(&v.TotalRequests),
			TotalTokens:   atomic.LoadInt64(&v.TotalTokens),
		}
	}

	// Copy model metrics
	for k, v := range km.PerModel {
		copy.PerModel[k] = &ModelMetrics{
			TotalRequests: atomic.LoadInt64(&v.TotalRequests),
			TotalTokens:   atomic.LoadInt64(&v.TotalTokens),
		}
	}

	return copy
}

// ResetMetrics clears all collected metrics and reinitializes the collector.
// This is primarily used for testing and administrative purposes.
func (c *MetricsCollector) ResetMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string]*KeyMetrics)
	// Reset histogram initialization flag and recreate
	c.histogramInit = sync.Once{}
	c.initializeHistogram()
}

// ResetMetricsForKey clears metrics for a specific API key.
// Note: Prometheus histogram data cannot be selectively removed,
// so histogram metrics for this key will remain until the next full reset.
func (c *MetricsCollector) ResetMetricsForKey(apiKey string) {
	if apiKey == "" {
		return // Prevent accidental deletion of empty key metrics
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.metrics, apiKey)
	// Note: Prometheus histograms cannot be selectively reset
	// This is a known limitation of the Prometheus client library
}

// sanitizeInput removes potentially dangerous characters and patterns from input strings
func (c *MetricsCollector) sanitizeInput(input, defaultValue string) string {
	if input == "" {
		return defaultValue
	}
	
	// Remove null bytes and control characters
	re := regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`)
	sanitized := re.ReplaceAllString(input, "_")
	
	// Remove potential injection patterns
	// SQL injection patterns
	sanitized = regexp.MustCompile(`(?i)(DROP|DELETE|INSERT|UPDATE|SELECT|UNION|ALTER|CREATE|EXEC|EXECUTE)`).ReplaceAllString(sanitized, "BLOCKED")
	
	// JSON injection patterns
	sanitized = regexp.MustCompile(`[{}"\\\n\r\t]`).ReplaceAllString(sanitized, "_")
	
	// Prometheus label injection patterns - remove braces and special chars
	sanitized = regexp.MustCompile(`[{}=",\n\r\t\\]`).ReplaceAllString(sanitized, "_")
	
	// Command injection patterns
	sanitized = regexp.MustCompile(`[$();&|<>]`).ReplaceAllString(sanitized, "_")
	
	// Limit length to prevent DoS
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}
	
	// If sanitization resulted in empty string, use default
	if strings.TrimSpace(sanitized) == "" {
		return defaultValue
	}
	
	return sanitized
}

// String returns a human-readable representation of the collector state
func (c *MetricsCollector) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return fmt.Sprintf("MetricsCollector{keys: %d, histogram: %v}", len(c.metrics), c.RequestLatency != nil)
}
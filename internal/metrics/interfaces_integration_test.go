package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// TestMetricsInterfaceDefinition tests the interface that should be added to internal/interfaces/interfaces.go
func TestMetricsInterfaceDefinition(t *testing.T) {
	// This test documents the expected interface definition
	// The actual interface should be added to internal/interfaces/interfaces.go
	
	t.Run("metrics_collector_interface", func(t *testing.T) {
		// Expected interface:
		/*
		// MetricsCollector collects and aggregates metrics for API requests
		type MetricsCollector interface {
			// RecordRequest records metrics for a completed request
			RecordRequest(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration)
			
			// GetMetrics returns the current aggregated metrics
			GetMetrics() map[string]any
			
			// GetMetricsForKey returns metrics for a specific API key
			GetMetricsForKey(apiKey string) (*KeyMetrics, bool)
			
			// ResetMetrics clears all collected metrics
			ResetMetrics()
			
			// ResetMetricsForKey clears metrics for a specific API key
			ResetMetricsForKey(apiKey string)
		}
		*/
		
		// Test that our collector would implement this interface
		collector := NewMetricsCollector()
		
		// Test RecordRequest
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		// Test GetMetrics
		metrics := collector.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Contains(t, metrics, "test-key")
		
		// Note: GetMetricsForKey, ResetMetrics, ResetMetricsForKey would need to be implemented
	})
	
	t.Run("metrics_exporter_interface", func(t *testing.T) {
		// Expected interface:
		/*
		// MetricsExporter exports metrics in various formats
		type MetricsExporter interface {
			// ExportJSON exports metrics as JSON
			ExportJSON() ([]byte, error)
			
			// ExportPrometheus returns an HTTP handler for Prometheus format
			ExportPrometheus() http.Handler
			
			// ExportCSV exports metrics as CSV (optional)
			ExportCSV() ([]byte, error)
		}
		*/
		
		collector := NewMetricsCollector()
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		// Test JSON export
		jsonData := ExportJSON(collector)
		assert.NotEmpty(t, jsonData)
		
		// Test Prometheus export
		handler := PrometheusHandler(collector)
		assert.NotNil(t, handler)
		
		var _ http.Handler = handler
	})
	
	t.Run("metrics_middleware_interface", func(t *testing.T) {
		// Expected interface:
		/*
		// MetricsMiddleware provides HTTP middleware for metrics collection
		type MetricsMiddleware interface {
			// Middleware returns HTTP middleware that collects metrics
			Middleware(next http.Handler) http.Handler
			
			// SetTokenExtractor sets a function to extract tokens from context
			SetTokenExtractor(extractor func(r *http.Request) (int, error))
			
			// SetModelExtractor sets a function to extract model from context
			SetModelExtractor(extractor func(r *http.Request) (string, error))
		}
		*/
		
		collector := NewMetricsCollector()
		middlewareFunc := MetricsMiddleware(collector)
		
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		wrappedHandler := middlewareFunc(handler)
		assert.NotNil(t, wrappedHandler)
		
		var _ http.Handler = wrappedHandler
		
		// Note: SetTokenExtractor, SetModelExtractor would need to be implemented
	})
}

// TestMetricsConfigurationInterfaceIntegration tests configuration integration
func TestMetricsConfigurationInterfaceIntegration(t *testing.T) {
	// Test the expected configuration structure
	expectedConfig := struct {
		Metrics struct {
			Enabled              bool          `yaml:"enabled"`
			CollectionInterval   time.Duration `yaml:"collection_interval"`
			RetentionPeriod      time.Duration `yaml:"retention_period"`
			MaxMemoryUsage       int64         `yaml:"max_memory_usage"`
			PrometheusEnabled    bool          `yaml:"prometheus_enabled"`
			JSONExportEnabled    bool          `yaml:"json_export_enabled"`
			MetricsEndpoint      string        `yaml:"metrics_endpoint"`
			AuthRequired         bool          `yaml:"auth_required"`
			AllowedAPIKeys       []string      `yaml:"allowed_api_keys"`
			MaskAPIKeys          bool          `yaml:"mask_api_keys"`
			SanitizeEndpoints    bool          `yaml:"sanitize_endpoints"`
		} `yaml:"metrics"`
	}{
		Metrics: struct {
			Enabled              bool          `yaml:"enabled"`
			CollectionInterval   time.Duration `yaml:"collection_interval"`
			RetentionPeriod      time.Duration `yaml:"retention_period"`
			MaxMemoryUsage       int64         `yaml:"max_memory_usage"`
			PrometheusEnabled    bool          `yaml:"prometheus_enabled"`
			JSONExportEnabled    bool          `yaml:"json_export_enabled"`
			MetricsEndpoint      string        `yaml:"metrics_endpoint"`
			AuthRequired         bool          `yaml:"auth_required"`
			AllowedAPIKeys       []string      `yaml:"allowed_api_keys"`
			MaskAPIKeys          bool          `yaml:"mask_api_keys"`
			SanitizeEndpoints    bool          `yaml:"sanitize_endpoints"`
		}{
			Enabled:              true,
			CollectionInterval:   time.Minute,
			RetentionPeriod:      24 * time.Hour,
			MaxMemoryUsage:       100 * 1024 * 1024, // 100MB
			PrometheusEnabled:    true,
			JSONExportEnabled:    true,
			MetricsEndpoint:      "/metrics",
			AuthRequired:         true,
			AllowedAPIKeys:       []string{"admin-key", "monitor-key"},
			MaskAPIKeys:          true,
			SanitizeEndpoints:    true,
		},
	}
	
	// Verify configuration structure
	assert.True(t, expectedConfig.Metrics.Enabled)
	assert.Equal(t, time.Minute, expectedConfig.Metrics.CollectionInterval)
	assert.Equal(t, "/metrics", expectedConfig.Metrics.MetricsEndpoint)
	assert.Len(t, expectedConfig.Metrics.AllowedAPIKeys, 2)
}

// TestContainerIntegration tests integration with the container system
func TestContainerIntegration(t *testing.T) {
	// This test documents how metrics should integrate with the container
	
	t.Run("container_metrics_methods", func(t *testing.T) {
		// Expected methods to be added to Container interface:
		/*
		// MetricsCollector returns the metrics collector instance
		MetricsCollector() interfaces.MetricsCollector
		
		// MetricsMiddleware returns the metrics middleware
		MetricsMiddleware() func(http.Handler) http.Handler
		
		// MetricsHandler returns the HTTP handler for metrics endpoint
		MetricsHandler() http.Handler
		*/
		
		// Test that metrics can be initialized independently
		collector := NewMetricsCollector()
		assert.NotNil(t, collector)
		
		middleware := MetricsMiddleware(collector)
		assert.NotNil(t, middleware)
		
		handler := PrometheusHandler(collector)
		assert.NotNil(t, handler)
	})
	
	t.Run("middleware_chain_integration", func(t *testing.T) {
		// Expected middleware chain order:
		// validation -> auth -> metrics -> rateLimiter -> tokenLimiter -> proxy
		
		collector := NewMetricsCollector()
		metricsMiddleware := MetricsMiddleware(collector)
		
		// Test that metrics middleware can be inserted in chain
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		// Simulate middleware chain
		var handler http.Handler = finalHandler
		handler = metricsMiddleware(handler) // Add metrics middleware
		
		assert.NotNil(t, handler)
		
		// Test that wrapped handler is still http.Handler
		var _ http.Handler = handler
	})
}

// TestMetricsWithExistingInterfaces tests integration with existing interfaces
func TestMetricsWithExistingInterfaces(t *testing.T) {
	t.Run("logger_integration", func(t *testing.T) {
		// Metrics should integrate with existing Logger interface
		// This would be used for logging metrics collection events, errors, etc.
		
		// Expected usage:
		/*
		type MetricsCollectorWithLogger struct {
			collector *MetricsCollector
			logger    interfaces.Logger
		}
		
		func (m *MetricsCollectorWithLogger) RecordRequest(apiKey, endpoint, model string, tokens, statusCode int, duration time.Duration) {
			m.logger.Debug("Recording metrics", map[string]any{
				"api_key": maskAPIKey(apiKey),
				"endpoint": endpoint,
				"model": model,
				"tokens": tokens,
				"status_code": statusCode,
				"duration_ms": duration.Milliseconds(),
			})
			m.collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, duration)
		}
		*/
		
		// For now, just test that metrics collection works independently
		collector := NewMetricsCollector()
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		metrics := collector.GetMetrics()
		assert.Contains(t, metrics, "test-key")
	})
	
	t.Run("config_integration", func(t *testing.T) {
		// Metrics should integrate with existing Config interface
		// This would extend the Config struct to include metrics configuration
		
		// Expected addition to interfaces.Config:
		/*
		type Config struct {
			// ... existing fields ...
			Metrics MetricsConfig `yaml:"metrics"`
		}
		
		type MetricsConfig struct {
			Enabled              bool          `yaml:"enabled"`
			CollectionInterval   time.Duration `yaml:"collection_interval"`
			// ... other metrics config fields ...
		}
		*/
		
		// For now, just test basic configuration concept
		type TestMetricsConfig struct {
			Enabled         bool   `yaml:"enabled"`
			MetricsEndpoint string `yaml:"metrics_endpoint"`
		}
		
		config := TestMetricsConfig{
			Enabled:         true,
			MetricsEndpoint: "/metrics",
		}
		
		assert.True(t, config.Enabled)
		assert.Equal(t, "/metrics", config.MetricsEndpoint)
	})
	
	t.Run("gateway_integration", func(t *testing.T) {
		// Metrics should integrate with Gateway interface
		// Gateway would expose metrics endpoint and provide health information
		
		// Expected additions to Gateway interface:
		/*
		// Health returns the health status including metrics system status
		Health() map[string]any // existing method would include metrics info
		
		// GetMetrics returns current metrics (if authorized)
		GetMetrics(apiKey string) (map[string]any, error)
		*/
		
		collector := NewMetricsCollector()
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		// Simulate health check that includes metrics status
		healthInfo := map[string]any{
			"status": "healthy",
			"metrics": map[string]any{
				"enabled":     true,
				"total_keys":  len(collector.GetMetrics()),
				"last_update": time.Now(),
			},
		}
		
		assert.Equal(t, "healthy", healthInfo["status"])
		metricsInfo := healthInfo["metrics"].(map[string]any)
		assert.True(t, metricsInfo["enabled"].(bool))
		assert.Equal(t, 1, metricsInfo["total_keys"].(int))
	})
}

// TestMetricsInterfaceCompliance tests that our implementations comply with expected interfaces
func TestMetricsInterfaceCompliance(t *testing.T) {
	t.Run("prometheus_collector_compliance", func(t *testing.T) {
		// Our MetricsCollector should implement prometheus.Collector
		collector := NewMetricsCollector()
		
		// Test Describe method
		descChan := make(chan *prometheus.Desc, 10)
		go func() {
			defer close(descChan)
			collector.Describe(descChan)
		}()
		
		descs := make([]*prometheus.Desc, 0)
		for desc := range descChan {
			descs = append(descs, desc)
		}
		
		assert.NotEmpty(t, descs, "Should provide metric descriptions")
		
		// Test Collect method
		metricChan := make(chan prometheus.Metric, 10)
		go func() {
			defer close(metricChan)
			collector.Collect(metricChan)
		}()
		
		metrics := make([]prometheus.Metric, 0)
		for metric := range metricChan {
			metrics = append(metrics, metric)
		}
		
		// Should not panic and should provide metrics
		// Actual metrics may be empty if no data recorded
		_ = metrics // Metrics collected successfully
	})
	
	t.Run("http_handler_compliance", func(t *testing.T) {
		collector := NewMetricsCollector()
		
		// PrometheusHandler should return http.Handler
		handler := PrometheusHandler(collector)
		var _ http.Handler = handler
		
		// MetricsMiddleware should return middleware that produces http.Handler
		middleware := MetricsMiddleware(collector)
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		wrappedHandler := middleware(baseHandler)
		var _ http.Handler = wrappedHandler
	})
	
	t.Run("json_marshaling_compliance", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		// ExportJSON should produce valid JSON
		jsonData := ExportJSON(collector)
		assert.NotEmpty(t, jsonData)
		
		// Should be valid JSON that can be unmarshaled
		var parsed map[string]interface{}
		err := json.Unmarshal(jsonData, &parsed)
		assert.NoError(t, err, "Should produce valid JSON")
		assert.Contains(t, parsed, "test-key", "Should contain expected data")
	})
}

// TestMetricsExtensibilityInterface tests that the metrics system is extensible
func TestMetricsExtensibilityInterface(t *testing.T) {
	t.Run("custom_collector_interface", func(t *testing.T) {
		// Should be possible to implement custom collectors
		// that provide the same interface
		
		// Example custom collector (just for interface testing)
		type CustomMetricsCollector struct {
			data map[string]int64
		}
		
		customCollector := &CustomMetricsCollector{
			data: make(map[string]int64),
		}
		
		// Custom implementation of core functionality
		recordFunc := func(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration) {
			key := fmt.Sprintf("%s:%s", apiKey, endpoint)
			customCollector.data[key]++
		}
		
		getMetricsFunc := func() map[string]any {
			result := make(map[string]any)
			for key, count := range customCollector.data {
				result[key] = count
			}
			return result
		}
		
		// Test custom implementation
		recordFunc("test-key", "/v1/test", "model", 100, 200, 100*time.Millisecond)
		recordFunc("test-key", "/v1/test", "model", 100, 200, 100*time.Millisecond)
		
		metrics := getMetricsFunc()
		assert.Contains(t, metrics, "test-key:/v1/test")
		assert.Equal(t, int64(2), metrics["test-key:/v1/test"])
	})
	
	t.Run("custom_exporter_interface", func(t *testing.T) {
		// Should be possible to implement custom exporters
		
		collector := NewMetricsCollector()
		collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
		
		// Example custom CSV exporter
		customCSVExporter := func(metrics map[string]any) string {
			var result strings.Builder
			result.WriteString("api_key,total_requests,successful_requests,failed_requests,total_tokens\n")
			
			for key, value := range metrics {
				if keyMetrics, ok := value.(*KeyMetrics); ok {
					result.WriteString(fmt.Sprintf("%s,%d,%d,%d,%d\n",
						key,
						keyMetrics.TotalRequests,
						keyMetrics.SuccessfulRequests,
						keyMetrics.FailedRequests,
						keyMetrics.TotalTokensConsumed,
					))
				}
			}
			
			return result.String()
		}
		
		csvOutput := customCSVExporter(collector.GetMetrics())
		assert.Contains(t, csvOutput, "api_key,total_requests")
		assert.Contains(t, csvOutput, "test-key,1,1,0,100")
	})
	
	t.Run("custom_middleware_interface", func(t *testing.T) {
		// Should be possible to implement custom middleware
		// that integrates with metrics collection
		
		collector := NewMetricsCollector()
		
		// Example custom middleware that adds request ID tracking
		customMetricsMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				
				// Add request ID to context (example)
				requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
				ctx := context.WithValue(r.Context(), contextKey("request_id"), requestID)
				r = r.WithContext(ctx)
				
				// Call next handler
				next.ServeHTTP(w, r)
				
				duration := time.Since(start)
				
				// Record with custom fields
				apiKey := r.Header.Get("Authorization")
				if apiKey != "" {
					collector.RecordRequest(apiKey, r.URL.Path, "custom-model", 100, 200, duration)
				}
			})
		}
		
		// Test custom middleware
		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		wrappedHandler := customMetricsMiddleware(baseHandler)
		var _ http.Handler = wrappedHandler
		
		// Verify it's a valid HTTP handler
		assert.NotNil(t, wrappedHandler)
	})
}
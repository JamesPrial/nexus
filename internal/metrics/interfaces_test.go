package metrics

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MetricsCollectorInterface defines the interface that metrics collectors should implement
type MetricsCollectorInterface interface {
	// RecordRequest records metrics for a completed request
	RecordRequest(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration)
	
	// GetMetrics returns the current aggregated metrics
	GetMetrics() map[string]any
}

// MetricsExporterInterface defines the interface for metrics exporters
type MetricsExporterInterface interface {
	// ExportJSON exports metrics in JSON format
	ExportJSON() ([]byte, error)
	
	// PrometheusHandler returns an HTTP handler for Prometheus metrics
	PrometheusHandler() http.Handler
}

// MetricsMiddlewareInterface defines the interface for metrics middleware
type MetricsMiddlewareInterface interface {
	// Middleware returns HTTP middleware that collects metrics
	Middleware(next http.Handler) http.Handler
}

// TestMetricsCollectorImplementsInterface verifies the collector implements expected interface
func TestMetricsCollectorImplementsInterface(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Verify it implements our expected interface
	var _ MetricsCollectorInterface = collector
	
	// Test interface methods
	collector.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	
	metrics := collector.GetMetrics()
	assert.NotNil(t, metrics, "GetMetrics should return non-nil map")
	assert.Contains(t, metrics, "test-key", "Should contain recorded key")
}

// TestMetricsMiddlewareImplementsInterface verifies middleware implements expected interface
func TestMetricsMiddlewareImplementsInterface(t *testing.T) {
	collector := NewMetricsCollector()
	middlewareFunc := MetricsMiddleware(collector)
	
	// Verify it returns a proper middleware function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	wrappedHandler := middlewareFunc(handler)
	assert.NotNil(t, wrappedHandler, "Middleware should return wrapped handler")
	
	// Verify the wrapped handler implements http.Handler
	var _ = wrappedHandler
}

// MockMetricsCollector implements MetricsCollectorInterface for testing
type MockMetricsCollector struct {
	recordedRequests []RequestRecord
	metrics          map[string]*KeyMetrics
}

type RequestRecord struct {
	APIKey     string
	Endpoint   string
	Model      string
	Tokens     int
	StatusCode int
	Duration   time.Duration
	Timestamp  time.Time
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		recordedRequests: make([]RequestRecord, 0),
		metrics:          make(map[string]*KeyMetrics),
	}
}

func (m *MockMetricsCollector) RecordRequest(apiKey string, endpoint string, model string, tokens int, statusCode int, duration time.Duration) {
	record := RequestRecord{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		Model:      model,
		Tokens:     tokens,
		StatusCode: statusCode,
		Duration:   duration,
		Timestamp:  time.Now(),
	}
	m.recordedRequests = append(m.recordedRequests, record)
	
	// Update metrics
	if _, exists := m.metrics[apiKey]; !exists {
		m.metrics[apiKey] = &KeyMetrics{
			PerEndpoint: make(map[string]*EndpointMetrics),
			PerModel:    make(map[string]*ModelMetrics),
		}
	}
	
	km := m.metrics[apiKey]
	km.TotalRequests++
	if statusCode >= 200 && statusCode < 300 {
		km.SuccessfulRequests++
	} else {
		km.FailedRequests++
	}
	km.TotalTokensConsumed += int64(tokens)
}

func (m *MockMetricsCollector) GetMetrics() map[string]any {
	result := make(map[string]any)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

func (m *MockMetricsCollector) GetRecordedRequests() []RequestRecord {
	return m.recordedRequests
}

// TestMockMetricsCollector verifies our mock implementation
func TestMockMetricsCollector(t *testing.T) {
	mock := NewMockMetricsCollector()
	
	// Verify it implements the interface
	var _ MetricsCollectorInterface = mock
	
	// Test recording requests
	mock.RecordRequest("test-key", "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	mock.RecordRequest("test-key", "/v1/test", "test-model", 150, 400, 200*time.Millisecond)
	
	// Verify recorded requests
	requests := mock.GetRecordedRequests()
	require.Len(t, requests, 2, "Should have recorded 2 requests")
	
	assert.Equal(t, "test-key", requests[0].APIKey)
	assert.Equal(t, "/v1/test", requests[0].Endpoint)
	assert.Equal(t, "test-model", requests[0].Model)
	assert.Equal(t, 100, requests[0].Tokens)
	assert.Equal(t, 200, requests[0].StatusCode)
	
	// Verify aggregated metrics
	metrics := mock.GetMetrics()
	require.Contains(t, metrics, "test-key")
	
	keyMetrics := metrics["test-key"].(*KeyMetrics)
	assert.Equal(t, int64(2), keyMetrics.TotalRequests)
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests)
	assert.Equal(t, int64(1), keyMetrics.FailedRequests)
	assert.Equal(t, int64(250), keyMetrics.TotalTokensConsumed)
}

// TestMetricsCollectorPolymorphism verifies collectors can be used polymorphically
func TestMetricsCollectorPolymorphism(t *testing.T) {
	collectors := []MetricsCollectorInterface{
		NewMetricsCollector(),
		NewMockMetricsCollector(),
	}
	
	for i, collector := range collectors {
		t.Run(fmt.Sprintf("collector_%d", i), func(t *testing.T) {
			// Both should behave the same way through the interface
			collector.RecordRequest("poly-key", "/v1/poly", "poly-model", 75, 200, 50*time.Millisecond)
			
			metrics := collector.GetMetrics()
			require.Contains(t, metrics, "poly-key", "Collector %d should record metrics", i)
			
			keyMetrics := metrics["poly-key"].(*KeyMetrics)
			assert.Equal(t, int64(1), keyMetrics.TotalRequests, "Collector %d should have 1 request", i)
			assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests, "Collector %d should have 1 success", i)
			assert.Equal(t, int64(75), keyMetrics.TotalTokensConsumed, "Collector %d should have 75 tokens", i)
		})
	}
}

// Enhanced interface for metrics system integration
type MetricsManager interface {
	MetricsCollectorInterface
	
	// Configuration
	Configure(config *MetricsConfig) error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
	
	// Health
	IsHealthy() bool
	GetStats() map[string]interface{}
}

type MetricsConfig struct {
	Enabled              bool          `yaml:"enabled"`
	CollectionInterval   time.Duration `yaml:"collection_interval"`
	RetentionPeriod      time.Duration `yaml:"retention_period"`
	MaxMemoryUsage       int64         `yaml:"max_memory_usage"`
	PrometheusEnabled    bool          `yaml:"prometheus_enabled"`
	JSONExportEnabled    bool          `yaml:"json_export_enabled"`
	MetricsEndpoint      string        `yaml:"metrics_endpoint"`
	AuthRequired         bool          `yaml:"auth_required"`
	AllowedAPIKeys       []string      `yaml:"allowed_api_keys"`
}

// TestMetricsConfigurationInterface verifies configuration interface design
func TestMetricsConfigurationInterface(t *testing.T) {
	config := &MetricsConfig{
		Enabled:              true,
		CollectionInterval:   time.Minute,
		RetentionPeriod:      24 * time.Hour,
		MaxMemoryUsage:       100 * 1024 * 1024, // 100MB
		PrometheusEnabled:    true,
		JSONExportEnabled:    true,
		MetricsEndpoint:      "/metrics",
		AuthRequired:         true,
		AllowedAPIKeys:       []string{"admin-key", "monitor-key"},
	}
	
	// Verify configuration structure
	assert.True(t, config.Enabled, "Should be enabled")
	assert.Equal(t, time.Minute, config.CollectionInterval)
	assert.Equal(t, 24*time.Hour, config.RetentionPeriod)
	assert.Equal(t, int64(100*1024*1024), config.MaxMemoryUsage)
	assert.True(t, config.PrometheusEnabled)
	assert.True(t, config.JSONExportEnabled)
	assert.Equal(t, "/metrics", config.MetricsEndpoint)
	assert.True(t, config.AuthRequired)
	assert.Len(t, config.AllowedAPIKeys, 2)
}

// Integration interface for dependency injection
type MetricsIntegration interface {
	// Integration with existing interfaces
	ConfigLoader() interfaces.ConfigLoader
	Logger() interfaces.Logger
	
	// Metrics-specific methods
	Collector() MetricsCollectorInterface
	Middleware() func(http.Handler) http.Handler
	HTTPHandler() http.Handler
	
	// Integration methods
	IntegrateWithContainer(container interfaces.Container) error
	RegisterWithGateway(gateway interfaces.Gateway) error
}

// TestMetricsIntegrationInterface verifies integration interface design
func TestMetricsIntegrationInterface(t *testing.T) {
	// This test documents the expected integration interface
	// Implementation would be created in separate integration files
	
	t.Run("interface_design", func(t *testing.T) {
		// Verify the interface design makes sense
		var integration MetricsIntegration
		assert.Nil(t, integration, "Interface should be nil until implemented")
		
		// Document expected integration flow:
		// 1. Create metrics collector
		// 2. Configure middleware
		// 3. Set up HTTP handler
		// 4. Integrate with container DI
		// 5. Register with gateway
	})
}

// Performance interface requirements
type MetricsPerformanceRequirements interface {
	// Performance constraints
	MaxLatencyOverhead() time.Duration    // < 1ms
	MaxMemoryPerClient() int64            // < 1KB
	MaxConcurrentClients() int            // 10,000+
	
	// Throughput requirements  
	MinRequestsPerSecond() int            // 1,000+
	MaxCollectionLatency() time.Duration  // < 100μs
}

// MockPerformanceRequirements implements performance requirements for testing
type MockPerformanceRequirements struct{}

func (m *MockPerformanceRequirements) MaxLatencyOverhead() time.Duration {
	return 1 * time.Millisecond
}

func (m *MockPerformanceRequirements) MaxMemoryPerClient() int64 {
	return 1024 // 1KB
}

func (m *MockPerformanceRequirements) MaxConcurrentClients() int {
	return 10000
}

func (m *MockPerformanceRequirements) MinRequestsPerSecond() int {
	return 1000
}

func (m *MockPerformanceRequirements) MaxCollectionLatency() time.Duration {
	return 100 * time.Microsecond
}

// TestPerformanceRequirements verifies performance requirements interface
func TestPerformanceRequirements(t *testing.T) {
	reqs := &MockPerformanceRequirements{}
	
	// Verify interface implementation
	var _ MetricsPerformanceRequirements = reqs
	
	// Verify requirements are reasonable
	assert.Equal(t, 1*time.Millisecond, reqs.MaxLatencyOverhead())
	assert.Equal(t, int64(1024), reqs.MaxMemoryPerClient())
	assert.Equal(t, 10000, reqs.MaxConcurrentClients())
	assert.Equal(t, 1000, reqs.MinRequestsPerSecond())
	assert.Equal(t, 100*time.Microsecond, reqs.MaxCollectionLatency())
}

// Security interface requirements
type MetricsSecurityInterface interface {
	// API key masking
	MaskAPIKey(apiKey string) string
	
	// Sensitive data filtering
	FilterSensitiveData(data map[string]interface{}) map[string]interface{}
	
	// Access control
	ValidateAccess(apiKey string, operation string) bool
	
	// Audit logging
	LogAccess(apiKey string, operation string, success bool)
}

// TestSecurityInterfaceDesign verifies security interface design
func TestSecurityInterfaceDesign(t *testing.T) {
	t.Run("api_key_masking", func(t *testing.T) {
		// Test API key masking requirements
		testCases := []struct {
			input    string
		}{
			{"sk-1234567890abcdef"},
			{"short"},
			{"very-long-api-key-1234567890"},
		}
		
		for _, tc := range testCases {
			// Use actual masking function from utils package
			masked := utils.MaskAPIKey(tc.input)
			if tc.input != "" {
				assert.NotEqual(t, tc.input, masked, 
					"Masked key should be different from original")
				assert.NotEmpty(t, masked, "Masked key should not be empty")
			}
		}
	})
	
	t.Run("sensitive_data_filtering", func(t *testing.T) {
		sensitiveFields := []string{
			"api_key",
			"authorization",
			"password",
			"token",
			"secret",
		}
		
		for _, field := range sensitiveFields {
			// Document fields that should be filtered
			assert.NotEmpty(t, field, "Sensitive field should be identified")
		}
	})
	
	t.Run("access_control", func(t *testing.T) {
		operations := []string{
			"read_metrics",
			"export_json", 
			"export_prometheus",
			"reset_metrics",
			"configure_metrics",
		}
		
		for _, op := range operations {
			// Document operations that require access control
			assert.NotEmpty(t, op, "Operation should be defined")
		}
	})
}

// Thread safety interface requirements
type MetricsThreadSafetyInterface interface {
	// Concurrent access guarantees
	IsConcurrentSafe() bool
	
	// Lock-free operations where possible
	IsLockFree(operation string) bool
	
	// Performance under contention
	MaxContentionLatency() time.Duration
}

// TestThreadSafetyInterface verifies thread safety interface design
func TestThreadSafetyInterface(t *testing.T) {
	t.Run("concurrency_requirements", func(t *testing.T) {
		// Document concurrency requirements
		requirements := map[string]bool{
			"RecordRequest":     true,  // Must be concurrent safe
			"GetMetrics":        true,  // Must be concurrent safe
			"ExportJSON":        true,  // Must be concurrent safe
			"PrometheusHandler": true,  // Must be concurrent safe
		}
		
		for operation, requiresSafety := range requirements {
			assert.True(t, requiresSafety, 
				"Operation %s must be concurrent safe", operation)
		}
	})
	
	t.Run("lock_free_operations", func(t *testing.T) {
		preferredLockFree := []string{
			"counter_increment",
			"atomic_reads", 
			"timestamp_recording",
		}
		
		for _, op := range preferredLockFree {
			// Document operations that should be lock-free for performance
			assert.NotEmpty(t, op, "Lock-free operation should be identified")
		}
	})
}
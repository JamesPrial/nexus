package metrics

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddlewareRecordsMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer key1")
	// Assume token count is set somehow, e.g., in context or calculated

	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*interfaces.KeyMetrics)

	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests)
	assert.Equal(t, 1, testutil.CollectAndCount(collector.RequestLatency)) // Check one observation
	// Assert tokens, endpoint, model - but need to set them in test
}

func TestMetricsMiddlewareCapturesFailure(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	req := httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer key1")

	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*interfaces.KeyMetrics)

	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(0), keyMetrics.SuccessfulRequests)
	assert.Equal(t, int64(1), keyMetrics.FailedRequests)
}

func TestMetricsMiddlewareConcurrentRequests(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/v1/chat", nil)
			req.Header.Set("Authorization", "Bearer key1")
			rr := httptest.NewRecorder()
			mw(handler).ServeHTTP(rr, req)
		}()
	}
	wg.Wait()

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*interfaces.KeyMetrics)

	assert.Equal(t, int64(50), keyMetrics.TotalRequests)
	assert.Equal(t, int64(50), keyMetrics.SuccessfulRequests)
}

func TestStatusRecorderSize(t *testing.T) {
	recorder := &statusRecorder{ResponseWriter: httptest.NewRecorder()}

	// Test writing data
	data := []byte("Hello, World!")
	n, err := recorder.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), recorder.Size())

	// Test writing more data
	moreData := []byte(" More data")
	n, err = recorder.Write(moreData)
	assert.NoError(t, err)
	assert.Equal(t, len(moreData), n)
	assert.Equal(t, len(data)+len(moreData), recorder.Size())
}

func TestContextFunctions(t *testing.T) {
	// Test SetModel and GetModel
	req := httptest.NewRequest("GET", "/test", nil)
	assert.Equal(t, "", GetModel(req))

	req = SetModel(req, "gpt-4")
	assert.Equal(t, "gpt-4", GetModel(req))

	// Test empty model
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2 = SetModel(req2, "")
	assert.Equal(t, "", GetModel(req2))

	// Test SetTokens and GetTokens
	req3 := httptest.NewRequest("GET", "/test", nil)
	assert.Equal(t, -1, GetTokens(req3))

	req3 = SetTokens(req3, 100)
	assert.Equal(t, 100, GetTokens(req3))

	// Test negative tokens (should not be set)
	req4 := httptest.NewRequest("GET", "/test", nil)
	req4 = SetTokens(req4, -5)
	assert.Equal(t, -1, GetTokens(req4))

	// Test SetAPIKey and GetAPIKey
	req5 := httptest.NewRequest("GET", "/test", nil)
	assert.Equal(t, "", GetAPIKey(req5))

	req5 = SetAPIKey(req5, "test-key")
	assert.Equal(t, "test-key", GetAPIKey(req5))

	// Test empty API key
	req6 := httptest.NewRequest("GET", "/test", nil)
	req6 = SetAPIKey(req6, "")
	assert.Equal(t, "", GetAPIKey(req6))
}

func TestDefaultMiddlewareConfig(t *testing.T) {
	config := DefaultMiddlewareConfig()

	assert.True(t, config.EnablePathNormalization)
	assert.Equal(t, 255, config.MaxPathLength)
	assert.True(t, config.SkipHealthChecks)
	assert.Equal(t, []string{"/health", "/healthz", "/ping", "/status"}, config.HealthCheckPaths)
}

func TestConfigurableMetricsMiddleware(t *testing.T) {
	collector := NewMetricsCollector()

	// Test with nil collector
	mw := ConfigurableMetricsMiddleware(nil, nil)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Test with default config
	mw = ConfigurableMetricsMiddleware(collector, nil)
	req = httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rr = httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	assert.Equal(t, 1, len(metrics))

	// Test with custom config
	config := &MiddlewareConfig{
		EnablePathNormalization: true,  // Enable to test path truncation
		MaxPathLength:          10,
		SkipHealthChecks:       true,
		HealthCheckPaths:       []string{"/health"},
	}
	collector.ResetMetrics()
	mw = ConfigurableMetricsMiddleware(collector, config)

	// Test health check skipping
	req = httptest.NewRequest("GET", "/health", nil)
	rr = httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)
	metrics = collector.GetMetrics()
	assert.Equal(t, 0, len(metrics)) // Health check should be skipped

	// Test normal request with path length limiting
	req = httptest.NewRequest("GET", "/this/is/a/very/long/path/that/exceeds/limit", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rr = httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)
	
	metrics = collector.GetMetrics()
	keyMetrics := metrics["test-key"].(*interfaces.KeyMetrics)
	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	// Path should be truncated to 10 chars when normalization is enabled
	for endpoint := range keyMetrics.PerEndpoint {
		assert.LessOrEqual(t, len(endpoint), 10)
	}

	// Test with normalization disabled
	config.EnablePathNormalization = false
	collector.ResetMetrics()
	mw = ConfigurableMetricsMiddleware(collector, config)

	req = httptest.NewRequest("GET", "/another/very/long/path", nil)
	req.Header.Set("Authorization", "Bearer test-key2")
	rr = httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)
	
	metrics = collector.GetMetrics()
	keyMetrics = metrics["test-key2"].(*interfaces.KeyMetrics)
	// Path should NOT be truncated when normalization is disabled
	for endpoint := range keyMetrics.PerEndpoint {
		assert.Equal(t, "/another/very/long/path", endpoint)
	}
}

func TestIsHealthCheckPath(t *testing.T) {
	healthPaths := []string{"/health", "/healthz", "/ping", "/status"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"exact match health", "/health", true},
		{"exact match healthz", "/healthz", true},
		{"exact match ping", "/ping", true},
		{"exact match status", "/status", true},
		{"health with subpath", "/health/ready", true},
		{"healthz with subpath", "/healthz/live", true},
		{"not health check", "/api/v1/users", false},
		{"partial match", "/healthy", false},
		{"empty path", "", false},
		{"root path", "/", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isHealthCheckPath(tc.path, healthPaths)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMetricsMiddlewareWithContext(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate setting model and tokens in the handler
		r = SetModel(r, "gpt-4")
		_ = SetTokens(r, 150)  // Context is not propagated in this test
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer context-test-key")
	
	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	keyMetrics := metrics["context-test-key"].(*interfaces.KeyMetrics)
	
	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests)
}

func TestMetricsMiddlewareResponseCapture(t *testing.T) {
	collector := NewMetricsCollector()
	config := &MiddlewareConfig{
		EnablePathNormalization: true,
		MaxPathLength:          255,
		SkipHealthChecks:       false,
		HealthCheckPaths:       []string{"/health"},
	}
	mw := ConfigurableMetricsMiddleware(collector, config)

	// Test that response body is captured correctly
	responseBody := "Test response body"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	
	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, responseBody, rr.Body.String())
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

	// Verify metrics
	metrics := collector.GetMetrics()
	keyMetrics := metrics["test-key"].(*interfaces.KeyMetrics)
	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
}

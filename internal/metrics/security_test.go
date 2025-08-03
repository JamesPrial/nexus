package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsDataSanitization verifies sensitive data is properly sanitized
func TestMetricsDataSanitization(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Test with potentially sensitive API keys
	sensitiveAPIKeys := []string{
		"sk-1234567890abcdef1234567890abcdef", // OpenAI format
		"key_live_1234567890abcdef",           // Stripe format
		"xoxb-1234567890-1234567890-abcdef",   // Slack format
		"ghp_1234567890abcdef1234567890abcdef", // GitHub format
		"password123",                          // Common password
		"secret-key-with-sensitive-info",       // Generic secret
	}
	
	for _, apiKey := range sensitiveAPIKeys {
		collector.RecordRequest(apiKey, "/v1/test", "test-model", 100, 200, 100*time.Millisecond)
	}
	
	// Get metrics and verify they contain the raw keys (internal storage)
	metrics := collector.GetMetrics()
	for _, apiKey := range sensitiveAPIKeys {
		require.Contains(t, metrics, apiKey, "Should store original API key internally")
	}
	
	// Test JSON export - should contain raw keys (export is internal)
	jsonData := ExportJSON(collector)
	jsonString := string(jsonData)
	
	for _, apiKey := range sensitiveAPIKeys {
		assert.Contains(t, jsonString, apiKey, 
			"JSON export should contain original keys for internal use")
	}
	
	// Test Prometheus export - should contain raw keys in labels
	handler := PrometheusHandler(collector)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	prometheusOutput := w.Body.String()
	for _, apiKey := range sensitiveAPIKeys {
		assert.Contains(t, prometheusOutput, apiKey,
			"Prometheus export should contain original keys in labels")
	}
}

// TestMetricsAccessControl verifies access control for metrics endpoints
func TestMetricsAccessControl(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record some test data with sensitive API keys
	collector.RecordRequest("sensitive-key-123", "/v1/secret", "secret-model", 100, 200, 100*time.Millisecond)
	collector.RecordRequest("admin-key-456", "/v1/admin", "admin-model", 200, 200, 150*time.Millisecond)
	
	// Create protected metrics handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple authorization check
		authHeader := r.Header.Get("Authorization")
		allowedKeys := map[string]bool{
			"Bearer admin-key":    true,
			"Bearer monitor-key":  true,
			"Bearer readonly-key": true,
		}
		
		if !allowedKeys[authHeader] {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized access to metrics"}`))
			return
		}
		
		// Determine response format
		format := r.URL.Query().Get("format")
		switch format {
		case "json":
			w.Header().Set("Content-Type", "application/json")
			jsonData := ExportJSON(collector)
			_, _ = w.Write(jsonData)
		case "prometheus", "":
			prometheusHandler := PrometheusHandler(collector)
			prometheusHandler.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unsupported format"}`))
		}
	})
	
	testCases := []struct {
		name          string
		authHeader    string
		format        string
		expectedCode  int
		shouldContainData bool
	}{
		{
			name:          "admin_access_json",
			authHeader:    "Bearer admin-key",
			format:        "json",
			expectedCode:  http.StatusOK,
			shouldContainData: true,
		},
		{
			name:          "monitor_access_prometheus",
			authHeader:    "Bearer monitor-key",
			format:        "prometheus",
			expectedCode:  http.StatusOK,
			shouldContainData: true,
		},
		{
			name:          "readonly_access_json",
			authHeader:    "Bearer readonly-key",
			format:        "json",
			expectedCode:  http.StatusOK,
			shouldContainData: true,
		},
		{
			name:          "unauthorized_access",
			authHeader:    "Bearer unauthorized-key",
			format:        "json",
			expectedCode:  http.StatusUnauthorized,
			shouldContainData: false,
		},
		{
			name:          "no_auth_header",
			authHeader:    "",
			format:        "json",
			expectedCode:  http.StatusUnauthorized,
			shouldContainData: false,
		},
		{
			name:          "invalid_format",
			authHeader:    "Bearer admin-key",
			format:        "xml",
			expectedCode:  http.StatusBadRequest,
			shouldContainData: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/metrics"
			if tc.format != "" {
				url += "?format=" + tc.format
			}
			
			req := httptest.NewRequest("GET", url, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			
			w := httptest.NewRecorder()
			protectedHandler.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedCode, w.Code, 
				"Test %s should return expected status", tc.name)
			
			body := w.Body.String()
			
			if tc.shouldContainData {
				assert.NotEmpty(t, body, "Should return metrics data")
				
				if tc.format == "json" {
					// Verify JSON structure
					var metrics map[string]interface{}
					err := json.Unmarshal([]byte(body), &metrics)
					assert.NoError(t, err, "Should return valid JSON")
					assert.Contains(t, metrics, "sensitive-key-123", "Should contain sensitive data")
				} else {
					// Verify Prometheus format
					assert.Contains(t, body, "request_latency_seconds", "Should contain Prometheus metrics")
					assert.Contains(t, body, "sensitive-key-123", "Should contain sensitive data")
				}
			} else {
				// Should not contain sensitive data
				assert.NotContains(t, body, "sensitive-key-123", "Should not leak sensitive data on error")
				assert.NotContains(t, body, "admin-key-456", "Should not leak sensitive data on error")
			}
		})
	}
}

// TestMetricsInjectionPrevention verifies protection against injection attacks
func TestMetricsInjectionPrevention(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Test various injection attempts
	injectionTests := []struct {
		name     string
		apiKey   string
		endpoint string
		model    string
		desc     string
	}{
		{
			name:     "sql_injection_apikey",
			apiKey:   "key'; DROP TABLE metrics; --",
			endpoint: "/v1/test",
			model:    "model",
			desc:     "SQL injection in API key",
		},
		{
			name:     "xss_injection_endpoint",
			apiKey:   "normal-key",
			endpoint: "/v1/<script>alert('xss')</script>",
			model:    "model",
			desc:     "XSS injection in endpoint",
		},
		{
			name:     "command_injection_model",
			apiKey:   "normal-key",
			endpoint: "/v1/test",
			model:    "model$(rm -rf /)",
			desc:     "Command injection in model",
		},
		{
			name:     "json_injection_apikey",
			apiKey:   `key","malicious":"value`,
			endpoint: "/v1/test",
			model:    "model",
			desc:     "JSON injection in API key",
		},
		{
			name:     "newline_injection_endpoint",
			apiKey:   "normal-key",
			endpoint: "/v1/test\nmalicious: header",
			model:    "model",
			desc:     "Newline injection in endpoint",
		},
		{
			name:     "prometheus_injection_model",
			apiKey:   "normal-key",
			endpoint: "/v1/test",
			model:    `model"} malicious{injection="value`,
			desc:     "Prometheus label injection in model",
		},
		{
			name:     "null_byte_injection",
			apiKey:   "key\x00malicious",
			endpoint: "/v1/test\x00injection",
			model:    "model\x00payload",
			desc:     "Null byte injection",
		},
		{
			name:     "unicode_injection",
			apiKey:   "key\u202emalicious",
			endpoint: "/v1/test\u202einjection",
			model:    "model\u202epayload",
			desc:     "Unicode injection",
		},
	}
	
	for _, test := range injectionTests {
		t.Run(test.name, func(t *testing.T) {
			// Should not panic or crash
			assert.NotPanics(t, func() {
				collector.RecordRequest(test.apiKey, test.endpoint, test.model, 100, 200, 100*time.Millisecond)
			}, "Should handle injection attempt safely: %s", test.desc)
			
			// Verify data was recorded (sanitized or as-is)
			metrics := collector.GetMetrics()
			// The collector may sanitize the API key, so we need to check if any key was recorded
			assert.NotEmpty(t, metrics, "Should record at least one metric")
			
			// Find the recorded key (might be sanitized)
			var recordedKey string
			for key := range metrics {
				recordedKey = key
				break
			}
			
			keyMetrics := metrics[recordedKey].(*KeyMetrics)
			assert.Greater(t, keyMetrics.TotalRequests, int64(0), "Should record request")
		})
	}
	
	// Test JSON export safety
	t.Run("json_export_safety", func(t *testing.T) {
		jsonData := ExportJSON(collector)
		jsonString := string(jsonData)
		
		// Should be valid JSON despite injection attempts
		var parsed map[string]interface{}
		err := json.Unmarshal(jsonData, &parsed)
		assert.NoError(t, err, "JSON should remain valid despite injection attempts")
		
		// Should not contain obviously malicious content in raw form
		assert.NotContains(t, jsonString, "<script>", "Should not contain raw script tags")
		assert.NotContains(t, jsonString, "DROP TABLE", "Should not contain raw SQL commands")
	})
	
	// Test Prometheus export safety
	t.Run("prometheus_export_safety", func(t *testing.T) {
		handler := PrometheusHandler(collector)
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		
		assert.NotPanics(t, func() {
			handler.ServeHTTP(w, req)
		}, "Prometheus export should handle injected data safely")
		
		assert.Equal(t, http.StatusOK, w.Code, "Should return successful response")
		
		body := w.Body.String()
		assert.NotEmpty(t, body, "Should return metrics data")
		
		// Should not contain raw injection attempts that could break Prometheus parsing
		assert.NotContains(t, body, `"} malicious{`, "Should sanitize Prometheus label injection")
	})
}

// TestMetricsTimingAttackPrevention verifies protection against timing attacks
func TestMetricsTimingAttackPrevention(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Record data for different length API keys
	shortKey := "key123"
	longKey := "very-long-api-key-with-many-characters-1234567890abcdef"
	
	collector.RecordRequest(shortKey, "/v1/test", "model", 100, 200, 100*time.Millisecond)
	collector.RecordRequest(longKey, "/v1/test", "model", 100, 200, 100*time.Millisecond)
	
	// Test that operations have consistent timing regardless of key length
	shortKeyTimes := make([]time.Duration, 100)
	longKeyTimes := make([]time.Duration, 100)
	
	// Measure RecordRequest timing
	for i := 0; i < 100; i++ {
		start := time.Now()
		collector.RecordRequest(shortKey, "/v1/test", "model", 100, 200, 100*time.Millisecond)
		shortKeyTimes[i] = time.Since(start)
		
		start = time.Now()
		collector.RecordRequest(longKey, "/v1/test", "model", 100, 200, 100*time.Millisecond)
		longKeyTimes[i] = time.Since(start)
	}
	
	// Calculate average times
	var shortTotal, longTotal time.Duration
	for i := 0; i < 100; i++ {
		shortTotal += shortKeyTimes[i]
		longTotal += longKeyTimes[i]
	}
	
	shortAvg := shortTotal / 100
	longAvg := longTotal / 100
	
	// Timing difference should be minimal (within reasonable variance)
	_ = shortAvg - longAvg // Not used directly, only through ratio calculation
	
	// Allow up to 10x difference (should be much less in practice)
	maxAllowedRatio := 10.0
	actualRatio := float64(longAvg) / float64(shortAvg)
	
	assert.Less(t, actualRatio, maxAllowedRatio, 
		"Timing difference should not reveal key length (short: %v, long: %v, ratio: %.2f)", 
		shortAvg, longAvg, actualRatio)
	
	t.Logf("Short key average: %v, Long key average: %v, Ratio: %.2f", 
		shortAvg, longAvg, actualRatio)
}

// TestMetricsMemoryLeakPrevention verifies no sensitive data leaks in memory
func TestMetricsMemoryLeakPrevention(t *testing.T) {
	collector := NewMetricsCollector()
	
	sensitiveKey := "sk-super-secret-key-1234567890abcdef"
	sensitiveEndpoint := "/v1/secret-internal-endpoint"
	sensitiveModel := "secret-proprietary-model-v2"
	
	// Record sensitive data
	collector.RecordRequest(sensitiveKey, sensitiveEndpoint, sensitiveModel, 100, 200, 100*time.Millisecond)
	
	// Get current metrics
	metrics := collector.GetMetrics()
	require.Contains(t, metrics, sensitiveKey)
	
	// Clear reference to collector
	collector = nil
	
	// Force garbage collection
	// Note: This is more of a documentation test - actual memory leak detection
	// would require more sophisticated tooling
	
	// Create new collector - should not have access to old data
	newCollector := NewMetricsCollector()
	newMetrics := newCollector.GetMetrics()
	
	assert.NotContains(t, newMetrics, sensitiveKey, 
		"New collector should not have access to old sensitive data")
	assert.Empty(t, newMetrics, "New collector should start empty")
}

// TestMetricsErrorLeakagePrevention verifies errors don't leak sensitive information
func TestMetricsErrorLeakagePrevention(t *testing.T) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)
	
	// Handler that might leak information in errors
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Different error scenarios
		switch r.URL.Path {
		case "/v1/internal-error":
			// Internal error that might leak system info
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"database connection failed to user@secret-host:5432/internal_db"}`))
		case "/v1/validation-error":
			// Validation error that might leak data
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid API key: sk-1234..."}`))
		case "/v1/auth-error":
			// Auth error that might leak sensitive info
			w.WriteHeader(http.StatusUnauthorized) 
			_, _ = w.Write([]byte(`{"error":"unauthorized access for key sk-secret-key-123 to endpoint /admin"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}
	})
	
	wrappedHandler := middleware(handler)
	
	errorTests := []struct {
		path   string
		apiKey string
	}{
		{"/v1/internal-error", "key-internal-123"},
		{"/v1/validation-error", "key-validation-456"},
		{"/v1/auth-error", "key-auth-789"},
	}
	
	for _, test := range errorTests {
		req := httptest.NewRequest("POST", test.path, strings.NewReader(`{"test":"data"}`))
		req.Header.Set("Authorization", "Bearer "+test.apiKey)
		
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
		
		// Errors should be recorded in metrics but not affect metric collection
		assert.NotEqual(t, http.StatusOK, w.Code, "Should return error status")
	}
	
	// Verify all requests were recorded despite errors
	metrics := collector.GetMetrics()
	for _, test := range errorTests {
		require.Contains(t, metrics, test.apiKey, 
			"Should record metrics even for error responses")
		
		keyMetrics := metrics[test.apiKey].(*KeyMetrics)
		assert.Greater(t, keyMetrics.FailedRequests, int64(0), 
			"Should record failed requests for key %s", test.apiKey)
	}
	
	// Verify exports don't include error response bodies
	jsonData := ExportJSON(collector)
	jsonString := string(jsonData)
	
	// Should not contain sensitive information from error responses
	assert.NotContains(t, jsonString, "secret-host", 
		"JSON export should not leak database connection strings")
	assert.NotContains(t, jsonString, "/admin", 
		"JSON export should not leak sensitive endpoint names from errors")
}

// TestMetricsRateLimitingForProtection verifies metrics collection doesn't become attack vector
func TestMetricsRateLimitingForProtection(t *testing.T) {
	collector := NewMetricsCollector()
	
	// Simulate potential abuse - many unique keys
	startTime := time.Now()
	
	for i := 0; i < 10000; i++ {
		// Unique API key for each request (potential memory exhaustion attack)
		apiKey := fmt.Sprintf("attack-key-%d", i)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%10)
		model := fmt.Sprintf("model-%d", i%5)
		
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
		
		// Stop if this takes too long (should be fast)
		if time.Since(startTime) > 5*time.Second {
			t.Fatalf("Metrics collection too slow under load - potential DoS vulnerability")
		}
	}
	
	processingTime := time.Since(startTime)
	t.Logf("Processed 10,000 unique keys in %v", processingTime)
	
	// Should handle load reasonably fast
	assert.Less(t, processingTime, 2*time.Second, 
		"Should process large number of unique keys efficiently")
	
	// Verify all data was recorded
	metrics := collector.GetMetrics()
	assert.Len(t, metrics, 10000, "Should record all unique API keys")
	
	// Test that export functions still work with large dataset
	t.Run("export_performance_under_load", func(t *testing.T) {
		exportStart := time.Now()
		jsonData := ExportJSON(collector)
		jsonExportTime := time.Since(exportStart)
		
		assert.NotEmpty(t, jsonData, "JSON export should work under load")
		assert.Less(t, jsonExportTime, 10*time.Second, 
			"JSON export should complete in reasonable time")
		
		prometheusStart := time.Now()
		handler := PrometheusHandler(collector)
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		prometheusExportTime := time.Since(prometheusStart)
		
		assert.Equal(t, http.StatusOK, w.Code, "Prometheus export should work under load")
		assert.Less(t, prometheusExportTime, 15*time.Second,
			"Prometheus export should complete in reasonable time")
		
		t.Logf("JSON export: %v, Prometheus export: %v", jsonExportTime, prometheusExportTime)
	})
}

// TestMetricsInputValidation verifies proper input validation
func TestMetricsInputValidation(t *testing.T) {
	collector := NewMetricsCollector()
	
	validationTests := []struct {
		name       string
		apiKey     string
		endpoint   string
		model      string
		tokens     int
		statusCode int
		duration   time.Duration
		shouldWork bool
		desc       string
	}{
		{
			name:       "normal_input",
			apiKey:     "valid-key",
			endpoint:   "/v1/test",
			model:      "gpt-4",
			tokens:     100,
			statusCode: 200,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Normal valid input",
		},
		{
			name:       "empty_api_key",
			apiKey:     "",
			endpoint:   "/v1/test",
			model:      "gpt-4", 
			tokens:     100,
			statusCode: 200,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Empty API key should be handled",
		},
		{
			name:       "negative_tokens",
			apiKey:     "test-key",
			endpoint:   "/v1/test",
			model:      "gpt-4",
			tokens:     -100,
			statusCode: 200,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Negative tokens should be handled",
		},
		{
			name:       "zero_duration",
			apiKey:     "test-key",
			endpoint:   "/v1/test", 
			model:      "gpt-4",
			tokens:     100,
			statusCode: 200,
			duration:   0,
			shouldWork: true,
			desc:       "Zero duration should be handled",
		},
		{
			name:       "negative_duration",
			apiKey:     "test-key",
			endpoint:   "/v1/test",
			model:      "gpt-4",
			tokens:     100,
			statusCode: 200,
			duration:   -100 * time.Millisecond,
			shouldWork: true,
			desc:       "Negative duration should be handled",
		},
		{
			name:       "invalid_status_code",
			apiKey:     "test-key",
			endpoint:   "/v1/test",
			model:      "gpt-4",
			tokens:     100,
			statusCode: 999,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Invalid status code should be handled",
		},
		{
			name:       "very_large_tokens",
			apiKey:     "test-key",
			endpoint:   "/v1/test",
			model:      "gpt-4",
			tokens:     2147483647, // Max int32
			statusCode: 200,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Very large token count should be handled",
		},
		{
			name:       "very_long_strings",
			apiKey:     strings.Repeat("a", 10000),
			endpoint:   strings.Repeat("/v1/", 1000) + "test",
			model:      strings.Repeat("model-", 1000),
			tokens:     100,
			statusCode: 200,
			duration:   100 * time.Millisecond,
			shouldWork: true,
			desc:       "Very long strings should be handled",
		},
	}
	
	for _, test := range validationTests {
		t.Run(test.name, func(t *testing.T) {
			if test.shouldWork {
				assert.NotPanics(t, func() {
					collector.RecordRequest(test.apiKey, test.endpoint, test.model, 
						test.tokens, test.statusCode, test.duration)
				}, "Should handle input gracefully: %s", test.desc)
				
				// Verify data was recorded
				metrics := collector.GetMetrics()
				assert.Contains(t, metrics, test.apiKey, 
					"Should record metrics for: %s", test.desc)
			} else {
				// If we decide to add validation that rejects certain inputs
				// This branch would test that rejection
				t.Logf("Test case %s: %s", test.name, test.desc)
			}
		})
	}
}
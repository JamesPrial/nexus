package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/stretchr/testify/assert"
)

func TestExportJSON(t *testing.T) {
	collector := NewMetricsCollector()
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)

	jsonData := ExportJSON(collector)
	var data map[string]any
	err := json.Unmarshal(jsonData, &data)
	assert.NoError(t, err)

	keyMetrics := data["key1"].(map[string]any)
	assert.Equal(t, float64(1), keyMetrics["TotalRequests"])
	assert.Equal(t, float64(1), keyMetrics["SuccessfulRequests"])
	// Further assertions for breakdowns, etc.
}

func TestPrometheusHandler(t *testing.T) {
	collector := NewMetricsCollector()
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := PrometheusHandler(collector)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "request_latency_seconds")
	assert.Contains(t, body, `api_key="key1"`)
	// Check for specific metrics values
}

func TestMetricsExporterSetters(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test SetAPIKeyMasking
	exporter.SetAPIKeyMasking(false)
	assert.False(t, exporter.maskAPIKeys)
	exporter.SetAPIKeyMasking(true)
	assert.True(t, exporter.maskAPIKeys)

	// Test SetSanitization
	exporter.SetSanitization(false)
	assert.False(t, exporter.sanitizeData)
	exporter.SetSanitization(true)
	assert.True(t, exporter.sanitizeData)
}

func TestExporterExportJSON(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test empty metrics
	data, err := exporter.ExportJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), data)

	// Add metrics
	collector.RecordRequest("testkey12345678", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)

	// Test with masking enabled
	exporter.SetAPIKeyMasking(true)
	data, err = exporter.ExportJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test*******5678")
	assert.NotContains(t, string(data), "testkey12345678")

	// Test with masking disabled
	exporter.SetAPIKeyMasking(false)
	data, err = exporter.ExportJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(data), "testkey12345678")
}

func TestExportCSV(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test empty metrics
	data, err := exporter.ExportCSV()
	assert.NoError(t, err)
	assert.Equal(t, []byte("api_key,total_requests,successful_requests,failed_requests,total_tokens\n"), data)

	// Add metrics
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	collector.RecordRequest("key1", "/v1/chat", "gpt-4", 200, 404, 600*time.Millisecond)
	collector.RecordRequest("key2", "/v1/completion", "gpt-3.5-turbo", 50, 200, 300*time.Millisecond)

	// Test CSV export with masking disabled
	exporter.SetAPIKeyMasking(false)
	data, err = exporter.ExportCSV()
	assert.NoError(t, err)
	csvStr := string(data)
	assert.Contains(t, csvStr, "api_key,total_requests,successful_requests,failed_requests,total_tokens")
	assert.Contains(t, csvStr, "key1,2,1,1,300")
	assert.Contains(t, csvStr, "key2,1,1,0,50")
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{"short key", "abc", "***"},
		{"exact 8 chars", "12345678", "********"},
		{"long key", "abcd1234efgh5678", "abcd********5678"},
		{"empty key", "", ""},
		{"very long key", "verylongapikeywithmanychars123456", "very*************************3456"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskAPIKey(tc.apiKey)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeForExport(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text", "hello world", "hello world"},
		{"control chars", "hello\x00world\x1f", "hello_world_"},
		{"html chars", "<script>alert('xss')</script>", "_script_alert(_xss_)_/script_"},
		{"quotes and escape", "test\"value\"\\path", "test_value__path"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeForExport(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPrometheusErrorLoggerPrintln(t *testing.T) {
	logger := &prometheusErrorLogger{}
	// Should not panic, just silently ignore
	logger.Println("test error", "another error")
}

func TestSanitizeMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Create test metrics
	metrics := map[string]any{
		"testkey12345678": map[string]any{"requests": 100},
		"<script>alert": map[string]any{"requests": 50},
	}

	// Test with all sanitization disabled
	exporter.SetAPIKeyMasking(false)
	exporter.SetSanitization(false)
	result := exporter.sanitizeMetrics(metrics)
	assert.Equal(t, metrics, result)

	// Test with API key masking only
	exporter.SetAPIKeyMasking(true)
	exporter.SetSanitization(false)
	result = exporter.sanitizeMetrics(metrics)
	assert.Contains(t, result, "test*******5678")
	assert.NotContains(t, result, "testkey12345678")
	assert.Contains(t, result, "<scr*****lert")

	// Test with full sanitization
	exporter.SetAPIKeyMasking(true)
	exporter.SetSanitization(true)
	result = exporter.sanitizeMetrics(metrics)
	assert.Contains(t, result, "test*******5678")
	assert.NotContains(t, result, "<script>")
	assert.Contains(t, result, "_scr*****lert")
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"bearer token", "Bearer mytoken123", "mytoken123"},
		{"no bearer prefix", "mytoken123", "mytoken123"},
		{"empty", "", ""},
		{"just bearer", "Bearer ", ""},
		{"lowercase bearer", "bearer mytoken", "bearer mytoken"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractBearerToken(tc.header)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleCSVExport(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test with CSV disabled
	config := &interfaces.MetricsConfig{CSVExportEnabled: false}
	w := httptest.NewRecorder()
	handleCSVExport(w, exporter, config)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "CSV export not enabled")

	// Test with CSV enabled
	config.CSVExportEnabled = true
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	w = httptest.NewRecorder()
	handleCSVExport(w, exporter, config)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "filename=metrics.csv")
	assert.Contains(t, w.Body.String(), "api_key,total_requests")
}

func TestHandleJSONExport(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test with JSON disabled
	config := &interfaces.MetricsConfig{JSONExportEnabled: false}
	w := httptest.NewRecorder()
	handleJSONExport(w, exporter, config)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "JSON export not enabled")

	// Test with JSON enabled
	config.JSONExportEnabled = true
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	w = httptest.NewRecorder()
	handleJSONExport(w, exporter, config)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var result map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
}

func TestHandlePrometheusExport(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)

	// Test with Prometheus disabled
	config := &interfaces.MetricsConfig{PrometheusEnabled: false}
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handlePrometheusExport(w, req, exporter, config)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Prometheus export not enabled")

	// Test with Prometheus enabled
	config.PrometheusEnabled = true
	collector.RecordRequest("key1", "/v1/chat", "gpt-3.5-turbo", 100, 200, 500*time.Millisecond)
	w = httptest.NewRecorder()
	handlePrometheusExport(w, req, exporter, config)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "request_latency_seconds")
}

func TestIsAllowedKey(t *testing.T) {
	allowedKeys := []string{"key1", "key2", "key3"}

	// Test allowed key
	assert.True(t, isAllowedKey("key1", allowedKeys))
	assert.True(t, isAllowedKey("key2", allowedKeys))

	// Test not allowed key
	assert.False(t, isAllowedKey("key4", allowedKeys))
	assert.False(t, isAllowedKey("", allowedKeys))

	// Test empty allowed list
	assert.False(t, isAllowedKey("key1", []string{}))
	assert.False(t, isAllowedKey("key1", nil))
}

func TestConstantTimeCompare(t *testing.T) {
	// Test equal strings
	assert.True(t, constantTimeCompare("test123", "test123"))
	assert.True(t, constantTimeCompare("", ""))

	// Test different strings
	assert.False(t, constantTimeCompare("test123", "test124"))
	assert.False(t, constantTimeCompare("test", "testing"))
	assert.False(t, constantTimeCompare("", "test"))
}

func TestAuthenticatedExportHandler(t *testing.T) {
	collector := NewMetricsCollector()
	exporter := NewMetricsExporter(collector)
	config := &interfaces.MetricsConfig{
		AuthRequired:      true,
		JSONExportEnabled: true,
		CSVExportEnabled:  true,
		PrometheusEnabled: true,
	}
	allowedKeys := []string{"validkey123"}

	handler := AuthenticatedExportHandler(exporter, config, allowedKeys)

	// Test without auth
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test with invalid auth
	req = httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer invalidkey")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test with valid auth - JSON format
	req = httptest.NewRequest("GET", "/metrics?format=json", nil)
	req.Header.Set("Authorization", "Bearer validkey123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Test CSV format
	req = httptest.NewRequest("GET", "/metrics?format=csv", nil)
	req.Header.Set("Authorization", "Bearer validkey123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))

	// Test Prometheus format (default)
	req = httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Authorization", "Bearer validkey123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test with auth not required
	config.AuthRequired = false
	handler = AuthenticatedExportHandler(exporter, config, allowedKeys)
	req = httptest.NewRequest("GET", "/metrics?format=json", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

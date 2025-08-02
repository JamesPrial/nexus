package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

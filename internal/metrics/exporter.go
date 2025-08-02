package metrics

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ExportJSON exports metrics as JSON
func ExportJSON(collector *MetricsCollector) []byte {
	metrics := collector.GetMetrics()
	data, _ := json.Marshal(metrics) // Ignore error for minimal impl
	return data
}

func PrometheusHandler(collector *MetricsCollector) http.Handler {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

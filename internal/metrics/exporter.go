// Package metrics provides comprehensive metrics collection and reporting for the Nexus API gateway.
package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsExporter implements interfaces.MetricsExporter for exporting metrics
// in multiple formats with security features like API key masking and input sanitization.
type MetricsExporter struct {
	collector    interfaces.MetricsCollector
	maskAPIKeys  bool
	sanitizeData bool
}

// NewMetricsExporter creates a new metrics exporter with secure defaults.
// API key masking and data sanitization are enabled by default.
func NewMetricsExporter(collector interfaces.MetricsCollector) *MetricsExporter {
	return &MetricsExporter{
		collector:    collector,
		maskAPIKeys:  true,
		sanitizeData: true,
	}
}

// SetAPIKeyMasking configures whether API keys should be masked in exports.
// This should only be disabled in development environments.
func (e *MetricsExporter) SetAPIKeyMasking(enabled bool) {
	e.maskAPIKeys = enabled
}

// SetSanitization configures whether data should be sanitized in exports.
// This should only be disabled in development environments.
func (e *MetricsExporter) SetSanitization(enabled bool) {
	e.sanitizeData = enabled
}

// ExportJSON exports metrics as JSON with optional sanitization and masking.
func (e *MetricsExporter) ExportJSON() ([]byte, error) {
	metrics := e.collector.GetMetrics()
	if metrics == nil {
		return []byte("{}"), nil
	}

	sanitized := e.sanitizeMetrics(metrics)
	data, err := json.Marshal(sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}
	
	return data, nil
}

// ExportPrometheus returns an HTTP handler for Prometheus format.
// The handler automatically applies Prometheus-compatible metric formatting.
func (e *MetricsExporter) ExportPrometheus() http.Handler {
	// Cast to concrete type for Prometheus registration
	if concreteCollector, ok := e.collector.(*MetricsCollector); ok {
		reg := prometheus.NewRegistry()
		if err := reg.Register(concreteCollector); err != nil {
			// If registration fails, return error handler
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, fmt.Sprintf("Prometheus registration failed: %v", err), http.StatusInternalServerError)
			})
		}
		return promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
			ErrorLog:      &prometheusErrorLogger{},
		})
	}
	
	// Fallback handler for non-concrete collectors
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Prometheus export not supported for this collector type", http.StatusInternalServerError)
	})
}

// ExportCSV exports metrics as CSV format with headers and proper escaping.
func (e *MetricsExporter) ExportCSV() ([]byte, error) {
	metrics := e.collector.GetMetrics()
	if metrics == nil {
		return []byte("api_key,total_requests,successful_requests,failed_requests,total_tokens\n"), nil
	}

	sanitized := e.sanitizeMetrics(metrics)
	
	var result strings.Builder
	writer := csv.NewWriter(&result)
	
	// Write CSV header
	header := []string{"api_key", "total_requests", "successful_requests", "failed_requests", "total_tokens"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write data rows
	for apiKey, metricsData := range sanitized {
		if keyMetrics, ok := metricsData.(*interfaces.KeyMetrics); ok {
			row := []string{
				apiKey,
				fmt.Sprintf("%d", keyMetrics.TotalRequests),
				fmt.Sprintf("%d", keyMetrics.SuccessfulRequests),
				fmt.Sprintf("%d", keyMetrics.FailedRequests),
				fmt.Sprintf("%d", keyMetrics.TotalTokensConsumed),
			}
			if err := writer.Write(row); err != nil {
				return nil, fmt.Errorf("failed to write CSV row for key %s: %w", apiKey, err)
			}
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	
	return []byte(result.String()), nil
}

// sanitizeMetrics applies security sanitization and API key masking to metrics data.
func (e *MetricsExporter) sanitizeMetrics(metrics map[string]any) map[string]any {
	if !e.sanitizeData && !e.maskAPIKeys {
		return metrics
	}
	
	sanitized := make(map[string]any, len(metrics))
	
	for key, value := range metrics {
		processedKey := key
		
		// Apply API key masking
		if e.maskAPIKeys {
			processedKey = maskAPIKey(processedKey)
		}
		
		// Apply general sanitization
		if e.sanitizeData {
			processedKey = sanitizeForExport(processedKey)
		}
		
		sanitized[processedKey] = value
	}
	
	return sanitized
}

// maskAPIKey masks an API key for security while preserving some identifiability.
// Shows first 4 and last 4 characters for keys longer than 8 characters.
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	prefix := apiKey[:4]
	suffix := apiKey[len(apiKey)-4:]
	masked := strings.Repeat("*", len(apiKey)-8)
	return prefix + masked + suffix
}

// sanitizeForExport removes potentially dangerous characters for export formats.
// This is more conservative than the collector-level sanitization.
func sanitizeForExport(s string) string {
	// Remove control characters and potentially dangerous patterns
	re := regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]|[<>"'&\\]`)
	return re.ReplaceAllString(s, "_")
}

// prometheusErrorLogger implements prometheus error logging
type prometheusErrorLogger struct{}

func (l *prometheusErrorLogger) Println(v ...interface{}) {
	// In production, you might want to use your application's logger here
	// For now, we silently ignore Prometheus errors to prevent log spam
}

// Backward compatibility functions for existing code

// ExportJSON provides backward compatibility for direct JSON export.
// Deprecated: Use MetricsExporter.ExportJSON() instead.
func ExportJSON(collector *MetricsCollector) []byte {
	if collector == nil {
		return []byte("{}")
	}

	metrics := collector.GetMetrics()
	
	// Convert to structure that matches test expectations (field names not JSON tags)
	compatibleMetrics := make(map[string]any)
	for apiKey, value := range metrics {
		if keyMetrics, ok := value.(*interfaces.KeyMetrics); ok {
			// Convert PerEndpoint
			perEndpoint := make(map[string]any)
			for ep, em := range keyMetrics.PerEndpoint {
				perEndpoint[ep] = map[string]any{
					"TotalRequests": em.TotalRequests,
					"TotalTokens":   em.TotalTokens,
				}
			}
			
			// Convert PerModel
			perModel := make(map[string]any)
			for model, mm := range keyMetrics.PerModel {
				perModel[model] = map[string]any{
					"TotalRequests": mm.TotalRequests,
					"TotalTokens":   mm.TotalTokens,
				}
			}
			
			// Create compatible structure with struct field names
			compatibleMetrics[apiKey] = map[string]any{
				"TotalRequests":       keyMetrics.TotalRequests,
				"SuccessfulRequests":  keyMetrics.SuccessfulRequests,
				"FailedRequests":      keyMetrics.FailedRequests,
				"TotalTokensConsumed": keyMetrics.TotalTokensConsumed,
				"PerEndpoint":         perEndpoint,
				"PerModel":            perModel,
			}
		}
	}
	
	data, err := json.Marshal(compatibleMetrics)
	if err != nil {
		// Return empty JSON on error for backward compatibility
		return []byte("{}")
	}
	return data
}

// PrometheusHandler provides backward compatibility for Prometheus export.
// Deprecated: Use MetricsExporter.ExportPrometheus() instead.
func PrometheusHandler(collector *MetricsCollector) http.Handler {
	if collector == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "No collector available", http.StatusInternalServerError)
		})
	}
	
	exporter := NewMetricsExporter(collector)
	return exporter.ExportPrometheus()
}

// AuthenticatedExportHandler creates an HTTP handler that requires authentication
// and supports multiple export formats based on query parameters.
func AuthenticatedExportHandler(exporter *MetricsExporter, config *interfaces.MetricsConfig, allowedKeys []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication if required
		if config.AuthRequired {
			apiKey := extractBearerToken(r.Header.Get("Authorization"))
			
			if !isAllowedKey(apiKey, allowedKeys) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		
		// Determine export format from query parameter
		format := strings.ToLower(r.URL.Query().Get("format"))
		
		switch format {
		case "csv":
			handleCSVExport(w, exporter, config)
		case "json":
			handleJSONExport(w, exporter, config)
		case "prometheus", "":
			// Default to Prometheus format
			handlePrometheusExport(w, r, exporter, config)
		default:
			http.Error(w, fmt.Sprintf("Unsupported format: %s", format), http.StatusBadRequest)
		}
	})
}

// extractBearerToken extracts the token from a Bearer authorization header
func extractBearerToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return authHeader
}

// handleCSVExport handles CSV format export requests
func handleCSVExport(w http.ResponseWriter, exporter *MetricsExporter, config *interfaces.MetricsConfig) {
	if !config.CSVExportEnabled {
		http.Error(w, "CSV export not enabled", http.StatusForbidden)
		return
	}
	
	data, err := exporter.ExportCSV()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export CSV: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=metrics.csv")
	w.Write(data)
}

// handleJSONExport handles JSON format export requests
func handleJSONExport(w http.ResponseWriter, exporter *MetricsExporter, config *interfaces.MetricsConfig) {
	if !config.JSONExportEnabled {
		http.Error(w, "JSON export not enabled", http.StatusForbidden)
		return
	}
	
	data, err := exporter.ExportJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export JSON: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handlePrometheusExport handles Prometheus format export requests
func handlePrometheusExport(w http.ResponseWriter, r *http.Request, exporter *MetricsExporter, config *interfaces.MetricsConfig) {
	if !config.PrometheusEnabled {
		http.Error(w, "Prometheus export not enabled", http.StatusForbidden)
		return
	}
	
	prometheusHandler := exporter.ExportPrometheus()
	prometheusHandler.ServeHTTP(w, r)
}

// isAllowedKey checks if an API key is in the allowed list using constant-time comparison
func isAllowedKey(apiKey string, allowedKeys []string) bool {
	if len(allowedKeys) == 0 {
		return false
	}
	
	for _, allowed := range allowedKeys {
		if constantTimeCompare(apiKey, allowed) {
			return true
		}
	}
	return false
}

// constantTimeCompare performs constant-time string comparison to prevent timing attacks
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}
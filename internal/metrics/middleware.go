// Package metrics provides comprehensive metrics collection and reporting for the Nexus API gateway.
package metrics

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
)

// Context keys for storing and retrieving metrics data from request context
type contextKey string

const (
	// ModelContextKey stores the AI model name in request context
	ModelContextKey contextKey = "metrics_model"
	// TokensContextKey stores the token count in request context
	TokensContextKey contextKey = "metrics_tokens"
	// APIKeyContextKey stores the API key in request context
	APIKeyContextKey contextKey = "metrics_api_key"
)

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code
// for metrics collection purposes.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader captures the status code and forwards the call
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write captures the response size and forwards the call
func (r *statusRecorder) Write(data []byte) (int, error) {
	size, err := r.ResponseWriter.Write(data)
	r.size += size
	return size, err
}

// Status returns the captured HTTP status code
func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK // Default status if WriteHeader wasn't called
	}
	return r.status
}

// Size returns the total number of bytes written to the response
func (r *statusRecorder) Size() int {
	return r.size
}

// MetricsMiddleware creates HTTP middleware that collects request metrics.
// It wraps handlers to automatically record request duration, status codes,
// and other metrics data extracted from the request context.
func MetricsMiddleware(collector interfaces.MetricsCollector) func(http.Handler) http.Handler {
	if collector == nil {
		// Return pass-through middleware if no collector provided
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Record start time for latency calculation
			startTime := time.Now()

			// Extract API key from various sources
			apiKey := extractAPIKey(r)
			
			// Store API key in context for downstream middleware
			if apiKey != "" {
				ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
				r = r.WithContext(ctx)
			}

			// Determine endpoint path for metrics
			endpoint := sanitizeEndpoint(r.URL.Path)
			
			// Wrap response writer to capture status and size
			recorder := &statusRecorder{ResponseWriter: w, status: 0, size: 0}
			
			// Process request through the chain
			next.ServeHTTP(recorder, r)

			// Calculate request duration
			duration := time.Since(startTime)

			// Extract additional metrics data from context
			model := extractModel(r)
			tokens := extractTokens(r)

			// Record metrics if we have an API key
			if apiKey != "" {
				collector.RecordRequest(
					apiKey,
					endpoint,
					model,
					tokens,
					recorder.Status(),
					duration,
				)
			}
		})
	}
}

// extractAPIKey extracts the API key from the request using multiple strategies.
// It checks the request context first, then falls back to the Authorization header.
func extractAPIKey(r *http.Request) string {
	// First, try to get API key from context (set by auth middleware)
	if apiKey, ok := r.Context().Value(APIKeyContextKey).(string); ok && apiKey != "" {
		return apiKey
	}
	
	// Fall back to Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	// Handle Bearer token format
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	
	// Return header value as-is for other formats
	return auth
}

// extractModel extracts the AI model name from the request context.
// Returns "unknown" if no model information is available.
func extractModel(r *http.Request) string {
	if model, ok := r.Context().Value(ModelContextKey).(string); ok && model != "" {
		return model
	}
	return "unknown"
}

// extractTokens extracts the token count from the request context.
// Returns 0 if no token information is available.
func extractTokens(r *http.Request) int {
	if tokens, ok := r.Context().Value(TokensContextKey).(int); ok && tokens >= 0 {
		return tokens
	}
	return 0
}

// sanitizeEndpoint cleans up endpoint paths for consistent metrics collection.
// It removes query parameters and normalizes the path format.
func sanitizeEndpoint(path string) string {
	if path == "" {
		return "/"
	}
	
	// Remove query parameters for cleaner grouping
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Limit length to prevent extremely long paths from causing issues
	if len(path) > 255 {
		path = path[:255]
	}
	
	return path
}

// Context manipulation functions for storing metrics data

// SetModel stores the AI model name in the request context.
// This should be called by middleware that can determine the model being used.
func SetModel(r *http.Request, model string) *http.Request {
	if model != "" {
		ctx := context.WithValue(r.Context(), ModelContextKey, model)
		return r.WithContext(ctx)
	}
	return r
}

// SetTokens stores the token count in the request context.
// This should be called by middleware that can count or estimate tokens.
func SetTokens(r *http.Request, tokens int) *http.Request {
	if tokens >= 0 {
		ctx := context.WithValue(r.Context(), TokensContextKey, tokens)
		return r.WithContext(ctx)
	}
	return r
}

// SetAPIKey stores the API key in the request context.
// This should be called by authentication middleware.
func SetAPIKey(r *http.Request, apiKey string) *http.Request {
	if apiKey != "" {
		ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
		return r.WithContext(ctx)
	}
	return r
}

// GetModel retrieves the AI model name from the request context.
// Returns empty string if no model is set.
func GetModel(r *http.Request) string {
	if model, ok := r.Context().Value(ModelContextKey).(string); ok {
		return model
	}
	return ""
}

// GetTokens retrieves the token count from the request context.
// Returns -1 if no token count is set.
func GetTokens(r *http.Request) int {
	if tokens, ok := r.Context().Value(TokensContextKey).(int); ok {
		return tokens
	}
	return -1
}

// GetAPIKey retrieves the API key from the request context.
// Returns empty string if no API key is set.
func GetAPIKey(r *http.Request) string {
	if apiKey, ok := r.Context().Value(APIKeyContextKey).(string); ok {
		return apiKey
	}
	return ""
}

// MiddlewareConfig provides configuration options for the metrics middleware
type MiddlewareConfig struct {
	// EnablePathNormalization controls whether endpoint paths are normalized
	EnablePathNormalization bool
	// MaxPathLength limits the maximum length of recorded endpoint paths
	MaxPathLength int
	// SkipHealthChecks controls whether to skip metrics for health check endpoints
	SkipHealthChecks bool
	// HealthCheckPaths lists paths that should be considered health checks
	HealthCheckPaths []string
}

// DefaultMiddlewareConfig returns sensible default configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		EnablePathNormalization: true,
		MaxPathLength:          255,
		SkipHealthChecks:       true,
		HealthCheckPaths:       []string{"/health", "/healthz", "/ping", "/status"},
	}
}

// ConfigurableMetricsMiddleware creates metrics middleware with custom configuration
func ConfigurableMetricsMiddleware(collector interfaces.MetricsCollector, config *MiddlewareConfig) func(http.Handler) http.Handler {
	if collector == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks if configured
			if config.SkipHealthChecks && isHealthCheckPath(r.URL.Path, config.HealthCheckPaths) {
				next.ServeHTTP(w, r)
				return
			}

			startTime := time.Now()

			apiKey := extractAPIKey(r)
			if apiKey != "" {
				ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
				r = r.WithContext(ctx)
			}

			endpoint := r.URL.Path
			if config.EnablePathNormalization {
				endpoint = sanitizeEndpoint(endpoint)
				if len(endpoint) > config.MaxPathLength {
					endpoint = endpoint[:config.MaxPathLength]
				}
			}
			
			recorder := &statusRecorder{ResponseWriter: w, status: 0, size: 0}
			next.ServeHTTP(recorder, r)

			duration := time.Since(startTime)
			model := extractModel(r)
			tokens := extractTokens(r)

			if apiKey != "" {
				collector.RecordRequest(apiKey, endpoint, model, tokens, recorder.Status(), duration)
			}
		})
	}
}

// isHealthCheckPath checks if a path matches any of the health check patterns
func isHealthCheckPath(path string, healthCheckPaths []string) bool {
	for _, healthPath := range healthCheckPaths {
		if path == healthPath || strings.HasPrefix(path, healthPath+"/") {
			return true
		}
	}
	return false
}
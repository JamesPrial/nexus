package interfaces

import (
	"net/http"
	"time"
)

// ConfigLoader handles loading configuration from various sources
type ConfigLoader interface {
	Load() (*Config, error)
}

// Config represents the application configuration
type Config struct {
	ListenPort int
	TargetURL  string
	LogLevel   string `yaml:"log_level"`
	APIKeys    map[string]string
	Limits     Limits
	TLS        *TLSConfig
	Metrics    MetricsConfig `yaml:"metrics"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

type Limits struct {
	RequestsPerSecond    int
	Burst                int
	ModelTokensPerMinute int
}

// RateLimiter provides rate limiting functionality
type RateLimiter interface {
	// Middleware returns HTTP middleware that enforces rate limits
	Middleware(next http.Handler) http.Handler

	// GetLimit returns the current limit for an API key
	GetLimit(apiKey string) (allowed bool, remaining int)

	// Reset clears the rate limit state for an API key
	Reset(apiKey string)
}

// TokenCounter calculates token usage from HTTP requests
type TokenCounter interface {
	// CountTokens estimates the number of tokens in a request
	CountTokens(r *http.Request) (int, error)
}

// Proxy handles forwarding requests to upstream services
type Proxy interface {
	// ServeHTTP implements http.Handler to proxy requests
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// SetTarget changes the upstream target URL
	SetTarget(targetURL string) error
}

// Gateway represents the main gateway service
type Gateway interface {
	// Start begins serving HTTP requests
	Start() error

	// Stop gracefully shuts down the gateway
	Stop() error

	// Health returns the health status of the gateway
	Health() map[string]any
}

// Logger provides structured logging
type Logger interface {
	Debug(msg string, fields map[string]any)
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

// KeyManager manages API key mapping and validation
type KeyManager interface {
	// ValidateClientKey checks if a client API key is valid
	ValidateClientKey(clientKey string) bool
	
	// GetUpstreamKey returns the upstream API key for a client key
	GetUpstreamKey(clientKey string) (string, error)
	
	// IsConfigured returns true if API key management is configured
	IsConfigured() bool
}

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

// MetricsExporter exports metrics in various formats
type MetricsExporter interface {
	// ExportJSON exports metrics as JSON
	ExportJSON() ([]byte, error)
	
	// ExportPrometheus returns an HTTP handler for Prometheus format
	ExportPrometheus() http.Handler
	
	// ExportCSV exports metrics as CSV
	ExportCSV() ([]byte, error)
}

// MetricsMiddleware provides HTTP middleware for metrics collection
type MetricsMiddleware interface {
	// Middleware returns HTTP middleware that collects metrics
	Middleware(next http.Handler) http.Handler
}

// KeyMetrics holds aggregated metrics for a single API key
type KeyMetrics struct {
	TotalRequests       int64 `json:"total_requests"`
	SuccessfulRequests  int64 `json:"successful_requests"`
	FailedRequests      int64 `json:"failed_requests"`
	TotalTokensConsumed int64 `json:"total_tokens_consumed"`
	PerEndpoint         map[string]*EndpointMetrics `json:"per_endpoint"`
	PerModel            map[string]*ModelMetrics `json:"per_model"`
}

// EndpointMetrics holds metrics for a specific endpoint
type EndpointMetrics struct {
	TotalRequests int64 `json:"total_requests"`
	TotalTokens   int64 `json:"total_tokens"`
}

// ModelMetrics holds metrics for a specific model
type ModelMetrics struct {
	TotalRequests int64 `json:"total_requests"`
	TotalTokens   int64 `json:"total_tokens"`
}

// MetricsConfig represents metrics system configuration
type MetricsConfig struct {
	Enabled           bool   `yaml:"enabled"`
	MetricsEndpoint   string `yaml:"metrics_endpoint"`
	PrometheusEnabled bool   `yaml:"prometheus_enabled"`
	JSONExportEnabled bool   `yaml:"json_export_enabled"`
	CSVExportEnabled  bool   `yaml:"csv_export_enabled"`
	AuthRequired      bool   `yaml:"auth_required"`
	MaskAPIKeys       bool   `yaml:"mask_api_keys"`
}

// Container holds application dependencies and provides dependency injection
type Container interface {
	// Config returns the loaded configuration
	Config() *Config

	// Logger returns the logger instance
	Logger() Logger

	// BuildHandler creates the complete middleware chain
	BuildHandler() http.Handler

	// MetricsCollector returns the metrics collector instance
	MetricsCollector() MetricsCollector

	// MetricsMiddleware returns the metrics middleware function
	MetricsMiddleware() func(http.Handler) http.Handler
}

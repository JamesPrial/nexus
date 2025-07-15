package interfaces

import (
	"net/http"
)

// ConfigLoader handles loading configuration from various sources
type ConfigLoader interface {
	Load() (*Config, error)
}

// Config represents the application configuration
type Config struct {
	ListenPort int
	TargetURL  string
	Limits     Limits
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

// Container holds application dependencies and provides dependency injection
type Container interface {
	// Config returns the loaded configuration
	Config() *Config
	
	// Logger returns the logger instance
	Logger() Logger
	
	// BuildHandler creates the complete middleware chain
	BuildHandler() http.Handler
}
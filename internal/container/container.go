package container

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/jamesprial/nexus/config"
	"github.com/jamesprial/nexus/internal/auth"
	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/logging"
	"github.com/jamesprial/nexus/internal/metrics"
	"github.com/jamesprial/nexus/internal/middleware"
	"github.com/jamesprial/nexus/internal/proxy"
	"golang.org/x/time/rate"
)

// Container holds all application dependencies
type Container struct {
	configLoader      interfaces.ConfigLoader
	rateLimiter       interfaces.RateLimiter
	tokenLimiter      interfaces.RateLimiter
	tokenCounter      interfaces.TokenCounter
	proxy             interfaces.Proxy
	logger            interfaces.Logger
	config            *interfaces.Config
	keyManager        interfaces.KeyManager
	authMiddleware    *auth.AuthMiddleware
	metricsCollector  interfaces.MetricsCollector
	metricsMiddleware func(http.Handler) http.Handler
}

// New creates a new dependency injection container
func New() *Container {
	return &Container{}
}

// SetConfigLoader sets the configuration loader
func (c *Container) SetConfigLoader(loader interfaces.ConfigLoader) {
	c.configLoader = loader
}

// SetLogger sets the logger implementation
func (c *Container) SetLogger(logger interfaces.Logger) {
	c.logger = logger
}

// ConfigLoader returns the configuration loader
func (c *Container) ConfigLoader() interfaces.ConfigLoader {
	return c.configLoader
}

// Logger returns the logger
func (c *Container) Logger() interfaces.Logger {
	return c.logger
}

// Config returns the loaded configuration
func (c *Container) Config() *interfaces.Config {
	return c.config
}

// RateLimiter returns the request rate limiter
func (c *Container) RateLimiter() interfaces.RateLimiter {
	return c.rateLimiter
}

// TokenLimiter returns the token rate limiter
func (c *Container) TokenLimiter() interfaces.RateLimiter {
	return c.tokenLimiter
}

// TokenCounter returns the token counter
func (c *Container) TokenCounter() interfaces.TokenCounter {
	return c.tokenCounter
}

// Proxy returns the proxy handler
func (c *Container) Proxy() interfaces.Proxy {
	return c.proxy
}

// MetricsCollector returns the metrics collector instance
func (c *Container) MetricsCollector() interfaces.MetricsCollector {
	return c.metricsCollector
}

// MetricsMiddleware returns the metrics middleware function
func (c *Container) MetricsMiddleware() func(http.Handler) http.Handler {
	return c.metricsMiddleware
}

// Initialize loads configuration and sets up all dependencies
func (c *Container) Initialize() error {
	// Load configuration
	if c.configLoader == nil {
		return fmt.Errorf("config loader not set")
	}

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	c.config = cfg

	// Set up logger if not already set
	if c.logger == nil {
		c.logger = logging.NewSlogLogger(cfg.LogLevel)
	}

	// Set up key manager and auth middleware
	// Convert from interfaces.Config to config.Config to maintain compatibility
	configForAuth := &config.Config{
		APIKeys: cfg.APIKeys,
	}
	c.keyManager = auth.NewFileKeyManager(configForAuth)
	c.authMiddleware = auth.NewAuthMiddleware(c.keyManager, c.logger)

	// Set up token counter
	c.tokenCounter = &proxy.DefaultTokenCounter{}

	// Set up rate limiter with TTL (1 hour)
	ttl := 1 * time.Hour
	perClientLimiter := proxy.NewPerClientRateLimiterWithTTL(
		rate.Limit(cfg.Limits.RequestsPerSecond),
		cfg.Limits.Burst,
		ttl,
		c.logger,
	)
	c.rateLimiter = perClientLimiter

	// Start cleanup routine for per-client rate limiter
	stopChan := make(chan struct{})
	go perClientLimiter.StartCleanup(5*time.Minute, stopChan)

	// Set up token limiter with proper burst calculation and TTL
	tokenBurst := max(cfg.Limits.ModelTokensPerMinute/6, 100)

	tokenLimiter := proxy.NewTokenLimiterWithTTL(
		cfg.Limits.ModelTokensPerMinute,
		tokenBurst,
		c.tokenCounter,
		ttl,
		c.logger,
	)
	c.tokenLimiter = tokenLimiter

	// Start cleanup routine for token limiter
	stopChan2 := make(chan struct{})
	go tokenLimiter.StartCleanup(5*time.Minute, stopChan2)

	// Set up proxy
	target, err := url.Parse(cfg.TargetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	c.proxy = &proxy.HTTPProxy{
		ReverseProxy: reverseProxy,
		Logger:       c.logger,
	}

	// Set up metrics collector if enabled
	if cfg.Metrics.Enabled {
		c.metricsCollector = metrics.NewMetricsCollector()
		c.metricsMiddleware = metrics.MetricsMiddleware(c.metricsCollector)
	}

	return nil
}

// BuildHandler creates the complete middleware chain
func (c *Container) BuildHandler() http.Handler {
	if c.proxy == nil {
		panic("container not initialized")
	}

	// Build middleware chain: validation -> auth -> metrics -> rateLimiter -> tokenLimiter -> proxy
	var handler http.Handler = http.HandlerFunc(c.proxy.ServeHTTP)
	handler = c.tokenLimiter.Middleware(handler)
	handler = c.rateLimiter.Middleware(handler)
	
	// Add metrics middleware if available
	if c.metricsMiddleware != nil {
		handler = c.metricsMiddleware(handler)
	}
	
	handler = c.authMiddleware.Middleware(handler)
	
	// Add request validation as the outermost middleware
	// Default to 10MB max body size
	validationMiddleware := middleware.NewRequestValidationMiddleware(10 * 1024 * 1024)
	handler = validationMiddleware(handler)

	return handler
}

package container

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/logging"
	"github.com/jamesprial/nexus/internal/proxy"
	"golang.org/x/time/rate"
)

// Container holds all application dependencies
type Container struct {
	configLoader interfaces.ConfigLoader
	rateLimiter  interfaces.RateLimiter
	tokenLimiter interfaces.RateLimiter
	tokenCounter interfaces.TokenCounter
	proxy        interfaces.Proxy
	logger       interfaces.Logger
	config       *interfaces.Config
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

	// Set up token counter
	c.tokenCounter = &proxy.DefaultTokenCounter{}

	// Set up rate limiter
	c.rateLimiter = proxy.NewPerClientRateLimiterWithLogger(
		rate.Limit(cfg.Limits.RequestsPerSecond),
		cfg.Limits.Burst,
		c.logger,
	)

	// Set up token limiter with proper burst calculation
	tokenBurst := cfg.Limits.ModelTokensPerMinute / 6
	if tokenBurst < 100 {
		tokenBurst = 100
	}

	c.tokenLimiter = proxy.NewTokenLimiterWithDeps(
		cfg.Limits.ModelTokensPerMinute,
		tokenBurst,
		c.tokenCounter,
		c.logger,
	)

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

	return nil
}

// BuildHandler creates the complete middleware chain
func (c *Container) BuildHandler() http.Handler {
	if c.proxy == nil {
		panic("container not initialized")
	}

	// Build middleware chain: rateLimiter -> tokenLimiter -> proxy
	var handler http.Handler = http.HandlerFunc(c.proxy.ServeHTTP)
	handler = c.tokenLimiter.Middleware(handler)
	handler = c.rateLimiter.Middleware(handler)

	return handler
}

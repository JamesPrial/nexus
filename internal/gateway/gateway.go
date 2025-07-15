package gateway

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/jamesprial/nexus/config"
	"github.com/jamesprial/nexus/internal/proxy"
	"golang.org/x/time/rate"
)

func Run() error {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	target, err := url.Parse(cfg.TargetURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	rateLimiter := proxy.NewRateLimiter(
		rate.Limit(cfg.Limits.RequestsPerSecond),
		cfg.Limits.Burst,
	)

	// Create token limiter with proper burst calculation
	// Burst should be reasonable for minute-based limits (e.g., 10% of minute limit)
	tokenBurst := cfg.Limits.ModelTokensPerMinute / 6 // ~10 seconds worth of tokens
	if tokenBurst < 100 {
		tokenBurst = 100 // Minimum burst of 100 tokens
	}
	
	tokenLimiter := proxy.NewTokenLimiter(
		cfg.Limits.ModelTokensPerMinute, // tokens per minute
		tokenBurst,                      // burst allowance
	)

	http.Handle("/", tokenLimiter.Middleware(rateLimiter.Middleware(reverseProxy)))

	listenAddr := fmt.Sprintf(":%d", cfg.ListenPort)
	log.Printf("Starting Nexus gateway on %s", listenAddr)
	return http.ListenAndServe(listenAddr, nil)
}

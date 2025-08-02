package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/metrics"
)

// Service implements interfaces.Gateway using dependency injection
type Service struct {
	container interfaces.Container
	server    *http.Server
	logger    interfaces.Logger
}

// NewService creates a new gateway service with dependency injection
func NewService(container interfaces.Container) interfaces.Gateway {
	return &Service{
		container: container,
		logger:    container.Logger(),
	}
}

// Start implements interfaces.Gateway.Start
func (s *Service) Start() error {
	config := s.container.Config()
	if config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Create main handler
	mainHandler := s.container.BuildHandler()
	
	// Create mux for routing
	mux := http.NewServeMux()
	
	// Register metrics endpoints if metrics are enabled
	if config.Metrics.Enabled {
		s.registerMetricsEndpoints(mux, config)
	}
	
	// Register catch-all handler for proxy
	mux.Handle("/", mainHandler)
	
	listenAddr := fmt.Sprintf(":%d", config.ListenPort)
	
	s.server = &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if s.logger != nil {
		s.logger.Info("Starting Nexus gateway", map[string]any{
			"listen_addr": listenAddr,
			"target_url":  config.TargetURL,
		})
	}

	// Start server in goroutine so Start() doesn't block
	errCh := make(chan error, 1)
	go func() {
		var err error
		if config.TLS != nil && config.TLS.Enabled {
			if s.logger != nil {
				s.logger.Info("Starting HTTPS server", map[string]any{
					"cert_file": config.TLS.CertFile,
					"key_file":  config.TLS.KeyFile,
				})
			}
			err = s.server.ListenAndServeTLS(config.TLS.CertFile, config.TLS.KeyFile)
		} else {
			if s.logger != nil {
				s.logger.Info("Starting HTTP server (no TLS)", map[string]any{})
			}
			err = s.server.ListenAndServe()
		}
		
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Give the server a moment to start
	select {
	case err := <-errCh:
		return fmt.Errorf("failed to start server: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
		return nil
	}
}

// Stop implements interfaces.Gateway.Stop
func (s *Service) Stop() error {
	if s.server == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("Stopping Nexus gateway", map[string]any{})
	}

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// Health implements interfaces.Gateway.Health
func (s *Service) Health() map[string]any {
	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	config := s.container.Config()
	if config != nil {
		health["config"] = map[string]any{
			"listen_port": config.ListenPort,
			"target_url":  config.TargetURL,
		}
	}

	return health
}

// registerMetricsEndpoints registers metrics endpoints with the mux
func (s *Service) registerMetricsEndpoints(mux *http.ServeMux, config *interfaces.Config) {
	collector := s.container.MetricsCollector()
	if collector == nil {
		return
	}

	// Create exporter
	exporter := metrics.NewMetricsExporter(collector)
	
	// Set up authentication keys (empty if auth not required)
	var allowedKeys []string
	if config.Metrics.AuthRequired {
		// For now, use all API keys as allowed keys
		// In production, you might want separate metrics keys
		for _, key := range config.APIKeys {
			allowedKeys = append(allowedKeys, key)
		}
	}

	// Register authenticated metrics handler
	metricsEndpoint := config.Metrics.MetricsEndpoint
	if metricsEndpoint == "" {
		metricsEndpoint = "/metrics"
	}
	
	metricsHandler := metrics.AuthenticatedExportHandler(exporter, &config.Metrics, allowedKeys)
	mux.Handle(metricsEndpoint, metricsHandler)

	if s.logger != nil {
		s.logger.Info("Registered metrics endpoints", map[string]any{
			"endpoint":            metricsEndpoint,
			"prometheus_enabled":  config.Metrics.PrometheusEnabled,
			"json_export_enabled": config.Metrics.JSONExportEnabled,
			"csv_export_enabled":  config.Metrics.CSVExportEnabled,
			"auth_required":       config.Metrics.AuthRequired,
		})
	}
}
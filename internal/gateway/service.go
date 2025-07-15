package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
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

	handler := s.container.BuildHandler()
	
	listenAddr := fmt.Sprintf(":%d", config.ListenPort)
	
	s.server = &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if s.logger != nil {
		s.logger.Info("Starting Nexus gateway", map[string]interface{}{
			"listen_addr": listenAddr,
			"target_url":  config.TargetURL,
		})
	}

	// Start server in goroutine so Start() doesn't block
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
		s.logger.Info("Stopping Nexus gateway", map[string]interface{}{})
	}

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// Health implements interfaces.Gateway.Health
func (s *Service) Health() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	config := s.container.Config()
	if config != nil {
		health["config"] = map[string]interface{}{
			"listen_port": config.ListenPort,
			"target_url":  config.TargetURL,
		}
	}

	return health
}
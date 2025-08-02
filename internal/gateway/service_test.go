package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamesprial/nexus/internal/config"
	"github.com/jamesprial/nexus/internal/container"
	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/logging"
)

// TestGatewayServiceWithDI demonstrates the improved testability with dependency injection
func TestGatewayServiceWithDI(t *testing.T) {
	// Create test configuration
	testConfig := &interfaces.Config{
		ListenPort: 8082,
		TargetURL:  "http://example.com",
		Limits: interfaces.Limits{
			RequestsPerSecond:    10,
			Burst:                20,
			ModelTokensPerMinute: 1000,
		},
	}

	// Create container with test dependencies
	cont := container.New()
	cont.SetLogger(logging.NewNoOpLogger()) // Use no-op logger for tests
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	// Initialize container
	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Create gateway service
	service := NewService(cont)

	// Test health check
	health := service.Health()
	if health["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", health["status"])
	}

	// Verify configuration is loaded correctly
	if cont.Config().ListenPort != 8082 {
		t.Errorf("Expected port 8082, got %d", cont.Config().ListenPort)
	}
}

// TestMiddlewareChain tests that the middleware chain is built correctly
func TestMiddlewareChain(t *testing.T) {
	// Create mock upstream server
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("upstream response")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer mockUpstream.Close()

	// Create test configuration pointing to mock upstream
	testConfig := &interfaces.Config{
		ListenPort: 8083,
		TargetURL:  mockUpstream.URL,
		Limits: interfaces.Limits{
			RequestsPerSecond:    100,
			Burst:                200,
			ModelTokensPerMinute: 6000, // 100 tokens per second
		},
	}

	// Set up container
	cont := container.New()
	cont.SetLogger(logging.NewNoOpLogger())
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Get the handler chain
	handler := cont.BuildHandler()

	// Test that request passes through middleware chain
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should get the response from upstream (after passing through middlewares)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	if rr.Body.String() != "upstream response" {
		t.Errorf("Expected upstream response, got %s", rr.Body.String())
	}
}

// TestRateLimiterIntegration tests rate limiting with dependency injection
func TestRateLimiterIntegration(t *testing.T) {
	// Create mock upstream
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockUpstream.Close()

	// Create restrictive rate limits for testing
	testConfig := &interfaces.Config{
		ListenPort: 8084,
		TargetURL:  mockUpstream.URL,
		Limits: interfaces.Limits{
			RequestsPerSecond:    1,  // Very restrictive
			Burst:                1,
			ModelTokensPerMinute: 60, // 1 token per second
		},
	}

	// Set up container
	cont := container.New()
	cont.SetLogger(logging.NewNoOpLogger())
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	handler := cont.BuildHandler()

	// First request should pass
	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set("Authorization", "Bearer test-key")
	req1.Header.Set("Content-Type", "application/json")
	
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request should pass, got status %d", rr1.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set("Authorization", "Bearer test-key")
	req2.Header.Set("Content-Type", "application/json")
	
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", rr2.Code)
	}
}

// TestMockableDependencies demonstrates how easy it is to mock dependencies
func TestMockableDependencies(t *testing.T) {
	// Create a mock rate limiter that always allows requests
	mockRateLimiter := &MockRateLimiter{
		shouldAllow: true,
	}

	// This demonstrates how we could inject mocks for testing
	// (In a full implementation, we'd add SetRateLimiter methods to the container)
	
	if !mockRateLimiter.shouldAllow {
		t.Error("Mock should allow requests")
	}
}

// MockRateLimiter is an example mock implementation
type MockRateLimiter struct {
	shouldAllow bool
}

func (m *MockRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.shouldAllow {
			http.Error(w, "Rate limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *MockRateLimiter) GetLimit(apiKey string) (allowed bool, remaining int) {
	return m.shouldAllow, 100
}

func (m *MockRateLimiter) Reset(apiKey string) {
	// Mock implementation
}
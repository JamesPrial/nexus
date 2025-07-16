package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/config"
	"github.com/jamesprial/nexus/internal/container"
	"github.com/jamesprial/nexus/internal/gateway"
	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/logging"
)

func TestGatewayIntegration(t *testing.T) {
	// --- Setup a mock upstream server ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// --- Create test configuration ---
	testConfig := &interfaces.Config{
		ListenPort: 8081,
		TargetURL:  mockServer.URL,
		Limits: interfaces.Limits{
			RequestsPerSecond:    1,
			Burst:                1,
			ModelTokensPerMinute: 1000,
		},
	}

	// --- Set up dependency injection container ---
	cont := container.New()
	cont.SetLogger(logging.NewNoOpLogger()) // Use silent logger for tests
	cont.SetConfigLoader(config.NewMemoryLoader(testConfig))

	// Initialize container
	if err := cont.Initialize(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Create gateway service
	gatewayService := gateway.NewService(cont)

	// Start the gateway
	if err := gatewayService.Start(); err != nil {
		t.Fatalf("Failed to start gateway: %v", err)
	}
	defer func() {
		if err := gatewayService.Stop(); err != nil {
			t.Errorf("Failed to stop gateway service: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// --- Create a client to send requests to the gateway ---
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8081", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "test-key")

	// --- Test 1: First request should be allowed ---
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	// --- Test 2: Second request immediately after should be denied ---
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status TooManyRequests, got %d", resp.StatusCode)
	}

	// --- Test 3: Wait for the rate limiter to allow another request ---
	time.Sleep(1 * time.Second)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK after waiting, got %d", resp.StatusCode)
	}

	// Wait for the global rate limiter to replenish before testing the next client
	time.Sleep(1 * time.Second)

	// --- Test 4: Test per-client RPS limiting ---
	// Wait for global limiter to replenish if needed
	time.Sleep(1 * time.Second)

	// Create requests with different API keys
	reqKey1, err := http.NewRequest("GET", "http://localhost:8081", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	reqKey1.Header.Set("Authorization", "key1")

	reqKey2, err := http.NewRequest("GET", "http://localhost:8081", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	reqKey2.Header.Set("Authorization", "key2")

	// Key1: First request allowed
	resp, err = client.Do(reqKey1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Key1 first: Expected OK, got %d", resp.StatusCode)
	}

	// Key1: Second immediate request should be denied (RPS=1, burst=1)
	resp, err = client.Do(reqKey1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Key1 second: Expected 429, got %d", resp.StatusCode)
	}

	// Key2: Should have independent limit, first request allowed despite Key1's denial
	resp, err = client.Do(reqKey2)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Key2 first: Expected OK, got %d", resp.StatusCode)
	}

	// Key2: Second immediate request denied
	resp, err = client.Do(reqKey2)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Key2 second: Expected 429, got %d", resp.StatusCode)
	}

	// --- Test 5: Test token limiter ---
	// (Existing token limiter test can remain, or expand if needed)
	// Create a new client for a new API key
	client2 := &http.Client{}
	req, err = http.NewRequest("GET", "http://localhost:8081", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "test-key-2")

	// First request should be allowed
	resp, err = client2.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	// Second request should be denied
	resp, err = client2.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status TooManyRequests, got %d", resp.StatusCode)
	}
}

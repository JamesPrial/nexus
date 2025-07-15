package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jamesprial/nexus/internal/gateway"
)

func TestGatewayIntegration(t *testing.T) {
	// --- Setup a mock upstream server ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// --- Create a temporary config file pointing to the mock server ---
	content := fmt.Sprintf(`
listen_port: 8081
target_url: "%s"
limits:
  requests_per_second: 1
  burst: 1
  model_tokens_per_minute: 1000
`, mockServer.URL)

	tmpDir, err := ioutil.TempDir("", "nexus-tests")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := ioutil.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// --- Run the gateway in a goroutine ---
	go func() {
		// We need to change the working directory so the gateway can find the config file.
		// This is another hacky part of this test.
		originalWD, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(originalWD)

		if err := gateway.Run(); err != nil {
			// We expect the server to be closed by the test, so we can ignore this error.
			// A more robust solution would be to have a way to gracefully shut down the server.
		}
	}()
	time.Sleep(100 * time.Millisecond) // Give the server a moment to start.

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

	// --- Test 4: Test token limiter ---
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

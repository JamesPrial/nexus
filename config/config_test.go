package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file for testing.
	content := []byte(`
listen_port: 8081
target_url: "http://localhost:9999"
limits:
  requests_per_second: 5
  burst: 10
  model_tokens_per_minute: 1000
`)
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the config from the temporary file.
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	// Check that the values were loaded correctly.
	if cfg.ListenPort != 8081 {
		t.Errorf("Expected ListenPort 8081, got %d", cfg.ListenPort)
	}
	if cfg.TargetURL != "http://localhost:9999" {
		t.Errorf("Expected TargetURL http://localhost:9999, got %s", cfg.TargetURL)
	}
	if cfg.Limits.RequestsPerSecond != 5 {
		t.Errorf("Expected RequestsPerSecond 5, got %d", cfg.Limits.RequestsPerSecond)
	}
	if cfg.Limits.Burst != 10 {
		t.Errorf("Expected Burst 10, got %d", cfg.Limits.Burst)
	}
	if cfg.Limits.ModelTokensPerMinute != 1000 {
		t.Errorf("Expected ModelTokensPerMinute 1000, got %d", cfg.Limits.ModelTokensPerMinute)
	}
}

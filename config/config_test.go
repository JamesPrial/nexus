package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		validate  func(*testing.T, *Config)
		expectErr bool
	}{
		{
			name: "basic configuration",
			content: `
listen_port: 8081
target_url: "http://localhost:9999"
limits:
  requests_per_second: 5
  burst: 10
  model_tokens_per_minute: 1000
`,
			validate: func(t *testing.T, cfg *Config) {
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
			},
		},
		{
			name: "with log level",
			content: `
listen_port: 8080
target_url: "http://example.com"
log_level: "debug"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.LogLevel != "debug" {
					t.Errorf("Expected LogLevel debug, got %s", cfg.LogLevel)
				}
			},
		},
		{
			name: "with API keys",
			content: `
listen_port: 8080
target_url: "http://example.com"
api_keys:
  "client1": "upstream1"
  "client2": "upstream2"
  "special-chars_123": "sk-1234567890"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
`,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.APIKeys) != 3 {
					t.Errorf("Expected 3 API keys, got %d", len(cfg.APIKeys))
				}
				if cfg.APIKeys["client1"] != "upstream1" {
					t.Errorf("Expected client1 -> upstream1, got %s", cfg.APIKeys["client1"])
				}
				if cfg.APIKeys["client2"] != "upstream2" {
					t.Errorf("Expected client2 -> upstream2, got %s", cfg.APIKeys["client2"])
				}
				if cfg.APIKeys["special-chars_123"] != "sk-1234567890" {
					t.Errorf("Expected special-chars_123 -> sk-1234567890, got %s", cfg.APIKeys["special-chars_123"])
				}
			},
		},
		{
			name: "with TLS configuration",
			content: `
listen_port: 8443
target_url: "https://example.com"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
tls:
  enabled: true
  cert_file: "/etc/ssl/cert.pem"
  key_file: "/etc/ssl/key.pem"
`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.TLS == nil {
					t.Fatal("Expected TLS configuration, got nil")
				}
				if !cfg.TLS.Enabled {
					t.Error("Expected TLS to be enabled")
				}
				if cfg.TLS.CertFile != "/etc/ssl/cert.pem" {
					t.Errorf("Expected cert_file /etc/ssl/cert.pem, got %s", cfg.TLS.CertFile)
				}
				if cfg.TLS.KeyFile != "/etc/ssl/key.pem" {
					t.Errorf("Expected key_file /etc/ssl/key.pem, got %s", cfg.TLS.KeyFile)
				}
			},
		},
		{
			name: "minimal configuration",
			content: `
listen_port: 8080
target_url: "http://localhost:9999"
limits:
  requests_per_second: 1
  burst: 1
  model_tokens_per_minute: 100
`,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.APIKeys != nil && len(cfg.APIKeys) > 0 {
					t.Error("Expected no API keys in minimal config")
				}
				if cfg.LogLevel != "" {
					t.Error("Expected empty log level in minimal config")
				}
				if cfg.TLS != nil {
					t.Error("Expected no TLS config in minimal config")
				}
			},
		},
		{
			name: "invalid YAML syntax",
			content: `
listen_port: 8080
target_url: "http://localhost:9999"
  invalid yaml indentation
limits:
  requests_per_second: 1
`,
			expectErr: true,
		},
		{
			name: "empty file",
			content: "",
			validate: func(t *testing.T, cfg *Config) {
				// Should load with zero values
				if cfg.ListenPort != 0 {
					t.Errorf("Expected ListenPort 0, got %d", cfg.ListenPort)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Load the config
			cfg, err := Load(tmpfile.Name())

			// Check error expectations
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate the loaded config
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoad_FileErrors(t *testing.T) {
	tests := []struct {
		name string
		setup func() string
		expectErr bool
	}{
		{
			name: "non-existent file",
			setup: func() string {
				return "/path/that/does/not/exist/config.yaml"
			},
			expectErr: true,
		},
		{
			name: "directory instead of file",
			setup: func() string {
				tmpdir, err := os.MkdirTemp("", "config-test-")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				return tmpdir
			},
			expectErr: true,
		},
		{
			name: "no read permissions",
			setup: func() string {
				tmpfile, err := os.CreateTemp("", "config-*.yaml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tmpfile.Close()
				
				// Remove read permissions
				if err := os.Chmod(tmpfile.Name(), 0000); err != nil {
					t.Fatalf("Failed to change permissions: %v", err)
				}
				
				return tmpfile.Name()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			
			// Clean up if it's a temp file/dir
			if path != "/path/that/does/not/exist/config.yaml" {
				defer os.RemoveAll(path)
			}

			_, err := Load(path)
			
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test that a Config struct has sensible zero values
	var cfg Config
	
	if cfg.ListenPort != 0 {
		t.Errorf("Expected default ListenPort 0, got %d", cfg.ListenPort)
	}
	if cfg.TargetURL != "" {
		t.Errorf("Expected default TargetURL empty, got %s", cfg.TargetURL)
	}
	if cfg.LogLevel != "" {
		t.Errorf("Expected default LogLevel empty, got %s", cfg.LogLevel)
	}
	if cfg.APIKeys != nil {
		t.Error("Expected default APIKeys to be nil")
	}
	if cfg.Limits.RequestsPerSecond != 0 {
		t.Errorf("Expected default RequestsPerSecond 0, got %d", cfg.Limits.RequestsPerSecond)
	}
	if cfg.Limits.Burst != 0 {
		t.Errorf("Expected default Burst 0, got %d", cfg.Limits.Burst)
	}
	if cfg.Limits.ModelTokensPerMinute != 0 {
		t.Errorf("Expected default ModelTokensPerMinute 0, got %d", cfg.Limits.ModelTokensPerMinute)
	}
	if cfg.TLS != nil {
		t.Error("Expected default TLS to be nil")
	}
}

func TestLoad_ComplexScenarios(t *testing.T) {
	t.Run("overrides and defaults", func(t *testing.T) {
		content := `
listen_port: 8080
target_url: "http://localhost:9999"
log_level: "info"
api_keys:
  "client1": "upstream1"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
tls:
  enabled: false
  cert_file: ""
  key_file: ""
`
		tmpfile, err := os.CreateTemp("", "config-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpfile.Close()

		cfg, err := Load(tmpfile.Name())
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify TLS is present but disabled
		if cfg.TLS == nil {
			t.Error("Expected TLS config to be present")
		} else if cfg.TLS.Enabled {
			t.Error("Expected TLS to be disabled")
		}
	})

	t.Run("unicode and special characters", func(t *testing.T) {
		content := `
listen_port: 8080
target_url: "http://localhost:9999"
api_keys:
  "client-中文": "upstream-日本語"
  "client@example.com": "sk-special!@#$%^&*()"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
`
		tmpfile, err := os.CreateTemp("", "config-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpfile.Close()

		cfg, err := Load(tmpfile.Name())
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if cfg.APIKeys["client-中文"] != "upstream-日本語" {
			t.Error("Failed to handle unicode characters in API keys")
		}
		if cfg.APIKeys["client@example.com"] != "sk-special!@#$%^&*()" {
			t.Error("Failed to handle special characters in API keys")
		}
	})
}

func BenchmarkLoad(b *testing.B) {
	// Create a test config file
	content := `
listen_port: 8080
target_url: "http://localhost:9999"
log_level: "info"
api_keys:
  "client1": "upstream1"
  "client2": "upstream2"
  "client3": "upstream3"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
tls:
  enabled: true
  cert_file: "/etc/ssl/cert.pem"
  key_file: "/etc/ssl/key.pem"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(tmpfile.Name())
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

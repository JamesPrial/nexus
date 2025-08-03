package config

import (
	"os"
	"testing"

	"github.com/jamesprial/nexus/config"
	"github.com/jamesprial/nexus/internal/interfaces"
)

func TestFileLoader_Load(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		validate  func(*testing.T, *interfaces.Config)
		expectErr bool
	}{
		{
			name: "complete configuration",
			content: `
listen_port: 8080
target_url: "http://example.com"
log_level: "debug"
api_keys:
  "client1": "upstream1"
  "client2": "upstream2"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 6000
tls:
  enabled: true
  cert_file: "/etc/ssl/cert.pem"
  key_file: "/etc/ssl/key.pem"
`,
			validate: func(t *testing.T, cfg *interfaces.Config) {
				if cfg.ListenPort != 8080 {
					t.Errorf("Expected ListenPort 8080, got %d", cfg.ListenPort)
				}
				if cfg.TargetURL != "http://example.com" {
					t.Errorf("Expected TargetURL http://example.com, got %s", cfg.TargetURL)
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("Expected LogLevel debug, got %s", cfg.LogLevel)
				}
				if len(cfg.APIKeys) != 2 {
					t.Errorf("Expected 2 API keys, got %d", len(cfg.APIKeys))
				}
				if cfg.Limits.RequestsPerSecond != 10 {
					t.Errorf("Expected RequestsPerSecond 10, got %d", cfg.Limits.RequestsPerSecond)
				}
				if cfg.Limits.Burst != 20 {
					t.Errorf("Expected Burst 20, got %d", cfg.Limits.Burst)
				}
				if cfg.Limits.ModelTokensPerMinute != 6000 {
					t.Errorf("Expected ModelTokensPerMinute 6000, got %d", cfg.Limits.ModelTokensPerMinute)
				}
				if cfg.TLS == nil {
					t.Fatal("Expected TLS config, got nil")
				}
				if !cfg.TLS.Enabled {
					t.Error("Expected TLS to be enabled")
				}
				if cfg.TLS.CertFile != "/etc/ssl/cert.pem" {
					t.Errorf("Expected CertFile /etc/ssl/cert.pem, got %s", cfg.TLS.CertFile)
				}
				if cfg.TLS.KeyFile != "/etc/ssl/key.pem" {
					t.Errorf("Expected KeyFile /etc/ssl/key.pem, got %s", cfg.TLS.KeyFile)
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
			validate: func(t *testing.T, cfg *interfaces.Config) {
				if cfg.LogLevel != "" {
					t.Error("Expected empty LogLevel")
				}
				if len(cfg.APIKeys) > 0 {
					t.Error("Expected no API keys")
				}
				if cfg.TLS != nil {
					t.Error("Expected no TLS config")
				}
			},
		},
		{
			name:      "non-existent file",
			content:   "", // Won't be used
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			
			if !tt.expectErr {
				// Create temp file
				tmpfile, err := os.CreateTemp("", "config-*.yaml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer func() { _ = os.Remove(tmpfile.Name()) }()

				if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				_ = tmpfile.Close()
				filePath = tmpfile.Name()
			} else {
				filePath = "/non/existent/path/config.yaml"
			}

			loader := NewFileLoader(filePath)
			cfg, err := loader.Load()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestMemoryLoader_Load(t *testing.T) {
	tests := []struct {
		name   string
		config *interfaces.Config
	}{
		{
			name: "complete configuration",
			config: &interfaces.Config{
				ListenPort: 8080,
				TargetURL:  "http://example.com",
				LogLevel:   "info",
				APIKeys: map[string]string{
					"client1": "upstream1",
					"client2": "upstream2",
				},
				Limits: interfaces.Limits{
					RequestsPerSecond:    10,
					Burst:                20,
					ModelTokensPerMinute: 6000,
				},
				TLS: &interfaces.TLSConfig{
					Enabled:  true,
					CertFile: "/etc/ssl/cert.pem",
					KeyFile:  "/etc/ssl/key.pem",
				},
			},
		},
		{
			name: "minimal configuration",
			config: &interfaces.Config{
				ListenPort: 8080,
				TargetURL:  "http://localhost:9999",
				Limits: interfaces.Limits{
					RequestsPerSecond:    1,
					Burst:                1,
					ModelTokensPerMinute: 100,
				},
			},
		},
		{
			name: "empty configuration",
			config: &interfaces.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewMemoryLoader(tt.config)
			cfg, err := loader.Load()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify it's a copy, not the same instance
			if cfg == tt.config {
				t.Error("Expected a copy of the config, not the same instance")
			}

			// Verify values match
			if cfg.ListenPort != tt.config.ListenPort {
				t.Errorf("Expected ListenPort %d, got %d", tt.config.ListenPort, cfg.ListenPort)
			}
			if cfg.TargetURL != tt.config.TargetURL {
				t.Errorf("Expected TargetURL %s, got %s", tt.config.TargetURL, cfg.TargetURL)
			}
			if cfg.LogLevel != tt.config.LogLevel {
				t.Errorf("Expected LogLevel %s, got %s", tt.config.LogLevel, cfg.LogLevel)
			}
			
			// Check API keys
			if tt.config.APIKeys != nil {
				if len(cfg.APIKeys) != len(tt.config.APIKeys) {
					t.Errorf("Expected %d API keys, got %d", len(tt.config.APIKeys), len(cfg.APIKeys))
				}
				for k, v := range tt.config.APIKeys {
					if cfg.APIKeys[k] != v {
						t.Errorf("Expected APIKey[%s] = %s, got %s", k, v, cfg.APIKeys[k])
					}
				}
			}

			// Check limits
			if cfg.Limits.RequestsPerSecond != tt.config.Limits.RequestsPerSecond {
				t.Errorf("Expected RequestsPerSecond %d, got %d", 
					tt.config.Limits.RequestsPerSecond, cfg.Limits.RequestsPerSecond)
			}
			if cfg.Limits.Burst != tt.config.Limits.Burst {
				t.Errorf("Expected Burst %d, got %d", tt.config.Limits.Burst, cfg.Limits.Burst)
			}
			if cfg.Limits.ModelTokensPerMinute != tt.config.Limits.ModelTokensPerMinute {
				t.Errorf("Expected ModelTokensPerMinute %d, got %d", 
					tt.config.Limits.ModelTokensPerMinute, cfg.Limits.ModelTokensPerMinute)
			}

			// Check TLS
			if tt.config.TLS != nil {
				if cfg.TLS == nil {
					t.Fatal("Expected TLS config, got nil")
				}
				if cfg.TLS.Enabled != tt.config.TLS.Enabled {
					t.Errorf("Expected TLS.Enabled %v, got %v", tt.config.TLS.Enabled, cfg.TLS.Enabled)
				}
				if cfg.TLS.CertFile != tt.config.TLS.CertFile {
					t.Errorf("Expected TLS.CertFile %s, got %s", tt.config.TLS.CertFile, cfg.TLS.CertFile)
				}
				if cfg.TLS.KeyFile != tt.config.TLS.KeyFile {
					t.Errorf("Expected TLS.KeyFile %s, got %s", tt.config.TLS.KeyFile, cfg.TLS.KeyFile)
				}
			} else if cfg.TLS != nil {
				t.Error("Expected no TLS config, got one")
			}
		})
	}
}

func TestMemoryLoader_Isolation(t *testing.T) {
	// Test that modifying the original config doesn't affect loaded configs
	original := &interfaces.Config{
		ListenPort: 8080,
		TargetURL:  "http://example.com",
		APIKeys: map[string]string{
			"client1": "upstream1",
		},
		Limits: interfaces.Limits{
			RequestsPerSecond: 10,
		},
	}

	loader := NewMemoryLoader(original)

	// Load first copy
	cfg1, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Modify original
	original.ListenPort = 9090
	original.APIKeys["client2"] = "upstream2"
	original.Limits.RequestsPerSecond = 20

	// Load second copy
	cfg2, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// First copy should still have original values
	if cfg1.ListenPort != 8080 {
		t.Errorf("cfg1 ListenPort changed: expected 8080, got %d", cfg1.ListenPort)
	}
	if len(cfg1.APIKeys) != 1 {
		t.Errorf("cfg1 APIKeys changed: expected 1 key, got %d", len(cfg1.APIKeys))
	}
	if cfg1.Limits.RequestsPerSecond != 10 {
		t.Errorf("cfg1 RequestsPerSecond changed: expected 10, got %d", cfg1.Limits.RequestsPerSecond)
	}

	// Second copy should have modified values
	if cfg2.ListenPort != 9090 {
		t.Errorf("cfg2 ListenPort wrong: expected 9090, got %d", cfg2.ListenPort)
	}
	if len(cfg2.APIKeys) != 2 {
		t.Errorf("cfg2 APIKeys wrong: expected 2 keys, got %d", len(cfg2.APIKeys))
	}
	if cfg2.Limits.RequestsPerSecond != 20 {
		t.Errorf("cfg2 RequestsPerSecond wrong: expected 20, got %d", cfg2.Limits.RequestsPerSecond)
	}
}

func TestFileLoader_ConfigCompatibility(t *testing.T) {
	// Test that FileLoader properly converts from config.Config to interfaces.Config
	content := `
listen_port: 8080
target_url: "http://example.com"
log_level: "info"
api_keys:
  "client1": "upstream1"
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
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpfile.Close()

	// Load with config package directly
	configPkg, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load with config package: %v", err)
	}

	// Load with FileLoader
	loader := NewFileLoader(tmpfile.Name())
	interfacesCfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load with FileLoader: %v", err)
	}

	// Compare values
	if interfacesCfg.ListenPort != configPkg.ListenPort {
		t.Errorf("ListenPort mismatch: %d vs %d", interfacesCfg.ListenPort, configPkg.ListenPort)
	}
	if interfacesCfg.TargetURL != configPkg.TargetURL {
		t.Errorf("TargetURL mismatch: %s vs %s", interfacesCfg.TargetURL, configPkg.TargetURL)
	}
	if interfacesCfg.LogLevel != configPkg.LogLevel {
		t.Errorf("LogLevel mismatch: %s vs %s", interfacesCfg.LogLevel, configPkg.LogLevel)
	}
	if len(interfacesCfg.APIKeys) != len(configPkg.APIKeys) {
		t.Errorf("APIKeys length mismatch: %d vs %d", len(interfacesCfg.APIKeys), len(configPkg.APIKeys))
	}
	if interfacesCfg.Limits.RequestsPerSecond != configPkg.Limits.RequestsPerSecond {
		t.Errorf("RequestsPerSecond mismatch: %d vs %d", 
			interfacesCfg.Limits.RequestsPerSecond, configPkg.Limits.RequestsPerSecond)
	}
	if interfacesCfg.Limits.Burst != configPkg.Limits.Burst {
		t.Errorf("Burst mismatch: %d vs %d", interfacesCfg.Limits.Burst, configPkg.Limits.Burst)
	}
	if interfacesCfg.Limits.ModelTokensPerMinute != configPkg.Limits.ModelTokensPerMinute {
		t.Errorf("ModelTokensPerMinute mismatch: %d vs %d", 
			interfacesCfg.Limits.ModelTokensPerMinute, configPkg.Limits.ModelTokensPerMinute)
	}

	// Check TLS conversion
	if configPkg.TLS != nil {
		if interfacesCfg.TLS == nil {
			t.Fatal("TLS config not converted")
		}
		if interfacesCfg.TLS.Enabled != configPkg.TLS.Enabled {
			t.Errorf("TLS.Enabled mismatch: %v vs %v", interfacesCfg.TLS.Enabled, configPkg.TLS.Enabled)
		}
		if interfacesCfg.TLS.CertFile != configPkg.TLS.CertFile {
			t.Errorf("TLS.CertFile mismatch: %s vs %s", interfacesCfg.TLS.CertFile, configPkg.TLS.CertFile)
		}
		if interfacesCfg.TLS.KeyFile != configPkg.TLS.KeyFile {
			t.Errorf("TLS.KeyFile mismatch: %s vs %s", interfacesCfg.TLS.KeyFile, configPkg.TLS.KeyFile)
		}
	}
}

func BenchmarkFileLoader_Load(b *testing.B) {
	content := `
listen_port: 8080
target_url: "http://example.com"
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
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpfile.Close()

	loader := NewFileLoader(tmpfile.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

func BenchmarkMemoryLoader_Load(b *testing.B) {
	config := &interfaces.Config{
		ListenPort: 8080,
		TargetURL:  "http://example.com",
		LogLevel:   "info",
		APIKeys: map[string]string{
			"client1": "upstream1",
			"client2": "upstream2",
			"client3": "upstream3",
		},
		Limits: interfaces.Limits{
			RequestsPerSecond:    10,
			Burst:                20,
			ModelTokensPerMinute: 6000,
		},
		TLS: &interfaces.TLSConfig{
			Enabled:  true,
			CertFile: "/etc/ssl/cert.pem",
			KeyFile:  "/etc/ssl/key.pem",
		},
	}

	loader := NewMemoryLoader(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}
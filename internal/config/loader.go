package config

import (
	"github.com/jamesprial/nexus/config"
	"github.com/jamesprial/nexus/internal/interfaces"
)

// FileLoader loads configuration from a YAML file
type FileLoader struct {
	filePath string
}

// NewFileLoader creates a new file-based configuration loader
func NewFileLoader(filePath string) *FileLoader {
	return &FileLoader{
		filePath: filePath,
	}
}

// Load implements interfaces.ConfigLoader
func (f *FileLoader) Load() (*interfaces.Config, error) {
	// Use the existing config.Load function
	cfg, err := config.Load(f.filePath)
	if err != nil {
		return nil, err
	}

	// Convert to interface config
	result := &interfaces.Config{
		ListenPort: cfg.ListenPort,
		TargetURL:  cfg.TargetURL,
		LogLevel:   cfg.LogLevel,
		APIKeys:    cfg.APIKeys,
		Limits: interfaces.Limits{
			RequestsPerSecond:    cfg.Limits.RequestsPerSecond,
			Burst:                cfg.Limits.Burst,
			ModelTokensPerMinute: cfg.Limits.ModelTokensPerMinute,
		},
	}
	
	// Convert TLS config if present
	if cfg.TLS != nil {
		result.TLS = &interfaces.TLSConfig{
			Enabled:  cfg.TLS.Enabled,
			CertFile: cfg.TLS.CertFile,
			KeyFile:  cfg.TLS.KeyFile,
		}
	}
	
	return result, nil
}

// MemoryLoader loads configuration from memory (useful for testing)
type MemoryLoader struct {
	config *interfaces.Config
}

// NewMemoryLoader creates a new in-memory configuration loader
func NewMemoryLoader(config *interfaces.Config) *MemoryLoader {
	return &MemoryLoader{
		config: config,
	}
}

// Load implements interfaces.ConfigLoader
func (m *MemoryLoader) Load() (*interfaces.Config, error) {
	// Return a copy to prevent modification
	result := &interfaces.Config{
		ListenPort: m.config.ListenPort,
		TargetURL:  m.config.TargetURL,
		LogLevel:   m.config.LogLevel,
		Limits: interfaces.Limits{
			RequestsPerSecond:    m.config.Limits.RequestsPerSecond,
			Burst:                m.config.Limits.Burst,
			ModelTokensPerMinute: m.config.Limits.ModelTokensPerMinute,
		},
	}
	
	// Deep copy API keys map
	if m.config.APIKeys != nil {
		result.APIKeys = make(map[string]string, len(m.config.APIKeys))
		for k, v := range m.config.APIKeys {
			result.APIKeys[k] = v
		}
	}
	
	// Copy TLS config if present
	if m.config.TLS != nil {
		result.TLS = &interfaces.TLSConfig{
			Enabled:  m.config.TLS.Enabled,
			CertFile: m.config.TLS.CertFile,
			KeyFile:  m.config.TLS.KeyFile,
		}
	}
	
	return result, nil
}
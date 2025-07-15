package config

import (
	"os"

	"github.com/jamesprial/nexus/internal/interfaces"
	"gopkg.in/yaml.v3"
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
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, err
	}

	// Use the existing config struct for YAML unmarshaling
	var yamlConfig struct {
		ListenPort int    `yaml:"listen_port"`
		TargetURL  string `yaml:"target_url"`
		Limits     struct {
			RequestsPerSecond    int `yaml:"requests_per_second"`
			Burst                int `yaml:"burst"`
			ModelTokensPerMinute int `yaml:"model_tokens_per_minute"`
		} `yaml:"limits"`
	}

	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, err
	}

	// Convert to interface config
	return &interfaces.Config{
		ListenPort: yamlConfig.ListenPort,
		TargetURL:  yamlConfig.TargetURL,
		Limits: interfaces.Limits{
			RequestsPerSecond:    yamlConfig.Limits.RequestsPerSecond,
			Burst:                yamlConfig.Limits.Burst,
			ModelTokensPerMinute: yamlConfig.Limits.ModelTokensPerMinute,
		},
	}, nil
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
	return &interfaces.Config{
		ListenPort: m.config.ListenPort,
		TargetURL:  m.config.TargetURL,
		Limits: interfaces.Limits{
			RequestsPerSecond:    m.config.Limits.RequestsPerSecond,
			Burst:                m.config.Limits.Burst,
			ModelTokensPerMinute: m.config.Limits.ModelTokensPerMinute,
		},
	}, nil
}
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenPort int               `yaml:"listen_port"`
	TargetURL  string            `yaml:"target_url"`
	LogLevel   string            `yaml:"log_level"`
	APIKeys    map[string]string `yaml:"api_keys"`
	Limits     Limits            `yaml:"limits"`
	TLS        *TLSConfig        `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type Limits struct {
	RequestsPerSecond    int `yaml:"requests_per_second"`
	Burst                int `yaml:"burst"`
	ModelTokensPerMinute int `yaml:"model_tokens_per_minute"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

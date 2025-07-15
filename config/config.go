package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenPort int    `yaml:"listen_port"`
	TargetURL  string `yaml:"target_url"`
	Limits     Limits `yaml:"limits"`
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

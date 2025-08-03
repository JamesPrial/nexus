package auth

import (
	"errors"
	"strings"

	"github.com/jamesprial/nexus/internal/config"
	"github.com/jamesprial/nexus/internal/interfaces"
)

var (
	ErrInvalidClientKey = errors.New("invalid client API key")
	ErrNoUpstreamKey    = errors.New("no upstream API key found")
)

// FileKeyManager implements interfaces.KeyManager using configuration file
type FileKeyManager struct {
	apiKeys map[string]string
}

// NewFileKeyManager creates a new FileKeyManager from configuration
func NewFileKeyManager(cfg *config.Config) interfaces.KeyManager {
	manager := &FileKeyManager{
		apiKeys: make(map[string]string),
	}

	// Copy API keys from config
	if cfg.APIKeys != nil {
		for clientKey, upstreamKey := range cfg.APIKeys {
			manager.apiKeys[clientKey] = upstreamKey
		}
	}

	return manager
}

// ValidateClientKey checks if a client API key is valid
func (f *FileKeyManager) ValidateClientKey(clientKey string) bool {
	if !f.IsConfigured() {
		// If not configured, accept any non-empty key
		return strings.TrimSpace(clientKey) != ""
	}

	_, exists := f.apiKeys[clientKey]
	return exists
}

// GetUpstreamKey returns the upstream API key for a client key
func (f *FileKeyManager) GetUpstreamKey(clientKey string) (string, error) {
	if !f.IsConfigured() {
		// If not configured, pass through the client key
		return clientKey, nil
	}

	upstreamKey, exists := f.apiKeys[clientKey]
	if !exists {
		return "", ErrInvalidClientKey
	}

	if upstreamKey == "" {
		return "", ErrNoUpstreamKey
	}

	return upstreamKey, nil
}

// IsConfigured returns true if API key management is configured
func (f *FileKeyManager) IsConfigured() bool {
	return len(f.apiKeys) > 0
}

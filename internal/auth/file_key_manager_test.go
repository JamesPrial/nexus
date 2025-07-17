package auth

import (
	"testing"

	"github.com/jamesprial/nexus/config"
)

func TestNewFileKeyManager(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected map[string]string
	}{
		{
			name: "valid configuration with API keys",
			config: &config.Config{
				APIKeys: map[string]string{
					"client1": "upstream1",
					"client2": "upstream2",
				},
			},
			expected: map[string]string{
				"client1": "upstream1",
				"client2": "upstream2",
			},
		},
		{
			name: "empty API keys map",
			config: &config.Config{
				APIKeys: map[string]string{},
			},
			expected: map[string]string{},
		},
		{
			name: "nil API keys map",
			config: &config.Config{
				APIKeys: nil,
			},
			expected: map[string]string{},
		},
		{
			name: "special characters in keys",
			config: &config.Config{
				APIKeys: map[string]string{
					"client-key_123":     "sk-1234567890abcdef",
					"client.with.dots":   "sk-abcdef1234567890",
					"client@company.com": "sk-fedcba0987654321",
				},
			},
			expected: map[string]string{
				"client-key_123":     "sk-1234567890abcdef",
				"client.with.dots":   "sk-abcdef1234567890",
				"client@company.com": "sk-fedcba0987654321",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileKeyManager(tt.config)
			fileManager := manager.(*FileKeyManager)

			if len(fileManager.apiKeys) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(fileManager.apiKeys))
			}

			for expectedKey, expectedValue := range tt.expected {
				if actualValue, exists := fileManager.apiKeys[expectedKey]; !exists {
					t.Errorf("expected key %s not found", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("for key %s, expected value %s, got %s", expectedKey, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestFileKeyManager_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name: "configured with API keys",
			config: &config.Config{
				APIKeys: map[string]string{
					"client1": "upstream1",
				},
			},
			expected: true,
		},
		{
			name: "empty API keys map",
			config: &config.Config{
				APIKeys: map[string]string{},
			},
			expected: false,
		},
		{
			name: "nil API keys map",
			config: &config.Config{
				APIKeys: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileKeyManager(tt.config)
			result := manager.IsConfigured()
			if result != tt.expected {
				t.Errorf("expected IsConfigured() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFileKeyManager_ValidateClientKey(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		clientKey string
		expected  bool
	}{
		{
			name: "valid client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"valid-client": "upstream-key",
				},
			},
			clientKey: "valid-client",
			expected:  true,
		},
		{
			name: "invalid client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"valid-client": "upstream-key",
				},
			},
			clientKey: "invalid-client",
			expected:  false,
		},
		{
			name: "empty client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"valid-client": "upstream-key",
				},
			},
			clientKey: "",
			expected:  false,
		},
		{
			name: "whitespace-only client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"valid-client": "upstream-key",
				},
			},
			clientKey: "   ",
			expected:  false,
		},
		{
			name: "valid key in unconfigured manager",
			config: &config.Config{
				APIKeys: nil,
			},
			clientKey: "any-key",
			expected:  true,
		},
		{
			name: "empty key in unconfigured manager",
			config: &config.Config{
				APIKeys: nil,
			},
			clientKey: "",
			expected:  false,
		},
		{
			name: "whitespace-only key in unconfigured manager",
			config: &config.Config{
				APIKeys: nil,
			},
			clientKey: "   ",
			expected:  false,
		},
		{
			name: "key with special characters",
			config: &config.Config{
				APIKeys: map[string]string{
					"client-key_123": "upstream-key",
				},
			},
			clientKey: "client-key_123",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileKeyManager(tt.config)
			result := manager.ValidateClientKey(tt.clientKey)
			if result != tt.expected {
				t.Errorf("expected ValidateClientKey(%s) to return %v, got %v", tt.clientKey, tt.expected, result)
			}
		})
	}
}

func TestFileKeyManager_GetUpstreamKey(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		clientKey     string
		expectedKey   string
		expectedError error
	}{
		{
			name: "valid client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"client1": "upstream1",
					"client2": "upstream2",
				},
			},
			clientKey:     "client1",
			expectedKey:   "upstream1",
			expectedError: nil,
		},
		{
			name: "invalid client key in configured manager",
			config: &config.Config{
				APIKeys: map[string]string{
					"client1": "upstream1",
				},
			},
			clientKey:     "invalid-client",
			expectedKey:   "",
			expectedError: ErrInvalidClientKey,
		},
		{
			name: "client key with empty upstream key",
			config: &config.Config{
				APIKeys: map[string]string{
					"client1": "",
				},
			},
			clientKey:     "client1",
			expectedKey:   "",
			expectedError: ErrNoUpstreamKey,
		},
		{
			name: "pass-through in unconfigured manager",
			config: &config.Config{
				APIKeys: nil,
			},
			clientKey:     "any-client-key",
			expectedKey:   "any-client-key",
			expectedError: nil,
		},
		{
			name: "empty key in unconfigured manager",
			config: &config.Config{
				APIKeys: map[string]string{},
			},
			clientKey:     "some-key",
			expectedKey:   "some-key",
			expectedError: nil,
		},
		{
			name: "special characters in keys",
			config: &config.Config{
				APIKeys: map[string]string{
					"client-key_123": "sk-1234567890abcdef",
				},
			},
			clientKey:     "client-key_123",
			expectedKey:   "sk-1234567890abcdef",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFileKeyManager(tt.config)
			key, err := manager.GetUpstreamKey(tt.clientKey)

			if key != tt.expectedKey {
				t.Errorf("expected upstream key %s, got %s", tt.expectedKey, key)
			}

			if err != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestFileKeyManager_ConcurrentAccess(t *testing.T) {
	config := &config.Config{
		APIKeys: map[string]string{
			"client1": "upstream1",
			"client2": "upstream2",
			"client3": "upstream3",
		},
	}

	manager := NewFileKeyManager(config)

	// Test concurrent read access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Multiple operations that should be safe for concurrent access
			_ = manager.IsConfigured()
			_ = manager.ValidateClientKey("client1")
			_, _ = manager.GetUpstreamKey("client2")
			_ = manager.ValidateClientKey("client3")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkFileKeyManager_ValidateClientKey(b *testing.B) {
	config := &config.Config{
		APIKeys: map[string]string{
			"client1": "upstream1",
			"client2": "upstream2",
			"client3": "upstream3",
		},
	}

	manager := NewFileKeyManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ValidateClientKey("client1")
	}
}

func BenchmarkFileKeyManager_GetUpstreamKey(b *testing.B) {
	config := &config.Config{
		APIKeys: map[string]string{
			"client1": "upstream1",
			"client2": "upstream2",
			"client3": "upstream3",
		},
	}

	manager := NewFileKeyManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetUpstreamKey("client1")
	}
}
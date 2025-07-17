package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockKeyManager implements interfaces.KeyManager for testing
type mockKeyManager struct {
	apiKeys   map[string]string
	configured bool
}

func (m *mockKeyManager) ValidateClientKey(clientKey string) bool {
	if !m.configured {
		return strings.TrimSpace(clientKey) != ""
	}
	_, exists := m.apiKeys[clientKey]
	return exists
}

func (m *mockKeyManager) GetUpstreamKey(clientKey string) (string, error) {
	if !m.configured {
		return clientKey, nil
	}
	
	upstreamKey, exists := m.apiKeys[clientKey]
	if !exists {
		return "", ErrInvalidClientKey
	}
	
	if upstreamKey == "" {
		return "", ErrNoUpstreamKey
	}
	
	return upstreamKey, nil
}

func (m *mockKeyManager) IsConfigured() bool {
	return m.configured
}

// mockLogger implements interfaces.Logger for testing
type mockLogger struct {
	logs []logEntry
}

type logEntry struct {
	level   string
	message string
	fields  map[string]any
}

func (m *mockLogger) Debug(msg string, fields map[string]any) {
	m.logs = append(m.logs, logEntry{"debug", msg, fields})
}

func (m *mockLogger) Info(msg string, fields map[string]any) {
	m.logs = append(m.logs, logEntry{"info", msg, fields})
}

func (m *mockLogger) Warn(msg string, fields map[string]any) {
	m.logs = append(m.logs, logEntry{"warn", msg, fields})
}

func (m *mockLogger) Error(msg string, fields map[string]any) {
	m.logs = append(m.logs, logEntry{"error", msg, fields})
}

func (m *mockLogger) hasLogLevel(level string) bool {
	for _, log := range m.logs {
		if log.level == level {
			return true
		}
	}
	return false
}

func (m *mockLogger) hasLogMessage(message string) bool {
	for _, log := range m.logs {
		if strings.Contains(log.message, message) {
			return true
		}
	}
	return false
}

func TestNewAuthMiddleware(t *testing.T) {
	keyManager := &mockKeyManager{
		apiKeys:   map[string]string{"client1": "upstream1"},
		configured: true,
	}
	logger := &mockLogger{}

	middleware := NewAuthMiddleware(keyManager, logger)

	if middleware == nil {
		t.Fatal("expected non-nil middleware")
	}

	if middleware.keyManager != keyManager {
		t.Error("expected keyManager to be set correctly")
	}

	if middleware.logger != logger {
		t.Error("expected logger to be set correctly")
	}
}

func TestAuthMiddleware_SuccessfulAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedHeader string
		clientKey      string
		upstreamKey    string
	}{
		{
			name:           "Bearer token transformation",
			authHeader:     "Bearer client-key-123",
			expectedHeader: "Bearer upstream-key-456",
			clientKey:      "client-key-123",
			upstreamKey:    "upstream-key-456",
		},
		{
			name:           "Raw token transformation",
			authHeader:     "client-key-123",
			expectedHeader: "upstream-key-456",
			clientKey:      "client-key-123",
			upstreamKey:    "upstream-key-456",
		},
		{
			name:           "Bearer with whitespace",
			authHeader:     "Bearer   client-key-123   ",
			expectedHeader: "Bearer upstream-key-456",
			clientKey:      "client-key-123",
			upstreamKey:    "upstream-key-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyManager := &mockKeyManager{
				apiKeys: map[string]string{
					tt.clientKey: tt.upstreamKey,
				},
				configured: true,
			}
			logger := &mockLogger{}
			middleware := NewAuthMiddleware(keyManager, logger)

			// Create a test handler that checks the Authorization header
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader != tt.expectedHeader {
					t.Errorf("expected Authorization header %s, got %s", tt.expectedHeader, authHeader)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create test request
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute middleware
			handler := middleware.Middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			// Check response
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}

			// Check that debug log was created
			if !logger.hasLogLevel("debug") {
				t.Error("expected debug log to be created")
			}
		})
	}
}

func TestAuthMiddleware_MissingAPIKey(t *testing.T) {
	keyManager := &mockKeyManager{
		apiKeys:   map[string]string{"valid-key": "upstream-key"},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	tests := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "completely missing header",
			authHeader: "",
		},
		{
			name:       "only whitespace",
			authHeader: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("POST", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Create next handler (should not be called)
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("next handler should not be called for missing API key")
			})

			// Execute middleware
			handler := middleware.Middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			// Check response
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}

			if !strings.Contains(rr.Body.String(), "Missing API key") {
				t.Errorf("expected 'Missing API key' in response body, got %s", rr.Body.String())
			}

			// Check that warning log was created
			if !logger.hasLogLevel("warn") {
				t.Error("expected warning log to be created")
			}
		})
	}
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"valid-key": "upstream-key",
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	tests := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "invalid Bearer token",
			authHeader: "Bearer invalid-key",
		},
		{
			name:       "invalid raw token",
			authHeader: "invalid-key",
		},
		{
			name:       "empty Bearer token",
			authHeader: "Bearer",
		},
		{
			name:       "Bearer with only whitespace",
			authHeader: "Bearer   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Create next handler (should not be called)
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("next handler should not be called for invalid API key")
			})

			// Execute middleware
			handler := middleware.Middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			// Check response
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}

			if !strings.Contains(rr.Body.String(), "Invalid API key") {
				t.Errorf("expected 'Invalid API key' in response body, got %s", rr.Body.String())
			}

			// Check that warning log was created
			if !logger.hasLogLevel("warn") {
				t.Error("expected warning log to be created")
			}
		})
	}
}

func TestAuthMiddleware_GetUpstreamKeyError(t *testing.T) {
	// Create a key manager that validates the key but fails to get upstream key
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"client-key": "", // Empty upstream key to trigger ErrNoUpstreamKey
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	// Create test request
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "client-key")
	
	// Create response recorder
	rr := httptest.NewRecorder()

	// Create next handler (should not be called)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for upstream key error")
	})

	// Execute middleware
	handler := middleware.Middleware(nextHandler)
	handler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Authentication failed") {
		t.Errorf("expected 'Authentication failed' in response body, got %s", rr.Body.String())
	}

	// Check that error log was created
	if !logger.hasLogLevel("error") {
		t.Error("expected error log to be created")
	}
}

func TestAuthMiddleware_UnconfiguredManager(t *testing.T) {
	// Test with unconfigured key manager (pass-through mode)
	keyManager := &mockKeyManager{
		apiKeys:   nil,
		configured: false,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	tests := []struct {
		name           string
		authHeader     string
		expectedHeader string
		expectSuccess  bool
	}{
		{
			name:           "pass-through Bearer token",
			authHeader:     "Bearer sk-test-key",
			expectedHeader: "Bearer sk-test-key",
			expectSuccess:  true,
		},
		{
			name:           "pass-through raw token",
			authHeader:     "sk-test-key",
			expectedHeader: "sk-test-key",
			expectSuccess:  true,
		},
		{
			name:           "empty token still fails",
			authHeader:     "",
			expectedHeader: "",
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("POST", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			// Create response recorder
			rr := httptest.NewRecorder()

			var nextHandlerCalled bool
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
				if tt.expectSuccess {
					authHeader := r.Header.Get("Authorization")
					if authHeader != tt.expectedHeader {
						t.Errorf("expected Authorization header %s, got %s", tt.expectedHeader, authHeader)
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			// Execute middleware
			handler := middleware.Middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			if tt.expectSuccess {
				if rr.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", rr.Code)
				}
				if !nextHandlerCalled {
					t.Error("expected next handler to be called")
				}
			} else {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("expected status 401, got %d", rr.Code)
				}
				if nextHandlerCalled {
					t.Error("expected next handler NOT to be called")
				}
			}
		})
	}
}

func TestAuthMiddleware_Logging(t *testing.T) {
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"valid-key": "upstream-key",
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	// Test successful authentication logging
	t.Run("successful authentication logs", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-key")
		rr := httptest.NewRecorder()

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Middleware(nextHandler)
		handler.ServeHTTP(rr, req)

		if !logger.hasLogMessage("authenticated and transformed") {
			t.Error("expected authentication success log message")
		}
	})

	// Test invalid key logging
	t.Run("invalid key logs", func(t *testing.T) {
		logger.logs = nil // Clear previous logs
		
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-key")
		rr := httptest.NewRecorder()

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Middleware(nextHandler)
		handler.ServeHTTP(rr, req)

		if !logger.hasLogMessage("Invalid client API key") {
			t.Error("expected invalid client key log message")
		}
	})
}

func TestAuthMiddleware_ConcurrentRequests(t *testing.T) {
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"client1": "upstream1",
			"client2": "upstream2",
			"client3": "upstream3",
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(nextHandler)

	// Launch multiple concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(clientNum int) {
			defer func() { done <- true }()
			
			var clientKey string
			switch clientNum % 3 {
			case 0:
				clientKey = "client1"
			case 1:
				clientKey = "client2"
			case 2:
				clientKey = "client3"
			}

			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+clientKey)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkAuthMiddleware_SuccessfulAuth(b *testing.B) {
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"client-key": "upstream-key",
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(nextHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer client-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkAuthMiddleware_InvalidAuth(b *testing.B) {
	keyManager := &mockKeyManager{
		apiKeys: map[string]string{
			"client-key": "upstream-key",
		},
		configured: true,
	}
	logger := &mockLogger{}
	middleware := NewAuthMiddleware(keyManager, logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(nextHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestValidationMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectStatus   int
		expectError    string
		maxBodySize    int64
	}{
		{
			name:         "valid POST request",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			contentType:  "application/json",
			expectStatus: http.StatusOK,
			maxBodySize:  1024 * 1024, // 1MB
		},
		{
			name:         "body too large",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         strings.Repeat("a", 1025), // Just over 1KB
			contentType:  "application/json",
			expectStatus: http.StatusRequestEntityTooLarge,
			expectError:  "Request body too large",
			maxBodySize:  1024, // 1KB limit
		},
		{
			name:         "missing content type",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"gpt-4"}`,
			contentType:  "",
			expectStatus: http.StatusBadRequest,
			expectError:  "Content-Type header is required",
		},
		{
			name:         "invalid content type",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"gpt-4"}`,
			contentType:  "text/plain",
			expectStatus: http.StatusBadRequest,
			expectError:  "Content-Type must be application/json",
		},
		{
			name:         "GET request bypasses validation",
			method:       "GET",
			path:         "/v1/models",
			body:         "",
			contentType:  "",
			expectStatus: http.StatusOK,
		},
		{
			name:         "empty body allowed for POST",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         "",
			contentType:  "application/json",
			expectStatus: http.StatusOK,
			maxBodySize:  1024,
		},
		{
			name:         "invalid JSON body",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"invalid json`,
			contentType:  "application/json",
			expectStatus: http.StatusBadRequest,
			expectError:  "Invalid JSON in request body",
		},
		{
			name:         "missing required fields",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"messages":[]}`, // Missing model field
			contentType:  "application/json",
			expectStatus: http.StatusBadRequest,
			expectError:  "Missing required field: model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the validation middleware
			var validationMiddleware func(http.Handler) http.Handler
			if tt.maxBodySize > 0 {
				validationMiddleware = NewRequestValidationMiddleware(tt.maxBodySize)
			} else {
				validationMiddleware = NewRequestValidationMiddleware(1024 * 1024) // Default 1MB
			}

			// Create a test handler that just returns OK
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Wrap with validation middleware
			wrappedHandler := validationMiddleware(handler)

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Execute request
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, rr.Code)
			}

			// Check error message if expected
			if tt.expectError != "" && !strings.Contains(rr.Body.String(), tt.expectError) {
				t.Errorf("Expected error message to contain %q, got %q", tt.expectError, rr.Body.String())
			}
		})
	}
}

func TestRequestValidationMiddleware_PreservesBody(t *testing.T) {
	validationMiddleware := NewRequestValidationMiddleware(1024 * 1024)

	originalBody := `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body in the handler
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		body := buf.String()
		
		if body != originalBody {
			t.Errorf("Body was modified: expected %q, got %q", originalBody, body)
		}
		
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := validationMiddleware(handler)

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(originalBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRequestValidationMiddleware_HeaderValidation(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		expectStatus int
		expectError  string
	}{
		{
			name: "valid headers",
			headers: map[string]string{
				"Authorization": "Bearer sk-test-key",
				"Content-Type":  "application/json",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "suspicious headers blocked",
			headers: map[string]string{
				"Authorization": "Bearer sk-test-key",
				"Content-Type":  "application/json",
				"X-Forwarded-For": "'; DROP TABLE users; --",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "Invalid header value",
		},
		{
			name: "header too long",
			headers: map[string]string{
				"Authorization": "Bearer " + strings.Repeat("a", 8192), // Very long header
				"Content-Type":  "application/json",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "Header too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationMiddleware := NewRequestValidationMiddleware(1024 * 1024)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := validationMiddleware(handler)

			body := `{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}`
			req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, rr.Code)
			}

			if tt.expectError != "" && !strings.Contains(rr.Body.String(), tt.expectError) {
				t.Errorf("Expected error message to contain %q, got %q", tt.expectError, rr.Body.String())
			}
		})
	}
}

func BenchmarkRequestValidationMiddleware(b *testing.B) {
	validationMiddleware := NewRequestValidationMiddleware(1024 * 1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := validationMiddleware(handler)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
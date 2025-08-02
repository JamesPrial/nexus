package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// MaxHeaderLength is the maximum allowed length for a single header value
	MaxHeaderLength = 8000
	
	// DefaultMaxBodySize is the default maximum request body size (10MB)
	DefaultMaxBodySize = 10 * 1024 * 1024
)

// NewRequestValidationMiddleware creates a middleware that validates incoming requests
func NewRequestValidationMiddleware(maxBodySize int64) func(http.Handler) http.Handler {
	if maxBodySize <= 0 {
		maxBodySize = DefaultMaxBodySize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET, HEAD, OPTIONS requests
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Validate headers
			if err := validateHeaders(r); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Validate Content-Type for POST/PUT/PATCH
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")
				if contentType == "" {
					http.Error(w, "Content-Type header is required", http.StatusBadRequest)
					return
				}
				
				// Check if content type is JSON (may include charset)
				if !strings.HasPrefix(contentType, "application/json") {
					http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
					return
				}
			}

			// Check Content-Length if provided
			if r.ContentLength > maxBodySize {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Read and validate body
			if r.Body != nil && r.Body != http.NoBody {
				// Read body with size limit
				bodyReader := io.LimitReader(r.Body, maxBodySize+1)
				bodyBytes, err := io.ReadAll(bodyReader)
				if err != nil {
					http.Error(w, "Failed to read request body", http.StatusBadRequest)
					return
				}

				// Check if body exceeded limit
				if int64(len(bodyBytes)) > maxBodySize {
					http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
					return
				}

				// Validate JSON if content type is JSON and body is not empty
				if len(bodyBytes) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					var jsonData map[string]interface{}
					if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
						http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
						return
					}

					// Validate required fields for specific endpoints
					if err := validateRequiredFields(r.URL.Path, jsonData); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}

				// Replace body with new reader so it can be read again
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			// Pass to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// validateHeaders checks for suspicious or invalid headers
func validateHeaders(r *http.Request) error {
	for name, values := range r.Header {
		for _, value := range values {
			// Check header length
			if len(value) > MaxHeaderLength {
				return fmt.Errorf("Header too long: %s", name)
			}

			// Check for SQL injection attempts in headers
			lowerValue := strings.ToLower(value)
			suspiciousPatterns := []string{
				"drop table",
				"delete from",
				"insert into",
				"update set",
				"<script",
				"javascript:",
				"onerror=",
			}

			for _, pattern := range suspiciousPatterns {
				if strings.Contains(lowerValue, pattern) {
					return fmt.Errorf("Invalid header value detected")
				}
			}
		}
	}
	return nil
}

// validateRequiredFields checks for required fields based on the endpoint
func validateRequiredFields(path string, data map[string]interface{}) error {
	switch path {
	case "/v1/chat/completions":
		// Check for required fields
		if _, ok := data["model"]; !ok {
			return fmt.Errorf("Missing required field: model")
		}
		if _, ok := data["messages"]; !ok {
			return fmt.Errorf("Missing required field: messages")
		}
	case "/v1/completions":
		if _, ok := data["model"]; !ok {
			return fmt.Errorf("Missing required field: model")
		}
		if _, ok := data["prompt"]; !ok {
			return fmt.Errorf("Missing required field: prompt")
		}
	case "/v1/embeddings":
		if _, ok := data["model"]; !ok {
			return fmt.Errorf("Missing required field: model")
		}
		if _, ok := data["input"]; !ok {
			return fmt.Errorf("Missing required field: input")
		}
	}
	return nil
}
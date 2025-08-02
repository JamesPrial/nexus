package proxy

import (
	"testing"
)

// Test for improved SetTarget validation
func TestHTTPProxy_SetTarget_Validation(t *testing.T) {
	tests := []struct {
		name        string
		targetURL   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty URL should error",
			targetURL:   "",
			expectError: true,
			errorMsg:    "target URL cannot be empty",
		},
		{
			name:        "whitespace URL should error",
			targetURL:   "   ",
			expectError: true,
			errorMsg:    "target URL cannot be empty",
		},
		{
			name:        "URL without scheme should error",
			targetURL:   "example.com",
			expectError: true,
			errorMsg:    "target URL must have a scheme",
		},
		{
			name:        "URL without host should error",
			targetURL:   "http://",
			expectError: true,
			errorMsg:    "target URL must have a host",
		},
		{
			name:        "valid HTTP URL",
			targetURL:   "http://example.com",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with path",
			targetURL:   "https://api.example.com/v1",
			expectError: false,
		},
		{
			name:        "valid URL with port",
			targetURL:   "http://localhost:8080",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &HTTPProxy{}
			err := proxy.SetTarget(tt.targetURL)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if proxy.target == nil {
					t.Error("Expected target to be set")
				}
				if proxy.ReverseProxy == nil {
					t.Error("Expected ReverseProxy to be set")
				}
			}
		})
	}
}
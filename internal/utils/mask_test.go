package utils

import (
	"testing"
)

func TestMaskSensitive(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		prefixLen int
		expected  string
	}{
		{
			name:      "empty string",
			input:     "",
			prefixLen: 10,
			expected:  "",
		},
		{
			name:      "short string",
			input:     "abc",
			prefixLen: 10,
			expected:  "***",
		},
		{
			name:      "exact length",
			input:     "1234567890",
			prefixLen: 10,
			expected:  "12345***",
		},
		{
			name:      "long string",
			input:     "sk-1234567890abcdefghijklmnop",
			prefixLen: 10,
			expected:  "sk-1234567********",
		},
		{
			name:      "bearer token",
			input:     "Bearer sk-1234567890abcdefghijklmnop",
			prefixLen: 10,
			expected:  "Bearer sk-1234567********",
		},
		{
			name:      "bearer with short token",
			input:     "Bearer abc",
			prefixLen: 10,
			expected:  "Bearer ***",
		},
		{
			name:      "zero prefix length",
			input:     "sk-1234567890abcdefghijklmnop",
			prefixLen: 0,
			expected:  "********",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitive(tt.input, tt.prefixLen)
			if result != tt.expected {
				t.Errorf("MaskSensitive(%q, %d) = %q, want %q", tt.input, tt.prefixLen, result, tt.expected)
			}
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty key",
			input:    "",
			expected: "",
		},
		{
			name:     "short key",
			input:    "key123",
			expected: "key***",
		},
		{
			name:     "standard API key",
			input:    "sk-proj-1234567890abcdefghijklmnop",
			expected: "sk-proj-12********",
		},
		{
			name:     "bearer token",
			input:    "Bearer sk-proj-1234567890abcdefghijklmnop",
			expected: "Bearer sk-proj-12********",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
package auth

import (
	"testing"
)

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrInvalidClientKey",
			err:      ErrInvalidClientKey,
			expected: "invalid client API key",
		},
		{
			name:     "ErrNoUpstreamKey",
			err:      ErrNoUpstreamKey,
			expected: "no upstream API key found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("expected non-nil error for %s", tt.name)
				return
			}

			if tt.err.Error() != tt.expected {
				t.Errorf("expected error message %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that errors are distinct
	if ErrInvalidClientKey == ErrNoUpstreamKey {
		t.Error("ErrInvalidClientKey and ErrNoUpstreamKey should be different errors")
	}

	// Test that errors are not nil
	if ErrInvalidClientKey == nil {
		t.Error("ErrInvalidClientKey should not be nil")
	}

	if ErrNoUpstreamKey == nil {
		t.Error("ErrNoUpstreamKey should not be nil")
	}
}
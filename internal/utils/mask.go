package utils

import "strings"

// MaskSensitive masks sensitive data like API keys, showing only a prefix
func MaskSensitive(s string, prefixLen int) string {
	if s == "" {
		return ""
	}
	
	// Handle Bearer prefix
	prefix := ""
	value := s
	if strings.HasPrefix(s, "Bearer ") {
		prefix = "Bearer "
		value = s[7:]
	}
	
	if len(value) <= prefixLen {
		return prefix + value
	}
	
	// Show first prefixLen characters and mask the rest
	masked := value[:prefixLen] + strings.Repeat("*", 8)
	return prefix + masked
}

// MaskAPIKey is a convenience function for masking API keys
func MaskAPIKey(key string) string {
	return MaskSensitive(key, 10)
}
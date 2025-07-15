package logging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/jamesprial/nexus/internal/interfaces"
)

// Level represents logging levels
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// StandardLogger implements interfaces.Logger using Go's standard log package
type StandardLogger struct {
	level Level
}

// NewStandardLogger creates a new standard logger
func NewStandardLogger(level Level) interfaces.Logger {
	return &StandardLogger{
		level: level,
	}
}

// Debug logs debug messages
func (s *StandardLogger) Debug(msg string, fields map[string]any) {
	if s.level <= LevelDebug {
		s.log("DEBUG", msg, fields)
	}
}

// Info logs info messages
func (s *StandardLogger) Info(msg string, fields map[string]any) {
	if s.level <= LevelInfo {
		s.log("INFO", msg, fields)
	}
}

// Warn logs warning messages
func (s *StandardLogger) Warn(msg string, fields map[string]any) {
	if s.level <= LevelWarn {
		s.log("WARN", msg, fields)
	}
}

// Error logs error messages
func (s *StandardLogger) Error(msg string, fields map[string]any) {
	if s.level <= LevelError {
		s.log("ERROR", msg, fields)
	}
}

// log is the internal logging method
func (s *StandardLogger) log(level, msg string, fields map[string]any) {
	timestamp := time.Now().Format(time.RFC3339)
	
	entry := map[string]any{
		"timestamp": timestamp,
		"level":     level,
		"message":   msg,
	}
	
	// Add fields to the log entry
	for k, v := range fields {
		entry[k] = v
	}
	
	// Convert to JSON for structured logging
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}
	
	log.Println(string(jsonBytes))
}

// NoOpLogger implements interfaces.Logger but does nothing (useful for testing)
type NoOpLogger struct{}

// NewNoOpLogger creates a logger that discards all messages
func NewNoOpLogger() interfaces.Logger {
	return &NoOpLogger{}
}

// Debug does nothing
func (n *NoOpLogger) Debug(msg string, fields map[string]any) {}

// Info does nothing
func (n *NoOpLogger) Info(msg string, fields map[string]any) {}

// Warn does nothing
func (n *NoOpLogger) Warn(msg string, fields map[string]any) {}

// Error does nothing
func (n *NoOpLogger) Error(msg string, fields map[string]any) {}
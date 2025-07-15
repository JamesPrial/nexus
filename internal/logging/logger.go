package logging

import (
	"log/slog"
	"os"

	"github.com/jamesprial/nexus/internal/interfaces"
)

// SlogLogger implements interfaces.Logger using Go's standard slog package.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new logger with the specified level.
func NewSlogLogger(levelStr string) interfaces.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// Debug logs debug messages.
func (s *SlogLogger) Debug(msg string, fields map[string]any) {
	s.logger.Debug(msg, fieldsToArgs(fields)...)
}

// Info logs info messages.
func (s *SlogLogger) Info(msg string, fields map[string]any) {
	s.logger.Info(msg, fieldsToArgs(fields)...)
}

// Warn logs warning messages.
func (s *SlogLogger) Warn(msg string, fields map[string]any) {
	s.logger.Warn(msg, fieldsToArgs(fields)...)
}

// Error logs error messages.
func (s *SlogLogger) Error(msg string, fields map[string]any) {
	s.logger.Error(msg, fieldsToArgs(fields)...)
}

// fieldsToArgs converts a map of fields to a slice of slog.Attr.
func fieldsToArgs(fields map[string]any) []any {
	args := make([]any, 0, len(fields))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	return args
}

// NoOpLogger implements interfaces.Logger but does nothing (useful for testing).
type NoOpLogger struct{}

// NewNoOpLogger creates a logger that discards all messages.
func NewNoOpLogger() interfaces.Logger {
	return &NoOpLogger{}
}

// Debug does nothing.
func (n *NoOpLogger) Debug(msg string, fields map[string]any) {}

// Info does nothing.
func (n *NoOpLogger) Info(msg string, fields map[string]any) {}

// Warn does nothing.
func (n *NoOpLogger) Warn(msg string, fields map[string]any) {}

// Error does nothing.
func (n *NoOpLogger) Error(msg string, fields map[string]any) {}

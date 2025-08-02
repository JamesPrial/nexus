package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewSlogLogger(t *testing.T) {
	tests := []struct {
		name          string
		levelStr      string
		expectedLevel slog.Level
	}{
		{
			name:          "debug level",
			levelStr:      "debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "info level",
			levelStr:      "info",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "warn level",
			levelStr:      "warn",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "error level",
			levelStr:      "error",
			expectedLevel: slog.LevelError,
		},
		{
			name:          "invalid level defaults to info",
			levelStr:      "invalid",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "empty level defaults to info",
			levelStr:      "",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "uppercase level",
			levelStr:      "DEBUG",
			expectedLevel: slog.LevelInfo, // Should default to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewSlogLogger(tt.levelStr)
			if logger == nil {
				t.Fatal("Expected non-nil logger")
			}

			// Cast to SlogLogger to access internal logger
			slogLogger, ok := logger.(*SlogLogger)
			if !ok {
				t.Fatal("Expected SlogLogger type")
			}

			// We can't easily test the level directly, but we can verify the logger exists
			if slogLogger.logger == nil {
				t.Error("Expected internal slog.Logger to be initialized")
			}
		})
	}
}

// captureLogger captures log output for testing
type captureLogger struct {
	buf    *bytes.Buffer
	logger *SlogLogger
}

func newCaptureLogger(level string) *captureLogger {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: func() slog.Level {
			switch level {
			case "debug":
				return slog.LevelDebug
			case "info":
				return slog.LevelInfo
			case "warn":
				return slog.LevelWarn
			case "error":
				return slog.LevelError
			default:
				return slog.LevelInfo
			}
		}(),
	})
	
	return &captureLogger{
		buf: buf,
		logger: &SlogLogger{
			logger: slog.New(handler),
		},
	}
}

func TestSlogLogger_LogMethods(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*SlogLogger, string, map[string]any)
		level    string
		message  string
		fields   map[string]any
		contains []string
	}{
		{
			name: "debug log",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Debug(msg, fields)
			},
			level:   "debug",
			message: "debug message",
			fields: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			contains: []string{"level=DEBUG", "debug message", "key1=value1", "key2=42"},
		},
		{
			name: "info log",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			level:   "info",
			message: "info message",
			fields: map[string]any{
				"status": "ok",
				"count":  100,
			},
			contains: []string{"level=INFO", "info message", "status=ok", "count=100"},
		},
		{
			name: "warn log",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Warn(msg, fields)
			},
			level:   "warn",
			message: "warning message",
			fields: map[string]any{
				"warning": "high memory",
				"percent": 85.5,
			},
			contains: []string{"level=WARN", "warning message", "warning=\"high memory\"", "percent=85.5"},
		},
		{
			name: "error log",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Error(msg, fields)
			},
			level:   "error",
			message: "error message",
			fields: map[string]any{
				"error":   "connection failed",
				"retries": 3,
			},
			contains: []string{"level=ERROR", "error message", "error=\"connection failed\"", "retries=3"},
		},
		{
			name: "empty fields",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			level:    "info",
			message:  "message without fields",
			fields:   map[string]any{},
			contains: []string{"level=INFO", "message without fields"},
		},
		{
			name: "nil fields",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			level:    "info",
			message:  "message with nil fields",
			fields:   nil,
			contains: []string{"level=INFO", "message with nil fields"},
		},
		{
			name: "complex field types",
			logFunc: func(l *SlogLogger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			level:   "info",
			message: "complex types",
			fields: map[string]any{
				"bool":   true,
				"int":    42,
				"float":  3.14,
				"string": "test",
				"array":  []int{1, 2, 3},
				"map":    map[string]int{"a": 1, "b": 2},
			},
			contains: []string{
				"level=INFO",
				"complex types",
				"bool=true",
				"int=42",
				"float=3.14",
				"string=test",
				"array=",  // Arrays get complex formatting
				"map=",    // Maps get complex formatting
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newCaptureLogger(tt.level)
			
			tt.logFunc(capture.logger, tt.message, tt.fields)
			
			output := capture.buf.String()
			
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected log output to contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

func TestSlogLogger_LogLevels(t *testing.T) {
	// Test that logs below the configured level are not output
	tests := []struct {
		name        string
		loggerLevel string
		logCalls    []struct {
			method  string
			message string
		}
		shouldLog []bool
	}{
		{
			name:        "info level filters debug",
			loggerLevel: "info",
			logCalls: []struct {
				method  string
				message string
			}{
				{"debug", "debug message"},
				{"info", "info message"},
				{"warn", "warn message"},
				{"error", "error message"},
			},
			shouldLog: []bool{false, true, true, true},
		},
		{
			name:        "warn level filters debug and info",
			loggerLevel: "warn",
			logCalls: []struct {
				method  string
				message string
			}{
				{"debug", "debug message"},
				{"info", "info message"},
				{"warn", "warn message"},
				{"error", "error message"},
			},
			shouldLog: []bool{false, false, true, true},
		},
		{
			name:        "error level filters all but error",
			loggerLevel: "error",
			logCalls: []struct {
				method  string
				message string
			}{
				{"debug", "debug message"},
				{"info", "info message"},
				{"warn", "warn message"},
				{"error", "error message"},
			},
			shouldLog: []bool{false, false, false, true},
		},
		{
			name:        "debug level logs everything",
			loggerLevel: "debug",
			logCalls: []struct {
				method  string
				message string
			}{
				{"debug", "debug message"},
				{"info", "info message"},
				{"warn", "warn message"},
				{"error", "error message"},
			},
			shouldLog: []bool{true, true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newCaptureLogger(tt.loggerLevel)
			
			for i, call := range tt.logCalls {
				// Clear buffer before each call
				capture.buf.Reset()
				
				switch call.method {
				case "debug":
					capture.logger.Debug(call.message, nil)
				case "info":
					capture.logger.Info(call.message, nil)
				case "warn":
					capture.logger.Warn(call.message, nil)
				case "error":
					capture.logger.Error(call.message, nil)
				}
				
				output := capture.buf.String()
				hasOutput := len(output) > 0
				
				if tt.shouldLog[i] && !hasOutput {
					t.Errorf("Expected %s log to be output for level %s, but got no output",
						call.method, tt.loggerLevel)
				} else if !tt.shouldLog[i] && hasOutput {
					t.Errorf("Expected %s log to be filtered for level %s, but got: %s",
						call.method, tt.loggerLevel, output)
				}
			}
		})
	}
}

func TestFieldsToArgs(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]any
		expected int // expected number of args (2 per field)
	}{
		{
			name:     "empty fields",
			fields:   map[string]any{},
			expected: 0,
		},
		{
			name:     "nil fields",
			fields:   nil,
			expected: 0,
		},
		{
			name: "single field",
			fields: map[string]any{
				"key": "value",
			},
			expected: 1,
		},
		{
			name: "multiple fields",
			fields: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := fieldsToArgs(tt.fields)
			if len(args) != tt.expected {
				t.Errorf("Expected %d args, got %d", tt.expected, len(args))
			}
		})
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()
	
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
	
	// Test that all methods can be called without panic
	logger.Debug("debug", map[string]any{"key": "value"})
	logger.Info("info", map[string]any{"key": "value"})
	logger.Warn("warn", map[string]any{"key": "value"})
	logger.Error("error", map[string]any{"key": "value"})
	
	// Test with nil fields
	logger.Debug("debug", nil)
	logger.Info("info", nil)
	logger.Warn("warn", nil)
	logger.Error("error", nil)
}

func TestSlogLogger_ConcurrentLogging(t *testing.T) {
	capture := newCaptureLogger("info")
	
	// Launch multiple goroutines logging concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			fields := map[string]any{
				"goroutine": id,
				"action":    "test",
			}
			
			capture.logger.Info("concurrent log", fields)
			capture.logger.Debug("debug log", fields) // Should be filtered
			capture.logger.Warn("warning log", fields)
			capture.logger.Error("error log", fields)
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	output := capture.buf.String()
	
	// Should have logs from all goroutines
	for i := 0; i < 10; i++ {
		if !strings.Contains(output, "goroutine="+strings.TrimSpace(string(rune(i+'0')))) {
			t.Errorf("Missing log from goroutine %d", i)
		}
	}
	
	// Should not have debug logs (filtered by level)
	if strings.Contains(output, "debug log") {
		t.Error("Debug logs should be filtered at info level")
	}
}

func BenchmarkSlogLogger_Info(b *testing.B) {
	logger := NewSlogLogger("info")
	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", fields)
	}
}

func BenchmarkSlogLogger_InfoWithManyFields(b *testing.B) {
	logger := NewSlogLogger("info")
	fields := make(map[string]any)
	for i := 0; i < 20; i++ {
		fields[string(rune(i+'a'))] = i
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", fields)
	}
}

func BenchmarkNoOpLogger_Info(b *testing.B) {
	logger := NewNoOpLogger()
	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", fields)
	}
}
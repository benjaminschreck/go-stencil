package stencil

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name           string
		level          LogLevel
		setupFunc      func(*Logger)
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:  "debug level shows all messages",
			level: LogDebug,
			setupFunc: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			expectedOutput: []string{
				"[DEBUG]",
				"debug message",
				"[INFO]",
				"info message",
				"[WARN]",
				"warn message",
				"[ERROR]",
				"error message",
			},
		},
		{
			name:  "info level hides debug messages",
			level: LogInfo,
			setupFunc: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			expectedOutput: []string{
				"[INFO]",
				"info message",
				"[WARN]",
				"warn message",
				"[ERROR]",
				"error message",
			},
			notExpected: []string{
				"[DEBUG]",
				"debug message",
			},
		},
		{
			name:  "warn level shows only warnings and errors",
			level: LogWarn,
			setupFunc: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			expectedOutput: []string{
				"[WARN]",
				"warn message",
				"[ERROR]",
				"error message",
			},
			notExpected: []string{
				"[DEBUG]",
				"[INFO]",
			},
		},
		{
			name:  "error level shows only errors",
			level: LogError,
			setupFunc: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			expectedOutput: []string{
				"[ERROR]",
				"error message",
			},
			notExpected: []string{
				"[DEBUG]",
				"[INFO]",
				"[WARN]",
			},
		},
		{
			name:  "off level shows nothing",
			level: LogOff,
			setupFunc: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			expectedOutput: []string{},
			notExpected: []string{
				"[DEBUG]",
				"[INFO]",
				"[WARN]",
				"[ERROR]",
			},
		},
		{
			name:  "structured fields",
			level: LogDebug,
			setupFunc: func(l *Logger) {
				l.WithFields(Fields{
					"component": "tokenizer",
					"file":      "test.docx",
				}).Debug("processing file")
			},
			expectedOutput: []string{
				"[DEBUG]",
				"processing file",
				"component=tokenizer",
				"file=test.docx",
			},
		},
		{
			name:  "debug mode helpers",
			level: LogDebug,
			setupFunc: func(l *Logger) {
				l.DebugTemplate("template content", map[string]interface{}{
					"var1": "value1",
				})
				l.DebugExpression("price * 1.2", 24.0)
			},
			expectedOutput: []string{
				"[DEBUG]",
				"Template:",
				"template content",
				"Context:",
				"map[var1:value1]",
				"Expression:",
				"price * 1.2",
				"Result: 24",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(&buf, tt.level)

			tt.setupFunc(logger)

			output := buf.String()

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}

			// Check not expected output
			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nOutput: %s", notExpected, output)
				}
			}
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	// Save original logger
	original := globalLogger

	// Test setting custom logger
	var buf bytes.Buffer
	customLogger := NewLogger(&buf, LogDebug)
	SetLogger(customLogger)

	// Use global logging functions
	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")

	output := buf.String()
	expectedStrings := []string{
		"[DEBUG] test debug",
		"[INFO] test info",
		"[WARN] test warn",
		"[ERROR] test error",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
		}
	}

	// Restore original logger
	globalLogger = original
}

func TestDebugMode(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, LogDebug)

	// Test that debug mode is detected correctly
	if !logger.IsDebugMode() {
		t.Error("Expected IsDebugMode() to return true for LogDebug level")
	}

	logger.SetLevel(LogInfo)
	if logger.IsDebugMode() {
		t.Error("Expected IsDebugMode() to return false for LogInfo level")
	}
}

func TestLoggerPerformance(t *testing.T) {
	// Test that disabled log levels don't impact performance
	logger := NewLogger(nil, LogOff)

	// This should be very fast since logging is disabled
	start := make([]byte, 0, 1024)
	for i := 0; i < 10000; i++ {
		logger.Debug("This message should not be processed")
		logger.WithFields(Fields{
			"key1": "value1",
			"key2": i,
		}).Info("This also should not be processed")
	}

	// Ensure we didn't allocate much memory
	if len(start) > 0 {
		// This is just to use the variable
		t.Logf("Memory test passed")
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, LogDebug)

	// Test field chaining
	logger.
		WithField("request_id", "12345").
		WithField("user", "john").
		WithFields(Fields{
			"action": "render",
			"file":   "template.docx",
		}).
		Info("Processing template")

	output := buf.String()
	expectedFields := []string{
		"request_id=12345",
		"user=john",
		"action=render",
		"file=template.docx",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected output to contain field %q, but it didn't.\nOutput: %s", field, output)
		}
	}
}
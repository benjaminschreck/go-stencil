package stencil

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogOff
)

func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarn:
		return "WARN"
	case LogError:
		return "ERROR"
	case LogOff:
		return "OFF"
	default:
		return "UNKNOWN"
	}
}

type Fields map[string]interface{}

type Logger struct {
	writer io.Writer
	level  LogLevel
	fields Fields
	mu     sync.Mutex
}

var (
	globalLogger     *Logger
	globalLoggerOnce sync.Once
)

func initGlobalLogger() {
	// Initialize global logger with default settings
	globalLoggerOnce.Do(func() {
		config := GetGlobalConfig()
		level := parseLogLevel(config.LogLevel)
		globalLogger = NewLogger(os.Stderr, level)
	})
}

func init() {
	// Defer logger initialization to avoid circular dependency
	initGlobalLogger()
}

func parseLogLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug":
		return LogDebug
	case "info":
		return LogInfo
	case "warn":
		return LogWarn
	case "error":
		return LogError
	case "off":
		return LogOff
	default:
		return LogInfo // Default to info
	}
}

func NewLogger(w io.Writer, level LogLevel) *Logger {
	if w == nil {
		w = io.Discard
	}
	return &Logger{
		writer: w,
		level:  level,
		fields: make(Fields),
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) IsDebugMode() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level == LogDebug
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		writer: l.writer,
		level:  l.level,
		fields: make(Fields, len(l.fields)+1),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

func (l *Logger) WithFields(fields Fields) *Logger {
	newLogger := &Logger{
		writer: l.writer,
		level:  l.level,
		fields: make(Fields, len(l.fields)+len(fields)),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Format message
	message := fmt.Sprintf(format, args...)

	// Build log line
	logLine := fmt.Sprintf("%s [%s] %s", timestamp, level.String(), message)

	// Add fields if any
	if len(l.fields) > 0 {
		logLine += " "
		first := true
		for k, v := range l.fields {
			if !first {
				logLine += " "
			}
			logLine += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
	}

	// Write to output
	fmt.Fprintln(l.writer, logLine)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogError, format, args...)
}

// Debug helpers for template development
func (l *Logger) DebugTemplate(template string, context interface{}) {
	if !l.IsDebugMode() {
		return
	}
	l.Debug("Template: %s", template)
	l.Debug("Context: %+v", context)
}

func (l *Logger) DebugExpression(expr string, result interface{}) {
	if !l.IsDebugMode() {
		return
	}
	l.Debug("Expression: %s", expr)
	l.Debug("Result: %v", result)
}

// Global logging functions
func SetLogger(logger *Logger) {
	globalLogger = logger
}

func GetLogger() *Logger {
	initGlobalLogger()
	return globalLogger
}

func Debug(format string, args ...interface{}) {
	initGlobalLogger()
	globalLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	initGlobalLogger()
	globalLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	initGlobalLogger()
	globalLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	initGlobalLogger()
	globalLogger.Error(format, args...)
}

func WithField(key string, value interface{}) *Logger {
	initGlobalLogger()
	return globalLogger.WithField(key, value)
}

func WithFields(fields Fields) *Logger {
	initGlobalLogger()
	return globalLogger.WithFields(fields)
}

// UpdateLoggerFromConfig updates the global logger based on the current global configuration
func UpdateLoggerFromConfig() {
	config := GetGlobalConfig()
	level := parseLogLevel(config.LogLevel)
	globalLogger.SetLevel(level)
}
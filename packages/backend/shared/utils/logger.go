package utils

import (
	"log/slog"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// NewLogger creates a new structured logger with the specified level
func NewLogger(level LogLevel) *slog.Logger {
	var slogLevel slog.Level

	switch strings.ToLower(string(level)) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	// Use TextHandler for better readability instead of JSON
	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}

// SetupLogging configures the global slog handler based on log level
// This should be called once at application startup to configure logging for the entire application
func SetupLogging(logLevel string) {
	var level slog.Level

	switch strings.ToLower(logLevel) {
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

	// Create a text handler with the specified log level
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	// Set the default logger for the entire application
	slog.SetDefault(slog.New(handler))
}

// GetLogLevelFromEnv gets the log level from environment variable
func GetLogLevelFromEnv(defaultLevel LogLevel) LogLevel {
	envLevel := os.Getenv("LOG_LEVEL")
	if envLevel == "" {
		return defaultLevel
	}

	switch strings.ToLower(envLevel) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return defaultLevel
	}
}

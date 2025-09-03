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

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
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

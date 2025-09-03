package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port                            string
	DataPath                        string
	LogLevel                        string
	Environment                     string
	IdempotencyCacheTTL             string
	IdempotencyCacheCleanupInterval string
	EnableJSONPersistence           string
	InventoryWorkerCount            string
	InventoryQueueBufferSize        string
	MaxEventsInQueue                string
	EventsFilePath                  string
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() *Config {
	// Load .env file if it exists
	// This will not override existing environment variables
	err := godotenv.Load()
	if err != nil {
		slog.Warn("Could not load .env file, continuing with system environment variables only", "error", err)
	} else {
		slog.Info("Successfully loaded .env file")
	}

	config := &Config{
		Port:                            getEnvWithDefault("PORT", "8080"),
		DataPath:                        getEnvWithDefault("DATA_PATH", "data/inventory_test_data.json"),
		LogLevel:                        getEnvWithDefault("LOG_LEVEL", "info"),
		Environment:                     getEnvWithDefault("ENVIRONMENT", "development"),
		IdempotencyCacheTTL:             getEnvWithDefault("IDEMPOTENCY_CACHE_TTL", "2m"),
		IdempotencyCacheCleanupInterval: getEnvWithDefault("IDEMPOTENCY_CACHE_CLEANUP_INTERVAL", "30s"),
		EnableJSONPersistence:           getEnvWithDefault("ENABLE_JSON_PERSISTENCE", "true"),
		InventoryWorkerCount:            getEnvWithDefault("INVENTORY_WORKER_COUNT", "1"),
		InventoryQueueBufferSize:        getEnvWithDefault("INVENTORY_QUEUE_BUFFER_SIZE", "100"),
		MaxEventsInQueue:                getEnvWithDefault("MAX_EVENTS_IN_QUEUE", "10000"),
		EventsFilePath:                  getEnvWithDefault("EVENTS_FILE_PATH", "./data/events.json"),
	}

	// Configure slog based on log level
	setupLogging(config.LogLevel)

	slog.Info("Configuration loaded",
		"port", config.Port,
		"environment", config.Environment,
		"logLevel", config.LogLevel,
		"dataPath", config.DataPath,
		"idempotencyCacheTTL", config.IdempotencyCacheTTL,
		"idempotencyCacheCleanupInterval", config.IdempotencyCacheCleanupInterval,
		"enableJSONPersistence", config.EnableJSONPersistence,
		"inventoryWorkerCount", config.InventoryWorkerCount,
		"inventoryQueueBufferSize", config.InventoryQueueBufferSize,
		"maxEventsInQueue", config.MaxEventsInQueue,
		"eventsFilePath", config.EventsFilePath)

	return config
}

// setupLogging configures the slog handler based on log level
func setupLogging(logLevel string) {
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

// getEnvWithDefault gets an environment variable with a default fallback
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/melibackend/shared/utils"
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

	// Configure slog based on log level using shared utils
	utils.SetupLogging(config.LogLevel)

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

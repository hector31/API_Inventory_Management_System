package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Port          int    `json:"port"`
	Environment   string `json:"environment"`
	LogLevel      string `json:"logLevel"`
	APIKeys       string `json:"apiKeys"`
	CentralAPIURL string `json:"centralApiUrl"`
	CentralAPIKey string `json:"centralApiKey"`
	DataDir       string `json:"dataDir"`
	SyncInterval  int    `json:"syncIntervalMinutes"`
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	cfg := &Config{
		Port:          getEnvAsInt("PORT", 8083),
		Environment:   getEnv("ENVIRONMENT", "development"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		APIKeys:       getEnv("API_KEYS", "store-s1-key,demo"),
		CentralAPIURL: getEnv("CENTRAL_API_URL", "http://inventory-management-system:8081"),
		CentralAPIKey: getEnv("CENTRAL_API_KEY", "demo"),
		DataDir:       getEnv("DATA_DIR", "/app/data"),
		SyncInterval:  getEnvAsInt("SYNC_INTERVAL_MINUTES", 5),
	}

	return cfg
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

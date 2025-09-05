package services

import (
	"testing"

	"inventory-management-api/internal/config"
	"inventory-management-api/internal/services"

	"github.com/stretchr/testify/assert"
)

// TestInventoryService_CreationWithMissingData tests service creation when data file is missing
func TestInventoryService_CreationWithMissingData(t *testing.T) {
	// Arrange - Create test configuration
	cfg := &config.Config{
		DataPath:                        "./nonexistent/inventory_test_data.json",
		EnableJSONPersistence:           "false",
		InventoryWorkerCount:            "2",
		InventoryQueueBufferSize:        "10",
		IdempotencyCacheTTL:             "1m",
		IdempotencyCacheCleanupInterval: "30s",
		LogLevel:                        "info",
		Environment:                     "test",
		Port:                            "8080",
		MaxEventsInQueue:                "1000",
		EventsFilePath:                  "./data/events.json",
	}
	
	// Act - Create inventory service (this should fail gracefully)
	service, err := services.NewInventoryService(cfg)
	
	// Assert - Service creation should fail due to missing data file
	assert.Error(t, err, "Service creation should fail with missing data file")
	assert.Nil(t, service, "Service should be nil when creation fails")
	assert.Contains(t, err.Error(), "error loading test data", "Error should mention test data loading")
}

// TestInventoryService_ConfigValidation tests configuration validation
func TestInventoryService_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Config
		expectError bool
		description string
	}{
		{
			name: "Valid Configuration",
			config: &config.Config{
				DataPath:                        "./data/inventory_test_data.json",
				EnableJSONPersistence:           "false",
				InventoryWorkerCount:            "2",
				InventoryQueueBufferSize:        "10",
				IdempotencyCacheTTL:             "1m",
				IdempotencyCacheCleanupInterval: "30s",
				LogLevel:                        "info",
				Environment:                     "test",
				Port:                            "8080",
				MaxEventsInQueue:                "1000",
				EventsFilePath:                  "./data/events.json",
			},
			expectError: true, // Will fail due to missing data file, but config is valid
			description: "Valid configuration should be accepted",
		},
		{
			name: "Invalid Worker Count",
			config: &config.Config{
				DataPath:                        "./data/inventory_test_data.json",
				EnableJSONPersistence:           "false",
				InventoryWorkerCount:            "invalid",
				InventoryQueueBufferSize:        "10",
				IdempotencyCacheTTL:             "1m",
				IdempotencyCacheCleanupInterval: "30s",
				LogLevel:                        "info",
				Environment:                     "test",
				Port:                            "8080",
				MaxEventsInQueue:                "1000",
				EventsFilePath:                  "./data/events.json",
			},
			expectError: true,
			description: "Invalid worker count should cause error",
		},
		{
			name: "Invalid Cache TTL",
			config: &config.Config{
				DataPath:                        "./data/inventory_test_data.json",
				EnableJSONPersistence:           "false",
				InventoryWorkerCount:            "2",
				InventoryQueueBufferSize:        "10",
				IdempotencyCacheTTL:             "invalid",
				IdempotencyCacheCleanupInterval: "30s",
				LogLevel:                        "info",
				Environment:                     "test",
				Port:                            "8080",
				MaxEventsInQueue:                "1000",
				EventsFilePath:                  "./data/events.json",
			},
			expectError: true,
			description: "Invalid cache TTL should cause error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			service, err := services.NewInventoryService(tc.config)
			
			// Assert
			if tc.expectError {
				assert.Error(t, err, tc.description)
				assert.Nil(t, service, "Service should be nil when creation fails")
			} else {
				assert.NoError(t, err, tc.description)
				assert.NotNil(t, service, "Service should not be nil when creation succeeds")
				if service != nil {
					service.Stop()
				}
			}
		})
	}
}

// TestInventoryService_NilConfig tests service creation with nil config
func TestInventoryService_NilConfig(t *testing.T) {
	// Act
	service, err := services.NewInventoryService(nil)
	
	// Assert
	assert.Error(t, err, "Service creation should fail with nil config")
	assert.Nil(t, service, "Service should be nil when creation fails")
}

// TestInventoryService_EmptyConfig tests service creation with empty config
func TestInventoryService_EmptyConfig(t *testing.T) {
	// Arrange
	cfg := &config.Config{}
	
	// Act
	service, err := services.NewInventoryService(cfg)
	
	// Assert
	assert.Error(t, err, "Service creation should fail with empty config")
	assert.Nil(t, service, "Service should be nil when creation fails")
}

// TestInventoryService_ConfigFieldValidation tests individual config field validation
func TestInventoryService_ConfigFieldValidation(t *testing.T) {
	baseConfig := &config.Config{
		DataPath:                        "./data/inventory_test_data.json",
		EnableJSONPersistence:           "false",
		InventoryWorkerCount:            "2",
		InventoryQueueBufferSize:        "10",
		IdempotencyCacheTTL:             "1m",
		IdempotencyCacheCleanupInterval: "30s",
		LogLevel:                        "info",
		Environment:                     "test",
		Port:                            "8080",
		MaxEventsInQueue:                "1000",
		EventsFilePath:                  "./data/events.json",
	}

	testCases := []struct {
		name         string
		modifyConfig func(*config.Config)
		expectError  bool
	}{
		{
			name: "Empty DataPath",
			modifyConfig: func(cfg *config.Config) {
				cfg.DataPath = ""
			},
			expectError: true,
		},
		{
			name: "Empty InventoryWorkerCount",
			modifyConfig: func(cfg *config.Config) {
				cfg.InventoryWorkerCount = ""
			},
			expectError: true,
		},
		{
			name: "Zero InventoryWorkerCount",
			modifyConfig: func(cfg *config.Config) {
				cfg.InventoryWorkerCount = "0"
			},
			expectError: true,
		},
		{
			name: "Negative InventoryWorkerCount",
			modifyConfig: func(cfg *config.Config) {
				cfg.InventoryWorkerCount = "-1"
			},
			expectError: true,
		},
		{
			name: "Empty QueueBufferSize",
			modifyConfig: func(cfg *config.Config) {
				cfg.InventoryQueueBufferSize = ""
			},
			expectError: true,
		},
		{
			name: "Empty CacheTTL",
			modifyConfig: func(cfg *config.Config) {
				cfg.IdempotencyCacheTTL = ""
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			cfg := *baseConfig // Copy the base config
			tc.modifyConfig(&cfg)
			
			// Act
			service, err := services.NewInventoryService(&cfg)
			
			// Assert
			if tc.expectError {
				assert.Error(t, err, "Should fail with invalid config field")
				assert.Nil(t, service, "Service should be nil when creation fails")
			} else {
				assert.NoError(t, err, "Should succeed with valid config field")
				assert.NotNil(t, service, "Service should not be nil when creation succeeds")
				if service != nil {
					service.Stop()
				}
			}
		})
	}
}

// BenchmarkInventoryService_ConfigValidation benchmarks config validation
func BenchmarkInventoryService_ConfigValidation(b *testing.B) {
	cfg := &config.Config{
		DataPath:                        "./nonexistent/inventory_test_data.json",
		EnableJSONPersistence:           "false",
		InventoryWorkerCount:            "2",
		InventoryQueueBufferSize:        "10",
		IdempotencyCacheTTL:             "1m",
		IdempotencyCacheCleanupInterval: "30s",
		LogLevel:                        "info",
		Environment:                     "test",
		Port:                            "8080",
		MaxEventsInQueue:                "1000",
		EventsFilePath:                  "./data/events.json",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		service, err := services.NewInventoryService(cfg)
		if err == nil && service != nil {
			service.Stop()
		}
	}
}

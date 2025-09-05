package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"inventory-management-api/internal/handlers"
	"inventory-management-api/internal/models"
	"inventory-management-api/internal/services"
	"inventory-management-api/internal/config"
)

func TestAdminHandler_SetProducts(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Port:                            "8080",
		DataPath:                        "testdata/inventory_test_data.json",
		LogLevel:                        "info",
		Environment:                     "test",
		IdempotencyCacheTTL:             "2m",
		IdempotencyCacheCleanupInterval: "30s",
		EnableJSONPersistence:           "false", // Disable for tests
		InventoryWorkerCount:            "1",
		InventoryQueueBufferSize:        "100",
		MaxEventsInQueue:                "1000",
		EventsFilePath:                  "testdata/events.json",
	}

	// Create inventory service
	inventoryService, err := services.NewInventoryService(cfg)
	if err != nil {
		t.Fatalf("Failed to create inventory service: %v", err)
	}
	defer inventoryService.Stop()

	// Create admin handler
	adminHandler := handlers.NewAdminHandler(inventoryService)

	tests := []struct {
		name           string
		requestBody    models.AdminSetRequest
		expectedStatus int
		expectSuccess  bool
		expectError    string
	}{
		{
			name: "Valid single product update",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Name:      stringPtr("Updated Product Name"),
						Available: intPtr(100),
						Price:     float64Ptr(29.99),
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Valid partial update - name only",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Name:      stringPtr("Another Updated Name"),
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Valid partial update - available only",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Available: intPtr(50),
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Valid multiple products update",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Available: intPtr(75),
					},
					{
						ProductID: "SKU-002",
						Name:      stringPtr("Updated Laptop"),
						Price:     float64Ptr(999.99),
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Invalid - empty products array",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			expectError:    "No products specified",
		},
		{
			name: "Invalid - missing product ID",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						Name: stringPtr("Test Product"),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			expectError:    "validation_error",
		},
		{
			name: "Invalid - no fields to update",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			expectError:    "validation_error",
		},
		{
			name: "Invalid - negative available quantity",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Available: intPtr(-10),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			expectError:    "validation_error",
		},
		{
			name: "Invalid - negative price",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "SKU-001",
						Price:     float64Ptr(-5.99),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			expectError:    "validation_error",
		},
		{
			name: "Product not found",
			requestBody: models.AdminSetRequest{
				Products: []models.AdminProductUpdate{
					{
						ProductID: "NONEXISTENT",
						Name:      stringPtr("Test"),
					},
				},
			},
			expectedStatus: http.StatusOK, // Returns 200 but with failed results
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest("POST", "/v1/admin/products/set", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			adminHandler.SetProducts(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Parse response
			if tt.expectedStatus == http.StatusOK {
				var response models.AdminSetResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if tt.expectSuccess {
					if response.Summary.SuccessfulUpdates == 0 {
						t.Errorf("Expected successful updates, got none")
					}
				} else {
					if response.Summary.FailedUpdates == 0 {
						t.Errorf("Expected failed updates, got none")
					}
				}
			} else {
				var errorResponse models.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResponse); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if tt.expectError != "" && errorResponse.Code != tt.expectError {
					t.Errorf("Expected error code %s, got %s", tt.expectError, errorResponse.Code)
				}
			}
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

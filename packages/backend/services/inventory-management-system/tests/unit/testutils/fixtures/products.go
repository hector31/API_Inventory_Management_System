package fixtures

import (
	"fmt"
	"time"

	"inventory-management-api/internal/models"
)

// ProductFixtures contains predefined test products for consistent testing
type ProductFixtures struct {
	StandardProduct    *TestProduct
	LowStockProduct    *TestProduct
	OutOfStockProduct  *TestProduct
	HighVersionProduct *TestProduct
	ExpensiveProduct   *TestProduct
	CheapProduct       *TestProduct
}

// TestProduct represents a complete test product
type TestProduct struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Available   int       `json:"available"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// GetProductFixtures returns a set of predefined test products
func GetProductFixtures() *ProductFixtures {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	return &ProductFixtures{
		StandardProduct: &TestProduct{
			ID:          "PROD-001",
			Name:        "Standard Test Product",
			Description: "A standard product for general testing",
			Price:       29.99,
			Available:   100,
			Version:     1,
			LastUpdated: baseTime,
		},
		LowStockProduct: &TestProduct{
			ID:          "PROD-002",
			Name:        "Low Stock Product",
			Description: "A product with low stock for boundary testing",
			Price:       19.99,
			Available:   5,
			Version:     1,
			LastUpdated: baseTime.Add(1 * time.Hour),
		},
		OutOfStockProduct: &TestProduct{
			ID:          "PROD-003",
			Name:        "Out of Stock Product",
			Description: "A product with zero stock for edge case testing",
			Price:       39.99,
			Available:   0,
			Version:     1,
			LastUpdated: baseTime.Add(2 * time.Hour),
		},
		HighVersionProduct: &TestProduct{
			ID:          "PROD-004",
			Name:        "High Version Product",
			Description: "A product with high version for concurrency testing",
			Price:       49.99,
			Available:   50,
			Version:     10,
			LastUpdated: baseTime.Add(3 * time.Hour),
		},
		ExpensiveProduct: &TestProduct{
			ID:          "PROD-005",
			Name:        "Expensive Product",
			Description: "An expensive product for value testing",
			Price:       999.99,
			Available:   25,
			Version:     1,
			LastUpdated: baseTime.Add(4 * time.Hour),
		},
		CheapProduct: &TestProduct{
			ID:          "PROD-006",
			Name:        "Cheap Product",
			Description: "A cheap product for value testing",
			Price:       1.99,
			Available:   1000,
			Version:     1,
			LastUpdated: baseTime.Add(5 * time.Hour),
		},
	}
}

// UpdateRequestFixtures contains predefined test update requests
type UpdateRequestFixtures struct {
	ValidDecrease      *models.UpdateRequest
	ValidIncrease      *models.UpdateRequest
	LargeDecrease      *models.UpdateRequest
	ZeroDelta          *models.UpdateRequest
	InvalidVersion     *models.UpdateRequest
	MissingProductID   *models.UpdateRequest
	MissingStoreID     *models.UpdateRequest
	InvalidIdempotency *models.UpdateRequest
}

// GetUpdateRequestFixtures returns predefined test update requests
func GetUpdateRequestFixtures() *UpdateRequestFixtures {
	return &UpdateRequestFixtures{
		ValidDecrease: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          -10,
			Version:        1,
			IdempotencyKey: "valid-decrease-key",
		},
		ValidIncrease: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          20,
			Version:        1,
			IdempotencyKey: "valid-increase-key",
		},
		LargeDecrease: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          -1000,
			Version:        1,
			IdempotencyKey: "large-decrease-key",
		},
		ZeroDelta: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          0,
			Version:        1,
			IdempotencyKey: "zero-delta-key",
		},
		InvalidVersion: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          -5,
			Version:        999,
			IdempotencyKey: "invalid-version-key",
		},
		MissingProductID: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "",
			Delta:          -5,
			Version:        1,
			IdempotencyKey: "missing-product-key",
		},
		MissingStoreID: &models.UpdateRequest{
			StoreID:        "",
			ProductID:      "PROD-001",
			Delta:          -5,
			Version:        1,
			IdempotencyKey: "missing-store-key",
		},
		InvalidIdempotency: &models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-001",
			Delta:          -5,
			Version:        1,
			IdempotencyKey: "",
		},
	}
}

// BatchUpdateFixtures contains predefined batch update requests
type BatchUpdateFixtures struct {
	ValidBatch         *models.UpdateRequest
	MixedValidityBatch *models.UpdateRequest
	EmptyBatch         *models.UpdateRequest
	LargeBatch         *models.UpdateRequest
}

// GetBatchUpdateFixtures returns predefined batch update requests
func GetBatchUpdateFixtures() *BatchUpdateFixtures {
	return &BatchUpdateFixtures{
		ValidBatch: &models.UpdateRequest{
			Updates: []models.ProductUpdate{
				{
					ProductID:      "PROD-001",
					Delta:          -5,
					Version:        1,
					IdempotencyKey: "batch-item-1",
				},
				{
					ProductID:      "PROD-002",
					Delta:          -2,
					Version:        1,
					IdempotencyKey: "batch-item-2",
				},
			},
		},
		MixedValidityBatch: &models.UpdateRequest{
			Updates: []models.ProductUpdate{
				{
					ProductID:      "PROD-001",
					Delta:          -5,
					Version:        1,
					IdempotencyKey: "mixed-valid-1",
				},
				{
					ProductID:      "PROD-003",
					Delta:          -10, // This will fail due to insufficient stock
					Version:        1,
					IdempotencyKey: "mixed-invalid-1",
				},
			},
		},
		EmptyBatch: &models.UpdateRequest{
			Updates: []models.ProductUpdate{},
		},
		LargeBatch: &models.UpdateRequest{
			Updates: generateLargeBatchUpdates(100),
		},
	}
}

// generateLargeBatchUpdates generates a large number of update requests for testing
func generateLargeBatchUpdates(count int) []models.ProductUpdate {
	updates := make([]models.ProductUpdate, count)
	for i := 0; i < count; i++ {
		updates[i] = models.ProductUpdate{
			ProductID:      "PROD-001",
			Delta:          -1,
			Version:        1,
			IdempotencyKey: generateIdempotencyKey(i),
		}
	}
	return updates
}

// generateIdempotencyKey generates a unique idempotency key for testing
func generateIdempotencyKey(index int) string {
	return fmt.Sprintf("large-batch-key-%d-%d", index, time.Now().UnixNano())
}

// ErrorScenarios contains test scenarios for error conditions
type ErrorScenarios struct {
	VersionConflict      TestScenario
	InsufficientStock    TestScenario
	ProductNotFound      TestScenario
	InvalidDelta         TestScenario
	DuplicateIdempotency TestScenario
}

// TestScenario represents a test scenario with setup and expected outcome
type TestScenario struct {
	Name           string
	Description    string
	SetupProduct   *TestProduct
	UpdateRequest  *models.UpdateRequest
	ExpectedError  string
	ExpectedStatus int
}

// GetErrorScenarios returns predefined error test scenarios
func GetErrorScenarios() *ErrorScenarios {
	return &ErrorScenarios{
		VersionConflict: TestScenario{
			Name:        "Version Conflict",
			Description: "Update with outdated version should fail",
			SetupProduct: &TestProduct{
				ID:        "PROD-VERSION-CONFLICT",
				Available: 50,
				Version:   5,
			},
			UpdateRequest: &models.UpdateRequest{
				StoreID:        "store-001",
				ProductID:      "PROD-VERSION-CONFLICT",
				Delta:          -10,
				Version:        3, // Outdated version
				IdempotencyKey: "version-conflict-key",
			},
			ExpectedError:  "version_conflict",
			ExpectedStatus: 409,
		},
		InsufficientStock: TestScenario{
			Name:        "Insufficient Stock",
			Description: "Update that would result in negative stock should fail",
			SetupProduct: &TestProduct{
				ID:        "PROD-INSUFFICIENT",
				Available: 5,
				Version:   1,
			},
			UpdateRequest: &models.UpdateRequest{
				StoreID:        "store-001",
				ProductID:      "PROD-INSUFFICIENT",
				Delta:          -10, // More than available
				Version:        1,
				IdempotencyKey: "insufficient-stock-key",
			},
			ExpectedError:  "insufficient_inventory",
			ExpectedStatus: 400,
		},
		ProductNotFound: TestScenario{
			Name:         "Product Not Found",
			Description:  "Update for non-existent product should fail",
			SetupProduct: nil, // No product setup
			UpdateRequest: &models.UpdateRequest{
				StoreID:        "store-001",
				ProductID:      "PROD-NOT-FOUND",
				Delta:          -5,
				Version:        1,
				IdempotencyKey: "not-found-key",
			},
			ExpectedError:  "product_not_found",
			ExpectedStatus: 404,
		},
	}
}

// PerformanceTestData contains data for performance testing
type PerformanceTestData struct {
	ConcurrentUpdates []models.UpdateRequest
	LargeInventory    []*TestProduct
	HighVolumeUpdates []models.UpdateRequest
}

// GetPerformanceTestData returns data for performance testing
func GetPerformanceTestData() *PerformanceTestData {
	// Generate concurrent updates for the same product
	concurrentUpdates := make([]models.UpdateRequest, 100)
	for i := 0; i < 100; i++ {
		concurrentUpdates[i] = models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      "PROD-CONCURRENT",
			Delta:          -1,
			Version:        1,
			IdempotencyKey: fmt.Sprintf("concurrent-key-%d", i),
		}
	}

	// Generate large inventory
	largeInventory := make([]*TestProduct, 1000)
	for i := 0; i < 1000; i++ {
		largeInventory[i] = &TestProduct{
			ID:        fmt.Sprintf("PROD-%04d", i),
			Available: 100,
			Version:   1,
		}
	}

	// Generate high volume updates
	highVolumeUpdates := make([]models.UpdateRequest, 10000)
	for i := 0; i < 10000; i++ {
		highVolumeUpdates[i] = models.UpdateRequest{
			StoreID:        "store-001",
			ProductID:      fmt.Sprintf("PROD-%04d", i%1000),
			Delta:          -1,
			Version:        1,
			IdempotencyKey: fmt.Sprintf("high-volume-key-%d", i),
		}
	}

	return &PerformanceTestData{
		ConcurrentUpdates: concurrentUpdates,
		LargeInventory:    largeInventory,
		HighVolumeUpdates: highVolumeUpdates,
	}
}

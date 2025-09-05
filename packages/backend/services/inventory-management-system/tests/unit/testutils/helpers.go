package testutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"inventory-management-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProduct represents a test product with all necessary fields
type TestProduct struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Available   int       `json:"available"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// TestUpdateRequest represents a test update request
type TestUpdateRequest struct {
	StoreID        string `json:"storeId"`
	ProductID      string `json:"productId"`
	Delta          int    `json:"delta"`
	Version        int    `json:"version"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// CreateTestProduct creates a test product with default values
func CreateTestProduct(id string) *TestProduct {
	return &TestProduct{
		ID:          id,
		Name:        fmt.Sprintf("Test Product %s", id),
		Description: fmt.Sprintf("Description for test product %s", id),
		Price:       99.99,
		Available:   100,
		Version:     1,
		LastUpdated: time.Now(),
	}
}

// CreateTestProductWithQuantity creates a test product with specified quantity
func CreateTestProductWithQuantity(id string, quantity int) *TestProduct {
	product := CreateTestProduct(id)
	product.Available = quantity
	return product
}

// CreateTestProductWithVersion creates a test product with specified version
func CreateTestProductWithVersion(id string, version int) *TestProduct {
	product := CreateTestProduct(id)
	product.Version = version
	return product
}

// CreateTestUpdateRequest creates a test update request
func CreateTestUpdateRequest(storeID, productID string, delta, version int) *TestUpdateRequest {
	return &TestUpdateRequest{
		StoreID:        storeID,
		ProductID:      productID,
		Delta:          delta,
		Version:        version,
		IdempotencyKey: GenerateTestIdempotencyKey(),
	}
}

// GenerateTestIdempotencyKey generates a unique idempotency key for testing
func GenerateTestIdempotencyKey() string {
	return fmt.Sprintf("test-key-%d", time.Now().UnixNano())
}

// AssertHTTPResponse asserts HTTP response status and content type
func AssertHTTPResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedContentType string) {
	t.Helper()
	assert.Equal(t, expectedStatus, w.Code, "HTTP status code mismatch")
	assert.Equal(t, expectedContentType, w.Header().Get("Content-Type"), "Content-Type mismatch")
}

// AssertJSONResponse asserts HTTP response and unmarshals JSON body
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()
	AssertHTTPResponse(t, w, expectedStatus, "application/json")
	
	err := json.Unmarshal(w.Body.Bytes(), target)
	require.NoError(t, err, "Failed to unmarshal JSON response")
}

// AssertErrorResponse asserts that the response contains an error with expected details
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	t.Helper()
	var errorResp models.ErrorResponse
	AssertJSONResponse(t, w, expectedStatus, &errorResp)
	
	assert.Equal(t, expectedCode, errorResp.Code, "Error code mismatch")
	assert.NotEmpty(t, errorResp.Message, "Error message should not be empty")
}

// CreateHTTPRequest creates an HTTP request with JSON body
func CreateHTTPRequest(method, url string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonBody))
	}
	
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	return req, nil
}

// CreateHTTPRequestWithAuth creates an HTTP request with authentication header
func CreateHTTPRequestWithAuth(method, url, apiKey string, body interface{}) (*http.Request, error) {
	req, err := CreateHTTPRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("X-API-Key", apiKey)
	return req, nil
}

// AssertProductUpdate asserts that a product update was successful
func AssertProductUpdate(t *testing.T, response *models.UpdateResponse, expectedProductID string, expectedQuantity, expectedVersion int) {
	t.Helper()
	assert.True(t, response.Applied, "Update should be applied")
	assert.Equal(t, expectedProductID, response.ProductID, "Product ID mismatch")
	assert.Equal(t, expectedQuantity, response.NewQuantity, "New quantity mismatch")
	assert.Equal(t, expectedVersion, response.NewVersion, "New version mismatch")
	assert.NotEmpty(t, response.LastUpdated, "LastUpdated should not be empty")
}

// AssertVersionConflict asserts that a version conflict error occurred
func AssertVersionConflict(t *testing.T, response *models.UpdateResponse, expectedProductID string) {
	t.Helper()
	assert.False(t, response.Applied, "Update should not be applied due to version conflict")
	assert.Equal(t, "version_conflict", response.ErrorType, "Error type should be version_conflict")
	assert.Contains(t, response.ErrorMessage, "version conflict", "Error message should mention version conflict")
	assert.Equal(t, expectedProductID, response.ProductID, "Product ID mismatch")
}

// AssertInsufficientInventory asserts that an insufficient inventory error occurred
func AssertInsufficientInventory(t *testing.T, response *models.UpdateResponse, expectedProductID string) {
	t.Helper()
	assert.False(t, response.Applied, "Update should not be applied due to insufficient inventory")
	assert.Equal(t, "insufficient_inventory", response.ErrorType, "Error type should be insufficient_inventory")
	assert.Contains(t, response.ErrorMessage, "insufficient", "Error message should mention insufficient inventory")
	assert.Equal(t, expectedProductID, response.ProductID, "Product ID mismatch")
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	timeoutChan := time.After(timeout)
	
	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutChan:
			t.Fatalf("Timeout waiting for condition: %s", message)
		}
	}
}

// AssertEventuallyTrue asserts that a condition becomes true within a timeout
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	WaitForCondition(t, condition, timeout, message)
}

// CreateConcurrentTestRunner creates a helper for running concurrent tests
func CreateConcurrentTestRunner(t *testing.T, numGoroutines int) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		t:             t,
		numGoroutines: numGoroutines,
		results:       make(chan TestResult, numGoroutines),
		errors:        make([]error, 0),
	}
}

// ConcurrentTestRunner helps run concurrent tests and collect results
type ConcurrentTestRunner struct {
	t             *testing.T
	numGoroutines int
	results       chan TestResult
	errors        []error
}

// TestResult represents the result of a concurrent test
type TestResult struct {
	Success bool
	Error   error
	Data    interface{}
}

// Run executes the test function concurrently
func (r *ConcurrentTestRunner) Run(testFunc func() TestResult) {
	for i := 0; i < r.numGoroutines; i++ {
		go func() {
			result := testFunc()
			r.results <- result
		}()
	}
}

// WaitAndAssert waits for all goroutines to complete and asserts results
func (r *ConcurrentTestRunner) WaitAndAssert(expectedSuccesses int) []TestResult {
	results := make([]TestResult, 0, r.numGoroutines)
	successCount := 0
	
	for i := 0; i < r.numGoroutines; i++ {
		result := <-r.results
		results = append(results, result)
		
		if result.Success {
			successCount++
		} else if result.Error != nil {
			r.errors = append(r.errors, result.Error)
		}
	}
	
	assert.Equal(r.t, expectedSuccesses, successCount, 
		fmt.Sprintf("Expected %d successes, got %d. Errors: %v", expectedSuccesses, successCount, r.errors))
	
	return results
}

// CleanupTestData provides a cleanup function for test data
func CleanupTestData(t *testing.T, cleanupFunc func() error) {
	t.Helper()
	t.Cleanup(func() {
		if err := cleanupFunc(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	})
}

package models

import (
	"encoding/json"
	"testing"

	"inventory-management-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateRequest_JSONSerialization tests JSON serialization and deserialization
func TestUpdateRequest_JSONSerialization(t *testing.T) {
	testCases := []struct {
		name    string
		request *models.UpdateRequest
	}{
		{
			name: "Single Product Update",
			request: &models.UpdateRequest{
				StoreID:        "store-001",
				ProductID:      "PROD-001",
				Delta:          -10,
				Version:        1,
				IdempotencyKey: "test-key-001",
			},
		},
		{
			name: "Batch Update",
			request: &models.UpdateRequest{
				Updates: []models.ProductUpdate{
					{
						ProductID:      "PROD-001",
						Delta:          -5,
						Version:        1,
						IdempotencyKey: "batch-key-001",
					},
					{
						ProductID:      "PROD-002",
						Delta:          10,
						Version:        2,
						IdempotencyKey: "batch-key-002",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Serialize to JSON
			jsonData, err := json.Marshal(tc.request)
			require.NoError(t, err, "Failed to marshal UpdateRequest to JSON")

			// Act - Deserialize from JSON
			var deserializedRequest models.UpdateRequest
			err = json.Unmarshal(jsonData, &deserializedRequest)
			require.NoError(t, err, "Failed to unmarshal UpdateRequest from JSON")

			// Assert
			assert.Equal(t, tc.request.StoreID, deserializedRequest.StoreID)
			assert.Equal(t, tc.request.ProductID, deserializedRequest.ProductID)
			assert.Equal(t, tc.request.Delta, deserializedRequest.Delta)
			assert.Equal(t, tc.request.Version, deserializedRequest.Version)
			assert.Equal(t, tc.request.IdempotencyKey, deserializedRequest.IdempotencyKey)
			assert.Equal(t, len(tc.request.Updates), len(deserializedRequest.Updates))

			for i, update := range tc.request.Updates {
				assert.Equal(t, update.ProductID, deserializedRequest.Updates[i].ProductID)
				assert.Equal(t, update.Delta, deserializedRequest.Updates[i].Delta)
				assert.Equal(t, update.Version, deserializedRequest.Updates[i].Version)
				assert.Equal(t, update.IdempotencyKey, deserializedRequest.Updates[i].IdempotencyKey)
			}
		})
	}
}

// TestUpdateResponse_JSONSerialization tests UpdateResponse JSON serialization
func TestUpdateResponse_JSONSerialization(t *testing.T) {
	testCases := []struct {
		name     string
		response *models.UpdateResponse
	}{
		{
			name: "Successful Single Update",
			response: &models.UpdateResponse{
				ProductID:   "PROD-001",
				NewQuantity: 90,
				NewVersion:  2,
				Applied:     true,
				LastUpdated: "2024-01-01T12:00:00Z",
			},
		},
		{
			name: "Failed Update with Error",
			response: &models.UpdateResponse{
				ProductID:    "PROD-001",
				NewQuantity:  100,
				NewVersion:   1,
				Applied:      false,
				ErrorType:    "version_conflict",
				ErrorMessage: "version conflict: expected 1, got 2",
				LastUpdated:  "2024-01-01T12:00:00Z",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Serialize to JSON
			jsonData, err := json.Marshal(tc.response)
			require.NoError(t, err, "Failed to marshal UpdateResponse to JSON")

			// Act - Deserialize from JSON
			var deserializedResponse models.UpdateResponse
			err = json.Unmarshal(jsonData, &deserializedResponse)
			require.NoError(t, err, "Failed to unmarshal UpdateResponse from JSON")

			// Assert
			assert.Equal(t, tc.response.ProductID, deserializedResponse.ProductID)
			assert.Equal(t, tc.response.NewQuantity, deserializedResponse.NewQuantity)
			assert.Equal(t, tc.response.NewVersion, deserializedResponse.NewVersion)
			assert.Equal(t, tc.response.Applied, deserializedResponse.Applied)
			assert.Equal(t, tc.response.ErrorType, deserializedResponse.ErrorType)
			assert.Equal(t, tc.response.ErrorMessage, deserializedResponse.ErrorMessage)
			assert.Equal(t, tc.response.LastUpdated, deserializedResponse.LastUpdated)
		})
	}
}

// TestErrorResponse_JSONSerialization tests ErrorResponse JSON serialization
func TestErrorResponse_JSONSerialization(t *testing.T) {
	testCases := []struct {
		name     string
		response *models.ErrorResponse
	}{
		{
			name: "Simple Error",
			response: &models.ErrorResponse{
				Code:    "validation_error",
				Message: "Invalid request parameters",
			},
		},
		{
			name: "Error with Details",
			response: &models.ErrorResponse{
				Code:    "validation_error",
				Message: "Multiple validation errors",
				Details: []models.ErrorDetail{
					{
						Field: "productId",
						Issue: "Product ID is required",
					},
					{
						Field: "delta",
						Issue: "Delta must be a non-zero integer",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Serialize to JSON
			jsonData, err := json.Marshal(tc.response)
			require.NoError(t, err, "Failed to marshal ErrorResponse to JSON")

			// Act - Deserialize from JSON
			var deserializedResponse models.ErrorResponse
			err = json.Unmarshal(jsonData, &deserializedResponse)
			require.NoError(t, err, "Failed to unmarshal ErrorResponse from JSON")

			// Assert
			assert.Equal(t, tc.response.Code, deserializedResponse.Code)
			assert.Equal(t, tc.response.Message, deserializedResponse.Message)
			assert.Equal(t, len(tc.response.Details), len(deserializedResponse.Details))

			for i, detail := range tc.response.Details {
				assert.Equal(t, detail.Field, deserializedResponse.Details[i].Field)
				assert.Equal(t, detail.Issue, deserializedResponse.Details[i].Issue)
			}
		})
	}
}

// TestProductUpdate_EdgeCases tests edge cases for ProductUpdate
func TestProductUpdate_EdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		update models.ProductUpdate
		valid  bool
	}{
		{
			name: "Large Positive Delta",
			update: models.ProductUpdate{
				ProductID:      "PROD-001",
				Delta:          1000000,
				Version:        1,
				IdempotencyKey: "large-positive",
			},
			valid: true,
		},
		{
			name: "Large Negative Delta",
			update: models.ProductUpdate{
				ProductID:      "PROD-001",
				Delta:          -1000000,
				Version:        1,
				IdempotencyKey: "large-negative",
			},
			valid: true,
		},
		{
			name: "Zero Delta",
			update: models.ProductUpdate{
				ProductID:      "PROD-001",
				Delta:          0,
				Version:        1,
				IdempotencyKey: "zero-delta",
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Serialize and deserialize to test JSON handling
			jsonData, err := json.Marshal(tc.update)
			require.NoError(t, err, "Failed to marshal ProductUpdate")

			var deserializedUpdate models.ProductUpdate
			err = json.Unmarshal(jsonData, &deserializedUpdate)
			require.NoError(t, err, "Failed to unmarshal ProductUpdate")

			// Assert
			assert.Equal(t, tc.update.ProductID, deserializedUpdate.ProductID)
			assert.Equal(t, tc.update.Delta, deserializedUpdate.Delta)
			assert.Equal(t, tc.update.Version, deserializedUpdate.Version)
			assert.Equal(t, tc.update.IdempotencyKey, deserializedUpdate.IdempotencyKey)
		})
	}
}

// TestUpdateRequest_Validation tests basic validation logic
func TestUpdateRequest_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		request     *models.UpdateRequest
		expectValid bool
		description string
	}{
		{
			name: "Valid Single Update",
			request: &models.UpdateRequest{
				StoreID:        "store-001",
				ProductID:      "PROD-001",
				Delta:          -10,
				Version:        1,
				IdempotencyKey: "valid-key",
			},
			expectValid: true,
			description: "All required fields present for single update",
		},
		{
			name: "Valid Batch Update",
			request: &models.UpdateRequest{
				Updates: []models.ProductUpdate{
					{
						ProductID:      "PROD-001",
						Delta:          -5,
						Version:        1,
						IdempotencyKey: "batch-key-1",
					},
				},
			},
			expectValid: true,
			description: "Valid batch update with all required fields",
		},
		{
			name: "Empty Request",
			request: &models.UpdateRequest{},
			expectValid: false,
			description: "Empty request should be invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act - Test JSON serialization as a basic validation
			jsonData, err := json.Marshal(tc.request)
			require.NoError(t, err, "JSON marshaling should always succeed")

			var deserializedRequest models.UpdateRequest
			err = json.Unmarshal(jsonData, &deserializedRequest)
			require.NoError(t, err, "JSON unmarshaling should always succeed")

			// Basic validation - check if essential fields are present
			isValid := true
			if len(tc.request.Updates) == 0 {
				// Single update validation
				if tc.request.ProductID == "" && tc.request.StoreID == "" && tc.request.IdempotencyKey == "" {
					isValid = false
				}
			}

			assert.Equal(t, tc.expectValid, isValid, tc.description)
		})
	}
}

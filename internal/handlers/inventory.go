package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"inventory-management-api/internal/models"

	"github.com/gorilla/mux"
)

// InventoryHandler handles inventory-related HTTP requests
type InventoryHandler struct{}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler() *InventoryHandler {
	return &InventoryHandler{}
}

// writeJSONResponse is a helper function to write JSON responses
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse is a helper function to write error responses
func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details []models.ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// UpdateInventory handles POST /inventory/updates - Mutate stock
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid JSON", nil)
		return
	}

	// Placeholder response - in real implementation, this would apply OCC and update state
	response := models.UpdateResponse{
		ProductID:   req.ProductID,
		NewQuantity: 20, // Placeholder value
		NewVersion:  req.Version + 1,
		Applied:     true,
		LastUpdated: "2025-09-02T10:00:00Z",
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// SyncInventory handles POST /inventory/sync - Bulk sync for reconciliation
func (h *InventoryHandler) SyncInventory(w http.ResponseWriter, r *http.Request) {
	var req models.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid JSON", nil)
		return
	}

	// Placeholder response
	response := models.SyncResponse{
		Updated: len(req.Products),
		Created: 0,
		Skipped: 0,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetProduct handles GET /inventory/{productId} - Read product
func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["productId"]

	// Placeholder response
	response := models.ProductResponse{
		ProductID:   productID,
		Available:   20,
		Version:     5,
		LastUpdated: "2025-09-02T10:00:00Z",
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetGlobalAvailability handles GET /inventory/global/{productId} - Global availability
func (h *InventoryHandler) GetGlobalAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["productId"]

	// Placeholder response
	response := models.GlobalAvailabilityResponse{
		ProductID:      productID,
		TotalAvailable: 420,
		PerStore: map[string]int{
			"store-1": 50,
			"store-7": 19,
			"store-3": 351,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// ListProducts handles GET /inventory - List products with cursor pagination
func (h *InventoryHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := r.URL.Query().Get("limit")

	// Placeholder response
	response := models.ListResponse{
		Items: []models.ProductResponse{
			{ProductID: "SKU-1", Available: 20, Version: 3, LastUpdated: "2025-09-02T10:00:00Z"},
			{ProductID: "SKU-2", Available: 5, Version: 1, LastUpdated: "2025-09-02T09:30:00Z"},
		},
		NextCursor: "", // Empty means no more pages
	}

	fmt.Printf("Listing products with cursor: %s, limit: %s\n", cursor, limit)
	writeJSONResponse(w, http.StatusOK, response)
}

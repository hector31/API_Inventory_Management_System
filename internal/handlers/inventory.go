package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"inventory-management-api/internal/models"
	"inventory-management-api/internal/services"

	"github.com/gorilla/mux"
)

// InventoryHandler handles inventory-related HTTP requests
type InventoryHandler struct {
	inventoryService *services.InventoryService
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(inventoryService *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
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

// GetProduct handles GET /v1/inventory/{productId} - Read product
func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["productId"]

	// Validate that productId is not empty
	if productID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "bad_request", "Product ID is required", []models.ErrorDetail{
			{Field: "productId", Issue: "cannot be empty"},
		})
		return
	}

	// Get the product from the service
	product, err := h.inventoryService.GetProduct(productID)
	if err != nil {
		// If product doesn't exist, return 404
		writeErrorResponse(w, http.StatusNotFound, "not_found", fmt.Sprintf("Product not found: %s", productID), nil)
		return
	}

	// Return successful response
	writeJSONResponse(w, http.StatusOK, product)
}

// ListProducts handles GET /v1/inventory - List products with cursor pagination
func (h *InventoryHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	// Parse the limit (default 50)
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	slog.Debug("Listing products",
		"cursor", cursor,
		"limit", limit,
		"remote_addr", r.RemoteAddr)

	// Get the product list from the service
	productList, err := h.inventoryService.ListProducts(cursor, limit)
	if err != nil {
		slog.Error("Failed to retrieve products", "error", err, "remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Error retrieving products", nil)
		return
	}

	slog.Info("Products listed successfully",
		"cursor", cursor,
		"limit", limit,
		"found_count", len(productList.Items),
		"remote_addr", r.RemoteAddr)

	writeJSONResponse(w, http.StatusOK, productList)
}

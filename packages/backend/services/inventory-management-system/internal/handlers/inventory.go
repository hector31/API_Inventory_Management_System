package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
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

// UpdateInventory handles POST /v1/inventory/updates - Mutate stock (single or batch)
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Invalid JSON in update request", "error", err, "remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid JSON", nil)
		return
	}

	// Determine if this is a single update or batch update
	if len(req.Updates) > 0 {
		// Batch update operation
		slog.Info("Processing batch inventory update",
			"store_id", req.StoreID,
			"update_count", len(req.Updates),
			"remote_addr", r.RemoteAddr)

		response := h.processBatchUpdate(req)

		// For batch updates, return 200 even if some items failed
		// The client can check individual results
		writeJSONResponse(w, http.StatusOK, response)
	} else {
		// Single update operation
		slog.Info("Processing single inventory update",
			"store_id", req.StoreID,
			"product_id", req.ProductID,
			"delta", req.Delta,
			"remote_addr", r.RemoteAddr)

		response := h.processSingleUpdate(req)

		// For single updates, return appropriate HTTP status based on result
		if !response.Applied {
			// Check if it's a version conflict or other error
			if req.Version >= 0 {
				// Likely a version conflict or business logic error
				writeJSONResponse(w, http.StatusConflict, response)
			} else {
				// Bad request (missing fields, etc.)
				writeJSONResponse(w, http.StatusBadRequest, response)
			}
		} else {
			writeJSONResponse(w, http.StatusOK, response)
		}
	}
}

// processSingleUpdate handles single product updates with OCC and idempotency
func (h *InventoryHandler) processSingleUpdate(req models.UpdateRequest) models.UpdateResponse {
	// Validate single update request
	if req.ProductID == "" {
		slog.Warn("Missing product ID in single update")
		return models.UpdateResponse{
			ProductID:    req.ProductID,
			ErrorType:    services.ErrTypeMissingProductID,
			ErrorMessage: "Missing product ID",
			Applied:      false,
		}
	}

	if req.IdempotencyKey == "" {
		slog.Warn("Missing idempotency key in single update", "product_id", req.ProductID)
		return models.UpdateResponse{
			ProductID:    req.ProductID,
			ErrorType:    services.ErrTypeInvalidRequest,
			ErrorMessage: "Missing idempotency key",
			Applied:      false,
		}
	}

	// Submit update to queue-based service
	result, err := h.inventoryService.UpdateInventory(
		req.ProductID,
		req.Delta,
		req.Version,
		req.IdempotencyKey,
		req.StoreID,
	)

	if err != nil {
		slog.Error("Failed to process single update",
			"product_id", req.ProductID,
			"error", err)
		return models.UpdateResponse{
			ProductID:    req.ProductID,
			Applied:      false,
			NewQuantity:  0,
			NewVersion:   0,
			ErrorType:    services.ErrTypeInternalError,
			ErrorMessage: err.Error(),
			LastUpdated:  "",
		}
	}

	response := models.UpdateResponse{
		ProductID:    req.ProductID,
		NewQuantity:  result.NewQuantity,
		NewVersion:   result.NewVersion,
		Applied:      result.Applied,
		LastUpdated:  result.LastUpdated,
		ErrorType:    result.ErrorType,
		ErrorMessage: result.ErrorMessage,
	}

	if result.Success {
		slog.Info("Single update processed successfully",
			"product_id", req.ProductID,
			"new_quantity", response.NewQuantity,
			"new_version", response.NewVersion,
			"delta", req.Delta,
			"idempotency_key", req.IdempotencyKey)
	} else {
		slog.Warn("Single update failed",
			"product_id", req.ProductID,
			"error_type", result.ErrorType,
			"error_message", result.ErrorMessage,
			"idempotency_key", req.IdempotencyKey)
	}

	return response
}

// processBatchUpdate handles batch product updates with OCC and idempotency
func (h *InventoryHandler) processBatchUpdate(req models.UpdateRequest) models.UpdateResponse {
	results := make([]models.ProductUpdateResult, 0, len(req.Updates))
	succeeded := 0
	failed := 0

	for _, update := range req.Updates {
		if update.ProductID == "" {
			slog.Warn("Missing product ID in batch update item")
			results = append(results, models.ProductUpdateResult{
				ProductID:    update.ProductID,
				Applied:      false,
				ErrorType:    services.ErrTypeMissingProductID,
				ErrorMessage: "Missing product ID",
			})
			failed++
			continue
		}

		if update.IdempotencyKey == "" {
			slog.Warn("Missing idempotency key in batch update item", "product_id", update.ProductID)
			results = append(results, models.ProductUpdateResult{
				ProductID:    update.ProductID,
				Applied:      false,
				ErrorType:    services.ErrTypeInvalidRequest,
				ErrorMessage: "Missing idempotency key",
			})
			failed++
			continue
		}

		// Submit update to queue-based service
		serviceResult, err := h.inventoryService.UpdateInventory(
			update.ProductID,
			update.Delta,
			update.Version,
			update.IdempotencyKey,
			req.StoreID,
		)

		var result models.ProductUpdateResult
		if err != nil {
			slog.Error("Failed to process batch update item",
				"product_id", update.ProductID,
				"error", err)
			result = models.ProductUpdateResult{
				ProductID:    update.ProductID,
				Applied:      false,
				ErrorType:    services.ErrTypeInternalError,
				ErrorMessage: err.Error(),
			}
			failed++
		} else if !serviceResult.Success {
			result = models.ProductUpdateResult{
				ProductID:    update.ProductID,
				NewQuantity:  serviceResult.NewQuantity,
				NewVersion:   serviceResult.NewVersion,
				Applied:      false,
				LastUpdated:  serviceResult.LastUpdated,
				ErrorType:    serviceResult.ErrorType,
				ErrorMessage: serviceResult.ErrorMessage,
			}
			failed++
		} else {
			result = models.ProductUpdateResult{
				ProductID:   update.ProductID,
				NewQuantity: serviceResult.NewQuantity,
				NewVersion:  serviceResult.NewVersion,
				Applied:     true,
				LastUpdated: serviceResult.LastUpdated,
			}
			succeeded++
		}

		results = append(results, result)

		slog.Debug("Batch update item processed",
			"product_id", update.ProductID,
			"new_quantity", result.NewQuantity,
			"applied", result.Applied,
			"idempotency_key", update.IdempotencyKey)
	}

	response := models.UpdateResponse{
		Results: results,
		Summary: &models.BatchSummary{
			Total:     len(req.Updates),
			Succeeded: succeeded,
			Failed:    failed,
		},
	}

	slog.Info("Batch update completed",
		"total", response.Summary.Total,
		"succeeded", response.Summary.Succeeded,
		"failed", response.Summary.Failed)

	return response
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

// ListProducts handles GET /v1/inventory - List products with offset-based pagination
func (h *InventoryHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	// Parse offset (default 0)
	offset := 0
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Parse limit (default 50, max 200)
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			// Cap the limit to prevent excessive resource usage
			if limit > 200 {
				limit = 200
			}
		}
	}

	slog.Debug("Listing products with pagination",
		"offset", offset,
		"limit", limit,
		"remote_addr", r.RemoteAddr)

	// Get all products from inventory service using existing method
	// We'll get all products and then apply our own pagination for deterministic results
	productList, err := h.inventoryService.ListProducts("", 0) // Get all products (limit 0 = no limit)
	if err != nil {
		slog.Error("Failed to get products from inventory service", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to slice and sort by product_id for deterministic pagination
	allProducts := productList.Items
	sort.Slice(allProducts, func(i, j int) bool {
		return allProducts[i].ProductID < allProducts[j].ProductID
	})

	// Calculate total count
	totalCount := len(allProducts)

	// Apply pagination
	var paginatedProducts []models.ProductResponse
	if offset < totalCount {
		end := offset + limit
		if end > totalCount {
			end = totalCount
		}
		paginatedProducts = allProducts[offset:end]
	} else {
		// Offset beyond available products
		paginatedProducts = []models.ProductResponse{}
	}

	// Create response with pagination metadata
	response := map[string]interface{}{
		"products": paginatedProducts,
		"pagination": map[string]interface{}{
			"offset":      offset,
			"limit":       limit,
			"total_count": totalCount,
			"has_more":    offset+limit < totalCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	slog.Debug("Successfully returned products",
		"returned_count", len(paginatedProducts),
		"total_count", totalCount,
		"offset", offset,
		"limit", limit)
}

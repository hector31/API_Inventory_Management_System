package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/melibackend/shared/client"
	"github.com/melibackend/shared/models"
	"github.com/melibackend/shared/storage"
	"github.com/melibackend/shared/sync"
)

// InventoryHandler handles inventory-related requests
type InventoryHandler struct {
	inventoryClient *client.InventoryClient
	localStorage    storage.LocalStorage
	syncManager     sync.SyncManager
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(inventoryClient *client.InventoryClient, localStorage storage.LocalStorage, syncManager sync.SyncManager) *InventoryHandler {
	return &InventoryHandler{
		inventoryClient: inventoryClient,
		localStorage:    localStorage,
		syncManager:     syncManager,
	}
}

// GetAllProducts handles GET /v1/store/inventory with pagination support (using local cache)
func (h *InventoryHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	slog.Info("Getting all products for store from local cache", "remote_addr", r.RemoteAddr)

	// Parse query parameters for pagination
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

	// Get all products from local storage
	allProducts, err := h.localStorage.GetAllProducts()
	if err != nil {
		slog.Error("Failed to get products from local storage", "error", err)
		h.writeErrorResponse(w, "storage_error", "Failed to retrieve products", http.StatusInternalServerError, nil)
		return
	}

	// Convert models.Product to response format with consistent field names
	var productResponses []map[string]interface{}
	for _, product := range allProducts {
		productResponse := map[string]interface{}{
			"productId":   product.ProductID,
			"name":        product.Name,
			"available":   product.Available,
			"version":     product.Version,
			"lastUpdated": product.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
			"price":       product.Price,
		}
		productResponses = append(productResponses, productResponse)
	}

	// Sort products by productId for deterministic pagination
	sort.Slice(productResponses, func(i, j int) bool {
		return productResponses[i]["productId"].(string) < productResponses[j]["productId"].(string)
	})

	// Calculate total count
	totalCount := len(productResponses)

	// Apply pagination
	var paginatedProducts []map[string]interface{}
	if offset < totalCount {
		end := offset + limit
		if end > totalCount {
			end = totalCount
		}
		paginatedProducts = productResponses[offset:end]
	} else {
		// Offset beyond available products
		paginatedProducts = []map[string]interface{}{}
	}

	// Create response with pagination metadata (matching Central API format)
	response := map[string]interface{}{
		"products": paginatedProducts,
		"pagination": map[string]interface{}{
			"offset":      offset,
			"limit":       limit,
			"total_count": totalCount,
			"has_more":    offset+limit < totalCount,
		},
	}

	slog.Info("Successfully retrieved products from local cache",
		"total_count", totalCount,
		"returned_count", len(paginatedProducts),
		"offset", offset,
		"limit", limit)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetProduct handles GET /v1/store/inventory/{productId} (now using local cache)
func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")

	slog.Info("Getting product for store from local cache", "product_id", productID, "remote_addr", r.RemoteAddr)

	product, err := h.localStorage.GetProduct(productID)
	if err != nil {
		slog.Error("Failed to get product from local storage", "product_id", productID, "error", err)

		if err.Error() == fmt.Sprintf("product not found: %s", productID) {
			h.writeErrorResponse(w, "product_not_found", "Product not found", http.StatusNotFound, map[string]string{"productId": productID})
			return
		}

		h.writeErrorResponse(w, "storage_error", "Failed to retrieve product", http.StatusInternalServerError, nil)
		return
	}

	// Convert to consistent response format
	productResponse := map[string]interface{}{
		"productId":   product.ProductID,
		"name":        product.Name,
		"available":   product.Available,
		"version":     product.Version,
		"lastUpdated": product.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
		"price":       product.Price,
	}

	slog.Info("Successfully retrieved product from local cache",
		"product_id", productID,
		"name", product.Name,
		"available", product.Available,
		"version", product.Version)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(productResponse)
}

// UpdateInventory handles POST /v1/store/inventory/updates
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	var updateReq models.UpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		slog.Error("Failed to decode update request", "error", err)
		h.writeErrorResponse(w, "invalid_request", "Invalid request body", http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("Processing inventory update for store",
		"store_id", updateReq.StoreID,
		"product_id", updateReq.ProductID,
		"delta", updateReq.Delta,
		"version", updateReq.Version,
		"remote_addr", r.RemoteAddr,
	)

	// Add store identifier to idempotency key to avoid conflicts
	updateReq.IdempotencyKey = fmt.Sprintf("store-s1-%s", updateReq.IdempotencyKey)

	updateResp, err := h.inventoryClient.UpdateInventory(updateReq)
	if err != nil {
		slog.Error("Failed to update inventory via central API",
			"product_id", updateReq.ProductID,
			"error", err,
		)

		// Enhanced error handling with proper HTTP status codes
		h.handleInventoryUpdateError(w, updateReq, err)
		return
	}

	// Update local cache if the central update was successful and applied
	if updateResp.Applied {
		if err := h.syncManager.UpdateLocalProduct(
			updateResp.ProductID,
			updateResp.NewQuantity,
			updateResp.NewVersion,
			time.Now(),
		); err != nil {
			slog.Warn("Failed to update local cache after successful central update",
				"product_id", updateResp.ProductID,
				"error", err,
			)
		} else {
			slog.Debug("Local cache updated after successful central update",
				"product_id", updateResp.ProductID,
				"new_quantity", updateResp.NewQuantity,
				"new_version", updateResp.NewVersion,
			)
		}
	}

	slog.Info("Successfully updated inventory",
		"product_id", updateReq.ProductID,
		"new_quantity", updateResp.NewQuantity,
		"new_version", updateResp.NewVersion,
		"applied", updateResp.Applied,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateResp)
}

// BatchUpdateInventory handles POST /v1/store/inventory/batch-updates
func (h *InventoryHandler) BatchUpdateInventory(w http.ResponseWriter, r *http.Request) {
	var batchReq models.BatchUpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		slog.Error("Failed to decode batch update request", "error", err)
		h.writeErrorResponse(w, "invalid_request", "Invalid request body", http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("Processing batch inventory update for store",
		"store_id", batchReq.StoreID,
		"update_count", len(batchReq.Updates),
		"remote_addr", r.RemoteAddr,
	)

	// Add store identifier to idempotency keys to avoid conflicts
	for i := range batchReq.Updates {
		batchReq.Updates[i].IdempotencyKey = fmt.Sprintf("store-s1-%s", batchReq.Updates[i].IdempotencyKey)
	}

	batchResp, err := h.inventoryClient.BatchUpdateInventory(batchReq)
	if err != nil {
		slog.Error("Failed to batch update inventory via central API",
			"store_id", batchReq.StoreID,
			"error", err,
		)
		h.writeErrorResponse(w, "batch_update_failed", "Failed to batch update inventory", http.StatusInternalServerError, map[string]interface{}{
			"storeId":     batchReq.StoreID,
			"updateCount": len(batchReq.Updates),
			"error":       err.Error(),
		})
		return
	}

	slog.Info("Successfully processed batch inventory update",
		"store_id", batchReq.StoreID,
		"total_count", batchResp.TotalCount,
		"success_count", batchResp.SuccessCount,
		"failure_count", batchResp.FailureCount,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(batchResp)
}

// GetSyncStatus handles GET /v1/store/sync/status
func (h *InventoryHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Getting sync status", "remote_addr", r.RemoteAddr)

	status := h.syncManager.GetSyncStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// ForceSync handles POST /v1/store/sync/force
func (h *InventoryHandler) ForceSync(w http.ResponseWriter, r *http.Request) {
	slog.Info("Force sync requested", "remote_addr", r.RemoteAddr)

	ctx := r.Context()
	if err := h.syncManager.ForceSync(ctx); err != nil {
		slog.Error("Force sync failed", "error", err)
		h.writeErrorResponse(w, "sync_failed", "Force sync failed", http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"message": "Force sync completed successfully",
		"status":  h.syncManager.GetSyncStatus(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetCacheStats handles GET /v1/store/cache/stats
func (h *InventoryHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Getting cache stats", "remote_addr", r.RemoteAddr)

	stats, err := h.localStorage.GetStorageStats()
	if err != nil {
		slog.Error("Failed to get storage stats", "error", err)
		h.writeErrorResponse(w, "stats_error", "Failed to get cache stats", http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// handleInventoryUpdateError handles errors from inventory updates with proper HTTP status codes
func (h *InventoryHandler) handleInventoryUpdateError(w http.ResponseWriter, updateReq models.UpdateRequest, err error) {
	errorStr := err.Error()

	slog.Info("Enhanced error handling called",
		"product_id", updateReq.ProductID,
		"error_string", errorStr)

	// Try to parse the error response from central API if it's a structured error
	if centralAPIError := h.parseCentralAPIError(errorStr); centralAPIError != nil {
		slog.Info("Successfully parsed central API error, returning standardized response",
			"error_type", centralAPIError.ErrorType,
			"status_code", centralAPIError.StatusCode)
		// Return the error in the standardized format expected by frontend
		h.writeStandardizedErrorResponse(w, centralAPIError, updateReq.ProductID)
		return
	}

	// Fallback: parse error string for known patterns
	switch {
	case strings.Contains(errorStr, "version conflict"):
		h.writeStandardizedErrorResponse(w, &StandardizedError{
			ErrorType:    "version_conflict",
			ErrorMessage: errorStr,
			StatusCode:   http.StatusConflict,
		}, updateReq.ProductID)

	case strings.Contains(errorStr, "insufficient inventory"):
		h.writeStandardizedErrorResponse(w, &StandardizedError{
			ErrorType:    "insufficient_inventory",
			ErrorMessage: errorStr,
			StatusCode:   http.StatusBadRequest,
		}, updateReq.ProductID)

	case strings.Contains(errorStr, "product not found"):
		h.writeStandardizedErrorResponse(w, &StandardizedError{
			ErrorType:    "product_not_found",
			ErrorMessage: errorStr,
			StatusCode:   http.StatusNotFound,
		}, updateReq.ProductID)

	case strings.Contains(errorStr, "invalid"):
		h.writeStandardizedErrorResponse(w, &StandardizedError{
			ErrorType:    "invalid_request",
			ErrorMessage: errorStr,
			StatusCode:   http.StatusBadRequest,
		}, updateReq.ProductID)

	default:
		// Actual server error
		h.writeStandardizedErrorResponse(w, &StandardizedError{
			ErrorType:    "server_error",
			ErrorMessage: "Internal server error occurred",
			StatusCode:   http.StatusInternalServerError,
		}, updateReq.ProductID)
	}
}

// StandardizedError represents a parsed error with proper categorization
type StandardizedError struct {
	ErrorType    string `json:"errorType"`
	ErrorMessage string `json:"errorMessage"`
	ProductID    string `json:"productId,omitempty"`
	NewVersion   int    `json:"newVersion,omitempty"`
	NewQuantity  int    `json:"newQuantity,omitempty"`
	LastUpdated  string `json:"lastUpdated,omitempty"`
	StatusCode   int    `json:"-"` // Not included in JSON response
}

// parseCentralAPIError attempts to parse structured error responses from central API
func (h *InventoryHandler) parseCentralAPIError(errorStr string) *StandardizedError {
	slog.Debug("Parsing central API error", "error_string", errorStr)

	// Look for the pattern "request failed with status XXX: {JSON}"
	statusPattern := "request failed with status "
	statusIndex := strings.Index(errorStr, statusPattern)
	if statusIndex == -1 {
		slog.Debug("No status pattern found in error string")
		return nil
	}

	// Find the JSON part after the status
	jsonStart := strings.Index(errorStr[statusIndex:], "{")
	if jsonStart == -1 {
		slog.Debug("No JSON start found after status pattern")
		return nil
	}
	jsonStart += statusIndex

	// Find the end of JSON (look for the last } before \n)
	jsonEnd := strings.Index(errorStr[jsonStart:], "\\n")
	if jsonEnd == -1 {
		jsonEnd = len(errorStr)
	} else {
		jsonEnd += jsonStart
	}

	// Find the actual end of JSON
	jsonEndBrace := strings.LastIndex(errorStr[jsonStart:jsonEnd], "}")
	if jsonEndBrace == -1 {
		slog.Debug("No JSON end found")
		return nil
	}
	jsonEnd = jsonStart + jsonEndBrace + 1

	jsonStr := errorStr[jsonStart:jsonEnd]
	slog.Debug("Extracted JSON from error", "json", jsonStr)

	// Try to parse as UpdateResponse (which contains error information)
	var updateResp struct {
		ProductID    string `json:"productId"`
		Applied      bool   `json:"applied"`
		NewQuantity  int    `json:"newQuantity"`
		NewVersion   int    `json:"newVersion"`
		ErrorType    string `json:"errorType"`
		ErrorMessage string `json:"errorMessage"`
		LastUpdated  string `json:"lastUpdated"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &updateResp); err != nil {
		slog.Debug("Failed to parse JSON as UpdateResponse", "error", err, "json", jsonStr)
		return nil
	}

	if updateResp.ErrorType == "" {
		slog.Debug("No errorType found in parsed response")
		return nil
	}

	statusCode := http.StatusInternalServerError

	// Map error types to proper HTTP status codes
	switch updateResp.ErrorType {
	case "version_conflict":
		statusCode = http.StatusConflict
	case "insufficient_inventory":
		statusCode = http.StatusBadRequest
	case "product_not_found":
		statusCode = http.StatusNotFound
	case "invalid_request", "invalid_delta", "missing_product_id":
		statusCode = http.StatusBadRequest
	default:
		statusCode = http.StatusInternalServerError
	}

	slog.Info("Successfully parsed central API error",
		"error_type", updateResp.ErrorType,
		"status_code", statusCode,
		"product_id", updateResp.ProductID,
		"new_version", updateResp.NewVersion)

	return &StandardizedError{
		ErrorType:    updateResp.ErrorType,
		ErrorMessage: updateResp.ErrorMessage,
		ProductID:    updateResp.ProductID,
		NewVersion:   updateResp.NewVersion,
		NewQuantity:  updateResp.NewQuantity,
		LastUpdated:  updateResp.LastUpdated,
		StatusCode:   statusCode,
	}
}

// writeStandardizedErrorResponse writes error response in the format expected by frontend
func (h *InventoryHandler) writeStandardizedErrorResponse(w http.ResponseWriter, stdErr *StandardizedError, productID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stdErr.StatusCode)

	// Create response in the format expected by frontend
	response := map[string]interface{}{
		"productId":    productID,
		"errorType":    stdErr.ErrorType,
		"errorMessage": stdErr.ErrorMessage,
	}

	// Include additional fields if available
	if stdErr.NewVersion > 0 {
		response["newVersion"] = stdErr.NewVersion
	}
	if stdErr.NewQuantity >= 0 {
		response["newQuantity"] = stdErr.NewQuantity
	}
	if stdErr.LastUpdated != "" {
		response["lastUpdated"] = stdErr.LastUpdated
	}

	json.NewEncoder(w).Encode(response)

	slog.Info("Returned standardized error response",
		"product_id", productID,
		"error_type", stdErr.ErrorType,
		"status_code", stdErr.StatusCode,
		"has_version", stdErr.NewVersion > 0,
		"has_quantity", stdErr.NewQuantity >= 0)
}

// writeErrorResponse writes a structured JSON error response (legacy format)
func (h *InventoryHandler) writeErrorResponse(w http.ResponseWriter, code, message string, statusCode int, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// writeSimpleErrorResponse writes a simple error response (for backward compatibility)
func (h *InventoryHandler) writeSimpleErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	h.writeErrorResponse(w, "internal_error", message, statusCode, nil)
}

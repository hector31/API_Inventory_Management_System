package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
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

		// Parse error type for better error handling
		errorCode := "update_failed"
		details := map[string]interface{}{
			"productId": updateReq.ProductID,
			"error":     err.Error(),
		}

		// Check for specific error types
		if err.Error() == "version conflict" {
			errorCode = "version_conflict"
			details["expectedVersion"] = updateReq.Version
		} else if err.Error() == "insufficient inventory" {
			errorCode = "insufficient_inventory"
		}

		h.writeErrorResponse(w, errorCode, "Failed to update inventory", http.StatusInternalServerError, details)
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

// writeErrorResponse writes a structured JSON error response
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

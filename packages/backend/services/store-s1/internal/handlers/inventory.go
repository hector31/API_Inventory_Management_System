package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

// GetAllProducts handles GET /v1/store/inventory (now using local cache)
func (h *InventoryHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	slog.Info("Getting all products for store from local cache", "remote_addr", r.RemoteAddr)

	products, err := h.localStorage.GetAllProducts()
	if err != nil {
		slog.Error("Failed to get products from local storage", "error", err)
		h.writeErrorResponse(w, "Failed to retrieve products", http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully retrieved products from local cache", "count", len(products))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}

// GetProduct handles GET /v1/store/inventory/{productId} (now using local cache)
func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")

	slog.Info("Getting product for store from local cache", "product_id", productID, "remote_addr", r.RemoteAddr)

	product, err := h.localStorage.GetProduct(productID)
	if err != nil {
		slog.Error("Failed to get product from local storage", "product_id", productID, "error", err)

		if err.Error() == fmt.Sprintf("product not found: %s", productID) {
			h.writeErrorResponse(w, "Product not found", http.StatusNotFound)
			return
		}

		h.writeErrorResponse(w, "Failed to retrieve product", http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully retrieved product from local cache", "product_id", productID, "available", product.Available, "version", product.Version)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}

// UpdateInventory handles POST /v1/store/inventory/updates
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	var updateReq models.UpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		slog.Error("Failed to decode update request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
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
		h.writeErrorResponse(w, "Failed to update inventory", http.StatusInternalServerError)
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
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
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
		h.writeErrorResponse(w, "Failed to batch update inventory", http.StatusInternalServerError)
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
		h.writeErrorResponse(w, "Force sync failed", http.StatusInternalServerError)
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
		h.writeErrorResponse(w, "Failed to get cache stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// writeErrorResponse writes an error response in JSON format
func (h *InventoryHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{
		Error: message,
	}

	json.NewEncoder(w).Encode(errorResp)
}

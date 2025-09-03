package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/melibackend/shared/client"
	"github.com/melibackend/shared/models"
)

// InventoryHandler handles inventory-related requests
type InventoryHandler struct {
	logger          *slog.Logger
	inventoryClient *client.InventoryClient
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(logger *slog.Logger, inventoryClient *client.InventoryClient) *InventoryHandler {
	return &InventoryHandler{
		logger:          logger,
		inventoryClient: inventoryClient,
	}
}

// GetAllProducts handles GET /v1/store/inventory
func (h *InventoryHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Getting all products for store", "remote_addr", r.RemoteAddr)

	products, err := h.inventoryClient.GetAllProducts()
	if err != nil {
		h.logger.Error("Failed to get products from central API", "error", err)
		h.writeErrorResponse(w, "Failed to retrieve products", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully retrieved products", "count", len(products))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}

// GetProduct handles GET /v1/store/inventory/{productId}
func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	
	h.logger.Info("Getting product for store", "product_id", productID, "remote_addr", r.RemoteAddr)

	product, err := h.inventoryClient.GetProduct(productID)
	if err != nil {
		h.logger.Error("Failed to get product from central API", "product_id", productID, "error", err)
		
		if err.Error() == fmt.Sprintf("product not found: %s", productID) {
			h.writeErrorResponse(w, "Product not found", http.StatusNotFound)
			return
		}
		
		h.writeErrorResponse(w, "Failed to retrieve product", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully retrieved product", "product_id", productID, "available", product.Available, "version", product.Version)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}

// UpdateInventory handles POST /v1/store/inventory/updates
func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	var updateReq models.UpdateRequest
	
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		h.logger.Error("Failed to decode update request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Processing inventory update for store", 
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
		h.logger.Error("Failed to update inventory via central API", 
			"product_id", updateReq.ProductID,
			"error", err,
		)
		h.writeErrorResponse(w, "Failed to update inventory", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully updated inventory", 
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
		h.logger.Error("Failed to decode batch update request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Processing batch inventory update for store", 
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
		h.logger.Error("Failed to batch update inventory via central API", 
			"store_id", batchReq.StoreID,
			"error", err,
		)
		h.writeErrorResponse(w, "Failed to batch update inventory", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully processed batch inventory update", 
		"store_id", batchReq.StoreID,
		"total_count", batchResp.TotalCount,
		"success_count", batchResp.SuccessCount,
		"failure_count", batchResp.FailureCount,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(batchResp)
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

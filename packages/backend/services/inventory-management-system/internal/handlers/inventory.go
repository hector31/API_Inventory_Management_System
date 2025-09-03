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
			ProductID: req.ProductID,
			Applied:   false,
		}
	}

	if req.IdempotencyKey == "" {
		slog.Warn("Missing idempotency key in single update", "product_id", req.ProductID)
		return models.UpdateResponse{
			ProductID: req.ProductID,
			Applied:   false,
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
			ProductID:   req.ProductID,
			Applied:     false,
			NewQuantity: 0,
			NewVersion:  0,
			LastUpdated: "",
		}
	}

	response := models.UpdateResponse{
		ProductID:   req.ProductID,
		NewQuantity: result.NewQuantity,
		NewVersion:  result.NewVersion,
		Applied:     result.Applied,
		LastUpdated: result.LastUpdated,
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
			"error", result.Error,
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
				ProductID: update.ProductID,
				Applied:   false,
				Error:     "Missing product ID",
			})
			failed++
			continue
		}

		if update.IdempotencyKey == "" {
			slog.Warn("Missing idempotency key in batch update item", "product_id", update.ProductID)
			results = append(results, models.ProductUpdateResult{
				ProductID: update.ProductID,
				Applied:   false,
				Error:     "Missing idempotency key",
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
				ProductID: update.ProductID,
				Applied:   false,
				Error:     err.Error(),
			}
			failed++
		} else if !serviceResult.Success {
			result = models.ProductUpdateResult{
				ProductID:   update.ProductID,
				NewQuantity: serviceResult.NewQuantity,
				NewVersion:  serviceResult.NewVersion,
				Applied:     false,
				LastUpdated: serviceResult.LastUpdated,
				Error:       serviceResult.Error.Error(),
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

// ListProducts handles GET /v1/inventory - List products with cursor pagination and replication support
func (h *InventoryHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	snapshot := r.URL.Query().Get("snapshot") == "true"
	sinceStr := r.URL.Query().Get("since")
	format := r.URL.Query().Get("format")

	// Parse the limit (default 50)
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Parse since offset for replication
	var sinceOffset int
	if sinceStr != "" {
		if parsedOffset, err := strconv.Atoi(sinceStr); err == nil && parsedOffset >= 0 {
			sinceOffset = parsedOffset
		}
	}

	slog.Debug("Listing products with parameters",
		"cursor", cursor,
		"limit", limit,
		"snapshot", snapshot,
		"since", sinceOffset,
		"format", format,
		"remote_addr", r.RemoteAddr)

	// Handle replication snapshot request
	if snapshot {
		h.handleSnapshotRequest(w, r, limit)
		return
	}

	// Handle replication changes request
	if sinceStr != "" {
		h.handleChangesRequest(w, r, sinceOffset, limit)
		return
	}

	// Handle regular product listing
	h.handleRegularListing(w, r, cursor, limit, format)
}

// handleSnapshotRequest handles ?snapshot=true requests (replaces /replication/snapshot)
func (h *InventoryHandler) handleSnapshotRequest(w http.ResponseWriter, r *http.Request, limit int) {
	slog.Info("Processing snapshot request", "limit", limit, "remote_addr", r.RemoteAddr)

	// Get all products for snapshot
	productList, err := h.inventoryService.ListProducts("", limit)
	if err != nil {
		slog.Error("Failed to retrieve snapshot", "error", err, "remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Error retrieving snapshot", nil)
		return
	}

	// Get system metadata for last offset
	metadata := h.inventoryService.GetSystemMetadata()

	// Create snapshot response
	snapshotResponse := map[string]interface{}{
		"state":      make(map[string]models.ProductResponse),
		"lastOffset": metadata.LastOffset,
		"timestamp":  metadata.LastUpdated,
		"total":      len(productList.Items),
	}

	// Convert items to state map
	stateMap := make(map[string]models.ProductResponse)
	for _, product := range productList.Items {
		stateMap[product.ProductID] = product
	}
	snapshotResponse["state"] = stateMap

	slog.Info("Snapshot generated successfully",
		"products_count", len(productList.Items),
		"last_offset", metadata.LastOffset,
		"remote_addr", r.RemoteAddr)

	writeJSONResponse(w, http.StatusOK, snapshotResponse)
}

// handleChangesRequest handles ?since=<offset> requests (replaces /replication/changes)
func (h *InventoryHandler) handleChangesRequest(w http.ResponseWriter, r *http.Request, sinceOffset, limit int) {
	longPollStr := r.URL.Query().Get("longPollSeconds")

	slog.Info("Processing changes request",
		"since_offset", sinceOffset,
		"limit", limit,
		"long_poll", longPollStr,
		"remote_addr", r.RemoteAddr)

	// Get system metadata for current offset
	metadata := h.inventoryService.GetSystemMetadata()

	// Create changes response (placeholder implementation)
	changesResponse := map[string]interface{}{
		"events":     []map[string]interface{}{}, // Empty for now - would contain actual change events
		"nextOffset": metadata.LastOffset,
		"hasMore":    false,
		"timestamp":  metadata.LastUpdated,
	}

	// If there are changes since the requested offset, we would populate events here
	if sinceOffset < metadata.LastOffset {
		// Placeholder event - in real implementation, this would come from event store
		events := []map[string]interface{}{
			{
				"seq":        metadata.LastOffset,
				"type":       "StockChanged",
				"productId":  "SKU-001",
				"storeId":    "store-1",
				"delta":      1,
				"newVersion": 13,
				"timestamp":  metadata.LastUpdated,
			},
		}
		changesResponse["events"] = events
	}

	slog.Info("Changes response generated",
		"since_offset", sinceOffset,
		"current_offset", metadata.LastOffset,
		"events_count", len(changesResponse["events"].([]map[string]interface{})),
		"remote_addr", r.RemoteAddr)

	writeJSONResponse(w, http.StatusOK, changesResponse)
}

// handleRegularListing handles standard product listing requests
func (h *InventoryHandler) handleRegularListing(w http.ResponseWriter, r *http.Request, cursor string, limit int, format string) {
	// Get the product list from the service
	productList, err := h.inventoryService.ListProducts(cursor, limit)
	if err != nil {
		slog.Error("Failed to retrieve products", "error", err, "remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Error retrieving products", nil)
		return
	}

	// Handle replication format
	if format == "replication" {
		// Return in replication-friendly format with metadata
		metadata := h.inventoryService.GetSystemMetadata()
		replicationResponse := map[string]interface{}{
			"items":      productList.Items,
			"nextCursor": productList.NextCursor,
			"metadata": map[string]interface{}{
				"lastOffset":    metadata.LastOffset,
				"totalProducts": metadata.TotalProducts,
				"lastUpdated":   metadata.LastUpdated,
			},
		}

		slog.Info("Products listed in replication format",
			"cursor", cursor,
			"limit", limit,
			"found_count", len(productList.Items),
			"last_offset", metadata.LastOffset,
			"remote_addr", r.RemoteAddr)

		writeJSONResponse(w, http.StatusOK, replicationResponse)
		return
	}

	// Standard response format with event offset for synchronization
	metadata := h.inventoryService.GetSystemMetadata()
	standardResponse := map[string]interface{}{
		"items":       productList.Items,
		"nextCursor":  productList.NextCursor,
		"eventOffset": metadata.LastOffset, // Current event offset for synchronization
	}

	slog.Info("Products listed successfully",
		"cursor", cursor,
		"limit", limit,
		"found_count", len(productList.Items),
		"event_offset", metadata.LastOffset,
		"remote_addr", r.RemoteAddr)

	writeJSONResponse(w, http.StatusOK, standardResponse)
}

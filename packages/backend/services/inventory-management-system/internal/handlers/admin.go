package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inventory-management-api/internal/models"
	"inventory-management-api/internal/services"
)

// AdminHandler handles admin-only endpoints
type AdminHandler struct {
	inventoryService *services.InventoryService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(inventoryService *services.InventoryService) *AdminHandler {
	return &AdminHandler{
		inventoryService: inventoryService,
	}
}

// SetProducts handles POST /api/v1/admin/products/set - Admin product update endpoint
func (h *AdminHandler) SetProducts(w http.ResponseWriter, r *http.Request) {
	slog.Info("Admin set products request received",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Parse request body
	var req models.AdminSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to parse admin set request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", nil)
		return
	}

	// Validate request
	if len(req.Products) == 0 {
		slog.Warn("Admin set request with no products",
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "No products specified", nil)
		return
	}

	// Validate individual products
	var validationErrors []models.ErrorDetail
	for i, product := range req.Products {
		if product.ProductID == "" {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].productId",
				Issue: "Product ID is required",
			})
		}

		// Check that at least one field is being updated
		if product.Name == nil && product.Available == nil && product.Price == nil {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].fields",
				Issue: "At least one field (name, available, price) must be specified",
			})
		}

		// Validate available quantity if provided
		if product.Available != nil && *product.Available < 0 {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].available",
				Issue: "Available quantity cannot be negative",
			})
		}

		// Validate price if provided
		if product.Price != nil && *product.Price < 0 {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].price",
				Issue: "Price cannot be negative",
			})
		}
	}

	if len(validationErrors) > 0 {
		slog.Warn("Admin set request validation failed",
			"validation_errors", len(validationErrors),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "validation_error", "Request validation failed", validationErrors)
		return
	}

	slog.Info("Processing admin set request",
		"product_count", len(req.Products),
		"remote_addr", r.RemoteAddr)

	// Process the admin set request
	response, err := h.inventoryService.AdminSetProducts(req.Products)
	if err != nil {
		slog.Error("Failed to process admin set request",
			"error", err,
			"product_count", len(req.Products),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to process admin set request", nil)
		return
	}

	// Log the results
	slog.Info("Admin set request completed",
		"total_requests", response.Summary.TotalRequests,
		"successful_updates", response.Summary.SuccessfulUpdates,
		"failed_updates", response.Summary.FailedUpdates,
		"remote_addr", r.RemoteAddr)

	// Return response
	writeJSONResponse(w, http.StatusOK, response)
}

// CreateProducts handles POST /api/v1/admin/products/create - Admin product creation endpoint
func (h *AdminHandler) CreateProducts(w http.ResponseWriter, r *http.Request) {
	slog.Info("Admin create products request received",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Parse request body
	var req models.AdminCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to parse admin create request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", nil)
		return
	}

	// Validate request
	if len(req.Products) == 0 {
		slog.Warn("Admin create request with no products",
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "No products specified", nil)
		return
	}

	// Validate individual products
	var validationErrors []models.ErrorDetail
	for i, product := range req.Products {
		if product.ProductID == "" {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].productId",
				Issue: "Product ID is required",
			})
		}

		if product.Name == "" {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].name",
				Issue: "Product name is required",
			})
		}

		if product.Available < 0 {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].available",
				Issue: "Available quantity cannot be negative",
			})
		}

		if product.Price < 0 {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "products[" + string(rune(i)) + "].price",
				Issue: "Price cannot be negative",
			})
		}
	}

	if len(validationErrors) > 0 {
		slog.Warn("Admin create request validation failed",
			"validation_errors", len(validationErrors),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "validation_error", "Request validation failed", validationErrors)
		return
	}

	slog.Info("Processing admin create request",
		"product_count", len(req.Products),
		"remote_addr", r.RemoteAddr)

	// Process the admin create request
	response, err := h.inventoryService.AdminCreateProducts(req.Products)
	if err != nil {
		slog.Error("Failed to process admin create request",
			"error", err,
			"product_count", len(req.Products),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to process admin create request", nil)
		return
	}

	// Log the results
	slog.Info("Admin create request completed",
		"total_requests", response.Summary.TotalRequests,
		"successful_creations", response.Summary.SuccessfulCreations,
		"failed_creations", response.Summary.FailedCreations,
		"remote_addr", r.RemoteAddr)

	// Return response
	writeJSONResponse(w, http.StatusOK, response)
}

// DeleteProducts handles DELETE /api/v1/admin/products/delete - Admin product deletion endpoint
func (h *AdminHandler) DeleteProducts(w http.ResponseWriter, r *http.Request) {
	slog.Info("Admin delete products request received",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Parse request body
	var req models.AdminDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to parse admin delete request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", nil)
		return
	}

	// Validate request
	if len(req.ProductIDs) == 0 {
		slog.Warn("Admin delete request with no product IDs",
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "No product IDs specified", nil)
		return
	}

	// Validate individual product IDs
	var validationErrors []models.ErrorDetail
	for i, productID := range req.ProductIDs {
		if productID == "" {
			validationErrors = append(validationErrors, models.ErrorDetail{
				Field: "productIds[" + string(rune(i)) + "]",
				Issue: "Product ID cannot be empty",
			})
		}
	}

	if len(validationErrors) > 0 {
		slog.Warn("Admin delete request validation failed",
			"validation_errors", len(validationErrors),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "validation_error", "Request validation failed", validationErrors)
		return
	}

	slog.Info("Processing admin delete request",
		"product_count", len(req.ProductIDs),
		"remote_addr", r.RemoteAddr)

	// Process the admin delete request
	response, err := h.inventoryService.AdminDeleteProducts(req.ProductIDs)
	if err != nil {
		slog.Error("Failed to process admin delete request",
			"error", err,
			"product_count", len(req.ProductIDs),
			"remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to process admin delete request", nil)
		return
	}

	// Log the results
	slog.Info("Admin delete request completed",
		"total_requests", response.Summary.TotalRequests,
		"successful_deletions", response.Summary.SuccessfulDeletions,
		"failed_deletions", response.Summary.FailedDeletions,
		"remote_addr", r.RemoteAddr)

	// Return response
	writeJSONResponse(w, http.StatusOK, response)
}

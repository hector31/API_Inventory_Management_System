package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/melibackend/shared/client"
	"github.com/melibackend/shared/models"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	logger          *slog.Logger
	inventoryClient *client.InventoryClient
	serviceName     string
	version         string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger *slog.Logger, inventoryClient *client.InventoryClient, serviceName, version string) *HealthHandler {
	return &HealthHandler{
		logger:          logger,
		inventoryClient: inventoryClient,
		serviceName:     serviceName,
		version:         version,
	}
}

// HealthCheck handles GET /health
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Health check requested", "remote_addr", r.RemoteAddr)

	// Check central API health
	centralHealth, err := h.inventoryClient.HealthCheck()
	if err != nil {
		h.logger.Error("Central API health check failed", "error", err)
		
		response := models.HealthResponse{
			Status:    "unhealthy",
			Service:   h.serviceName,
			Version:   h.version,
			Timestamp: time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	h.logger.Debug("Central API health check successful", "central_status", centralHealth.Status)

	response := models.HealthResponse{
		Status:    "healthy",
		Service:   h.serviceName,
		Version:   h.version,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

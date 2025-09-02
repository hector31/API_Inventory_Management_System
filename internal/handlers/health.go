package handlers

import (
	"net/http"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET /health - Health check endpoint
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

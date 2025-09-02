package middleware

import (
	"encoding/json"
	"net/http"

	"inventory-management-api/internal/models"
)

// AuthMiddleware provides API key authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "API key required", nil)
			return
		}
		// For now, accept any API key - in real implementation, validate against allowed keys
		next.ServeHTTP(w, r)
	})
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

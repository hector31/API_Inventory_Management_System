package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"inventory-management-api/internal/models"
)

// AuthMiddleware provides API key authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			slog.Warn("Authentication failed: missing API key", "remote_addr", r.RemoteAddr)
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "API key required", nil)
			return
		}

		// Validate API key against environment variable
		if !isValidAPIKey(apiKey) {
			slog.Warn("Authentication failed: invalid API key", "remote_addr", r.RemoteAddr, "provided_key", apiKey)
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Invalid API key", nil)
			return
		}

		slog.Debug("Authentication successful", "remote_addr", r.RemoteAddr, "api_key", apiKey)
		next.ServeHTTP(w, r)
	})
}

// isValidAPIKey checks if the provided API key is valid
func isValidAPIKey(apiKey string) bool {
	// Get valid API keys from environment variable
	apiKeysStr := os.Getenv("API_KEYS")
	if apiKeysStr == "" {
		apiKeysStr = "demo" // Default fallback
	}

	validKeys := strings.Split(apiKeysStr, ",")
	for _, validKey := range validKeys {
		if strings.TrimSpace(validKey) == apiKey {
			return true
		}
	}
	return false
}

// AdminAuthMiddleware provides admin-only API key authentication
func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			slog.Warn("Admin authentication failed: missing API key", "remote_addr", r.RemoteAddr)
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Admin API key required", nil)
			return
		}

		// Validate API key against admin keys
		if !isValidAdminAPIKey(apiKey) {
			slog.Warn("Admin authentication failed: invalid admin API key", "remote_addr", r.RemoteAddr, "provided_key", apiKey)
			writeErrorResponse(w, http.StatusForbidden, "forbidden", "Admin access required", nil)
			return
		}

		slog.Debug("Admin authentication successful", "remote_addr", r.RemoteAddr, "api_key", apiKey)
		next.ServeHTTP(w, r)
	})
}

// isValidAdminAPIKey checks if the provided API key has admin privileges
func isValidAdminAPIKey(apiKey string) bool {
	// Get admin API keys from environment variable
	adminKeysStr := os.Getenv("ADMIN_API_KEYS")
	if adminKeysStr == "" {
		// Fallback: check if it's in the regular API keys and has admin prefix
		if strings.HasPrefix(apiKey, "admin-") {
			return isValidAPIKey(apiKey)
		}
		return false
	}

	validAdminKeys := strings.Split(adminKeysStr, ",")
	for _, validKey := range validAdminKeys {
		if strings.TrimSpace(validKey) == apiKey {
			return true
		}
	}
	return false
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

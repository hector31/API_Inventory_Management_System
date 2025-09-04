package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/melibackend/shared/models"
)

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(validAPIKeys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")

			if apiKey == "" {
				slog.Warn("Missing API key", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
				writeErrorResponse(w, "Missing API key", http.StatusUnauthorized)
				return
			}

			// Check if the API key is valid
			valid := false
			for _, validKey := range validAPIKeys {
				if apiKey == validKey {
					valid = true
					break
				}
			}

			if !valid {
				slog.Warn("Invalid API key", "remote_addr", r.RemoteAddr, "api_key", maskAPIKey(apiKey), "path", r.URL.Path)
				writeErrorResponse(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			slog.Debug("Authentication successful", "remote_addr", r.RemoteAddr, "api_key", maskAPIKey(apiKey))
			next.ServeHTTP(w, r)
		})
	}
}

// writeErrorResponse writes an error response in JSON format
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{
		Error: message,
	}

	json.NewEncoder(w).Encode(errorResp)
}

// maskAPIKey masks an API key for logging (shows only first 4 characters)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 4 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-4)
}

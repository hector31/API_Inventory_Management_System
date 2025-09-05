package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"inventory-management-api/internal/middleware"
)

// RateLimitStatusHandler handles rate limiting status requests
type RateLimitStatusHandler struct {
	rateLimiter *middleware.RateLimiter
}

// NewRateLimitStatusHandler creates a new rate limit status handler
func NewRateLimitStatusHandler(rateLimiter *middleware.RateLimiter) *RateLimitStatusHandler {
	return &RateLimitStatusHandler{
		rateLimiter: rateLimiter,
	}
}

// GetRateLimitStatus returns current rate limiting statistics
func (h *RateLimitStatusHandler) GetRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Getting rate limit status", "remote_addr", r.RemoteAddr)

	if h.rateLimiter == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "rate_limiter_unavailable", "Rate limiter not available", nil)
		return
	}

	stats := h.rateLimiter.GetRateLimitStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		slog.Error("Failed to encode rate limit status response", "error", err)
		writeErrorResponse(w, http.StatusInternalServerError, "encoding_error", "Failed to encode response", nil)
		return
	}

	slog.Debug("Rate limit status retrieved successfully", "active_ip_limits", stats["active_ip_limits"])
}

// ResetRateLimits resets all rate limiting counters (admin only)
func (h *RateLimitStatusHandler) ResetRateLimits(w http.ResponseWriter, r *http.Request) {
	slog.Info("Resetting rate limits", "remote_addr", r.RemoteAddr)

	if h.rateLimiter == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "rate_limiter_unavailable", "Rate limiter not available", nil)
		return
	}

	h.rateLimiter.ResetRateLimits()

	response := map[string]interface{}{
		"message": "Rate limits reset successfully",
		"timestamp": "2024-01-01T00:00:00Z", // This would be actual timestamp in real implementation
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode rate limit reset response", "error", err)
		writeErrorResponse(w, http.StatusInternalServerError, "encoding_error", "Failed to encode response", nil)
		return
	}

	slog.Info("Rate limits reset successfully")
}

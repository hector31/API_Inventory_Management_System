package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"inventory-management-api/internal/events"
	"inventory-management-api/internal/models"
)

// EventsHandler handles event streaming requests
type EventsHandler struct {
	eventQueue *events.EventQueue
	logger     *slog.Logger
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(eventQueue *events.EventQueue, logger *slog.Logger) *EventsHandler {
	return &EventsHandler{
		eventQueue: eventQueue,
		logger:     logger,
	}
}

// GetEvents handles GET /v1/inventory/events
func (h *EventsHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		h.writeErrorResponse(w, "offset parameter is required", http.StatusBadRequest)
		return
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		h.writeErrorResponse(w, "invalid offset parameter", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	waitStr := r.URL.Query().Get("wait")
	waitSeconds := 0 // default
	if waitStr != "" {
		if parsedWait, err := strconv.Atoi(waitStr); err == nil && parsedWait >= 0 && parsedWait <= 60 {
			waitSeconds = parsedWait
		}
	}

	h.logger.Info("Events request received",
		"offset", offset,
		"limit", limit,
		"wait", waitSeconds,
		"remote_addr", r.RemoteAddr,
	)

	// Try to get events immediately
	events, nextOffset, hasMore := h.eventQueue.GetEvents(offset, limit)

	// If no events and wait > 0, use long polling
	if len(events) == 0 && waitSeconds > 0 {
		h.logger.Debug("No events available, starting long polling",
			"offset", offset,
			"wait_seconds", waitSeconds,
		)

		// Wait for new events or timeout
		waitChan := h.eventQueue.WaitForEvents(offset, time.Duration(waitSeconds)*time.Second)

		select {
		case <-waitChan:
			// New events might be available, try again
			events, nextOffset, hasMore = h.eventQueue.GetEvents(offset, limit)
			h.logger.Debug("Long polling completed with events",
				"offset", offset,
				"events_count", len(events),
			)
		case <-r.Context().Done():
			// Client disconnected
			h.logger.Debug("Client disconnected during long polling",
				"offset", offset,
			)
			return
		}
	}

	// Prepare response
	response := models.EventsResponse{
		Events:     events,
		NextOffset: nextOffset,
		HasMore:    hasMore,
		Count:      len(events),
	}

	h.logger.Info("Events response sent",
		"offset", offset,
		"events_count", len(events),
		"next_offset", nextOffset,
		"has_more", hasMore,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response in JSON format
func (h *EventsHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{
		Code:    "EVENTS_ERROR",
		Message: message,
	}

	json.NewEncoder(w).Encode(errorResp)
}

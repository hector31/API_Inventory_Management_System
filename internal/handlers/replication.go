package handlers

import (
	"fmt"
	"net/http"

	"inventory-management-api/internal/models"
)

// ReplicationHandler handles replication-related HTTP requests
type ReplicationHandler struct{}

// NewReplicationHandler creates a new replication handler
func NewReplicationHandler() *ReplicationHandler {
	return &ReplicationHandler{}
}

// GetSnapshot handles GET /replication/snapshot - Bootstrap full state
func (h *ReplicationHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	// Placeholder response
	response := models.ReplicationSnapshot{
		State: map[string]models.ProductResponse{
			"SKU-1": {ProductID: "SKU-1", Available: 20, Version: 3, LastUpdated: "2025-09-02T10:00:00Z"},
			"SKU-2": {ProductID: "SKU-2", Available: 5, Version: 1, LastUpdated: "2025-09-02T09:30:00Z"},
		},
		LastOffset: 1287,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// GetChanges handles GET /replication/changes - Get ordered deltas for long-polling
func (h *ReplicationHandler) GetChanges(w http.ResponseWriter, r *http.Request) {
	fromOffset := r.URL.Query().Get("fromOffset")
	limit := r.URL.Query().Get("limit")
	longPollSeconds := r.URL.Query().Get("longPollSeconds")

	// Placeholder response
	response := models.ReplicationChanges{
		Events: []models.ReplicationEvent{
			{
				Seq:        1288,
				Type:       "StockDecreased",
				ProductID:  "SKU-2",
				StoreID:    "store-7",
				Delta:      -1,
				NewVersion: 2,
				Timestamp:  "2025-09-02T10:00:00Z",
			},
		},
		NextOffset: 1288,
		HasMore:    false,
	}

	fmt.Printf("Getting replication changes from offset: %s, limit: %s, longPoll: %s\n", fromOffset, limit, longPollSeconds)
	writeJSONResponse(w, http.StatusOK, response)
}

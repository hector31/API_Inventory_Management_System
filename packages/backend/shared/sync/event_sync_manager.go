package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/melibackend/shared/client"
	"github.com/melibackend/shared/models"
	"github.com/melibackend/shared/storage"
)

// EventSyncManager handles event-driven synchronization between central API and local storage
type EventSyncManager struct {
	client                  *client.InventoryClient
	localStorage            storage.LocalStorage
	syncIntervalSeconds     int
	eventWaitTimeoutSeconds int
	eventBatchLimit         int
	syncMutex               sync.Mutex
	stopChan                chan struct{}
	status                  *storage.SyncStatus
	statusMutex             sync.RWMutex

	// Circuit breaker for fallback to full sync
	consecutiveFailures    int
	maxConsecutiveFailures int
	fallbackMode           bool
}

// EventSyncConfig holds configuration for the event sync manager
type EventSyncConfig struct {
	SyncIntervalSeconds     int
	EventWaitTimeoutSeconds int
	EventBatchLimit         int
	MaxConsecutiveFailures  int
}

// NewEventSyncManager creates a new event-driven sync manager
func NewEventSyncManager(client *client.InventoryClient, localStorage storage.LocalStorage, config EventSyncConfig) *EventSyncManager {
	return &EventSyncManager{
		client:                  client,
		localStorage:            localStorage,
		syncIntervalSeconds:     config.SyncIntervalSeconds,
		eventWaitTimeoutSeconds: config.EventWaitTimeoutSeconds,
		eventBatchLimit:         config.EventBatchLimit,
		stopChan:                make(chan struct{}),
		status: &storage.SyncStatus{
			InProgress:      false,
			LastSyncSuccess: false,
			ProductCount:    0,
			ErrorMessage:    "",
		},
		maxConsecutiveFailures: config.MaxConsecutiveFailures,
	}
}

// Start begins the event-driven sync manager
func (m *EventSyncManager) Start(ctx context.Context) error {
	slog.Info("Starting event-driven sync manager")

	// Check if we need initial setup (first time startup)
	lastOffset, err := m.localStorage.GetLastEventOffset()
	if err != nil {
		return fmt.Errorf("failed to get last event offset: %w", err)
	}

	if lastOffset == 0 {
		// First time startup - perform initial full sync
		if err := m.InitialSync(ctx); err != nil {
			return fmt.Errorf("initial sync failed: %w", err)
		}
	} else {
		slog.Info("Resuming event sync from offset", "offset", lastOffset)
	}

	// Start periodic event polling in background
	go m.eventPollingLoop(ctx)

	return nil
}

// Stop stops the sync manager
func (m *EventSyncManager) Stop() {
	slog.Info("Stopping event-driven sync manager")
	close(m.stopChan)
}

// InitialSync performs a complete synchronization and sets up event tracking
func (m *EventSyncManager) InitialSync(ctx context.Context) error {
	m.syncMutex.Lock()
	defer m.syncMutex.Unlock()

	slog.Info("Starting initial sync with event offset setup")
	startTime := time.Now()

	m.updateSyncStatus(true, false, 0, "", time.Time{})

	// Get all products with metadata including current event offset
	products, eventOffset, err := m.client.GetAllProductsWithMetadata()
	if err != nil {
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to get products from central API: %w", err)
	}

	slog.Info("Retrieved products from central API",
		"count", len(products),
		"event_offset", eventOffset)

	// Sync all products to local storage
	if err := m.localStorage.SyncAllProducts(products); err != nil {
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to sync products to local storage: %w", err)
	}

	// Set the event offset as our starting point for future event polling
	if err := m.localStorage.SetLastEventOffset(eventOffset); err != nil {
		slog.Warn("Failed to set initial event offset", "error", err, "offset", eventOffset)
	}

	// Update sync time
	syncTime := time.Now()
	if err := m.localStorage.SetLastSyncTime(syncTime); err != nil {
		slog.Warn("Failed to update last sync time", "error", err)
	}

	duration := time.Since(startTime)
	m.updateSyncStatus(false, true, len(products), "", syncTime)

	slog.Info("Initial sync completed successfully",
		"products_synced", len(products),
		"event_offset", eventOffset,
		"duration", duration,
	)

	// Reset failure counters after successful sync
	m.consecutiveFailures = 0
	m.fallbackMode = false

	return nil
}

// eventPollingLoop runs the continuous event polling
func (m *EventSyncManager) eventPollingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.syncIntervalSeconds) * time.Second)
	defer ticker.Stop()

	slog.Info("Event polling loop started",
		"interval_seconds", m.syncIntervalSeconds,
		"wait_timeout_seconds", m.eventWaitTimeoutSeconds,
		"batch_limit", m.eventBatchLimit)

	tickCount := 0
	for {
		select {
		case <-ctx.Done():
			slog.Info("Event polling loop stopped due to context cancellation", "total_ticks", tickCount)
			return
		case <-m.stopChan:
			slog.Info("Event polling loop stopped", "total_ticks", tickCount)
			return
		case <-ticker.C:
			tickCount++
			slog.Debug("Ticker fired", "tick_count", tickCount)

			// Handle polling with error recovery (synchronously to avoid goroutine issues)
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Panic in polling", "tick", tickCount, "panic", r)
					}
				}()

				if err := m.pollForEvents(ctx); err != nil {
					slog.Error("Polling failed", "tick", tickCount, "error", err)
					m.handleSyncError(err)
				} else {
					slog.Debug("Polling completed successfully", "tick", tickCount)
				}
			}()
		}
	}
}

// pollForEvents polls for new events and applies them
func (m *EventSyncManager) pollForEvents(ctx context.Context) error {
	lastOffset, err := m.localStorage.GetLastEventOffset()
	if err != nil {
		return fmt.Errorf("failed to get last event offset: %w", err)
	}

	slog.Debug("Polling for events",
		"from_offset", lastOffset,
		"limit", m.eventBatchLimit,
		"wait_timeout", m.eventWaitTimeoutSeconds)

	// Get events from the central API
	eventsResponse, err := m.client.GetEvents(lastOffset, m.eventBatchLimit, m.eventWaitTimeoutSeconds)
	if err != nil {
		return m.handleEventError(err, lastOffset)
	}

	// Validate response
	if err := m.validateEventsResponse(eventsResponse, lastOffset); err != nil {
		return err
	}

	// Apply events if any
	if len(eventsResponse.Events) > 0 {
		if err := m.applyEvents(eventsResponse.Events); err != nil {
			return fmt.Errorf("failed to apply events: %w", err)
		}

		slog.Info("Applied events successfully",
			"count", len(eventsResponse.Events),
			"from_offset", lastOffset,
			"to_offset", eventsResponse.NextOffset)
	} else {
		slog.Debug("No new events available")
	}

	// Reset failure counter on successful poll
	m.consecutiveFailures = 0
	if m.fallbackMode {
		slog.Info("Exiting fallback mode after successful event sync")
		m.fallbackMode = false
	}

	return nil
}

// applyEvents applies a batch of events to local storage
func (m *EventSyncManager) applyEvents(events []models.Event) error {
	if err := m.localStorage.ApplyEvents(events); err != nil {
		return err
	}

	// Update last processed offset
	if len(events) > 0 {
		lastEvent := events[len(events)-1]
		if err := m.localStorage.SetLastEventOffset(lastEvent.Offset + 1); err != nil {
			slog.Warn("Failed to update last event offset", "error", err, "offset", lastEvent.Offset+1)
		}
	}

	return nil
}

// validateEventsResponse validates the events response for consistency
func (m *EventSyncManager) validateEventsResponse(response *models.EventsResponse, expectedOffset int64) error {
	// Check if response offset is lower than our local offset (central system reset)
	if response.NextOffset < expectedOffset {
		return fmt.Errorf("response offset (%d) is lower than expected (%d): central system may have been reset",
			response.NextOffset, expectedOffset)
	}

	// Check for large gaps (possible data loss)
	if len(response.Events) > 0 {
		firstEventOffset := response.Events[0].Offset
		if firstEventOffset > expectedOffset+int64(m.eventBatchLimit) {
			return fmt.Errorf("large gap detected: expected offset %d, got %d: possible data loss",
				expectedOffset, firstEventOffset)
		}

		// Validate event sequence
		for i, event := range response.Events {
			expectedEventOffset := firstEventOffset + int64(i)
			if event.Offset != expectedEventOffset {
				return fmt.Errorf("event sequence gap: expected offset %d, got %d",
					expectedEventOffset, event.Offset)
			}
		}
	}

	return nil
}

// handleEventError handles specific event-related errors
func (m *EventSyncManager) handleEventError(err error, lastOffset int64) error {
	errorMsg := err.Error()

	// Handle 410 Gone - offset not found
	if contains(errorMsg, "410 Gone") || contains(errorMsg, "offset not found") {
		slog.Warn("Offset not found, central system may have restarted",
			"last_offset", lastOffset, "error", err)
		return m.triggerFullSyncFallback("offset_not_found")
	}

	// Handle other specific errors that should trigger fallback
	if contains(errorMsg, "central system may have been reset") ||
		contains(errorMsg, "possible data loss") ||
		contains(errorMsg, "event sequence gap") {
		slog.Warn("Data consistency issue detected", "error", err)
		return m.triggerFullSyncFallback("data_consistency_issue")
	}

	// For other errors, just return them to be handled by the general error handler
	return err
}

// triggerFullSyncFallback triggers a full sync as fallback
func (m *EventSyncManager) triggerFullSyncFallback(reason string) error {
	slog.Warn("Triggering full sync fallback", "reason", reason)

	ctx := context.Background()
	if err := m.InitialSync(ctx); err != nil {
		return fmt.Errorf("fallback full sync failed: %w", err)
	}

	m.fallbackMode = true

	return nil
}

// handleSyncError handles general sync errors with circuit breaker logic
func (m *EventSyncManager) handleSyncError(err error) {
	m.consecutiveFailures++
	slog.Error("Event sync failed",
		"error", err,
		"consecutive_failures", m.consecutiveFailures,
		"max_failures", m.maxConsecutiveFailures)

	// Update sync status
	m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})

	// Check if we should enter fallback mode
	if m.consecutiveFailures >= m.maxConsecutiveFailures && !m.fallbackMode {
		slog.Warn("Too many consecutive failures, entering fallback mode",
			"failures", m.consecutiveFailures)

		// Try full sync as fallback in a separate goroutine to avoid blocking
		go func() {
			ctx := context.Background()
			if fallbackErr := m.InitialSync(ctx); fallbackErr != nil {
				slog.Error("Fallback full sync also failed", "error", fallbackErr)
			} else {
				m.fallbackMode = true
				slog.Info("Fallback full sync completed successfully")
			}
		}()
	}
}

// updateSyncStatus updates the internal sync status
func (m *EventSyncManager) updateSyncStatus(inProgress, success bool, productCount int, errorMessage string, syncTime time.Time) {
	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()

	m.status.InProgress = inProgress
	m.status.LastSyncSuccess = success
	m.status.ProductCount = productCount
	m.status.ErrorMessage = errorMessage
	if !syncTime.IsZero() {
		m.status.LastSyncTime = syncTime
	}
}

// GetSyncStatus returns the current sync status
func (m *EventSyncManager) GetSyncStatus() *storage.SyncStatus {
	m.statusMutex.RLock()
	defer m.statusMutex.RUnlock()

	// Return a copy to avoid race conditions
	status := *m.status
	return &status
}

// ForceSync forces an immediate full synchronization
func (m *EventSyncManager) ForceSync(ctx context.Context) error {
	slog.Info("Force sync requested")
	return m.InitialSync(ctx)
}

// UpdateLocalProduct updates a single product in local storage after a successful write operation
func (m *EventSyncManager) UpdateLocalProduct(productID string, available int, version int, lastUpdated time.Time) error {
	slog.Debug("Updating local product",
		"product_id", productID,
		"available", available,
		"version", version,
	)

	return m.localStorage.UpdateProduct(productID, available, version, lastUpdated)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

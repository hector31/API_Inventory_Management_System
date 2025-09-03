package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"inventory-management-api/internal/models"
)

// EventQueue manages the event queue with file persistence
type EventQueue struct {
	mu           sync.RWMutex
	events       []models.Event
	nextOffset   int64
	filePath     string
	maxEvents    int
	logger       *slog.Logger
	writeChan    chan models.Event
	stopChan     chan struct{}
	waiters      map[int64][]chan struct{}
	waitersMutex sync.RWMutex
}

// EventQueueConfig holds configuration for the event queue
type EventQueueConfig struct {
	FilePath  string
	MaxEvents int
	Logger    *slog.Logger
}

// NewEventQueue creates a new event queue
func NewEventQueue(config EventQueueConfig) (*EventQueue, error) {
	eq := &EventQueue{
		events:    make([]models.Event, 0),
		filePath:  config.FilePath,
		maxEvents: config.MaxEvents,
		logger:    config.Logger,
		writeChan: make(chan models.Event, 1000), // Buffer for async writes
		stopChan:  make(chan struct{}),
		waiters:   make(map[int64][]chan struct{}),
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create events directory: %w", err)
	}

	// Load existing events from file
	if err := eq.loadFromFile(); err != nil {
		eq.logger.Warn("Failed to load events from file, starting fresh", "error", err)
		eq.nextOffset = 0
	}

	// Start async writer goroutine
	go eq.asyncWriter()

	eq.logger.Info("Event queue initialized",
		"file_path", config.FilePath,
		"max_events", config.MaxEvents,
		"loaded_events", len(eq.events),
		"next_offset", eq.nextOffset,
	)

	return eq, nil
}

// PublishEvent adds a new event to the queue
func (eq *EventQueue) PublishEvent(eventType, productID string, data models.ProductResponse, version int) {
	event := models.Event{
		Offset:    eq.getNextOffset(),
		Timestamp: time.Now().Format(time.RFC3339),
		EventType: eventType,
		ProductID: productID,
		Data:      data,
		Version:   version,
	}

	// Send to async writer (non-blocking)
	select {
	case eq.writeChan <- event:
		eq.logger.Debug("Event queued for writing",
			"offset", event.Offset,
			"event_type", event.EventType,
			"product_id", event.ProductID,
		)
	default:
		eq.logger.Error("Event write channel full, dropping event",
			"offset", event.Offset,
			"event_type", event.EventType,
			"product_id", event.ProductID,
		)
	}
}

// GetEvents retrieves events starting from the given offset
func (eq *EventQueue) GetEvents(fromOffset int64, limit int) ([]models.Event, int64, bool) {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	var result []models.Event
	hasMore := false

	// Find starting index
	startIdx := -1
	for i, event := range eq.events {
		if event.Offset >= fromOffset {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		// No events found from the requested offset
		return result, eq.nextOffset, false
	}

	// Collect events up to limit
	endIdx := startIdx + limit
	if endIdx > len(eq.events) {
		endIdx = len(eq.events)
	} else {
		hasMore = true
	}

	result = make([]models.Event, endIdx-startIdx)
	copy(result, eq.events[startIdx:endIdx])

	nextOffset := eq.nextOffset
	if len(result) > 0 {
		nextOffset = result[len(result)-1].Offset + 1
	}

	return result, nextOffset, hasMore
}

// WaitForEvents waits for new events to arrive or timeout
func (eq *EventQueue) WaitForEvents(fromOffset int64, timeout time.Duration) <-chan struct{} {
	eq.waitersMutex.Lock()
	defer eq.waitersMutex.Unlock()

	// Check if events already exist
	eq.mu.RLock()
	hasEvents := false
	for _, event := range eq.events {
		if event.Offset >= fromOffset {
			hasEvents = true
			break
		}
	}
	eq.mu.RUnlock()

	notifyChan := make(chan struct{}, 1)

	if hasEvents {
		// Events already available, notify immediately
		close(notifyChan)
		return notifyChan
	}

	// Add to waiters
	if eq.waiters[fromOffset] == nil {
		eq.waiters[fromOffset] = make([]chan struct{}, 0)
	}
	eq.waiters[fromOffset] = append(eq.waiters[fromOffset], notifyChan)

	// Set timeout
	go func() {
		time.Sleep(timeout)
		select {
		case <-notifyChan:
			// Already notified
		default:
			close(notifyChan)
		}
	}()

	return notifyChan
}

// GetCurrentOffset returns the current event offset (next offset to be assigned)
func (eq *EventQueue) GetCurrentOffset() int64 {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	return eq.nextOffset
}

// Close shuts down the event queue
func (eq *EventQueue) Close() error {
	eq.logger.Info("Shutting down event queue")

	close(eq.stopChan)

	// Wait a bit for async writer to finish
	time.Sleep(100 * time.Millisecond)

	// Final save
	return eq.saveToFile()
}

// getNextOffset returns the next available offset (thread-safe)
func (eq *EventQueue) getNextOffset() int64 {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	offset := eq.nextOffset
	eq.nextOffset++
	return offset
}

// asyncWriter handles writing events to memory and file asynchronously
func (eq *EventQueue) asyncWriter() {
	for {
		select {
		case event := <-eq.writeChan:
			eq.addEventToMemory(event)
			eq.notifyWaiters(event.Offset)

		case <-eq.stopChan:
			eq.logger.Info("Event queue async writer stopping")
			return
		}
	}
}

// addEventToMemory adds an event to the in-memory queue and manages rotation
func (eq *EventQueue) addEventToMemory(event models.Event) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	// Add event to memory
	eq.events = append(eq.events, event)

	// Rotate if necessary
	if len(eq.events) > eq.maxEvents {
		// Remove oldest events, keep recent ones
		keepCount := eq.maxEvents * 3 / 4 // Keep 75% of max events
		eq.events = eq.events[len(eq.events)-keepCount:]

		eq.logger.Info("Event queue rotated",
			"removed_events", len(eq.events)+keepCount-len(eq.events),
			"remaining_events", len(eq.events),
		)
	}

	// Immediately save to file for data consistency (synchronous for reliability)
	if err := eq.saveToFile(); err != nil {
		eq.logger.Error("Failed to save events to file immediately", "error", err)
	} else {
		eq.logger.Debug("Event saved to file immediately", "offset", event.Offset)
	}
}

// notifyWaiters notifies all waiters waiting for events at or after the given offset
func (eq *EventQueue) notifyWaiters(offset int64) {
	eq.waitersMutex.Lock()
	defer eq.waitersMutex.Unlock()

	for waitOffset, waiters := range eq.waiters {
		if waitOffset <= offset {
			for _, waiter := range waiters {
				select {
				case <-waiter:
					// Already closed
				default:
					close(waiter)
				}
			}
			delete(eq.waiters, waitOffset)
		}
	}
}

// loadFromFile loads events from the persistent file
func (eq *EventQueue) loadFromFile() error {
	data, err := os.ReadFile(eq.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, start fresh
		}
		return fmt.Errorf("failed to read events file: %w", err)
	}

	var fileData struct {
		Events     []models.Event `json:"events"`
		NextOffset int64          `json:"nextOffset"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal events: %w", err)
	}

	eq.events = fileData.Events
	eq.nextOffset = fileData.NextOffset

	return nil
}

// saveToFile saves events to the persistent file
func (eq *EventQueue) saveToFile() error {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	fileData := struct {
		Events     []models.Event `json:"events"`
		NextOffset int64          `json:"nextOffset"`
	}{
		Events:     eq.events,
		NextOffset: eq.nextOffset,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempFile := eq.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp events file: %w", err)
	}

	if err := os.Rename(tempFile, eq.filePath); err != nil {
		return fmt.Errorf("failed to rename temp events file: %w", err)
	}

	return nil
}

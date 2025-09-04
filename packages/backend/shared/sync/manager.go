package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/melibackend/shared/client"
	"github.com/melibackend/shared/storage"
)

// SyncManager defines the interface for synchronization managers
type SyncManager interface {
	Start(ctx context.Context) error
	Stop()
	InitialSync(ctx context.Context) error
	ForceSync(ctx context.Context) error
	GetSyncStatus() *storage.SyncStatus
	UpdateLocalProduct(productID string, available int, version int, lastUpdated time.Time) error
}

// Manager handles synchronization between central API and local storage
type Manager struct {
	client       *client.InventoryClient
	localStorage storage.LocalStorage
	logger       *slog.Logger
	syncInterval time.Duration
	syncMutex    sync.Mutex
	stopChan     chan struct{}
	status       *storage.SyncStatus
	statusMutex  sync.RWMutex
}

// NewManager creates a new sync manager
func NewManager(client *client.InventoryClient, localStorage storage.LocalStorage, logger *slog.Logger) *Manager {
	return &Manager{
		client:       client,
		localStorage: localStorage,
		logger:       logger,
		syncInterval: 5 * time.Minute, // Default sync interval
		stopChan:     make(chan struct{}),
		status: &storage.SyncStatus{
			InProgress:      false,
			LastSyncSuccess: false,
		},
	}
}

// SetSyncInterval sets the automatic sync interval
func (m *Manager) SetSyncInterval(interval time.Duration) {
	m.syncInterval = interval
}

// Start begins the sync manager with initial sync and periodic updates
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting sync manager")

	// Perform initial sync
	if err := m.InitialSync(ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Start periodic sync in background
	go m.periodicSync(ctx)

	return nil
}

// Stop stops the sync manager
func (m *Manager) Stop() {
	m.logger.Info("Stopping sync manager")
	close(m.stopChan)
}

// InitialSync performs a complete synchronization of all products
func (m *Manager) InitialSync(ctx context.Context) error {
	m.syncMutex.Lock()
	defer m.syncMutex.Unlock()

	m.logger.Info("Starting initial sync")
	startTime := time.Now()

	m.updateSyncStatus(true, false, 0, "", time.Time{})

	// Get all products from central API
	m.logger.Info("Attempting to get products from central API")
	products, err := m.client.GetAllProducts()
	if err != nil {
		m.logger.Error("Failed to get products from central API", "error", err)
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to get products from central API: %w", err)
	}

	m.logger.Info("Retrieved products from central API", "count", len(products))

	// Log first product for debugging
	if len(products) > 0 {
		m.logger.Info("First product sample",
			"productId", products[0].ProductID,
			"name", products[0].Name,
			"available", products[0].Available)
	}

	// Sync all products to local storage
	if err := m.localStorage.SyncAllProducts(products); err != nil {
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to sync products to local storage: %w", err)
	}

	// Update sync time
	syncTime := time.Now()
	if err := m.localStorage.SetLastSyncTime(syncTime); err != nil {
		m.logger.Warn("Failed to update last sync time", "error", err)
	}

	duration := time.Since(startTime)
	m.updateSyncStatus(false, true, len(products), "", syncTime)

	m.logger.Info("Initial sync completed successfully",
		"products_synced", len(products),
		"duration", duration,
	)

	return nil
}

// IncrementalSync performs an incremental sync (placeholder for future implementation)
func (m *Manager) IncrementalSync(ctx context.Context) error {
	m.syncMutex.Lock()
	defer m.syncMutex.Unlock()

	m.logger.Debug("Starting incremental sync")
	startTime := time.Now()

	// For now, perform a full sync
	// In the future, this could use timestamps or version numbers for incremental updates
	return m.performFullSync(ctx, startTime)
}

// ForceSync forces an immediate full synchronization
func (m *Manager) ForceSync(ctx context.Context) error {
	m.logger.Info("Force sync requested")
	return m.InitialSync(ctx)
}

// GetSyncStatus returns the current sync status
func (m *Manager) GetSyncStatus() *storage.SyncStatus {
	m.statusMutex.RLock()
	defer m.statusMutex.RUnlock()

	// Return a copy to avoid race conditions
	status := *m.status
	return &status
}

// UpdateLocalProduct updates a single product in local storage after a successful write operation
func (m *Manager) UpdateLocalProduct(productID string, available int, version int, lastUpdated time.Time) error {
	m.logger.Debug("Updating local product",
		"product_id", productID,
		"available", available,
		"version", version,
	)

	return m.localStorage.UpdateProduct(productID, available, version, lastUpdated)
}

// periodicSync runs periodic synchronization in the background
func (m *Manager) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(m.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Periodic sync stopped due to context cancellation")
			return
		case <-m.stopChan:
			m.logger.Info("Periodic sync stopped")
			return
		case <-ticker.C:
			if err := m.IncrementalSync(ctx); err != nil {
				m.logger.Error("Periodic sync failed", "error", err)
			}
		}
	}
}

// performFullSync performs a complete synchronization
func (m *Manager) performFullSync(ctx context.Context, startTime time.Time) error {
	m.updateSyncStatus(true, false, 0, "", time.Time{})

	// Get all products from central API
	products, err := m.client.GetAllProducts()
	if err != nil {
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to get products from central API: %w", err)
	}

	// Sync all products to local storage
	if err := m.localStorage.SyncAllProducts(products); err != nil {
		m.updateSyncStatus(false, false, 0, err.Error(), time.Time{})
		return fmt.Errorf("failed to sync products to local storage: %w", err)
	}

	// Update sync time
	syncTime := time.Now()
	if err := m.localStorage.SetLastSyncTime(syncTime); err != nil {
		m.logger.Warn("Failed to update last sync time", "error", err)
	}

	duration := time.Since(startTime)
	m.updateSyncStatus(false, true, len(products), "", syncTime)

	m.logger.Debug("Full sync completed",
		"products_synced", len(products),
		"duration", duration,
	)

	return nil
}

// updateSyncStatus updates the internal sync status
func (m *Manager) updateSyncStatus(inProgress, success bool, productCount int, errorMsg string, syncTime time.Time) {
	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()

	m.status.InProgress = inProgress
	m.status.LastSyncSuccess = success
	m.status.ProductCount = productCount
	m.status.ErrorMessage = errorMsg

	if !syncTime.IsZero() {
		m.status.LastSyncTime = syncTime
	}
}

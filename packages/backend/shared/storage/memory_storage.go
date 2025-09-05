package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/melibackend/shared/models"
)

// MemoryStorage implements LocalStorage using in-memory storage with file persistence
type MemoryStorage struct {
	mu              sync.RWMutex
	products        map[string]models.Product
	lastSyncTime    time.Time
	lastEventOffset int64
	initializedAt   time.Time
	dataFile        string
	metaFile        string
}

// StorageMetadata holds metadata about the storage
type StorageMetadata struct {
	LastSyncTime    time.Time `json:"lastSyncTime"`
	LastEventOffset int64     `json:"lastEventOffset"`
	InitializedAt   time.Time `json:"initializedAt"`
	ProductCount    int       `json:"productCount"`
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage(dataDir string) *MemoryStorage {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// If we can't create the directory, use current directory
		dataDir = "."
	}

	return &MemoryStorage{
		products:      make(map[string]models.Product),
		initializedAt: time.Now(),
		dataFile:      filepath.Join(dataDir, "local_inventory.json"),
		metaFile:      filepath.Join(dataDir, "storage_metadata.json"),
	}
}

// Initialize the storage and load existing data if available
func (ms *MemoryStorage) Initialize() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	slog.Debug("Initializing memory storage",
		"data_file", ms.dataFile,
		"meta_file", ms.metaFile)

	// Try to load existing data
	if err := ms.loadFromFile(); err != nil {
		// If loading fails, start with empty storage
		ms.products = make(map[string]models.Product)
		ms.lastSyncTime = time.Time{}
		slog.Info("🆕 Created new empty database - no existing data found",
			"error", err.Error(),
			"data_file", ms.dataFile)
	} else {
		slog.Info("📂 Loaded existing database from local files",
			"product_count", len(ms.products),
			"last_sync_time", ms.lastSyncTime.Format(time.RFC3339),
			"last_event_offset", ms.lastEventOffset,
			"data_file", ms.dataFile)
	}

	return nil
}

// Close the storage and persist data
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.saveToFile()
}

// SyncAllProducts replaces all products with the provided list
func (ms *MemoryStorage) SyncAllProducts(products []models.Product) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldProductCount := len(ms.products)

	slog.Info("🔄 Starting full database synchronization",
		"old_product_count", oldProductCount,
		"new_product_count", len(products))

	// Clear existing products
	ms.products = make(map[string]models.Product)

	// Add all new products
	for _, product := range products {
		ms.products[product.ProductID] = product
	}

	ms.lastSyncTime = time.Now()

	slog.Info("✅ Full database synchronization completed",
		"products_replaced", oldProductCount,
		"products_loaded", len(ms.products),
		"sync_time", ms.lastSyncTime.Format(time.RFC3339))

	// Persist to file
	return ms.saveToFile()
}

// GetLastSyncTime returns the last synchronization time
func (ms *MemoryStorage) GetLastSyncTime() (time.Time, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.lastSyncTime, nil
}

// SetLastSyncTime sets the last synchronization time
func (ms *MemoryStorage) SetLastSyncTime(t time.Time) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.lastSyncTime = t
	return ms.saveMetadata()
}

// GetLastEventOffset returns the last processed event offset by reading from the metadata file
func (ms *MemoryStorage) GetLastEventOffset() (int64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Always read from the metadata file to get the most current value
	if data, err := os.ReadFile(ms.metaFile); err == nil {
		var meta StorageMetadata
		if err := json.Unmarshal(data, &meta); err == nil {
			// Update the in-memory variable with the value from file
			ms.lastEventOffset = meta.LastEventOffset
			return meta.LastEventOffset, nil
		}
	}

	// If file doesn't exist or can't be read, return the in-memory value (fallback)
	return ms.lastEventOffset, nil
}

// SetLastEventOffset sets the last processed event offset
func (ms *MemoryStorage) SetLastEventOffset(offset int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.lastEventOffset = offset
	return ms.saveMetadata()
}

// ApplyEvents applies a batch of events to the local storage
func (ms *MemoryStorage) ApplyEvents(events []models.Event) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	slog.Debug("Applying events to local storage", "event_count", len(events))

	eventsProcessed := 0
	eventsSkipped := 0

	for _, event := range events {
		// Since events now contain complete product information,
		// we can create the product directly from event data
		product := models.Product{
			ProductID: event.Data.ProductID,
			Name:      event.Data.Name,
			Available: event.Data.Available,
			Version:   event.Data.Version,
			Price:     event.Data.Price,
		}

		// Parse the timestamp
		if lastUpdated, err := time.Parse(time.RFC3339, event.Data.LastUpdated); err == nil {
			product.LastUpdated = lastUpdated
		} else {
			product.LastUpdated = time.Now()
		}

		// Apply the event based on type
		switch event.EventType {
		case models.EventTypeProductUpdated:
			ms.products[event.ProductID] = product
			eventsProcessed++
			slog.Debug("Product updated in local storage",
				"product_id", event.ProductID,
				"name", product.Name,
				"available", product.Available,
				"version", product.Version,
				"offset", event.Offset)

		case models.EventTypeProductCreated:
			ms.products[event.ProductID] = product
			eventsProcessed++
			slog.Info("Product created in local storage",
				"product_id", event.ProductID,
				"name", product.Name,
				"available", product.Available,
				"price", product.Price,
				"version", product.Version,
				"offset", event.Offset)

		case models.EventTypeProductDeleted:
			// Check if product exists before deletion
			if _, exists := ms.products[event.ProductID]; exists {
				delete(ms.products, event.ProductID)
				eventsProcessed++
				slog.Info("Product deleted from local storage",
					"product_id", event.ProductID,
					"name", event.Data.Name,
					"offset", event.Offset)
			} else {
				eventsSkipped++
				slog.Warn("Attempted to delete non-existent product",
					"product_id", event.ProductID,
					"offset", event.Offset)
			}

		default:
			eventsSkipped++
			slog.Warn("Unknown event type, skipping",
				"event_type", event.EventType,
				"product_id", event.ProductID,
				"offset", event.Offset)
			continue
		}

		// Update the last processed offset
		if event.Offset >= ms.lastEventOffset {
			ms.lastEventOffset = event.Offset + 1
		}
	}

	// Log summary of applied events
	if len(events) > 0 {
		slog.Info("Successfully applied events to local storage",
			"events_received", len(events),
			"events_processed", eventsProcessed,
			"events_skipped", eventsSkipped,
			"total_products", len(ms.products),
			"last_offset", ms.lastEventOffset)
	}

	// Save to file after applying all events
	return ms.saveToFile()
}

// GetProduct retrieves a single product by ID
func (ms *MemoryStorage) GetProduct(productID string) (*models.Product, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	product, exists := ms.products[productID]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	return &product, nil
}

// GetAllProducts returns all products
func (ms *MemoryStorage) GetAllProducts() ([]models.Product, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	products := make([]models.Product, 0, len(ms.products))
	for _, product := range ms.products {
		products = append(products, product)
	}

	return products, nil
}

// UpsertProduct inserts or updates a product
func (ms *MemoryStorage) UpsertProduct(product models.Product) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.products[product.ProductID] = product
	return nil
}

// UpdateProduct updates specific fields of a product
func (ms *MemoryStorage) UpdateProduct(productID string, available int, version int, lastUpdated time.Time) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	product, exists := ms.products[productID]
	if !exists {
		return fmt.Errorf("product not found: %s", productID)
	}

	product.Available = available
	product.Version = version
	product.LastUpdated = lastUpdated
	ms.products[productID] = product

	return nil
}

// DeleteProduct removes a product from storage
func (ms *MemoryStorage) DeleteProduct(productID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.products[productID]; !exists {
		return fmt.Errorf("product not found: %s", productID)
	}

	delete(ms.products, productID)
	return nil
}

// BatchUpsertProducts inserts or updates multiple products
func (ms *MemoryStorage) BatchUpsertProducts(products []models.Product) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, product := range products {
		ms.products[product.ProductID] = product
	}

	return nil
}

// GetProductCount returns the number of products in storage
func (ms *MemoryStorage) GetProductCount() (int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return len(ms.products), nil
}

// GetStorageStats returns storage statistics
func (ms *MemoryStorage) GetStorageStats() (*StorageStats, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats := &StorageStats{
		ProductCount:   len(ms.products),
		LastSyncTime:   ms.lastSyncTime,
		InitializedAt:  ms.initializedAt,
		LastUpdateTime: time.Now(),
		MemoryUsage:    int64(memStats.Alloc),
	}

	// Calculate approximate storage size
	if len(ms.products) > 0 {
		// Estimate based on JSON serialization size
		sampleData, _ := json.Marshal(ms.products)
		stats.StorageSize = int64(len(sampleData))
	}

	return stats, nil
}

// loadFromFile loads products and metadata from files
func (ms *MemoryStorage) loadFromFile() error {
	var productsLoaded, metadataLoaded bool
	var productCount int

	// Load products
	if data, err := os.ReadFile(ms.dataFile); err == nil {
		if err := json.Unmarshal(data, &ms.products); err != nil {
			slog.Error("❌ Failed to parse products file",
				"file", ms.dataFile,
				"error", err.Error())
			return fmt.Errorf("failed to unmarshal products: %w", err)
		}
		productsLoaded = true
		productCount = len(ms.products)
		slog.Debug("✅ Products file loaded successfully",
			"file", ms.dataFile,
			"product_count", productCount,
			"file_size_bytes", len(data))
	} else {
		slog.Debug("📄 Products file not found",
			"file", ms.dataFile,
			"error", err.Error())
	}

	// Load metadata
	if data, err := os.ReadFile(ms.metaFile); err == nil {
		var meta StorageMetadata
		if err := json.Unmarshal(data, &meta); err == nil {
			ms.lastSyncTime = meta.LastSyncTime
			ms.lastEventOffset = meta.LastEventOffset
			ms.initializedAt = meta.InitializedAt
			metadataLoaded = true
			slog.Debug("✅ Metadata file loaded successfully",
				"file", ms.metaFile,
				"last_sync_time", meta.LastSyncTime.Format(time.RFC3339),
				"last_event_offset", meta.LastEventOffset,
				"initialized_at", meta.InitializedAt.Format(time.RFC3339),
				"file_size_bytes", len(data))
		} else {
			slog.Warn("❌ Failed to parse metadata file",
				"file", ms.metaFile,
				"error", err.Error())
		}
	} else {
		slog.Debug("📄 Metadata file not found",
			"file", ms.metaFile,
			"error", err.Error())
	}

	// Return error only if no files were loaded at all
	if !productsLoaded && !metadataLoaded {
		return fmt.Errorf("no existing data files found")
	}

	return nil
}

// saveToFile saves products and metadata to files
func (ms *MemoryStorage) saveToFile() error {
	// Save products
	data, err := json.MarshalIndent(ms.products, "", "  ")
	if err != nil {
		slog.Error("❌ Failed to marshal products for saving", "error", err.Error())
		return fmt.Errorf("failed to marshal products: %w", err)
	}

	if err := os.WriteFile(ms.dataFile, data, 0644); err != nil {
		slog.Error("❌ Failed to write products file",
			"file", ms.dataFile,
			"error", err.Error())
		return fmt.Errorf("failed to write products file: %w", err)
	}

	slog.Debug("💾 Products saved to file",
		"file", ms.dataFile,
		"product_count", len(ms.products),
		"file_size_bytes", len(data))

	// Save metadata
	return ms.saveMetadata()
}

// saveMetadata saves only the metadata
func (ms *MemoryStorage) saveMetadata() error {
	meta := StorageMetadata{
		LastSyncTime:    ms.lastSyncTime,
		LastEventOffset: ms.lastEventOffset,
		InitializedAt:   ms.initializedAt,
		ProductCount:    len(ms.products),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		slog.Error("❌ Failed to marshal metadata for saving", "error", err.Error())
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(ms.metaFile, data, 0644); err != nil {
		slog.Error("❌ Failed to write metadata file",
			"file", ms.metaFile,
			"error", err.Error())
		return err
	}

	slog.Debug("💾 Metadata saved to file",
		"file", ms.metaFile,
		"last_sync_time", meta.LastSyncTime.Format(time.RFC3339),
		"last_event_offset", meta.LastEventOffset,
		"product_count", meta.ProductCount,
		"file_size_bytes", len(data))

	return nil
}

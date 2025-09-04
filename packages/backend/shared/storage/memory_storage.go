package storage

import (
	"encoding/json"
	"fmt"
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

	// Try to load existing data
	if err := ms.loadFromFile(); err != nil {
		// If loading fails, start with empty storage
		ms.products = make(map[string]models.Product)
		ms.lastSyncTime = time.Time{}
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

	// Clear existing products
	ms.products = make(map[string]models.Product)

	// Add all new products
	for _, product := range products {
		ms.products[product.ProductID] = product
	}

	ms.lastSyncTime = time.Now()

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

	for _, event := range events {
		// Since events now contain complete product information,
		// we can create the product directly from event data
		product := models.Product{
			ProductID: event.Data.ProductID,
			Name:      event.Data.Name,
			Available: event.Data.Available,
			Version:   event.Data.Version,
		}

		// Parse the timestamp
		if lastUpdated, err := time.Parse(time.RFC3339, event.Data.LastUpdated); err == nil {
			product.LastUpdated = lastUpdated
		} else {
			product.LastUpdated = time.Now()
		}

		// Apply the event based on type
		switch event.EventType {
		case models.EventTypeProductUpdated, models.EventTypeProductCreated:
			ms.products[event.ProductID] = product

			// Event applied successfully (debug logging can be removed in production)
		default:
			// Unknown event type, skip processing
			// Unknown event type, skip processing
			continue
		}

		// Update the last processed offset
		if event.Offset >= ms.lastEventOffset {
			ms.lastEventOffset = event.Offset + 1
		}
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
	// Load products
	if data, err := os.ReadFile(ms.dataFile); err == nil {
		if err := json.Unmarshal(data, &ms.products); err != nil {
			return fmt.Errorf("failed to unmarshal products: %w", err)
		}
	}

	// Load metadata
	if data, err := os.ReadFile(ms.metaFile); err == nil {
		var meta StorageMetadata
		if err := json.Unmarshal(data, &meta); err == nil {
			ms.lastSyncTime = meta.LastSyncTime
			ms.lastEventOffset = meta.LastEventOffset
			ms.initializedAt = meta.InitializedAt
		}
	}

	return nil
}

// saveToFile saves products and metadata to files
func (ms *MemoryStorage) saveToFile() error {
	// Save products
	data, err := json.MarshalIndent(ms.products, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal products: %w", err)
	}

	if err := os.WriteFile(ms.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write products file: %w", err)
	}

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
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(ms.metaFile, data, 0644)
}

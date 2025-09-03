package storage

import (
	"time"

	"github.com/melibackend/shared/models"
)

// LocalStorage defines the interface for local inventory storage
type LocalStorage interface {
	// Initialize the storage
	Initialize() error

	// Close the storage
	Close() error

	// Sync operations
	SyncAllProducts(products []models.Product) error
	GetLastSyncTime() (time.Time, error)
	SetLastSyncTime(t time.Time) error

	// Product operations
	GetProduct(productID string) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
	UpsertProduct(product models.Product) error
	UpdateProduct(productID string, available int, version int, lastUpdated time.Time) error

	// Batch operations
	BatchUpsertProducts(products []models.Product) error

	// Statistics
	GetProductCount() (int, error)
	GetStorageStats() (*StorageStats, error)
}

// StorageStats provides information about the local storage
type StorageStats struct {
	ProductCount   int       `json:"productCount"`
	LastSyncTime   time.Time `json:"lastSyncTime"`
	StorageSize    int64     `json:"storageSize"`
	MemoryUsage    int64     `json:"memoryUsage"`
	InitializedAt  time.Time `json:"initializedAt"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

// SyncStatus represents the synchronization status
type SyncStatus struct {
	InProgress      bool          `json:"inProgress"`
	LastSyncTime    time.Time     `json:"lastSyncTime"`
	LastSyncSuccess bool          `json:"lastSyncSuccess"`
	ProductCount    int           `json:"productCount"`
	SyncDuration    time.Duration `json:"syncDuration"`
	ErrorMessage    string        `json:"errorMessage,omitempty"`
}

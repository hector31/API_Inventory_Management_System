package services

import (
	"log/slog"
	"sync"
	"time"
)

// ProductLockManager manages fine-grained locks per product ID
type ProductLockManager struct {
	locks    map[string]*sync.RWMutex
	locksMux sync.RWMutex
}

// NewProductLockManager creates a new product lock manager
func NewProductLockManager() *ProductLockManager {
	return &ProductLockManager{
		locks: make(map[string]*sync.RWMutex),
	}
}

// GetProductLock returns a mutex for the specified product ID
// Creates a new mutex if one doesn't exist for this product
func (plm *ProductLockManager) GetProductLock(productID string) *sync.RWMutex {
	plm.locksMux.RLock()
	if lock, exists := plm.locks[productID]; exists {
		plm.locksMux.RUnlock()
		return lock
	}
	plm.locksMux.RUnlock()

	// Need to create a new lock
	plm.locksMux.Lock()
	defer plm.locksMux.Unlock()

	// Double-check in case another goroutine created it
	if lock, exists := plm.locks[productID]; exists {
		return lock
	}

	// Create new lock for this product
	newLock := &sync.RWMutex{}
	plm.locks[productID] = newLock
	
	slog.Debug("Created new product lock", "product_id", productID)
	return newLock
}

// LockProductForWrite acquires a write lock for the specified product
func (plm *ProductLockManager) LockProductForWrite(productID string) *sync.RWMutex {
	lock := plm.GetProductLock(productID)
	lock.Lock()
	slog.Debug("Acquired write lock for product", "product_id", productID)
	return lock
}

// LockProductForRead acquires a read lock for the specified product
func (plm *ProductLockManager) LockProductForRead(productID string) *sync.RWMutex {
	lock := plm.GetProductLock(productID)
	lock.RLock()
	slog.Debug("Acquired read lock for product", "product_id", productID)
	return lock
}

// UnlockProductWrite releases a write lock for the specified product
func (plm *ProductLockManager) UnlockProductWrite(productID string, lock *sync.RWMutex) {
	lock.Unlock()
	slog.Debug("Released write lock for product", "product_id", productID)
}

// UnlockProductRead releases a read lock for the specified product
func (plm *ProductLockManager) UnlockProductRead(productID string, lock *sync.RWMutex) {
	lock.RUnlock()
	slog.Debug("Released read lock for product", "product_id", productID)
}

// WithProductWriteLock executes a function while holding a write lock for the product
func (plm *ProductLockManager) WithProductWriteLock(productID string, fn func()) {
	start := time.Now()
	lock := plm.LockProductForWrite(productID)
	defer plm.UnlockProductWrite(productID, lock)
	
	fn()
	
	duration := time.Since(start)
	slog.Debug("Product write operation completed", 
		"product_id", productID, 
		"duration", duration.String())
}

// WithProductReadLock executes a function while holding a read lock for the product
func (plm *ProductLockManager) WithProductReadLock(productID string, fn func()) {
	start := time.Now()
	lock := plm.LockProductForRead(productID)
	defer plm.UnlockProductRead(productID, lock)
	
	fn()
	
	duration := time.Since(start)
	slog.Debug("Product read operation completed", 
		"product_id", productID, 
		"duration", duration.String())
}

// GetLockStats returns statistics about the lock manager
func (plm *ProductLockManager) GetLockStats() map[string]interface{} {
	plm.locksMux.RLock()
	defer plm.locksMux.RUnlock()
	
	return map[string]interface{}{
		"total_product_locks": len(plm.locks),
		"lock_manager_type":   "fine_grained_per_product",
	}
}

// CleanupUnusedLocks removes locks for products that are no longer needed
// This is optional and can be called periodically to prevent memory growth
func (plm *ProductLockManager) CleanupUnusedLocks(activeProductIDs map[string]bool) {
	plm.locksMux.Lock()
	defer plm.locksMux.Unlock()
	
	removedCount := 0
	for productID := range plm.locks {
		if !activeProductIDs[productID] {
			delete(plm.locks, productID)
			removedCount++
		}
	}
	
	if removedCount > 0 {
		slog.Info("Cleaned up unused product locks", 
			"removed_locks", removedCount,
			"remaining_locks", len(plm.locks))
	}
}

package cache

import (
	"log/slog"
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration time
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// TTLCache implements a thread-safe cache with TTL (Time To Live) functionality
type TTLCache struct {
	items       map[string]*CacheEntry
	mutex       sync.RWMutex
	ttl         time.Duration
	cleanupTicker *time.Ticker
	stopCleanup chan bool
}

// NewTTLCache creates a new TTL cache with specified TTL and cleanup interval
func NewTTLCache(ttl, cleanupInterval time.Duration) *TTLCache {
	cache := &TTLCache{
		items:       make(map[string]*CacheEntry),
		ttl:         ttl,
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine
	cache.cleanupTicker = time.NewTicker(cleanupInterval)
	go cache.cleanupExpiredEntries()

	slog.Info("TTL cache initialized", 
		"ttl", ttl.String(), 
		"cleanup_interval", cleanupInterval.String())

	return cache
}

// Set stores a value in the cache with TTL
func (c *TTLCache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiresAt := time.Now().Add(c.ttl)
	c.items[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	slog.Debug("Cache entry set", 
		"key", key, 
		"expires_at", expiresAt.Format(time.RFC3339))
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		slog.Debug("Cache entry expired", "key", key)
		return nil, false
	}

	slog.Debug("Cache hit", "key", key)
	return entry.Value, true
}

// Delete removes a specific key from the cache
func (c *TTLCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
	slog.Debug("Cache entry deleted", "key", key)
}

// Size returns the current number of items in the cache (including expired ones)
func (c *TTLCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// ActiveSize returns the number of non-expired items in the cache
func (c *TTLCache) ActiveSize() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	activeCount := 0
	for _, entry := range c.items {
		if now.Before(entry.ExpiresAt) {
			activeCount++
		}
	}
	return activeCount
}

// Clear removes all items from the cache
func (c *TTLCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	itemCount := len(c.items)
	c.items = make(map[string]*CacheEntry)
	
	slog.Info("Cache cleared", "removed_items", itemCount)
}

// Stop stops the cleanup goroutine and cleans up resources
func (c *TTLCache) Stop() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
	close(c.stopCleanup)
	slog.Info("TTL cache stopped")
}

// cleanupExpiredEntries runs periodically to remove expired entries
func (c *TTLCache) cleanupExpiredEntries() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.performCleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// performCleanup removes expired entries from the cache
func (c *TTLCache) performCleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Find expired keys
	for key, entry := range c.items {
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired keys
	for _, key := range expiredKeys {
		delete(c.items, key)
	}

	if len(expiredKeys) > 0 {
		slog.Debug("Cache cleanup completed", 
			"expired_entries", len(expiredKeys),
			"remaining_entries", len(c.items))
	}
}

// GetStats returns cache statistics
func (c *TTLCache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	activeCount := 0
	expiredCount := 0

	for _, entry := range c.items {
		if now.Before(entry.ExpiresAt) {
			activeCount++
		} else {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(c.items),
		"active_entries":  activeCount,
		"expired_entries": expiredCount,
		"ttl_duration":    c.ttl.String(),
	}
}

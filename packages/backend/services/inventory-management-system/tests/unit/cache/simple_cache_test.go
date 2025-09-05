package cache

import (
	"testing"
	"time"

	"inventory-management-api/internal/cache"

	"github.com/stretchr/testify/assert"
)

// TestTTLCache_BasicOperations tests basic cache operations
func TestTTLCache_BasicOperations(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	key := "test-key"
	value := "test-value"
	
	// Act & Assert - Set and Get
	ttlCache.Set(key, value)
	retrievedValue, exists := ttlCache.Get(key)
	
	assert.True(t, exists, "Key should exist in cache")
	assert.Equal(t, value, retrievedValue, "Retrieved value should match set value")
}

// TestTTLCache_NonExistentKey tests getting a non-existent key
func TestTTLCache_NonExistentKey(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	// Act
	value, exists := ttlCache.Get("non-existent-key")
	
	// Assert
	assert.False(t, exists, "Non-existent key should not exist")
	assert.Nil(t, value, "Value for non-existent key should be nil")
}

// TestTTLCache_UpdateExistingKey tests updating an existing key
func TestTTLCache_UpdateExistingKey(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	key := "update-key"
	originalValue := "original-value"
	updatedValue := "updated-value"
	
	// Act
	ttlCache.Set(key, originalValue)
	ttlCache.Set(key, updatedValue)
	retrievedValue, exists := ttlCache.Get(key)
	
	// Assert
	assert.True(t, exists, "Key should exist after update")
	assert.Equal(t, updatedValue, retrievedValue, "Retrieved value should be the updated value")
}

// TestTTLCache_Delete tests deleting items from cache
func TestTTLCache_Delete(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	key := "delete-key"
	value := "delete-value"
	
	// Act
	ttlCache.Set(key, value)
	
	// Verify item exists
	_, exists := ttlCache.Get(key)
	assert.True(t, exists, "Key should exist before deletion")
	
	// Delete the item
	ttlCache.Delete(key)
	
	// Assert
	_, exists = ttlCache.Get(key)
	assert.False(t, exists, "Key should not exist after deletion")
}

// TestTTLCache_Clear tests clearing all items from cache
func TestTTLCache_Clear(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	// Add multiple items
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	for key, value := range items {
		ttlCache.Set(key, value)
	}
	
	// Verify all items exist
	for key := range items {
		_, exists := ttlCache.Get(key)
		assert.True(t, exists, "Key %s should exist before clear", key)
	}
	
	// Act
	ttlCache.Clear()
	
	// Assert
	for key := range items {
		_, exists := ttlCache.Get(key)
		assert.False(t, exists, "Key %s should not exist after clear", key)
	}
}

// TestTTLCache_Size tests getting the cache size
func TestTTLCache_Size(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	// Initially empty
	assert.Equal(t, 0, ttlCache.Size(), "Cache should be empty initially")
	
	// Add items
	ttlCache.Set("key1", "value1")
	assert.Equal(t, 1, ttlCache.Size(), "Cache size should be 1 after adding one item")
	
	ttlCache.Set("key2", "value2")
	assert.Equal(t, 2, ttlCache.Size(), "Cache size should be 2 after adding two items")
	
	// Update existing item (should not change size)
	ttlCache.Set("key1", "updated-value1")
	assert.Equal(t, 2, ttlCache.Size(), "Cache size should remain 2 after updating existing item")
	
	// Delete item
	ttlCache.Delete("key1")
	assert.Equal(t, 1, ttlCache.Size(), "Cache size should be 1 after deleting one item")
	
	// Clear cache
	ttlCache.Clear()
	assert.Equal(t, 0, ttlCache.Size(), "Cache size should be 0 after clearing")
}

// TestTTLCache_Expiration tests that items expire after TTL
func TestTTLCache_Expiration(t *testing.T) {
	// Arrange
	shortTTL := 100 * time.Millisecond
	ttlCache := cache.NewTTLCache(shortTTL, 50*time.Millisecond)
	defer ttlCache.Stop()
	
	key := "expiring-key"
	value := "expiring-value"
	
	// Act
	ttlCache.Set(key, value)
	
	// Verify item exists immediately
	retrievedValue, exists := ttlCache.Get(key)
	assert.True(t, exists, "Key should exist immediately after setting")
	assert.Equal(t, value, retrievedValue, "Value should match immediately after setting")
	
	// Wait for expiration
	time.Sleep(shortTTL + 100*time.Millisecond)
	
	// Assert - Item should eventually be cleaned up
	// Note: We give it some time for the cleanup goroutine to run
	eventually := func() bool {
		_, exists := ttlCache.Get(key)
		return !exists
	}
	
	// Wait up to 1 second for cleanup
	for i := 0; i < 10; i++ {
		if eventually() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	_, exists = ttlCache.Get(key)
	assert.False(t, exists, "Key should expire after TTL")
}

// TestTTLCache_GetStats tests cache statistics
func TestTTLCache_GetStats(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	// Act
	stats := ttlCache.GetStats()
	
	// Assert
	assert.NotNil(t, stats, "Stats should not be nil")
	
	// Add some items and check stats again
	ttlCache.Set("key1", "value1")
	ttlCache.Set("key2", "value2")
	
	stats = ttlCache.GetStats()
	assert.NotNil(t, stats, "Stats should not be nil after adding items")
}

// TestTTLCache_Stop tests stopping the cache
func TestTTLCache_Stop(t *testing.T) {
	// Arrange
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	
	// Add some items
	ttlCache.Set("stop-key", "stop-value")
	assert.Equal(t, 1, ttlCache.Size(), "Item should exist before stop")
	
	// Act - Stop the cache
	ttlCache.Stop()
	
	// Assert - Cache operations should still work but cleanup stops
	ttlCache.Set("after-stop-key", "after-stop-value")
	value, exists := ttlCache.Get("after-stop-key")
	assert.True(t, exists, "Cache operations should still work after stop")
	assert.Equal(t, "after-stop-value", value, "Value should be correct after stop")
}

// BenchmarkTTLCache_Set benchmarks cache set operations
func BenchmarkTTLCache_Set(b *testing.B) {
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		key := "bench-key"
		value := "bench-value"
		ttlCache.Set(key, value)
	}
}

// BenchmarkTTLCache_Get benchmarks cache get operations
func BenchmarkTTLCache_Get(b *testing.B) {
	ttlCache := cache.NewTTLCache(time.Minute, 30*time.Second)
	defer ttlCache.Stop()
	
	// Pre-populate cache
	ttlCache.Set("bench-key", "bench-value")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ttlCache.Get("bench-key")
	}
}

# JSON Persistence and TTL Cache Implementation Guide

This guide explains the implementation of JSON file persistence and TTL-based idempotency cache in the Inventory Management API.

## Overview

The enhanced inventory service now includes:

1. **JSON File Persistence**: Automatic saving of inventory changes back to the JSON file
2. **TTL-Based Idempotency Cache**: Time-limited cache with automatic cleanup
3. **Environment Variable Configuration**: Configurable cache and persistence settings

## Features Implemented

### 1. JSON File Persistence

**Automatic File Updates:**
- All successful inventory updates are persisted back to `data/inventory_test_data.json`
- Atomic file operations using temporary files to prevent corruption
- Configurable via `ENABLE_JSON_PERSISTENCE` environment variable

**Benefits:**
- Data survives server restarts
- No data loss on system failures
- Maintains consistency between in-memory and file storage

### 2. TTL-Based Idempotency Cache

**Cache Features:**
- Configurable TTL (Time To Live) for cache entries
- Automatic cleanup of expired entries
- Thread-safe operations with proper locking
- Memory-efficient with automatic garbage collection

**Configuration:**
- `IDEMPOTENCY_CACHE_TTL`: How long to keep cache entries (default: 2m)
- `IDEMPOTENCY_CACHE_CLEANUP_INTERVAL`: How often to clean expired entries (default: 30s)

## Environment Variables

### Cache Configuration
```bash
# TTL for idempotency cache entries
IDEMPOTENCY_CACHE_TTL=2m

# Cleanup interval for expired entries
IDEMPOTENCY_CACHE_CLEANUP_INTERVAL=30s
```

### Persistence Configuration
```bash
# Enable/disable JSON file persistence
ENABLE_JSON_PERSISTENCE=true
```

## Implementation Details

### TTL Cache Structure
```go
type TTLCache struct {
    items       map[string]*CacheEntry
    mutex       sync.RWMutex
    ttl         time.Duration
    cleanupTicker *time.Ticker
    stopCleanup chan bool
}

type CacheEntry struct {
    Value     interface{}
    ExpiresAt time.Time
}
```

### JSON Persistence Process
1. **Atomic Write**: Write to temporary file first
2. **Atomic Replace**: Rename temp file to original
3. **Error Handling**: Clean up temp file on failure
4. **Non-Blocking**: File save errors don't fail the update operation

### Service Initialization
```go
service := &InventoryService{
    updateQueue:           make(chan *UpdateRequest, 100),
    idempotencyCache:      cache.NewTTLCache(cacheTTL, cleanupInterval),
    dataFilePath:          cfg.DataPath,
    enableJSONPersistence: enablePersistence,
}
```

## Testing the Implementation

### 1. Test JSON Persistence

**Step 1: Check initial state**
```bash
curl -H 'X-API-Key: demo' http://localhost:8082/v1/inventory/SKU-001
# Response: {"productId":"SKU-001","available":45,"version":12,"lastUpdated":"2025-09-02T10:30:00Z"}
```

**Step 2: Apply an update**
```bash
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8082/v1/inventory/updates \
  --data-binary @test_single_update_success.json
```

**Step 3: Verify the update**
```bash
curl -H 'X-API-Key: demo' http://localhost:8082/v1/inventory/SKU-001
# Response should show updated quantity and incremented version
```

**Step 4: Restart server and verify persistence**
```bash
# Stop server (Ctrl+C)
# Start server again
go run ./cmd/server

# Check if changes persisted
curl -H 'X-API-Key: demo' http://localhost:8082/v1/inventory/SKU-001
# Should show the updated values, not the original JSON file values
```

### 2. Test TTL Cache Behavior

**Test Idempotency Within TTL Window:**
```bash
# First request
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8082/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-002","delta":3,"version":8,"idempotencyKey":"ttl-test-001"}'

# Immediate duplicate request (should return cached result)
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8082/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-002","delta":3,"version":8,"idempotencyKey":"ttl-test-001"}'
```

**Test Cache Expiration:**
```bash
# Wait for TTL to expire (default 2 minutes)
# Then try the same request again - should be processed as new request
```

### 3. Test Cache Cleanup

**Monitor Cache Statistics:**
The cache provides statistics that can be monitored:
- Total entries
- Active (non-expired) entries  
- Expired entries
- TTL duration

**Cleanup Process:**
- Runs every 30 seconds (configurable)
- Removes expired entries automatically
- Logs cleanup activity at debug level

## Configuration Examples

### Development Environment
```bash
# .env
IDEMPOTENCY_CACHE_TTL=1m
IDEMPOTENCY_CACHE_CLEANUP_INTERVAL=15s
ENABLE_JSON_PERSISTENCE=true
LOG_LEVEL=debug
```

### Production Environment
```bash
# .env
IDEMPOTENCY_CACHE_TTL=5m
IDEMPOTENCY_CACHE_CLEANUP_INTERVAL=1m
ENABLE_JSON_PERSISTENCE=true
LOG_LEVEL=info
```

### Testing Environment
```bash
# .env
IDEMPOTENCY_CACHE_TTL=30s
IDEMPOTENCY_CACHE_CLEANUP_INTERVAL=10s
ENABLE_JSON_PERSISTENCE=false
LOG_LEVEL=debug
```

## Logging and Monitoring

### Cache-Related Logs
```
# Cache initialization
level=INFO msg="TTL cache initialized" ttl=2m0s cleanup_interval=30s

# Cache operations
level=DEBUG msg="Cache entry set" key=test-key expires_at=2025-09-02T15:32:00Z
level=DEBUG msg="Cache hit" key=test-key
level=DEBUG msg="Cache entry expired" key=test-key

# Cache cleanup
level=DEBUG msg="Cache cleanup completed" expired_entries=5 remaining_entries=10
```

### Persistence-Related Logs
```
# File save operations
level=DEBUG msg="Saving inventory data to file" path=data/inventory_test_data.json
level=INFO msg="Inventory data saved to file successfully" path=data/inventory_test_data.json products_count=10 last_offset=1288

# Persistence errors
level=ERROR msg="Failed to persist inventory data to file" error="permission denied" product_id=SKU-001
```

## Benefits

### JSON Persistence
1. **Data Durability**: Changes survive server restarts
2. **Backup Integration**: File-based storage integrates with backup systems
3. **Debugging**: Easy to inspect current state via file system
4. **Migration**: Simple data export/import capabilities

### TTL Cache
1. **Memory Efficiency**: Automatic cleanup prevents memory leaks
2. **Configurable Behavior**: Adjustable TTL and cleanup intervals
3. **Production Ready**: Proper resource management and monitoring
4. **Performance**: Fast in-memory lookups with controlled growth

## Error Handling

### File Persistence Errors
- Non-blocking: Update operations succeed even if file save fails
- Logged: All persistence errors are logged for monitoring
- Atomic: Uses temporary files to prevent corruption

### Cache Errors
- Graceful degradation: Cache failures don't affect core functionality
- Resource cleanup: Proper shutdown procedures prevent resource leaks
- Monitoring: Cache statistics available for health checks

## Migration from Previous Implementation

### Before (Simple Map)
```go
idempotencyKeys map[string]*UpdateResult
idempotencyMux  sync.RWMutex
```

### After (TTL Cache)
```go
idempotencyCache *cache.TTLCache
```

### Benefits of Migration
1. **Memory Management**: Automatic cleanup vs. indefinite growth
2. **Configuration**: Environment-based settings vs. hardcoded behavior
3. **Monitoring**: Built-in statistics vs. no visibility
4. **Production Ready**: Proper resource management vs. memory leaks

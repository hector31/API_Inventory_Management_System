# Optimistic Concurrency Control (OCC) and Idempotency Guide

This guide explains the implementation of Optimistic Concurrency Control and Idempotency in the Inventory Management API.

## Overview

The inventory update endpoint (`POST /v1/inventory/updates`) implements:

1. **Optimistic Concurrency Control (OCC)**: Version-based conflict detection
2. **Idempotency**: Duplicate request prevention using idempotency keys
3. **Queue-Based Processing**: Asynchronous processing using Go channels to simulate distributed systems

## Architecture

### Queue-Based Processing
```
Client Request → Handler → Queue (Go Channel) → Sequential Processor → Response
```

- **Buffered Channel**: 100-item buffer for update requests
- **Sequential Processing**: Updates processed one at a time to ensure consistency
- **Timeout Handling**: 5-second timeout for queue submission, 10-second timeout for processing

### Concurrency Control
- **Read-Write Mutex**: Protects inventory data during concurrent access
- **Version Checking**: Each product has a version number that increments with each update
- **Conflict Detection**: Updates fail if the provided version doesn't match current version

### Idempotency
- **Key-Based Caching**: Results cached by idempotency key
- **Duplicate Prevention**: Identical requests return cached results without re-processing
- **Memory Storage**: In-memory cache for demonstration (production would use Redis/database)

## API Usage

### Successful Update

**Request:**
```json
POST /v1/inventory/updates
{
  "storeId": "store-1",
  "productId": "SKU-001",
  "delta": 5,
  "version": 12,
  "idempotencyKey": "unique-key-001"
}
```

**Response (200 OK):**
```json
{
  "productId": "SKU-001",
  "newQuantity": 50,
  "newVersion": 13,
  "applied": true,
  "lastUpdated": "2025-09-02T15:30:00Z"
}
```

### Version Conflict

**Request:**
```json
POST /v1/inventory/updates
{
  "storeId": "store-1",
  "productId": "SKU-001",
  "delta": 3,
  "version": 10,
  "idempotencyKey": "conflict-key-001"
}
```

**Response (409 Conflict):**
```json
{
  "productId": "SKU-001",
  "applied": false
}
```

### Idempotent Request

**First Request:**
```json
POST /v1/inventory/updates
{
  "storeId": "store-1",
  "productId": "SKU-002",
  "delta": 2,
  "version": 8,
  "idempotencyKey": "duplicate-key-001"
}
```

**Response (200 OK):**
```json
{
  "productId": "SKU-002",
  "newQuantity": 25,
  "newVersion": 9,
  "applied": true,
  "lastUpdated": "2025-09-02T15:31:00Z"
}
```

**Duplicate Request (same idempotencyKey):**
```json
POST /v1/inventory/updates
{
  "storeId": "store-1",
  "productId": "SKU-002",
  "delta": 2,
  "version": 8,
  "idempotencyKey": "duplicate-key-001"
}
```

**Response (200 OK - cached result):**
```json
{
  "productId": "SKU-002",
  "newQuantity": 25,
  "newVersion": 9,
  "applied": true,
  "lastUpdated": "2025-09-02T15:31:00Z"
}
```

### Batch Updates with Mixed Results

**Request:**
```json
POST /v1/inventory/updates
{
  "storeId": "store-1",
  "updates": [
    {
      "productId": "SKU-003",
      "delta": 10,
      "version": 15,
      "idempotencyKey": "batch-success-001"
    },
    {
      "productId": "SKU-004",
      "delta": -5,
      "version": 999,
      "idempotencyKey": "batch-conflict-001"
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "results": [
    {
      "productId": "SKU-003",
      "newQuantity": 77,
      "newVersion": 16,
      "applied": true,
      "lastUpdated": "2025-09-02T15:32:00Z"
    },
    {
      "productId": "SKU-004",
      "applied": false,
      "error": "version conflict: expected 6, got 999"
    }
  ],
  "summary": {
    "total": 2,
    "succeeded": 1,
    "failed": 1
  }
}
```

## Testing Scenarios

### 1. Test Successful Update
```bash
# Get current state
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-001

# Apply update with correct version
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-001","delta":5,"version":12,"idempotencyKey":"test-001"}'

# Verify new state
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-001
```

### 2. Test Version Conflict
```bash
# Try update with old version
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-001","delta":3,"version":10,"idempotencyKey":"conflict-001"}'
```

### 3. Test Idempotency
```bash
# First request
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-002","delta":2,"version":8,"idempotencyKey":"duplicate-001"}'

# Duplicate request (same idempotencyKey)
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-002","delta":2,"version":8,"idempotencyKey":"duplicate-001"}'
```

### 4. Test Concurrent Updates
```bash
# Simulate concurrent updates (run simultaneously)
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-003","delta":1,"version":15,"idempotencyKey":"concurrent-001"}' &

curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8081/v1/inventory/updates \
  -d '{"storeId":"store-1","productId":"SKU-003","delta":2,"version":15,"idempotencyKey":"concurrent-002"}' &

wait
```

## Implementation Details

### Service Layer
- **UpdateInventory()**: Public method that submits requests to queue
- **processUpdateQueue()**: Background worker that processes requests sequentially
- **processUpdateInternal()**: Core logic with OCC and idempotency checks
- **cacheIdempotencyResult()**: Stores results for duplicate prevention

### Error Handling
- **Version Conflicts**: Return 409 Conflict for single updates
- **Missing Fields**: Return 400 Bad Request
- **Product Not Found**: Return appropriate error response
- **Queue Timeouts**: Return 500 Internal Server Error

### Logging
- **Structured Logging**: All operations logged with context
- **Debug Level**: Queue submissions and processing details
- **Info Level**: Successful operations and conflicts
- **Warn Level**: Version conflicts and validation errors
- **Error Level**: System failures and timeouts

## Benefits

1. **Data Consistency**: OCC prevents lost updates in concurrent scenarios
2. **Reliability**: Idempotency ensures safe retries
3. **Scalability**: Queue-based processing simulates distributed architecture
4. **Observability**: Comprehensive logging for monitoring and debugging
5. **Performance**: Non-blocking queue operations with timeouts

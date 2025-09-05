# Store API Service (Store S1)

The Store API serves as a **local cache layer** in the distributed inventory management system, providing fast read access to inventory data while maintaining synchronization with the Central Inventory API through event-driven architecture.

## 🎯 API Overview

### Role in the Distributed System
- **Local Cache Layer**: Maintains a local copy of inventory data for fast read operations
- **Event Consumer**: Continuously synchronizes with Central API via event streaming
- **Write Proxy**: Forwards all write operations to the Central API with store-specific idempotency
- **Fallback Handler**: Provides graceful degradation when Central API is temporarily unavailable

### Key Capabilities
- ✅ **Event-Driven Synchronization** with real-time updates from Central API
- ✅ **Local Caching** with file persistence for fast read operations
- ✅ **Long Polling** for efficient event consumption with configurable timeouts
- ✅ **Circuit Breaker Pattern** with automatic fallback to full synchronization
- ✅ **Store-Specific Idempotency** with prefixed keys to avoid conflicts
- ✅ **Graceful Degradation** when Central API is unavailable

### Architecture Position
```
Store Frontend → Store API (Local Cache) → Central API (Source of Truth)
                      ↑
                Event Stream (Real-time Sync)
```

## 🚀 Quick Start Guide

### Running with Docker (Recommended)
```bash
# Build the container
docker build -t store-s1-api .

# Run with default configuration
docker run -p 8083:8083 \
  -e CENTRAL_API_URL=http://central-api:8081 \
  -e CENTRAL_API_KEY=demo \
  store-s1-api

# Run with custom synchronization settings
docker run -p 8083:8083 \
  -e CENTRAL_API_URL=http://central-api:8081 \
  -e CENTRAL_API_KEY=demo \
  -e SYNC_INTERVAL_SECONDS=10 \
  -e EVENT_WAIT_TIMEOUT_SECONDS=30 \
  -e EVENT_BATCH_LIMIT=200 \
  store-s1-api
```

### Running with Go (Development)
```bash
# Install dependencies
go mod download

# Set environment variables
export PORT=8083
export CENTRAL_API_URL=http://localhost:8081
export CENTRAL_API_KEY=demo
export LOG_LEVEL=debug
export API_KEYS=store-s1-key,demo

# Run the server
go run cmd/server/main.go

# Or build and run
go build -o store-api cmd/server/main.go
./store-api
```

### Health Check & Verification
```bash
# Verify the Store API is running
curl http://localhost:8083/health

# Check synchronization status
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/status

# Test local cache access
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/inventory
```

## 📚 API Endpoints Documentation

### Authentication
All protected endpoints require an API key via the `X-API-Key` header:
```bash
# Store-specific API key or demo key
X-API-Key: store-s1-key
# or
X-API-Key: demo
```

### Store Inventory Endpoints (`/v1/store/*`)

#### 1. Get All Products (Local Cache)
**GET** `/v1/store/inventory?offset=0&limit=50`

Retrieves products from the local cache with pagination support.

**Query Parameters:**
- `offset` (optional): Starting position for pagination (default: 0)
- `limit` (optional): Number of items per page (default: 50, max: 200)

**Response:**
```json
{
  "products": [
    {
      "productId": "PROD-001",
      "name": "Wireless Headphones",
      "available": 10,
      "version": 6,
      "lastUpdated": "2024-01-15T10:30:00Z",
      "price": 99.99
    }
  ],
  "pagination": {
    "offset": 0,
    "limit": 50,
    "total": 150,
    "hasMore": true
  },
  "cacheInfo": {
    "lastSyncTime": "2024-01-15T10:29:45Z",
    "totalProducts": 150
  }
}
```

#### 2. Get Single Product (Local Cache)
**GET** `/v1/store/inventory/{productId}`

Retrieves a specific product from the local cache.

**Response:**
```json
{
  "productId": "PROD-001",
  "name": "Wireless Headphones",
  "available": 10,
  "version": 6,
  "lastUpdated": "2024-01-15T10:30:00Z",
  "price": 99.99
}
```

**Error Response (Product Not Found):**
```json
{
  "error": {
    "code": "product_not_found",
    "message": "Product not found in local cache",
    "details": {
      "productId": "PROD-999",
      "suggestion": "Product may not exist or cache may be out of sync"
    }
  }
}
```

#### 3. Update Inventory (Proxy to Central)
**POST** `/v1/store/inventory/updates`

Updates product inventory by forwarding the request to the Central API with store-specific idempotency.

**Request:**
```json
{
  "storeId": "store-s1",
  "productId": "PROD-001",
  "delta": -2,
  "version": 6,
  "idempotencyKey": "order-12345"
}
```

**Response (Success):**
```json
{
  "productId": "PROD-001",
  "newQuantity": 8,
  "newVersion": 7,
  "applied": true,
  "lastUpdated": "2024-01-15T10:30:00Z"
}
```

**Response (Version Conflict):**
```json
{
  "productId": "PROD-001",
  "applied": false,
  "newQuantity": 10,
  "newVersion": 7,
  "errorType": "version_conflict",
  "errorMessage": "version conflict: expected 7, got 6"
}
```

#### 4. Batch Update Inventory (Proxy to Central)
**POST** `/v1/store/inventory/batch-updates`

Performs batch inventory updates via the Central API.

**Request:**
```json
{
  "storeId": "store-s1",
  "updates": [
    {
      "productId": "PROD-001",
      "delta": -1,
      "version": 6,
      "idempotencyKey": "batch-item-1"
    },
    {
      "productId": "PROD-002",
      "delta": -3,
      "version": 2,
      "idempotencyKey": "batch-item-2"
    }
  ]
}
```

**Response:**
```json
{
  "storeId": "store-s1",
  "totalCount": 2,
  "successCount": 1,
  "failureCount": 1,
  "results": [
    {
      "productId": "PROD-001",
      "newQuantity": 9,
      "newVersion": 7,
      "applied": true,
      "lastUpdated": "2024-01-15T10:30:00Z"
    },
    {
      "productId": "PROD-002",
      "applied": false,
      "errorType": "version_conflict",
      "errorMessage": "version conflict: expected 3, got 2"
    }
  ]
}
```

### Synchronization Management Endpoints

#### 5. Get Sync Status
**GET** `/v1/store/sync/status`

Returns the current synchronization status and statistics.

**Response:**
```json
{
  "inProgress": false,
  "lastSyncTime": "2024-01-15T10:29:45Z",
  "lastSyncSuccess": true,
  "productCount": 150,
  "syncDuration": "1.2s",
  "errorMessage": "",
  "eventSync": {
    "lastEventOffset": 1045,
    "consecutiveFailures": 0,
    "fallbackMode": false,
    "nextPollTime": "2024-01-15T10:30:15Z"
  }
}
```

#### 6. Force Synchronization
**POST** `/v1/store/sync/force`

Triggers an immediate full synchronization with the Central API.

**Response:**
```json
{
  "message": "Force sync completed successfully",
  "status": {
    "inProgress": false,
    "lastSyncTime": "2024-01-15T10:30:00Z",
    "lastSyncSuccess": true,
    "productCount": 150,
    "syncDuration": "2.1s"
  }
}
```

#### 7. Get Cache Statistics
**GET** `/v1/store/cache/stats`

Returns detailed statistics about the local cache.

**Response:**
```json
{
  "totalProducts": 150,
  "cacheSize": "2.4MB",
  "lastSyncTime": "2024-01-15T10:29:45Z",
  "lastEventOffset": 1045,
  "initializedAt": "2024-01-15T09:00:00Z",
  "uptime": "1h30m45s",
  "memoryUsage": {
    "allocated": "15.2MB",
    "inUse": "12.8MB"
  }
}
```

## ⚙️ Configuration Reference

### Environment Variables

#### Basic Service Configuration
```bash
PORT=8083                                    # Server port (default: 8083)
ENVIRONMENT=development                      # Environment: development, staging, production
LOG_LEVEL=info                              # Logging level: debug, info, warn, error
API_KEYS=store-s1-key,demo                  # Comma-separated API keys for this store
```

#### Central API Connection
```bash
CENTRAL_API_URL=http://inventory-management-system:8081  # Central API endpoint
CENTRAL_API_KEY=demo                                     # API key for Central API access
```

#### Data Storage
```bash
DATA_DIR=/app/data                          # Directory for local cache persistence
```

#### Event-Driven Synchronization
```bash
SYNC_INTERVAL_SECONDS=30                    # Event polling interval (10-300 seconds)
EVENT_WAIT_TIMEOUT_SECONDS=20               # Long polling timeout (5-60 seconds)
EVENT_BATCH_LIMIT=100                       # Maximum events per request (10-500)
```

#### Legacy Fallback Configuration
```bash
SYNC_INTERVAL_MINUTES=5                     # Full sync interval when in fallback mode
```

### Configuration Examples

#### High-Frequency Store (Busy Store)
```bash
SYNC_INTERVAL_SECONDS=10                    # Poll every 10 seconds
EVENT_WAIT_TIMEOUT_SECONDS=5                # Short timeout for quick updates
EVENT_BATCH_LIMIT=200                       # Handle more events per batch
```

#### Low-Frequency Store (Small Store)
```bash
SYNC_INTERVAL_SECONDS=60                    # Poll every minute
EVENT_WAIT_TIMEOUT_SECONDS=30               # Longer timeout to reduce requests
EVENT_BATCH_LIMIT=50                        # Smaller batches
```

#### Development/Testing
```bash
LOG_LEVEL=debug                             # Verbose logging
SYNC_INTERVAL_SECONDS=5                     # Frequent polling for testing
EVENT_WAIT_TIMEOUT_SECONDS=2                # Quick timeouts
EVENT_BATCH_LIMIT=10                        # Small batches for debugging
```

#### Production
```bash
LOG_LEVEL=info                              # Standard logging
SYNC_INTERVAL_SECONDS=30                    # Balanced polling
EVENT_WAIT_TIMEOUT_SECONDS=20               # Standard timeout
EVENT_BATCH_LIMIT=100                       # Standard batch size
```

## 🏗️ Architecture Details

### Event-Driven Synchronization

The Store API maintains synchronization with the Central API through a sophisticated event-driven system:

#### Event Polling Mechanism
```go
// Continuous polling loop
1. Poll Central API for events from last known offset
2. Use long polling (wait up to EVENT_WAIT_TIMEOUT_SECONDS)
3. Apply received events to local cache
4. Update last processed offset
5. Repeat every SYNC_INTERVAL_SECONDS
```

#### Event Processing Flow
```json
// Event structure from Central API
{
  "offset": 1045,
  "timestamp": "2024-01-15T10:30:00Z",
  "eventType": "inventory_updated",
  "productId": "PROD-001",
  "version": 7,
  "data": {
    "productId": "PROD-001",
    "name": "Wireless Headphones",
    "available": 8,
    "version": 7,
    "lastUpdated": "2024-01-15T10:30:00Z",
    "price": 99.99
  }
}
```

#### Event Types Handled
- **`inventory_updated`**: Product quantity changed
- **`product_created`**: New product added to inventory
- **`product_deleted`**: Product removed from inventory
- **`product_modified`**: Product properties (name, price) changed

### Local Caching Strategy

#### In-Memory Storage with File Persistence
- **Primary Storage**: In-memory map for fast access
- **Persistence**: JSON files for durability across restarts
- **Metadata Tracking**: Last sync time, event offset, product count
- **Thread Safety**: Read/write locks for concurrent access

#### Cache Structure
```go
type MemoryStorage struct {
    products        map[string]models.Product  // Product cache
    lastSyncTime    time.Time                  // Last successful sync
    lastEventOffset int64                      // Last processed event offset
    initializedAt   time.Time                  // Cache initialization time
}
```

#### TTL Management
- **No Explicit TTL**: Cache stays fresh through event-driven updates
- **Staleness Detection**: Monitors sync failures and triggers fallback
- **Automatic Refresh**: Events provide real-time cache updates

### Fallback Mechanisms

#### Circuit Breaker Pattern
```go
// Failure handling with automatic fallback
consecutiveFailures := 0
maxConsecutiveFailures := 5

if consecutiveFailures >= maxConsecutiveFailures {
    // Enter fallback mode
    triggerFullSyncFallback()
}
```

#### Fallback Scenarios
1. **Event Offset Not Found (410 Gone)**
   - Central API may have restarted
   - Triggers immediate full synchronization
   - Resets event offset to current

2. **Consecutive Event Failures**
   - Network issues or API unavailability
   - After 5 consecutive failures, switches to full sync mode
   - Continues attempting event sync in background

3. **Data Consistency Issues**
   - Event sequence gaps detected
   - Possible data loss scenarios
   - Triggers full resynchronization

#### Graceful Degradation
- **Read Operations**: Continue serving from local cache
- **Write Operations**: Return appropriate errors when Central API unavailable
- **Status Reporting**: Clear indication of sync status and issues

### Store-Specific Idempotency Handling

#### Idempotency Key Prefixing
```go
// Store-specific prefix to avoid conflicts between stores
originalKey := "order-12345"
storeSpecificKey := "store-s1-order-12345"

// Ensures each store's operations are isolated
```

#### Benefits of Store-Specific Keys
- **Conflict Prevention**: Same order ID across stores won't conflict
- **Store Isolation**: Each store's operations are independent
- **Debugging**: Easy to identify which store originated an operation
- **Audit Trail**: Clear tracking of operations per store

## 📊 Synchronization Features

### Event Polling Configuration

#### Polling Intervals
```bash
# Aggressive polling (high-traffic stores)
SYNC_INTERVAL_SECONDS=10        # Poll every 10 seconds

# Standard polling (normal stores)
SYNC_INTERVAL_SECONDS=30        # Poll every 30 seconds

# Conservative polling (low-traffic stores)
SYNC_INTERVAL_SECONDS=60        # Poll every minute
```

#### Long Polling Benefits
- **Reduced Latency**: Events delivered within seconds of occurrence
- **Efficient Resource Usage**: Fewer HTTP requests than short polling
- **Configurable Timeout**: Balance between responsiveness and resource usage

### Batch Event Processing

#### Batch Configuration
```bash
EVENT_BATCH_LIMIT=100           # Process up to 100 events per request
```

#### Batch Processing Flow
```go
1. Request events from offset with limit
2. Validate event sequence and consistency
3. Apply all events atomically to local cache
4. Update last processed offset
5. Persist changes to disk
```

#### Error Handling in Batches
- **Partial Failures**: Individual event failures don't stop batch processing
- **Rollback**: Failed batches trigger offset reset and retry
- **Logging**: Detailed logging of batch processing results

### Fallback Full Synchronization

#### When Full Sync Triggers
1. **Initial Startup**: First-time synchronization
2. **Offset Reset**: When event offset is no longer valid
3. **Circuit Breaker**: After consecutive event sync failures
4. **Manual Trigger**: Via force sync endpoint
5. **Data Consistency**: When event gaps are detected

#### Full Sync Process
```go
1. Lock synchronization to prevent concurrent operations
2. Fetch all products from Central API
3. Replace entire local cache atomically
4. Update metadata (sync time, product count)
5. Reset event offset to current
6. Resume event-driven synchronization
```

## 🛠️ Development Guide

### Project Structure
```
packages/backend/services/store-s1/
├── cmd/server/              # Application entry point
├── internal/
│   ├── config/             # Configuration management
│   └── handlers/           # HTTP request handlers
└── shared/                 # Shared components
    ├── client/             # Central API client
    ├── models/             # Data models
    ├── storage/            # Local storage implementation
    ├── sync/               # Synchronization managers
    └── middleware/         # HTTP middleware
```

### Key Components

#### InventoryHandler (`internal/handlers/`)
- HTTP request handling for store endpoints
- Local cache access for read operations
- Proxy functionality for write operations
- Error handling and response formatting

#### EventSyncManager (`shared/sync/`)
- Event-driven synchronization logic
- Circuit breaker implementation
- Fallback mechanism coordination
- Status tracking and reporting

#### MemoryStorage (`shared/storage/`)
- In-memory cache with file persistence
- Thread-safe concurrent access
- Metadata management
- Storage statistics

#### InventoryClient (`shared/client/`)
- HTTP client for Central API communication
- Request/response handling
- Error parsing and propagation
- Health check functionality

### Local Development Setup

#### Prerequisites
```bash
# Go 1.21 or later
go version

# Central API running (for integration)
curl http://localhost:8081/health
```

#### Development Workflow
```bash
# 1. Clone and setup
git clone <repository>
cd packages/backend/services/store-s1

# 2. Install dependencies
go mod download

# 3. Setup environment
cp .env.example .env
# Edit .env with your settings

# 4. Run in development mode
go run cmd/server/main.go

# 5. Test endpoints
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/inventory
```

### Testing Procedures

#### Unit Tests
```bash
# Run unit tests
go test ./internal/...

# Run with coverage
go test -cover ./internal/...

# Test specific component
go test ./internal/handlers/
```

#### Integration Tests
```bash
# Test with real Central API
export CENTRAL_API_URL=http://localhost:8081
export CENTRAL_API_KEY=demo
go test -tags=integration ./tests/

# Test synchronization
go test ./shared/sync/ -v
```

#### Manual Testing
```bash
# Test sync status
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/status

# Test force sync
curl -X POST -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/force

# Test cache stats
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/cache/stats
```

## 🔧 Troubleshooting

### Common Issues

#### Sync Failures
**Symptom**: `lastSyncSuccess: false` in sync status
**Causes**:
- Central API unavailable
- Network connectivity issues
- Authentication problems

**Solutions**:
```bash
# Check Central API connectivity
curl http://central-api:8081/health

# Verify API key
curl -H "X-API-Key: your-key" http://central-api:8081/v1/inventory

# Force resync
curl -X POST -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/force
```

#### Event Offset Issues
**Symptom**: `410 Gone` errors in logs, frequent full syncs
**Cause**: Central API restarted or event queue rotated
**Solution**: Automatic - system will trigger full sync and reset offset

#### High Memory Usage
**Symptom**: Increasing memory consumption
**Causes**: Large product catalog, memory leaks
**Solutions**:
- Monitor cache stats: `GET /v1/store/cache/stats`
- Restart service if memory usage is excessive
- Consider reducing `EVENT_BATCH_LIMIT` for large catalogs

#### Stale Cache Data
**Symptom**: Local cache doesn't match Central API
**Causes**: Event sync failures, network issues
**Solutions**:
```bash
# Check sync status
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/status

# Force full sync
curl -X POST -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/force

# Compare with Central API
curl -H "X-API-Key: demo" http://central-api:8081/v1/inventory
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/inventory
```

### Debug Commands

#### Health and Status
```bash
# Service health
curl http://localhost:8083/health

# Sync status with details
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/status

# Cache statistics
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/cache/stats
```

#### Sync Management
```bash
# Force immediate sync
curl -X POST -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/force

# Check specific product in cache
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/inventory/PROD-001
```

#### Log Analysis
```bash
# Monitor sync events
docker logs store-s1 | grep "event sync"

# Monitor errors
docker logs store-s1 | grep "ERROR"

# Monitor sync status changes
docker logs store-s1 | grep "sync status"
```

### Performance Tuning

#### Sync Frequency Optimization
```bash
# High-traffic stores (frequent updates)
SYNC_INTERVAL_SECONDS=10
EVENT_WAIT_TIMEOUT_SECONDS=5

# Low-traffic stores (infrequent updates)
SYNC_INTERVAL_SECONDS=60
EVENT_WAIT_TIMEOUT_SECONDS=30
```

#### Batch Size Optimization
```bash
# Large catalogs (many products)
EVENT_BATCH_LIMIT=200

# Small catalogs (few products)
EVENT_BATCH_LIMIT=50
```

#### Memory Optimization
- Monitor cache size via `/v1/store/cache/stats`
- Consider periodic restarts for very large catalogs
- Implement cache size limits if needed

---

**🚀 Ready to deploy?** The Store API provides fast, reliable local caching with real-time synchronization, ensuring your store has immediate access to inventory data while maintaining consistency with the central system!

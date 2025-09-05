# Central Inventory Management API

The Central Inventory Management API serves as the **single source of truth** for all inventory operations in the distributed system. It provides high-performance, fault-tolerant inventory management with advanced features including Optimistic Concurrency Control (OCC), event-driven architecture, and comprehensive observability.

## 🎯 API Overview

### Role in the Distributed System
- **Single Source of Truth**: Authoritative inventory data for all stores
- **Event Publisher**: Publishes inventory changes to store APIs via event streaming
- **Concurrency Manager**: Handles concurrent updates using OCC with version-based conflict resolution
- **Admin Interface**: Provides administrative operations for product management

### Key Capabilities
- ✅ **Optimistic Concurrency Control** with version-based conflict detection
- ✅ **Idempotency** with TTL-based caching to prevent duplicate operations
- ✅ **Event-Driven Architecture** with real-time event streaming and long polling
- ✅ **Rate Limiting** with IP-based and admin-specific controls
- ✅ **Comprehensive Observability** with structured logging and OpenTelemetry metrics
- ✅ **Worker Pool Architecture** for high-throughput concurrent processing

## 🚀 Quick Start Guide

### Running with Docker (Recommended)
```bash
# Build the container
docker build -t central-inventory-api .

# Run with default configuration
docker run -p 8081:8081 -p 9080:9080 central-inventory-api

# Run with custom environment variables
docker run -p 8081:8081 -p 9080:9080 \
  -e LOG_LEVEL=debug \
  -e INVENTORY_WORKER_COUNT=8 \
  -e RATE_LIMIT_REQUESTS_PER_MINUTE=200 \
  central-inventory-api
```

### Running with Go (Development)
```bash
# Install dependencies
go mod download

# Set environment variables (optional)
export PORT=8081
export LOG_LEVEL=debug
export API_KEYS=demo,dev-key
export ADMIN_API_KEYS=admin-demo,admin-dev-key

# Run the server
go run cmd/server/main.go

# Or build and run
go build -o inventory-api cmd/server/main.go
./inventory-api
```

### Health Check
```bash
# Verify the API is running
curl http://localhost:8081/health

# Check metrics endpoint
curl http://localhost:9080/metrics
```

## 📚 API Endpoints Documentation

### Authentication
All endpoints require an API key via the `X-API-Key` header:
```bash
# Regular endpoints
X-API-Key: demo

# Admin endpoints (require admin API key)
X-API-Key: admin-demo
```

### Inventory Endpoints (`/v1/inventory/*`)

#### 1. Update Inventory
**POST** `/v1/inventory/updates`

Updates product inventory with OCC and idempotency support.

**Single Update Request:**
```json
{
  "storeId": "store-s1",
  "productId": "PROD-001",
  "delta": -2,
  "version": 5,
  "idempotencyKey": "store-s1-order-12345"
}
```

**Batch Update Request:**
```json
{
  "storeId": "store-s1",
  "updates": [
    {
      "productId": "PROD-001",
      "delta": -1,
      "version": 5,
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
  "productId": "PROD-001",
  "newQuantity": 8,
  "newVersion": 6,
  "applied": true,
  "lastUpdated": "2024-01-15T10:30:00Z"
}
```

**Error Response (Version Conflict):**
```json
{
  "productId": "PROD-001",
  "applied": false,
  "newQuantity": 10,
  "newVersion": 6,
  "errorType": "version_conflict",
  "errorMessage": "version conflict: expected 6, got 5"
}
```

#### 2. Get Product
**GET** `/v1/inventory/{productId}`

Retrieves detailed information for a specific product.

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

#### 3. List Products
**GET** `/v1/inventory?limit=50&cursor=next_page_token`

Lists products with pagination support.

**Query Parameters:**
- `limit` (optional): Number of items per page (default: 50, max: 100)
- `cursor` (optional): Pagination cursor for next page

**Response:**
```json
{
  "items": [
    {
      "productId": "PROD-001",
      "name": "Wireless Headphones",
      "available": 10,
      "version": 6,
      "lastUpdated": "2024-01-15T10:30:00Z",
      "price": 99.99
    }
  ],
  "nextCursor": "eyJvZmZzZXQiOjUwfQ=="
}
```

#### 4. Event Streaming
**GET** `/v1/inventory/events?offset=0&limit=100&wait=30`

Streams inventory change events with long polling support.

**Query Parameters:**
- `offset` (required): Starting event offset
- `limit` (optional): Maximum events to return (default: 100)
- `wait` (optional): Long polling timeout in seconds (0-60)

**Response:**
```json
{
  "events": [
    {
      "offset": 1001,
      "timestamp": "2024-01-15T10:30:00Z",
      "eventType": "inventory_updated",
      "productId": "PROD-001",
      "version": 6,
      "data": {
        "productId": "PROD-001",
        "name": "Wireless Headphones",
        "available": 8,
        "version": 6,
        "lastUpdated": "2024-01-15T10:30:00Z",
        "price": 99.99
      }
    }
  ],
  "nextOffset": 1002,
  "hasMore": false,
  "count": 1
}
```

### Admin Endpoints (`/v1/admin/*`)

#### 1. Create Products
**POST** `/v1/admin/products/create`

Creates new products in the inventory.

**Request:**
```json
{
  "products": [
    {
      "productId": "PROD-NEW-001",
      "name": "New Product",
      "available": 100,
      "price": 29.99
    }
  ]
}
```

**Response:**
```json
{
  "results": [
    {
      "productId": "PROD-NEW-001",
      "success": true,
      "newVersion": 1,
      "lastUpdated": "2024-01-15T10:30:00Z"
    }
  ],
  "summary": {
    "totalRequests": 1,
    "successfulCreations": 1,
    "failedCreations": 0
  }
}
```

#### 2. Set Product Properties
**PUT** `/v1/admin/products/set`

Updates product properties (name, available quantity, price).

**Request:**
```json
{
  "products": [
    {
      "productId": "PROD-001",
      "name": "Updated Product Name",
      "available": 50,
      "price": 79.99
    }
  ]
}
```

#### 3. Delete Products
**DELETE** `/v1/admin/products/delete`

Deletes products from the inventory.

**Request:**
```json
{
  "productIds": ["PROD-001", "PROD-002"]
}
```

#### 4. Rate Limit Status
**GET** `/v1/admin/rate-limit/status`

Returns current rate limiting status and statistics.

**Response:**
```json
{
  "enabled": true,
  "type": "ip",
  "requestsPerMinute": 100,
  "adminRequestsPerMinute": 50,
  "currentClients": [
    {
      "clientId": "192.168.1.100",
      "requestCount": 45,
      "windowStart": "2024-01-15T10:29:00Z",
      "isBlocked": false
    }
  ]
}
```

## ⚙️ Configuration Reference

### Environment Variables

#### Server Configuration
```bash
PORT=8081                                    # Server port (default: 8080)
LOG_LEVEL=info                              # Logging level: debug, info, warn, error
ENVIRONMENT=development                      # Environment: development, staging, production
```

#### Authentication
```bash
API_KEYS=demo,central-api-key               # Comma-separated regular API keys
ADMIN_API_KEYS=admin-demo,admin-central-key # Comma-separated admin API keys
```

#### Worker Pool & Performance
```bash
INVENTORY_WORKER_COUNT=4                    # Number of worker goroutines (1-10)
INVENTORY_QUEUE_BUFFER_SIZE=500             # Update queue buffer size (100-1000)
```

#### Data Persistence
```bash
DATA_PATH=./data/inventory_test_data.json   # Inventory data file path
ENABLE_JSON_PERSISTENCE=true               # Enable file persistence (true/false)
```

#### Caching & Idempotency
```bash
IDEMPOTENCY_CACHE_TTL=2m                   # TTL for idempotency cache (e.g., 1m, 5m)
IDEMPOTENCY_CACHE_CLEANUP_INTERVAL=30s     # Cache cleanup interval
```

#### Event System
```bash
MAX_EVENTS_IN_QUEUE=10000                  # Maximum events in memory
EVENTS_FILE_PATH=./data/events.json        # Events persistence file
```

#### Rate Limiting
```bash
RATE_LIMIT_ENABLED=true                    # Enable rate limiting (true/false)
RATE_LIMIT_TYPE=ip                         # Type: ip, global, both
RATE_LIMIT_REQUESTS_PER_MINUTE=100         # Regular endpoint limit
RATE_LIMIT_WINDOW_MINUTES=1                # Rate limit window
RATE_LIMIT_ADMIN_REQUESTS_PER_MINUTE=50    # Admin endpoint limit
```

### Configuration Examples

#### High-Performance Setup
```bash
INVENTORY_WORKER_COUNT=8
INVENTORY_QUEUE_BUFFER_SIZE=1000
RATE_LIMIT_REQUESTS_PER_MINUTE=500
MAX_EVENTS_IN_QUEUE=50000
```

#### Development Setup
```bash
LOG_LEVEL=debug
ENVIRONMENT=development
RATE_LIMIT_ENABLED=false
ENABLE_JSON_PERSISTENCE=true
```

#### Production Setup
```bash
LOG_LEVEL=info
ENVIRONMENT=production
INVENTORY_WORKER_COUNT=6
RATE_LIMIT_ENABLED=true
RATE_LIMIT_TYPE=both
IDEMPOTENCY_CACHE_TTL=5m
```

## 🏗️ Architecture Details

### Optimistic Concurrency Control (OCC) Implementation

The Central API implements OCC using version-based conflict detection:

#### Version Management
- Each product has a `version` field that increments with every change
- Clients must provide the expected version when making updates
- Updates fail with `version_conflict` error if versions don't match

#### OCC Flow
```go
// 1. Client reads product with current version
GET /v1/inventory/PROD-001
// Response: {"productId": "PROD-001", "available": 10, "version": 5}

// 2. Client submits update with expected version
POST /v1/inventory/updates
{
  "productId": "PROD-001",
  "delta": -2,
  "version": 5,  // Must match current version
  "idempotencyKey": "order-12345"
}

// 3. API validates version and applies update atomically
// Success: version incremented to 6
// Conflict: returns current version and quantity for retry
```

#### Product-Level Locking
- Fine-grained locking per product ID (not global locking)
- Read operations use read locks for concurrent access
- Write operations use write locks for exclusive access
- Prevents deadlocks and maximizes concurrency

### Event-Driven Architecture

#### Event Queue System
- **In-Memory Queue**: High-performance event storage with configurable rotation
- **File Persistence**: Events persisted to disk for durability
- **Long Polling**: Clients can wait for new events (0-60 seconds)
- **Offset-Based**: Sequential event ordering with offset tracking

#### Event Types
```json
{
  "eventType": "inventory_updated",    // Product quantity changed
  "eventType": "product_created",      // New product added
  "eventType": "product_deleted",      // Product removed
  "eventType": "product_modified"      // Product properties changed
}
```

#### Event Publishing Flow
```go
// 1. Inventory update processed successfully
// 2. Event published to queue asynchronously
// 3. Store APIs poll for events via /v1/inventory/events
// 4. Store APIs update local cache based on events
```

### Idempotency Handling

#### TTL-Based Cache
- Idempotency keys cached with configurable TTL (default: 2 minutes)
- Automatic cleanup of expired entries
- Thread-safe concurrent access

#### Idempotency Flow
```go
// 1. Check if idempotency key exists in cache
if cachedResult := cache.Get(idempotencyKey); cachedResult != nil {
    return cachedResult // Return cached result immediately
}

// 2. Process update operation
result := processUpdate(request)

// 3. Cache result for future identical requests
cache.Set(idempotencyKey, result, TTL)
```

#### Key Generation Best Practices
```bash
# Store-specific prefix to avoid conflicts
"store-s1-order-12345"
"store-s2-purchase-67890"

# Include operation context
"store-s1-restock-batch-001"
"admin-bulk-update-20240115"
```

### Rate Limiting Mechanisms

#### IP-Based Rate Limiting
- Tracks requests per IP address
- Sliding window algorithm
- Configurable requests per minute
- Automatic reset after window expires

#### Admin Rate Limiting
- Separate limits for admin endpoints
- Typically lower limits due to higher resource usage
- Independent of regular endpoint limits

#### Rate Limiting Types
```bash
# IP-based: Limit per client IP
RATE_LIMIT_TYPE=ip

# Global: Limit total requests across all clients
RATE_LIMIT_TYPE=global

# Both: Apply whichever limit is hit first
RATE_LIMIT_TYPE=both
```

## 📊 Observability Features

### Structured Logging
- **JSON Format**: Machine-readable logs with structured fields
- **Contextual Information**: Request ID, client IP, operation details
- **Log Levels**: debug, info, warn, error with configurable filtering
- **Performance Metrics**: Request duration, queue sizes, worker utilization

#### Log Examples
```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Inventory update processed successfully",
  "product_id": "PROD-001",
  "delta": -2,
  "new_quantity": 8,
  "new_version": 6,
  "idempotency_key": "store-s1-order-12345",
  "processing_time_ms": 15
}
```

### OpenTelemetry Metrics

#### Business Metrics
- `inventory_api_requests_total`: Total API requests by endpoint and method
- `inventory_api_errors_total`: Error count by status code and endpoint
- `inventory_api_request_duration_seconds`: Request latency histograms
- `inventory_updates_processed_total`: Successful inventory updates
- `inventory_version_conflicts_total`: OCC version conflicts

#### System Metrics
- `inventory_worker_queue_size`: Current queue size
- `inventory_worker_active_count`: Active worker goroutines
- `inventory_cache_hits_total`: Idempotency cache hit rate
- `inventory_events_published_total`: Events published to queue
- `inventory_events_queue_size`: Current event queue size

#### Client Metrics (Advanced)
- `inventory_api_requests_by_client_ip_type`: Requests by IP type (external/internal/localhost)
- `inventory_rate_limit_violations_total`: Rate limiting violations by IP type
- `inventory_api_response_time_by_client_type`: Response time by client type

### Monitoring Integration
- **Prometheus**: Metrics scraping endpoint at `:9080/metrics`
- **Grafana**: Pre-built dashboard with 33 panels
- **Health Checks**: Comprehensive health endpoint with dependency status
- **Alerting**: Ready-to-use alert rules for critical metrics

## 🛠️ Development Guide

### Project Structure
```
packages/backend/services/inventory-management-system/
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP request handlers
│   ├── services/        # Business logic layer
│   ├── events/          # Event queue implementation
│   ├── middleware/      # HTTP middleware (auth, rate limiting)
│   ├── models/          # Request/response models
│   ├── cache/           # TTL cache implementation
│   └── telemetry/       # Observability and metrics
├── data/                # Sample data and persistence
├── monitoring/          # Grafana dashboards and configs
├── tests/               # Test suites
└── docs/                # Additional documentation
```

### Key Components

#### InventoryService (`internal/services/`)
- Core business logic for inventory operations
- OCC implementation with product-level locking
- Worker pool for concurrent processing
- Idempotency cache management

#### EventQueue (`internal/events/`)
- Event publishing and streaming
- File persistence with rotation
- Long polling support
- Offset-based event ordering

#### Middleware (`internal/middleware/`)
- API key authentication
- Rate limiting enforcement
- Request logging and metrics
- CORS handling

### Testing Procedures

#### Unit Tests
```bash
# Run all unit tests
go test ./internal/...

# Run with coverage
go test -cover ./internal/...

# Run specific test suite
go test ./internal/services/
```

#### Integration Tests
```bash
# Run integration tests
go test ./tests/integration/

# Test with real dependencies
docker-compose -f docker-compose.test.yml up -d
go test ./tests/integration/ -tags=integration
```

#### Load Testing
```bash
# Run load tests
go test ./tests/load/

# Custom load test
./tests/load/run_load_test.sh --concurrent=50 --duration=60s
```

### Deployment Considerations

#### Container Deployment
- Multi-stage Docker build for optimized images
- Non-root user for security
- Health checks for orchestration
- Volume mounts for data persistence

#### Environment-Specific Configs
```bash
# Development
LOG_LEVEL=debug
RATE_LIMIT_ENABLED=false

# Staging
LOG_LEVEL=info
INVENTORY_WORKER_COUNT=4

# Production
LOG_LEVEL=warn
INVENTORY_WORKER_COUNT=8
RATE_LIMIT_ENABLED=true
```

#### Scaling Considerations
- Horizontal scaling requires external event queue (Redis/RabbitMQ)
- Database migration for multi-instance deployments
- Load balancer configuration for sticky sessions
- Shared cache for idempotency across instances

## 🔧 Troubleshooting

### Common Issues

#### Version Conflicts
**Symptom**: High rate of `version_conflict` errors
**Cause**: Concurrent updates or stale client data
**Solution**:
- Implement exponential backoff retry logic
- Reduce update frequency
- Check client-side caching strategies

#### High Memory Usage
**Symptom**: Increasing memory consumption
**Cause**: Large event queue or cache growth
**Solution**:
- Reduce `MAX_EVENTS_IN_QUEUE`
- Lower `IDEMPOTENCY_CACHE_TTL`
- Monitor with `go_memstats_*` metrics

#### Rate Limiting Issues
**Symptom**: Clients receiving 429 errors
**Cause**: Rate limits too restrictive
**Solution**:
- Increase `RATE_LIMIT_REQUESTS_PER_MINUTE`
- Check client request patterns
- Consider `RATE_LIMIT_TYPE=global` for burst handling

#### Event Queue Lag
**Symptom**: Store APIs not receiving events promptly
**Cause**: Event queue overflow or processing delays
**Solution**:
- Increase `INVENTORY_WORKER_COUNT`
- Monitor `inventory_events_queue_size` metric
- Check event processing performance

### Debug Commands
```bash
# Check current configuration
curl http://localhost:8081/health

# Monitor metrics
curl http://localhost:9080/metrics | grep inventory_

# Check rate limit status (admin key required)
curl -H "X-API-Key: admin-demo" http://localhost:8081/v1/admin/rate-limit/status

# View recent events
curl -H "X-API-Key: demo" "http://localhost:8081/v1/inventory/events?offset=0&limit=10"
```

### Performance Tuning
- **Worker Count**: Start with CPU cores × 2, adjust based on load
- **Queue Buffer**: Increase for high-throughput scenarios
- **Cache TTL**: Balance between memory usage and idempotency protection
- **Event Queue Size**: Size based on expected event volume and retention needs

---

**🚀 Ready to integrate?** The Central API provides the foundation for your distributed inventory system with enterprise-grade reliability and performance!

# Inventory API Testing Guide

This guide shows how to test the implemented endpoints of the inventory API v1.

## Setup

The server should be running on `http://localhost:8080` with the command:
```bash
go run ./cmd/server
```

All endpoints require authentication with the header `X-API-Key: demo`.

## Available Test Data

The following products are available in the `data/inventory_test_data.json` file:

- **SKU-001**: Smartphone Samsung Galaxy S24 (45 unidades)
- **SKU-002**: Laptop Dell XPS 13 (23 unidades)
- **SKU-003**: Auriculares Sony WH-1000XM5 (67 unidades)
- **SKU-004**: Tablet iPad Air (12 unidades)
- **SKU-005**: Smart TV LG OLED 55" (8 unidades)
- **SKU-006**: Consola PlayStation 5 (0 unidades - agotado)
- **SKU-007**: Cámara Canon EOS R6 (15 unidades)
- **SKU-008**: Smartwatch Apple Watch Series 9 (34 unidades)
- **SKU-009**: Altavoz Bluetooth JBL Charge 5 (89 unidades)
- **SKU-010**: Monitor Gaming ASUS ROG 27" (19 unidades)

## Implemented Endpoints

### 1. POST /v1/inventory/updates - Update Inventory (Single or Batch)

Updates inventory for one or multiple products. Supports both single product updates and batch operations in the same endpoint.

#### Single Product Update

**Example:**
```bash
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8080/v1/inventory/updates \
  -d '{
    "storeId": "store-1",
    "productId": "SKU-001",
    "delta": 5,
    "version": 12,
    "idempotencyKey": "single-update-1"
  }'
```

**Single Update Response (200):**
```json
{
  "productId": "SKU-001",
  "newQuantity": 20,
  "newVersion": 13,
  "applied": true,
  "lastUpdated": "2025-09-02T10:00:00Z"
}
```

#### Batch Product Updates

**Example:**
```bash
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8080/v1/inventory/updates \
  -d '{
    "storeId": "store-1",
    "updates": [
      {
        "productId": "SKU-001",
        "delta": 5,
        "version": 12,
        "idempotencyKey": "batch-1-sku001"
      },
      {
        "productId": "SKU-002",
        "delta": -2,
        "version": 8,
        "idempotencyKey": "batch-1-sku002"
      }
    ]
  }'
```

**Batch Update Response (200):**
```json
{
  "results": [
    {
      "productId": "SKU-001",
      "newQuantity": 50,
      "newVersion": 13,
      "applied": true,
      "lastUpdated": "2025-09-02T10:00:00Z"
    },
    {
      "productId": "SKU-002",
      "newQuantity": 21,
      "newVersion": 9,
      "applied": true,
      "lastUpdated": "2025-09-02T10:00:00Z"
    }
  ],
  "summary": {
    "total": 2,
    "succeeded": 2,
    "failed": 0
  }
}
```

### 2. GET /v1/inventory/{productId} - Get Product

Retrieves information for a specific product, including total availability across all stores.

**Example - Existing product:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory/SKU-001
```

**Successful response (200):**
```json
{
  "productId": "SKU-001",
  "available": 45,
  "version": 12,
  "lastUpdated": "2025-09-02T10:30:00Z"
}
```

**Note:** The `available` field represents the total inventory across all stores in the centralized system.

**Example - Non-existing product:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory/SKU-999
```

**Error response (404):**
```json
{
  "code": "not_found",
  "message": "Product not found: SKU-999"
}
```

**Example - Out of stock product:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory/SKU-006
```

**Response (200):**
```json
{
  "productId": "SKU-006",
  "available": 0,
  "version": 18,
  "lastUpdated": "2025-09-02T12:00:00Z"
}
```

### 3. GET /v1/inventory - List Products with Replication Support

Retrieves a list of all products with optional pagination and replication capabilities. This unified endpoint replaces the separate replication endpoints and supports multiple query parameters for different use cases.

#### Standard Product Listing

**Example - All products:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory
```

**Example - With limit:**
```bash
curl -H 'X-API-Key: demo' "http://localhost:8080/v1/inventory?limit=3"
```

**Standard Response (200):**
```json
{
  "items": [
    {
      "productId": "SKU-008",
      "available": 34,
      "version": 11,
      "lastUpdated": "2025-09-02T09:15:00Z"
    },
    {
      "productId": "SKU-010",
      "available": 19,
      "version": 5,
      "lastUpdated": "2025-09-02T08:45:00Z"
    }
  ],
  "nextCursor": ""
}
```

#### Replication Snapshot (replaces /replication/snapshot)

**Example:**
```bash
curl -H 'X-API-Key: demo' "http://localhost:8080/v1/inventory?snapshot=true"
```

**Snapshot Response (200):**
```json
{
  "state": {
    "SKU-001": {
      "productId": "SKU-001",
      "available": 45,
      "version": 12,
      "lastUpdated": "2025-09-02T10:30:00Z"
    },
    "SKU-002": {
      "productId": "SKU-002",
      "available": 23,
      "version": 8,
      "lastUpdated": "2025-09-02T09:45:00Z"
    }
  },
  "lastOffset": 1287,
  "timestamp": "2025-09-02T12:00:00Z",
  "total": 10
}
```

#### Replication Changes (replaces /replication/changes)

**Example:**
```bash
curl -H 'X-API-Key: demo' "http://localhost:8080/v1/inventory?since=1200&limit=100"
```

**Changes Response (200):**
```json
{
  "events": [
    {
      "seq": 1287,
      "type": "StockChanged",
      "productId": "SKU-001",
      "storeId": "store-1",
      "delta": 1,
      "newVersion": 13,
      "timestamp": "2025-09-02T12:00:00Z"
    }
  ],
  "nextOffset": 1287,
  "hasMore": false,
  "timestamp": "2025-09-02T12:00:00Z"
}
```

#### Replication Format

**Example:**
```bash
curl -H 'X-API-Key: demo' "http://localhost:8080/v1/inventory?format=replication&limit=5"
```

**Replication Format Response (200):**
```json
{
  "items": [
    {
      "productId": "SKU-001",
      "available": 45,
      "version": 12,
      "lastUpdated": "2025-09-02T10:30:00Z"
    }
  ],
  "nextCursor": "",
  "metadata": {
    "lastOffset": 1287,
    "totalProducts": 10,
    "lastUpdated": "2025-09-02T12:00:00Z"
  }
}
```

#### Query Parameters

| Parameter | Description | Example | Use Case |
|-----------|-------------|---------|----------|
| `limit` | Maximum number of items to return | `?limit=50` | Pagination control |
| `cursor` | Pagination cursor for next page | `?cursor=abc123` | Standard pagination |
| `snapshot` | Return full state snapshot | `?snapshot=true` | Initial replication sync |
| `since` | Return changes since offset | `?since=1200` | Incremental replication |
| `format` | Response format | `?format=replication` | Include metadata for replication |
| `longPollSeconds` | Long polling timeout (with since) | `?since=1200&longPollSeconds=30` | Real-time change detection |

## Recommended Test Cases

### Successful Cases
1. **Single product update**: Update one product with delta and version
2. **Batch product updates**: Update multiple products in one request
3. **Product with stock**: `SKU-001`, `SKU-003`, `SKU-009`
4. **Product without stock**: `SKU-006`
5. **List with limit**: `?limit=5`
6. **Complete list**: No parameters
7. **Replication snapshot**: `?snapshot=true`
8. **Replication changes**: `?since=1200`
9. **Replication format**: `?format=replication`

### Error Cases
1. **Non-existing product**: `SKU-999`
2. **Empty ID**: `/v1/inventory/`
3. **No authentication**: Omit `X-API-Key` header

### Authentication Validation
```bash
# Without authentication header - should return 401
curl http://localhost:8080/v1/inventory/SKU-001
```

**Expected response (401):**
```json
{
  "code": "unauthorized",
  "message": "API key required"
}
```

## Health Check (Unversioned)

The health check endpoint doesn't require authentication or versioning:

```bash
curl http://localhost:8080/health
```

**Response (200):**
```json
{
  "status": "healthy"
}
```

## Technical Notes

- **Versioning**: All business endpoints use the `/v1/` prefix
- **Authentication**: `X-API-Key` header required for versioned endpoints
- **Format**: All responses are JSON
- **Status Codes**: 200 (success), 400 (bad request), 401 (unauthorized), 404 (not found), 500 (internal error)
- **Data**: Loaded from `data/inventory_test_data.json` when server starts
- **Inventory Model**: Centralized inventory system - all stores interact with the same product inventory totals
- **Availability**: The `available` field represents total inventory across all stores, not per-store breakdowns
- **Metadata**: System includes metadata for replication (lastOffset), caching (lastUpdated), and monitoring (totalProducts)
- **Store Management**: Store information is not managed by this inventory API - handled by separate store management services
- **Unified Replication**: Single `/v1/inventory` endpoint handles all replication needs through query parameters
- **Query Parameter Flexibility**: Multiple parameters can be combined for specific use cases

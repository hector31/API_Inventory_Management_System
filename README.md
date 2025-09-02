# Inventory Management API

A distributed inventory management system API built for the technical interview exercise. This server implements all the endpoints defined in the Architecture_brief.md with basic placeholder functionality.

## Quick Start

### Prerequisites
- Go 1.22 or higher
- Git

### Installation and Setup

1. Navigate to the project directory:
```bash
cd API_Inventory_Management_System
```

2. Install dependencies:
```bash
make deps
# or manually: go mod tidy
```

3. Run the server:
```bash
make run
# or manually: go run ./cmd/server
```

The server will start on port 8080 by default.

## API Endpoints

All endpoints require authentication via `X-API-Key` header. For testing, use `demo` as the API key.

### Central Inventory API

#### Mutate Stock
```bash
POST /inventory/updates
Content-Type: application/json
X-API-Key: demo

{
  "storeId": "store-7",
  "productId": "SKU-123",
  "delta": -1,
  "version": 7,
  "idempotencyKey": "unique-key-123"
}
```

#### Bulk Sync
```bash
POST /inventory/sync
Content-Type: application/json
X-API-Key: demo

{
  "storeId": "store-7",
  "mode": "merge",
  "products": [
    {"id": "SKU-1", "qty": 20, "version": 3}
  ]
}
```

#### Read Product
```bash
GET /inventory/{productId}
X-API-Key: demo
```

#### Global Availability
```bash
GET /inventory/global/{productId}
X-API-Key: demo
```

#### List Products
```bash
GET /inventory?cursor=&limit=50
X-API-Key: demo
```

### Replication API

#### Get Snapshot
```bash
GET /replication/snapshot
X-API-Key: demo
```

#### Get Changes
```bash
GET /replication/changes?fromOffset=1287&limit=500&longPollSeconds=20
X-API-Key: demo
```



### Health Check
```bash
GET /health
# No authentication required
```

## Testing the API

### Example curl commands:

1. **Create/Update inventory:**
```bash
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST http://localhost:8080/inventory/updates \
  -d '{"storeId":"store-7","productId":"SKU-123","delta":5,"version":0,"idempotencyKey":"test-key-1"}'
```

2. **Read product:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/inventory/SKU-123
```

3. **Get global availability:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/inventory/global/SKU-123
```

4. **List products:**
```bash
curl -H 'X-API-Key: demo' http://localhost:8080/inventory
```

5. **Health check:**
```bash
curl http://localhost:8080/health
```

## Current Implementation Status

This is a **basic server setup** with route definitions and placeholder responses. The current implementation includes:

✅ **Completed:**
- All Central Inventory API endpoints defined in Architecture_brief.md
- Proper HTTP methods (GET, POST)
- Authentication middleware (X-API-Key header)
- Structured request/response types
- Error handling with standard error format
- JSON request/response handling
- Route parameter extraction
- Query parameter handling
- Clean project structure following Go best practices
- Separation of concerns with proper package organization

🚧 **Placeholder/Mock Responses:**
- All endpoints return static placeholder data
- No actual inventory state management
- No persistence layer
- No optimistic concurrency control (OCC)
- No idempotency handling
- No replication logic

## Next Steps for Full Implementation

1. **State Management:** Implement in-memory inventory state
2. **Persistence:** Add JSON file-based persistence (events.log.jsonl, state.json)
3. **Concurrency Control:** Implement optimistic concurrency control (OCC)
4. **Idempotency:** Add idempotency key handling
5. **Replication:** Implement event-based replication system
6. **Store Node:** Create separate store node server (cmd/store)
7. **Observability:** Add OpenTelemetry integration
8. **Testing:** Add comprehensive unit and integration tests
9. **Service Layer:** Add business logic layer between handlers and data

## Project Structure

The project follows Go best practices and standard project layout:

```
API_Inventory_Management_System/
├── cmd/
│   └── server/          # Application entry point
│       └── main.go      # Main server application
├── internal/            # Private application code
│   ├── handlers/        # HTTP request handlers
│   │   ├── inventory.go # Inventory management endpoints
│   │   ├── replication.go # Replication endpoints
│   │   └── health.go    # Health check endpoint
│   ├── middleware/      # HTTP middleware
│   │   └── auth.go      # Authentication middleware
│   └── models/          # Data models and types
│       └── types.go     # Request/response types
├── docs/                # Documentation
├── go.mod              # Go module definition
├── Makefile            # Build and run commands
└── README.md           # This file
```

## Architecture

This server follows the distributed architecture outlined in the Architecture_brief.md:
- **Central Inventory API:** Single source of truth for inventory mutations
- **Replication System:** Event-based synchronization between central and stores
- **Consistency Model:** CP (Consistency + Partition tolerance) for writes, eventual consistency for reads

## Environment Variables

- `PORT`: Server port (default: 8080)
- `API_KEYS`: Comma-separated list of valid API keys (default: demo)

## Make Commands

- `make run`: Start the server
- `make build`: Build the binary
- `make test`: Run tests
- `make deps`: Install dependencies
- `make clean`: Clean build artifacts
- `make fmt`: Format code
- `make help`: Show available commands

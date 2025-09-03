# Project Structure

This document describes the organization and structure of the Inventory Management API project.

## Directory Overview

```
API_Inventory_Management_System/
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── internal/              # Private application code
│   ├── cache/            # Caching implementations
│   ├── config/           # Configuration management
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models and types
│   └── services/         # Business logic services
├── tests/                # All test files organized by type
│   ├── load/            # Load testing files
│   ├── unit/            # Unit tests (future)
│   └── integration/     # Integration tests (future)
├── docs/                # Documentation
├── data/                # Data files
└── Configuration files  # Go modules, environment, etc.
```

## Detailed Structure

### `/cmd/server/`
Main application entry point following Go project layout standards.
- `main.go` - Server initialization and startup

### `/internal/`
Private application code that cannot be imported by other projects.

#### `/internal/cache/`
Caching implementations and utilities.
- `ttl_cache.go` - TTL-based cache with automatic cleanup

#### `/internal/config/`
Configuration management and environment variable handling.
- `config.go` - Configuration loading and validation

#### `/internal/handlers/`
HTTP request handlers for API endpoints.
- `inventory.go` - Inventory management endpoints
- `health.go` - Health check endpoint

#### `/internal/middleware/`
HTTP middleware for cross-cutting concerns.
- `auth.go` - API key authentication
- `logging.go` - Request/response logging
- `cors.go` - CORS handling

#### `/internal/models/`
Data models, types, and structures.
- `types.go` - Request/response types and data models

#### `/internal/services/`
Business logic and service layer implementations.
- `inventory_service.go` - Core inventory business logic
- `product_lock_manager.go` - Fine-grained product locking

### `/tests/`
Organized test files by test type.

#### `/tests/load/`
Load testing files using the `hey` tool.
- `load_test_script.sh` - Automated bash load testing
- `load_test.ps1` - Automated PowerShell load testing
- `load_test_*.json` - Test payload files
- `README.md` - Load testing documentation

#### `/tests/unit/`
Unit tests for individual components (future implementation).
- Will contain `*_test.go` files for each component

#### `/tests/integration/`
Integration tests for API endpoints (future implementation).
- Will contain end-to-end API testing

### `/docs/`
Comprehensive project documentation.
- `API_Testing_Guide.md` - API endpoint testing guide
- `Architecture_brief.md` - System architecture overview
- `Environment_Variables_Guide.md` - Configuration documentation
- `Fine_Grained_Locking_Guide.md` - Concurrency implementation
- `JSON_Persistence_and_TTL_Cache_Guide.md` - Persistence features
- `Load_Testing_Guide.md` - Load testing procedures
- `OCC_and_Idempotency_Guide.md` - OCC and idempotency features
- `Project_Structure.md` - This document
- `Structured_Logging_Guide.md` - Logging implementation
- `Worker_Pool_Configuration_Guide.md` - Worker pool configuration

### `/data/`
Data files and test data.
- `inventory_test_data.json` - Sample inventory data

### Root Files
- `go.mod` / `go.sum` - Go module definition and dependencies
- `.env` / `.env.example` - Environment configuration
- `README.md` - Project overview and quick start
- `Makefile` - Build and development commands

## Design Principles

### Go Project Layout
Follows the standard Go project layout:
- `cmd/` for application entry points
- `internal/` for private application code
- Clear separation of concerns

### Test Organization
Tests are organized by type rather than mirroring source structure:
- `load/` - Performance and concurrency testing
- `unit/` - Component-level testing
- `integration/` - End-to-end testing

### Documentation Structure
Documentation is comprehensive and organized by feature:
- Implementation guides for each major feature
- Testing procedures and examples
- Configuration and deployment guides

## File Naming Conventions

### Go Files
- `snake_case.go` for file names
- `PascalCase` for exported types and functions
- `camelCase` for unexported types and functions

### Test Files
- `*_test.go` for unit tests
- `load_test_*.json` for load test payloads
- `load_test_*.sh/.ps1` for load test scripts

### Documentation
- `PascalCase_With_Underscores.md` for documentation files
- Clear, descriptive names indicating content

## Import Organization

### Internal Imports
```go
import (
    "inventory-management-api/internal/cache"
    "inventory-management-api/internal/config"
    "inventory-management-api/internal/models"
)
```

### External Dependencies
Minimal external dependencies, primarily:
- Standard library packages
- No external frameworks (following Go philosophy)

## Development Workflow

### Running Tests
```bash
# Load tests
cd tests/load && ./load_test_script.sh

# Unit tests (future)
go test ./tests/unit/...

# Integration tests (future)
go test ./tests/integration/...
```

### Building
```bash
# Build server
go build ./cmd/server

# Run server
go run ./cmd/server
```

### Documentation
All major features are documented in `/docs/` with:
- Implementation details
- Usage examples
- Testing procedures
- Configuration options

## Future Expansion

The structure is designed to accommodate future growth:
- Additional services in `/internal/services/`
- New handlers in `/internal/handlers/`
- Comprehensive test suites in `/tests/`
- Feature-specific documentation in `/docs/`

This organization ensures maintainability, testability, and follows Go community best practices.

# Unit Tests

This directory is reserved for unit tests of individual components in the Inventory Management API.

## Purpose

Unit tests focus on testing individual functions, methods, and components in isolation to ensure they work correctly with various inputs and edge cases.

## Planned Test Coverage

### Service Layer (`internal/services/`)
- `inventory_service_test.go` - Inventory service business logic
- `product_lock_manager_test.go` - Product-level locking functionality
- Cache implementations and TTL behavior

### Models (`internal/models/`)
- `types_test.go` - Data model validation and serialization
- Request/response structure testing

### Cache (`internal/cache/`)
- `ttl_cache_test.go` - TTL cache functionality
- Cache expiration and cleanup testing

### Configuration (`internal/config/`)
- `config_test.go` - Configuration loading and validation

## Running Unit Tests

```bash
# Run all unit tests
go test ./tests/unit/...

# Run with coverage
go test -cover ./tests/unit/...

# Run with verbose output
go test -v ./tests/unit/...

# Run specific test
go test ./tests/unit/inventory_service_test.go
```

## Test Structure

Each test file should follow Go testing conventions:
```go
package unit

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestFunctionName(t *testing.T) {
    // Arrange
    // Act  
    // Assert
}
```

## Test Data

Test fixtures and mock data should be placed in:
- `fixtures/` - Static test data files
- `mocks/` - Mock implementations for dependencies

## Best Practices

1. **Test Naming**: Use descriptive names that explain what is being tested
2. **Isolation**: Each test should be independent
3. **Coverage**: Aim for high test coverage of critical business logic
4. **Edge Cases**: Test boundary conditions and error scenarios
5. **Performance**: Include performance-sensitive unit tests

## Future Implementation

Unit tests will be implemented to cover:
- Optimistic Concurrency Control logic
- Idempotency key handling
- Product lock manager functionality
- Cache TTL behavior
- Configuration validation
- Error handling scenarios

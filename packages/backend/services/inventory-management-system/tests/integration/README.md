# Integration Tests

This directory is reserved for integration tests that verify the complete functionality of API endpoints and system interactions.

## Purpose

Integration tests validate that different components work together correctly, including:
- HTTP endpoint functionality
- Request/response handling
- Authentication and authorization
- Database persistence
- Error handling
- End-to-end workflows

## Planned Test Coverage

### API Endpoints
- `inventory_endpoints_test.go` - All inventory API endpoints
- `health_endpoint_test.go` - Health check functionality
- `authentication_test.go` - API key authentication

### Workflows
- `update_workflow_test.go` - Complete update workflow testing
- `concurrency_test.go` - Multi-client concurrency scenarios
- `error_handling_test.go` - Error response validation

### Data Persistence
- `json_persistence_test.go` - File persistence functionality
- `data_consistency_test.go` - Data integrity validation

## Running Integration Tests

```bash
# Start the server first
go run ./cmd/server &
SERVER_PID=$!

# Run integration tests
go test ./tests/integration/...

# Stop the server
kill $SERVER_PID
```

## Test Structure

Integration tests should use the standard Go testing framework with HTTP client testing:

```go
package integration

import (
    "net/http"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestInventoryUpdateEndpoint(t *testing.T) {
    // Setup test server
    // Make HTTP request
    // Validate response
    // Cleanup
}
```

## Test Environment

### Server Configuration
Integration tests should use a dedicated test configuration:
```bash
export PORT=8082
export LOG_LEVEL=debug
export ENABLE_JSON_PERSISTENCE=false
export INVENTORY_WORKER_COUNT=2
```

### Test Data
- `testdata/` - Test JSON files and fixtures
- `setup/` - Test environment setup scripts
- `cleanup/` - Test cleanup utilities

## Test Scenarios

### Basic Functionality
- ✅ GET /health returns 200 OK
- ✅ GET /v1/inventory returns product list
- ✅ GET /v1/inventory/{id} returns specific product
- ✅ POST /v1/inventory/updates processes updates

### Authentication
- ✅ Valid API key allows access
- ✅ Invalid API key returns 401 Unauthorized
- ✅ Missing API key returns 401 Unauthorized

### Error Handling
- ✅ Invalid JSON returns 400 Bad Request
- ✅ Missing fields return appropriate errors
- ✅ Version conflicts return 409 Conflict
- ✅ Non-existent products return 404 Not Found

### Concurrency
- ✅ Multiple clients can update different products
- ✅ Version conflicts are handled correctly
- ✅ Idempotency keys prevent duplicates

### Data Persistence
- ✅ Updates are persisted to JSON file
- ✅ Server restart preserves data
- ✅ File corruption is handled gracefully

## Best Practices

1. **Test Isolation**: Each test should clean up after itself
2. **Realistic Data**: Use realistic test data and scenarios
3. **Error Testing**: Test both success and failure cases
4. **Performance**: Include basic performance validation
5. **Documentation**: Document complex test scenarios

## CI/CD Integration

Integration tests can be automated in CI/CD pipelines:
```bash
# Example CI script
./start_test_server.sh
go test ./tests/integration/... -v
./stop_test_server.sh
```

## Future Implementation

Integration tests will be implemented to cover:
- Complete API endpoint functionality
- Authentication and authorization flows
- Concurrency and race condition scenarios
- Error handling and edge cases
- Data persistence and recovery
- Performance baseline validation

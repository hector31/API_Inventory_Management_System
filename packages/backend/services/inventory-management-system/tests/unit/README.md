# Unit Tests for Inventory Management System

This directory contains comprehensive unit tests for the central inventory management API, organized to mirror the source code structure for easy navigation and maintenance.

## Directory Structure

```
tests/unit/
├── README.md                          # This file
├── testutils/                         # Test utilities and helpers
│   ├── mocks/                         # Mock implementations
│   ├── fixtures/                      # Test data fixtures
│   └── helpers.go                     # Common test helper functions
├── handlers/                          # Tests for HTTP handlers
│   ├── inventory_test.go              # Inventory CRUD operations tests
│   ├── health_test.go                 # Health check endpoint tests
│   └── events_test.go                 # Event handling tests
├── services/                          # Tests for business logic services
│   ├── inventory_service_test.go      # Core inventory service tests
│   ├── product_lock_manager_test.go   # Locking mechanism tests
│   └── validation_test.go             # Business validation tests
├── middleware/                        # Tests for middleware components
│   ├── auth_test.go                   # Authentication middleware tests
│   ├── validation_test.go             # Request validation middleware tests
│   └── error_handling_test.go         # Error handling middleware tests
├── cache/                             # Tests for caching layer
│   └── ttl_cache_test.go              # TTL cache implementation tests
├── events/                            # Tests for event system
│   └── queue_test.go                  # Event queue tests
├── models/                            # Tests for data models
│   └── types_test.go                  # Model validation and serialization tests
├── config/                            # Tests for configuration
│   └── config_test.go                 # Configuration loading and validation tests
└── telemetry/                         # Tests for telemetry and monitoring
    ├── api_test.go                    # API telemetry tests
    ├── middleware_test.go             # Telemetry middleware tests
    └── common_test.go                 # Common telemetry utilities tests
```

## Testing Conventions

### File Naming
- All test files follow the `*_test.go` naming convention
- Test files are placed in the same package as the code they test
- Mock files are placed in `testutils/mocks/` directory

### Test Function Naming
- Test functions follow the pattern: `Test<FunctionName>_<Scenario>`
- Benchmark functions follow the pattern: `Benchmark<FunctionName>`
- Example: `TestUpdateInventory_SuccessfulUpdate`, `TestUpdateInventory_VersionConflict`

### Test Structure
All tests follow the AAA (Arrange, Act, Assert) pattern:
```go
func TestFunction_Scenario(t *testing.T) {
    // Arrange - Set up test data and dependencies

    // Act - Execute the function under test

    // Assert - Verify the results
}
```

## Test Categories

### 1. Happy Path Tests
- Test successful operations with valid inputs
- Verify correct return values and side effects
- Test normal business flows

### 2. Error Handling Tests
- Test invalid inputs and edge cases
- Verify proper error messages and status codes
- Test error propagation and recovery

### 3. Edge Cases and Boundary Conditions
- Test with minimum and maximum values
- Test with empty or null inputs
- Test concurrent access scenarios

### 4. Performance Tests
- Benchmark critical operations
- Test under high load conditions
- Verify memory usage and garbage collection

## Mock Strategy

### External Dependencies
- Database operations are mocked using interfaces
- HTTP clients are mocked for external API calls
- File system operations are mocked for testing

### Internal Dependencies
- Services are mocked when testing handlers
- Repositories are mocked when testing services
- Cache implementations are mocked for isolation

## Test Data Management

### Fixtures
- Common test data is stored in `testutils/fixtures/`
- Fixtures are loaded using helper functions
- Test data is isolated between tests

### Setup and Teardown
- Use `t.Cleanup()` for test cleanup
- Reset global state between tests
- Ensure tests can run in any order

## Running Tests

### Quick Start
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run comprehensive test suite
make test-all

# Run specific package
make test-handlers
make test-services
make test-cache
make test-models
```

### Manual Commands
```bash
# All tests
go test ./tests/unit/...

# Specific package
go test ./tests/unit/handlers/

# With coverage
go test -cover ./tests/unit/...

# With race detection
go test -race ./tests/unit/...

# Benchmarks
go test -bench=. ./tests/unit/...

# Using test script
chmod +x run_tests.sh
./run_tests.sh
```

### Test Environments
Set environment variables to configure test behavior:
```bash
# CI environment (optimized for CI/CD)
TEST_ENVIRONMENT=ci make test

# Development environment (verbose logging)
TEST_ENVIRONMENT=development make test

# Performance testing
TEST_ENVIRONMENT=performance make test-bench
```

# Tests Directory

This directory contains all test files for the Inventory Management API, organized by test type.

## Directory Structure

```
tests/
├── load/           # Load testing files (hey tool, JSON payloads, scripts)
├── unit/           # Unit tests for individual components
├── integration/    # Integration tests for API endpoints
└── README.md       # This file
```

## Test Types

### Load Tests (`load/`)
Performance and concurrency testing using the `hey` tool to verify:
- Fine-grained locking implementation
- Worker pool distribution
- Optimistic Concurrency Control (OCC)
- Version conflict handling
- High-throughput scenarios

### Unit Tests (`unit/`)
Individual component testing for:
- Service layer logic
- Cache implementations
- Lock managers
- Data models
- Utility functions

### Integration Tests (`integration/`)
End-to-end API testing for:
- HTTP endpoint functionality
- Authentication and authorization
- Request/response validation
- Error handling
- Database persistence

## Running Tests

### Load Tests
```bash
# Navigate to load test directory
cd tests/load

# Run automated load tests
./load_test_script.sh        # Linux/Mac
.\load_test.ps1             # Windows PowerShell
```

### Unit Tests
```bash
# Run all unit tests
go test ./tests/unit/...

# Run with coverage
go test -cover ./tests/unit/...
```

### Integration Tests
```bash
# Run integration tests (requires running server)
go test ./tests/integration/...
```

## Test Data

Test data files are organized within each test type directory:
- Load test JSON payloads in `load/`
- Unit test fixtures in `unit/fixtures/`
- Integration test data in `integration/testdata/`

## Best Practices

1. **Isolation**: Each test should be independent and not rely on other tests
2. **Cleanup**: Tests should clean up any resources they create
3. **Naming**: Use descriptive test names that explain what is being tested
4. **Documentation**: Include comments explaining complex test scenarios
5. **Data**: Use realistic test data that represents actual usage patterns

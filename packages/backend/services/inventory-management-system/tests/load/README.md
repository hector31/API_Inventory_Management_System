# Load Testing

This directory contains load testing files for the Inventory Management API, specifically designed to test the fine-grained locking implementation and concurrent processing capabilities.

## ✨ **New Features - Dynamic Load Testing**

### ✅ **Automatic Version Fetching**
The load testing scripts now automatically fetch current product versions from the API before running tests, eliminating the need for manual version updates.

### ✅ **Dynamic Payload Generation**
Test payloads are generated on-the-fly with:
- Current version numbers from the API
- Unique idempotency keys with timestamps
- Proper product IDs and deltas for each test scenario

### ✅ **Repeatable Testing**
Tests can be run multiple times in succession without manual intervention, making them suitable for:
- Continuous Integration (CI/CD) pipelines
- Automated testing workflows
- Repeated performance benchmarking

### ✅ **Error Handling**
Scripts include comprehensive error handling for:
- Server connectivity issues
- API response validation
- Version fetching failures

## Files

### Test Scripts (Updated with Dynamic Functionality)
- `load_test_script.sh` - **Dynamic** Bash script for Linux/Mac load testing
- `load_test.ps1` - **Dynamic** PowerShell script for Windows load testing

### Legacy Test Data (Reference Only)
- `load_test_sku001.json` - SKU-001 update payload template
- `load_test_sku002.json` - SKU-002 update payload template
- `load_test_sku003.json` - SKU-003 update payload template
- `load_test_same_product.json` - Same product conflict testing template
- `load_test_batch_mixed.json` - Batch update template
- `load_test_different_products.json` - Different product template

**Note**: JSON files are now used as templates only. The scripts generate dynamic payloads with current versions automatically.

## Prerequisites

### Install hey tool
```bash
go install github.com/rakyll/hey@latest
```

### Start the server
```bash
# From project root directory
export INVENTORY_WORKER_COUNT=4
export INVENTORY_QUEUE_BUFFER_SIZE=500
export LOG_LEVEL=debug
go run ./cmd/server
```

## Running Load Tests

### ✅ **Automated Dynamic Testing (Recommended)**
```bash
# Navigate to this directory
cd tests/load

# Run all test scenarios with automatic version fetching
./load_test_script.sh        # Linux/Mac
.\load_test.ps1             # Windows PowerShell
```

**What happens automatically:**
1. 🔍 **Server Health Check** - Verifies server is running
2. 📡 **Version Fetching** - Gets current product versions via API calls
3. 🔄 **Dynamic Payload Generation** - Creates test payloads with current versions
4. 🧪 **Test Execution** - Runs all test scenarios with fresh data
5. 🧹 **Cleanup** - Removes temporary files

**Benefits:**
- ✅ No manual version updates required
- ✅ Tests work immediately after any previous test runs
- ✅ Suitable for CI/CD automation
- ✅ Eliminates 409 Conflict errors from outdated versions

### Manual Testing

#### Test 1: Different Products (Concurrent)
```bash
hey -n 100 -c 10 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_sku001.json \
    http://localhost:8081/v1/inventory/updates
```

#### Test 2: Same Product (Version Conflicts)
```bash
hey -n 50 -c 10 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_same_product.json \
    http://localhost:8081/v1/inventory/updates
```

#### Test 3: Batch Operations
```bash
hey -n 30 -c 5 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_batch_mixed.json \
    http://localhost:8081/v1/inventory/updates
```

## Test Scenarios

### Scenario 1: Concurrent Different Products
**Objective**: Verify different products can be updated concurrently
**Expected**: All requests succeed, multiple workers active
**Files**: `load_test_sku001.json`, `load_test_sku002.json`, `load_test_sku003.json`

### Scenario 2: Version Conflicts
**Objective**: Verify OCC handles concurrent updates to same product
**Expected**: Only 1 success, others get 409 Conflict
**Files**: `load_test_same_product.json`

### Scenario 3: Batch Processing
**Objective**: Test batch updates with multiple products
**Expected**: Concurrent processing within batch operations
**Files**: `load_test_batch_mixed.json`

### Scenario 4: High Concurrency Stress
**Objective**: Maximum load testing with multiple products
**Expected**: High throughput, no deadlocks, proper distribution
**Files**: All JSON files used simultaneously

## Expected Results

### Performance Metrics
- **Different Products**: 150-200 req/sec, 100% success rate
- **Same Product**: 50-80 req/sec, 2-10% success rate (due to conflicts)
- **Batch Operations**: 80-120 req/sec, 90-100% success rate
- **High Concurrency**: 200-300 req/sec, 95-100% success rate

### Server Log Indicators
```
level=DEBUG msg="Acquired write lock for product" product_id=SKU-001
level=DEBUG msg="Update processed by worker" worker_id=1 product_id=SKU-001
level=WARN msg="Version conflict detected" product_id=SKU-001
```

### Success Criteria
- ✅ No HTTP 500 errors
- ✅ Worker distribution across all workers
- ✅ Proper version conflict handling (409 responses)
- ✅ Concurrent processing of different products
- ✅ No deadlocks or timeouts

## Troubleshooting

### Common Issues
1. **Server not running**: Start server first with `go run ./cmd/server`
2. **Wrong versions**: Update JSON files with current product versions
3. **hey not found**: Install with `go install github.com/rakyll/hey@latest`
4. **Port conflicts**: Check server is running on correct port (8081)

### Updating Test Data
Before running tests, check current product versions:
```bash
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-001
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-002
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-003
```

Update the `version` fields in JSON files to match current values.

## Integration with CI/CD

These load tests can be integrated into CI/CD pipelines:
```bash
# Example CI script
./tests/load/load_test_script.sh > load_test_results.txt
# Parse results and fail build if performance degrades
```

# Load Testing Guide for Fine-Grained Locking

This guide provides comprehensive load testing procedures to verify the fine-grained locking implementation works correctly under concurrent load.

## Test Files Location

All load testing files are organized in the `tests/load/` directory:
```
tests/load/
├── load_test_script.sh      # Automated bash script
├── load_test.ps1           # Automated PowerShell script
├── load_test_*.json        # Test payload files
└── README.md              # Load testing documentation
```

## Prerequisites

### Install hey tool
```bash
# Install hey load testing tool
go install github.com/rakyll/hey@latest
```

### Start the server with multiple workers
```bash
# Start server with 4 workers for optimal concurrency testing
export INVENTORY_WORKER_COUNT=4
export INVENTORY_QUEUE_BUFFER_SIZE=500
export LOG_LEVEL=debug
go run ./cmd/server
```

## Test Scenarios

### Scenario 1: Different Products (Concurrent Processing)

**Objective**: Verify that updates to different products run concurrently without blocking each other.

**Expected Results**:
- ✅ All requests should succeed (200 OK)
- ✅ Multiple workers should process requests simultaneously
- ✅ High throughput with minimal response time variance

**Commands**:
```bash
# Navigate to load test directory
cd tests/load

# Test SKU-001 updates
hey -n 100 -c 10 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_sku001.json \
    http://localhost:8081/v1/inventory/updates

# Test SKU-002 updates (run simultaneously in another terminal)
hey -n 100 -c 10 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_sku002.json \
    http://localhost:8081/v1/inventory/updates
```

### Scenario 2: Same Product (Version Conflicts)

**Objective**: Verify that multiple updates to the same product are properly serialized with OCC.

**Expected Results**:
- ✅ Only 1 request succeeds initially (200 OK)
- ✅ Subsequent requests get version conflicts (409 Conflict)
- ✅ No data corruption or race conditions
- ✅ Proper error messages in responses

**Command**:
```bash
# Multiple clients updating same product with same version
hey -n 50 -c 10 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_same_product.json \
    http://localhost:8081/v1/inventory/updates
```

### Scenario 3: Batch Updates (Mixed Products)

**Objective**: Test batch operations with multiple products to verify concurrent processing within batches.

**Expected Results**:
- ✅ Batch operations complete successfully
- ✅ Individual products within batch processed concurrently
- ✅ Proper summary statistics in responses

**Command**:
```bash
# Batch updates with mixed products
hey -n 30 -c 5 \
    -m POST \
    -H "X-API-Key: demo" \
    -H "Content-Type: application/json" \
    -D load_test_batch_mixed.json \
    http://localhost:8081/v1/inventory/updates
```

### Scenario 4: High Concurrency Stress Test

**Objective**: Maximum stress test with multiple products and high concurrency.

**Expected Results**:
- ✅ No HTTP 500 errors
- ✅ No deadlocks or timeouts
- ✅ Graceful handling of high load
- ✅ Worker distribution across all available workers

**Commands (run simultaneously)**:
```bash
# Terminal 1: SKU-001 load
hey -n 200 -c 20 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_sku001.json http://localhost:8081/v1/inventory/updates &

# Terminal 2: SKU-002 load  
hey -n 200 -c 20 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_sku002.json http://localhost:8081/v1/inventory/updates &

# Terminal 3: SKU-003 load
hey -n 200 -c 20 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_sku003.json http://localhost:8081/v1/inventory/updates &

# Wait for all to complete
wait
```

## Windows PowerShell Commands

### Scenario 1: Different Products
```powershell
# Terminal 1
hey -n 100 -c 10 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_sku001.json http://localhost:8081/v1/inventory/updates

# Terminal 2 (run simultaneously)
hey -n 100 -c 10 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_sku002.json http://localhost:8081/v1/inventory/updates
```

### Scenario 2: Same Product Conflicts
```powershell
hey -n 50 -c 10 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_same_product.json http://localhost:8081/v1/inventory/updates
```

### Scenario 3: Batch Operations
```powershell
hey -n 30 -c 5 -m POST -H "X-API-Key: demo" -H "Content-Type: application/json" -D load_test_batch_mixed.json http://localhost:8081/v1/inventory/updates
```

## Expected Results Analysis

### Success Metrics

#### Different Products Test
```
Summary:
  Total:        2.5000 secs
  Slowest:      0.0500 secs
  Fastest:      0.0100 secs
  Average:      0.0250 secs
  Requests/sec: 40.00

Status code distribution:
  [200] 100 responses    ✅ All successful
```

#### Same Product Test
```
Summary:
  Total:        3.0000 secs
  Slowest:      0.0800 secs
  Fastest:      0.0150 secs
  Average:      0.0400 secs
  Requests/sec: 16.67

Status code distribution:
  [200] 1 responses     ✅ One success
  [409] 49 responses    ✅ Version conflicts
```

### Server Log Analysis

#### Concurrent Processing (Good)
```
level=DEBUG msg="Acquired write lock for product" product_id=SKU-001
level=DEBUG msg="Acquired write lock for product" product_id=SKU-002  
level=DEBUG msg="Update processed by worker" worker_id=1 product_id=SKU-001
level=DEBUG msg="Update processed by worker" worker_id=2 product_id=SKU-002
level=DEBUG msg="Released write lock for product" product_id=SKU-001
level=DEBUG msg="Released write lock for product" product_id=SKU-002
```

#### Version Conflicts (Expected)
```
level=WARN msg="Version conflict detected" product_id=SKU-001 expected_version=13 provided_version=13
level=INFO msg="Idempotent request detected, returning cached result" idempotency_key=load-test-same-123
```

#### Worker Distribution (Good)
```
level=DEBUG msg="Update processed by worker" worker_id=1 product_id=SKU-001
level=DEBUG msg="Update processed by worker" worker_id=2 product_id=SKU-002
level=DEBUG msg="Update processed by worker" worker_id=3 product_id=SKU-003
level=DEBUG msg="Update processed by worker" worker_id=4 product_id=SKU-001
```

## Troubleshooting

### High Error Rates
- **Symptom**: Many 500 errors
- **Cause**: Possible deadlocks or resource exhaustion
- **Solution**: Check server logs for errors, reduce concurrency

### Poor Performance
- **Symptom**: High response times
- **Cause**: Lock contention or insufficient workers
- **Solution**: Increase worker count, check lock statistics

### Version Conflict Issues
- **Symptom**: Unexpected version conflicts
- **Cause**: Incorrect version numbers in test data
- **Solution**: Update JSON files with current product versions

## Performance Benchmarks

### Expected Performance (4 Workers)

| Test Scenario | Requests/sec | Avg Response Time | Success Rate |
|---------------|--------------|-------------------|--------------|
| Different Products | 150-200 | 20-50ms | 100% |
| Same Product | 50-80 | 30-80ms | 2-10% |
| Batch Mixed | 80-120 | 40-100ms | 90-100% |
| High Concurrency | 200-300 | 50-150ms | 95-100% |

### Monitoring Commands

#### Check Current Product Versions
```bash
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-001
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-002
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-003
```

#### Health Check
```bash
curl http://localhost:8081/health
```

## Test Data Management

### Update Test Files with Current Versions
Before running tests, update the JSON files with current product versions:

1. Check current versions:
```bash
curl -H 'X-API-Key: demo' http://localhost:8081/v1/inventory/SKU-001
```

2. Update JSON files with correct version numbers
3. Ensure unique idempotency keys for each test run

### Reset Test Environment
To reset the test environment:
1. Restart the server
2. Check that JSON file has been restored to original state
3. Verify all products have expected initial versions

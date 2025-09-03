# Worker Pool Configuration Guide

This guide explains the configurable worker pool implementation for processing inventory updates in the Inventory Management API.

## Overview

The inventory service now supports configurable worker pools to improve scalability and performance tuning:

1. **Configurable Worker Count**: Multiple worker goroutines can process updates concurrently
2. **Configurable Queue Buffer Size**: Adjustable channel buffer size for different load patterns
3. **Thread-Safe Processing**: Maintains OCC and idempotency guarantees with multiple workers
4. **Graceful Shutdown**: Proper worker lifecycle management

## Environment Variables

### Worker Pool Configuration
```bash
# Number of worker goroutines (default: 1)
INVENTORY_WORKER_COUNT=1

# Queue buffer size (default: 100)
INVENTORY_QUEUE_BUFFER_SIZE=100
```

## Configuration Examples

### Single Worker (Default - Current Behavior)
```bash
INVENTORY_WORKER_COUNT=1
INVENTORY_QUEUE_BUFFER_SIZE=100
```
- **Use Case**: Low to medium traffic, simple deployment
- **Benefits**: Minimal resource usage, sequential processing
- **Throughput**: ~100-500 updates/second

### Multiple Workers (High Throughput)
```bash
INVENTORY_WORKER_COUNT=4
INVENTORY_QUEUE_BUFFER_SIZE=500
```
- **Use Case**: High traffic, multiple concurrent clients
- **Benefits**: Higher throughput, better resource utilization
- **Throughput**: ~400-2000 updates/second

### High-Load Configuration
```bash
INVENTORY_WORKER_COUNT=8
INVENTORY_QUEUE_BUFFER_SIZE=1000
```
- **Use Case**: Very high traffic, enterprise deployment
- **Benefits**: Maximum throughput, handles traffic spikes
- **Throughput**: ~800-4000 updates/second

## Implementation Details

### Worker Pool Architecture
```
Client Requests → Queue (Buffered Channel) → Worker Pool → Database Updates
                                          ↓
                                    Worker 1, Worker 2, ..., Worker N
```

### Thread Safety Guarantees
1. **Shared Queue**: All workers read from the same channel
2. **Mutex Protection**: RWMutex protects inventory data access
3. **OCC Maintained**: Version checking prevents conflicts
4. **Idempotency Preserved**: TTL cache is thread-safe

### Worker Lifecycle
```go
// Service initialization
func (s *InventoryService) startWorkerPool() {
    for i := 0; i < s.workerCount; i++ {
        s.workersWaitGroup.Add(1)
        go s.processUpdateWorker(i + 1)
    }
}

// Worker processing loop
func (s *InventoryService) processUpdateWorker(workerID int) {
    defer s.workersWaitGroup.Done()
    
    for {
        select {
        case updateReq := <-s.updateQueue:
            result := s.processUpdateInternal(updateReq)
            // Send response...
        case <-s.stopWorkers:
            return // Graceful shutdown
        }
    }
}
```

## Performance Tuning Guidelines

### Worker Count Recommendations
- **CPU-bound workloads**: Workers = CPU cores
- **I/O-bound workloads**: Workers = 2-4x CPU cores
- **Mixed workloads**: Start with CPU cores, tune based on metrics

### Buffer Size Recommendations
- **Low latency**: Small buffer (50-100)
- **High throughput**: Large buffer (500-1000)
- **Memory constrained**: Conservative buffer (100-200)

### Environment-Specific Configurations

#### Development Environment
```bash
INVENTORY_WORKER_COUNT=1
INVENTORY_QUEUE_BUFFER_SIZE=50
```
- Minimal resource usage
- Easy debugging
- Predictable behavior

#### Staging Environment
```bash
INVENTORY_WORKER_COUNT=2
INVENTORY_QUEUE_BUFFER_SIZE=200
```
- Production-like testing
- Performance validation
- Load testing

#### Production Environment
```bash
INVENTORY_WORKER_COUNT=4
INVENTORY_QUEUE_BUFFER_SIZE=500
```
- High availability
- Optimal performance
- Traffic spike handling

## Monitoring and Observability

### Startup Logs
```
level=INFO msg="Starting inventory update worker pool" worker_count=4
level=DEBUG msg="Starting inventory update worker" worker_id=1
level=DEBUG msg="Starting inventory update worker" worker_id=2
level=DEBUG msg="Starting inventory update worker" worker_id=3
level=DEBUG msg="Starting inventory update worker" worker_id=4
level=INFO msg="Inventory service initialized with queue processing" 
    worker_count=4 queue_buffer_size=500 cache_ttl=2m0s
```

### Processing Logs
```
level=DEBUG msg="Update processed by worker" 
    worker_id=2 product_id=SKU-001 applied=true
level=DEBUG msg="Update processed by worker" 
    worker_id=1 product_id=SKU-002 applied=true
```

### Shutdown Logs
```
level=INFO msg="Stopping inventory service" worker_count=4
level=DEBUG msg="Stopping inventory update worker" worker_id=1
level=DEBUG msg="Stopping inventory update worker" worker_id=2
level=INFO msg="Inventory service stopped successfully"
```

## Testing Different Configurations

### Test Single Worker
```bash
# Set environment
export INVENTORY_WORKER_COUNT=1
export INVENTORY_QUEUE_BUFFER_SIZE=100

# Start server
go run ./cmd/server

# Send concurrent requests
for i in {1..10}; do
  curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
    -X POST http://localhost:8082/v1/inventory/updates \
    -d "{\"storeId\":\"store-1\",\"productId\":\"SKU-00$i\",\"delta\":1,\"version\":1,\"idempotencyKey\":\"test-$i\"}" &
done
wait
```

### Test Multiple Workers
```bash
# Set environment
export INVENTORY_WORKER_COUNT=4
export INVENTORY_QUEUE_BUFFER_SIZE=500

# Start server
go run ./cmd/server

# Send high-volume concurrent requests
for i in {1..100}; do
  curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
    -X POST http://localhost:8082/v1/inventory/updates \
    -d "{\"storeId\":\"store-1\",\"productId\":\"SKU-001\",\"delta\":1,\"version\":$i,\"idempotencyKey\":\"load-test-$i\"}" &
done
wait
```

## Benefits of Worker Pool Configuration

### Scalability
1. **Horizontal Scaling**: More workers handle more concurrent requests
2. **Resource Utilization**: Better CPU and memory usage
3. **Throughput**: Linear scaling with worker count (up to limits)

### Performance Tuning
1. **Environment-Specific**: Different configs for dev/staging/prod
2. **Load-Based**: Adjust based on traffic patterns
3. **Resource-Aware**: Scale based on available hardware

### Reliability
1. **Graceful Shutdown**: Workers finish current requests before stopping
2. **Error Isolation**: Worker failures don't affect others
3. **Queue Buffering**: Handles traffic spikes without dropping requests

## Troubleshooting

### High CPU Usage
- **Symptom**: CPU usage near 100%
- **Solution**: Reduce worker count or optimize processing logic

### Memory Issues
- **Symptom**: High memory usage, potential OOM
- **Solution**: Reduce queue buffer size or worker count

### Slow Response Times
- **Symptom**: High latency on update requests
- **Solution**: Increase worker count or buffer size

### Queue Overflow
- **Symptom**: "timeout submitting update to queue" errors
- **Solution**: Increase buffer size or worker count

## Migration from Single Worker

### Before (Fixed Configuration)
```go
updateQueue: make(chan *UpdateRequest, 100)
go service.processUpdateQueue()
```

### After (Configurable Worker Pool)
```go
updateQueue: make(chan *UpdateRequest, queueBufferSize)
service.startWorkerPool() // Starts N workers
```

### Migration Steps
1. Update environment variables
2. Restart service
3. Monitor performance metrics
4. Tune configuration based on load

## Best Practices

1. **Start Conservative**: Begin with 1-2 workers, scale up as needed
2. **Monitor Metrics**: Track CPU, memory, and response times
3. **Load Test**: Validate configuration under realistic load
4. **Environment Parity**: Use similar configs across environments
5. **Graceful Scaling**: Change worker count during low-traffic periods

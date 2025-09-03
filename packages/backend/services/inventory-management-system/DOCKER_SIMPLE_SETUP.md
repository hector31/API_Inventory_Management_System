# Simple Docker Setup

This is a streamlined Docker setup for the Inventory Management API with essential services only.

## Quick Start

### Prerequisites
- Docker and Docker Compose installed

### Start All Services
```bash
docker-compose up -d
```

### Access Services
- **Inventory API**: http://localhost:8082
- **Grafana Dashboard**: http://localhost:3001 (admin/admin123)
- **Prometheus Metrics**: http://localhost:9090

### Stop All Services
```bash
docker-compose down
```

## Services Included

### 1. Inventory API (Port 8082)
- Main application service
- Fine-grained locking with OCC
- Worker pool for concurrent processing
- Health check endpoint: `/health`

### 2. Grafana (Port 3001)
- Metrics visualization dashboard
- Default credentials: admin/admin123
- Ready for future metrics implementation

### 3. Prometheus (Port 9090)
- Metrics collection and storage
- Configured to scrape inventory API metrics
- Web UI for metrics exploration

## Configuration

### Environment Variables
The inventory API uses these default settings:
```bash
PORT=8081                           # Internal container port
LOG_LEVEL=info                      # Logging level
INVENTORY_WORKER_COUNT=4            # Worker pool size
INVENTORY_QUEUE_BUFFER_SIZE=500     # Queue buffer size
ENABLE_JSON_PERSISTENCE=true       # File persistence
DATA_PATH=./data/inventory_test_data.json  # Data file path
CACHE_TTL_MINUTES=30               # Cache TTL
CACHE_CLEANUP_INTERVAL_MINUTES=10  # Cache cleanup interval
```

### Data Persistence
- Inventory data is persisted in Docker volume `inventory_data`
- Grafana settings are persisted in Docker volume `grafana_data`
- Prometheus data is persisted in Docker volume `prometheus_data`

## Testing the Setup

### Health Check
```bash
curl http://localhost:8082/health
# Expected: {"status":"healthy"}
```

### API Test
```bash
curl -H "X-API-Key: demo" http://localhost:8082/v1/inventory/SKU-001
# Expected: Product information with current version and quantity
```

### Container Status
```bash
docker-compose ps
# Should show all 3 containers running and healthy
```

## Common Commands

### View Logs
```bash
# All services
docker-compose logs -f

# Inventory API only
docker-compose logs -f inventory-api

# Last 20 lines
docker-compose logs --tail=20 inventory-api
```

### Restart Services
```bash
# Restart all
docker-compose restart

# Restart specific service
docker-compose restart inventory-api
```

### Update and Rebuild
```bash
# Stop, rebuild, and start
docker-compose down
docker-compose build
docker-compose up -d
```

## Troubleshooting

### Port Conflicts
If you get port binding errors, check what's using the ports:
```bash
# Windows
netstat -ano | findstr :8082
netstat -ano | findstr :3001
netstat -ano | findstr :9090

# Linux/Mac
lsof -i :8082
lsof -i :3001
lsof -i :9090
```

### Container Issues
```bash
# Check container status
docker-compose ps

# View detailed logs
docker-compose logs inventory-api

# Restart problematic service
docker-compose restart inventory-api
```

### Clean Reset
```bash
# Stop and remove everything including volumes
docker-compose down -v

# Remove unused Docker resources
docker system prune -f

# Start fresh
docker-compose up -d
```

## File Structure

```
API_Inventory_Management_System/
├── docker-compose.yml          # Main Docker Compose configuration
├── Dockerfile                  # Multi-stage Docker build
├── monitoring/
│   └── prometheus.yml          # Prometheus configuration
└── data/
    └── inventory_test_data.json # Application data
```

## What Was Simplified

### Removed Files
- `docker-compose.override.yml` - Development overrides
- `docker-compose.prod.yml` - Production configuration

### Removed Services
- `redis` - Caching service (not needed yet)
- `load-tester` - Load testing container (use external tools)

### Removed Complexity
- Service profiles (monitoring, testing, caching)
- Multiple environment configurations
- Complex volume mappings
- Unnecessary network configurations

## Benefits of This Setup

### Simplicity
- ✅ Single command to start everything
- ✅ No complex configuration files
- ✅ Easy to understand and modify

### Functionality
- ✅ Full inventory API functionality
- ✅ Ready for metrics implementation
- ✅ Data persistence
- ✅ Health monitoring

### Development Ready
- ✅ Consistent environment
- ✅ Easy debugging with logs
- ✅ Quick iteration cycle

This simplified setup provides all the essential functionality while being easy to understand and use!

# Docker Deployment Guide

This guide covers containerization and deployment of the Inventory Management API using Docker and Docker Compose.

## Overview

The application is containerized using a multi-stage Docker build with the following features:
- **Multi-stage build** for optimized image size
- **Non-root user** for security
- **Health checks** for container monitoring
- **Volume mounts** for data persistence
- **Environment variable** configuration
- **Production-ready** setup with monitoring

## Quick Start

### Development Environment
```bash
# Build and run development stack
make docker-run

# Or manually
docker-compose up -d
```

### Production Environment
```bash
# Run production stack
make docker-prod

# Or manually
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Docker Files

### Dockerfile
- **Multi-stage build** with Go 1.21 and Alpine Linux
- **Security hardened** with non-root user
- **Optimized** for production with minimal attack surface
- **Health check** endpoint monitoring

### Docker Compose Files
- `docker-compose.yml` - Base configuration
- `docker-compose.override.yml` - Development overrides
- `docker-compose.prod.yml` - Production configuration

## Services

### Core Services

#### inventory-api
- **Port**: 8081
- **Health Check**: `/health` endpoint
- **Data Persistence**: Volume mounted to `/app/data`
- **Environment Variables**: Configurable via compose files

### Monitoring Stack (Optional)

#### Prometheus
- **Port**: 9090
- **Purpose**: Metrics collection
- **Profile**: `monitoring`

#### Grafana
- **Port**: 3000
- **Purpose**: Visualization and dashboards
- **Default Credentials**: admin/dev123 (development)
- **Profile**: `monitoring`

### Support Services (Optional)

#### Redis
- **Port**: 6379
- **Purpose**: Caching layer (future enhancement)
- **Profile**: `caching`

#### Load Tester
- **Purpose**: Load testing environment
- **Profile**: `testing`

## Environment Variables

### Application Configuration
```bash
PORT=8081                           # Server port
LOG_LEVEL=info                      # Logging level (debug, info, warn, error)
INVENTORY_WORKER_COUNT=4            # Number of worker goroutines
INVENTORY_QUEUE_BUFFER_SIZE=500     # Queue buffer size
ENABLE_JSON_PERSISTENCE=true       # Enable file persistence
DATA_PATH=./data/inventory_test_data.json  # Data file path
CACHE_TTL_MINUTES=30               # Cache TTL in minutes
CACHE_CLEANUP_INTERVAL_MINUTES=10  # Cache cleanup interval
```

### Monitoring Configuration
```bash
GRAFANA_ADMIN_PASSWORD=secure_password_123  # Grafana admin password
```

## Deployment Scenarios

### Development Deployment
```bash
# Start development environment
make docker-run

# View logs
make docker-logs

# Stop services
make docker-stop
```

**Features:**
- Debug logging enabled
- All monitoring services available
- Source code mounting (optional)
- Load testing environment

### Production Deployment
```bash
# Start production environment
make docker-prod

# Or with monitoring
make docker-monitoring
```

**Features:**
- Optimized resource limits
- Security hardening
- Performance tuning
- Monitoring stack

### Testing Deployment
```bash
# Start with load testing
make docker-test

# Run load tests
docker exec -it load-tester ./load_test_script.sh
```

## Data Persistence

### Volumes
- `inventory_data` - Application data persistence
- `inventory_logs` - Application logs
- `prometheus_data` - Prometheus metrics data
- `grafana_data` - Grafana dashboards and settings
- `redis_data` - Redis cache data

### Backup Strategy
```bash
# Backup data volume
docker run --rm -v inventory_data:/data -v $(pwd):/backup alpine tar czf /backup/inventory_backup.tar.gz /data

# Restore data volume
docker run --rm -v inventory_data:/data -v $(pwd):/backup alpine tar xzf /backup/inventory_backup.tar.gz -C /
```

## Networking

### Default Network
- **Name**: `inventory_network`
- **Driver**: bridge
- **Subnet**: 172.20.0.0/16

### Port Mapping
- **8081**: Inventory API
- **3000**: Grafana
- **9090**: Prometheus
- **6379**: Redis

## Security Considerations

### Container Security
- **Non-root user**: Application runs as user `appuser` (UID 1001)
- **Read-only filesystem**: Production containers use read-only root filesystem
- **No new privileges**: Security option prevents privilege escalation
- **Minimal base image**: Alpine Linux for reduced attack surface

### Network Security
- **Internal networking**: Services communicate via Docker network
- **Port exposure**: Only necessary ports exposed to host
- **Environment variables**: Sensitive data via environment variables

## Monitoring and Observability

### Health Checks
```bash
# Check container health
docker-compose ps

# Manual health check
curl http://localhost:8081/health
```

### Logs
```bash
# View application logs
make docker-logs

# View all service logs
docker-compose logs -f
```

### Metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

## Troubleshooting

### Common Issues

#### Container Won't Start
```bash
# Check logs
docker-compose logs inventory-api

# Check container status
docker-compose ps
```

#### Port Conflicts
```bash
# Check port usage
netstat -tulpn | grep :8081

# Use different ports
PORT=8082 docker-compose up -d
```

#### Volume Permissions
```bash
# Fix volume permissions
sudo chown -R 1001:1001 /opt/inventory/data
```

### Performance Tuning

#### Resource Limits
Adjust in `docker-compose.prod.yml`:
```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 1G
```

#### Worker Configuration
```bash
# Increase workers for high load
INVENTORY_WORKER_COUNT=8 docker-compose up -d
```

## Scaling

### Horizontal Scaling
```bash
# Scale API service
docker-compose up -d --scale inventory-api=3
```

### Load Balancing
Add nginx or traefik for load balancing multiple instances.

## CI/CD Integration

### Build Pipeline
```bash
# Build image
docker build -t inventory-api:${VERSION} .

# Tag for registry
docker tag inventory-api:${VERSION} registry.example.com/inventory-api:${VERSION}

# Push to registry
docker push registry.example.com/inventory-api:${VERSION}
```

### Deployment Pipeline
```bash
# Deploy to staging
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# Deploy to production
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

# Makefile for Inventory Management API

.PHONY: run run-store build test clean deps help docker-build docker-run docker-stop docker-logs docker-test docker-prod docker-monitoring

# Default environment variables
PORT ?= 8080
STORE_PORT ?= 8081
API_KEYS ?= demo

# Run the central inventory API server
run:
	@echo "Starting Central Inventory API on port $(PORT)..."
	PORT=$(PORT) API_KEYS=$(API_KEYS) go run ./cmd/server

# Build the application
build:
	@echo "Building inventory management API..."
	go build -o bin/inventory-api ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Local Development:"
	@echo "  run           - Start the central inventory API server"
	@echo "  build         - Build the application"
	@echo "  test          - Run tests"
	@echo "  deps          - Install dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo ""
	@echo "Docker Development:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  docker-stop   - Stop Docker containers"
	@echo "  docker-logs   - View Docker logs"
	@echo ""
	@echo "  help          - Show this help message"

# Docker Development Targets
docker-build:
	@echo "Building Docker image..."
	docker build -t inventory-api:latest .

docker-run:
	@echo "Starting inventory management stack..."
	docker-compose up -d
	@echo "✅ Services started successfully!"
	@echo ""
	@echo "🔗 Access URLs:"
	@echo "  • Inventory API: http://localhost:8082"
	@echo "  • Grafana Dashboard: http://localhost:3001 (admin/admin123)"
	@echo "  • Prometheus Metrics: http://localhost:9090"
	@echo ""
	@echo "📋 Quick Commands:"
	@echo "  • View logs: make docker-logs"
	@echo "  • Stop services: make docker-stop"

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs:
	@echo "Viewing Docker logs..."
	docker-compose logs -f inventory-api

# Utility targets
docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

docker-rebuild: docker-clean docker-build docker-run

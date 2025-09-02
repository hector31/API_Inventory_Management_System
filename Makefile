# Makefile for Inventory Management API

.PHONY: run run-store build test clean deps

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
	@echo "  run       - Start the central inventory API server"
	@echo "  build     - Build the application"
	@echo "  test      - Run tests"
	@echo "  deps      - Install dependencies"
	@echo "  clean     - Clean build artifacts"
	@echo "  fmt       - Format code"
	@echo "  lint      - Lint code"
	@echo "  help      - Show this help message"

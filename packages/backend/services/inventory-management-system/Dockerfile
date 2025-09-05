# Multi-stage Dockerfile for Inventory Management API
# Stage 1: Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux target
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o inventory-api ./cmd/server

# Stage 2: Runtime stage
FROM alpine:3.18

# Install ca-certificates for HTTPS requests and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Create data directory with proper permissions
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

# Copy binary from builder stage
COPY --from=builder /app/inventory-api .

# Copy data files
COPY --chown=appuser:appgroup data/ ./data/

# Switch to non-root user
USER appuser

# Environment variables with defaults
ENV PORT=8081
ENV LOG_LEVEL=info
ENV INVENTORY_WORKER_COUNT=4
ENV INVENTORY_QUEUE_BUFFER_SIZE=500
ENV ENABLE_JSON_PERSISTENCE=true
ENV DATA_PATH=./data/inventory_test_data.json
ENV CACHE_TTL_MINUTES=30
ENV CACHE_CLEANUP_INTERVAL_MINUTES=10

# Expose ports
EXPOSE 8081
EXPOSE 9080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/health || exit 1

# Command to run the application
CMD ["./inventory-api"]

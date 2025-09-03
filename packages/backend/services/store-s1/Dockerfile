# Multi-stage Dockerfile for Store S1 API
# Stage 1: Build stage
FROM golang:1.22-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy shared module first
COPY shared/ ./shared/

# Copy store-s1 module
COPY services/store-s1/ ./services/store-s1/

# Set working directory to store-s1
WORKDIR /app/services/store-s1

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o store-s1-api ./cmd/server

# Stage 2: Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Create necessary directories and set permissions
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appgroup /app

# Copy binary from builder stage
COPY --from=builder /app/services/store-s1/store-s1-api .

# Change to non-root user
USER appuser

# Expose port
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8083/health || exit 1

# Run the application
CMD ["./store-s1-api"]

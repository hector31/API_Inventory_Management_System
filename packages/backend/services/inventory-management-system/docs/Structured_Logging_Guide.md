# Structured Logging Guide

This guide explains the structured logging implementation using Go's built-in `slog` package in the Inventory Management API.

## Overview

The application uses **structured logging** with Go's `slog` package, which provides:
- **Structured output**: Key-value pairs instead of plain text
- **Configurable log levels**: Control verbosity via environment variables
- **Performance**: Built-in, optimized logging
- **Consistency**: Standardized logging across the application

## Log Levels

The application supports four log levels (from most to least verbose):

| Level | Description | When to Use | Example |
|-------|-------------|-------------|---------|
| `debug` | Detailed diagnostic information | Development, troubleshooting | Function entry/exit, variable values |
| `info` | General information | Production monitoring | Server startup, successful operations |
| `warn` | Warning conditions | Potential issues | Deprecated features, recoverable errors |
| `error` | Error conditions | Failures that need attention | Failed operations, exceptions |

## Configuration

### Environment Variable
Set the `LOG_LEVEL` environment variable to control logging verbosity:

```bash
# Show all logs (most verbose)
export LOG_LEVEL=debug

# Show info, warn, and error logs (default)
export LOG_LEVEL=info

# Show only warn and error logs
export LOG_LEVEL=warn

# Show only error logs (least verbose)
export LOG_LEVEL=error
```

### .env File
```bash
# .env
LOG_LEVEL=debug
```

### Runtime Configuration
The log level is configured at application startup in `internal/config/config.go`:

```go
func setupLogging(logLevel string) {
    var level slog.Level
    
    switch logLevel {
    case "debug":
        level = slog.LevelDebug
    case "info":
        level = slog.LevelInfo
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    })
    
    slog.SetDefault(slog.New(handler))
}
```

## Log Output Format

### Structured Format
All logs use key-value pairs for structured data:

```
time=2025-09-02T10:00:00.000Z level=INFO msg="Configuration loaded" port=8080 environment=development logLevel=debug dataPath=data/inventory_test_data.json
```

### Key Components
- **time**: ISO 8601 timestamp
- **level**: Log level (DEBUG, INFO, WARN, ERROR)
- **msg**: Human-readable message
- **Additional fields**: Context-specific key-value pairs

## Usage Examples

### Application Startup
```go
slog.Info("Starting Inventory Management API", "version", "1.0.0")
slog.Info("Configuration loaded", 
    "port", cfg.Port, 
    "environment", cfg.Environment,
    "logLevel", cfg.LogLevel)
```

### Service Operations
```go
slog.Debug("Loading test data", "path", dataPath)
slog.Info("Test data loaded successfully", 
    "path", dataPath, 
    "products_count", len(s.data.Products))
```

### HTTP Requests
```go
slog.Debug("Retrieving product", "product_id", productID)
slog.Warn("Product not found", "product_id", productID)
slog.Info("Products listed successfully", 
    "cursor", cursor, 
    "limit", limit, 
    "found_count", len(productList.Items),
    "remote_addr", r.RemoteAddr)
```

### Authentication
```go
slog.Warn("Authentication failed: missing API key", "remote_addr", r.RemoteAddr)
slog.Warn("Authentication failed: invalid API key", 
    "remote_addr", r.RemoteAddr, 
    "provided_key", apiKey)
slog.Debug("Authentication successful", 
    "remote_addr", r.RemoteAddr, 
    "api_key", apiKey)
```

### Error Handling
```go
slog.Error("Failed to initialize inventory service", "error", err)
slog.Error("Failed to read test data file", "path", dataPath, "error", err)
```

## Log Level Examples

### DEBUG Level (LOG_LEVEL=debug)
Shows all logs including detailed diagnostic information:
```
time=2025-09-02T10:00:00.000Z level=DEBUG msg="Loading test data" path=data/inventory_test_data.json
time=2025-09-02T10:00:00.001Z level=INFO msg="Test data loaded successfully" path=data/inventory_test_data.json products_count=10 last_offset=1287
time=2025-09-02T10:00:00.002Z level=DEBUG msg="HTTP handlers initialized"
time=2025-09-02T10:00:00.003Z level=DEBUG msg="Available endpoints" v1_endpoints="[POST /v1/inventory/updates, GET /v1/inventory/{productId}]"
time=2025-09-02T10:00:00.004Z level=DEBUG msg="Retrieving product" product_id=SKU-001
time=2025-09-02T10:00:00.005Z level=DEBUG msg="Authentication successful" remote_addr=127.0.0.1:12345 api_key=demo
```

### INFO Level (LOG_LEVEL=info)
Shows operational information without debug details:
```
time=2025-09-02T10:00:00.000Z level=INFO msg="Successfully loaded .env file"
time=2025-09-02T10:00:00.001Z level=INFO msg="Configuration loaded" port=8080 environment=development
time=2025-09-02T10:00:00.002Z level=INFO msg="Starting Inventory Management API" version=1.0.0
time=2025-09-02T10:00:00.003Z level=INFO msg="Test data loaded successfully" path=data/inventory_test_data.json products_count=10
time=2025-09-02T10:00:00.004Z level=INFO msg="Server ready to accept connections" address=:8080
```

### WARN Level (LOG_LEVEL=warn)
Shows only warnings and errors:
```
time=2025-09-02T10:00:00.000Z level=WARN msg="Could not load .env file, continuing with system environment variables only" error="open .env: no such file or directory"
time=2025-09-02T10:00:00.001Z level=WARN msg="Authentication failed: invalid API key" remote_addr=127.0.0.1:12345 provided_key=invalid
time=2025-09-02T10:00:00.002Z level=WARN msg="Product not found" product_id=SKU-999
```

### ERROR Level (LOG_LEVEL=error)
Shows only critical errors:
```
time=2025-09-02T10:00:00.000Z level=ERROR msg="Failed to initialize inventory service" error="error reading test data file: open data/inventory_test_data.json: no such file or directory"
time=2025-09-02T10:00:00.001Z level=ERROR msg="Server failed to start" error="listen tcp :8080: bind: address already in use"
```

## Best Practices

### 1. Use Appropriate Log Levels
```go
// ✅ Good: Use debug for detailed diagnostics
slog.Debug("Processing request", "method", r.Method, "path", r.URL.Path)

// ✅ Good: Use info for important operations
slog.Info("User authenticated", "user_id", userID)

// ✅ Good: Use warn for recoverable issues
slog.Warn("Rate limit approaching", "current", current, "limit", limit)

// ✅ Good: Use error for failures
slog.Error("Database connection failed", "error", err)
```

### 2. Include Relevant Context
```go
// ✅ Good: Include relevant context
slog.Info("Product updated", 
    "product_id", productID, 
    "old_quantity", oldQty, 
    "new_quantity", newQty,
    "user_id", userID)

// ❌ Bad: Missing context
slog.Info("Product updated")
```

### 3. Use Structured Fields
```go
// ✅ Good: Structured key-value pairs
slog.Error("Validation failed", 
    "field", "email", 
    "value", email, 
    "error", "invalid format")

// ❌ Bad: Unstructured message
slog.Error(fmt.Sprintf("Validation failed for email %s: invalid format", email))
```

### 4. Consistent Field Names
```go
// ✅ Good: Consistent naming
slog.Info("Request processed", "user_id", userID, "request_id", reqID)
slog.Error("Request failed", "user_id", userID, "request_id", reqID, "error", err)

// ❌ Bad: Inconsistent naming
slog.Info("Request processed", "userId", userID, "reqId", reqID)
slog.Error("Request failed", "user_id", userID, "request_id", reqID, "error", err)
```

## Testing Log Levels

### Test Debug Level
```bash
LOG_LEVEL=debug go run ./cmd/server
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory/SKU-001
```

### Test Info Level
```bash
LOG_LEVEL=info go run ./cmd/server
curl -H 'X-API-Key: demo' http://localhost:8080/v1/inventory/SKU-001
```

### Test Authentication Warnings
```bash
LOG_LEVEL=warn go run ./cmd/server
curl -H 'X-API-Key: invalid' http://localhost:8080/v1/inventory/SKU-001
```

## Migration from Previous Logging

### Before (fmt.Printf/log.Printf)
```go
log.Printf("Configuration loaded: Port=%s, Environment=%s", port, env)
fmt.Printf("Listing products with cursor: %s, limit: %d\n", cursor, limit)
```

### After (slog)
```go
slog.Info("Configuration loaded", "port", port, "environment", env)
slog.Debug("Listing products", "cursor", cursor, "limit", limit)
```

## Benefits

1. **Structured Data**: Easy to parse and analyze
2. **Configurable Verbosity**: Control log output per environment
3. **Performance**: Built-in, optimized logging
4. **Consistency**: Standardized format across application
5. **Tooling**: Compatible with log aggregation systems
6. **Debugging**: Rich context for troubleshooting

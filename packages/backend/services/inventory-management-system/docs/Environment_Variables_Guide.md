# Environment Variables Guide

This guide explains how to configure the Inventory Management API using environment variables and .env files.

## Quick Start

1. **Copy the example file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit the .env file** with your desired configuration

3. **Run the server:**
   ```bash
   go run ./cmd/server
   ```

## Why .env Files Don't Work Automatically in Go

Unlike some frameworks (Node.js with dotenv), **Go doesn't automatically load .env files**. Here's why:

- **`os.Getenv()` is system-level**: It only reads from the process environment, not files
- **No built-in file parsing**: Go requires explicit loading of .env files
- **Security by design**: Prevents accidental loading of configuration files

## How We Solve This

We use the **`github.com/joho/godotenv`** library to load .env files before reading environment variables.

### Code Example

```go
package main

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

func main() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Printf("Warning: Could not load .env file: %v", err)
    }
    
    // Now os.Getenv() will read from .env file
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
}
```

## Configuration Structure

Our application uses a structured configuration approach:

```go
type Config struct {
    Port        string   // Server port
    Host        string   // Server host
    APIKeys     []string // Valid API keys
    DataPath    string   // Path to test data
    LogLevel    string   // Logging level
    LogFormat   string   // Log format (text/json)
    Environment string   // Environment (dev/staging/prod)
}
```

## Environment Variables vs .env Files

### System Environment Variables
```bash
# Set in shell/system
export PORT=8080
export API_KEYS=demo,prod-key

# Run application
go run ./cmd/server
```

**Characteristics:**
- ✅ **Higher priority**: Override .env file values
- ✅ **Production ready**: Standard for containers/cloud
- ✅ **Secure**: Not stored in code repository
- ❌ **Development friction**: Must set manually each time

### .env File Variables
```bash
# .env file
PORT=8080
API_KEYS=demo,test-key
```

**Characteristics:**
- ✅ **Development friendly**: Easy to configure locally
- ✅ **Version controlled**: Can include .env.example
- ✅ **Team consistency**: Same config across developers
- ❌ **Security risk**: Should not contain production secrets
- ❌ **Lower priority**: Overridden by system variables

## Priority Order

Our configuration loading follows this priority (highest to lowest):

1. **System Environment Variables** (highest priority)
2. **.env File Variables**
3. **Default Values** (lowest priority)

### Example:
```bash
# .env file
PORT=8080

# System environment
export PORT=9000

# Result: Server runs on port 9000 (system env wins)
```

## Available Configuration Options

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `PORT` | `8080` | Server port | `8080` |
| `HOST` | `localhost` | Server host | `localhost` |
| `API_KEYS` | `demo` | Valid API keys (comma-separated) | `demo,test,prod` |
| `DATA_PATH` | `data/inventory_test_data.json` | Test data file path | `data/products.json` |
| `LOG_LEVEL` | `info` | Logging level | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` | Log output format | `text`, `json` |
| `ENVIRONMENT` | `development` | Runtime environment | `development`, `staging`, `production` |

## Best Practices

### Development
```bash
# Use .env file for local development
cp .env.example .env
# Edit .env with your preferences
```

### Production
```bash
# Use system environment variables
export PORT=8080
export API_KEYS=secure-production-key
export ENVIRONMENT=production
```

### Docker
```dockerfile
# Dockerfile
ENV PORT=8080
ENV ENVIRONMENT=production

# Or use docker-compose with .env
```

```yaml
# docker-compose.yml
services:
  api:
    environment:
      - PORT=${PORT}
      - API_KEYS=${API_KEYS}
    env_file:
      - .env
```

## Security Considerations

### ✅ Safe for .env files:
- Development configuration
- Non-sensitive defaults
- Feature flags
- Debug settings

### ❌ Never put in .env files:
- Production API keys
- Database passwords
- JWT secrets
- Third-party service tokens

### Recommended approach:
```bash
# .env (safe for git)
PORT=8080
ENVIRONMENT=development
LOG_LEVEL=debug

# System environment (production secrets)
export API_KEYS=super-secure-production-key
export DB_PASSWORD=secret-password
```

## Testing Configuration

### Test with .env file:
```bash
# Ensure .env exists
cat .env

# Run server
go run ./cmd/server
```

### Test with system variables:
```bash
# Override .env values
export PORT=9000
export API_KEYS=test-override

# Run server
go run ./cmd/server
```

### Verify configuration loading:
```bash
# Check server startup logs
2025/09/02 10:00:00 Successfully loaded .env file
2025/09/02 10:00:00 Configuration loaded: Port=8080, Environment=development, APIKeys=[demo test-key prod-key]
```

## Troubleshooting

### .env file not loading:
- ✅ Check file exists: `ls -la .env`
- ✅ Check file location: Must be in working directory
- ✅ Check file format: No spaces around `=`
- ✅ Check logs: Look for "Successfully loaded .env file"

### Environment variables not working:
- ✅ Check priority: System env overrides .env
- ✅ Check spelling: Variable names are case-sensitive
- ✅ Check defaults: Application may use fallback values
- ✅ Check parsing: Some values need special handling (comma-separated lists)

### API key validation failing:
- ✅ Check .env file: `API_KEYS=demo,test-key`
- ✅ Check request header: `X-API-Key: demo`
- ✅ Check logs: Look for configuration loading messages
- ✅ Test with curl: `curl -H 'X-API-Key: demo' http://localhost:8080/health`

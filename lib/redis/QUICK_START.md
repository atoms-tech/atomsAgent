# Redis Client Quick Start

## Installation

The Redis client is already part of agentapi. Dependencies are installed:

```bash
go get github.com/redis/go-redis/v9
```

## Configuration

Add to your `.env` file:

```bash
REDIS_URL=rediss://default:AYseAAIncDFhMDY2ZmJlNWNlYzg0ZWNhYmFlYjRmYjliNmQ2NGUwOXAxMzU2MTQ@neat-sloth-35614.upstash.io:6379
REDIS_REST_URL=https://neat-sloth-35614.upstash.io
REDIS_TOKEN=AYseAAIncDFhMDY2ZmJlNWNlYzg0ZWNhYmFlYjRmYjliNmQ2NGUwOXAxMzU2MTQ
```

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/coder/agentapi/lib/redis"
)

func main() {
    // 1. Create config
    config := redis.DefaultConfig()
    config.URL = os.Getenv("REDIS_URL")
    config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
    config.Token = os.Getenv("REDIS_TOKEN")

    // 2. Initialize client
    client, err := redis.NewRedisClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 3. Use it
    ctx := context.Background()

    // Set a value with 1 hour TTL
    err = client.Set(ctx, "mykey", "myvalue", 1*time.Hour)
    if err != nil {
        log.Fatal(err)
    }

    // Get the value
    value, err := client.Get(ctx, "mykey")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Value: %s", value)
}
```

## Common Operations

```go
ctx := context.Background()

// Store a value (no expiration)
client.Set(ctx, "key", "value", 0)

// Store with TTL
client.Set(ctx, "session:123", "data", 1*time.Hour)

// Get a value
value, err := client.Get(ctx, "key")

// Check existence
exists, err := client.Exists(ctx, "key")

// Increment counter
client.Increment(ctx, "counter")

// Delete
client.Delete(ctx, "key")

// Health check
err := client.Health()
```

## Integration with Health Check

```go
import (
    "github.com/coder/agentapi/lib/health"
    "github.com/coder/agentapi/lib/redis"
)

// In your main.go or setup function
redisClient, _ := redis.NewRedisClient(config)
healthChecker := health.NewHealthChecker(db, fastmcpClient)
healthChecker.RegisterCheck("redis", redis.NewHealthCheck(redisClient))
```

## Running Tests

```bash
# Unit tests
go test ./lib/redis/

# Integration tests (requires Redis credentials)
export REDIS_URL="rediss://..."
export REDIS_REST_URL="https://..."
export REDIS_TOKEN="..."
go test -v ./lib/redis/

# Benchmarks
go test -bench=. ./lib/redis/
```

## Common Patterns

### Session Storage

```go
sessionID := "session:user123"
sessionData := `{"user_id": "123", "email": "user@example.com"}`

// Store session
client.Set(ctx, sessionID, sessionData, 1*time.Hour)

// Retrieve session
data, _ := client.Get(ctx, sessionID)

// Invalidate session
client.Delete(ctx, sessionID)
```

### Caching

```go
cacheKey := "cache:user:123"

// Try cache first
if cached, err := client.Get(ctx, cacheKey); err == nil && cached != "" {
    return cached // Cache hit
}

// Fetch from database
data := fetchFromDB()

// Store in cache
client.Set(ctx, cacheKey, data, 5*time.Minute)
```

### Rate Limiting

```go
userID := "user:456"
timestamp := time.Now().Format("2006-01-02-15:04")
key := fmt.Sprintf("ratelimit:%s:%s", userID, timestamp)

// Increment request count
client.Increment(ctx, key)

// Get count
countStr, _ := client.Get(ctx, key)
var count int
fmt.Sscanf(countStr, "%d", &count)

// Set TTL on first request
if count == 1 {
    client.Set(ctx, key, countStr, 1*time.Minute)
}

if count > 100 {
    return errors.New("rate limit exceeded")
}
```

## Protocol Fallback

The client automatically falls back from Native to REST:

```go
// Client tries native first
client.Set(ctx, "key", "value", 0)

// If native fails, automatically uses REST
log.Printf("Using protocol: %s", client.GetActiveProtocol())
// Output: "rest" (if native failed)
```

## Error Handling

```go
import "errors"

value, err := client.Get(ctx, "key")
if err != nil {
    if errors.Is(err, redis.ErrClientClosed) {
        // Client was closed
    } else if errors.Is(err, redis.ErrConnectionFailed) {
        // Both protocols failed
    } else if errors.Is(err, context.Canceled) {
        // Context was cancelled
    } else {
        // Other errors
    }
}
```

## Troubleshooting

### Connection Failed
- Check REDIS_URL is correct
- Verify network connectivity
- Ensure TLS is working (rediss://)

### REST Fallback Not Working
- Verify REDIS_REST_URL is set
- Check REDIS_TOKEN is correct
- Ensure both URL and Token are configured

### Performance Issues
- Increase PoolSize in config
- Adjust timeout values
- Monitor connection pool utilization

## Production Checklist

- [ ] Environment variables configured
- [ ] Health check registered
- [ ] Client.Close() called on shutdown
- [ ] Appropriate TTLs set on keys
- [ ] Error handling implemented
- [ ] Monitoring in place

## More Information

- Full documentation: `lib/redis/README.md`
- Implementation details: `lib/redis/IMPLEMENTATION_SUMMARY.md`
- Examples: `lib/redis/example_integration.go`
- Tests: `lib/redis/client_test.go`

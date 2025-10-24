# Redis Client Library

Production-ready Redis client for agentapi with dual-protocol support (native Redis and REST API fallback).

## Features

- **Dual Protocol Support**: Native Redis (TCP) with automatic REST API fallback
- **Connection Pooling**: Efficient connection management with configurable pool sizes
- **Retry Logic**: Exponential backoff with configurable retry attempts
- **Health Checks**: Built-in health monitoring and status reporting
- **Context Support**: Full context.Context support for cancellation and timeouts
- **Thread-Safe**: Safe for concurrent use across goroutines
- **Graceful Shutdown**: Proper resource cleanup

## Installation

The Redis client is part of the agentapi project. To use it:

```go
import "github.com/coder/agentapi/lib/redis"
```

Ensure the Redis dependency is installed:

```bash
go get github.com/redis/go-redis/v9
```

## Configuration

### Environment Variables

Add the following to your `.env` file:

```bash
# Native Redis connection (rediss protocol for TLS)
REDIS_URL=rediss://default:{token}@neat-sloth-35614.upstash.io:6379

# REST API fallback
REDIS_REST_URL=https://neat-sloth-35614.upstash.io
REDIS_TOKEN=AYseAAIncDFhMDY2ZmJlNWNlYzg0ZWNhYmFlYjRmYjliNmQ2NGUwOXAxMzU2MTQ
```

### Code Configuration

```go
// Create configuration with defaults
config := redis.DefaultConfig()

// Override specific settings
config.URL = os.Getenv("REDIS_URL")
config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
config.Token = os.Getenv("REDIS_TOKEN")
config.PreferredProtocol = redis.ProtocolNative
config.PoolSize = 20
config.MaxRetries = 5

// Create client
client, err := redis.NewRedisClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## Usage Examples

### Basic Operations

```go
ctx := context.Background()

// Set a value
err := client.Set(ctx, "user:1:name", "John Doe", 0)

// Set with TTL (1 hour)
err = client.Set(ctx, "session:abc", "session-data", 1*time.Hour)

// Get a value
value, err := client.Get(ctx, "user:1:name")

// Check if key exists
exists, err := client.Exists(ctx, "user:1:name")

// Increment a counter
err = client.Increment(ctx, "api:requests:count")

// Delete a key
err = client.Delete(ctx, "user:1:name")
```

### Session Management

```go
func StoreSession(client *redis.RedisClient, sessionID string, data string) error {
    ctx := context.Background()

    // Store session with 1 hour expiration
    key := fmt.Sprintf("session:%s", sessionID)
    return client.Set(ctx, key, data, 1*time.Hour)
}

func GetSession(client *redis.RedisClient, sessionID string) (string, error) {
    ctx := context.Background()

    key := fmt.Sprintf("session:%s", sessionID)
    return client.Get(ctx, key)
}

func InvalidateSession(client *redis.RedisClient, sessionID string) error {
    ctx := context.Background()

    key := fmt.Sprintf("session:%s", sessionID)
    return client.Delete(ctx, key)
}
```

### Rate Limiting

```go
func CheckRateLimit(client *redis.RedisClient, userID string, maxRequests int) (bool, error) {
    ctx := context.Background()

    // Create time-based key (per minute)
    timestamp := time.Now().Format("2006-01-02-15:04")
    key := fmt.Sprintf("ratelimit:%s:%s", userID, timestamp)

    // Increment request counter
    if err := client.Increment(ctx, key); err != nil {
        return false, err
    }

    // Set expiration on first request
    countStr, err := client.Get(ctx, key)
    if err != nil {
        return false, err
    }

    var count int
    fmt.Sscanf(countStr, "%d", &count)

    // Set TTL if this is the first request
    if count == 1 {
        client.Set(ctx, key, countStr, 1*time.Minute)
    }

    // Check if limit exceeded
    return count <= maxRequests, nil
}
```

### Caching

```go
func GetOrFetchUser(client *redis.RedisClient, userID string) (string, error) {
    ctx := context.Background()

    // Try cache first
    cacheKey := fmt.Sprintf("cache:user:%s", userID)
    cached, err := client.Get(ctx, cacheKey)
    if err != nil {
        return "", err
    }

    if cached != "" {
        // Cache hit
        return cached, nil
    }

    // Cache miss - fetch from database
    userData := fetchUserFromDB(userID) // Your DB fetch logic

    // Store in cache with 5 minute TTL
    if err := client.Set(ctx, cacheKey, userData, 5*time.Minute); err != nil {
        // Log error but don't fail the request
        log.Printf("Failed to cache user data: %v", err)
    }

    return userData, nil
}
```

### Health Checks

```go
// Basic health check
if err := client.Health(); err != nil {
    log.Printf("Redis unhealthy: %v", err)
}

// Get active protocol
protocol := client.GetActiveProtocol()
fmt.Printf("Using protocol: %s\n", protocol) // "native" or "rest"

// Integration with health package
healthCheck := redis.NewHealthCheck(client)
if err := healthCheck.Check(context.Background()); err != nil {
    // Handle unhealthy state
}

// Get detailed status
status, err := healthCheck.GetStatus(context.Background())
fmt.Printf("Redis Status: %+v\n", status)
```

## Protocol Fallback

The client automatically falls back to REST API if the native connection fails:

```go
config := redis.DefaultConfig()
config.URL = "rediss://..."           // Primary: Native Redis
config.RESTBaseURL = "https://..."    // Fallback: REST API
config.Token = "..."
config.PreferredProtocol = redis.ProtocolNative

client, _ := redis.NewRedisClient(config)

// Client will try native first
// If native fails, automatically switches to REST
err := client.Set(ctx, "key", "value", 0)

// Check which protocol is active
if client.GetActiveProtocol() == redis.ProtocolREST {
    log.Println("Using REST fallback")
}
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `URL` | string | "" | Native Redis URL (rediss://...) |
| `RESTBaseURL` | string | "" | REST API base URL |
| `Token` | string | "" | REST API authentication token |
| `MaxRetries` | int | 3 | Maximum retry attempts |
| `MinRetryBackoff` | time.Duration | 100ms | Minimum retry delay |
| `MaxRetryBackoff` | time.Duration | 3s | Maximum retry delay |
| `DialTimeout` | time.Duration | 5s | Connection timeout |
| `ReadTimeout` | time.Duration | 3s | Read operation timeout |
| `WriteTimeout` | time.Duration | 3s | Write operation timeout |
| `PoolSize` | int | 10 | Maximum connection pool size |
| `MinIdleConns` | int | 2 | Minimum idle connections |
| `MaxIdleTime` | time.Duration | 5m | Maximum idle time |
| `PreferredProtocol` | Protocol | ProtocolNative | Preferred protocol (native/rest) |

## Error Handling

```go
import "errors"

_, err := client.Get(ctx, "key")
if err != nil {
    if errors.Is(err, redis.ErrClientClosed) {
        // Client has been closed
    } else if errors.Is(err, redis.ErrConnectionFailed) {
        // Both protocols failed
    } else if errors.Is(err, context.Canceled) {
        // Context was cancelled
    } else {
        // Other errors
    }
}
```

## Testing

Run tests with:

```bash
# Unit tests (no Redis required)
go test ./lib/redis/

# Integration tests (requires Redis)
export REDIS_URL="rediss://..."
export REDIS_REST_URL="https://..."
export REDIS_TOKEN="..."
go test -v ./lib/redis/

# Benchmarks
go test -bench=. ./lib/redis/
```

## Best Practices

1. **Always use context**: Pass context for cancellation and timeout control
2. **Set appropriate TTLs**: Prevent memory bloat by setting expiration times
3. **Handle errors gracefully**: Don't fail requests on cache errors
4. **Use connection pooling**: Reuse the same client instance
5. **Close clients properly**: Use `defer client.Close()` for cleanup
6. **Monitor health**: Regularly check `client.Health()` in production
7. **Use meaningful keys**: Follow a consistent naming pattern (e.g., `resource:id:field`)

## Key Naming Conventions

Recommended patterns:

```
session:{session_id}           // Session data
user:{user_id}:{field}         // User data
cache:{resource}:{id}          // Cache entries
ratelimit:{user_id}:{window}   // Rate limiting counters
lock:{resource}:{id}           // Distributed locks
counter:{metric}               // Application counters
```

## Thread Safety

The Redis client is thread-safe and can be used concurrently:

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        key := fmt.Sprintf("concurrent:key:%d", id)
        client.Set(context.Background(), key, "value", 0)
    }(i)
}

wg.Wait()
```

## Graceful Shutdown

```go
// Create client
client, _ := redis.NewRedisClient(config)

// Set up signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// Wait for signal
<-sigChan

// Graceful shutdown
log.Println("Shutting down Redis client...")
if err := client.Close(); err != nil {
    log.Printf("Error closing Redis client: %v", err)
}
```

## Troubleshooting

### Connection Issues

If you get connection errors:

1. Verify your `REDIS_URL` is correct
2. Check network connectivity to Redis server
3. Ensure firewall allows connections on port 6379
4. Verify TLS/SSL settings (use `rediss://` for TLS)

### Fallback Not Working

If REST fallback doesn't activate:

1. Verify `REDIS_REST_URL` is set
2. Check `REDIS_TOKEN` is correct
3. Ensure both protocols are configured in `Config`

### Performance Issues

If experiencing slow operations:

1. Increase `PoolSize` for high concurrency
2. Adjust timeout values
3. Monitor connection pool utilization
4. Consider using pipelining for bulk operations

## License

Part of the agentapi project. See main LICENSE file.

# Rate Limiter Quick Start Guide

## 5-Minute Setup

### 1. Import the Package
```go
import (
    "github.com/coder/agentapi/lib/ratelimit"
    "github.com/coder/agentapi/lib/redis"
)
```

### 2. Setup Redis Client
```go
// Create Redis client
redisConfig := redis.DefaultConfig()
redisConfig.URL = "redis://localhost:6379"  // or from env
redisClient, err := redis.NewRedisClient(redisConfig)
if err != nil {
    log.Fatal(err)
}
defer redisClient.Close()
```

### 3. Create Rate Limiter
```go
// Use defaults: 60 req/min, burst of 10
limiter, err := ratelimit.NewRateLimiter(redisClient, ratelimit.DefaultConfig())
if err != nil {
    log.Fatal(err)
}
```

### 4. Add HTTP Middleware
```go
// Setup middleware
middlewareConfig := ratelimit.DefaultMiddlewareConfig(limiter)

// Create your routes
mux := http.NewServeMux()
mux.HandleFunc("/api/v1/data", handleData)

// Wrap with rate limiter
handler := ratelimit.Middleware(middlewareConfig)(mux)

// Start server
http.ListenAndServe(":8080", handler)
```

### 5. Integration with Auth (Optional)
```go
// In your auth middleware, add user info to context
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := validateJWT(r)  // Your JWT validation

        // Add to context for rate limiter
        ctx := r.Context()
        ctx = ratelimit.WithUserID(ctx, claims.UserID)
        ctx = ratelimit.WithOrgID(ctx, claims.OrgID)
        ctx = ratelimit.WithAdminStatus(ctx, claims.Role == "admin")

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Chain middlewares: auth -> rate limit -> handler
handler := authMiddleware(ratelimit.Middleware(middlewareConfig)(mux))
```

## Common Configurations

### Strict Rate Limiting (10 req/min)
```go
config := ratelimit.DefaultConfig()
config.RequestsPerMinute = 10
config.BurstSize = 2
limiter, _ := ratelimit.NewRateLimiter(redisClient, config)
```

### Relaxed Rate Limiting (1000 req/min)
```go
config := ratelimit.DefaultConfig()
config.RequestsPerMinute = 1000
config.BurstSize = 100
limiter, _ := ratelimit.NewRateLimiter(redisClient, config)
```

### Per-Endpoint Limits
```go
config := ratelimit.DefaultConfig()
config.EndpointLimits = map[string]ratelimit.EndpointLimit{
    "/api/v1/upload": {
        RequestsPerMinute: 10,   // Strict
        BurstSize:         2,
        Enabled:           true,
    },
    "/api/v1/search": {
        RequestsPerMinute: 200,  // Relaxed
        BurstSize:         50,
        Enabled:           true,
    },
    "/api/v1/public": {
        Enabled: false,          // No rate limiting
    },
}
limiter, _ := ratelimit.NewRateLimiter(redisClient, config)
```

## Direct Usage (Without Middleware)

```go
// Check if request should be allowed
allowed, remaining, resetAt, err := limiter.AllowRequest(
    ctx,
    "user123",      // User ID
    "org456",       // Org ID
    "/api/v1/data", // Endpoint
)

if err != nil {
    // Handle error
    http.Error(w, "Rate limit check failed", 500)
    return
}

if !allowed {
    // Set headers
    w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt).Seconds())))

    // Return 429
    http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
    return
}

// Process request normally
// Remaining tokens available in 'remaining' variable
```

## IP-Based Rate Limiting (Anonymous Users)

```go
ipAddress := getClientIP(r)  // Extract from request

allowed, remaining, resetAt, err := limiter.AllowRequestByIP(
    ctx,
    ipAddress,
    "/api/v1/public",
)

if !allowed {
    http.Error(w, "Too many requests", 429)
    return
}
```

## Environment Variables

```bash
# .env or environment
REDIS_URL=redis://localhost:6379
# or
REDIS_URL=rediss://user:pass@host:port/db
```

```go
redisConfig := redis.DefaultConfig()
redisConfig.URL = os.Getenv("REDIS_URL")
```

## Testing Your Rate Limiter

### Test with curl
```bash
# Make multiple requests quickly
for i in {1..15}; do
    curl -i http://localhost:8080/api/v1/data
    echo "Request $i"
done

# You should see:
# - First 10 requests: 200 OK (burst allowed)
# - Remaining requests: 429 Too Many Requests
```

### Check Headers
```bash
curl -I http://localhost:8080/api/v1/data

# Response headers:
# X-RateLimit-Limit: 60
# X-RateLimit-Remaining: 59
# X-RateLimit-Reset: 1698765432
```

### When Rate Limited
```bash
curl -I http://localhost:8080/api/v1/data

# Response (after exceeding limit):
# HTTP/1.1 429 Too Many Requests
# Retry-After: 45
# X-RateLimit-Remaining: 0
```

## Common Patterns

### Pattern 1: Global Default + Endpoint Overrides
```go
config := ratelimit.DefaultConfig()
config.RequestsPerMinute = 60  // Global default

// Override specific endpoints
config.EndpointLimits["/api/v1/heavy"] = ratelimit.EndpointLimit{
    RequestsPerMinute: 10,
    BurstSize:         2,
    Enabled:           true,
}
```

### Pattern 2: Skip Certain Paths
```go
middlewareConfig := ratelimit.DefaultMiddlewareConfig(limiter)
middlewareConfig.SkipPaths = []string{
    "/health",
    "/metrics",
    "/ping",
    "/api/v1/public",
}
```

### Pattern 3: Custom Error Handler
```go
middlewareConfig.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err *ratelimit.RateLimitError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusTooManyRequests)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": "Rate limit exceeded",
        "retry_after_seconds": int(err.RetryAfter.Seconds()),
        "reset_at": err.ResetAt,
    })
}
```

## Troubleshooting

### Issue: Rate limiting not working
```go
// Enable detailed logging
middlewareConfig.DetailedLogging = true

// Check logs for:
// - "Rate limit check passed"
// - "Rate limit exceeded"
```

### Issue: All requests getting rate limited
```go
// Increase limits
config.BurstSize = 20           // More burst capacity
config.RequestsPerMinute = 120  // Higher rate

// Or disable for testing
config.EndpointLimits["/api/v1/test"] = ratelimit.EndpointLimit{
    Enabled: false,
}
```

### Issue: Redis connection errors
```go
// Test Redis connection
if err := redisClient.Health(); err != nil {
    log.Printf("Redis unhealthy: %v", err)
}
```

## Next Steps

1. Read the full [README.md](./README.md) for advanced features
2. Check [example_integration.go](./example_integration.go) for more examples
3. Review [IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md) for architecture details
4. Run tests: `go test ./lib/ratelimit`

## Support

- Documentation: See README.md
- Examples: See example_integration.go
- Tests: See limiter_test.go
- Package docs: `go doc github.com/coder/agentapi/lib/ratelimit`

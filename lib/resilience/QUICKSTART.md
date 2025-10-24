# Circuit Breaker Quick Start Guide

This guide will get you up and running with the circuit breaker pattern in 5 minutes.

## Installation

The circuit breaker is part of the agentapi library:

```go
import "github.com/coder/agentapi/lib/resilience"
```

## Basic Usage

### 1. Create a Circuit Breaker

```go
// Use default configuration
cb := resilience.MustNewCircuitBreaker("my-service", resilience.DefaultCBConfig())

// Or customize it
config := resilience.CBConfig{
    FailureThreshold:      5,  // Open after 5 failures
    SuccessThreshold:      2,  // Close after 2 successes in half-open
    Timeout:               30 * time.Second,
    MaxConcurrentRequests: 1,  // In half-open state
}
cb := resilience.MustNewCircuitBreaker("my-service", config)
```

### 2. Protect Your Operations

```go
err := cb.Execute(context.Background(), func() error {
    // Your operation here
    resp, err := http.Get("https://api.example.com")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 500 {
        return fmt.Errorf("server error: %d", resp.StatusCode)
    }
    return nil
})

if err != nil {
    if errors.Is(err, resilience.ErrCircuitOpen) {
        // Circuit is open, use fallback or return error
        return useFallback()
    }
    return err
}
```

### 3. Check Circuit Status

```go
// Get current state
state := cb.State() // "closed", "open", or "half-open"

// Get detailed statistics
stats := cb.Stats()
fmt.Printf("Total requests: %d\n", stats.TotalRequests)
fmt.Printf("Success rate: %.2f%%\n",
    float64(stats.TotalSuccesses)/float64(stats.TotalRequests)*100)
fmt.Printf("Current state: %s\n", stats.State)
```

## Common Patterns

### HTTP Client with Fallback

```go
func fetchData(url string, cb *resilience.CircuitBreaker) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var data []byte
    err := cb.Execute(ctx, func() error {
        resp, err := http.Get(url)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        data, err = io.ReadAll(resp.Body)
        return err
    })

    if err != nil {
        if errors.Is(err, resilience.ErrCircuitOpen) {
            // Return cached data
            return getCachedData(url)
        }
        return nil, err
    }

    return data, nil
}
```

### Database Queries

```go
type Database struct {
    db *sql.DB
    cb *resilience.CircuitBreaker
}

func NewDatabase(dsn string) (*Database, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }

    config := resilience.CBConfig{
        FailureThreshold: 3,
        SuccessThreshold: 2,
        Timeout:          60 * time.Second,
    }

    return &Database{
        db: db,
        cb: resilience.MustNewCircuitBreaker("database", config),
    }, nil
}

func (d *Database) Query(ctx context.Context, query string) (*sql.Rows, error) {
    var rows *sql.Rows

    err := d.cb.Execute(ctx, func() error {
        var err error
        rows, err = d.db.QueryContext(ctx, query)
        return err
    })

    return rows, err
}
```

### Multiple Services

```go
// Create a manager for multiple circuit breakers
manager := resilience.NewMultiCircuitBreaker(resilience.DefaultCBConfig())

// Use different circuit breakers for different services
err := manager.Execute(ctx, "auth-service", func() error {
    return callAuthService()
})

err = manager.Execute(ctx, "payment-service", func() error {
    return callPaymentService()
})

// Check health of all services
health := manager.GetHealthStatus()
fmt.Printf("Healthy: %v\n", health.Healthy)
fmt.Printf("Degraded: %v\n", health.Degraded)
fmt.Printf("Unhealthy: %v\n", health.Unhealthy)
```

### With Retry Logic

```go
cbConfig := resilience.CBConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
}

retryConfig := resilience.RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  100 * time.Millisecond,
    MaxDelay:      5 * time.Second,
    BackoffFactor: 2.0,
}

cbr := resilience.NewCircuitBreakerWithRetry("api", cbConfig, retryConfig)

err := cbr.Execute(ctx, func() error {
    return callAPI()
})
```

### With Fallback

```go
cbf := resilience.NewCircuitBreakerWithFallback(
    "cache",
    resilience.DefaultCBConfig(),
    func() (string, error) {
        // Fallback: return default value
        return "default-value", nil
    },
)

result, err := cbf.Execute(ctx, func() (string, error) {
    return cache.Get("key")
})
// result will be "default-value" if cache fails
```

### State Change Monitoring

```go
config := resilience.CBConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
    OnStateChange: func(name string, from resilience.State, to resilience.State) {
        // Log state changes
        log.Printf("[%s] Circuit breaker: %s -> %s", name, from, to)

        // Send alerts
        if to == resilience.StateOpen {
            alert.Send(fmt.Sprintf("Circuit breaker %s opened", name))
        }

        // Update metrics
        metrics.CircuitBreakerState.WithLabelValues(name).Set(float64(to))
    },
}

cb := resilience.MustNewCircuitBreaker("critical-service", config)
```

## Configuration Guidelines

### For Fast-Failing Services (e.g., Internal APIs)

```go
config := resilience.CBConfig{
    FailureThreshold:      3,   // Fail fast
    SuccessThreshold:      2,
    Timeout:               10 * time.Second,
    MaxConcurrentRequests: 5,
}
```

### For External Services (e.g., Third-party APIs)

```go
config := resilience.CBConfig{
    FailureThreshold:      10,  // More tolerant
    SuccessThreshold:      3,
    Timeout:               60 * time.Second,
    MaxConcurrentRequests: 3,
}
```

### For Database Connections

```go
config := resilience.CBConfig{
    FailureThreshold:      5,
    SuccessThreshold:      2,
    Timeout:               30 * time.Second,
    MaxConcurrentRequests: 10,  // Allow more concurrent queries
}
```

### For Cache Services

```go
config := resilience.CBConfig{
    FailureThreshold:      3,
    SuccessThreshold:      1,   // Recover quickly
    Timeout:               5 * time.Second,
    MaxConcurrentRequests: 20,
}
```

## Error Handling

The circuit breaker returns specific errors that you should handle:

```go
err := cb.Execute(ctx, fn)

switch {
case errors.Is(err, resilience.ErrCircuitOpen):
    // Circuit is open, requests are being rejected
    // Use fallback or return cached data
    return useFallback()

case errors.Is(err, resilience.ErrTooManyRequests):
    // Too many concurrent requests in half-open state
    // Retry after a delay
    time.Sleep(100 * time.Millisecond)
    return retry()

case errors.Is(err, resilience.ErrCircuitBreakerTimeout):
    // Operation timed out
    return handleTimeout()

default:
    // Other error from your function
    return err
}
```

## Testing

### Unit Tests

```go
func TestMyServiceWithCircuitBreaker(t *testing.T) {
    config := resilience.CBConfig{
        FailureThreshold: 2,
        SuccessThreshold: 1,
        Timeout:          100 * time.Millisecond,
    }
    cb := resilience.MustNewCircuitBreaker("test", config)

    // Test successful execution
    err := cb.Execute(context.Background(), func() error {
        return nil
    })
    assert.NoError(t, err)

    // Test circuit opening after failures
    for i := 0; i < 2; i++ {
        cb.Execute(context.Background(), func() error {
            return errors.New("error")
        })
    }

    assert.Equal(t, "open", cb.State())
}
```

### Integration Tests

```go
func TestServiceIntegration(t *testing.T) {
    cb := resilience.MustNewCircuitBreaker("integration", resilience.DefaultCBConfig())

    // Simulate real service calls
    for i := 0; i < 100; i++ {
        err := cb.Execute(context.Background(), func() error {
            return callRealService()
        })

        if err != nil && !errors.Is(err, resilience.ErrCircuitOpen) {
            t.Errorf("unexpected error: %v", err)
        }
    }

    stats := cb.Stats()
    t.Logf("Stats: %+v", stats)
}
```

## Monitoring

### Health Check Endpoint

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := manager.GetHealthStatus()

    status := "healthy"
    statusCode := http.StatusOK

    if len(health.Unhealthy) > 0 {
        status = "unhealthy"
        statusCode = http.StatusServiceUnavailable
    } else if len(health.Degraded) > 0 {
        status = "degraded"
        statusCode = http.StatusOK
    }

    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":    status,
        "healthy":   health.Healthy,
        "degraded":  health.Degraded,
        "unhealthy": health.Unhealthy,
    })
}
```

### Metrics Endpoint

```go
func metricsHandler(w http.ResponseWriter, r *http.Request) {
    allCBs := manager.GetAll()

    metrics := make(map[string]resilience.CBStats)
    for name, cb := range allCBs {
        metrics[name] = cb.Stats()
    }

    json.NewEncoder(w).Encode(metrics)
}
```

## Best Practices

1. **One Circuit Breaker per Service**: Don't share circuit breakers across different services
2. **Set Appropriate Timeouts**: Match your service's expected response time
3. **Monitor State Changes**: Log or alert on state transitions
4. **Implement Fallbacks**: Always have a fallback strategy
5. **Test in Production-like Conditions**: Use realistic failure scenarios
6. **Use Context Timeouts**: Always pass contexts with timeouts
7. **Don't Panic on Open Circuit**: Handle `ErrCircuitOpen` gracefully
8. **Review Thresholds Regularly**: Adjust based on actual service behavior

## Next Steps

- Read the full [README.md](./README.md) for detailed documentation
- Check [example_test.go](./example_test.go) for more examples
- Review [patterns.go](./patterns.go) for advanced patterns
- See [prometheus_example.go](./prometheus_example.go) for monitoring integration

## Need Help?

- Review the test files for comprehensive examples
- Check the inline documentation in the source code
- Open an issue on GitHub if you encounter problems

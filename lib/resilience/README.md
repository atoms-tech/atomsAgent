# Circuit Breaker Resilience Pattern

A robust, production-ready implementation of the Circuit Breaker pattern in Go, providing automatic failure detection and recovery for distributed systems.

## Overview

The Circuit Breaker pattern prevents cascading failures in distributed systems by monitoring for failures and temporarily blocking requests to failing services. This implementation provides three states (Closed, Open, Half-Open) with comprehensive metrics tracking and state management.

## Features

- **Three-State Circuit Breaker**: Closed, Open, and Half-Open states
- **Thread-Safe**: Full concurrency support with mutex-based synchronization
- **Configurable Thresholds**: Customizable failure and success thresholds
- **Automatic Recovery**: Transitions to half-open state after timeout
- **Metrics Tracking**: Comprehensive statistics including latencies and state transitions
- **State Callbacks**: Hook into state changes for monitoring and alerting
- **Panic Recovery**: Gracefully handles panics in protected functions
- **Context Support**: Respects context cancellation and timeouts
- **Bulkhead Pattern**: Limit concurrent requests in half-open state

## Installation

```go
import "github.com/coder/agentapi/lib/resilience"
```

## Quick Start

```go
// Create a circuit breaker with default configuration
cb := resilience.MustNewCircuitBreaker("my-service", resilience.DefaultCBConfig())

// Execute a protected operation
ctx := context.Background()
err := cb.Execute(ctx, func() error {
    // Your operation here
    return callExternalService()
})

if err != nil {
    if errors.Is(err, resilience.ErrCircuitOpen) {
        // Circuit is open, use fallback
        return useFallback()
    }
    return err
}
```

## Configuration

### CBConfig

```go
type CBConfig struct {
    // Number of consecutive failures before opening
    FailureThreshold uint32

    // Number of consecutive successes to close from half-open
    SuccessThreshold uint32

    // Duration to stay open before transitioning to half-open
    Timeout time.Duration

    // Maximum concurrent requests in half-open state
    MaxConcurrentRequests uint32

    // Callback when state changes
    OnStateChange func(name string, from State, to State)
}
```

### Default Configuration

```go
config := resilience.DefaultCBConfig()
// Returns:
// {
//     FailureThreshold: 5,
//     SuccessThreshold: 2,
//     Timeout: 30 seconds,
//     MaxConcurrentRequests: 1,
// }
```

### Custom Configuration

```go
config := resilience.CBConfig{
    FailureThreshold:      10,
    SuccessThreshold:      3,
    Timeout:               60 * time.Second,
    MaxConcurrentRequests: 5,
    OnStateChange: func(name string, from resilience.State, to resilience.State) {
        log.Printf("[%s] State changed: %s -> %s", name, from, to)
    },
}

cb := resilience.MustNewCircuitBreaker("payment-service", config)
```

## Circuit Breaker States

### Closed (Normal Operation)

- All requests are executed normally
- Successes and failures are counted
- After `FailureThreshold` consecutive failures, transitions to **Open**

### Open (Reject Requests)

- All requests are rejected immediately with `ErrCircuitOpen`
- No actual execution occurs (fast fail)
- After `Timeout` duration, transitions to **Half-Open**

### Half-Open (Test Recovery)

- Limited requests are allowed (controlled by `MaxConcurrentRequests`)
- Success → increment success counter
  - After `SuccessThreshold` successes, transitions to **Closed**
- Failure → transitions immediately back to **Open**

## State Transition Diagram

```
         ┌─────────────┐
         │   Closed    │
         │  (Normal)   │
         └──────┬──────┘
                │
                │ FailureThreshold
                │ failures
                ▼
         ┌─────────────┐
         │    Open     │
         │  (Reject)   │
         └──────┬──────┘
                │
                │ Timeout
                │ expires
                ▼
         ┌─────────────┐
         │ Half-Open   │
         │   (Test)    │
         └──────┬──────┘
                │
       ┌────────┴────────┐
       │                 │
   Success           Failure
       │                 │
       ▼                 ▼
   Closed             Open
```

## API Reference

### Creating Circuit Breakers

```go
// With error handling
cb, err := resilience.NewCircuitBreaker("service-name", config)
if err != nil {
    log.Fatal(err)
}

// Panic on error (for convenience)
cb := resilience.MustNewCircuitBreaker("service-name", config)
```

### Executing Operations

```go
err := cb.Execute(ctx, func() error {
    // Protected operation
    return doSomething()
})
```

### Getting State

```go
// As string
state := cb.State() // "closed", "open", or "half-open"

// As enum
stateEnum := cb.StateEnum() // resilience.StateClosed, etc.
```

### Getting Statistics

```go
stats := cb.Stats()
fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
fmt.Printf("Success Rate: %.2f%%\n",
    float64(stats.TotalSuccesses)/float64(stats.TotalRequests)*100)
fmt.Printf("Consecutive Failures: %d\n", stats.ConsecutiveFailures)
fmt.Printf("Last Error: %v\n", stats.LastError)
fmt.Printf("State: %s\n", stats.State)
```

### Resetting Circuit Breaker

```go
// Manually reset to closed state
cb.Reset()
```

## Metrics

The circuit breaker tracks detailed metrics:

```go
type CBStats struct {
    TotalRequests        uint64        // Total number of requests
    TotalSuccesses       uint64        // Total successful requests
    TotalFailures        uint64        // Total failed requests
    ConsecutiveSuccesses uint32        // Current consecutive successes
    ConsecutiveFailures  uint32        // Current consecutive failures
    LastError            error         // Last error encountered
    LastErrorTime        time.Time     // Time of last error
    State                State         // Current state
    StateChangedAt       time.Time     // When state last changed
}
```

### Metrics Snapshot

```go
snapshot := cb.metrics.GetMetrics()
fmt.Printf("Average Latency: %v\n", snapshot.AvgLatency)
fmt.Printf("P95 Latency: %v\n", snapshot.P95Latency)
fmt.Printf("P99 Latency: %v\n", snapshot.P99Latency)
fmt.Printf("Rejected Requests: %d\n", snapshot.RequestsRejected)
```

## Examples

### HTTP Client with Circuit Breaker

```go
func makeHTTPRequest(url string, cb *resilience.CircuitBreaker) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return cb.Execute(ctx, func() error {
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return err
        }

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode >= 500 {
            return fmt.Errorf("server error: %d", resp.StatusCode)
        }

        return nil
    })
}
```

### Database Queries

```go
func queryDatabase(query string, cb *resilience.CircuitBreaker) (*sql.Rows, error) {
    ctx := context.Background()
    var rows *sql.Rows

    err := cb.Execute(ctx, func() error {
        var err error
        rows, err = db.QueryContext(ctx, query)
        return err
    })

    return rows, err
}
```

### With Fallback

```go
func getDataWithFallback(key string, cb *resilience.CircuitBreaker) (string, error) {
    ctx := context.Background()
    var data string

    err := cb.Execute(ctx, func() error {
        var err error
        data, err = cache.Get(key)
        return err
    })

    if err != nil {
        if errors.Is(err, resilience.ErrCircuitOpen) {
            // Circuit is open, use fallback immediately
            return getDatabaseData(key)
        }
        // Other error, try fallback
        return getDatabaseData(key)
    }

    return data, nil
}
```

### Monitoring State Changes

```go
config := resilience.CBConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
    OnStateChange: func(name string, from resilience.State, to resilience.State) {
        // Send alert
        if to == resilience.StateOpen {
            alerting.Send(fmt.Sprintf("Circuit breaker [%s] opened", name))
        }

        // Update metrics
        metrics.CircuitBreakerStateChange.WithLabelValues(name, to.String()).Inc()

        // Log
        log.Printf("[%s] State transition: %s -> %s", name, from, to)
    },
}
```

### Multiple Services

```go
type ServiceRegistry struct {
    circuitBreakers map[string]*resilience.CircuitBreaker
    mu              sync.RWMutex
}

func (r *ServiceRegistry) GetCircuitBreaker(serviceName string) *resilience.CircuitBreaker {
    r.mu.RLock()
    cb, exists := r.circuitBreakers[serviceName]
    r.mu.RUnlock()

    if !exists {
        r.mu.Lock()
        defer r.mu.Unlock()

        // Double-check after acquiring write lock
        if cb, exists = r.circuitBreakers[serviceName]; !exists {
            cb = resilience.MustNewCircuitBreaker(serviceName, resilience.DefaultCBConfig())
            r.circuitBreakers[serviceName] = cb
        }
    }

    return cb
}
```

## Error Handling

The circuit breaker returns specific errors:

```go
// Circuit is open
if errors.Is(err, resilience.ErrCircuitOpen) {
    // Use fallback or return cached data
}

// Too many concurrent requests in half-open state
if errors.Is(err, resilience.ErrTooManyRequests) {
    // Retry later or use fallback
}

// Operation timeout
if errors.Is(err, resilience.ErrCircuitBreakerTimeout) {
    // Handle timeout
}
```

## Best Practices

1. **Name Your Circuit Breakers**: Use descriptive names for monitoring
   ```go
   cb := resilience.MustNewCircuitBreaker("user-service-api", config)
   ```

2. **Set Appropriate Thresholds**: Based on your service characteristics
   ```go
   // For critical, fast-failing services
   config := resilience.CBConfig{
       FailureThreshold: 3,
       Timeout:          10 * time.Second,
   }

   // For less critical, slower services
   config := resilience.CBConfig{
       FailureThreshold: 10,
       Timeout:          60 * time.Second,
   }
   ```

3. **Implement Fallbacks**: Always have a fallback strategy
   ```go
   if errors.Is(err, resilience.ErrCircuitOpen) {
       return getCachedData()
   }
   ```

4. **Monitor State Changes**: Hook into state changes for alerts
   ```go
   OnStateChange: func(name string, from, to resilience.State) {
       if to == resilience.StateOpen {
           sendAlert(name)
       }
   }
   ```

5. **Use Context Timeouts**: Prevent hanging operations
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   cb.Execute(ctx, fn)
   ```

6. **One Circuit Breaker per Service**: Don't share across different services
   ```go
   // Good
   authCB := resilience.MustNewCircuitBreaker("auth-service", config)
   paymentCB := resilience.MustNewCircuitBreaker("payment-service", config)

   // Bad
   sharedCB := resilience.MustNewCircuitBreaker("shared", config)
   ```

## Performance Considerations

- **Thread-Safe**: Uses RWMutex for optimal read performance
- **Lock-Free Reads**: State queries use read locks
- **Non-Blocking Callbacks**: State change callbacks run in goroutines
- **Minimal Overhead**: Fast path for closed state has minimal overhead
- **Bounded Memory**: Latency tracking keeps only last 100 samples

## Testing

Run tests:

```bash
go test ./lib/resilience/...
```

Run with race detector:

```bash
go test -race ./lib/resilience/...
```

Run benchmarks:

```bash
go test -bench=. ./lib/resilience/...
```

## License

Part of the [agentapi](https://github.com/coder/agentapi) project.

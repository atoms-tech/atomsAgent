# Migration Guide: Basic to Enhanced FastMCP HTTP Client

## Overview

This guide helps you migrate from the basic `FastMCPHTTPClient` to the enhanced version with retry logic, DLQ, and metrics.

## Migration Options

### Option 1: Drop-In Replacement (Recommended for New Code)

Replace the basic client with the enhanced version:

**Before:**
```go
import "your-project/lib/mcp"

client := mcp.NewFastMCPHTTPClient("http://localhost:8000")

// Basic calls
result, err := client.CallTool(ctx, clientID, toolName, args)
tools, err := client.ListTools(ctx, clientID)
err := client.Connect(ctx, clientID, config)
```

**After:**
```go
import "your-project/lib/mcp"

client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")

// Enhanced calls with retry
result, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
tools, err := client.ListToolsWithRetry(ctx, clientID)
err := client.ConnectWithRetry(ctx, clientID, config)
```

### Option 2: Gradual Migration (Recommended for Existing Code)

Keep existing code working while gradually adopting enhanced features:

**Step 1: Create enhanced client**
```go
// Replace this
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")

// With this
client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")
```

**Step 2: Use both old and new methods**
```go
// Old method still works (embedded client)
result, err := client.CallTool(ctx, clientID, toolName, args)

// New method with retry
result, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
```

**Step 3: Gradually replace critical paths**
```go
// Critical operations: use enhanced methods
result, err := client.CallToolWithRetry(ctx, clientID, criticalTool, args)

// Non-critical operations: keep using old methods initially
result, err := client.CallTool(ctx, clientID, nonCriticalTool, args)
```

## Feature-by-Feature Migration

### 1. Basic Retry → Enhanced Retry

**Before:**
```go
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
result, err := client.CallTool(ctx, clientID, toolName, args)
// Retries happen automatically but with basic logic
```

**After:**
```go
client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")
result, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
// Retries with exponential backoff + jitter
```

**Benefits:**
- Exponential backoff with jitter prevents thundering herd
- Better handling of retryable vs non-retryable errors
- Comprehensive logging of retry attempts
- Metrics tracking

### 2. Add Dead Letter Queue

**Before:**
```go
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
// Failed operations are lost
```

**After:**
```go
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
dlq := mcp.NewRedisDLQ(redisClient)

client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    "http://localhost:8000",
    dlq,    // Enable DLQ
    nil,    // Use default metrics
)
// Failed operations are stored for inspection/retry
```

**Benefits:**
- Track failed operations
- Manual retry capability
- Root cause analysis
- Production debugging

### 3. Add Metrics

**Before:**
```go
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
// No visibility into retries or failures
```

**After:**
```go
metrics := mcp.InitMCPMetrics("myapp")
client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    "http://localhost:8000",
    nil,      // No DLQ
    metrics,  // Enable metrics
)

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":2112", nil)
```

**Benefits:**
- Prometheus metrics
- Retry rate tracking
- Success/failure rates
- Operation latencies
- Grafana dashboards

### 4. Add Timeout Control

**Before:**
```go
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
result, err := client.CallTool(ctx, clientID, toolName, args)
// Uses default 30s timeout per request
```

**After:**
```go
client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")

// Option 1: Set client-wide timeout
client.SetTimeout(45 * time.Second)

// Option 2: Use context timeout for specific operation
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
result, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
```

**Benefits:**
- Total retry timeout (5 minutes)
- Per-request timeout (configurable)
- Context cancellation support

## Complete Migration Example

### Before (Basic Client)

```go
package main

import (
    "context"
    "log"
    "your-project/lib/mcp"
)

func main() {
    client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
    ctx := context.Background()

    config := mcp.HTTPMCPConfig{
        Transport: "stdio",
        Command:   "npx",
        Args:      []string{"-y", "@modelcontextprotocol/server-everything"},
    }

    if err := client.Connect(ctx, "client-1", config); err != nil {
        log.Fatalf("Connect failed: %v", err)
    }

    result, err := client.CallTool(ctx, "client-1", "echo", map[string]any{
        "message": "Hello!",
    })
    if err != nil {
        log.Fatalf("Call failed: %v", err)
    }

    log.Printf("Result: %+v", result)
}
```

### After (Enhanced Client with All Features)

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/redis/go-redis/v9"
    "your-project/lib/mcp"
)

func main() {
    // 1. Setup Redis for DLQ
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer redisClient.Close()

    // 2. Create DLQ
    dlq := mcp.NewRedisDLQ(redisClient)

    // 3. Initialize metrics
    metrics := mcp.InitMCPMetrics("myapp")

    // 4. Create enhanced client
    client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
        "http://localhost:8000",
        dlq,
        metrics,
    )

    // 5. Expose metrics endpoint
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        log.Println("Metrics server on :2112")
        http.ListenAndServe(":2112", nil)
    }()

    // 6. Setup DLQ cleanup
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        for range ticker.C {
            dlq.Cleanup(context.Background(), 7*24*time.Hour)
        }
    }()

    // 7. Use client with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    config := mcp.HTTPMCPConfig{
        Transport: "stdio",
        Command:   "npx",
        Args:      []string{"-y", "@modelcontextprotocol/server-everything"},
    }

    // 8. Connect with retry
    if err := client.ConnectWithRetry(ctx, "client-1", config); err != nil {
        log.Printf("Connect failed after retries: %v", err)
        // Check DLQ for details
        if failedOps, err := dlq.GetByOperation(ctx, "connect", 10); err == nil {
            log.Printf("Found %d failed connect operations in DLQ", len(failedOps))
        }
        return
    }

    // 9. Call tool with retry
    result, err := client.CallToolWithRetry(ctx, "client-1", "echo", map[string]any{
        "message": "Hello!",
    })
    if err != nil {
        log.Printf("Call failed after retries: %v", err)
        return
    }

    log.Printf("Result: %+v", result)

    // 10. Monitor DLQ
    if stats, err := dlq.GetStats(ctx); err == nil {
        log.Printf("DLQ Stats: %+v", stats)
    }
}
```

## Step-by-Step Migration Plan

### Phase 1: Preparation (Week 1)

1. **Review existing code**
   ```bash
   grep -r "NewFastMCPHTTPClient" .
   grep -r "CallTool\|Connect\|ListTools" .
   ```

2. **Setup infrastructure**
   - Deploy Redis for DLQ
   - Configure Prometheus/Grafana
   - Update monitoring

3. **Run tests**
   ```bash
   go test -v ./lib/mcp/
   ```

### Phase 2: Initial Deployment (Week 2)

1. **Deploy to staging**
   - Replace basic client with enhanced client
   - Enable metrics only (no DLQ yet)
   - Monitor for issues

2. **Update critical paths**
   - Payment processing
   - User authentication
   - Data synchronization

3. **Verify metrics**
   ```promql
   rate(fastmcp_operations_total[5m])
   rate(fastmcp_retry_attempts_total[5m])
   ```

### Phase 3: Full Rollout (Week 3)

1. **Enable DLQ**
   ```go
   dlq := mcp.NewRedisDLQ(redisClient)
   client.SetDeadLetterQueue(dlq)
   ```

2. **Setup monitoring**
   - Configure alerts
   - Create dashboards
   - Document procedures

3. **Deploy to production**
   - Gradual rollout
   - Monitor closely
   - Have rollback plan ready

### Phase 4: Optimization (Week 4)

1. **Analyze metrics**
   - Review retry patterns
   - Optimize timeouts
   - Tune configuration

2. **Process DLQ**
   - Implement manual retry
   - Fix root causes
   - Update documentation

3. **Performance tuning**
   - Benchmark operations
   - Optimize bottlenecks
   - Update configuration

## Configuration Migration

### Environment Variables

**Before:**
```bash
FASTMCP_URL=http://localhost:8000
```

**After:**
```bash
# Service
FASTMCP_BASE_URL=http://localhost:8000

# Redis (for DLQ)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Metrics
METRICS_PORT=2112
METRICS_NAMESPACE=myapp

# Timeouts
FASTMCP_REQUEST_TIMEOUT=30s
FASTMCP_RETRY_TIMEOUT=5m
```

### Docker Compose Migration

**Before:**
```yaml
services:
  app:
    image: myapp:latest
    environment:
      - FASTMCP_URL=http://fastmcp:8000
```

**After:**
```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  app:
    image: myapp:latest
    environment:
      - FASTMCP_BASE_URL=http://fastmcp:8000
      - REDIS_ADDR=redis:6379
      - METRICS_PORT=2112
    ports:
      - "2112:2112"  # Metrics
    depends_on:
      - redis
```

## Testing Migration

### Unit Tests

**Before:**
```go
func TestBasicClient(t *testing.T) {
    server := httptest.NewServer(/* ... */)
    client := mcp.NewFastMCPHTTPClient(server.URL)
    // Test
}
```

**After:**
```go
func TestEnhancedClient(t *testing.T) {
    server := httptest.NewServer(/* ... */)
    client := mcp.NewEnhancedFastMCPHTTPClient(server.URL)

    // Test with retry
    err := client.ConnectWithRetry(ctx, "test-client", config)

    // Verify metrics
    assert.NotNil(t, client.GetMetrics())
}
```

### Integration Tests

Add tests for:
- Retry behavior
- DLQ functionality
- Metrics recording
- Timeout handling

```go
func TestRetryBehavior(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts < 3 {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))

    client := mcp.NewEnhancedFastMCPHTTPClient(server.URL)
    err := client.ConnectWithRetry(ctx, "client", config)

    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)
}
```

## Monitoring Migration

### Prometheus Alerts

Add alerts for:

```yaml
groups:
  - name: fastmcp_enhanced
    rules:
      - alert: HighRetryRate
        expr: rate(fastmcp_retry_attempts_total[5m]) > 10
        annotations:
          summary: High retry rate detected

      - alert: DLQGrowth
        expr: fastmcp_dlq_operations_total > 100
        annotations:
          summary: DLQ is growing

      - alert: LowSuccessRate
        expr: |
          sum(rate(fastmcp_operations_total{status="success"}[5m])) /
          sum(rate(fastmcp_operations_total[5m])) < 0.95
        annotations:
          summary: Success rate below 95%
```

### Grafana Dashboards

Create dashboards for:
1. Retry metrics
2. Success/failure rates
3. DLQ size and trends
4. Operation latencies
5. Backoff delays

## Rollback Plan

If issues occur:

### Quick Rollback (Keep Enhanced Client, Disable Features)

```go
// Disable DLQ
client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    "http://localhost:8000",
    nil,  // No DLQ
    nil,  // No custom metrics
)

// Use basic methods temporarily
result, err := client.CallTool(ctx, clientID, toolName, args)
```

### Full Rollback (Revert to Basic Client)

```go
// Revert to basic client
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
result, err := client.CallTool(ctx, clientID, toolName, args)
```

## Common Issues and Solutions

### Issue 1: Redis Connection Failures

**Problem:** DLQ not working due to Redis connection issues

**Solution:**
```go
// Graceful degradation
if err := redisClient.Ping(ctx).Err(); err != nil {
    log.Printf("Redis unavailable, DLQ disabled: %v", err)
    dlq = nil  // Disable DLQ
}

client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    baseURL,
    dlq,  // Will be nil if Redis unavailable
    metrics,
)
```

### Issue 2: High Retry Rate

**Problem:** Too many retries causing performance issues

**Solution:**
```go
// Add circuit breaker
import "your-project/lib/resilience"

cb := resilience.NewCircuitBreaker(resilience.Config{
    MaxFailures: 5,
    ResetTimeout: 60 * time.Second,
})

err := cb.Execute(func() error {
    _, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
    return err
})
```

### Issue 3: DLQ Growing Too Large

**Problem:** DLQ accumulating too many failed operations

**Solution:**
```go
// Aggressive cleanup
go func() {
    ticker := time.NewTicker(30 * time.Minute)
    for range ticker.C {
        dlq.Cleanup(ctx, 24*time.Hour)  // Cleanup after 1 day
    }
}()

// Alert when DLQ > threshold
if count, _ := dlq.Count(ctx); count > 50 {
    alert("DLQ threshold exceeded")
}
```

## Success Criteria

Migration is successful when:

- ✅ All tests passing
- ✅ Metrics visible in Prometheus
- ✅ DLQ operational (if enabled)
- ✅ Success rate ≥ 95%
- ✅ No increase in error rates
- ✅ Retry patterns as expected
- ✅ Documentation updated
- ✅ Team trained

## Support

For migration help:
1. Review `ENHANCED_RETRY_README.md`
2. Check `QUICK_REFERENCE.md`
3. Run examples: `enhanced_client_example.go`
4. Check test suite: `enhanced_client_test.go`

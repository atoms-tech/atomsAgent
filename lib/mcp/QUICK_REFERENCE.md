# FastMCP Enhanced Client - Quick Reference

## Quick Start

```go
import "your-project/lib/mcp"

// Create client
client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")

// Connect with retry
ctx := context.Background()
err := client.ConnectWithRetry(ctx, "client-1", mcp.HTTPMCPConfig{
    Transport: "stdio",
    Command:   "npx",
    Args:      []string{"-y", "@modelcontextprotocol/server-everything"},
})

// Call tool with retry
result, err := client.CallToolWithRetry(ctx, "client-1", "echo", map[string]any{
    "message": "Hello!",
})

// List tools with retry
tools, err := client.ListToolsWithRetry(ctx, "client-1")
```

## Configuration Cheat Sheet

### Retry Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| Initial Delay | 100ms | First retry delay |
| Max Delay | 30s | Maximum backoff delay |
| Multiplier | 2.0 | Exponential factor |
| Jitter | ±10% | Random variance |
| Max Retries | 3 | Total attempts |
| Retry Timeout | 5min | Total time limit |
| Request Timeout | 30s | Per-request limit |

### Retryable Status Codes

```
429 Too Many Requests
500 Internal Server Error
502 Bad Gateway
503 Service Unavailable
504 Gateway Timeout
```

### Non-Retryable

```
All 4xx except 429
Context cancellation
Marshaling errors
```

## Common Patterns

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
result, err := client.CallToolWithRetry(ctx, "client-1", "tool", args)
```

### With DLQ

```go
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
dlq := mcp.NewRedisDLQ(redisClient)
client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    "http://localhost:8000",
    dlq,
    nil,
)
```

### With Custom Metrics

```go
metrics := mcp.InitMCPMetrics("myapp")
client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
    "http://localhost:8000",
    nil,
    metrics,
)
http.Handle("/metrics", promhttp.Handler())
```

## DLQ Operations

### Store Failed Operation
```go
// Automatic when using *WithRetry methods
```

### List Failed Operations
```go
ops, err := dlq.List(ctx, 100)
```

### Get by Operation Type
```go
connectFailures, err := dlq.GetByOperation(ctx, "connect", 50)
```

### Get by Client ID
```go
clientFailures, err := dlq.GetByClientID(ctx, "client-1", 50)
```

### Get Statistics
```go
stats, err := dlq.GetStats(ctx)
fmt.Printf("Total: %d\n", stats.TotalOperations)
```

### Cleanup Old Entries
```go
err := dlq.Cleanup(ctx, 7*24*time.Hour) // Older than 7 days
```

### Delete Specific Entry
```go
err := dlq.Delete(ctx, "operation-id")
```

## Metrics

### Prometheus Queries

```promql
# Retry rate
rate(fastmcp_retry_attempts_total[5m])

# Success rate
sum(rate(fastmcp_operations_total{status="success"}[5m])) /
sum(rate(fastmcp_operations_total[5m]))

# p95 backoff
histogram_quantile(0.95, rate(fastmcp_retry_backoff_seconds_bucket[5m]))

# DLQ size
fastmcp_dlq_operations_total

# p99 operation latency
histogram_quantile(0.99, rate(fastmcp_operation_duration_seconds_bucket[5m]))
```

### Alert Rules

```yaml
groups:
  - name: fastmcp
    rules:
      - alert: HighRetryRate
        expr: rate(fastmcp_retry_attempts_total[5m]) > 10
        for: 5m
        annotations:
          summary: High retry rate detected

      - alert: DLQGrowth
        expr: fastmcp_dlq_operations_total > 100
        for: 10m
        annotations:
          summary: Dead letter queue is growing

      - alert: LowSuccessRate
        expr: |
          sum(rate(fastmcp_operations_total{status="success"}[5m])) /
          sum(rate(fastmcp_operations_total[5m])) < 0.95
        for: 5m
        annotations:
          summary: Success rate below 95%
```

## Logging Patterns

### Retry Attempt
```
[FastMCP Enhanced] Retry 2/3 for call_tool after 220ms (error: HTTP 503, cumulative time: 350ms)
```

### Max Retries Exceeded
```
[FastMCP Enhanced] Max retries exceeded for connect (total time: 1.2s, last error: HTTP 503)
```

### DLQ Storage
```
[FastMCP Enhanced] Stored failed operation in DLQ: call_tool-client-1-1234567890 (operation: call_tool, error: max retries exceeded)
```

## Error Handling

### Check Error Type

```go
result, err := client.CallToolWithRetry(ctx, "client-1", "tool", args)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        // Timeout
    } else if ctx.Err() == context.Canceled {
        // Cancelled
    } else {
        // Other error (check logs and DLQ)
    }
}
```

### Retry Failed Operation

```go
ops, _ := dlq.List(ctx, 100)
for _, op := range ops {
    if shouldRetry(op) {
        // Reconstruct request and retry
        switch op.Operation {
        case "connect":
            retryConnect(client, op)
        case "call_tool":
            retryCallTool(client, op)
        }
    }
}
```

## Testing

### Unit Test with Mock Server

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(ConnectResponse{Success: true})
}))
defer server.Close()

client := mcp.NewEnhancedFastMCPHTTPClient(server.URL)
err := client.ConnectWithRetry(ctx, "test-client", config)
```

### Integration Test with Redis

```go
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
if err := redisClient.Ping(ctx).Err(); err != nil {
    t.Skip("Redis not available")
}

dlq := mcp.NewRedisDLQ(redisClient)
// Test DLQ operations
```

## Performance

### Benchmarks

```bash
go test -bench=. ./lib/mcp/
```

### Memory Profile

```bash
go test -memprofile=mem.prof ./lib/mcp/
go tool pprof mem.prof
```

### CPU Profile

```bash
go test -cpuprofile=cpu.prof ./lib/mcp/
go tool pprof cpu.prof
```

## Troubleshooting

### High Retry Rate

1. Check FastMCP service health: `curl http://localhost:8000/health`
2. Review network connectivity
3. Examine DLQ for error patterns: `dlq.GetStats(ctx)`
4. Check Prometheus metrics

### DLQ Growing

1. Get statistics: `dlq.GetStats(ctx)`
2. List recent failures: `dlq.List(ctx, 10)`
3. Check for systematic issues
4. Implement manual retry

### Slow Operations

1. Check p99 latency: `fastmcp_operation_duration_seconds{quantile="0.99"}`
2. Review backoff delays: `fastmcp_retry_backoff_seconds`
3. Adjust timeouts if needed
4. Profile the FastMCP service

## Best Practices

### ✅ DO

- Use context with timeout
- Monitor DLQ size
- Set up alerts
- Log aggregation
- Regular DLQ cleanup
- Circuit breaker for critical paths

### ❌ DON'T

- Ignore context cancellation
- Leave DLQ unbounded
- Skip metrics monitoring
- Use infinite retries
- Ignore retry patterns
- Retry non-idempotent operations blindly

## Environment Variables

```bash
# Service
FASTMCP_BASE_URL=http://localhost:8000

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Metrics
METRICS_PORT=2112

# Logging
LOG_LEVEL=info
```

## Docker Compose Example

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  fastmcp:
    image: fastmcp:latest
    ports:
      - "8000:8000"

  app:
    build: .
    environment:
      - FASTMCP_BASE_URL=http://fastmcp:8000
      - REDIS_ADDR=redis:6379
    ports:
      - "2112:2112"  # Metrics
    depends_on:
      - redis
      - fastmcp
```

## References

- Full Documentation: `ENHANCED_RETRY_README.md`
- Implementation Details: `IMPLEMENTATION_SUMMARY.md`
- Examples: `enhanced_client_example.go`
- Tests: `enhanced_client_test.go`

# Prometheus Metrics - Quick Reference

## Initialization

```go
import "github.com/coder/agentapi/lib/metrics"

mr := metrics.NewMetricsRegistry()
```

## HTTP Metrics (Automatic)

```go
// Add middleware to router
router.Use(mr.HTTPMiddleware)

// Expose endpoint
router.Handle("/metrics", mr.HTTPHandler())
router.Get("/metrics/json", mr.JSONHandler())
```

Metrics collected automatically:
- `http_requests_total{method,path,status}`
- `http_request_duration_seconds{method,path}`
- `http_requests_in_flight`
- `http_response_size_bytes{method,path}`

## MCP Metrics

### Connection Tracking
```go
// Connect
err := connect()
mr.RecordMCPConnection("server-name", err == nil)

// Disconnect
mr.RecordMCPDisconnection("server-name")
```

### Operation Timing (Recommended)
```go
done := mr.MCPOperationTimer("server-name", "tool_call")
defer done(success)
// Your operation here
```

### Manual Recording
```go
start := time.Now()
result, err := operation()
duration := time.Since(start)
mr.RecordMCPOperation("server-name", "query", duration, err == nil)

if err != nil {
    mr.RecordMCPError("server-name", "query_error")
}
```

Metrics:
- `mcp_connections_active`
- `mcp_connection_errors_total{mcp_name,error_type}`
- `mcp_operations_total{mcp_name,operation,status}`
- `mcp_operation_duration_seconds{mcp_name,operation}`

## Session Metrics

```go
// Create session
sessionID := "session-123"
mr.RecordSessionCreated(sessionID)

// Delete session (automatically records duration)
mr.RecordSessionDeleted(sessionID)

// Get active count
count := mr.GetActiveSessionCount()
```

Metrics:
- `session_count`
- `session_created_total`
- `session_deleted_total`
- `session_duration_seconds`

## Database Metrics

### With Timer (Recommended)
```go
done := mr.DBQueryTimer("SELECT")
defer done(err)

rows, err := db.Query("SELECT * FROM users")
```

### Manual Recording
```go
start := time.Now()
result, err := db.Exec("INSERT INTO users VALUES (?)", user)
duration := time.Since(start)

mr.RecordDBQuery("INSERT", duration, err)
```

### Connection Pool
```go
stats := db.Stats()
mr.RecordDBConnection(stats.OpenConnections, nil)
```

Metrics:
- `database_query_duration_seconds{query_type}`
- `database_queries_total{query_type,status}`
- `database_connections_active`
- `database_connection_errors_total`

## Cache Metrics

```go
// Record hit/miss
value, exists := cache.Get("key")
if exists {
    mr.RecordCacheHit("cache-name")
} else {
    mr.RecordCacheMiss("cache-name")
}

// With timer
done := mr.CacheOperationTimer("cache-name", "get")
defer done()
value := cache.Get("key")

// Update size
mr.UpdateCacheSize("cache-name", len(cache))
```

Metrics:
- `cache_hits_total{cache_name}`
- `cache_misses_total{cache_name}`
- `cache_operation_duration_seconds{cache_name,operation}`
- `cache_size_items{cache_name}`

## System Metrics

```go
import "runtime"

var m runtime.MemStats
runtime.ReadMemStats(&m)

mr.UpdateSystemMetrics(
    runtime.NumGoroutine(),
    m.Alloc,
    m.HeapAlloc,
)
```

Run in background:
```go
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        mr.UpdateSystemMetrics(runtime.NumGoroutine(), m.Alloc, m.HeapAlloc)
    }
}()
```

Metrics:
- `goroutines_count`
- `memory_allocated_bytes`
- `memory_heap_bytes`

## Context Integration

```go
// Add to context
ctx := metrics.WithMetrics(context.Background(), mr)

// Retrieve from context
if m := metrics.FromContext(ctx); m != nil {
    m.RecordCacheHit("my-cache")
}
```

## Common Prometheus Queries

### HTTP Metrics
```promql
# Request rate
rate(http_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
```

### MCP Metrics
```promql
# Active connections
mcp_connections_active

# Error rate
rate(mcp_connection_errors_total[5m])

# Operation duration P95
histogram_quantile(0.95, rate(mcp_operation_duration_seconds_bucket[5m]))
```

### Session Metrics
```promql
# Active sessions
session_count

# Session duration P95
histogram_quantile(0.95, rate(session_duration_seconds_bucket[5m]))
```

### Database Metrics
```promql
# Query duration P95 by type
histogram_quantile(0.95, sum(rate(database_query_duration_seconds_bucket[5m])) by (le, query_type))

# Query rate
sum(rate(database_queries_total[5m])) by (query_type, status)
```

### Cache Metrics
```promql
# Hit ratio
sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))

# Cache size
cache_size_items
```

## Endpoints

### Prometheus Format (Default)
```bash
curl http://localhost:8080/metrics
```

Output:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/api/users",status="200"} 42
```

### JSON Format
```bash
curl http://localhost:8080/metrics/json
```

Output:
```json
{
  "timestamp": 1640000000,
  "metrics": [
    {
      "name": "session_count",
      "type": "GAUGE",
      "value": 5
    }
  ]
}
```

## Testing

### Run tests
```bash
go test ./lib/metrics/...
```

### Run with coverage
```bash
go test -cover ./lib/metrics/...
```

### Run with race detector
```bash
go test -race ./lib/metrics/...
```

### Run benchmarks
```bash
go test -bench=. ./lib/metrics/...
```

## Performance

- HTTP Middleware: ~692 ns/request
- MCP Operation: ~113 ns/operation
- Cache Hit: ~33 ns/operation
- Total overhead: < 1% for typical workloads

## Best Practices

### ✓ DO
- Use timer patterns for automatic duration tracking
- Keep label cardinality low
- Use consistent cache/server names
- Record errors separately
- Run system metrics collector in background

### ✗ DON'T
- Put user IDs or session IDs in labels
- Create metrics per-item (use consistent names)
- Block on metrics collection
- Skip error recording
- Use long metric names

## Troubleshooting

### Metrics not appearing
```go
// Verify middleware is added
router.Use(mr.HTTPMiddleware)

// Check endpoint works
curl http://localhost:8080/metrics
```

### High memory usage
- Reduce histogram buckets
- Check for high cardinality labels
- Monitor with system metrics

### Slow performance
- Verify async collection for system metrics
- Check benchmark results
- Profile with pprof if needed

## Files

- `prometheus.go` - Core implementation (621 lines)
- `prometheus_test.go` - Test suite (488 lines)
- `example_integration.go` - Examples (394 lines)
- `README.md` - Full documentation (556 lines)
- `INTEGRATION_GUIDE.md` - Integration steps (488 lines)
- `SUMMARY.md` - Implementation summary (385 lines)
- `grafana-dashboard.json` - Dashboard template
- `QUICK_REFERENCE.md` - This file

## Resources

- Prometheus: https://prometheus.io/
- Grafana: https://grafana.com/
- Go Client: https://github.com/prometheus/client_golang

---

**Quick Start**: Initialize → Add Middleware → Expose Endpoint → Use Helpers

# Prometheus Metrics Package

Comprehensive Prometheus metrics implementation for monitoring AgentAPI applications.

## Features

- **HTTP Metrics**: Request counts, latency, response sizes, in-flight requests
- **MCP Metrics**: Connection tracking, operation metrics, error rates
- **Session Metrics**: Active sessions, creation/deletion rates, session duration
- **Database Metrics**: Query latency, connection pool metrics, error rates
- **Cache Metrics**: Hit/miss rates, operation duration, cache size
- **System Metrics**: Goroutine count, memory usage

## Installation

The package uses the official Prometheus client library:

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

## Quick Start

### 1. Initialize Metrics Registry

```go
import "github.com/coder/agentapi/lib/metrics"

// Create a new metrics registry
metricsRegistry := metrics.NewMetricsRegistry()
```

### 2. Add HTTP Middleware

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/coder/agentapi/lib/metrics"
)

router := chi.NewRouter()

// Add metrics middleware
router.Use(metricsRegistry.HTTPMiddleware)

// Your routes
router.Get("/api/users", handleUsers)
```

### 3. Expose Metrics Endpoint

```go
// Prometheus format (default)
router.Handle("/metrics", metricsRegistry.HTTPHandler())

// JSON format (optional)
router.Get("/metrics/json", metricsRegistry.JSONHandler())
```

## Metrics Overview

### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_requests_total` | Counter | method, path, status | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | method, path | Request latency |
| `http_requests_in_flight` | Gauge | - | Current active requests |
| `http_response_size_bytes` | Histogram | method, path | Response size |

### MCP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `mcp_connections_active` | Gauge | - | Active MCP connections |
| `mcp_connection_errors_total` | Counter | mcp_name, error_type | Connection failures |
| `mcp_operations_total` | Counter | mcp_name, operation, status | MCP operations |
| `mcp_operation_duration_seconds` | Histogram | mcp_name, operation | Operation latency |

### Session Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `session_count` | Gauge | - | Active sessions |
| `session_created_total` | Counter | - | Total sessions created |
| `session_deleted_total` | Counter | - | Total sessions deleted |
| `session_duration_seconds` | Histogram | - | Session duration |

### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `database_query_duration_seconds` | Histogram | query_type | Query latency |
| `database_queries_total` | Counter | query_type, status | Query count |
| `database_connections_active` | Gauge | - | Active connections |
| `database_connection_errors_total` | Counter | - | Connection errors |

### Cache Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `cache_hits_total` | Counter | cache_name | Cache hits |
| `cache_misses_total` | Counter | cache_name | Cache misses |
| `cache_operation_duration_seconds` | Histogram | cache_name, operation | Operation latency |
| `cache_size_items` | Gauge | cache_name | Cache size |

### System Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `goroutines_count` | Gauge | - | Active goroutines |
| `memory_allocated_bytes` | Gauge | - | Allocated memory |
| `memory_heap_bytes` | Gauge | - | Heap memory |

## Usage Examples

### HTTP Middleware

The middleware automatically records all HTTP metrics:

```go
router := chi.NewRouter()
router.Use(metricsRegistry.HTTPMiddleware)

router.Get("/api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    // Your handler code
    // Metrics are automatically recorded:
    // - http_requests_total{method="GET",path="/api/users/{id}",status="200"}
    // - http_request_duration_seconds{method="GET",path="/api/users/{id}"}
})
```

### MCP Connection Tracking

```go
// Record successful connection
metricsRegistry.RecordMCPConnection("my-mcp-server", true)

// Record failed connection
metricsRegistry.RecordMCPConnection("my-mcp-server", false)

// Record disconnection
metricsRegistry.RecordMCPDisconnection("my-mcp-server")
```

### MCP Operations with Timer

```go
// Using timer for automatic duration tracking
done := metricsRegistry.MCPOperationTimer("my-mcp-server", "tool_call")
defer done(success) // Call with success status

// Your MCP operation
result, err := mcpServer.CallTool(ctx, "tool_name", params)
success := err == nil
```

### Manual MCP Operation Recording

```go
start := time.Now()
result, err := mcpServer.Query(ctx, query)
duration := time.Since(start)

metricsRegistry.RecordMCPOperation("my-mcp-server", "query", duration, err == nil)

if err != nil {
    metricsRegistry.RecordMCPError("my-mcp-server", "query_error")
}
```

### Session Lifecycle

```go
// Create session
sessionID := "session-abc123"
metricsRegistry.RecordSessionCreated(sessionID)

// Session is active...

// Delete session (automatically records duration)
metricsRegistry.RecordSessionDeleted(sessionID)

// Get active session count
count := metricsRegistry.GetActiveSessionCount()
```

### Database Queries

```go
// Using timer
done := metricsRegistry.DBQueryTimer("SELECT")
defer done(err)

// Execute query
rows, err := db.Query("SELECT * FROM users")
```

```go
// Manual recording
start := time.Now()
_, err := db.Exec("INSERT INTO users VALUES (?)", user)
duration := time.Since(start)

metricsRegistry.RecordDBQuery("INSERT", duration, err)
```

### Cache Operations

```go
// Record cache hit/miss
value, exists := cache.Get("user:123")
if exists {
    metricsRegistry.RecordCacheHit("user-cache")
} else {
    metricsRegistry.RecordCacheMiss("user-cache")
}

// Record cache operation with timer
done := metricsRegistry.CacheOperationTimer("user-cache", "get")
defer done()
value := cache.Get("key")

// Update cache size
metricsRegistry.UpdateCacheSize("user-cache", cache.Len())
```

### System Metrics

```go
import "runtime"

// Collect and update system metrics
var m runtime.MemStats
runtime.ReadMemStats(&m)

metricsRegistry.UpdateSystemMetrics(
    runtime.NumGoroutine(),
    m.Alloc,
    m.HeapAlloc,
)

// Run periodically in a goroutine
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        metricsRegistry.UpdateSystemMetrics(
            runtime.NumGoroutine(),
            m.Alloc,
            m.HeapAlloc,
        )
    }
}()
```

### Using Context

```go
import "context"

// Add metrics to context
ctx := metrics.WithMetrics(context.Background(), metricsRegistry)

// Later, retrieve from context
func myHandler(ctx context.Context) {
    if m := metrics.FromContext(ctx); m != nil {
        m.RecordCacheHit("my-cache")
    }
}
```

## Integration with Existing Code

### With Chi Router

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/cors"
    "github.com/coder/agentapi/lib/metrics"
)

func setupServer() *chi.Mux {
    router := chi.NewRouter()
    metricsRegistry := metrics.NewMetricsRegistry()

    // Middleware
    router.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"*"},
    }))
    router.Use(metricsRegistry.HTTPMiddleware)

    // Routes
    router.Get("/api/users", handleUsers)
    router.Handle("/metrics", metricsRegistry.HTTPHandler())

    return router
}
```

### With Database

```go
import "database/sql"

type Database struct {
    db      *sql.DB
    metrics *metrics.MetricsRegistry
}

func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
    done := d.metrics.DBQueryTimer("SELECT")
    defer done(err)

    rows, err := d.db.Query(query, args...)
    return rows, err
}

func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
    done := d.metrics.DBQueryTimer("EXEC")
    defer done(err)

    result, err := d.db.Exec(query, args...)
    return result, err
}
```

### With MCP Client

```go
type MCPClient struct {
    name    string
    metrics *metrics.MetricsRegistry
}

func (c *MCPClient) Connect() error {
    err := c.actualConnect()
    c.metrics.RecordMCPConnection(c.name, err == nil)
    return err
}

func (c *MCPClient) CallTool(ctx context.Context, tool string, params interface{}) (interface{}, error) {
    done := c.metrics.MCPOperationTimer(c.name, "tool_call")
    defer done(err == nil)

    result, err := c.actualCall(ctx, tool, params)
    if err != nil {
        c.metrics.RecordMCPError(c.name, "tool_call_error")
    }
    return result, err
}

func (c *MCPClient) Close() error {
    err := c.actualClose()
    if err == nil {
        c.metrics.RecordMCPDisconnection(c.name)
    }
    return err
}
```

## Prometheus Configuration

### Scrape Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'agentapi'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Example Queries

```promql
# Request rate
rate(http_requests_total[5m])

# Request latency (95th percentile)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# Active MCP connections
mcp_connections_active

# MCP error rate
rate(mcp_connection_errors_total[5m])

# Session count
session_count

# Database query latency (99th percentile)
histogram_quantile(0.99, rate(database_query_duration_seconds_bucket[5m]))

# Cache hit rate
sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))
```

## Grafana Dashboard

### Example Panel Queries

**HTTP Request Rate:**
```promql
sum(rate(http_requests_total[5m])) by (method)
```

**Response Time P95:**
```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, method))
```

**MCP Connections:**
```promql
mcp_connections_active
```

**Database Query Duration:**
```promql
histogram_quantile(0.95, sum(rate(database_query_duration_seconds_bucket[5m])) by (le, query_type))
```

**Cache Hit Ratio:**
```promql
sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))
```

## JSON Export

The package supports JSON export for custom integrations:

```bash
curl http://localhost:8080/metrics/json
```

Response format:
```json
{
  "timestamp": 1640000000,
  "metrics": [
    {
      "name": "session_count",
      "help": "Current number of active sessions",
      "type": "GAUGE",
      "labels": {},
      "value": 5
    },
    {
      "name": "cache_hits_total",
      "help": "Total number of cache hits by cache name",
      "type": "COUNTER",
      "labels": {
        "cache_name": "user-cache"
      },
      "value": 1234
    }
  ]
}
```

## Best Practices

### 1. Label Cardinality

Avoid high-cardinality labels (user IDs, session IDs in labels):

```go
// Bad - creates unique metric for each user
metrics.RecordCacheHit("user-" + userID)

// Good - use consistent cache names
metrics.RecordCacheHit("user-cache")
```

### 2. Metric Naming

Follow Prometheus naming conventions:
- Use base unit (seconds, bytes)
- Add `_total` suffix for counters
- Use descriptive names

### 3. Histogram Buckets

Customize buckets based on your use case:

```go
// For very fast operations
Buckets: []float64{.0001, .0005, .001, .005, .01, .05}

// For API calls
Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10}

// For long-running operations
Buckets: []float64{1, 5, 10, 30, 60, 120, 300}
```

### 4. Timer Pattern

Always use defer with timers:

```go
done := metrics.MCPOperationTimer("server", "operation")
defer done(success)

// Your code here
// Timer automatically records duration when function returns
```

### 5. Error Handling

Always track errors:

```go
result, err := operation()
if err != nil {
    metrics.RecordMCPError("server", "operation_error")
}
metrics.RecordMCPOperation("server", "operation", duration, err == nil)
```

## Testing

Run tests:

```bash
go test ./lib/metrics/...
```

Run benchmarks:

```bash
go test -bench=. ./lib/metrics/...
```

## Performance

The metrics package is designed for minimal overhead:

- Counters: ~50-100ns per operation
- Histograms: ~200-500ns per observation
- HTTP middleware: ~1-2µs per request

## License

See LICENSE file in the repository root.

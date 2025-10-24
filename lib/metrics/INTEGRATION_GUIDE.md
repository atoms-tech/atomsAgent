# Metrics Integration Guide for AgentAPI

This guide shows how to integrate Prometheus metrics into the existing AgentAPI codebase.

## Quick Integration Steps

### 1. Initialize Metrics in Server Setup

In `lib/httpapi/server.go`, add metrics initialization:

```go
import "github.com/coder/agentapi/lib/metrics"

type Server struct {
    router       chi.Router
    api          huma.API
    port         int
    srv          *http.Server
    mu           sync.RWMutex
    logger       *slog.Logger
    conversation *st.Conversation
    agentio      *termexec.Process
    agentType    mf.AgentType
    emitter      *EventEmitter
    chatBasePath string
    tempDir      string
    metrics      *metrics.MetricsRegistry  // Add this field
}

func NewServer(config ServerConfig) (*Server, error) {
    // ... existing code ...

    // Initialize metrics
    metricsRegistry := metrics.NewMetricsRegistry()

    s := &Server{
        // ... existing fields ...
        metrics: metricsRegistry,
    }

    // Setup router with metrics middleware
    s.setupRouter()

    // Start system metrics collection
    go s.collectSystemMetrics()

    return s, nil
}
```

### 2. Add Metrics Middleware to Router

```go
func (s *Server) setupRouter() {
    s.router = chi.NewRouter()

    // CORS (existing)
    s.router.Use(cors.Handler(cors.Options{
        AllowedOrigins: s.allowedOrigins,
        // ... other options ...
    }))

    // Add metrics middleware
    s.router.Use(s.metrics.HTTPMiddleware)

    // Add metrics to context for handlers
    s.router.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := metrics.WithMetrics(r.Context(), s.metrics)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    })

    // ... rest of your middleware and routes ...
}
```

### 3. Expose Metrics Endpoints

```go
func (s *Server) setupRoutes() {
    // ... existing routes ...

    // Metrics endpoints
    s.router.Handle("/metrics", s.metrics.HTTPHandler())
    s.router.Get("/metrics/json", s.metrics.JSONHandler())

    // Health check (excluded from detailed metrics)
    s.router.Get("/health", s.handleHealth)
}
```

### 4. Add System Metrics Collection

```go
import "runtime"

func (s *Server) collectSystemMetrics() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            var m runtime.MemStats
            runtime.ReadMemStats(&m)

            s.metrics.UpdateSystemMetrics(
                runtime.NumGoroutine(),
                m.Alloc,
                m.HeapAlloc,
            )
        }
    }
}
```

## Integration with Existing Components

### MCP Integration

In `lib/mcp/client.go` or wherever you manage MCP connections:

```go
type MCPManager struct {
    servers map[string]*MCPServer
    metrics *metrics.MetricsRegistry
}

func (m *MCPManager) Connect(name string) error {
    start := time.Now()
    err := m.actualConnect(name)

    // Record connection attempt
    m.metrics.RecordMCPConnection(name, err == nil)

    if err != nil {
        m.metrics.RecordMCPError(name, "connection_failed")
        return err
    }

    return nil
}

func (m *MCPManager) CallTool(name string, tool string, params interface{}) (interface{}, error) {
    done := m.metrics.MCPOperationTimer(name, "tool_call")
    defer func() {
        done(err == nil)
    }()

    result, err := m.actualCall(name, tool, params)
    if err != nil {
        m.metrics.RecordMCPError(name, "tool_call_error")
    }

    return result, err
}

func (m *MCPManager) Disconnect(name string) {
    m.actualDisconnect(name)
    m.metrics.RecordMCPDisconnection(name)
}
```

### Session Integration

In `lib/session/manager.go`:

```go
type SessionManager struct {
    sessions map[string]*Session
    metrics  *metrics.MetricsRegistry
    mu       sync.RWMutex
}

func (sm *SessionManager) CreateSession(ctx context.Context, userID string) (*Session, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session := &Session{
        ID:        generateSessionID(),
        UserID:    userID,
        CreatedAt: time.Now(),
    }

    sm.sessions[session.ID] = session

    // Record session creation
    sm.metrics.RecordSessionCreated(session.ID)

    return session, nil
}

func (sm *SessionManager) DeleteSession(sessionID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    if _, exists := sm.sessions[sessionID]; !exists {
        return errors.New("session not found")
    }

    delete(sm.sessions, sessionID)

    // Record session deletion (automatically tracks duration)
    sm.metrics.RecordSessionDeleted(sessionID)

    return nil
}
```

### Database Integration

In your database wrapper (e.g., `lib/db/database.go`):

```go
type Database struct {
    db      *sql.DB
    metrics *metrics.MetricsRegistry
}

func (d *Database) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    done := d.metrics.DBQueryTimer("SELECT")
    defer func() {
        done(err)
    }()

    rows, err := d.db.QueryContext(ctx, query, args...)

    // Update connection pool metrics
    stats := d.db.Stats()
    d.metrics.RecordDBConnection(stats.OpenConnections, nil)

    return rows, err
}

func (d *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    // Determine query type
    queryType := "EXEC"
    upperQuery := strings.ToUpper(strings.TrimSpace(query))
    if strings.HasPrefix(upperQuery, "INSERT") {
        queryType = "INSERT"
    } else if strings.HasPrefix(upperQuery, "UPDATE") {
        queryType = "UPDATE"
    } else if strings.HasPrefix(upperQuery, "DELETE") {
        queryType = "DELETE"
    }

    done := d.metrics.DBQueryTimer(queryType)
    defer func() {
        done(err)
    }()

    result, err := d.db.ExecContext(ctx, query, args...)
    return result, err
}
```

### Cache Integration

If you have a cache implementation:

```go
type Cache struct {
    data    map[string]interface{}
    metrics *metrics.MetricsRegistry
    mu      sync.RWMutex
    name    string
}

func (c *Cache) Get(key string) (interface{}, bool) {
    done := c.metrics.CacheOperationTimer(c.name, "get")
    defer done()

    c.mu.RLock()
    defer c.mu.RUnlock()

    value, exists := c.data[key]

    if exists {
        c.metrics.RecordCacheHit(c.name)
    } else {
        c.metrics.RecordCacheMiss(c.name)
    }

    return value, exists
}

func (c *Cache) Set(key string, value interface{}) {
    done := c.metrics.CacheOperationTimer(c.name, "set")
    defer done()

    c.mu.Lock()
    defer c.mu.Unlock()

    c.data[key] = value
    c.metrics.UpdateCacheSize(c.name, len(c.data))
}
```

## Environment Configuration

No environment variables are required for basic operation. However, you can configure Prometheus scraping:

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'agentapi'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

## Docker Compose Integration

If using Docker Compose, add Prometheus:

```yaml
version: '3.8'

services:
  agentapi:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - SUPABASE_URL=${SUPABASE_URL}

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
    depends_on:
      - prometheus

volumes:
  prometheus-data:
  grafana-data:
```

## Verification

### 1. Check Metrics Endpoint

```bash
curl http://localhost:8080/metrics
```

Should return Prometheus-formatted metrics.

### 2. Check JSON Endpoint

```bash
curl http://localhost:8080/metrics/json | jq
```

Should return JSON-formatted metrics.

### 3. Test Prometheus Scraping

1. Start Prometheus: `docker-compose up -d prometheus`
2. Open http://localhost:9090
3. Query: `http_requests_total`

### 4. Setup Grafana Dashboard

1. Open http://localhost:3000 (admin/admin)
2. Add Prometheus data source: http://prometheus:9090
3. Import dashboard or create new panels with the queries from README.md

## Performance Impact

Based on benchmarks:

- **HTTP Middleware**: ~692 ns/request (~0.0007ms)
- **MCP Operation Recording**: ~113 ns/operation
- **Cache Hit Recording**: ~33 ns/operation

Total overhead is negligible (< 1% for typical workloads).

## Troubleshooting

### Metrics not appearing

1. Verify middleware is registered:
   ```go
   router.Use(metrics.HTTPMiddleware)
   ```

2. Check metrics endpoint is accessible:
   ```bash
   curl -v http://localhost:8080/metrics
   ```

3. Verify Prometheus scrape config is correct

### High cardinality warnings

If you see high cardinality warnings:

1. Review your labels - avoid user IDs, session IDs, etc.
2. Use the path sanitization built into HTTPMiddleware
3. Use consistent cache names instead of per-item names

### Memory usage

Metrics use minimal memory (~1-2MB for typical applications). If concerned:

1. Reduce histogram buckets
2. Reduce label count
3. Monitor with system metrics

## Next Steps

1. **Set up Prometheus**: Follow Docker Compose section
2. **Create Grafana dashboards**: Use example queries from README.md
3. **Set up alerts**: Configure Prometheus alerting rules
4. **Monitor in production**: Start with basic metrics, expand as needed

## Example Alerts

### prometheus-alerts.yml

```yaml
groups:
  - name: agentapi
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
          /
          sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }}%"

      - alert: HighMCPErrorRate
        expr: rate(mcp_connection_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "MCP connection errors detected"

      - alert: SlowDatabaseQueries
        expr: |
          histogram_quantile(0.95,
            rate(database_query_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database queries are slow"
          description: "P95 latency is {{ $value }}s"
```

## Support

For issues or questions:
- Check the README.md for usage examples
- Review test files for integration patterns
- Open an issue on GitHub

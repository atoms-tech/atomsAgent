# Health Check System

A comprehensive health check system for monitoring application components with support for Kubernetes probes, caching, timeout handling, and detailed error reporting.

## Features

- **Multiple Health Checks**: Database, FastMCP, filesystem, and memory checks
- **Kubernetes Support**: Ready, live, and health endpoints
- **Timeout Handling**: 5-second timeout per check
- **Result Caching**: 10-second cache to prevent excessive checking
- **Concurrent Execution**: All checks run in parallel
- **Detailed Reporting**: JSON response with individual check status
- **Extensible**: Easy to add custom health checks

## Quick Start

```go
package main

import (
    "database/sql"
    "log"
    "net/http"

    "github.com/coder/agentapi/lib/health"
    "github.com/coder/agentapi/lib/mcp"
)

func main() {
    // Initialize components
    db, _ := sql.Open("sqlite3", "./app.db")
    defer db.Close()

    fastmcpClient, _ := mcp.NewFastMCPClient()
    defer fastmcpClient.Close()

    // Create health checker
    checker := health.NewHealthChecker(db, fastmcpClient)

    // Create HTTP handler
    handler := health.NewHandler(checker)

    // Register routes
    mux := http.NewServeMux()
    handler.RegisterRoutes(mux)

    // Start server
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Endpoints

### GET /health

Returns detailed health status with individual check results.

**Response (200 OK):**
```json
{
  "status": "UP",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": {
      "name": "database",
      "status": "UP",
      "duration_ms": 5
    },
    "fastmcp": {
      "name": "fastmcp",
      "status": "UP",
      "duration_ms": 2
    },
    "filesystem": {
      "name": "filesystem",
      "status": "UP",
      "duration_ms": 1
    },
    "memory": {
      "name": "memory",
      "status": "UP",
      "duration_ms": 0
    }
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "DOWN",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": {
      "name": "database",
      "status": "DOWN",
      "error": "database ping failed: connection refused",
      "duration_ms": 5001
    }
  }
}
```

### GET /ready

Kubernetes readiness probe. Returns 200 if ready, 503 if not ready.

**Response (200 OK):**
```
OK
```

**Response (503 Service Unavailable):**
```
Service Unavailable
```

### GET /live

Kubernetes liveness probe. Always returns 200 if the process is running.

**Response (200 OK):**
```
OK
```

## Health Check Status

- **UP**: Component is healthy
- **DOWN**: Component is unhealthy (critical failure)
- **DEGRADED**: Component is partially functional (not currently used by default checks)

## Built-in Checks

### DatabaseCheck

Verifies database connectivity by:
1. Pinging the database connection
2. Executing a simple test query (`SELECT 1`)

```go
checker.RegisterCheck("database", &health.DatabaseCheck{
    db: dbConnection,
})
```

### FastMCPCheck

Verifies FastMCP client is available and initialized.

```go
checker.RegisterCheck("fastmcp", &health.FastMCPCheck{
    client: fastmcpClient,
})
```

### FileSystemCheck

Verifies filesystem access by:
1. Reading the current directory
2. Creating and deleting a temporary file

```go
checker.RegisterCheck("filesystem", &health.FileSystemCheck{})
```

### MemoryCheck

Monitors memory usage with configurable thresholds.

```go
checker.RegisterCheck("memory", &health.MemoryCheck{
    MaxMemoryUsagePercent: 90.0, // Alert at 90% usage
})
```

## Custom Health Checks

Implement the `HealthCheck` interface:

```go
type HealthCheck interface {
    Check(ctx context.Context) error
}
```

### Example: Redis Health Check

```go
type RedisCheck struct {
    client *redis.Client
}

func (rc *RedisCheck) Check(ctx context.Context) error {
    return rc.client.Ping(ctx).Err()
}

// Register the check
checker.RegisterCheck("redis", &RedisCheck{client: redisClient})
```

### Example: HTTP API Health Check

```go
checker.RegisterCheck("external-api", health.HealthCheckFunc(func(ctx context.Context) error {
    client := &http.Client{Timeout: 3 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/health", nil)
    if err != nil {
        return err
    }

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    return nil
}))
```

## Kubernetes Integration

### Deployment Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi
spec:
  template:
    spec:
      containers:
      - name: agentapi
        image: agentapi:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 5
          failureThreshold: 3
```

### Probe Differences

- **Liveness Probe** (`/live`): Determines if the container should be restarted
  - Always returns 200 if the process is running
  - Failure triggers container restart

- **Readiness Probe** (`/ready`): Determines if the container should receive traffic
  - Returns 200 if system is ready (UP or DEGRADED)
  - Returns 503 if system is DOWN
  - Failure removes pod from service endpoints

## Configuration

### Timeout Configuration

Default timeout for each check is 5 seconds. This is defined by the `CheckTimeout` constant.

```go
const CheckTimeout = 5 * time.Second
```

### Cache Configuration

Results are cached for 10 seconds by default to prevent excessive checking.

```go
const CacheDuration = 10 * time.Second
```

### Custom Timeouts with Middleware

```go
mux.HandleFunc("/health", health.WithTimeout(10*time.Second, handler.Health))
```

## Programmatic Usage

### Check Overall Health

```go
ctx := context.Background()
status := checker.Check(ctx)

if status.Overall == health.StatusDown {
    log.Println("System is down!")
    // Send alert
}
```

### Check Readiness

```go
if !checker.Ready(ctx) {
    log.Println("System not ready, delaying startup")
    time.Sleep(5 * time.Second)
}
```

### Monitor Individual Checks

```go
status := checker.Check(ctx)

for name, check := range status.Checks {
    if check.Status != health.StatusUp {
        log.Printf("Check %s failed: %s", name, check.Error)

        // Send metric to monitoring system
        sendMetric("health_check_status", map[string]interface{}{
            "component": name,
            "status": check.Status,
            "duration_ms": check.Duration.Milliseconds(),
        })
    }
}
```

## Monitoring Integration

### Prometheus Metrics

```go
// Example Prometheus metrics
var (
    healthCheckDuration = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "health_check_duration_milliseconds",
            Help: "Duration of health checks in milliseconds",
        },
        []string{"check"},
    )

    healthCheckStatus = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "health_check_status",
            Help: "Status of health checks (1 = UP, 0 = DOWN)",
        },
        []string{"check"},
    )
)

// Update metrics
status := checker.Check(ctx)
for name, check := range status.Checks {
    healthCheckDuration.WithLabelValues(name).Set(float64(check.Duration.Milliseconds()))

    var statusValue float64
    if check.Status == health.StatusUp {
        statusValue = 1.0
    }
    healthCheckStatus.WithLabelValues(name).Set(statusValue)
}
```

### Periodic Health Checks

```go
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
    status := checker.Check(context.Background())

    // Log to monitoring system
    log.Printf("Health check: overall=%s checks=%d",
        status.Overall, len(status.Checks))
}
```

## Testing

### Running Tests

```bash
go test ./lib/health/...
```

### Test Coverage

```bash
go test -cover ./lib/health/...
```

### Example Test

```go
func TestCustomCheck(t *testing.T) {
    checker := health.NewHealthChecker(nil, nil)

    checker.RegisterCheck("custom", health.HealthCheckFunc(func(ctx context.Context) error {
        return nil // Always healthy
    }))

    status := checker.Check(context.Background())

    if status.Overall != health.StatusUp {
        t.Errorf("Expected UP, got %s", status.Overall)
    }
}
```

## Best Practices

1. **Use Appropriate Timeouts**: Set realistic timeouts for external service checks
2. **Avoid Expensive Checks**: Keep checks lightweight and fast
3. **Cache Results**: Leverage the built-in caching to prevent excessive checking
4. **Monitor Check Duration**: Alert on slow health checks
5. **Separate Liveness and Readiness**: Use `/live` for process health, `/ready` for service health
6. **Handle Degraded State**: Consider implementing degraded status for partial failures
7. **Log Failures**: Always log health check failures for debugging
8. **Test Your Checks**: Write tests for custom health checks

## Troubleshooting

### Health Check Timeouts

If checks are timing out:
- Verify external services are responsive
- Check network connectivity
- Review timeout configuration
- Consider increasing `CheckTimeout`

### Database Check Failures

Common causes:
- Connection pool exhausted
- Database unreachable
- Authentication issues
- Firewall blocking connections

### Memory Check Failures

If memory checks fail:
- Review memory usage patterns
- Check for memory leaks
- Adjust `MaxMemoryUsagePercent` threshold
- Monitor garbage collection metrics

## API Reference

### Types

```go
type Status string
const (
    StatusUp       Status = "UP"
    StatusDown     Status = "DOWN"
    StatusDegraded Status = "DEGRADED"
)

type HealthCheck interface {
    Check(ctx context.Context) error
}

type HealthChecker struct { ... }
type CheckStatus struct { ... }
type HealthStatus struct { ... }
```

### Functions

```go
func NewHealthChecker(db *sql.DB, fastmcpClient *mcp.FastMCPClient) *HealthChecker
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck)
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus
func (hc *HealthChecker) Ready(ctx context.Context) bool
```

## License

See the main project LICENSE file.

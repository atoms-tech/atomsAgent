# Health Check Integration Guide

This guide shows how to integrate the health check system into your AgentAPI application.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Integration with Existing Server](#integration-with-existing-server)
3. [Adding to HTTP API](#adding-to-http-api)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Monitoring Setup](#monitoring-setup)
6. [Custom Checks](#custom-checks)

## Quick Start

### 1. Basic Setup

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"

    "github.com/coder/agentapi/lib/health"
    "github.com/coder/agentapi/lib/mcp"
)

func main() {
    // Initialize your dependencies
    db, err := sql.Open("sqlite3", "./app.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    fastmcpClient, err := mcp.NewFastMCPClient()
    if err != nil {
        log.Fatal(err)
    }
    defer fastmcpClient.Close()

    // Create health checker
    healthChecker := health.NewHealthChecker(db, fastmcpClient)

    // Create HTTP handler
    healthHandler := health.NewHandler(healthChecker)

    // Register routes
    mux := http.NewServeMux()
    healthHandler.RegisterRoutes(mux)

    // Add your application routes
    mux.HandleFunc("/api/v1/status", yourStatusHandler)

    // Start server
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Integration with Existing Server

### Integrating with lib/httpapi/server.go

Add health checks to the existing HTTP server:

```go
// In lib/httpapi/server.go

import (
    "github.com/coder/agentapi/lib/health"
)

// Add to Server struct
type Server struct {
    // ... existing fields ...
    healthChecker *health.HealthChecker
}

// In NewServer function
func NewServer(ctx context.Context, config ServerConfig) (*Server, error) {
    // ... existing initialization ...

    // Initialize health checker
    // Note: You'll need to pass db and fastmcpClient as part of ServerConfig
    healthChecker := health.NewHealthChecker(config.DB, config.FastMCPClient)

    s := &Server{
        // ... existing fields ...
        healthChecker: healthChecker,
    }

    // Register routes including health checks
    s.registerRoutes()

    return s, nil
}

// Update registerRoutes to include health endpoints
func (s *Server) registerRoutes() {
    // ... existing routes ...

    // Add health check endpoints
    healthHandler := health.NewHandler(s.healthChecker)

    // Register on chi router
    s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        healthHandler.Health(w, r)
    })
    s.router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
        healthHandler.Ready(w, r)
    })
    s.router.Get("/live", func(w http.ResponseWriter, r *http.Request) {
        healthHandler.Live(w, r)
    })
}
```

### Update ServerConfig

```go
// In lib/httpapi/server.go

type ServerConfig struct {
    // ... existing fields ...
    DB             *sql.DB
    FastMCPClient  *mcp.FastMCPClient
}
```

### Update cmd/server/server.go

```go
// In cmd/server/server.go

func run(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Initialize database (if not already done)
    db, err := sql.Open("sqlite3", "./agentapi.db")
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    defer db.Close()

    // Initialize FastMCP client (if not already done)
    fastmcpClient, err := mcp.NewFastMCPClient()
    if err != nil {
        return fmt.Errorf("failed to create FastMCP client: %w", err)
    }
    defer fastmcpClient.Close()

    // Create server with health checks
    server, err := httpapi.NewServer(ctx, httpapi.ServerConfig{
        // ... existing config ...
        DB:            db,
        FastMCPClient: fastmcpClient,
    })

    // ... rest of the code ...
}
```

## Adding to HTTP API

### Option 1: Separate Health Server

Run health checks on a separate port for isolation:

```go
func main() {
    // Main application server
    appServer := http.Server{
        Addr:    ":8080",
        Handler: appRouter,
    }

    // Health check server on different port
    healthChecker := health.NewHealthChecker(db, fastmcpClient)
    healthHandler := health.NewHandler(healthChecker)

    healthMux := http.NewServeMux()
    healthHandler.RegisterRoutes(healthMux)

    healthServer := http.Server{
        Addr:    ":8081",
        Handler: healthMux,
    }

    // Start both servers
    go func() {
        log.Fatal(appServer.ListenAndServe())
    }()

    log.Fatal(healthServer.ListenAndServe())
}
```

### Option 2: Same Server, Different Paths

```go
func main() {
    mux := http.NewServeMux()

    // Application routes
    mux.HandleFunc("/api/", apiHandler)

    // Health check routes
    healthChecker := health.NewHealthChecker(db, fastmcpClient)
    healthHandler := health.NewHandler(healthChecker)
    healthHandler.RegisterRoutes(mux)

    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Kubernetes Deployment

### Deployment with Probes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi
  labels:
    app: agentapi
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentapi
  template:
    metadata:
      labels:
        app: agentapi
    spec:
      containers:
      - name: agentapi
        image: agentapi:latest
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP

        # Liveness probe - restart if unhealthy
        livenessProbe:
          httpGet:
            path: /live
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3

        # Readiness probe - remove from service if not ready
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3

        # Startup probe - wait for slow startup
        startupProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 0
          periodSeconds: 10
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 30  # 5 minutes to start

        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

### Service Configuration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agentapi
  labels:
    app: agentapi
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: agentapi
```

### Ingress with Health Checks

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agentapi
  annotations:
    nginx.ingress.kubernetes.io/health-check-path: "/health"
spec:
  rules:
  - host: agentapi.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: agentapi
            port:
              number: 80
```

## Monitoring Setup

### Prometheus Integration

```go
package main

import (
    "context"
    "time"

    "github.com/coder/agentapi/lib/health"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    healthCheckDuration = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "health_check_duration_milliseconds",
            Help: "Duration of health checks in milliseconds",
        },
        []string{"check"},
    )

    healthCheckStatus = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "health_check_status",
            Help: "Status of health checks (1 = UP, 0 = DOWN)",
        },
        []string{"check"},
    )
)

func startHealthMetrics(checker *health.HealthChecker) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        status := checker.Check(context.Background())

        for name, check := range status.Checks {
            // Record duration
            healthCheckDuration.WithLabelValues(name).
                Set(float64(check.Duration.Milliseconds()))

            // Record status
            var statusValue float64
            if check.Status == health.StatusUp {
                statusValue = 1.0
            }
            healthCheckStatus.WithLabelValues(name).Set(statusValue)
        }
    }
}
```

### DataDog Integration

```go
package main

import (
    "context"
    "time"

    "github.com/DataDog/datadog-go/statsd"
    "github.com/coder/agentapi/lib/health"
)

func startHealthMetrics(checker *health.HealthChecker, client *statsd.Client) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        status := checker.Check(context.Background())

        for name, check := range status.Checks {
            // Send gauge metric
            tags := []string{
                "component:" + name,
                "status:" + string(check.Status),
            }

            client.Gauge("health.check.duration",
                float64(check.Duration.Milliseconds()), tags, 1)

            // Send service check
            serviceStatus := statsd.Ok
            if check.Status == health.StatusDown {
                serviceStatus = statsd.Critical
            }

            client.ServiceCheck("health.check",
                serviceStatus, tags, check.Error)
        }
    }
}
```

## Custom Checks

### Database Connection Pool Check

```go
type DBPoolCheck struct {
    db *sql.DB
    maxOpenConns int
}

func (c *DBPoolCheck) Check(ctx context.Context) error {
    stats := c.db.Stats()

    // Check if connection pool is exhausted
    if stats.OpenConnections >= c.maxOpenConns {
        return fmt.Errorf("connection pool exhausted: %d/%d",
            stats.OpenConnections, c.maxOpenConns)
    }

    // Check for connection issues
    if stats.WaitCount > 0 && stats.WaitDuration > 5*time.Second {
        return fmt.Errorf("high connection wait time: %v", stats.WaitDuration)
    }

    return nil
}

// Register
checker.RegisterCheck("db-pool", &DBPoolCheck{
    db: db,
    maxOpenConns: 100,
})
```

### External Service Check

```go
type ExternalServiceCheck struct {
    url string
    timeout time.Duration
}

func (c *ExternalServiceCheck) Check(ctx context.Context) error {
    client := &http.Client{Timeout: c.timeout}

    req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
    if err != nil {
        return err
    }

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    return nil
}

// Register
checker.RegisterCheck("external-api", &ExternalServiceCheck{
    url: "https://api.example.com/health",
    timeout: 3 * time.Second,
})
```

### Disk Space Check

```go
type DiskSpaceCheck struct {
    path string
    minFreeGB int
}

func (c *DiskSpaceCheck) Check(ctx context.Context) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(c.path, &stat); err != nil {
        return fmt.Errorf("failed to get disk stats: %w", err)
    }

    // Calculate free space in GB
    freeGB := int(stat.Bavail * uint64(stat.Bsize) / (1024 * 1024 * 1024))

    if freeGB < c.minFreeGB {
        return fmt.Errorf("low disk space: %dGB free (min: %dGB)",
            freeGB, c.minFreeGB)
    }

    return nil
}

// Register
checker.RegisterCheck("disk-space", &DiskSpaceCheck{
    path: "/",
    minFreeGB: 10,
})
```

## Testing Your Integration

### Unit Tests

```go
func TestHealthCheckIntegration(t *testing.T) {
    // Create test database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Create health checker
    checker := health.NewHealthChecker(db, nil)

    // Verify health
    status := checker.Check(context.Background())
    if status.Overall != health.StatusUp {
        t.Errorf("Expected UP, got %s", status.Overall)
    }
}
```

### Integration Tests

```go
func TestHealthEndpoints(t *testing.T) {
    // Setup server with health checks
    server := setupTestServer(t)
    defer server.Close()

    tests := []struct {
        endpoint string
        wantCode int
    }{
        {"/health", http.StatusOK},
        {"/ready", http.StatusOK},
        {"/live", http.StatusOK},
    }

    for _, tt := range tests {
        resp, err := http.Get(server.URL + tt.endpoint)
        if err != nil {
            t.Fatal(err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != tt.wantCode {
            t.Errorf("%s: got %d, want %d",
                tt.endpoint, resp.StatusCode, tt.wantCode)
        }
    }
}
```

## Best Practices

1. **Use Separate Ports**: Consider running health checks on a different port from your main application
2. **Set Appropriate Timeouts**: Balance between catching real issues and avoiding false positives
3. **Monitor Check Duration**: Alert on slow health checks
4. **Test in Staging**: Verify health checks work correctly before deploying to production
5. **Document Dependencies**: Clearly document what each health check verifies
6. **Handle Graceful Shutdown**: Continue serving health checks during graceful shutdown
7. **Cache Wisely**: Use the built-in caching to avoid overwhelming dependencies

## Troubleshooting

### Health checks timing out

- Check network connectivity to dependencies
- Verify timeout configuration is appropriate
- Look for slow queries or operations

### False positives

- Review check thresholds
- Add retry logic for transient failures
- Consider degraded state for non-critical failures

### Kubernetes pod restarts

- Check liveness probe configuration
- Review failure threshold
- Verify initialDelaySeconds is sufficient

## Additional Resources

- [Kubernetes Probe Documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Health Check Pattern](https://microservices.io/patterns/observability/health-check-api.html)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/instrumentation/)

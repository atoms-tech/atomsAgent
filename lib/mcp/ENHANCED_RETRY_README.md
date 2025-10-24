# Enhanced FastMCP HTTP Client with Retry Logic

This document describes the enhanced retry logic implementation for the FastMCP HTTP client, including exponential backoff with jitter, dead letter queue support, and comprehensive metrics.

## Features

### 1. Exponential Backoff with Jitter

The enhanced client implements sophisticated retry logic with the following characteristics:

- **Initial delay**: 100ms
- **Multiplier**: 2.0 (doubles each retry)
- **Max delay**: 30 seconds
- **Jitter**: ±10% random variance to prevent thundering herd
- **Max retries**: 3 attempts

**Backoff progression**:
- Attempt 1: ~100ms (90-110ms with jitter)
- Attempt 2: ~200ms (180-220ms with jitter)
- Attempt 3: ~400ms (360-440ms with jitter)

### 2. Retry Configuration

**Retryable status codes**:
- `429` - Too Many Requests
- `500` - Internal Server Error
- `502` - Bad Gateway
- `503` - Service Unavailable
- `504` - Gateway Timeout

**Non-retryable**:
- All 4xx errors except 429
- Context cancellation errors
- Request marshaling errors

### 3. Timeout Handling

- **Per-request timeout**: 30 seconds
- **Total retry timeout**: 5 minutes
- **Context cancellation support**: Respects parent context cancellation

### 4. Dead Letter Queue (DLQ)

Failed operations are stored in Redis for manual retry/inspection with:

- **TTL**: 7 days by default
- **Storage format**: JSON with full request details
- **Indexing**: By timestamp, operation type, and client ID
- **Cleanup**: Automatic removal of old entries

### 5. Prometheus Metrics

Comprehensive metrics tracking:

- `fastmcp_retry_attempts_total` - Counter of retry attempts by operation and reason
- `fastmcp_retry_backoff_seconds` - Histogram of backoff delays
- `fastmcp_operations_total` - Counter of operations by type and status
- `fastmcp_operation_duration_seconds` - Histogram of operation duration
- `fastmcp_dlq_operations_total` - Counter of operations sent to DLQ

### 6. Comprehensive Logging

Each retry logs:
- Attempt number
- Operation type
- Backoff duration
- Error details
- Cumulative time elapsed

## Usage

### Basic Usage (Enhanced Client)

```go
package main

import (
	"context"
	"log"
	"time"

	"your-project/lib/mcp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Create enhanced client
	client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")

	// Optional: Configure Redis DLQ
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	dlq := mcp.NewRedisDLQ(redisClient)
	client.SetDeadLetterQueue(dlq)

	ctx := context.Background()

	// Connect with retry
	config := mcp.HTTPMCPConfig{
		Transport: "stdio",
		Command:   "/path/to/mcp/server",
		Args:      []string{"arg1", "arg2"},
	}

	err := client.ConnectWithRetry(ctx, "client-1", config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Call tool with retry
	result, err := client.CallToolWithRetry(ctx, "client-1", "my_tool", map[string]any{
		"param1": "value1",
		"param2": 42,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	log.Printf("Tool result: %+v", result)

	// List tools with retry
	tools, err := client.ListToolsWithRetry(ctx, "client-1")
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	log.Printf("Available tools: %+v", tools)
}
```

### Using with Custom Metrics

```go
package main

import (
	"context"
	"net/http"

	"your-project/lib/mcp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize custom metrics with namespace
	metrics := mcp.InitMCPMetrics("myapp")

	// Create client with custom metrics
	client := mcp.NewEnhancedFastMCPHTTPClientWithOptions(
		"http://localhost:8000",
		nil,      // No DLQ
		metrics,  // Custom metrics
	)

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	// Use client...
	ctx := context.Background()
	// ... operations ...
}
```

### Dead Letter Queue Operations

```go
package main

import (
	"context"
	"log"
	"time"

	"your-project/lib/mcp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Create Redis DLQ
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	dlq := mcp.NewRedisDLQ(redisClient)

	ctx := context.Background()

	// List failed operations
	failedOps, err := dlq.List(ctx, 100)
	if err != nil {
		log.Fatalf("Failed to list DLQ: %v", err)
	}

	for _, op := range failedOps {
		log.Printf("Failed operation: %s, Error: %s, Retries: %d",
			op.Operation, op.LastError, op.RetryCount)
	}

	// Get operations by type
	connectFailures, err := dlq.GetByOperation(ctx, "connect", 50)
	if err != nil {
		log.Fatalf("Failed to get connect failures: %v", err)
	}

	// Get operations by client
	clientFailures, err := dlq.GetByClientID(ctx, "client-1", 50)
	if err != nil {
		log.Fatalf("Failed to get client failures: %v", err)
	}

	// Get DLQ statistics
	stats, err := dlq.GetStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	log.Printf("DLQ Stats: %+v", stats)

	// Cleanup old entries (older than 30 days)
	err = dlq.Cleanup(ctx, 30*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to cleanup DLQ: %v", err)
	}

	// Delete specific operation
	err = dlq.Delete(ctx, "operation-id")
	if err != nil {
		log.Fatalf("Failed to delete operation: %v", err)
	}
}
```

### Context and Timeout Handling

```go
package main

import (
	"context"
	"log"
	"time"

	"your-project/lib/mcp"
)

func main() {
	client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")

	// Set overall timeout for operation (including all retries)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// This will retry up to 3 times or until context timeout
	result, err := client.CallToolWithRetry(ctx, "client-1", "slow_tool", nil)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Operation timed out after all retries")
		} else {
			log.Printf("Operation failed: %v", err)
		}
		return
	}

	log.Printf("Success: %+v", result)
}
```

## Monitoring and Observability

### Prometheus Metrics Example

```promql
# Total retry attempts
sum(rate(fastmcp_retry_attempts_total[5m])) by (operation, reason)

# Average backoff delay
histogram_quantile(0.95, rate(fastmcp_retry_backoff_seconds_bucket[5m]))

# Success rate
sum(rate(fastmcp_operations_total{status="success"}[5m])) /
sum(rate(fastmcp_operations_total[5m]))

# Operations sent to DLQ
rate(fastmcp_dlq_operations_total[5m])

# Operation duration by percentile
histogram_quantile(0.99, rate(fastmcp_operation_duration_seconds_bucket[5m]))
```

### Grafana Dashboard Example

Create dashboards to visualize:

1. **Retry Rate Panel**: Shows retry attempts over time
2. **Backoff Duration Panel**: Histogram of backoff delays
3. **Success Rate Panel**: Percentage of successful operations
4. **DLQ Size Panel**: Number of operations in dead letter queue
5. **Operation Duration Panel**: p50, p95, p99 latencies

## Configuration

### Environment Variables

```bash
# FastMCP service URL
FASTMCP_BASE_URL=http://localhost:8000

# Redis for DLQ
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Metrics endpoint
METRICS_PORT=2112
```

### Redis DLQ Configuration

```go
// Custom TTL for DLQ entries
dlq := mcp.NewRedisDLQWithTTL(redisClient, 14*24*time.Hour) // 14 days

// Run periodic cleanup
go func() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := dlq.Cleanup(context.Background(), 7*24*time.Hour); err != nil {
			log.Printf("DLQ cleanup failed: %v", err)
		}
	}
}()
```

## Best Practices

### 1. Use Context Timeouts

Always use context with timeout to prevent indefinite retries:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### 2. Monitor DLQ Size

Set up alerts for DLQ growth:

```promql
# Alert if DLQ has > 100 operations
fastmcp_dlq_operations_total > 100
```

### 3. Implement Circuit Breaker

For additional resilience, combine with circuit breaker:

```go
import "your-project/lib/resilience"

cb := resilience.NewCircuitBreaker(resilience.Config{
	MaxFailures:  5,
	ResetTimeout: 60 * time.Second,
})

err := cb.Execute(func() error {
	_, err := client.CallToolWithRetry(ctx, "client-1", "tool", nil)
	return err
})
```

### 4. Log Aggregation

Send logs to centralized logging system:

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

// Enhanced client will log to stderr by default
// Configure your log aggregation to capture these
```

### 5. Graceful Degradation

Handle DLQ operations for manual retry:

```go
// Periodic DLQ processor
go func() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ops, _ := dlq.List(ctx, 10)
		for _, op := range ops {
			// Attempt manual retry logic
			if shouldRetry(op) {
				retryFailedOperation(client, op)
			}
		}
	}
}()
```

## Comparison: Basic vs Enhanced Client

| Feature | Basic Client | Enhanced Client |
|---------|-------------|-----------------|
| Exponential Backoff | ✓ | ✓ |
| Jitter | ✗ | ✓ |
| Configurable Retry Logic | Limited | Comprehensive |
| Dead Letter Queue | ✗ | ✓ |
| Prometheus Metrics | ✗ | ✓ |
| Comprehensive Logging | ✗ | ✓ |
| Context Support | ✓ | ✓ |
| Per-Request Timeout | ✓ | ✓ |
| Total Retry Timeout | ✗ | ✓ |
| Status Code Filtering | Basic | Advanced |

## Migration Guide

### From Basic Client to Enhanced Client

1. **Replace client initialization**:

```go
// Before
client := mcp.NewFastMCPHTTPClient("http://localhost:8000")

// After
client := mcp.NewEnhancedFastMCPHTTPClient("http://localhost:8000")
```

2. **Update method calls**:

```go
// Before
result, err := client.CallTool(ctx, clientID, toolName, args)

// After (with retry)
result, err := client.CallToolWithRetry(ctx, clientID, toolName, args)
```

3. **Add DLQ support (optional)**:

```go
dlq := mcp.NewRedisDLQ(redisClient)
client.SetDeadLetterQueue(dlq)
```

4. **Add metrics endpoint (optional)**:

```go
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":2112", nil)
```

## Troubleshooting

### High Retry Rate

If you see high retry rates:

1. Check FastMCP service health
2. Review network connectivity
3. Examine error patterns in DLQ
4. Consider increasing timeouts

### DLQ Growing

If DLQ continues to grow:

1. Investigate root cause of failures
2. Implement manual retry processor
3. Review operation patterns
4. Check for systematic issues

### Slow Operations

If operations are slow:

1. Check p99 latency metrics
2. Review backoff delays
3. Consider adjusting retry configuration
4. Profile the FastMCP service

## License

Same as the parent project.

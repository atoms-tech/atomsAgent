# MCP Circuit Breaker Implementation

This document describes the circuit breaker protection added to the MCP handler in `lib/api/mcp.go`.

## Overview

Circuit breakers have been added to protect all MCP operations from cascading failures and provide graceful degradation when MCP servers are unavailable or experiencing issues.

## Architecture

### Circuit Breaker Structure

```go
type mcpCircuitBreakers struct {
    connect        *resilience.CircuitBreaker
    callTool       *resilience.CircuitBreaker
    listTools      *resilience.CircuitBreaker
    disconnect     *resilience.CircuitBreaker
    testConnection *resilience.CircuitBreaker
}
```

Each operation type has its own dedicated circuit breaker, allowing fine-grained control and isolation of failures.

### Configuration

All circuit breakers are configured with the following parameters:

- **FailureThreshold**: 5 consecutive failures before opening
- **SuccessThreshold**: 2 consecutive successes to close from half-open
- **Timeout**: 30 seconds in open state before transitioning to half-open
- **MaxConcurrentRequests**: 100 requests allowed in half-open state

```go
config := resilience.CBConfig{
    FailureThreshold:      5,
    SuccessThreshold:      2,
    Timeout:               30 * time.Second,
    MaxConcurrentRequests: 100,
    OnStateChange:         onCircuitBreakerStateChange,
}
```

## Protected Operations

### 1. Test Connection (`TestMCPConnection`)

The test connection endpoint wraps both the connection attempt and tool listing in circuit breaker protection:

```go
err := h.breakers.testConnection.Execute(testCtx, func() error {
    connectErr := h.fastmcpClient.ConnectMCP(testCtx, mcpConfig)
    if connectErr != nil {
        return connectErr
    }

    // List tools with circuit breaker protection
    h.breakers.listTools.Execute(testCtx, func() error {
        tools, toolsErr := h.fastmcpClient.ListTools(testCtx, testID)
        // Process tools...
        return toolsErr
    })

    return nil
})
```

### 2. Disconnect (`DeleteMCPConfiguration`)

Disconnect operations are protected when deleting MCP configurations:

```go
disconnectErr := h.breakers.disconnect.Execute(disconnectCtx, func() error {
    return h.fastmcpClient.DisconnectMCP(disconnectCtx, mcpID)
})
```

### 3. Wrapper Methods

Public wrapper methods are provided for use throughout the codebase:

```go
// ConnectMCPWithBreaker - Protected connect operation
func (h *MCPHandler) ConnectMCPWithBreaker(ctx context.Context, config mcp.MCPConfig) error

// DisconnectMCPWithBreaker - Protected disconnect operation
func (h *MCPHandler) DisconnectMCPWithBreaker(ctx context.Context, mcpID string) error

// ListToolsWithBreaker - Protected list tools operation
func (h *MCPHandler) ListToolsWithBreaker(ctx context.Context, mcpID string) ([]mcp.Tool, error)

// CallToolWithBreaker - Protected call tool operation
func (h *MCPHandler) CallToolWithBreaker(ctx context.Context, mcpID, toolName string, args map[string]any) (any, error)
```

## Error Handling

### Circuit Open (503 Service Unavailable)

When the circuit breaker is open, clients receive:

```json
{
  "error": "Service temporarily unavailable",
  "code": "circuit_breaker_open",
  "message": "The test_connection operation is currently unavailable due to repeated failures. Please try again in 30 seconds.",
  "details": {
    "operation": "test_connection",
    "circuit_state": "open",
    "retry_after_seconds": 30
  }
}
```

**HTTP Headers:**
- `Retry-After: 30`
- `Status: 503 Service Unavailable`

### Too Many Requests (429 Too Many Requests)

When in half-open state with max concurrent requests exceeded:

```json
{
  "error": "Too many requests",
  "code": "circuit_breaker_half_open",
  "message": "The test_connection operation is recovering and cannot accept more requests at this time. Please try again shortly.",
  "details": {
    "operation": "test_connection",
    "circuit_state": "half-open",
    "retry_after_seconds": 5
  }
}
```

**HTTP Headers:**
- `Retry-After: 5`
- `Status: 429 Too Many Requests`

## Graceful Degradation

### Fallback Responses

The implementation provides a helper method for degraded service responses:

```go
func (h *MCPHandler) getDegradedServiceResponse(operation string) map[string]any {
    return map[string]any{
        "status":  "degraded",
        "message": fmt.Sprintf("The %s operation is currently degraded. Using cached or fallback data.", operation),
        "operation": operation,
        "timestamp": time.Now().UTC(),
    }
}
```

### Partial Success

During test connections, if listing tools fails but the connection succeeds, the operation continues and returns partial results:

```go
if listErr := h.breakers.listTools.Execute(testCtx, func() error {
    // List tools...
}); listErr != nil {
    // Log error but continue with other operations
    log.Printf("Failed to list tools during test: %v", listErr)
}
```

## Monitoring and Metrics

### State Change Logging

Circuit breaker state changes are automatically logged:

```go
func onCircuitBreakerStateChange(name string, from resilience.State, to resilience.State) {
    log.Printf("[MCP Circuit Breaker] %s: State changed from %s to %s",
        name, from.String(), to.String())
}
```

### Metrics Collection

Metrics are collected for each circuit breaker operation:

```go
func (h *MCPHandler) logCircuitBreakerMetrics(ctx context.Context, operation string, breaker *resilience.CircuitBreaker) {
    stats := breaker.Stats()

    log.Printf("[Circuit Breaker Metrics] Operation: %s, State: %s, Total Requests: %d, Successes: %d, Failures: %d, Consecutive Failures: %d",
        operation,
        stats.State.String(),
        stats.TotalRequests,
        stats.TotalSuccesses,
        stats.TotalFailures,
        stats.ConsecutiveFailures,
    )
}
```

### Stats API

Get statistics for all circuit breakers:

```go
stats := h.GetCircuitBreakerStats()
// Returns: map[string]resilience.CBStats
// Keys: "connect", "call_tool", "list_tools", "disconnect", "test_connection"
```

### State API

Get current state of all circuit breakers:

```go
states := h.GetCircuitBreakerState()
// Returns: map[string]string
// Values: "closed", "open", or "half-open"
```

## Health Checks

The handler provides a health check endpoint that includes circuit breaker status:

```go
health := h.HealthCheck()
// Returns:
{
    "status": "healthy" | "degraded",
    "circuit_breakers": {
        "states": {
            "connect": "closed",
            "call_tool": "closed",
            "list_tools": "closed",
            "disconnect": "closed",
            "test_connection": "closed"
        },
        "stats": {
            "connect": { /* CBStats */ },
            "call_tool": { /* CBStats */ },
            // ... etc
        }
    }
}
```

If any circuit breaker is open, the overall status is marked as "degraded".

## Administrative Operations

### Reset Circuit Breakers

Reset all circuit breakers to closed state:

```go
h.ResetCircuitBreakers()
```

This is useful for:
- Manual recovery after fixing upstream issues
- Testing and development
- Emergency operations

**WARNING**: Only reset circuit breakers if you're certain the underlying issue has been resolved.

## Prometheus Integration (TODO)

The implementation includes placeholders for Prometheus metrics:

```go
// TODO: Export to metrics system (Prometheus)
// metrics.RecordCircuitBreakerStateChange(name, to.String())
```

### Recommended Metrics

1. **Circuit Breaker State Gauge**
   ```
   mcp_circuit_breaker_state{operation="connect"} 0  # closed
   mcp_circuit_breaker_state{operation="connect"} 1  # open
   mcp_circuit_breaker_state{operation="connect"} 2  # half-open
   ```

2. **Request Counters**
   ```
   mcp_circuit_breaker_requests_total{operation="connect",result="success"}
   mcp_circuit_breaker_requests_total{operation="connect",result="failure"}
   mcp_circuit_breaker_requests_total{operation="connect",result="rejected"}
   ```

3. **State Transition Counter**
   ```
   mcp_circuit_breaker_state_changes_total{operation="connect",from="closed",to="open"}
   ```

4. **Consecutive Failure Gauge**
   ```
   mcp_circuit_breaker_consecutive_failures{operation="connect"}
   ```

## Best Practices

### 1. Context Timeout

Always use a context with timeout when calling circuit breaker-protected operations:

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

err := h.ConnectMCPWithBreaker(ctx, config)
```

### 2. Error Handling

Check for circuit breaker-specific errors:

```go
if err == resilience.ErrCircuitOpen || err == resilience.ErrTooManyRequests {
    h.handleCircuitBreakerError(w, err, operation)
    return
}
```

### 3. Monitoring

Regularly monitor circuit breaker states and metrics:

```go
// Log metrics after important operations
h.logCircuitBreakerMetrics(ctx, "test_connection", h.breakers.testConnection)
```

### 4. Graceful Degradation

Design endpoints to continue working with partial data when circuit breakers trip:

```go
// Continue processing even if optional operations fail
if err := h.ListToolsWithBreaker(ctx, mcpID); err != nil {
    log.Printf("Failed to list tools: %v", err)
    // Continue with empty tools list
}
```

## Testing

### Simulating Failures

To test circuit breaker behavior:

1. Make 5 consecutive failed requests to trigger opening
2. Wait 30 seconds for transition to half-open
3. Make successful requests to close the circuit

### Testing Half-Open State

1. Open the circuit breaker
2. Wait for timeout (30 seconds)
3. Make more than 100 concurrent requests
4. Verify that excess requests receive 429 errors

### Testing Recovery

1. Open the circuit breaker
2. Wait for timeout
3. Make 2 successful requests
4. Verify circuit closes

## Migration Guide

### For Existing Code

If you have existing code that directly calls `fastmcpClient`, update it to use the wrapper methods:

**Before:**
```go
err := h.fastmcpClient.ConnectMCP(ctx, config)
```

**After:**
```go
err := h.ConnectMCPWithBreaker(ctx, config)
```

### For New Code

Always use the circuit breaker-protected methods for MCP operations:

- `ConnectMCPWithBreaker()` instead of `fastmcpClient.ConnectMCP()`
- `DisconnectMCPWithBreaker()` instead of `fastmcpClient.DisconnectMCP()`
- `ListToolsWithBreaker()` instead of `fastmcpClient.ListTools()`
- `CallToolWithBreaker()` instead of `fastmcpClient.CallTool()`

## Configuration Tuning

The default configuration works well for most scenarios, but you may want to adjust:

### More Aggressive Protection

```go
config := resilience.CBConfig{
    FailureThreshold:      3,  // Open after 3 failures
    SuccessThreshold:      3,  // Require 3 successes to close
    Timeout:               60 * time.Second,  // Stay open longer
    MaxConcurrentRequests: 10,  // Allow fewer test requests
}
```

### More Lenient Protection

```go
config := resilience.CBConfig{
    FailureThreshold:      10,  // Allow more failures
    SuccessThreshold:      1,   // Faster recovery
    Timeout:               15 * time.Second,  // Shorter timeout
    MaxConcurrentRequests: 200,  // Allow more test requests
}
```

## Troubleshooting

### Circuit Breaker Stuck Open

**Symptoms:** Circuit breaker remains open even though backend is healthy

**Solutions:**
1. Check if underlying service has actually recovered
2. Verify timeout configuration is appropriate
3. Manually reset: `h.ResetCircuitBreakers()`
4. Check logs for state transitions

### Too Many 429 Errors

**Symptoms:** Clients receive many 429 errors during recovery

**Solutions:**
1. Increase `MaxConcurrentRequests` in configuration
2. Implement client-side retry with backoff
3. Add request queuing on client side

### Circuit Opens Too Quickly

**Symptoms:** Circuit breaker opens with occasional failures

**Solutions:**
1. Increase `FailureThreshold`
2. Implement retry logic before circuit breaker
3. Add filtering for expected/transient errors

## Future Enhancements

1. **Per-MCP Circuit Breakers**: Individual breakers for each MCP server
2. **Adaptive Thresholds**: Automatically adjust thresholds based on error rates
3. **Bulkhead Pattern**: Limit concurrent requests per MCP server
4. **Fallback Cache**: Return cached results when circuit is open
5. **Prometheus Metrics**: Full integration with metrics system
6. **Admin Dashboard**: UI for monitoring and managing circuit breakers
7. **Dynamic Configuration**: Hot-reload circuit breaker settings
8. **Error Classification**: Different handling for different error types

## References

- [Circuit Breaker Implementation](../resilience/circuit_breaker.go)
- [Resilience Patterns](../resilience/patterns.go)
- [Circuit Breaker Metrics](../resilience/metrics.go)
- [MCP Client](../mcp/fastmcp_client.go)

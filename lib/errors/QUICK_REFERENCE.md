# MCP Errors Quick Reference

## Error Constructors

### Connection Errors

```go
// Generic connection failure
NewConnectionError(endpoint string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// Connection timeout
NewConnectionTimeoutError(endpoint string, timeout time.Duration, err error) *MCPError
// Returns: 504 Gateway Timeout, Retryable: true, Temporary: true

// Connection refused
NewConnectionRefusedError(endpoint string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true
```

### Tool Execution Errors

```go
// Generic tool execution failure
NewToolExecutionError(toolName string, err error) *MCPError
// Returns: 500 Internal Server Error, Retryable: false, Temporary: false

// Tool not found
NewToolNotFoundError(toolName string) *MCPError
// Returns: 404 Not Found, Retryable: false, Temporary: false

// Invalid arguments
NewInvalidArgumentsError(toolName string, reason string, err error) *MCPError
// Returns: 400 Bad Request, Retryable: false, Temporary: false
```

### Timeout Errors

```go
// Generic timeout
NewTimeoutError(operation string, timeout time.Duration, err error) *MCPError
// Returns: 504 Gateway Timeout, Retryable: true, Temporary: true

// Operation-specific timeout
NewOperationTimeoutError(operation string, resource string, timeout time.Duration, err error) *MCPError
// Returns: 504 Gateway Timeout, Retryable: true, Temporary: true

// Context deadline exceeded
NewDeadlineExceededError(operation string, deadline time.Time, err error) *MCPError
// Returns: 504 Gateway Timeout, Retryable: true, Temporary: true
```

### Authentication Errors

```go
// Generic authentication failure
NewAuthenticationError(reason string, err error) *MCPError
// Returns: 401 Unauthorized, Retryable: false, Temporary: false

// Expired credentials
NewAuthExpiredError(expiredAt time.Time, err error) *MCPError
// Returns: 401 Unauthorized, Retryable: true, Temporary: false

// Invalid credentials
NewAuthInvalidError(reason string, err error) *MCPError
// Returns: 401 Unauthorized, Retryable: false, Temporary: false

// OAuth failure
NewOAuthFailureError(provider string, reason string, err error) *MCPError
// Returns: 401 Unauthorized, Retryable: false, Temporary: false
```

### Rate Limiting Errors

```go
// Rate limit exceeded
NewRateLimitError(resource string, limit int, retryAfter time.Duration, err error) *MCPError
// Returns: 429 Too Many Requests, Retryable: true, Temporary: true

// Quota exceeded
NewQuotaExceededError(resource string, quota int, used int, err error) *MCPError
// Returns: 429 Too Many Requests, Retryable: false, Temporary: false

// Request throttled
NewThrottledError(resource string, retryAfter time.Duration, err error) *MCPError
// Returns: 429 Too Many Requests, Retryable: true, Temporary: true
```

### Server Errors

```go
// Generic MCP server error
NewServerError(message string, err error) *MCPError
// Returns: 502 Bad Gateway, Retryable: true, Temporary: true

// Server unavailable
NewServerUnavailableError(endpoint string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// Server internal error
NewServerInternalError(message string, err error) *MCPError
// Returns: 502 Bad Gateway, Retryable: true, Temporary: true
```

### Circuit Breaker Errors

```go
// Circuit breaker open
NewCircuitOpenError(operation string, openedAt time.Time, nextRetry time.Time, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// Circuit breaker half-open
NewCircuitHalfOpenError(operation string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// Too many concurrent requests
NewTooManyRequestsError(operation string, err error) *MCPError
// Returns: 429 Too Many Requests, Retryable: true, Temporary: true
```

### Configuration Errors

```go
// Invalid configuration field
NewInvalidConfigError(field string, reason string, err error) *MCPError
// Returns: 400 Bad Request, Retryable: false, Temporary: false

// Missing required configuration
NewMissingConfigError(field string, err error) *MCPError
// Returns: 400 Bad Request, Retryable: false, Temporary: false

// Configuration validation failed
NewConfigValidationError(validationErrors []string, err error) *MCPError
// Returns: 400 Bad Request, Retryable: false, Temporary: false
```

### Resource Errors

```go
// Resource not found
NewResourceNotFoundError(resourceType string, resourceID string, err error) *MCPError
// Returns: 404 Not Found, Retryable: false, Temporary: false

// Resource locked
NewResourceLockedError(resourceType string, resourceID string, err error) *MCPError
// Returns: 409 Conflict, Retryable: true, Temporary: true

// Resources exhausted
NewResourceExhaustedError(resourceType string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true
```

### Network Errors

```go
// Generic network error
NewNetworkError(operation string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// DNS resolution failure
NewDNSError(hostname string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: true, Temporary: true

// TLS/SSL error
NewTLSError(reason string, err error) *MCPError
// Returns: 503 Service Unavailable, Retryable: false, Temporary: false
```

## Error Utilities

```go
// Check if error is temporary
IsTemporary(err error) bool

// Check if error should be retried
IsRetryable(err error) bool

// Get HTTP status code
GetStatusCode(err error) int

// Get error code
GetErrorCode(err error) ErrorCode

// Get detailed error message
GetDetailedMessage(err error) string

// Check if error is MCPError
IsMCPError(err error) bool

// Convert to MCPError
AsMCPError(err error) (*MCPError, bool)
```

## Error Wrapping

```go
// Wrap FastMCP errors with automatic classification
WrapFastMCPError(operation string, resource string, err error) error

// Example
result, err := fastmcpClient.CallTool(ctx, clientID, toolName, args)
if err != nil {
    return nil, WrapFastMCPError("call_tool", toolName, err)
}
```

## HTTP Integration

```go
// Write error response to HTTP
WriteErrorResponse(w http.ResponseWriter, err error, requestID string)

// Create error response struct
NewErrorResponse(err error, requestID string) ErrorResponse

// Example
func handler(w http.ResponseWriter, r *http.Request) {
    result, err := doWork()
    if err != nil {
        WriteErrorResponse(w, err, requestID)
        return
    }
    // ... success response
}
```

## Adding Context

```go
err := NewConnectionError("localhost:8000", originalErr).
    WithOperation("connect").
    WithResource("mcp-server").
    WithMetadata("attempt", 3).
    WithMetadata("timeout", "5s")
```

## Structured Logging

```go
err := NewToolExecutionError("search", originalErr)
fields := err.LogFields()

// Use with your logger
log.Error().Fields(fields).Msg("Operation failed")
```

## Error Codes

| Error Code | HTTP | Retry | Temp |
|-----------|------|-------|------|
| `MCP_CONNECTION_ERROR` | 503 | Yes | Yes |
| `MCP_CONNECTION_TIMEOUT` | 504 | Yes | Yes |
| `MCP_CONNECTION_REFUSED` | 503 | Yes | Yes |
| `MCP_TOOL_EXECUTION_ERROR` | 500 | No | No |
| `MCP_TOOL_NOT_FOUND` | 404 | No | No |
| `MCP_INVALID_ARGUMENTS` | 400 | No | No |
| `MCP_TIMEOUT` | 504 | Yes | Yes |
| `MCP_OPERATION_TIMEOUT` | 504 | Yes | Yes |
| `MCP_DEADLINE_EXCEEDED` | 504 | Yes | Yes |
| `MCP_AUTHENTICATION_ERROR` | 401 | No | No |
| `MCP_AUTH_EXPIRED` | 401 | Yes | No |
| `MCP_AUTH_INVALID` | 401 | No | No |
| `MCP_OAUTH_FAILURE` | 401 | No | No |
| `MCP_RATE_LIMIT_EXCEEDED` | 429 | Yes | Yes |
| `MCP_QUOTA_EXCEEDED` | 429 | No | No |
| `MCP_THROTTLED` | 429 | Yes | Yes |
| `MCP_SERVER_ERROR` | 502 | Yes | Yes |
| `MCP_SERVER_UNAVAILABLE` | 503 | Yes | Yes |
| `MCP_SERVER_INTERNAL_ERROR` | 502 | Yes | Yes |
| `MCP_CIRCUIT_BREAKER_OPEN` | 503 | Yes | Yes |
| `MCP_CIRCUIT_BREAKER_HALF_OPEN` | 503 | Yes | Yes |
| `MCP_TOO_MANY_REQUESTS` | 429 | Yes | Yes |
| `MCP_INVALID_CONFIG` | 400 | No | No |
| `MCP_MISSING_CONFIG` | 400 | No | No |
| `MCP_CONFIG_VALIDATION_ERROR` | 400 | No | No |
| `MCP_RESOURCE_NOT_FOUND` | 404 | No | No |
| `MCP_RESOURCE_LOCKED` | 409 | Yes | Yes |
| `MCP_RESOURCE_EXHAUSTED` | 503 | Yes | Yes |
| `MCP_NETWORK_ERROR` | 503 | Yes | Yes |
| `MCP_DNS_ERROR` | 503 | Yes | Yes |
| `MCP_TLS_ERROR` | 503 | No | No |

## Common Patterns

### Retry with Backoff

```go
for attempt := 1; attempt <= maxRetries; attempt++ {
    err := doWork()
    if err == nil {
        return nil
    }

    if !errors.IsRetryable(err) {
        return err // Don't retry permanent failures
    }

    backoff := time.Second * time.Duration(1 << uint(attempt-1))
    time.Sleep(backoff)
}
```

### HTTP Handler

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")

    result, err := h.processRequest(r)
    if err != nil {
        errors.WriteErrorResponse(w, err, requestID)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Circuit Breaker Integration

```go
err := circuitBreaker.Execute(ctx, func() error {
    return client.Connect(ctx)
})

if err != nil {
    if errors.Is(err, resilience.ErrCircuitOpen) {
        stats := circuitBreaker.Stats()
        return errors.NewCircuitOpenError(
            "connect",
            stats.StateChangedAt,
            stats.StateChangedAt.Add(30*time.Second),
            err,
        )
    }
    return err
}
```

### Configuration Validation

```go
func ValidateConfig(cfg *Config) error {
    var validationErrors []string

    if cfg.Endpoint == "" {
        validationErrors = append(validationErrors, "endpoint is required")
    }

    if cfg.Timeout <= 0 {
        validationErrors = append(validationErrors, "timeout must be > 0")
    }

    if len(validationErrors) > 0 {
        return errors.NewConfigValidationError(validationErrors, nil)
    }

    return nil
}
```

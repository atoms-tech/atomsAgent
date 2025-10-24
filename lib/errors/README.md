# MCP Errors Package

Comprehensive error handling for MCP (Model Context Protocol) operations in the AgentAPI system.

## Overview

The `errors` package provides:

- **Structured Error Types**: Well-defined error types for all MCP operations
- **Rich Context**: Include operation, resource, and metadata in errors
- **Error Classification**: Determine if errors are temporary, retryable, or permanent
- **HTTP Integration**: Automatic HTTP status code mapping
- **Error Wrapping**: Compatible with Go 1.13+ error chains
- **JSON Serialization**: API-ready error responses
- **Logging Support**: Structured logging fields

## Installation

```go
import "github.com/coder/agentapi/lib/errors"
```

## Quick Start

### Creating Errors

```go
// Connection error
err := errors.NewConnectionError("localhost:8000", originalErr)

// Tool execution error
err := errors.NewToolExecutionError("my_tool", originalErr).
    WithOperation("call_tool").
    WithMetadata("attempt", 3)

// Timeout error
err := errors.NewOperationTimeoutError("connect", "localhost:8000", 5*time.Second, originalErr)

// Authentication error
err := errors.NewAuthenticationError("invalid token", originalErr)

// Rate limit error
err := errors.NewRateLimitError("api_calls", 100, 60*time.Second, originalErr)
```

### Checking Error Properties

```go
if errors.IsRetryable(err) {
    // Retry the operation
    time.Sleep(backoff)
    retry()
}

if errors.IsTemporary(err) {
    // Temporary issue, may resolve itself
    log.Warn("Temporary error encountered", "error", err)
}

statusCode := errors.GetStatusCode(err)
errorCode := errors.GetErrorCode(err)
detailedMsg := errors.GetDetailedMessage(err)
```

### HTTP Error Responses

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    result, err := callMCPTool("my_tool", args)
    if err != nil {
        // Automatically sets correct HTTP status and returns JSON
        errors.WriteErrorResponse(w, err, requestID)
        return
    }

    // Success case
    json.NewEncoder(w).Encode(result)
}
```

### Wrapping FastMCP Errors

```go
// Automatically classify and wrap FastMCP errors
result, err := fastmcpClient.CallTool(ctx, clientID, toolName, args)
if err != nil {
    // Wraps with appropriate error type based on error content
    return nil, errors.WrapFastMCPError("call_tool", toolName, err)
}
```

## Error Types

### Connection Errors

```go
// Generic connection failure
NewConnectionError(endpoint string, err error) *MCPError

// Connection timeout
NewConnectionTimeoutError(endpoint string, timeout time.Duration, err error) *MCPError

// Connection refused by server
NewConnectionRefusedError(endpoint string, err error) *MCPError
```

**Properties:**
- HTTP Status: 503 (Service Unavailable) or 504 (Gateway Timeout)
- Retryable: Yes
- Temporary: Yes

**Example:**
```go
err := errors.NewConnectionError("http://mcp-server:8000",
    errors.New("dial tcp: connection refused"))

// Returns 503, suggests retry
fmt.Println(errors.GetStatusCode(err))      // 503
fmt.Println(errors.IsRetryable(err))        // true
fmt.Println(errors.IsTemporary(err))        // true
```

### Tool Execution Errors

```go
// Generic tool execution failure
NewToolExecutionError(toolName string, err error) *MCPError

// Tool not found
NewToolNotFoundError(toolName string) *MCPError

// Invalid arguments provided
NewInvalidArgumentsError(toolName string, reason string, err error) *MCPError
```

**Properties:**
- HTTP Status: 500 (Internal Server Error), 404 (Not Found), or 400 (Bad Request)
- Retryable: No (usually)
- Temporary: No

**Example:**
```go
err := errors.NewToolExecutionError("search_database",
    errors.New("SQL syntax error")).
    WithMetadata("query", "SELECT * FORM users"). // typo in query
    WithMetadata("line", 1)

if !errors.IsRetryable(err) {
    // Don't retry - permanent failure
    log.Error("Tool execution failed permanently", "error", err)
}
```

### Timeout Errors

```go
// Generic timeout
NewTimeoutError(operation string, timeout time.Duration, err error) *MCPError

// Operation-specific timeout
NewOperationTimeoutError(operation string, resource string, timeout time.Duration, err error) *MCPError

// Context deadline exceeded
NewDeadlineExceededError(operation string, deadline time.Time, err error) *MCPError
```

**Properties:**
- HTTP Status: 504 (Gateway Timeout)
- Retryable: Yes
- Temporary: Yes

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := client.CallTool(ctx, "slow_tool", args)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        err = errors.NewDeadlineExceededError("call_tool",
            time.Now().Add(10*time.Second), err)
    }

    // Can retry with longer timeout
    if errors.IsRetryable(err) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        result, err = client.CallTool(ctx, "slow_tool", args)
    }
}
```

### Authentication Errors

```go
// Generic authentication failure
NewAuthenticationError(reason string, err error) *MCPError

// Expired credentials
NewAuthExpiredError(expiredAt time.Time, err error) *MCPError

// Invalid credentials
NewAuthInvalidError(reason string, err error) *MCPError

// OAuth-specific failure
NewOAuthFailureError(provider string, reason string, err error) *MCPError
```

**Properties:**
- HTTP Status: 401 (Unauthorized)
- Retryable: Sometimes (expired credentials can be refreshed)
- Temporary: No

**Example:**
```go
// Expired token case
token, err := validateToken(authHeader)
if err != nil {
    if isExpired(token) {
        return errors.NewAuthExpiredError(token.ExpiresAt, err)
    }
    return errors.NewAuthInvalidError("malformed token", err)
}

// OAuth failure case
accessToken, err := oauthClient.Exchange(ctx, code)
if err != nil {
    return errors.NewOAuthFailureError("google",
        "token exchange failed", err).
        WithMetadata("oauth_error", err.Error())
}
```

### Rate Limiting Errors

```go
// Rate limit exceeded
NewRateLimitError(resource string, limit int, retryAfter time.Duration, err error) *MCPError

// Quota exceeded
NewQuotaExceededError(resource string, quota int, used int, err error) *MCPError

// Request throttled
NewThrottledError(resource string, retryAfter time.Duration, err error) *MCPError
```

**Properties:**
- HTTP Status: 429 (Too Many Requests)
- Retryable: Yes (with backoff)
- Temporary: Yes (for rate limits) or No (for quota)

**Example:**
```go
err := errors.NewRateLimitError("api_calls", 100, 60*time.Second, nil).
    WithMetadata("current_rate", 120).
    WithMetadata("window", "1m")

// Extract retry-after from metadata
if mcpErr, ok := errors.AsMCPError(err); ok {
    if retryAfter, ok := mcpErr.Metadata["retry_after"].(string); ok {
        duration, _ := time.ParseDuration(retryAfter)
        time.Sleep(duration)
        retry()
    }
}
```

### Circuit Breaker Errors

```go
// Circuit breaker is open
NewCircuitOpenError(operation string, openedAt time.Time, nextRetry time.Time, err error) *MCPError

// Circuit breaker is half-open
NewCircuitHalfOpenError(operation string, err error) *MCPError

// Too many concurrent requests
NewTooManyRequestsError(operation string, err error) *MCPError
```

**Properties:**
- HTTP Status: 503 (Service Unavailable) or 429 (Too Many Requests)
- Retryable: Yes
- Temporary: Yes

**Example:**
```go
// Circuit breaker integration
err := circuitBreaker.Execute(ctx, func() error {
    return client.Connect(ctx)
})

if err != nil {
    if errors.Is(err, resilience.ErrCircuitOpen) {
        stats := circuitBreaker.Stats()
        return errors.NewCircuitOpenError("connect",
            stats.StateChangedAt,
            stats.StateChangedAt.Add(30*time.Second),
            err)
    }
}
```

### Configuration Errors

```go
// Invalid configuration field
NewInvalidConfigError(field string, reason string, err error) *MCPError

// Missing required configuration
NewMissingConfigError(field string, err error) *MCPError

// Configuration validation failed
NewConfigValidationError(validationErrors []string, err error) *MCPError
```

**Properties:**
- HTTP Status: 400 (Bad Request)
- Retryable: No
- Temporary: No

**Example:**
```go
func ValidateConfig(cfg *MCPConfig) error {
    var validationErrors []string

    if cfg.Endpoint == "" {
        validationErrors = append(validationErrors, "endpoint is required")
    }

    if cfg.Type != "http" && cfg.Type != "sse" && cfg.Type != "stdio" {
        validationErrors = append(validationErrors,
            "type must be http, sse, or stdio")
    }

    if len(validationErrors) > 0 {
        return errors.NewConfigValidationError(validationErrors, nil)
    }

    return nil
}
```

### Server Errors

```go
// Generic MCP server error
NewServerError(message string, err error) *MCPError

// Server unavailable
NewServerUnavailableError(endpoint string, err error) *MCPError

// Server internal error
NewServerInternalError(message string, err error) *MCPError
```

**Properties:**
- HTTP Status: 502 (Bad Gateway) or 503 (Service Unavailable)
- Retryable: Yes
- Temporary: Yes

### Resource Errors

```go
// Resource not found
NewResourceNotFoundError(resourceType string, resourceID string, err error) *MCPError

// Resource locked
NewResourceLockedError(resourceType string, resourceID string, err error) *MCPError

// Resources exhausted
NewResourceExhaustedError(resourceType string, err error) *MCPError
```

**Properties:**
- HTTP Status: 404 (Not Found), 409 (Conflict), or 503 (Service Unavailable)
- Retryable: Depends on type
- Temporary: Depends on type

### Network Errors

```go
// Generic network error
NewNetworkError(operation string, err error) *MCPError

// DNS resolution failure
NewDNSError(hostname string, err error) *MCPError

// TLS/SSL error
NewTLSError(reason string, err error) *MCPError
```

**Properties:**
- HTTP Status: 503 (Service Unavailable)
- Retryable: Usually yes
- Temporary: Usually yes

## Error Context

Add rich context to errors:

```go
err := errors.NewToolExecutionError("search", originalErr).
    WithOperation("call_tool").
    WithResource("vector_db").
    WithMetadata("query", searchQuery).
    WithMetadata("limit", 100).
    WithMetadata("attempt", 3).
    WithMetadata("duration", time.Since(start))
```

## Error Utilities

### Checking Error Properties

```go
// Check if error is temporary
if errors.IsTemporary(err) {
    log.Info("Temporary error, may resolve", "error", err)
}

// Check if error should be retried
if errors.IsRetryable(err) {
    backoff := time.Second * time.Duration(math.Pow(2, float64(attempt)))
    time.Sleep(backoff)
    return retry()
}

// Get HTTP status code
statusCode := errors.GetStatusCode(err)

// Get error code
errorCode := errors.GetErrorCode(err)

// Get detailed message
detailedMsg := errors.GetDetailedMessage(err)
```

### Type Assertions

```go
// Check if error is MCPError
if errors.IsMCPError(err) {
    log.Info("MCP-specific error occurred")
}

// Convert to MCPError
if mcpErr, ok := errors.AsMCPError(err); ok {
    fmt.Printf("Error code: %s\n", mcpErr.Code)
    fmt.Printf("Operation: %s\n", mcpErr.Operation)
    fmt.Printf("Resource: %s\n", mcpErr.Resource)
    fmt.Printf("Metadata: %+v\n", mcpErr.Metadata)
}
```

### Error Chaining

```go
// Errors support Go 1.13+ error unwrapping
originalErr := errors.New("network timeout")
wrappedErr := errors.NewConnectionTimeoutError("localhost", 5*time.Second, originalErr)

// Check using errors.Is
if errors.Is(wrappedErr, originalErr) {
    fmt.Println("Wrapped error contains original")
}

// Unwrap the chain
cause := errors.Unwrap(wrappedErr)
```

## HTTP Integration

### Writing Error Responses

```go
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")

    result, err := h.processRequest(r)
    if err != nil {
        // Automatically:
        // - Sets correct HTTP status code
        // - Returns JSON error response
        // - Includes request ID and timestamp
        errors.WriteErrorResponse(w, err, requestID)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Custom Error Responses

```go
err := errors.NewAuthenticationError("invalid API key", nil)

// Get error response struct
response := err.ToErrorResponse("req-123")

// Customize before sending
response.Error.Metadata["support_url"] = "https://support.example.com"

// Write to response
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(errors.GetStatusCode(err))
json.NewEncoder(w).Encode(response)
```

### Example JSON Response

```json
{
  "error": {
    "code": "MCP_AUTHENTICATION_ERROR",
    "message": "Authentication failed: invalid API key",
    "operation": "connect",
    "resource": "mcp-server",
    "metadata": {
      "reason": "invalid API key",
      "support_url": "https://support.example.com"
    },
    "temporary": false,
    "retryable": false
  },
  "request_id": "req-123",
  "timestamp": "2025-10-23T12:34:56Z"
}
```

## Logging Integration

### Structured Logging

```go
err := errors.NewToolExecutionError("search", originalErr).
    WithOperation("call_tool").
    WithMetadata("attempt", 3)

// Get structured log fields
fields := err.LogFields()

// Use with your logger (e.g., zerolog, zap, logrus)
log.Error().
    Fields(fields).
    Msg("Tool execution failed")

// Output includes:
// - error_code: MCP_TOOL_EXECUTION_ERROR
// - message: Failed to execute tool 'search'
// - operation: call_tool
// - resource: search
// - timestamp: 2025-10-23T12:34:56Z
// - temporary: false
// - retryable: false
// - http_status: 500
// - cause: original error message
// - meta_attempt: 3
```

## Wrapping FastMCP Errors

The `WrapFastMCPError` function automatically classifies errors:

```go
// Automatically detects error type from content
result, err := fastmcpClient.CallTool(ctx, clientID, toolName, args)
if err != nil {
    // Returns appropriate MCPError based on error content:
    // - "timeout" -> TimeoutError
    // - "connection refused" -> ConnectionRefusedError
    // - "unauthorized" -> AuthenticationError
    // - "rate limit" -> RateLimitError
    // - "tool not found" -> ToolNotFoundError
    // - etc.
    wrapped := errors.WrapFastMCPError("call_tool", toolName, err)

    // Now you can use error utilities
    if errors.IsRetryable(wrapped) {
        // Retry logic
    }

    return nil, wrapped
}
```

## Best Practices

### 1. Always Add Context

```go
// Bad - minimal context
return errors.NewServerError("failed", err)

// Good - rich context
return errors.NewServerError("failed to process request", err).
    WithOperation("process_request").
    WithResource(requestID).
    WithMetadata("user_id", userID).
    WithMetadata("duration", time.Since(start))
```

### 2. Use Appropriate Error Types

```go
// Bad - generic error for specific case
return errors.NewServerError("authentication failed", err)

// Good - specific error type
return errors.NewAuthenticationError("invalid credentials", err)
```

### 3. Check Error Properties Before Retry

```go
// Bad - retry everything
for i := 0; i < 3; i++ {
    err := doSomething()
    if err == nil {
        return nil
    }
}

// Good - only retry retryable errors
for attempt := 0; attempt < 3; attempt++ {
    err := doSomething()
    if err == nil {
        return nil
    }

    if !errors.IsRetryable(err) {
        return err // Don't retry permanent failures
    }

    if !errors.IsTemporary(err) {
        log.Warn("Non-temporary error, may not resolve", "error", err)
    }

    backoff := time.Second * time.Duration(math.Pow(2, float64(attempt)))
    time.Sleep(backoff)
}
```

### 4. Use WrapFastMCPError Consistently

```go
// Wrap all FastMCP errors for consistent classification
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
    result, err := c.fastmcpClient.CallTool(ctx, c.ID, toolName, args)
    if err != nil {
        return nil, errors.WrapFastMCPError("call_tool", toolName, err)
    }
    return result, nil
}
```

### 5. Log with Structured Fields

```go
err := errors.NewConnectionError("localhost:8000", originalErr)

// Use LogFields for consistent structured logging
log.Error().
    Fields(err.LogFields()).
    Msg("Connection failed")
```

## Complete Example

Here's a complete example showing error handling in an MCP client:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/coder/agentapi/lib/errors"
    "github.com/coder/agentapi/lib/mcp"
    "github.com/coder/agentapi/lib/resilience"
)

type MCPService struct {
    client         *mcp.Client
    circuitBreaker *resilience.CircuitBreaker
}

func (s *MCPService) CallTool(ctx context.Context, toolName string, args map[string]any) (any, error) {
    // Execute with circuit breaker protection
    var result any
    var err error

    cbErr := s.circuitBreaker.Execute(ctx, func() error {
        result, err = s.client.CallTool(ctx, toolName, args)
        if err != nil {
            // Wrap FastMCP errors
            err = errors.WrapFastMCPError("call_tool", toolName, err)
        }
        return err
    })

    if cbErr != nil {
        // Circuit breaker error
        if errors.Is(cbErr, resilience.ErrCircuitOpen) {
            stats := s.circuitBreaker.Stats()
            return nil, errors.NewCircuitOpenError(
                "call_tool",
                stats.StateChangedAt,
                stats.StateChangedAt.Add(30*time.Second),
                cbErr,
            ).WithResource(toolName)
        }
        return nil, cbErr
    }

    return result, nil
}

func (s *MCPService) CallToolWithRetry(ctx context.Context, toolName string, args map[string]any) (any, error) {
    maxAttempts := 3

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        result, err := s.CallTool(ctx, toolName, args)
        if err == nil {
            return result, nil
        }

        // Don't retry non-retryable errors
        if !errors.IsRetryable(err) {
            log.Printf("Non-retryable error: %v", errors.GetDetailedMessage(err))
            return nil, err
        }

        // Log temporary errors
        if errors.IsTemporary(err) {
            log.Printf("Temporary error on attempt %d/%d: %v",
                attempt, maxAttempts, err)
        }

        // Don't sleep after last attempt
        if attempt < maxAttempts {
            backoff := time.Second * time.Duration(1<<uint(attempt-1))
            log.Printf("Retrying in %v...", backoff)
            time.Sleep(backoff)
        }
    }

    return nil, errors.NewServerError("max retries exceeded", nil).
        WithOperation("call_tool").
        WithResource(toolName).
        WithMetadata("attempts", maxAttempts)
}

func handleAPIRequest(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")

    // Parse request...
    toolName := r.URL.Query().Get("tool")
    if toolName == "" {
        err := errors.NewInvalidConfigError("tool", "tool parameter required", nil)
        errors.WriteErrorResponse(w, err, requestID)
        return
    }

    // Call tool...
    svc := &MCPService{} // initialized elsewhere
    result, err := svc.CallToolWithRetry(r.Context(), toolName, map[string]any{})
    if err != nil {
        // Log error with structured fields
        if mcpErr, ok := errors.AsMCPError(err); ok {
            log.Printf("Tool call failed: %+v", mcpErr.LogFields())
        }

        // Write error response (automatic status code and JSON)
        errors.WriteErrorResponse(w, err, requestID)
        return
    }

    // Success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]any{
        "result": result,
        "request_id": requestID,
    })
}
```

## Error Code Reference

| Error Code | HTTP Status | Retryable | Temporary | Description |
|------------|-------------|-----------|-----------|-------------|
| `MCP_CONNECTION_ERROR` | 503 | Yes | Yes | Failed to connect to MCP server |
| `MCP_CONNECTION_TIMEOUT` | 504 | Yes | Yes | Connection attempt timed out |
| `MCP_CONNECTION_REFUSED` | 503 | Yes | Yes | Connection refused by server |
| `MCP_TOOL_EXECUTION_ERROR` | 500 | No | No | Tool execution failed |
| `MCP_TOOL_NOT_FOUND` | 404 | No | No | Requested tool not found |
| `MCP_INVALID_ARGUMENTS` | 400 | No | No | Invalid arguments provided |
| `MCP_TIMEOUT` | 504 | Yes | Yes | Generic operation timeout |
| `MCP_OPERATION_TIMEOUT` | 504 | Yes | Yes | Specific operation timed out |
| `MCP_DEADLINE_EXCEEDED` | 504 | Yes | Yes | Context deadline exceeded |
| `MCP_AUTHENTICATION_ERROR` | 401 | No | No | Authentication failed |
| `MCP_AUTH_EXPIRED` | 401 | Yes | No | Credentials expired |
| `MCP_AUTH_INVALID` | 401 | No | No | Invalid credentials |
| `MCP_OAUTH_FAILURE` | 401 | No | No | OAuth authentication failed |
| `MCP_RATE_LIMIT_EXCEEDED` | 429 | Yes | Yes | Rate limit exceeded |
| `MCP_QUOTA_EXCEEDED` | 429 | No | No | Quota exceeded |
| `MCP_THROTTLED` | 429 | Yes | Yes | Request throttled |
| `MCP_SERVER_ERROR` | 502 | Yes | Yes | MCP server error |
| `MCP_SERVER_UNAVAILABLE` | 503 | Yes | Yes | Server unavailable |
| `MCP_SERVER_INTERNAL_ERROR` | 502 | Yes | Yes | Server internal error |
| `MCP_CIRCUIT_BREAKER_OPEN` | 503 | Yes | Yes | Circuit breaker is open |
| `MCP_CIRCUIT_BREAKER_HALF_OPEN` | 503 | Yes | Yes | Circuit breaker is half-open |
| `MCP_TOO_MANY_REQUESTS` | 429 | Yes | Yes | Too many concurrent requests |
| `MCP_INVALID_CONFIG` | 400 | No | No | Invalid configuration |
| `MCP_MISSING_CONFIG` | 400 | No | No | Missing configuration |
| `MCP_CONFIG_VALIDATION_ERROR` | 400 | No | No | Configuration validation failed |
| `MCP_RESOURCE_NOT_FOUND` | 404 | No | No | Resource not found |
| `MCP_RESOURCE_LOCKED` | 409 | Yes | Yes | Resource is locked |
| `MCP_RESOURCE_EXHAUSTED` | 503 | Yes | Yes | Resources exhausted |
| `MCP_NETWORK_ERROR` | 503 | Yes | Yes | Network error |
| `MCP_DNS_ERROR` | 503 | Yes | Yes | DNS resolution failed |
| `MCP_TLS_ERROR` | 503 | No | No | TLS/SSL error |

## Testing

Run tests:

```bash
go test -v ./lib/errors/
```

Run benchmarks:

```bash
go test -bench=. ./lib/errors/
```

## License

Part of the AgentAPI project.

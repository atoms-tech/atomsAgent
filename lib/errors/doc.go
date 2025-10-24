// Package errors provides comprehensive error handling for MCP (Model Context Protocol) operations.
//
// This package defines structured error types with rich context, classification utilities,
// and HTTP integration for API responses. All error types implement the standard error
// interface and support Go 1.13+ error unwrapping.
//
// # Error Types
//
// The package provides specialized error types for different failure scenarios:
//
//   - Connection Errors: Network connectivity issues
//   - Tool Execution Errors: MCP tool invocation failures
//   - Timeout Errors: Operation timeouts and deadline exceeded
//   - Authentication Errors: Auth failures including OAuth
//   - Rate Limiting Errors: Rate limits and quota exhaustion
//   - Server Errors: MCP server-side failures
//   - Circuit Breaker Errors: Circuit breaker state errors
//   - Configuration Errors: Invalid or missing configuration
//   - Resource Errors: Resource not found or locked
//   - Network Errors: DNS, TLS, and network failures
//
// # Basic Usage
//
// Create structured errors with context:
//
//	err := errors.NewConnectionError("localhost:8000", originalErr)
//	err.WithOperation("connect").
//	    WithMetadata("attempt", 3).
//	    WithMetadata("timeout", "5s")
//
// Check error properties:
//
//	if errors.IsRetryable(err) {
//	    // Retry the operation
//	}
//
//	if errors.IsTemporary(err) {
//	    // May resolve on its own
//	}
//
// Get HTTP status code:
//
//	statusCode := errors.GetStatusCode(err) // 503
//
// # HTTP Integration
//
// Write error responses to HTTP:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    result, err := callMCPTool(r.Context(), "my_tool")
//	    if err != nil {
//	        errors.WriteErrorResponse(w, err, requestID)
//	        return
//	    }
//	    // ... success response
//	}
//
// This automatically:
//   - Sets the correct HTTP status code
//   - Returns JSON error response
//   - Includes error code, message, and metadata
//   - Adds request ID and timestamp
//
// # Error Wrapping
//
// Wrap FastMCP errors for automatic classification:
//
//	result, err := fastmcpClient.CallTool(ctx, clientID, toolName, args)
//	if err != nil {
//	    // Automatically classifies based on error content
//	    return nil, errors.WrapFastMCPError("call_tool", toolName, err)
//	}
//
// The wrapper automatically detects:
//   - Timeout errors from context or message
//   - Connection errors (refused, failed, etc.)
//   - Authentication errors
//   - Rate limiting errors
//   - Tool-specific errors
//
// # Retry Logic
//
// Use error classification for smart retry logic:
//
//	for attempt := 1; attempt <= maxRetries; attempt++ {
//	    err := doSomething()
//	    if err == nil {
//	        return nil
//	    }
//
//	    if !errors.IsRetryable(err) {
//	        return err // Don't retry permanent failures
//	    }
//
//	    backoff := calculateBackoff(attempt)
//	    time.Sleep(backoff)
//	}
//
// # Structured Logging
//
// Get structured log fields from errors:
//
//	err := errors.NewToolExecutionError("search", originalErr).
//	    WithOperation("call_tool").
//	    WithMetadata("attempt", 3)
//
//	fields := err.LogFields()
//	// Returns map with:
//	// - error_code: MCP_TOOL_EXECUTION_ERROR
//	// - message: Failed to execute tool 'search'
//	// - operation: call_tool
//	// - resource: search
//	// - timestamp: 2025-10-23T12:34:56Z
//	// - temporary: false
//	// - retryable: false
//	// - http_status: 500
//	// - cause: original error
//	// - meta_attempt: 3
//
// # Error Codes
//
// All errors include standardized error codes:
//
//   - MCP_CONNECTION_ERROR - Connection failure
//   - MCP_TOOL_EXECUTION_ERROR - Tool execution failed
//   - MCP_TIMEOUT - Operation timeout
//   - MCP_AUTHENTICATION_ERROR - Auth failure
//   - MCP_RATE_LIMIT_EXCEEDED - Rate limit hit
//   - MCP_CIRCUIT_BREAKER_OPEN - Circuit breaker open
//   - MCP_INVALID_CONFIG - Invalid configuration
//   - And many more...
//
// # Error Context
//
// Errors include rich contextual information:
//
//   - Operation: What operation was being performed
//   - Resource: What resource was involved
//   - Timestamp: When the error occurred
//   - Metadata: Custom key-value data
//   - HTTP Status: Appropriate HTTP status code
//   - Retryable: Whether operation should be retried
//   - Temporary: Whether error is temporary
//
// # Best Practices
//
// 1. Always add context to errors:
//
//	return errors.NewToolExecutionError("search", err).
//	    WithOperation("call_tool").
//	    WithMetadata("query", searchQuery)
//
// 2. Use appropriate error types:
//
//	// Bad
//	return errors.NewServerError("auth failed", err)
//
//	// Good
//	return errors.NewAuthenticationError("invalid token", err)
//
// 3. Check error properties before retry:
//
//	if !errors.IsRetryable(err) {
//	    return err // Don't retry
//	}
//
// 4. Use WrapFastMCPError consistently:
//
//	result, err := fastmcpClient.CallTool(ctx, clientID, toolName, args)
//	if err != nil {
//	    return nil, errors.WrapFastMCPError("call_tool", toolName, err)
//	}
//
// 5. Log with structured fields:
//
//	log.Error().Fields(err.LogFields()).Msg("Operation failed")
//
// # JSON Serialization
//
// Errors serialize to clean JSON for API responses:
//
//	{
//	  "error": {
//	    "code": "MCP_AUTHENTICATION_ERROR",
//	    "message": "Authentication failed: invalid API key",
//	    "operation": "connect",
//	    "resource": "mcp-server",
//	    "metadata": {
//	      "reason": "invalid API key"
//	    },
//	    "temporary": false,
//	    "retryable": false
//	  },
//	  "request_id": "req-123",
//	  "timestamp": "2025-10-23T12:34:56Z"
//	}
//
// # Error Unwrapping
//
// Errors support Go 1.13+ error chains:
//
//	originalErr := errors.New("network timeout")
//	wrappedErr := errors.NewConnectionError("localhost", originalErr)
//
//	if errors.Is(wrappedErr, originalErr) {
//	    // true - can check original error
//	}
//
//	if mcpErr, ok := errors.AsMCPError(wrappedErr); ok {
//	    // Access MCPError fields
//	}
//
// # Performance
//
// The package is optimized for production use:
//
//   - Error creation: ~630 ns/op, 496 B/op
//   - Error wrapping: ~840 ns/op, 488 B/op
//   - JSON serialization: ~1190 ns/op, 448 B/op
//   - Detailed messages: ~2460 ns/op, 1257 B/op
//
// See the README.md file for more detailed documentation and examples.
package errors

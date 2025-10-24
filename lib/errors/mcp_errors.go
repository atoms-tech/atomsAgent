// Package errors provides comprehensive error types and utilities for MCP operations.
// This package defines structured error types with context, wrapping capabilities,
// and utilities for error classification and serialization.
package errors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrorCode represents standardized error codes for MCP operations
type ErrorCode string

const (
	// Connection errors
	ErrCodeConnection        ErrorCode = "MCP_CONNECTION_ERROR"
	ErrCodeConnectionTimeout ErrorCode = "MCP_CONNECTION_TIMEOUT"
	ErrCodeConnectionRefused ErrorCode = "MCP_CONNECTION_REFUSED"

	// Execution errors
	ErrCodeToolExecution    ErrorCode = "MCP_TOOL_EXECUTION_ERROR"
	ErrCodeToolNotFound     ErrorCode = "MCP_TOOL_NOT_FOUND"
	ErrCodeInvalidArguments ErrorCode = "MCP_INVALID_ARGUMENTS"

	// Timeout errors
	ErrCodeTimeout          ErrorCode = "MCP_TIMEOUT"
	ErrCodeOperationTimeout ErrorCode = "MCP_OPERATION_TIMEOUT"
	ErrCodeDeadlineExceeded ErrorCode = "MCP_DEADLINE_EXCEEDED"

	// Authentication errors
	ErrCodeAuthentication ErrorCode = "MCP_AUTHENTICATION_ERROR"
	ErrCodeAuthExpired    ErrorCode = "MCP_AUTH_EXPIRED"
	ErrCodeAuthInvalid    ErrorCode = "MCP_AUTH_INVALID"
	ErrCodeOAuthFailure   ErrorCode = "MCP_OAUTH_FAILURE"

	// Rate limiting errors
	ErrCodeRateLimit     ErrorCode = "MCP_RATE_LIMIT_EXCEEDED"
	ErrCodeQuotaExceeded ErrorCode = "MCP_QUOTA_EXCEEDED"
	ErrCodeThrottled     ErrorCode = "MCP_THROTTLED"

	// Server errors
	ErrCodeServerError       ErrorCode = "MCP_SERVER_ERROR"
	ErrCodeServerUnavailable ErrorCode = "MCP_SERVER_UNAVAILABLE"
	ErrCodeServerInternal    ErrorCode = "MCP_SERVER_INTERNAL_ERROR"

	// Circuit breaker errors
	ErrCodeCircuitOpen     ErrorCode = "MCP_CIRCUIT_BREAKER_OPEN"
	ErrCodeCircuitHalfOpen ErrorCode = "MCP_CIRCUIT_BREAKER_HALF_OPEN"
	ErrCodeTooManyRequests ErrorCode = "MCP_TOO_MANY_REQUESTS"

	// Configuration errors
	ErrCodeInvalidConfig    ErrorCode = "MCP_INVALID_CONFIG"
	ErrCodeMissingConfig    ErrorCode = "MCP_MISSING_CONFIG"
	ErrCodeConfigValidation ErrorCode = "MCP_CONFIG_VALIDATION_ERROR"

	// Resource errors
	ErrCodeResourceNotFound  ErrorCode = "MCP_RESOURCE_NOT_FOUND"
	ErrCodeResourceLocked    ErrorCode = "MCP_RESOURCE_LOCKED"
	ErrCodeResourceExhausted ErrorCode = "MCP_RESOURCE_EXHAUSTED"

	// Network errors
	ErrCodeNetworkError ErrorCode = "MCP_NETWORK_ERROR"
	ErrCodeDNSError     ErrorCode = "MCP_DNS_ERROR"
	ErrCodeTLSError     ErrorCode = "MCP_TLS_ERROR"
)

// MCPError is the base error type for all MCP-related errors with rich context
type MCPError struct {
	// Core error information
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Operation string    `json:"operation,omitempty"`
	Resource  string    `json:"resource,omitempty"`

	// Error chain
	Err error `json:"-"` // Original error, not serialized

	// Contextual information
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`

	// Classification
	Temporary  bool `json:"temporary"`
	Retryable  bool `json:"retryable"`
	HTTPStatus int  `json:"-"` // HTTP status code for API responses
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (operation=%s, resource=%s): %v",
			e.Code, e.Message, e.Operation, e.Resource, e.Err)
	}
	return fmt.Sprintf("%s: %s (operation=%s, resource=%s)",
		e.Code, e.Message, e.Operation, e.Resource)
}

// Unwrap implements error unwrapping for Go 1.13+ error chains
func (e *MCPError) Unwrap() error {
	return e.Err
}

// WithMetadata adds metadata to the error
func (e *MCPError) WithMetadata(key string, value any) *MCPError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithOperation sets the operation context
func (e *MCPError) WithOperation(op string) *MCPError {
	e.Operation = op
	return e
}

// WithResource sets the resource context
func (e *MCPError) WithResource(resource string) *MCPError {
	e.Resource = resource
	return e
}

// newMCPError creates a new MCP error with default values
func newMCPError(code ErrorCode, message string, err error) *MCPError {
	return &MCPError{
		Code:      code,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// ============================================================================
// Connection Errors
// ============================================================================

// NewConnectionError creates a new MCP connection error
func NewConnectionError(endpoint string, err error) *MCPError {
	return newMCPError(
		ErrCodeConnection,
		"Failed to connect to MCP server",
		err,
	).WithResource(endpoint).
		WithMetadata("endpoint", endpoint).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewConnectionTimeoutError creates a timeout error during connection
func NewConnectionTimeoutError(endpoint string, timeout time.Duration, err error) *MCPError {
	return newMCPError(
		ErrCodeConnectionTimeout,
		fmt.Sprintf("Connection timeout after %v", timeout),
		err,
	).WithResource(endpoint).
		WithMetadata("endpoint", endpoint).
		WithMetadata("timeout", timeout.String()).
		withHTTPStatus(http.StatusGatewayTimeout).
		withRetryable(true).
		withTemporary(true)
}

// NewConnectionRefusedError creates a connection refused error
func NewConnectionRefusedError(endpoint string, err error) *MCPError {
	return newMCPError(
		ErrCodeConnectionRefused,
		"Connection refused by MCP server",
		err,
	).WithResource(endpoint).
		WithMetadata("endpoint", endpoint).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Tool Execution Errors
// ============================================================================

// NewToolExecutionError creates a tool execution error
func NewToolExecutionError(toolName string, err error) *MCPError {
	return newMCPError(
		ErrCodeToolExecution,
		fmt.Sprintf("Failed to execute tool '%s'", toolName),
		err,
	).WithResource(toolName).
		WithMetadata("tool", toolName).
		withHTTPStatus(http.StatusInternalServerError).
		withRetryable(false).
		withTemporary(false)
}

// NewToolNotFoundError creates a tool not found error
func NewToolNotFoundError(toolName string) *MCPError {
	return newMCPError(
		ErrCodeToolNotFound,
		fmt.Sprintf("Tool '%s' not found", toolName),
		nil,
	).WithResource(toolName).
		WithMetadata("tool", toolName).
		withHTTPStatus(http.StatusNotFound).
		withRetryable(false).
		withTemporary(false)
}

// NewInvalidArgumentsError creates an invalid arguments error
func NewInvalidArgumentsError(toolName string, reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeInvalidArguments,
		fmt.Sprintf("Invalid arguments for tool '%s': %s", toolName, reason),
		err,
	).WithResource(toolName).
		WithMetadata("tool", toolName).
		WithMetadata("reason", reason).
		withHTTPStatus(http.StatusBadRequest).
		withRetryable(false).
		withTemporary(false)
}

// ============================================================================
// Timeout Errors
// ============================================================================

// NewTimeoutError creates a generic timeout error
func NewTimeoutError(operation string, timeout time.Duration, err error) *MCPError {
	return newMCPError(
		ErrCodeTimeout,
		fmt.Sprintf("Operation timeout after %v", timeout),
		err,
	).WithOperation(operation).
		WithMetadata("timeout", timeout.String()).
		withHTTPStatus(http.StatusGatewayTimeout).
		withRetryable(true).
		withTemporary(true)
}

// NewOperationTimeoutError creates an operation-specific timeout error
func NewOperationTimeoutError(operation string, resource string, timeout time.Duration, err error) *MCPError {
	return newMCPError(
		ErrCodeOperationTimeout,
		fmt.Sprintf("Operation '%s' timed out after %v", operation, timeout),
		err,
	).WithOperation(operation).
		WithResource(resource).
		WithMetadata("timeout", timeout.String()).
		withHTTPStatus(http.StatusGatewayTimeout).
		withRetryable(true).
		withTemporary(true)
}

// NewDeadlineExceededError creates a deadline exceeded error
func NewDeadlineExceededError(operation string, deadline time.Time, err error) *MCPError {
	return newMCPError(
		ErrCodeDeadlineExceeded,
		fmt.Sprintf("Deadline exceeded for operation '%s'", operation),
		err,
	).WithOperation(operation).
		WithMetadata("deadline", deadline.Format(time.RFC3339)).
		withHTTPStatus(http.StatusGatewayTimeout).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Authentication Errors
// ============================================================================

// NewAuthenticationError creates a generic authentication error
func NewAuthenticationError(reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeAuthentication,
		fmt.Sprintf("Authentication failed: %s", reason),
		err,
	).WithMetadata("reason", reason).
		withHTTPStatus(http.StatusUnauthorized).
		withRetryable(false).
		withTemporary(false)
}

// NewAuthExpiredError creates an expired authentication error
func NewAuthExpiredError(expiredAt time.Time, err error) *MCPError {
	return newMCPError(
		ErrCodeAuthExpired,
		"Authentication credentials have expired",
		err,
	).WithMetadata("expired_at", expiredAt.Format(time.RFC3339)).
		withHTTPStatus(http.StatusUnauthorized).
		withRetryable(true).
		withTemporary(false)
}

// NewAuthInvalidError creates an invalid authentication error
func NewAuthInvalidError(reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeAuthInvalid,
		fmt.Sprintf("Invalid authentication: %s", reason),
		err,
	).WithMetadata("reason", reason).
		withHTTPStatus(http.StatusUnauthorized).
		withRetryable(false).
		withTemporary(false)
}

// NewOAuthFailureError creates an OAuth-specific error
func NewOAuthFailureError(provider string, reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeOAuthFailure,
		fmt.Sprintf("OAuth authentication failed for provider '%s': %s", provider, reason),
		err,
	).WithMetadata("provider", provider).
		WithMetadata("reason", reason).
		withHTTPStatus(http.StatusUnauthorized).
		withRetryable(false).
		withTemporary(false)
}

// ============================================================================
// Rate Limiting Errors
// ============================================================================

// NewRateLimitError creates a rate limit exceeded error
func NewRateLimitError(resource string, limit int, retryAfter time.Duration, err error) *MCPError {
	return newMCPError(
		ErrCodeRateLimit,
		fmt.Sprintf("Rate limit exceeded for resource '%s'", resource),
		err,
	).WithResource(resource).
		WithMetadata("limit", limit).
		WithMetadata("retry_after", retryAfter.String()).
		withHTTPStatus(http.StatusTooManyRequests).
		withRetryable(true).
		withTemporary(true)
}

// NewQuotaExceededError creates a quota exceeded error
func NewQuotaExceededError(resource string, quota int, used int, err error) *MCPError {
	return newMCPError(
		ErrCodeQuotaExceeded,
		fmt.Sprintf("Quota exceeded for resource '%s' (%d/%d used)", resource, used, quota),
		err,
	).WithResource(resource).
		WithMetadata("quota", quota).
		WithMetadata("used", used).
		withHTTPStatus(http.StatusTooManyRequests).
		withRetryable(false).
		withTemporary(false)
}

// NewThrottledError creates a throttled request error
func NewThrottledError(resource string, retryAfter time.Duration, err error) *MCPError {
	return newMCPError(
		ErrCodeThrottled,
		fmt.Sprintf("Request throttled for resource '%s'", resource),
		err,
	).WithResource(resource).
		WithMetadata("retry_after", retryAfter.String()).
		withHTTPStatus(http.StatusTooManyRequests).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Server Errors
// ============================================================================

// NewServerError creates a generic MCP server error
func NewServerError(message string, err error) *MCPError {
	return newMCPError(
		ErrCodeServerError,
		message,
		err,
	).withHTTPStatus(http.StatusBadGateway).
		withRetryable(true).
		withTemporary(true)
}

// NewServerUnavailableError creates a server unavailable error
func NewServerUnavailableError(endpoint string, err error) *MCPError {
	return newMCPError(
		ErrCodeServerUnavailable,
		fmt.Sprintf("MCP server unavailable at '%s'", endpoint),
		err,
	).WithResource(endpoint).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewServerInternalError creates a server internal error
func NewServerInternalError(message string, err error) *MCPError {
	return newMCPError(
		ErrCodeServerInternal,
		fmt.Sprintf("MCP server internal error: %s", message),
		err,
	).withHTTPStatus(http.StatusBadGateway).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Circuit Breaker Errors
// ============================================================================

// NewCircuitOpenError creates a circuit breaker open error
func NewCircuitOpenError(operation string, openedAt time.Time, nextRetry time.Time, err error) *MCPError {
	return newMCPError(
		ErrCodeCircuitOpen,
		fmt.Sprintf("Circuit breaker is open for operation '%s'", operation),
		err,
	).WithOperation(operation).
		WithMetadata("opened_at", openedAt.Format(time.RFC3339)).
		WithMetadata("next_retry", nextRetry.Format(time.RFC3339)).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewCircuitHalfOpenError creates a circuit breaker half-open error
func NewCircuitHalfOpenError(operation string, err error) *MCPError {
	return newMCPError(
		ErrCodeCircuitHalfOpen,
		fmt.Sprintf("Circuit breaker is half-open for operation '%s', limiting requests", operation),
		err,
	).WithOperation(operation).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewTooManyRequestsError creates a too many requests error (circuit breaker)
func NewTooManyRequestsError(operation string, err error) *MCPError {
	return newMCPError(
		ErrCodeTooManyRequests,
		fmt.Sprintf("Too many concurrent requests for operation '%s'", operation),
		err,
	).WithOperation(operation).
		withHTTPStatus(http.StatusTooManyRequests).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Configuration Errors
// ============================================================================

// NewInvalidConfigError creates an invalid configuration error
func NewInvalidConfigError(field string, reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeInvalidConfig,
		fmt.Sprintf("Invalid configuration for field '%s': %s", field, reason),
		err,
	).WithMetadata("field", field).
		WithMetadata("reason", reason).
		withHTTPStatus(http.StatusBadRequest).
		withRetryable(false).
		withTemporary(false)
}

// NewMissingConfigError creates a missing configuration error
func NewMissingConfigError(field string, err error) *MCPError {
	return newMCPError(
		ErrCodeMissingConfig,
		fmt.Sprintf("Missing required configuration field '%s'", field),
		err,
	).WithMetadata("field", field).
		withHTTPStatus(http.StatusBadRequest).
		withRetryable(false).
		withTemporary(false)
}

// NewConfigValidationError creates a configuration validation error
func NewConfigValidationError(validationErrors []string, err error) *MCPError {
	return newMCPError(
		ErrCodeConfigValidation,
		"Configuration validation failed",
		err,
	).WithMetadata("validation_errors", validationErrors).
		withHTTPStatus(http.StatusBadRequest).
		withRetryable(false).
		withTemporary(false)
}

// ============================================================================
// Resource Errors
// ============================================================================

// NewResourceNotFoundError creates a resource not found error
func NewResourceNotFoundError(resourceType string, resourceID string, err error) *MCPError {
	return newMCPError(
		ErrCodeResourceNotFound,
		fmt.Sprintf("%s '%s' not found", resourceType, resourceID),
		err,
	).WithResource(resourceID).
		WithMetadata("resource_type", resourceType).
		WithMetadata("resource_id", resourceID).
		withHTTPStatus(http.StatusNotFound).
		withRetryable(false).
		withTemporary(false)
}

// NewResourceLockedError creates a resource locked error
func NewResourceLockedError(resourceType string, resourceID string, err error) *MCPError {
	return newMCPError(
		ErrCodeResourceLocked,
		fmt.Sprintf("%s '%s' is locked", resourceType, resourceID),
		err,
	).WithResource(resourceID).
		WithMetadata("resource_type", resourceType).
		WithMetadata("resource_id", resourceID).
		withHTTPStatus(http.StatusConflict).
		withRetryable(true).
		withTemporary(true)
}

// NewResourceExhaustedError creates a resource exhausted error
func NewResourceExhaustedError(resourceType string, err error) *MCPError {
	return newMCPError(
		ErrCodeResourceExhausted,
		fmt.Sprintf("%s resources exhausted", resourceType),
		err,
	).WithMetadata("resource_type", resourceType).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// ============================================================================
// Network Errors
// ============================================================================

// NewNetworkError creates a generic network error
func NewNetworkError(operation string, err error) *MCPError {
	return newMCPError(
		ErrCodeNetworkError,
		fmt.Sprintf("Network error during operation '%s'", operation),
		err,
	).WithOperation(operation).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewDNSError creates a DNS resolution error
func NewDNSError(hostname string, err error) *MCPError {
	return newMCPError(
		ErrCodeDNSError,
		fmt.Sprintf("DNS resolution failed for hostname '%s'", hostname),
		err,
	).WithMetadata("hostname", hostname).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(true).
		withTemporary(true)
}

// NewTLSError creates a TLS/SSL error
func NewTLSError(reason string, err error) *MCPError {
	return newMCPError(
		ErrCodeTLSError,
		fmt.Sprintf("TLS/SSL error: %s", reason),
		err,
	).WithMetadata("reason", reason).
		withHTTPStatus(http.StatusServiceUnavailable).
		withRetryable(false).
		withTemporary(false)
}

// ============================================================================
// Helper methods for setting error properties
// ============================================================================

func (e *MCPError) withHTTPStatus(status int) *MCPError {
	e.HTTPStatus = status
	return e
}

func (e *MCPError) withRetryable(retryable bool) *MCPError {
	e.Retryable = retryable
	return e
}

func (e *MCPError) withTemporary(temporary bool) *MCPError {
	e.Temporary = temporary
	return e
}

// ============================================================================
// Error Checking Utilities
// ============================================================================

// IsTemporary returns true if the error is temporary and may succeed if retried
func IsTemporary(err error) bool {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.Temporary
	}
	return false
}

// IsRetryable returns true if the error should be retried
func IsRetryable(err error) bool {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.Retryable
	}
	return false
}

// GetStatusCode returns the HTTP status code for the error
func GetStatusCode(err error) int {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		if mcpErr.HTTPStatus > 0 {
			return mcpErr.HTTPStatus
		}
	}
	return http.StatusInternalServerError
}

// GetErrorCode returns the error code for the error
func GetErrorCode(err error) ErrorCode {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.Code
	}
	return ErrCodeServerError
}

// GetDetailedMessage returns a detailed error message with all context
func GetDetailedMessage(err error) string {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		msg := fmt.Sprintf("[%s] %s", mcpErr.Code, mcpErr.Message)

		if mcpErr.Operation != "" {
			msg += fmt.Sprintf(" | operation=%s", mcpErr.Operation)
		}
		if mcpErr.Resource != "" {
			msg += fmt.Sprintf(" | resource=%s", mcpErr.Resource)
		}

		if len(mcpErr.Metadata) > 0 {
			metadataJSON, _ := json.Marshal(mcpErr.Metadata)
			msg += fmt.Sprintf(" | metadata=%s", string(metadataJSON))
		}

		if mcpErr.Err != nil {
			msg += fmt.Sprintf(" | cause=%v", mcpErr.Err)
		}

		return msg
	}
	return err.Error()
}

// IsMCPError returns true if the error is an MCPError
func IsMCPError(err error) bool {
	var mcpErr *MCPError
	return errors.As(err, &mcpErr)
}

// AsMCPError converts an error to MCPError if possible
func AsMCPError(err error) (*MCPError, bool) {
	var mcpErr *MCPError
	ok := errors.As(err, &mcpErr)
	return mcpErr, ok
}

// ============================================================================
// Error Serialization
// ============================================================================

// ErrorResponse represents an HTTP error response
type ErrorResponse struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Operation string         `json:"operation,omitempty"`
	Resource  string         `json:"resource,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Temporary bool           `json:"temporary"`
	Retryable bool           `json:"retryable"`
}

// ToJSON serializes the error to JSON for API responses
func (e *MCPError) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToErrorDetail())
}

// ToErrorDetail converts MCPError to ErrorDetail for serialization
func (e *MCPError) ToErrorDetail() ErrorDetail {
	return ErrorDetail{
		Code:      e.Code,
		Message:   e.Message,
		Operation: e.Operation,
		Resource:  e.Resource,
		Metadata:  e.Metadata,
		Temporary: e.Temporary,
		Retryable: e.Retryable,
	}
}

// ToErrorResponse converts MCPError to a full ErrorResponse
func (e *MCPError) ToErrorResponse(requestID string) ErrorResponse {
	return ErrorResponse{
		Error:     e.ToErrorDetail(),
		RequestID: requestID,
		Timestamp: e.Timestamp.Format(time.RFC3339),
	}
}

// NewErrorResponse creates an ErrorResponse from any error
func NewErrorResponse(err error, requestID string) ErrorResponse {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.ToErrorResponse(requestID)
	}

	// Wrap non-MCP errors
	genericErr := newMCPError(
		ErrCodeServerError,
		err.Error(),
		err,
	).withHTTPStatus(http.StatusInternalServerError).
		withRetryable(false).
		withTemporary(false)

	return genericErr.ToErrorResponse(requestID)
}

// WriteErrorResponse writes an error response to an HTTP response writer
func WriteErrorResponse(w http.ResponseWriter, err error, requestID string) {
	response := NewErrorResponse(err, requestID)
	statusCode := GetStatusCode(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if jsonErr := json.NewEncoder(w).Encode(response); jsonErr != nil {
		// Fallback if JSON encoding fails
		w.Write([]byte(fmt.Sprintf(`{"error":{"code":"ENCODING_ERROR","message":"Failed to encode error response: %s"}}`, jsonErr.Error())))
	}
}

// ============================================================================
// Error Wrapping for FastMCP Errors
// ============================================================================

// WrapFastMCPError wraps a FastMCP error with appropriate MCP error type
func WrapFastMCPError(operation string, resource string, err error) error {
	if err == nil {
		return nil
	}

	// If already an MCPError, update operation/resource context
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		// Update operation if provided (allows more specific context)
		if operation != "" {
			mcpErr.Operation = operation
		}
		// Update resource if provided (allows more specific context)
		if resource != "" {
			mcpErr.Resource = resource
		}
		return mcpErr
	}

	// Analyze error message/type to determine appropriate error type
	errMsg := err.Error()

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(strings.ToLower(errMsg), "timeout") ||
		strings.Contains(strings.ToLower(errMsg), "deadline exceeded") {
		return NewOperationTimeoutError(operation, resource, 30*time.Second, err)
	}

	// Check for connection errors
	if strings.Contains(strings.ToLower(errMsg), "connection refused") {
		return NewConnectionRefusedError(resource, err).WithOperation(operation)
	}
	if strings.Contains(strings.ToLower(errMsg), "connection") ||
		strings.Contains(strings.ToLower(errMsg), "connect") {
		return NewConnectionError(resource, err).WithOperation(operation)
	}

	// Check for authentication errors
	if strings.Contains(strings.ToLower(errMsg), "unauthorized") ||
		strings.Contains(strings.ToLower(errMsg), "authentication") ||
		strings.Contains(strings.ToLower(errMsg), "auth") {
		return NewAuthenticationError("authentication failed", err).
			WithOperation(operation).
			WithResource(resource)
	}

	// Check for rate limiting
	if strings.Contains(strings.ToLower(errMsg), "rate limit") ||
		strings.Contains(strings.ToLower(errMsg), "too many requests") {
		return NewRateLimitError(resource, 0, 0, err).WithOperation(operation)
	}

	// Check for tool-related errors
	if strings.Contains(strings.ToLower(errMsg), "tool not found") ||
		strings.Contains(strings.ToLower(errMsg), "unknown tool") {
		return NewToolNotFoundError(resource).WithOperation(operation)
	}
	if operation == "call_tool" {
		return NewToolExecutionError(resource, err).WithOperation(operation)
	}

	// Default to generic server error
	return NewServerError(errMsg, err).
		WithOperation(operation).
		WithResource(resource)
}

// ============================================================================
// Logging Helpers
// ============================================================================

// LogFields returns a map of fields suitable for structured logging
func (e *MCPError) LogFields() map[string]any {
	fields := map[string]any{
		"error_code": string(e.Code),
		"message":    e.Message,
		"timestamp":  e.Timestamp.Format(time.RFC3339),
		"temporary":  e.Temporary,
		"retryable":  e.Retryable,
	}

	if e.Operation != "" {
		fields["operation"] = e.Operation
	}
	if e.Resource != "" {
		fields["resource"] = e.Resource
	}
	if e.HTTPStatus > 0 {
		fields["http_status"] = e.HTTPStatus
	}
	if e.Err != nil {
		fields["cause"] = e.Err.Error()
	}

	// Add all metadata
	for k, v := range e.Metadata {
		fields["meta_"+k] = v
	}

	return fields
}

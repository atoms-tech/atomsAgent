package errors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMCPError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *MCPError
		contains []string
	}{
		{
			name: "basic error",
			err: newMCPError(
				ErrCodeConnection,
				"Failed to connect",
				nil,
			).WithOperation("connect").WithResource("localhost:8000"),
			contains: []string{
				"MCP_CONNECTION_ERROR",
				"Failed to connect",
				"operation=connect",
				"resource=localhost:8000",
			},
		},
		{
			name: "error with cause",
			err: newMCPError(
				ErrCodeToolExecution,
				"Tool execution failed",
				errors.New("network timeout"),
			).WithOperation("call_tool").WithResource("test_tool"),
			contains: []string{
				"MCP_TOOL_EXECUTION_ERROR",
				"Tool execution failed",
				"network timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("Error string missing expected substring\nGot: %s\nWant substring: %s", errStr, substr)
				}
			}
		})
	}
}

func TestMCPError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := NewConnectionError("localhost", cause)

	if !errors.Is(err, cause) {
		t.Error("Expected error to unwrap to cause")
	}

	var mcpErr *MCPError
	if !errors.As(err, &mcpErr) {
		t.Error("Expected error to be MCPError")
	}
}

func TestMCPError_Metadata(t *testing.T) {
	err := NewConnectionError("localhost", nil).
		WithMetadata("attempt", 3).
		WithMetadata("duration", 5*time.Second)

	if len(err.Metadata) != 3 { // endpoint + attempt + duration
		t.Errorf("Expected 3 metadata entries, got %d", len(err.Metadata))
	}

	if v, ok := err.Metadata["attempt"].(int); !ok || v != 3 {
		t.Errorf("Expected attempt=3, got %v", err.Metadata["attempt"])
	}
}

func TestConnectionErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "connection error",
			constructor: func() *MCPError {
				return NewConnectionError("localhost:8000", nil)
			},
			wantCode:      ErrCodeConnection,
			wantStatus:    http.StatusServiceUnavailable,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "connection timeout",
			constructor: func() *MCPError {
				return NewConnectionTimeoutError("localhost:8000", 5*time.Second, nil)
			},
			wantCode:      ErrCodeConnectionTimeout,
			wantStatus:    http.StatusGatewayTimeout,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "connection refused",
			constructor: func() *MCPError {
				return NewConnectionRefusedError("localhost:8000", nil)
			},
			wantCode:      ErrCodeConnectionRefused,
			wantStatus:    http.StatusServiceUnavailable,
			wantRetryable: true,
			wantTemporary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestToolExecutionErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "tool execution error",
			constructor: func() *MCPError {
				return NewToolExecutionError("test_tool", nil)
			},
			wantCode:      ErrCodeToolExecution,
			wantStatus:    http.StatusInternalServerError,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "tool not found",
			constructor: func() *MCPError {
				return NewToolNotFoundError("missing_tool")
			},
			wantCode:      ErrCodeToolNotFound,
			wantStatus:    http.StatusNotFound,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "invalid arguments",
			constructor: func() *MCPError {
				return NewInvalidArgumentsError("test_tool", "missing required field", nil)
			},
			wantCode:      ErrCodeInvalidArguments,
			wantStatus:    http.StatusBadRequest,
			wantRetryable: false,
			wantTemporary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestTimeoutErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "generic timeout",
			constructor: func() *MCPError {
				return NewTimeoutError("connect", 5*time.Second, nil)
			},
			wantCode:      ErrCodeTimeout,
			wantStatus:    http.StatusGatewayTimeout,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "operation timeout",
			constructor: func() *MCPError {
				return NewOperationTimeoutError("call_tool", "test_tool", 10*time.Second, nil)
			},
			wantCode:      ErrCodeOperationTimeout,
			wantStatus:    http.StatusGatewayTimeout,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "deadline exceeded",
			constructor: func() *MCPError {
				return NewDeadlineExceededError("list_tools", time.Now(), nil)
			},
			wantCode:      ErrCodeDeadlineExceeded,
			wantStatus:    http.StatusGatewayTimeout,
			wantRetryable: true,
			wantTemporary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestAuthenticationErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "generic auth error",
			constructor: func() *MCPError {
				return NewAuthenticationError("invalid credentials", nil)
			},
			wantCode:      ErrCodeAuthentication,
			wantStatus:    http.StatusUnauthorized,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "auth expired",
			constructor: func() *MCPError {
				return NewAuthExpiredError(time.Now().Add(-1*time.Hour), nil)
			},
			wantCode:      ErrCodeAuthExpired,
			wantStatus:    http.StatusUnauthorized,
			wantRetryable: true,
			wantTemporary: false,
		},
		{
			name: "auth invalid",
			constructor: func() *MCPError {
				return NewAuthInvalidError("malformed token", nil)
			},
			wantCode:      ErrCodeAuthInvalid,
			wantStatus:    http.StatusUnauthorized,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "oauth failure",
			constructor: func() *MCPError {
				return NewOAuthFailureError("google", "token refresh failed", nil)
			},
			wantCode:      ErrCodeOAuthFailure,
			wantStatus:    http.StatusUnauthorized,
			wantRetryable: false,
			wantTemporary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestRateLimitErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "rate limit exceeded",
			constructor: func() *MCPError {
				return NewRateLimitError("api_calls", 100, 60*time.Second, nil)
			},
			wantCode:      ErrCodeRateLimit,
			wantStatus:    http.StatusTooManyRequests,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "quota exceeded",
			constructor: func() *MCPError {
				return NewQuotaExceededError("storage", 1000, 1200, nil)
			},
			wantCode:      ErrCodeQuotaExceeded,
			wantStatus:    http.StatusTooManyRequests,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "throttled",
			constructor: func() *MCPError {
				return NewThrottledError("write_ops", 30*time.Second, nil)
			},
			wantCode:      ErrCodeThrottled,
			wantStatus:    http.StatusTooManyRequests,
			wantRetryable: true,
			wantTemporary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestCircuitBreakerErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "circuit open",
			constructor: func() *MCPError {
				return NewCircuitOpenError("connect", time.Now(), time.Now().Add(30*time.Second), nil)
			},
			wantCode:      ErrCodeCircuitOpen,
			wantStatus:    http.StatusServiceUnavailable,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "circuit half-open",
			constructor: func() *MCPError {
				return NewCircuitHalfOpenError("connect", nil)
			},
			wantCode:      ErrCodeCircuitHalfOpen,
			wantStatus:    http.StatusServiceUnavailable,
			wantRetryable: true,
			wantTemporary: true,
		},
		{
			name: "too many requests",
			constructor: func() *MCPError {
				return NewTooManyRequestsError("connect", nil)
			},
			wantCode:      ErrCodeTooManyRequests,
			wantStatus:    http.StatusTooManyRequests,
			wantRetryable: true,
			wantTemporary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestConfigErrors(t *testing.T) {
	tests := []struct {
		name          string
		constructor   func() *MCPError
		wantCode      ErrorCode
		wantStatus    int
		wantRetryable bool
		wantTemporary bool
	}{
		{
			name: "invalid config",
			constructor: func() *MCPError {
				return NewInvalidConfigError("endpoint", "must be a valid URL", nil)
			},
			wantCode:      ErrCodeInvalidConfig,
			wantStatus:    http.StatusBadRequest,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "missing config",
			constructor: func() *MCPError {
				return NewMissingConfigError("api_key", nil)
			},
			wantCode:      ErrCodeMissingConfig,
			wantStatus:    http.StatusBadRequest,
			wantRetryable: false,
			wantTemporary: false,
		},
		{
			name: "config validation",
			constructor: func() *MCPError {
				return NewConfigValidationError([]string{"field1 invalid", "field2 required"}, nil)
			},
			wantCode:      ErrCodeConfigValidation,
			wantStatus:    http.StatusBadRequest,
			wantRetryable: false,
			wantTemporary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			validateError(t, err, tt.wantCode, tt.wantStatus, tt.wantRetryable, tt.wantTemporary)
		})
	}
}

func TestErrorCheckingUtilities(t *testing.T) {
	temporaryErr := NewConnectionError("localhost", nil)
	permanentErr := NewInvalidConfigError("endpoint", "invalid", nil)

	t.Run("IsTemporary", func(t *testing.T) {
		if !IsTemporary(temporaryErr) {
			t.Error("Expected connection error to be temporary")
		}
		if IsTemporary(permanentErr) {
			t.Error("Expected config error to not be temporary")
		}
		if IsTemporary(errors.New("generic error")) {
			t.Error("Expected generic error to not be temporary")
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		if !IsRetryable(temporaryErr) {
			t.Error("Expected connection error to be retryable")
		}
		if IsRetryable(permanentErr) {
			t.Error("Expected config error to not be retryable")
		}
	})

	t.Run("GetStatusCode", func(t *testing.T) {
		if code := GetStatusCode(temporaryErr); code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", code)
		}
		if code := GetStatusCode(permanentErr); code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", code)
		}
		if code := GetStatusCode(errors.New("generic")); code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for generic error, got %d", code)
		}
	})

	t.Run("GetErrorCode", func(t *testing.T) {
		if code := GetErrorCode(temporaryErr); code != ErrCodeConnection {
			t.Errorf("Expected code %s, got %s", ErrCodeConnection, code)
		}
	})

	t.Run("GetDetailedMessage", func(t *testing.T) {
		err := NewToolExecutionError("test_tool", errors.New("network error")).
			WithOperation("call_tool").
			WithMetadata("attempt", 3)

		msg := GetDetailedMessage(err)
		if !strings.Contains(msg, "MCP_TOOL_EXECUTION_ERROR") {
			t.Error("Expected detailed message to contain error code")
		}
		if !strings.Contains(msg, "operation=call_tool") {
			t.Error("Expected detailed message to contain operation")
		}
		if !strings.Contains(msg, "network error") {
			t.Error("Expected detailed message to contain cause")
		}
	})

	t.Run("IsMCPError", func(t *testing.T) {
		if !IsMCPError(temporaryErr) {
			t.Error("Expected IsMCPError to return true for MCPError")
		}
		if IsMCPError(errors.New("generic")) {
			t.Error("Expected IsMCPError to return false for generic error")
		}
	})

	t.Run("AsMCPError", func(t *testing.T) {
		mcpErr, ok := AsMCPError(temporaryErr)
		if !ok || mcpErr == nil {
			t.Error("Expected AsMCPError to convert to MCPError")
		}

		_, ok = AsMCPError(errors.New("generic"))
		if ok {
			t.Error("Expected AsMCPError to fail for generic error")
		}
	})
}

func TestErrorSerialization(t *testing.T) {
	err := NewToolExecutionError("test_tool", errors.New("execution failed")).
		WithOperation("call_tool").
		WithMetadata("attempt", 3)

	t.Run("ToJSON", func(t *testing.T) {
		jsonBytes, jsonErr := err.ToJSON()
		if jsonErr != nil {
			t.Fatalf("Failed to serialize to JSON: %v", jsonErr)
		}

		var detail ErrorDetail
		if unmarshalErr := json.Unmarshal(jsonBytes, &detail); unmarshalErr != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
		}

		if detail.Code != ErrCodeToolExecution {
			t.Errorf("Expected code %s, got %s", ErrCodeToolExecution, detail.Code)
		}
		if detail.Operation != "call_tool" {
			t.Errorf("Expected operation call_tool, got %s", detail.Operation)
		}
	})

	t.Run("ToErrorResponse", func(t *testing.T) {
		response := err.ToErrorResponse("req-123")

		if response.RequestID != "req-123" {
			t.Errorf("Expected request ID req-123, got %s", response.RequestID)
		}
		if response.Error.Code != ErrCodeToolExecution {
			t.Errorf("Expected code %s, got %s", ErrCodeToolExecution, response.Error.Code)
		}
		if response.Timestamp == "" {
			t.Error("Expected timestamp to be set")
		}
	})

	t.Run("NewErrorResponse with MCPError", func(t *testing.T) {
		response := NewErrorResponse(err, "req-456")
		if response.RequestID != "req-456" {
			t.Errorf("Expected request ID req-456, got %s", response.RequestID)
		}
	})

	t.Run("NewErrorResponse with generic error", func(t *testing.T) {
		genericErr := errors.New("something went wrong")
		response := NewErrorResponse(genericErr, "req-789")

		if response.Error.Code != ErrCodeServerError {
			t.Errorf("Expected generic errors to use code %s, got %s", ErrCodeServerError, response.Error.Code)
		}
		if response.Error.Message != "something went wrong" {
			t.Errorf("Expected message to match error text")
		}
	})
}

func TestWriteErrorResponse(t *testing.T) {
	err := NewAuthenticationError("invalid token", nil)

	w := httptest.NewRecorder()
	WriteErrorResponse(w, err, "req-test")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.RequestID != "req-test" {
		t.Errorf("Expected request ID req-test, got %s", response.RequestID)
	}
	if response.Error.Code != ErrCodeAuthentication {
		t.Errorf("Expected code %s, got %s", ErrCodeAuthentication, response.Error.Code)
	}
}

func TestWrapFastMCPError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		resource  string
		err       error
		wantCode  ErrorCode
	}{
		{
			name:      "nil error",
			operation: "connect",
			resource:  "localhost",
			err:       nil,
			wantCode:  "",
		},
		{
			name:      "existing MCPError",
			operation: "call_tool",
			resource:  "test_tool",
			err:       NewConnectionError("localhost", nil),
			wantCode:  ErrCodeConnection,
		},
		{
			name:      "timeout error",
			operation: "connect",
			resource:  "localhost",
			err:       context.DeadlineExceeded,
			wantCode:  ErrCodeOperationTimeout,
		},
		{
			name:      "timeout in message",
			operation: "list_tools",
			resource:  "localhost",
			err:       errors.New("operation timeout after 5s"),
			wantCode:  ErrCodeOperationTimeout,
		},
		{
			name:      "connection refused",
			operation: "connect",
			resource:  "localhost",
			err:       errors.New("connection refused"),
			wantCode:  ErrCodeConnectionRefused,
		},
		{
			name:      "connection error",
			operation: "connect",
			resource:  "localhost",
			err:       errors.New("failed to connect"),
			wantCode:  ErrCodeConnection,
		},
		{
			name:      "auth error",
			operation: "connect",
			resource:  "localhost",
			err:       errors.New("unauthorized access"),
			wantCode:  ErrCodeAuthentication,
		},
		{
			name:      "rate limit",
			operation: "call_tool",
			resource:  "test_tool",
			err:       errors.New("rate limit exceeded"),
			wantCode:  ErrCodeRateLimit,
		},
		{
			name:      "tool not found",
			operation: "call_tool",
			resource:  "missing_tool",
			err:       errors.New("tool not found"),
			wantCode:  ErrCodeToolNotFound,
		},
		{
			name:      "tool execution error",
			operation: "call_tool",
			resource:  "test_tool",
			err:       errors.New("execution failed"),
			wantCode:  ErrCodeToolExecution,
		},
		{
			name:      "generic error",
			operation: "unknown",
			resource:  "resource",
			err:       errors.New("something went wrong"),
			wantCode:  ErrCodeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapFastMCPError(tt.operation, tt.resource, tt.err)

			if tt.err == nil {
				if wrapped != nil {
					t.Error("Expected nil for nil input error")
				}
				return
			}

			mcpErr, ok := AsMCPError(wrapped)
			if !ok {
				t.Fatal("Expected wrapped error to be MCPError")
			}

			if mcpErr.Code != tt.wantCode {
				t.Errorf("Expected code %s, got %s", tt.wantCode, mcpErr.Code)
			}

			if mcpErr.Operation != tt.operation {
				t.Errorf("Expected operation %s, got %s", tt.operation, mcpErr.Operation)
			}

			if mcpErr.Resource != tt.resource {
				t.Errorf("Expected resource %s, got %s", tt.resource, mcpErr.Resource)
			}
		})
	}
}

func TestLogFields(t *testing.T) {
	err := NewToolExecutionError("test_tool", errors.New("execution failed")).
		WithOperation("call_tool").
		WithMetadata("attempt", 3).
		WithMetadata("duration", 5*time.Second)

	fields := err.LogFields()

	expectedFields := []string{
		"error_code",
		"message",
		"timestamp",
		"temporary",
		"retryable",
		"operation",
		"resource",
		"http_status",
		"cause",
		"meta_attempt",
		"meta_duration",
		"meta_tool",
	}

	for _, field := range expectedFields {
		if _, ok := fields[field]; !ok {
			t.Errorf("Expected field %s to be present in log fields", field)
		}
	}

	if fields["error_code"] != string(ErrCodeToolExecution) {
		t.Errorf("Expected error_code to be %s, got %v", ErrCodeToolExecution, fields["error_code"])
	}

	if fields["operation"] != "call_tool" {
		t.Errorf("Expected operation to be call_tool, got %v", fields["operation"])
	}
}

// Helper function to validate error properties
func validateError(t *testing.T, err *MCPError, wantCode ErrorCode, wantStatus int, wantRetryable, wantTemporary bool) {
	t.Helper()

	if err.Code != wantCode {
		t.Errorf("Expected code %s, got %s", wantCode, err.Code)
	}

	if err.HTTPStatus != wantStatus {
		t.Errorf("Expected HTTP status %d, got %d", wantStatus, err.HTTPStatus)
	}

	if err.Retryable != wantRetryable {
		t.Errorf("Expected retryable %v, got %v", wantRetryable, err.Retryable)
	}

	if err.Temporary != wantTemporary {
		t.Errorf("Expected temporary %v, got %v", wantTemporary, err.Temporary)
	}

	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if err.Metadata == nil {
		t.Error("Expected metadata to be initialized")
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewConnectionError("localhost:8000", errors.New("connection failed"))
	}
}

func BenchmarkErrorWrapping(b *testing.B) {
	baseErr := errors.New("connection failed")
	for i := 0; i < b.N; i++ {
		_ = WrapFastMCPError("connect", "localhost:8000", baseErr)
	}
}

func BenchmarkErrorSerialization(b *testing.B) {
	err := NewToolExecutionError("test_tool", errors.New("execution failed")).
		WithOperation("call_tool").
		WithMetadata("attempt", 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = err.ToJSON()
	}
}

func BenchmarkGetDetailedMessage(b *testing.B) {
	err := NewToolExecutionError("test_tool", errors.New("execution failed")).
		WithOperation("call_tool").
		WithMetadata("attempt", 3).
		WithMetadata("duration", 5*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetDetailedMessage(err)
	}
}

// Example usage tests
func ExampleNewConnectionError() {
	err := NewConnectionError("localhost:8000", errors.New("dial tcp: connection refused"))
	fmt.Println(GetStatusCode(err))
	fmt.Println(IsRetryable(err))
	// Output:
	// 503
	// true
}

func ExampleWrapFastMCPError() {
	err := errors.New("timeout waiting for response")
	wrapped := WrapFastMCPError("call_tool", "test_tool", err)

	mcpErr, _ := AsMCPError(wrapped)
	fmt.Println(mcpErr.Code)
	fmt.Println(mcpErr.Operation)
	// Output:
	// MCP_OPERATION_TIMEOUT
	// call_tool
}

func ExampleWriteErrorResponse() {
	err := NewAuthenticationError("invalid API key", nil)

	w := httptest.NewRecorder()
	WriteErrorResponse(w, err, "req-123")

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Content-Type"))
	// Output:
	// 401
	// application/json
}

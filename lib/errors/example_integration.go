package errors

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// This file demonstrates how to integrate the MCP errors package
// with the FastMCP client and HTTP handlers.

// ============================================================================
// Example 1: MCP Client with Error Wrapping
// ============================================================================

// ExampleMCPClient shows how to wrap an MCP client to use structured errors
type ExampleMCPClient struct {
	endpoint string
	timeout  time.Duration
}

// Connect demonstrates connection error handling
func (c *ExampleMCPClient) Connect(ctx context.Context) error {
	// Simulate connection attempt
	err := c.attemptConnect(ctx)
	if err != nil {
		// Wrap with appropriate error type
		if isTimeout(err) {
			return NewConnectionTimeoutError(c.endpoint, c.timeout, err)
		}
		if isRefused(err) {
			return NewConnectionRefusedError(c.endpoint, err)
		}
		return NewConnectionError(c.endpoint, err)
	}
	return nil
}

// CallTool demonstrates tool execution error handling
func (c *ExampleMCPClient) CallTool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	// Validate arguments first
	if err := c.validateArguments(toolName, args); err != nil {
		return nil, NewInvalidArgumentsError(toolName, err.Error(), err)
	}

	// Execute tool with timeout
	start := time.Now()
	result, err := c.executeTool(ctx, toolName, args)
	duration := time.Since(start)

	if err != nil {
		// Wrap error with context
		mcpErr := WrapFastMCPError("call_tool", toolName, err)

		// Add execution metadata
		if e, ok := AsMCPError(mcpErr); ok {
			e.WithMetadata("duration", duration.String()).
				WithMetadata("args_count", len(args))
		}

		return nil, mcpErr
	}

	return result, nil
}

// Helper methods (simulated)
func (c *ExampleMCPClient) attemptConnect(ctx context.Context) error {
	// Simulated implementation
	return nil
}

func (c *ExampleMCPClient) executeTool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	// Simulated implementation
	return nil, nil
}

func (c *ExampleMCPClient) validateArguments(toolName string, args map[string]any) error {
	// Simulated validation
	return nil
}

func isTimeout(err error) bool {
	// Check if error is a timeout
	return false
}

func isRefused(err error) bool {
	// Check if connection was refused
	return false
}

// ============================================================================
// Example 2: HTTP Handler with Error Responses
// ============================================================================

// ExampleHTTPHandler demonstrates HTTP error handling
type ExampleHTTPHandler struct {
	client *ExampleMCPClient
}

// HandleToolCall shows how to handle errors in HTTP endpoints
func (h *ExampleHTTPHandler) HandleToolCall(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Parse request
	var req struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}

	if err := parseJSON(r, &req); err != nil {
		// Invalid request format
		mcpErr := NewInvalidConfigError("request_body",
			"failed to parse JSON", err)
		WriteErrorResponse(w, mcpErr, requestID)
		return
	}

	// Validate required fields
	if req.Tool == "" {
		mcpErr := NewMissingConfigError("tool", nil)
		WriteErrorResponse(w, mcpErr, requestID)
		return
	}

	// Call tool with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.client.CallTool(ctx, req.Tool, req.Args)
	if err != nil {
		// Error is already wrapped by CallTool
		// Log with structured fields
		if mcpErr, ok := AsMCPError(err); ok {
			logError(mcpErr, requestID)
		}

		// Write error response (automatic status code and JSON)
		WriteErrorResponse(w, err, requestID)
		return
	}

	// Success response
	writeSuccess(w, result, requestID)
}

// Helper functions
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func parseJSON(r *http.Request, v any) error {
	// Simulated JSON parsing
	return nil
}

func logError(err *MCPError, requestID string) {
	fields := err.LogFields()
	fields["request_id"] = requestID
	log.Printf("Error: %+v", fields)
}

func writeSuccess(w http.ResponseWriter, result any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Write JSON response
}

// ============================================================================
// Example 3: Retry Logic with Error Classification
// ============================================================================

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig returns default retry settings
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry non-retryable errors
		if !IsRetryable(err) {
			log.Printf("Non-retryable error: %v", GetDetailedMessage(err))
			return err
		}

		// Don't sleep after last attempt
		if attempt >= config.MaxAttempts {
			break
		}

		// Calculate backoff delay
		delay := calculateBackoff(attempt, config.BaseDelay, config.MaxDelay)

		// Log retry attempt
		log.Printf("Attempt %d/%d failed (retrying in %v): %v",
			attempt, config.MaxAttempts, delay, err)

		// Check for context cancellation during sleep
		select {
		case <-ctx.Done():
			return NewDeadlineExceededError("retry", time.Now(), ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	return NewServerError(
		fmt.Sprintf("operation failed after %d attempts", config.MaxAttempts),
		lastErr,
	).WithMetadata("attempts", config.MaxAttempts)
}

func calculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: base * 2^(attempt-1)
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// ============================================================================
// Example 4: Circuit Breaker Integration
// ============================================================================

// CircuitBreakerWrapper wraps operations with circuit breaker
type CircuitBreakerWrapper struct {
	client *ExampleMCPClient
	// circuitBreaker would be from lib/resilience package
}

// CallToolWithCircuitBreaker shows circuit breaker error handling
func (w *CircuitBreakerWrapper) CallToolWithCircuitBreaker(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (map[string]any, error) {
	// Simulated circuit breaker execution
	// In real code, this would use lib/resilience.CircuitBreaker

	// Check circuit state
	state := w.getCircuitState()
	if state == "open" {
		openedAt := w.getOpenedAt()
		nextRetry := openedAt.Add(30 * time.Second)

		return nil, NewCircuitOpenError(
			"call_tool",
			openedAt,
			nextRetry,
			nil,
		).WithResource(toolName)
	}

	if state == "half-open" {
		// Limited requests allowed
		if !w.canProceed() {
			return nil, NewCircuitHalfOpenError("call_tool", nil).
				WithResource(toolName)
		}
	}

	// Execute the operation
	result, err := w.client.CallTool(ctx, toolName, args)
	if err != nil {
		w.recordFailure(err)
		return nil, err
	}

	w.recordSuccess()
	return result, nil
}

// Helper methods (simulated)
func (w *CircuitBreakerWrapper) getCircuitState() string {
	return "closed"
}

func (w *CircuitBreakerWrapper) getOpenedAt() time.Time {
	return time.Now()
}

func (w *CircuitBreakerWrapper) canProceed() bool {
	return true
}

func (w *CircuitBreakerWrapper) recordFailure(err error) {}
func (w *CircuitBreakerWrapper) recordSuccess()          {}

// ============================================================================
// Example 5: Rate Limiting Error Handling
// ============================================================================

// RateLimitedClient demonstrates rate limit error handling
type RateLimitedClient struct {
	client      *ExampleMCPClient
	rateLimiter *RateLimiter
}

// RateLimiter simulates a rate limiter
type RateLimiter struct {
	limit    int
	window   time.Duration
	requests int
}

// CallWithRateLimit demonstrates rate limit checking
func (c *RateLimitedClient) CallWithRateLimit(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (map[string]any, error) {
	// Check rate limit
	if !c.rateLimiter.Allow() {
		retryAfter := c.rateLimiter.RetryAfter()
		return nil, NewRateLimitError(
			"tool_calls",
			c.rateLimiter.limit,
			retryAfter,
			nil,
		).WithResource(toolName).
			WithMetadata("current_rate", c.rateLimiter.requests).
			WithMetadata("window", c.rateLimiter.window.String())
	}

	// Execute tool call
	return c.client.CallTool(ctx, toolName, args)
}

// Allow checks if request is allowed
func (rl *RateLimiter) Allow() bool {
	// Simulated rate limit check
	return true
}

// RetryAfter returns duration until next allowed request
func (rl *RateLimiter) RetryAfter() time.Duration {
	return 60 * time.Second
}

// ============================================================================
// Example 6: Authentication Error Handling
// ============================================================================

// AuthenticatedClient demonstrates authentication error handling
type AuthenticatedClient struct {
	client   *ExampleMCPClient
	tokenMgr *TokenManager
}

// TokenManager manages authentication tokens
type TokenManager struct {
	token     string
	expiresAt time.Time
}

// CallWithAuth demonstrates authentication error handling
func (c *AuthenticatedClient) CallWithAuth(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (map[string]any, error) {
	// Check token expiration
	if c.tokenMgr.IsExpired() {
		// Try to refresh token
		if err := c.tokenMgr.Refresh(ctx); err != nil {
			return nil, NewAuthExpiredError(
				c.tokenMgr.expiresAt,
				err,
			).WithMetadata("refresh_failed", true)
		}
	}

	// Validate token
	if !c.tokenMgr.IsValid() {
		return nil, NewAuthInvalidError(
			"token validation failed",
			nil,
		)
	}

	// Execute with valid token
	result, err := c.client.CallTool(ctx, toolName, args)
	if err != nil {
		// Check if error is authentication-related
		if isAuthError(err) {
			// Token might be invalid, force refresh
			c.tokenMgr.Invalidate()
		}
		return nil, err
	}

	return result, nil
}

// Token manager methods
func (tm *TokenManager) IsExpired() bool {
	return time.Now().After(tm.expiresAt)
}

func (tm *TokenManager) IsValid() bool {
	return tm.token != "" && !tm.IsExpired()
}

func (tm *TokenManager) Refresh(ctx context.Context) error {
	// Simulated token refresh
	return nil
}

func (tm *TokenManager) Invalidate() {
	tm.token = ""
}

func isAuthError(err error) bool {
	code := GetErrorCode(err)
	return code == ErrCodeAuthentication ||
		code == ErrCodeAuthExpired ||
		code == ErrCodeAuthInvalid
}

// ============================================================================
// Example 7: Configuration Validation
// ============================================================================

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	Endpoint   string
	Type       string
	Timeout    time.Duration
	MaxRetries int
	Auth       *AuthConfig
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string
	Token    string
	ClientID string
	Secret   string
}

// ValidateMCPConfig validates MCP configuration and returns detailed errors
func ValidateMCPConfig(cfg *MCPServerConfig) error {
	var validationErrors []string

	// Validate endpoint
	if cfg.Endpoint == "" {
		validationErrors = append(validationErrors, "endpoint is required")
	}

	// Validate type
	validTypes := map[string]bool{
		"http":  true,
		"sse":   true,
		"stdio": true,
	}
	if !validTypes[cfg.Type] {
		validationErrors = append(validationErrors,
			"type must be one of: http, sse, stdio")
	}

	// Validate timeout
	if cfg.Timeout <= 0 {
		validationErrors = append(validationErrors,
			"timeout must be greater than 0")
	}

	// Validate auth if provided
	if cfg.Auth != nil {
		if err := validateAuthConfig(cfg.Auth); err != nil {
			if mcpErr, ok := AsMCPError(err); ok {
				// Extract validation errors from auth config error
				if authErrs, ok := mcpErr.Metadata["validation_errors"].([]string); ok {
					validationErrors = append(validationErrors, authErrs...)
				}
			}
		}
	}

	// Return validation error if any errors found
	if len(validationErrors) > 0 {
		return NewConfigValidationError(validationErrors, nil)
	}

	return nil
}

func validateAuthConfig(auth *AuthConfig) error {
	var validationErrors []string

	validAuthTypes := map[string]bool{
		"bearer": true,
		"oauth":  true,
		"none":   true,
	}

	if !validAuthTypes[auth.Type] {
		validationErrors = append(validationErrors,
			"auth type must be one of: bearer, oauth, none")
	}

	if auth.Type == "bearer" && auth.Token == "" {
		validationErrors = append(validationErrors,
			"token is required for bearer authentication")
	}

	if auth.Type == "oauth" {
		if auth.ClientID == "" {
			validationErrors = append(validationErrors,
				"client_id is required for OAuth authentication")
		}
		if auth.Secret == "" {
			validationErrors = append(validationErrors,
				"secret is required for OAuth authentication")
		}
	}

	if len(validationErrors) > 0 {
		return NewConfigValidationError(validationErrors, nil)
	}

	return nil
}

// ============================================================================
// Example 8: Comprehensive Error Handling Service
// ============================================================================

// MCPService demonstrates comprehensive error handling
type MCPService struct {
	client         *ExampleMCPClient
	retryConfig    RetryConfig
	rateLimiter    *RateLimiter
	circuitBreaker *CircuitBreakerWrapper
	tokenMgr       *TokenManager
}

// NewMCPService creates a new MCP service with error handling
func NewMCPService(endpoint string) (*MCPService, error) {
	// Validate configuration
	cfg := &MCPServerConfig{
		Endpoint:   endpoint,
		Type:       "http",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	if err := ValidateMCPConfig(cfg); err != nil {
		return nil, err
	}

	client := &ExampleMCPClient{
		endpoint: endpoint,
		timeout:  cfg.Timeout,
	}

	return &MCPService{
		client:      client,
		retryConfig: DefaultRetryConfig(),
		rateLimiter: &RateLimiter{
			limit:  100,
			window: time.Minute,
		},
		circuitBreaker: &CircuitBreakerWrapper{
			client: client,
		},
		tokenMgr: &TokenManager{},
	}, nil
}

// CallTool executes a tool with comprehensive error handling
func (s *MCPService) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (map[string]any, error) {
	// Wrap in retry logic
	var result map[string]any
	err := RetryWithBackoff(ctx, s.retryConfig, func() error {
		var callErr error
		result, callErr = s.callToolInternal(ctx, toolName, args)
		return callErr
	})

	return result, err
}

func (s *MCPService) callToolInternal(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (map[string]any, error) {
	// Check rate limit
	if !s.rateLimiter.Allow() {
		return nil, NewRateLimitError(
			"tool_calls",
			s.rateLimiter.limit,
			s.rateLimiter.RetryAfter(),
			nil,
		)
	}

	// Check authentication
	if !s.tokenMgr.IsValid() {
		if err := s.tokenMgr.Refresh(ctx); err != nil {
			return nil, NewAuthExpiredError(s.tokenMgr.expiresAt, err)
		}
	}

	// Execute with circuit breaker
	return s.circuitBreaker.CallToolWithCircuitBreaker(ctx, toolName, args)
}

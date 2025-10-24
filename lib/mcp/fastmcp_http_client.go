package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

const (
	// Default configuration values
	defaultBaseURL = "http://localhost:8000"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3

	// API endpoints
	endpointConnect    = "/api/connect"
	endpointDisconnect = "/api/disconnect"
	endpointCallTool   = "/api/call_tool"
	endpointListTools  = "/api/list_tools"
	endpointHealth     = "/health"
)

// FastMCPHTTPClient is an HTTP client for the FastMCP service
type FastMCPHTTPClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	mu         sync.RWMutex
}

// ConnectRequest represents a request to connect to an MCP server
type ConnectRequest struct {
	ClientID     string            `json:"client_id"`
	Transport    string            `json:"transport"`     // stdio, sse, or http
	OAuthProvider string           `json:"oauth_provider,omitempty"`
	MCPURL       string            `json:"mcp_url,omitempty"` // For SSE/HTTP transports
	Command      string            `json:"command,omitempty"` // For stdio transport
	Args         []string          `json:"args,omitempty"`    // For stdio transport
	Env          map[string]string `json:"env,omitempty"`     // Environment variables
}

// DisconnectRequest represents a request to disconnect from an MCP server
type DisconnectRequest struct {
	ClientID string `json:"client_id"`
}

// ToolCallRequest represents a request to call a tool
type ToolCallRequest struct {
	ClientID  string         `json:"client_id"`
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
}

// ListToolsRequest represents a request to list tools
type ListToolsRequest struct {
	ClientID string `json:"client_id"`
}

// ConnectResponse represents the response from a connect request
type ConnectResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DisconnectResponse represents the response from a disconnect request
type DisconnectResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ToolCallResponse represents the response from a tool call
type ToolCallResponse struct {
	Success bool           `json:"success"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ListToolsResponse represents the response from listing tools
type ListToolsResponse struct {
	Success bool   `json:"success"`
	Tools   []Tool `json:"tools,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HealthResponse represents the response from the health check
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// HTTPMCPConfig represents the configuration for connecting to an MCP server via HTTP
type HTTPMCPConfig struct {
	Transport     string            `json:"transport"`     // stdio, sse, or http
	OAuthProvider string            `json:"oauth_provider,omitempty"`
	MCPURL        string            `json:"mcp_url,omitempty"` // For SSE/HTTP
	Command       string            `json:"command,omitempty"` // For stdio
	Args          []string          `json:"args,omitempty"`    // For stdio
	Env           map[string]string `json:"env,omitempty"`
}

// NewFastMCPHTTPClient creates a new FastMCP HTTP client
func NewFastMCPHTTPClient(baseURL string) *FastMCPHTTPClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &FastMCPHTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		timeout: defaultTimeout,
	}
}

// Connect establishes a connection to an MCP server
func (c *FastMCPHTTPClient) Connect(ctx context.Context, clientID string, config HTTPMCPConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := ConnectRequest{
		ClientID:      clientID,
		Transport:     config.Transport,
		OAuthProvider: config.OAuthProvider,
		MCPURL:        config.MCPURL,
		Command:       config.Command,
		Args:          config.Args,
		Env:           config.Env,
	}

	var resp ConnectResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, endpointConnect, req, &resp); err != nil {
		return fmt.Errorf("connect request failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("connect failed: %s", resp.Error)
	}

	return nil
}

// Disconnect closes the connection to an MCP server
func (c *FastMCPHTTPClient) Disconnect(ctx context.Context, clientID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := DisconnectRequest{
		ClientID: clientID,
	}

	var resp DisconnectResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, endpointDisconnect, req, &resp); err != nil {
		return fmt.Errorf("disconnect request failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("disconnect failed: %s", resp.Error)
	}

	return nil
}

// CallTool invokes a tool on the MCP server
func (c *FastMCPHTTPClient) CallTool(ctx context.Context, clientID string, toolName string, args map[string]any) (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	req := ToolCallRequest{
		ClientID:  clientID,
		ToolName:  toolName,
		Arguments: args,
	}

	var resp ToolCallResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, endpointCallTool, req, &resp); err != nil {
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("tool call failed: %s", resp.Error)
	}

	return resp.Result, nil
}

// ListTools retrieves the list of available tools from the MCP server
func (c *FastMCPHTTPClient) ListTools(ctx context.Context, clientID string) ([]Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	req := ListToolsRequest{
		ClientID: clientID,
	}

	var resp ListToolsResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, endpointListTools, req, &resp); err != nil {
		return nil, fmt.Errorf("list tools request failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("list tools failed: %s", resp.Error)
	}

	return resp.Tools, nil
}

// Health checks the health of the FastMCP service
func (c *FastMCPHTTPClient) Health(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var resp HealthResponse
	if err := c.doRequest(ctx, http.MethodGet, endpointHealth, nil, &resp); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != "ok" && resp.Status != "healthy" {
		return fmt.Errorf("service unhealthy: %s", resp.Status)
	}

	return nil
}

// SetTimeout sets the HTTP client timeout
func (c *FastMCPHTTPClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// doRequest performs an HTTP request without retry logic
func (c *FastMCPHTTPClient) doRequest(ctx context.Context, method, endpoint string, reqBody, respBody any) error {
	url := c.baseURL + endpoint

	var body io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("request timeout: %w", ctx.Err())
			}
			return fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract error message from response
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respData, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error)
		}
		if json.Unmarshal(respData, &errResp) == nil && errResp.Message != "" {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respData))
	}

	// Unmarshal response if respBody is provided
	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry logic
func (c *FastMCPHTTPClient) doRequestWithRetry(ctx context.Context, method, endpoint string, reqBody, respBody any) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Don't retry if context is already cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := c.doRequest(ctx, method, endpoint, reqBody, respBody)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx) or context errors
		if ctx.Err() != nil {
			return err
		}
		if !isRetryableError(err) {
			return err
		}

		// Don't sleep on the last attempt
		if attempt < maxRetries {
			backoff := calculateBackoff(attempt)
			select {
			case <-time.After(backoff):
				// Continue to next retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on network errors, timeouts, and 5xx server errors
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"HTTP 500",
		"HTTP 502",
		"HTTP 503",
		"HTTP 504",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// calculateBackoff calculates exponential backoff duration
func calculateBackoff(attempt int) time.Duration {
	// Base delay: 100ms, 200ms, 400ms, 800ms...
	baseDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second

	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

// toLower converts a string to lowercase
func toLower(s string) string {
	var result []rune
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

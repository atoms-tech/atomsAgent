package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Enhanced retry constants with exponential backoff and jitter
const (
	enhancedInitialRetryDelay = 100 * time.Millisecond
	enhancedMaxRetryDelay     = 30 * time.Second
	enhancedRetryMultiplier   = 2.0
	enhancedJitterPercent     = 0.10 // ±10% jitter
	enhancedRetryTimeout      = 5 * time.Minute
	enhancedMaxRetries        = 3
	enhancedPerRequestTimeout = 30 * time.Second
)

// EnhancedRetryableStatusCodes are HTTP status codes that should trigger a retry
var EnhancedRetryableStatusCodes = map[int]bool{
	429: true, // Too Many Requests
	500: true, // Internal Server Error
	502: true, // Bad Gateway
	503: true, // Service Unavailable
	504: true, // Gateway Timeout
}

// DeadLetterQueue interface for storing failed operations
type DeadLetterQueue interface {
	Store(ctx context.Context, operation FailedOperation) error
	Get(ctx context.Context, id string) (*FailedOperation, error)
	List(ctx context.Context, limit int) ([]FailedOperation, error)
	Delete(ctx context.Context, id string) error
	Cleanup(ctx context.Context, olderThan time.Duration) error
}

// FailedOperation represents an operation that failed after all retries
type FailedOperation struct {
	ID          string                 `json:"id"`
	ClientID    string                 `json:"client_id"`
	Operation   string                 `json:"operation"` // connect, disconnect, call_tool, list_tools
	Endpoint    string                 `json:"endpoint"`
	RequestBody map[string]interface{} `json:"request_body"`
	LastError   string                 `json:"last_error"`
	RetryCount  int                    `json:"retry_count"`
	CreatedAt   time.Time              `json:"created_at"`
	LastAttempt time.Time              `json:"last_attempt"`
}

// MCPMetrics holds Prometheus metrics for MCP operations
type MCPMetrics struct {
	RetryCount     *prometheus.CounterVec
	RetryBackoff   *prometheus.HistogramVec
	OperationTotal *prometheus.CounterVec
	OperationTime  *prometheus.HistogramVec
	DLQOperations  prometheus.Counter
}

// InitMCPMetrics initializes Prometheus metrics for the MCP client
func InitMCPMetrics(namespace string) *MCPMetrics {
	if namespace == "" {
		namespace = "fastmcp"
	}

	return &MCPMetrics{
		RetryCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "retry_attempts_total",
				Help:      "Total number of retry attempts by operation and reason",
			},
			[]string{"operation", "reason"},
		),
		RetryBackoff: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "retry_backoff_seconds",
				Help:      "Backoff delay duration in seconds by operation",
				Buckets:   []float64{.001, .01, .1, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"operation"},
		),
		OperationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "operations_total",
				Help:      "Total number of operations by type and status",
			},
			[]string{"operation", "status"},
		),
		OperationTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "operation_duration_seconds",
				Help:      "Operation duration in seconds by operation type",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
			},
			[]string{"operation"},
		),
		DLQOperations: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "dlq_operations_total",
				Help:      "Total number of operations sent to dead letter queue",
			},
		),
	}
}

// EnhancedFastMCPHTTPClient is an enhanced HTTP client for the FastMCP service with
// improved retry logic, dead letter queue support, and comprehensive metrics
type EnhancedFastMCPHTTPClient struct {
	*FastMCPHTTPClient // Embed the original client for backward compatibility

	// Dead Letter Queue support
	dlqStore DeadLetterQueue

	// Metrics
	metrics *MCPMetrics

	// Random generator for jitter (with seed for reproducibility)
	rng *rand.Rand

	mu sync.RWMutex
}

// NewEnhancedFastMCPHTTPClient creates a new enhanced FastMCP HTTP client
func NewEnhancedFastMCPHTTPClient(baseURL string) *EnhancedFastMCPHTTPClient {
	return &EnhancedFastMCPHTTPClient{
		FastMCPHTTPClient: NewFastMCPHTTPClient(baseURL),
		metrics:           InitMCPMetrics("fastmcp"),
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewEnhancedFastMCPHTTPClientWithOptions creates a new enhanced FastMCP HTTP client with custom options
func NewEnhancedFastMCPHTTPClientWithOptions(baseURL string, dlq DeadLetterQueue, metrics *MCPMetrics) *EnhancedFastMCPHTTPClient {
	if metrics == nil {
		metrics = InitMCPMetrics("fastmcp")
	}

	return &EnhancedFastMCPHTTPClient{
		FastMCPHTTPClient: NewFastMCPHTTPClient(baseURL),
		dlqStore:          dlq,
		metrics:           metrics,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetDeadLetterQueue sets the dead letter queue for failed operations
func (c *EnhancedFastMCPHTTPClient) SetDeadLetterQueue(dlq DeadLetterQueue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dlqStore = dlq
}

// ConnectWithRetry establishes a connection to an MCP server with enhanced retry logic
func (c *EnhancedFastMCPHTTPClient) ConnectWithRetry(ctx context.Context, clientID string, config HTTPMCPConfig) error {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.OperationTime.WithLabelValues("connect").Observe(time.Since(start).Seconds())
		}
	}()

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
	err := c.doRequestWithEnhancedRetry(ctx, "connect", http.MethodPost, endpointConnect, req, &resp)

	if err != nil {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("connect", "error").Inc()
		}
		// Store in DLQ if available
		c.storeToDLQ(ctx, "connect", clientID, endpointConnect, req, err)
		return fmt.Errorf("connect request failed: %w", err)
	}

	if !resp.Success {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("connect", "failure").Inc()
		}
		return fmt.Errorf("connect failed: %s", resp.Error)
	}

	if c.metrics != nil {
		c.metrics.OperationTotal.WithLabelValues("connect", "success").Inc()
	}
	return nil
}

// CallToolWithRetry invokes a tool on the MCP server with enhanced retry logic
func (c *EnhancedFastMCPHTTPClient) CallToolWithRetry(ctx context.Context, clientID string, toolName string, args map[string]any) (map[string]any, error) {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.OperationTime.WithLabelValues("call_tool").Observe(time.Since(start).Seconds())
		}
	}()

	req := ToolCallRequest{
		ClientID:  clientID,
		ToolName:  toolName,
		Arguments: args,
	}

	var resp ToolCallResponse
	err := c.doRequestWithEnhancedRetry(ctx, "call_tool", http.MethodPost, endpointCallTool, req, &resp)

	if err != nil {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("call_tool", "error").Inc()
		}
		// Store in DLQ if available
		c.storeToDLQ(ctx, "call_tool", clientID, endpointCallTool, req, err)
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	if !resp.Success {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("call_tool", "failure").Inc()
		}
		return nil, fmt.Errorf("tool call failed: %s", resp.Error)
	}

	if c.metrics != nil {
		c.metrics.OperationTotal.WithLabelValues("call_tool", "success").Inc()
	}
	return resp.Result, nil
}

// ListToolsWithRetry retrieves the list of available tools from the MCP server with enhanced retry logic
func (c *EnhancedFastMCPHTTPClient) ListToolsWithRetry(ctx context.Context, clientID string) ([]Tool, error) {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.OperationTime.WithLabelValues("list_tools").Observe(time.Since(start).Seconds())
		}
	}()

	req := ListToolsRequest{
		ClientID: clientID,
	}

	var resp ListToolsResponse
	err := c.doRequestWithEnhancedRetry(ctx, "list_tools", http.MethodPost, endpointListTools, req, &resp)

	if err != nil {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("list_tools", "error").Inc()
		}
		// Store in DLQ if available
		c.storeToDLQ(ctx, "list_tools", clientID, endpointListTools, req, err)
		return nil, fmt.Errorf("list tools request failed: %w", err)
	}

	if !resp.Success {
		if c.metrics != nil {
			c.metrics.OperationTotal.WithLabelValues("list_tools", "failure").Inc()
		}
		return nil, fmt.Errorf("list tools failed: %s", resp.Error)
	}

	if c.metrics != nil {
		c.metrics.OperationTotal.WithLabelValues("list_tools", "success").Inc()
	}
	return resp.Tools, nil
}

// doRequestWithEnhancedRetry performs an HTTP request with enhanced exponential backoff, jitter, and metrics
func (c *EnhancedFastMCPHTTPClient) doRequestWithEnhancedRetry(ctx context.Context, operation, method, endpoint string, reqBody, respBody any) error {
	// Create a timeout context for the entire retry operation
	retryCtx, cancel := context.WithTimeout(ctx, enhancedRetryTimeout)
	defer cancel()

	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= enhancedMaxRetries; attempt++ {
		// Don't retry if context is already cancelled
		if retryCtx.Err() != nil {
			if c.metrics != nil {
				c.metrics.RetryCount.WithLabelValues(operation, "context_cancelled").Inc()
			}
			return retryCtx.Err()
		}

		// Create per-request context with timeout
		reqCtx, reqCancel := context.WithTimeout(retryCtx, enhancedPerRequestTimeout)
		err := c.doRequestInternal(reqCtx, method, endpoint, reqBody, respBody)
		reqCancel()

		if err == nil {
			// Success - record metrics if this was a retry
			if c.metrics != nil && attempt > 0 {
				c.metrics.RetryCount.WithLabelValues(operation, "success_after_retry").Inc()
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if retryCtx.Err() != nil {
			// Context cancelled or deadline exceeded
			if c.metrics != nil {
				c.metrics.RetryCount.WithLabelValues(operation, "context_cancelled").Inc()
			}
			return retryCtx.Err()
		}

		if !c.isRetryableErrorEnhanced(err) {
			// Non-retryable error
			if c.metrics != nil {
				c.metrics.RetryCount.WithLabelValues(operation, "non_retryable_error").Inc()
			}
			return err
		}

		// Don't sleep on the last attempt
		if attempt < enhancedMaxRetries {
			backoff := c.calculateBackoffWithJitter(attempt)

			// Log retry attempt with comprehensive details
			log.Printf("[FastMCP Enhanced] Retry %d/%d for %s after %v (error: %v, cumulative time: %v)",
				attempt+1, enhancedMaxRetries, operation, backoff, err, time.Since(startTime))

			// Record metrics
			if c.metrics != nil {
				c.metrics.RetryCount.WithLabelValues(operation, "retryable_error").Inc()
				c.metrics.RetryBackoff.WithLabelValues(operation).Observe(backoff.Seconds())
			}

			// Wait for backoff duration or context cancellation
			select {
			case <-time.After(backoff):
				// Continue to next retry
			case <-retryCtx.Done():
				if c.metrics != nil {
					c.metrics.RetryCount.WithLabelValues(operation, "context_cancelled_during_backoff").Inc()
				}
				return retryCtx.Err()
			}
		}
	}

	// All retries exhausted
	log.Printf("[FastMCP Enhanced] Max retries exceeded for %s (total time: %v, last error: %v)",
		operation, time.Since(startTime), lastErr)

	if c.metrics != nil {
		c.metrics.RetryCount.WithLabelValues(operation, "max_retries_exceeded").Inc()
	}

	return fmt.Errorf("max retries exceeded after %v: %w", time.Since(startTime), lastErr)
}

// doRequestInternal performs the actual HTTP request
func (c *EnhancedFastMCPHTTPClient) doRequestInternal(ctx context.Context, method, endpoint string, reqBody, respBody any) error {
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

// isRetryableErrorEnhanced determines if an error is retryable
func (c *EnhancedFastMCPHTTPClient) isRetryableErrorEnhanced(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for retryable status codes (429, 500, 502, 503, 504)
	for code := range EnhancedRetryableStatusCodes {
		pattern := fmt.Sprintf("HTTP %d", code)
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	// Retry on network errors and timeouts
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"timeout",
		"temporary failure",
		"service unavailable",
		"no such host",
		"network is unreachable",
		"broken pipe",
		"i/o timeout",
		"EOF",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	// Don't retry on context cancellation (this is intentional)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	return false
}

// calculateBackoffWithJitter calculates exponential backoff duration with ±10% jitter
func (c *EnhancedFastMCPHTTPClient) calculateBackoffWithJitter(attempt int) time.Duration {
	// Calculate base exponential backoff: 100ms * 2^attempt
	delay := time.Duration(float64(enhancedInitialRetryDelay) * math.Pow(enhancedRetryMultiplier, float64(attempt)))

	// Cap at maximum delay (30 seconds)
	if delay > enhancedMaxRetryDelay {
		delay = enhancedMaxRetryDelay
	}

	// Add jitter: ±10% random variance
	jitterRange := float64(delay) * enhancedJitterPercent
	jitter := (c.rng.Float64()*2 - 1) * jitterRange // Random value between -jitterRange and +jitterRange

	finalDelay := time.Duration(float64(delay) + jitter)

	// Ensure delay is never negative or too small
	if finalDelay < enhancedInitialRetryDelay {
		finalDelay = enhancedInitialRetryDelay
	}

	return finalDelay
}

// storeToDLQ stores a failed operation to the dead letter queue
func (c *EnhancedFastMCPHTTPClient) storeToDLQ(ctx context.Context, operation, clientID, endpoint string, reqBody any, err error) {
	if c.dlqStore == nil {
		log.Printf("[FastMCP Enhanced] No DLQ configured, skipping storage for failed %s operation", operation)
		return
	}

	// Convert request body to map
	reqBodyMap := make(map[string]interface{})
	if reqBody != nil {
		data, jsonErr := json.Marshal(reqBody)
		if jsonErr == nil {
			_ = json.Unmarshal(data, &reqBodyMap)
		}
	}

	failedOp := FailedOperation{
		ID:          fmt.Sprintf("%s-%s-%d", operation, clientID, time.Now().UnixNano()),
		ClientID:    clientID,
		Operation:   operation,
		Endpoint:    endpoint,
		RequestBody: reqBodyMap,
		LastError:   err.Error(),
		RetryCount:  enhancedMaxRetries,
		CreatedAt:   time.Now(),
		LastAttempt: time.Now(),
	}

	if storeErr := c.dlqStore.Store(ctx, failedOp); storeErr != nil {
		log.Printf("[FastMCP Enhanced] Failed to store operation in DLQ: %v", storeErr)
	} else {
		log.Printf("[FastMCP Enhanced] Stored failed operation in DLQ: %s (operation: %s, error: %s)",
			failedOp.ID, operation, err.Error())
		if c.metrics != nil {
			c.metrics.DLQOperations.Inc()
		}
	}
}

// GetMetrics returns the metrics instance for external monitoring
func (c *EnhancedFastMCPHTTPClient) GetMetrics() *MCPMetrics {
	return c.metrics
}

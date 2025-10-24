package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/resilience"
)

// Example: Using circuit breaker-protected MCP operations
func ExampleMCPWithCircuitBreaker(handler *MCPHandler) {
	ctx := context.Background()

	// Example 1: Connect to MCP server with circuit breaker protection
	config := mcp.MCPConfig{
		ID:       "example-mcp-1",
		Name:     "Example MCP Server",
		Type:     "http",
		Endpoint: "https://mcp.example.com",
		AuthType: "bearer",
		Auth: map[string]string{
			"token": "your-auth-token",
		},
	}

	// Connect with circuit breaker protection
	ctx1, cancel1 := context.WithTimeout(ctx, 30*time.Second)
	defer cancel1()

	err := handler.ConnectMCPWithBreaker(ctx1, config)
	if err != nil {
		switch err {
		case resilience.ErrCircuitOpen:
			log.Println("Circuit breaker is open - MCP service is unavailable")
			// Handle degraded service - maybe use cached data
			return
		case resilience.ErrTooManyRequests:
			log.Println("Too many requests - circuit breaker is recovering")
			// Implement backoff and retry
			time.Sleep(5 * time.Second)
			return
		default:
			log.Printf("Failed to connect: %v", err)
			return
		}
	}

	log.Println("Successfully connected to MCP server")

	// Example 2: List tools with circuit breaker protection
	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Second)
	defer cancel2()

	tools, err := handler.ListToolsWithBreaker(ctx2, "example-mcp-1")
	if err != nil {
		if err == resilience.ErrCircuitOpen {
			log.Println("Cannot list tools - circuit is open")
			// Use cached tool list if available
			return
		}
		log.Printf("Failed to list tools: %v", err)
		return
	}

	log.Printf("Found %d tools", len(tools))
	for _, tool := range tools {
		log.Printf("  - %s: %s", tool.Name, tool.Description)
	}

	// Example 3: Call a tool with circuit breaker protection
	ctx3, cancel3 := context.WithTimeout(ctx, 20*time.Second)
	defer cancel3()

	result, err := handler.CallToolWithBreaker(ctx3, "example-mcp-1", "example_tool", map[string]any{
		"param1": "value1",
		"param2": 123,
	})

	if err != nil {
		if err == resilience.ErrCircuitOpen || err == resilience.ErrTooManyRequests {
			log.Println("Circuit breaker prevented tool call")
			// Provide fallback behavior
			return
		}
		log.Printf("Failed to call tool: %v", err)
		return
	}

	log.Printf("Tool result: %v", result)

	// Example 4: Check circuit breaker health
	health := handler.HealthCheck()
	log.Printf("MCP Handler Health: %v", health["status"])
	log.Printf("Circuit Breaker States: %v", health["circuit_breakers"])

	// Example 5: Disconnect with circuit breaker protection
	ctx4, cancel4 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel4()

	err = handler.DisconnectMCPWithBreaker(ctx4, "example-mcp-1")
	if err != nil {
		log.Printf("Warning: Failed to disconnect cleanly: %v", err)
		// Continue anyway - disconnect errors are usually not critical
	}

	log.Println("Successfully disconnected from MCP server")
}

// Example: Monitoring circuit breaker metrics
func ExampleMonitorCircuitBreakers(handler *MCPHandler) {
	// Get all circuit breaker states
	states := handler.GetCircuitBreakerState()
	for operation, state := range states {
		log.Printf("Operation %s: %s", operation, state)
	}

	// Get detailed statistics
	stats := handler.GetCircuitBreakerStats()
	for operation, stat := range stats {
		log.Printf("Operation: %s", operation)
		log.Printf("  State: %s", stat.State.String())
		log.Printf("  Total Requests: %d", stat.TotalRequests)
		log.Printf("  Successes: %d", stat.TotalSuccesses)
		log.Printf("  Failures: %d", stat.TotalFailures)
		log.Printf("  Consecutive Failures: %d", stat.ConsecutiveFailures)
		if !stat.LastErrorTime.IsZero() {
			log.Printf("  Last Error: %v at %s", stat.LastError, stat.LastErrorTime)
		}
	}
}

// Example: Implementing retry logic with circuit breaker
func ExampleRetryWithCircuitBreaker(handler *MCPHandler, config mcp.MCPConfig) error {
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := handler.ConnectMCPWithBreaker(ctx, config)
		if err == nil {
			return nil // Success
		}

		// Don't retry if circuit is open
		if err == resilience.ErrCircuitOpen {
			return fmt.Errorf("circuit breaker open, aborting retry: %w", err)
		}

		// For other errors, implement exponential backoff
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			log.Printf("Attempt %d failed, retrying in %s: %v", attempt+1, delay, err)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// Example: Handling graceful degradation
func ExampleGracefulDegradation(handler *MCPHandler, mcpID string) ([]mcp.Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get tools from MCP server
	tools, err := handler.ListToolsWithBreaker(ctx, mcpID)

	// If circuit is open, return cached tools
	if err == resilience.ErrCircuitOpen {
		log.Println("Circuit breaker open, using cached tools")
		// In a real implementation, you would fetch from cache
		cachedTools := getCachedTools(mcpID)
		return cachedTools, nil
	}

	// If too many requests, implement backoff
	if err == resilience.ErrTooManyRequests {
		log.Println("Too many requests, waiting before retry")
		time.Sleep(5 * time.Second)

		// Retry once
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		return handler.ListToolsWithBreaker(ctx2, mcpID)
	}

	// For other errors, fall back to cache if available
	if err != nil {
		log.Printf("Error listing tools: %v, attempting fallback", err)
		cachedTools := getCachedTools(mcpID)
		if cachedTools != nil {
			return cachedTools, nil
		}
		return nil, err
	}

	// Success - update cache
	updateToolsCache(mcpID, tools)
	return tools, nil
}

// Example: Admin operations
func ExampleAdminOperations(handler *MCPHandler) {
	// Check health before maintenance
	health := handler.HealthCheck()
	if health["status"] != "healthy" {
		log.Printf("Warning: System is degraded: %v", health)
	}

	// During maintenance, you might want to reset circuit breakers
	log.Println("Resetting all circuit breakers for maintenance")
	handler.ResetCircuitBreakers()

	// Verify all circuits are closed
	states := handler.GetCircuitBreakerState()
	allClosed := true
	for operation, state := range states {
		if state != "closed" {
			log.Printf("Warning: %s circuit is %s", operation, state)
			allClosed = false
		}
	}

	if allClosed {
		log.Println("All circuit breakers are closed and ready")
	}
}

// Helper functions (placeholder implementations)
func getCachedTools(mcpID string) []mcp.Tool {
	// TODO: Implement actual cache lookup
	log.Printf("Looking up cached tools for MCP: %s", mcpID)
	return nil
}

func updateToolsCache(mcpID string, tools []mcp.Tool) {
	// TODO: Implement cache update
	log.Printf("Updating cache with %d tools for MCP: %s", len(tools), mcpID)
}

// Example: Testing circuit breaker behavior
func ExampleTestCircuitBreaker(handler *MCPHandler) {
	config := mcp.MCPConfig{
		ID:       "test-mcp",
		Name:     "Test MCP",
		Type:     "http",
		Endpoint: "https://failing-server.example.com", // This will fail
		AuthType: "none",
	}

	// Make requests until circuit opens
	log.Println("Testing circuit breaker failure threshold...")
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := handler.ConnectMCPWithBreaker(ctx, config)
		cancel()

		if err == resilience.ErrCircuitOpen {
			log.Printf("Circuit opened after %d attempts", i+1)
			break
		}

		log.Printf("Attempt %d: %v", i+1, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Check state
	state := handler.GetCircuitBreakerState()["connect"]
	log.Printf("Circuit breaker state: %s", state)

	// Wait for timeout and test half-open state
	log.Println("Waiting for circuit breaker timeout...")
	time.Sleep(31 * time.Second)

	state = handler.GetCircuitBreakerState()["connect"]
	log.Printf("Circuit breaker state after timeout: %s", state)

	// The next request will transition to half-open
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	handler.ConnectMCPWithBreaker(ctx, config)

	state = handler.GetCircuitBreakerState()["connect"]
	log.Printf("Circuit breaker state after request: %s", state)
}

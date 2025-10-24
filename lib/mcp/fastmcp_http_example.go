package mcp

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the FastMCP HTTP client
func ExampleUsage() {
	// Create a new client
	client := NewFastMCPHTTPClient("http://localhost:8000")

	// Optional: Set custom timeout
	client.SetTimeout(60 * time.Second)

	ctx := context.Background()

	// Check service health
	if err := client.Health(ctx); err != nil {
		log.Fatalf("Service is not healthy: %v", err)
	}
	fmt.Println("FastMCP service is healthy")

	// Connect to an MCP server using stdio transport
	clientID := "my-app-client"
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
		Args:      []string{"-m", "mcp.server.example"},
		Env: map[string]string{
			"DEBUG": "true",
		},
	}

	if err := client.Connect(ctx, clientID, config); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected to MCP server")

	// List available tools
	tools, err := client.ListTools(ctx, clientID)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Call a tool
	result, err := client.CallTool(ctx, clientID, "echo", map[string]any{
		"message": "Hello, FastMCP!",
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Printf("Tool result: %v\n", result)

	// Disconnect when done
	if err := client.Disconnect(ctx, clientID); err != nil {
		log.Fatalf("Failed to disconnect: %v", err)
	}
	fmt.Println("Disconnected from MCP server")
}

// ExampleSSETransport demonstrates connecting via Server-Sent Events
func ExampleSSETransport() {
	client := NewFastMCPHTTPClient("http://localhost:8000")
	ctx := context.Background()

	clientID := "sse-client"
	config := HTTPMCPConfig{
		Transport: "sse",
		MCPURL:    "https://example.com/mcp/sse",
	}

	if err := client.Connect(ctx, clientID, config); err != nil {
		log.Fatalf("Failed to connect via SSE: %v", err)
	}
	defer client.Disconnect(ctx, clientID)

	// Use the connection...
	tools, _ := client.ListTools(ctx, clientID)
	fmt.Printf("SSE connection established, %d tools available\n", len(tools))
}

// ExampleHTTPTransport demonstrates connecting via HTTP
func ExampleHTTPTransport() {
	client := NewFastMCPHTTPClient("http://localhost:8000")
	ctx := context.Background()

	clientID := "http-client"
	config := HTTPMCPConfig{
		Transport: "http",
		MCPURL:    "https://api.example.com/mcp",
	}

	if err := client.Connect(ctx, clientID, config); err != nil {
		log.Fatalf("Failed to connect via HTTP: %v", err)
	}
	defer client.Disconnect(ctx, clientID)

	// Use the connection...
	tools, _ := client.ListTools(ctx, clientID)
	fmt.Printf("HTTP connection established, %d tools available\n", len(tools))
}

// ExampleOAuthTransport demonstrates connecting with OAuth authentication
func ExampleOAuthTransport() {
	client := NewFastMCPHTTPClient("http://localhost:8000")
	ctx := context.Background()

	clientID := "oauth-client"
	config := HTTPMCPConfig{
		Transport:     "http",
		MCPURL:        "https://api.example.com/mcp",
		OAuthProvider: "google",
	}

	if err := client.Connect(ctx, clientID, config); err != nil {
		log.Fatalf("Failed to connect with OAuth: %v", err)
	}
	defer client.Disconnect(ctx, clientID)

	// Use the connection...
	fmt.Println("OAuth connection established")
}

// ExampleWithContextTimeout demonstrates using context timeouts
func ExampleWithContextTimeout() {
	client := NewFastMCPHTTPClient("http://localhost:8000")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientID := "timeout-client"
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
		Args:      []string{"-m", "mcp.server.slow"},
	}

	// This will fail if the connection takes more than 5 seconds
	if err := client.Connect(ctx, clientID, config); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Connection timed out after 5 seconds")
		} else {
			log.Printf("Connection failed: %v", err)
		}
		return
	}

	fmt.Println("Connection succeeded within timeout")
	defer client.Disconnect(context.Background(), clientID)
}

// ExampleErrorHandling demonstrates comprehensive error handling
func ExampleErrorHandling() {
	client := NewFastMCPHTTPClient("http://localhost:8000")
	ctx := context.Background()

	// Check if service is available
	if err := client.Health(ctx); err != nil {
		log.Printf("WARNING: FastMCP service is not available: %v", err)
		log.Printf("Make sure the service is running on http://localhost:8000")
		return
	}

	clientID := "error-handling-client"
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "invalid-command",
	}

	// Try to connect
	if err := client.Connect(ctx, clientID, config); err != nil {
		log.Printf("Expected error occurred: %v", err)
		// Handle the error appropriately
		// - Retry with backoff
		// - Use a fallback configuration
		// - Alert the user
		return
	}

	// Try to call a non-existent tool
	_, err := client.CallTool(ctx, clientID, "non-existent-tool", nil)
	if err != nil {
		log.Printf("Tool call failed as expected: %v", err)
	}

	// Always disconnect, even if there were errors
	client.Disconnect(ctx, clientID)
}

// ExampleMultipleClients demonstrates managing multiple MCP connections
func ExampleMultipleClients() {
	httpClient := NewFastMCPHTTPClient("http://localhost:8000")
	ctx := context.Background()

	// Connect to multiple MCP servers
	clients := []struct {
		id     string
		config HTTPMCPConfig
	}{
		{
			id: "weather-mcp",
			config: HTTPMCPConfig{
				Transport: "stdio",
				Command:   "python",
				Args:      []string{"-m", "mcp.server.weather"},
			},
		},
		{
			id: "database-mcp",
			config: HTTPMCPConfig{
				Transport: "stdio",
				Command:   "python",
				Args:      []string{"-m", "mcp.server.database"},
			},
		},
		{
			id: "api-mcp",
			config: HTTPMCPConfig{
				Transport: "sse",
				MCPURL:    "https://api.example.com/mcp",
			},
		},
	}

	// Connect to all servers
	for _, c := range clients {
		if err := httpClient.Connect(ctx, c.id, c.config); err != nil {
			log.Printf("Failed to connect to %s: %v", c.id, err)
			continue
		}
		fmt.Printf("Connected to %s\n", c.id)
		defer httpClient.Disconnect(ctx, c.id)
	}

	// Use different clients for different purposes
	weatherTools, _ := httpClient.ListTools(ctx, "weather-mcp")
	dbTools, _ := httpClient.ListTools(ctx, "database-mcp")
	apiTools, _ := httpClient.ListTools(ctx, "api-mcp")

	fmt.Printf("Weather MCP: %d tools\n", len(weatherTools))
	fmt.Printf("Database MCP: %d tools\n", len(dbTools))
	fmt.Printf("API MCP: %d tools\n", len(apiTools))

	// Call tools on specific clients
	result, err := httpClient.CallTool(ctx, "weather-mcp", "get_forecast", map[string]any{
		"city": "San Francisco",
	})
	if err != nil {
		log.Printf("Failed to get weather: %v", err)
	} else {
		fmt.Printf("Weather forecast: %v\n", result)
	}
}

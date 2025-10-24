package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coder/agentapi/lib/logging"
	"github.com/redis/go-redis/v9"
)

// ExampleMCPBasicUsage demonstrates basic MCP state management
func ExampleMCPBasicUsage() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create logger
	logger := logging.GetLogger("mcp_state_example")
	logger.SetLevel(logging.INFO)

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
		Logger:      logger,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Register an MCP client
	sessionID := "user-session-123"
	mcpName := "filesystem-mcp"
	endpoint := "http://localhost:3000/mcp"

	err = manager.RegisterMCPClient(ctx, sessionID, mcpName, endpoint)
	if err != nil {
		log.Fatalf("Failed to register MCP client: %v", err)
	}
	fmt.Printf("Registered MCP client: %s\n", mcpName)

	// Get client state
	state, err := manager.GetMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		log.Fatalf("Failed to get MCP client: %v", err)
	}
	fmt.Printf("Client status: %s, endpoint: %s\n", state.Status, state.Endpoint)

	// Update tool count
	err = manager.UpdateToolCount(ctx, sessionID, mcpName, 10)
	if err != nil {
		log.Fatalf("Failed to update tool count: %v", err)
	}
	fmt.Println("Updated tool count")

	// Mark as active
	err = manager.MarkActive(ctx, sessionID, mcpName)
	if err != nil {
		log.Fatalf("Failed to mark active: %v", err)
	}
	fmt.Println("Marked client as active")

	// Unregister when done
	err = manager.UnregisterMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		log.Fatalf("Failed to unregister MCP client: %v", err)
	}
	fmt.Println("Unregistered MCP client")
}

// ExampleMCPMultipleClients demonstrates managing multiple MCP clients
func ExampleMCPMultipleClients() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	sessionID := "user-session-456"

	// Register multiple MCP clients
	mcpClients := []struct {
		name     string
		endpoint string
	}{
		{"filesystem-mcp", "http://localhost:3000/mcp"},
		{"database-mcp", "http://localhost:3001/mcp"},
		{"api-mcp", "http://localhost:3002/mcp"},
	}

	for _, client := range mcpClients {
		err := manager.RegisterMCPClient(ctx, sessionID, client.name, client.endpoint)
		if err != nil {
			log.Fatalf("Failed to register %s: %v", client.name, err)
		}
		fmt.Printf("Registered: %s\n", client.name)
	}

	// List all clients for the session
	clients, err := manager.ListMCPClients(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to list MCP clients: %v", err)
	}

	fmt.Printf("\nTotal clients for session: %d\n", len(clients))
	for _, client := range clients {
		fmt.Printf("  - %s: %s (status: %s, tools: %d)\n",
			client.MCPName, client.Endpoint, client.Status, client.ToolCount)
	}

	// Clean up
	for _, client := range mcpClients {
		err := manager.UnregisterMCPClient(ctx, sessionID, client.name)
		if err != nil {
			log.Printf("Failed to unregister %s: %v", client.name, err)
		}
	}
}

// ExampleMCPWithMetadata demonstrates using custom metadata
func ExampleMCPWithMetadata() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	sessionID := "user-session-789"
	mcpName := "custom-mcp"

	// Register client
	err = manager.RegisterMCPClient(ctx, sessionID, mcpName, "http://localhost:3000/mcp")
	if err != nil {
		log.Fatalf("Failed to register MCP client: %v", err)
	}

	// Set custom metadata
	metadata := map[string]interface{}{
		"version":      "2.0.0",
		"environment":  "production",
		"region":       "us-west-2",
		"capabilities": []string{"tools", "resources", "prompts"},
	}

	err = manager.SetMetadata(ctx, sessionID, mcpName, metadata)
	if err != nil {
		log.Fatalf("Failed to set metadata: %v", err)
	}
	fmt.Println("Set custom metadata")

	// Retrieve and display metadata
	state, err := manager.GetMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		log.Fatalf("Failed to get MCP client: %v", err)
	}

	fmt.Println("\nClient metadata:")
	for key, value := range state.Metadata {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Clean up
	manager.UnregisterMCPClient(ctx, sessionID, mcpName)
}

// ExampleMCPCleanup demonstrates cleanup of expired clients
func ExampleMCPCleanup() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Register some clients
	for i := 0; i < 5; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		mcpName := fmt.Sprintf("mcp-%d", i)
		err := manager.RegisterMCPClient(ctx, sessionID, mcpName, "http://localhost:3000")
		if err != nil {
			log.Printf("Failed to register client: %v", err)
		}
	}

	fmt.Println("Registered 5 MCP clients")

	// Run cleanup (removes clients inactive for more than 7 days)
	err = manager.CleanupExpiredClients(ctx, 7*24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to cleanup expired clients: %v", err)
	}

	fmt.Println("Cleanup completed")

	// Get metrics
	metrics := manager.GetMetrics()
	fmt.Println("\nMetrics:")
	for key, value := range metrics {
		fmt.Printf("  %s: %d\n", key, value)
	}
}

// ExampleMCPMonitoring demonstrates monitoring and metrics
func ExampleMCPMonitoring() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create logger with debug level
	logger := logging.GetLogger("mcp_state_monitor")
	logger.SetLevel(logging.DEBUG)

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
		Logger:      logger,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Simulate some operations
	sessionID := "monitoring-session"

	for i := 0; i < 10; i++ {
		mcpName := fmt.Sprintf("mcp-%d", i)
		err := manager.RegisterMCPClient(ctx, sessionID, mcpName, "http://localhost:3000")
		if err != nil {
			log.Printf("Failed to register: %v", err)
		}

		// Simulate activity
		err = manager.MarkActive(ctx, sessionID, mcpName)
		if err != nil {
			log.Printf("Failed to mark active: %v", err)
		}

		err = manager.UpdateToolCount(ctx, sessionID, mcpName, i*5)
		if err != nil {
			log.Printf("Failed to update tool count: %v", err)
		}
	}

	// List all clients
	clients, err := manager.ListMCPClients(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to list clients: %v", err)
	}

	fmt.Printf("Total clients: %d\n", len(clients))

	// Get and display metrics
	metrics := manager.GetMetrics()
	fmt.Println("\nOperational Metrics:")
	fmt.Printf("  Registrations:   %d\n", metrics["registrations"])
	fmt.Printf("  Unregistrations: %d\n", metrics["unregistrations"])
	fmt.Printf("  Gets:            %d\n", metrics["gets"])
	fmt.Printf("  Lists:           %d\n", metrics["lists"])
	fmt.Printf("  Errors:          %d\n", metrics["errors"])
	fmt.Printf("  Reconnections:   %d\n", metrics["reconnections"])

	// Clean up
	for i := 0; i < 10; i++ {
		mcpName := fmt.Sprintf("mcp-%d", i)
		manager.UnregisterMCPClient(ctx, sessionID, mcpName)
	}
}

// ExampleMCPRecovery demonstrates connection recovery
func ExampleMCPRecovery() {
	// Create Redis client with connection pool settings
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})
	defer redisClient.Close()

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "mcp-1", "http://localhost:3000")
	if err != nil {
		log.Fatalf("Failed to register: %v", err)
	}
	fmt.Println("Registered MCP client")

	// Simulate connection issues and recovery
	// In a real scenario, this would happen automatically on network issues
	fmt.Println("\nAttempting connection recovery...")
	err = manager.RecoverConnection(ctx)
	if err != nil {
		log.Printf("Recovery failed: %v", err)
	} else {
		fmt.Println("Connection recovered successfully")
	}

	// Verify operations still work
	state, err := manager.GetMCPClient(ctx, "session-1", "mcp-1")
	if err != nil {
		log.Fatalf("Failed to get client after recovery: %v", err)
	}
	fmt.Printf("Client state after recovery: %s\n", state.Status)

	// Clean up
	manager.UnregisterMCPClient(ctx, "session-1", "mcp-1")
}

// ExampleMCPBackgroundCleanup demonstrates running cleanup in the background
func ExampleMCPBackgroundCleanup() {
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Create MCP state manager
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: redisClient,
		TTL:         24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP state manager: %v", err)
	}
	defer manager.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Stopping background cleanup")
				return
			case <-ticker.C:
				fmt.Println("Running periodic cleanup...")
				err := manager.CleanupExpiredClients(ctx, 7*24*time.Hour)
				if err != nil {
					log.Printf("Cleanup error: %v", err)
				} else {
					fmt.Println("Cleanup completed")
				}
			}
		}
	}()

	// Register some clients
	for i := 0; i < 3; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		mcpName := fmt.Sprintf("mcp-%d", i)
		err := manager.RegisterMCPClient(ctx, sessionID, mcpName, "http://localhost:3000")
		if err != nil {
			log.Printf("Failed to register: %v", err)
		}
	}

	fmt.Println("Background cleanup is running...")
	fmt.Println("Registered 3 MCP clients")

	// In a real application, this would run indefinitely
	// Here we'll just demonstrate for a few seconds
	time.Sleep(3 * time.Second)

	// Clean up
	for i := 0; i < 3; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		mcpName := fmt.Sprintf("mcp-%d", i)
		manager.UnregisterMCPClient(ctx, sessionID, mcpName)
	}
}

package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// ExampleEnhancedClient demonstrates the usage of the enhanced FastMCP HTTP client
// with retry logic, DLQ, and metrics
func ExampleEnhancedClient() {
	// 1. Create Redis client for DLQ
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available, DLQ disabled: %v", err)
		redisClient = nil
	}

	// 2. Create DLQ (optional)
	var dlq *RedisDLQ
	if redisClient != nil {
		dlq = NewRedisDLQ(redisClient)
		log.Println("DLQ enabled with Redis")
	}

	// 3. Initialize custom metrics
	metrics := InitMCPMetrics("example_app")

	// 4. Create enhanced client
	client := NewEnhancedFastMCPHTTPClientWithOptions(
		"http://localhost:8000",
		dlq,
		metrics,
	)

	// 5. Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 6. Start DLQ cleanup routine (if enabled)
	if dlq != nil {
		go runDLQCleanup(dlq)
		go monitorDLQ(dlq)
	}

	// 7. Example operations
	demoOperations(client)
}

// demoOperations demonstrates various client operations
func demoOperations(client *EnhancedFastMCPHTTPClient) {
	ctx := context.Background()

	// Example 1: Connect with retry
	log.Println("\n=== Example 1: Connect with retry ===")
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-everything"},
	}

	if err := client.ConnectWithRetry(ctx, "demo-client-1", config); err != nil {
		log.Printf("Connect failed: %v", err)
	} else {
		log.Println("Successfully connected")
	}

	// Example 2: List tools with retry
	log.Println("\n=== Example 2: List tools with retry ===")
	tools, err := client.ListToolsWithRetry(ctx, "demo-client-1")
	if err != nil {
		log.Printf("List tools failed: %v", err)
	} else {
		log.Printf("Found %d tools", len(tools))
		for i, tool := range tools {
			log.Printf("  %d. %s: %s", i+1, tool.Name, tool.Description)
		}
	}

	// Example 3: Call tool with retry
	log.Println("\n=== Example 3: Call tool with retry ===")
	result, err := client.CallToolWithRetry(ctx, "demo-client-1", "echo", map[string]any{
		"message": "Hello from enhanced client!",
	})
	if err != nil {
		log.Printf("Tool call failed: %v", err)
	} else {
		log.Printf("Tool result: %+v", result)
	}

	// Example 4: Call tool with timeout
	log.Println("\n=== Example 4: Call tool with timeout ===")
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err = client.CallToolWithRetry(ctxTimeout, "demo-client-1", "slow_operation", nil)
	if err != nil {
		if ctxTimeout.Err() == context.DeadlineExceeded {
			log.Println("Operation timed out")
		} else {
			log.Printf("Tool call failed: %v", err)
		}
	} else {
		log.Printf("Tool result: %+v", result)
	}

	// Example 5: Simulate failures to populate DLQ
	log.Println("\n=== Example 5: Simulate failures ===")
	for i := 0; i < 3; i++ {
		_, err := client.CallToolWithRetry(ctx, "demo-client-1", "nonexistent_tool", nil)
		if err != nil {
			log.Printf("Expected failure %d: %v", i+1, err)
		}
	}
}

// runDLQCleanup periodically cleans up old DLQ entries
func runDLQCleanup(dlq *RedisDLQ) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("DLQ cleanup routine started (runs every hour)")

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Clean up entries older than 7 days
		if err := dlq.Cleanup(ctx, 7*24*time.Hour); err != nil {
			log.Printf("DLQ cleanup error: %v", err)
		} else {
			log.Println("DLQ cleanup completed successfully")
		}

		cancel()
	}
}

// monitorDLQ periodically monitors and reports DLQ statistics
func monitorDLQ(dlq *RedisDLQ) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("DLQ monitor started (reports every 5 minutes)")

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		stats, err := dlq.GetStats(ctx)
		if err != nil {
			log.Printf("Failed to get DLQ stats: %v", err)
			cancel()
			continue
		}

		log.Printf("\n=== DLQ Statistics ===")
		log.Printf("Total operations: %d", stats.TotalOperations)
		log.Printf("Operations by type:")
		for opType, count := range stats.OperationCounts {
			log.Printf("  %s: %d", opType, count)
		}

		if stats.OldestOperation != nil {
			log.Printf("Oldest operation: %v", stats.OldestOperation.Format(time.RFC3339))
		}
		if stats.NewestOperation != nil {
			log.Printf("Newest operation: %v", stats.NewestOperation.Format(time.RFC3339))
		}
		log.Println("======================")

		cancel()
	}
}

// ExampleDLQManualRetry demonstrates how to manually retry failed operations from the DLQ
func ExampleDLQManualRetry() {
	// Setup
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	dlq := NewRedisDLQ(redisClient)
	client := NewEnhancedFastMCPHTTPClient("http://localhost:8000")
	client.SetDeadLetterQueue(dlq)

	ctx := context.Background()

	// Get failed operations
	failedOps, err := dlq.List(ctx, 100)
	if err != nil {
		log.Fatalf("Failed to list DLQ: %v", err)
	}

	log.Printf("Found %d failed operations in DLQ", len(failedOps))

	// Attempt to retry each operation
	for _, op := range failedOps {
		log.Printf("\nRetrying operation: %s (ID: %s)", op.Operation, op.ID)
		log.Printf("  Original error: %s", op.LastError)
		log.Printf("  Retry count: %d", op.RetryCount)
		log.Printf("  Last attempt: %v", op.LastAttempt.Format(time.RFC3339))

		// Check if operation should be retried
		if time.Since(op.LastAttempt) < 5*time.Minute {
			log.Println("  Skipping: Too recent")
			continue
		}

		// Attempt retry based on operation type
		var retryErr error
		switch op.Operation {
		case "connect":
			retryErr = retryConnect(client, op)
		case "call_tool":
			retryErr = retryCallTool(client, op)
		case "list_tools":
			retryErr = retryListTools(client, op)
		default:
			log.Printf("  Unknown operation type: %s", op.Operation)
			continue
		}

		if retryErr != nil {
			log.Printf("  Retry failed: %v", retryErr)
		} else {
			log.Println("  Retry successful! Removing from DLQ")
			if err := dlq.Delete(ctx, op.ID); err != nil {
				log.Printf("  Failed to delete from DLQ: %v", err)
			}
		}
	}
}

// retryConnect retries a failed connect operation
func retryConnect(client *EnhancedFastMCPHTTPClient, op FailedOperation) error {
	// Reconstruct the config from request body
	var req ConnectRequest
	data, _ := json.Marshal(op.RequestBody)
	json.Unmarshal(data, &req)

	config := HTTPMCPConfig{
		Transport:     req.Transport,
		OAuthProvider: req.OAuthProvider,
		MCPURL:        req.MCPURL,
		Command:       req.Command,
		Args:          req.Args,
		Env:           req.Env,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	return client.ConnectWithRetry(ctx, op.ClientID, config)
}

// retryCallTool retries a failed call_tool operation
func retryCallTool(client *EnhancedFastMCPHTTPClient, op FailedOperation) error {
	// Reconstruct the request from request body
	var req ToolCallRequest
	data, _ := json.Marshal(op.RequestBody)
	json.Unmarshal(data, &req)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, err := client.CallToolWithRetry(ctx, op.ClientID, req.ToolName, req.Arguments)
	return err
}

// retryListTools retries a failed list_tools operation
func retryListTools(client *EnhancedFastMCPHTTPClient, op FailedOperation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.ListToolsWithRetry(ctx, op.ClientID)
	return err
}

// ExampleMetricsIntegration demonstrates how to integrate with existing metrics
func ExampleMetricsIntegration() {
	// Create client with custom namespace
	metrics := InitMCPMetrics("production_app")
	client := NewEnhancedFastMCPHTTPClientWithOptions(
		"http://localhost:8000",
		nil,
		metrics,
	)

	// Access metrics for custom logic
	metricsInstance := client.GetMetrics()

	// Example: Monitor retry rate
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// Your custom metrics analysis here
			log.Println("Metrics available via Prometheus /metrics endpoint")
		}
	}()

	// Use the client...
	ctx := context.Background()
	client.ConnectWithRetry(ctx, "prod-client", HTTPMCPConfig{
		Transport: "stdio",
		Command:   "server-command",
	})

	// Metrics are automatically recorded
	fmt.Printf("Metrics instance: %+v\n", metricsInstance)
}

// ExampleAdvancedConfiguration demonstrates advanced configuration options
func ExampleAdvancedConfiguration() {
	// Custom Redis configuration
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "your-password",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Custom DLQ with TTL
	dlq := NewRedisDLQWithTTL(redisClient, 14*24*time.Hour) // 14 days

	// Custom metrics namespace
	metrics := InitMCPMetrics("custom_namespace")

	// Create client
	client := NewEnhancedFastMCPHTTPClientWithOptions(
		"http://custom-fastmcp-service:8000",
		dlq,
		metrics,
	)

	// Set custom timeout
	client.SetTimeout(45 * time.Second)

	// Use the client
	ctx := context.Background()
	config := HTTPMCPConfig{
		Transport: "http",
		MCPURL:    "http://mcp-server:3000",
	}

	if err := client.ConnectWithRetry(ctx, "custom-client", config); err != nil {
		log.Printf("Error: %v", err)
	}
}

package redis_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coder/agentapi/lib/redis"
)

// Example demonstrates complete Redis client usage with environment configuration
func Example() {
	// Load configuration from environment
	config := redis.DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.Token = os.Getenv("REDIS_TOKEN")

	// Create client
	client, err := redis.NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Perform health check
	if err := client.Health(); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}

	fmt.Printf("Connected using protocol: %s\n", client.GetActiveProtocol())

	ctx := context.Background()

	// Set a value
	err = client.Set(ctx, "example:key", "example-value", 5*time.Minute)
	if err != nil {
		log.Fatalf("Failed to set value: %v", err)
	}

	// Get the value
	value, err := client.Get(ctx, "example:key")
	if err != nil {
		log.Fatalf("Failed to get value: %v", err)
	}

	fmt.Printf("Retrieved value: %s\n", value)

	// Check if key exists
	exists, err := client.Exists(ctx, "example:key")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}

	fmt.Printf("Key exists: %v\n", exists)
}

// ExampleRedisClient_caching shows how to use Redis for caching
func ExampleRedisClient_caching() {
	config := redis.DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.Token = os.Getenv("REDIS_TOKEN")

	client, err := redis.NewRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Cache key
	cacheKey := "cache:product:123"

	// Try to get from cache
	cached, err := client.Get(ctx, cacheKey)
	if err != nil {
		log.Fatal(err)
	}

	if cached != "" {
		fmt.Println("Cache hit!")
	} else {
		fmt.Println("Cache miss, storing data...")
		// Store with 10 minute TTL
		productData := `{"id": 123, "name": "Widget", "price": 19.99}`
		client.Set(ctx, cacheKey, productData, 10*time.Minute)
	}
}

// ExampleRedisClient_sessionManagement shows session management with Redis
func ExampleRedisClient_sessionManagement() {
	config := redis.DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.Token = os.Getenv("REDIS_TOKEN")

	client, err := redis.NewRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create session
	sessionID := "session:user123:abc"
	sessionData := `{"user_id": "user123", "created_at": "2025-10-23T12:00:00Z"}`

	// Store session with 1 hour TTL
	err = client.Set(ctx, sessionID, sessionData, 1*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Session created")

	// Validate session
	exists, err := client.Exists(ctx, sessionID)
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		fmt.Println("Session is valid")
	}

	// Invalidate session
	err = client.Delete(ctx, sessionID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Session invalidated")
}

// ExampleRedisClient_rateLimiting demonstrates rate limiting with Redis
func ExampleRedisClient_rateLimiting() {
	config := redis.DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.Token = os.Getenv("REDIS_TOKEN")

	client, err := redis.NewRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Rate limit: 100 requests per minute
	userID := "user:789"
	timestamp := time.Now().Format("2006-01-02-15:04")
	key := fmt.Sprintf("ratelimit:%s:%s", userID, timestamp)

	// Increment counter
	err = client.Increment(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	// Get current count
	countStr, err := client.Get(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	fmt.Sscanf(countStr, "%d", &count)

	// Set TTL if first request
	if count == 1 {
		client.Set(ctx, key, countStr, 1*time.Minute)
	}

	maxRequests := 100
	if count > maxRequests {
		fmt.Println("Rate limit exceeded")
	} else {
		fmt.Printf("Request allowed (%d/%d)\n", count, maxRequests)
	}
}

// ExampleHealthCheck_integration demonstrates health check integration
func ExampleHealthCheck_integration() {
	config := redis.DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.Token = os.Getenv("REDIS_TOKEN")

	client, err := redis.NewRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create health check
	healthCheck := redis.NewHealthCheck(client)

	// Perform check
	ctx := context.Background()
	if err := healthCheck.Check(ctx); err != nil {
		fmt.Printf("Redis is unhealthy: %v\n", err)
		return
	}

	// Get detailed status
	status, err := healthCheck.GetStatus(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Redis Status: available=%v, protocol=%v\n",
		status["available"], status["protocol"])
}

package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// ExampleBasicUsage demonstrates basic Redis client operations
func ExampleBasicUsage() {
	// Create client configuration
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")

	// Initialize client
	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Health check
	if err := client.Health(); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}

	fmt.Printf("Connected using protocol: %s\n", client.GetActiveProtocol())

	// Set a value
	if err := client.Set(ctx, "user:1:name", "John Doe", 0); err != nil {
		log.Fatalf("Failed to set value: %v", err)
	}

	// Get a value
	name, err := client.Get(ctx, "user:1:name")
	if err != nil {
		log.Fatalf("Failed to get value: %v", err)
	}
	fmt.Printf("User name: %s\n", name)

	// Check if key exists
	exists, err := client.Exists(ctx, "user:1:name")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("Key exists: %v\n", exists)

	// Delete key
	if err := client.Delete(ctx, "user:1:name"); err != nil {
		log.Fatalf("Failed to delete key: %v", err)
	}
}

// ExampleSessionManagement demonstrates using Redis for session management
func ExampleSessionManagement() {
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create session with 1 hour TTL
	sessionID := "session:abc123"
	sessionData := `{"user_id": "123", "email": "user@example.com"}`

	if err := client.Set(ctx, sessionID, sessionData, 1*time.Hour); err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	fmt.Println("Session created")

	// Retrieve session
	data, err := client.Get(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}
	fmt.Printf("Session data: %s\n", data)

	// Check if session exists
	exists, err := client.Exists(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to check session: %v", err)
	}
	fmt.Printf("Session active: %v\n", exists)

	// Invalidate session
	if err := client.Delete(ctx, sessionID); err != nil {
		log.Fatalf("Failed to delete session: %v", err)
	}
	fmt.Println("Session invalidated")
}

// ExampleRateLimiting demonstrates using Redis for rate limiting
func ExampleRateLimiting() {
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Rate limiting: 10 requests per minute per user
	userID := "user:456"
	rateLimitKey := fmt.Sprintf("ratelimit:%s:%s", userID, time.Now().Format("2006-01-02-15:04"))
	maxRequests := 10

	// Increment request counter
	if err := client.Increment(ctx, rateLimitKey); err != nil {
		log.Fatalf("Failed to increment counter: %v", err)
	}

	// Set TTL on first request
	exists, _ := client.Exists(ctx, rateLimitKey)
	if !exists {
		if err := client.Set(ctx, rateLimitKey, "1", 1*time.Minute); err != nil {
			log.Fatalf("Failed to set TTL: %v", err)
		}
	}

	// Check request count
	countStr, err := client.Get(ctx, rateLimitKey)
	if err != nil {
		log.Fatalf("Failed to get count: %v", err)
	}

	var count int
	fmt.Sscanf(countStr, "%d", &count)

	if count > maxRequests {
		fmt.Println("Rate limit exceeded")
	} else {
		fmt.Printf("Request allowed (%d/%d)\n", count, maxRequests)
	}
}

// ExampleCaching demonstrates using Redis as a cache
func ExampleCaching() {
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Cache key
	cacheKey := "api:response:users:123"

	// Try to get from cache
	cached, err := client.Get(ctx, cacheKey)
	if err != nil {
		log.Fatalf("Failed to get from cache: %v", err)
	}

	if cached != "" {
		fmt.Println("Cache hit!")
		fmt.Printf("Cached data: %s\n", cached)
	} else {
		fmt.Println("Cache miss, fetching from database...")

		// Simulate database fetch
		data := `{"id": 123, "name": "Jane Smith", "email": "jane@example.com"}`

		// Store in cache with 5 minute TTL
		if err := client.Set(ctx, cacheKey, data, 5*time.Minute); err != nil {
			log.Fatalf("Failed to cache data: %v", err)
		}
		fmt.Println("Data cached successfully")
	}
}

// ExampleHealthCheck demonstrates health checking integration
func ExampleHealthCheck() {
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Perform health check
	if err := client.Health(); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		fmt.Println("Status: UNHEALTHY")
		return
	}

	fmt.Println("Status: HEALTHY")
	fmt.Printf("Active protocol: %s\n", client.GetActiveProtocol())
}

// ExampleFallbackScenario demonstrates automatic protocol fallback
func ExampleFallbackScenario() {
	// Configure with both native and REST
	config := DefaultConfig()
	config.URL = os.Getenv("REDIS_URL")
	config.Token = os.Getenv("REDIS_TOKEN")
	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	config.PreferredProtocol = ProtocolNative

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Printf("Initial protocol: %s\n", client.GetActiveProtocol())

	// Perform operations - client will automatically fallback if native fails
	if err := client.Set(ctx, "test:key", "test:value", 1*time.Minute); err != nil {
		log.Fatalf("Failed to set value: %v", err)
	}

	fmt.Printf("Active protocol after operation: %s\n", client.GetActiveProtocol())

	// Get value
	val, err := client.Get(ctx, "test:key")
	if err != nil {
		log.Fatalf("Failed to get value: %v", err)
	}
	fmt.Printf("Retrieved value: %s\n", val)
}

// ExampleWithEnvironmentVariables demonstrates loading configuration from environment
func ExampleWithEnvironmentVariables() {
	// Set up configuration from environment variables
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "rediss://default:token@localhost:6379"
	}

	restBaseURL := os.Getenv("REDIS_REST_URL")
	if restBaseURL == "" {
		restBaseURL = "https://localhost"
	}

	token := os.Getenv("REDIS_TOKEN")
	if token == "" {
		token = "your-token-here"
	}

	config := DefaultConfig()
	config.URL = redisURL
	config.RESTBaseURL = restBaseURL
	config.Token = token

	client, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	fmt.Println("Redis client initialized from environment variables")
	fmt.Printf("Protocol: %s\n", client.GetActiveProtocol())
}

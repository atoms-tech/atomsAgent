// Package redis provides a production-ready Redis client with dual-protocol support.
//
// This package offers a Redis client that supports both native Redis protocol (TCP)
// and REST API fallback, ensuring high availability and resilience.
//
// Features:
//   - Native Redis connection with TLS support (rediss://)
//   - REST API fallback for environments without direct TCP access
//   - Automatic protocol fallback on connection failures
//   - Connection pooling and retry logic with exponential backoff
//   - Health check functionality
//   - Thread-safe concurrent operations
//   - Context support for cancellation and timeouts
//
// Basic Usage:
//
//	config := redis.DefaultConfig()
//	config.URL = os.Getenv("REDIS_URL")
//	config.RESTBaseURL = os.Getenv("REDIS_REST_URL")
//	config.Token = os.Getenv("REDIS_TOKEN")
//
//	client, err := redis.NewRedisClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	ctx := context.Background()
//	err = client.Set(ctx, "key", "value", 1*time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Configuration:
//
// The client can be configured using environment variables:
//   - REDIS_URL: Native Redis connection URL (e.g., rediss://default:token@host:6379)
//   - REDIS_REST_URL: REST API base URL (e.g., https://host)
//   - REDIS_TOKEN: REST API authentication token
//
// Health Checks:
//
// The package integrates with the health check system:
//
//	healthCheck := redis.NewHealthCheck(client)
//	if err := healthCheck.Check(ctx); err != nil {
//	    log.Printf("Redis unhealthy: %v", err)
//	}
//
// For more examples, see example_integration.go
package redis

package redis

import (
	"context"
	"fmt"
)

// HealthCheck implements the health.HealthCheck interface for Redis
type HealthCheck struct {
	client *RedisClient
}

// NewHealthCheck creates a new Redis health check
func NewHealthCheck(client *RedisClient) *HealthCheck {
	return &HealthCheck{
		client: client,
	}
}

// Check performs the health check for Redis
func (hc *HealthCheck) Check(ctx context.Context) error {
	if hc.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	// Perform health check using client's Health method
	if err := hc.client.Health(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// GetStatus returns a detailed status of the Redis connection
func (hc *HealthCheck) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	if hc.client == nil {
		status["available"] = false
		status["error"] = "client is nil"
		return status, fmt.Errorf("redis client is nil")
	}

	// Perform health check
	err := hc.client.Health()
	status["available"] = err == nil
	status["protocol"] = string(hc.client.GetActiveProtocol())

	if err != nil {
		status["error"] = err.Error()
		return status, err
	}

	return status, nil
}

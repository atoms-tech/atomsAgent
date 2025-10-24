package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes for DLQ
	dlqKeyPrefix = "mcp:dlq:"
	dlqListKey   = "mcp:dlq:list"

	// Default TTL for DLQ entries (7 days)
	defaultDLQTTL = 7 * 24 * time.Hour
)

// RedisDLQ implements the DeadLetterQueue interface using Redis
type RedisDLQ struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisDLQ creates a new Redis-based Dead Letter Queue
func NewRedisDLQ(client *redis.Client) *RedisDLQ {
	return &RedisDLQ{
		client: client,
		ttl:    defaultDLQTTL,
	}
}

// NewRedisDLQWithTTL creates a new Redis-based Dead Letter Queue with custom TTL
func NewRedisDLQWithTTL(client *redis.Client, ttl time.Duration) *RedisDLQ {
	return &RedisDLQ{
		client: client,
		ttl:    ttl,
	}
}

// Store stores a failed operation in the dead letter queue
func (d *RedisDLQ) Store(ctx context.Context, operation FailedOperation) error {
	// Serialize the operation to JSON
	data, err := json.Marshal(operation)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	// Store in Redis with TTL
	key := dlqKeyPrefix + operation.ID
	if err := d.client.Set(ctx, key, data, d.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store in Redis: %w", err)
	}

	// Add to sorted set for listing (score is timestamp)
	score := float64(operation.LastAttempt.Unix())
	if err := d.client.ZAdd(ctx, dlqListKey, redis.Z{
		Score:  score,
		Member: operation.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to DLQ list: %w", err)
	}

	return nil
}

// Get retrieves a failed operation from the dead letter queue by ID
func (d *RedisDLQ) Get(ctx context.Context, id string) (*FailedOperation, error) {
	key := dlqKeyPrefix + id

	data, err := d.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("operation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var operation FailedOperation
	if err := json.Unmarshal(data, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	return &operation, nil
}

// List retrieves a list of failed operations from the dead letter queue
// Returns up to 'limit' most recent operations
func (d *RedisDLQ) List(ctx context.Context, limit int) ([]FailedOperation, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	// Get most recent operation IDs from sorted set (highest scores first)
	ids, err := d.client.ZRevRange(ctx, dlqListKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list from DLQ: %w", err)
	}

	// Retrieve each operation
	operations := make([]FailedOperation, 0, len(ids))
	for _, id := range ids {
		op, err := d.Get(ctx, id)
		if err != nil {
			// Skip missing operations (they may have expired)
			continue
		}
		operations = append(operations, *op)
	}

	return operations, nil
}

// Delete removes a failed operation from the dead letter queue
func (d *RedisDLQ) Delete(ctx context.Context, id string) error {
	key := dlqKeyPrefix + id

	// Delete from Redis
	if err := d.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	// Remove from sorted set
	if err := d.client.ZRem(ctx, dlqListKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from DLQ list: %w", err)
	}

	return nil
}

// Cleanup removes operations older than the specified duration
func (d *RedisDLQ) Cleanup(ctx context.Context, olderThan time.Duration) error {
	// Calculate the cutoff timestamp
	cutoff := time.Now().Add(-olderThan).Unix()

	// Get operations older than cutoff
	ids, err := d.client.ZRangeByScore(ctx, dlqListKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", cutoff),
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to query old operations: %w", err)
	}

	if len(ids) == 0 {
		return nil
	}

	// Delete old operations
	for _, id := range ids {
		if err := d.Delete(ctx, id); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Failed to delete operation %s: %v\n", id, err)
		}
	}

	return nil
}

// Count returns the total number of operations in the dead letter queue
func (d *RedisDLQ) Count(ctx context.Context) (int64, error) {
	count, err := d.client.ZCard(ctx, dlqListKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count DLQ operations: %w", err)
	}
	return count, nil
}

// GetByOperation retrieves all failed operations for a specific operation type
func (d *RedisDLQ) GetByOperation(ctx context.Context, operation string, limit int) ([]FailedOperation, error) {
	if limit <= 0 {
		limit = 100
	}

	// Get all operation IDs
	ids, err := d.client.ZRevRange(ctx, dlqListKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list from DLQ: %w", err)
	}

	// Filter by operation type
	operations := make([]FailedOperation, 0)
	for _, id := range ids {
		if len(operations) >= limit {
			break
		}

		op, err := d.Get(ctx, id)
		if err != nil {
			continue
		}

		if op.Operation == operation {
			operations = append(operations, *op)
		}
	}

	return operations, nil
}

// GetByClientID retrieves all failed operations for a specific client ID
func (d *RedisDLQ) GetByClientID(ctx context.Context, clientID string, limit int) ([]FailedOperation, error) {
	if limit <= 0 {
		limit = 100
	}

	// Get all operation IDs
	ids, err := d.client.ZRevRange(ctx, dlqListKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list from DLQ: %w", err)
	}

	// Filter by client ID
	operations := make([]FailedOperation, 0)
	for _, id := range ids {
		if len(operations) >= limit {
			break
		}

		op, err := d.Get(ctx, id)
		if err != nil {
			continue
		}

		if op.ClientID == clientID {
			operations = append(operations, *op)
		}
	}

	return operations, nil
}

// Stats returns statistics about the dead letter queue
type DLQStats struct {
	TotalOperations int64            `json:"total_operations"`
	OperationCounts map[string]int64 `json:"operation_counts"`
	OldestOperation *time.Time       `json:"oldest_operation,omitempty"`
	NewestOperation *time.Time       `json:"newest_operation,omitempty"`
}

// GetStats returns statistics about the dead letter queue
func (d *RedisDLQ) GetStats(ctx context.Context) (*DLQStats, error) {
	stats := &DLQStats{
		OperationCounts: make(map[string]int64),
	}

	// Get total count
	count, err := d.Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalOperations = count

	if count == 0 {
		return stats, nil
	}

	// Get oldest and newest timestamps
	oldest, err := d.client.ZRange(ctx, dlqListKey, 0, 0).Result()
	if err == nil && len(oldest) > 0 {
		op, err := d.Get(ctx, oldest[0])
		if err == nil {
			stats.OldestOperation = &op.LastAttempt
		}
	}

	newest, err := d.client.ZRevRange(ctx, dlqListKey, 0, 0).Result()
	if err == nil && len(newest) > 0 {
		op, err := d.Get(ctx, newest[0])
		if err == nil {
			stats.NewestOperation = &op.LastAttempt
		}
	}

	// Count operations by type
	ids, err := d.client.ZRange(ctx, dlqListKey, 0, -1).Result()
	if err != nil {
		return stats, nil
	}

	for _, id := range ids {
		op, err := d.Get(ctx, id)
		if err != nil {
			continue
		}
		stats.OperationCounts[op.Operation]++
	}

	return stats, nil
}

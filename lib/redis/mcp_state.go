package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/logging"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultTTL is the default time-to-live for MCP client state entries
	DefaultTTL = 24 * time.Hour

	// DefaultMaxAge is the default maximum age for cleanup operations
	DefaultMaxAge = 7 * 24 * time.Hour

	// KeyPrefix is the prefix for all MCP state keys
	KeyPrefix = "mcp"

	// DefaultMaxRetries is the default number of retries for Redis operations
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default delay between retries
	DefaultRetryDelay = 100 * time.Millisecond
)

// MCPClientState represents the state of an MCP client connection
type MCPClientState struct {
	SessionID    string                 `json:"session_id"`
	MCPName      string                 `json:"mcp_name"`
	Status       string                 `json:"status"` // active, inactive
	Endpoint     string                 `json:"endpoint"`
	LastActivity time.Time              `json:"last_activity"`
	ToolCount    int                    `json:"tool_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MCPStateManager manages MCP client state in Redis
type MCPStateManager struct {
	client  *redis.Client
	logger  *logging.Logger
	ttl     time.Duration
	mu      sync.RWMutex
	metrics *StateMetrics
}

// StateMetrics tracks operational metrics for the state manager
type StateMetrics struct {
	mu              sync.RWMutex
	registrations   int64
	unregistrations int64
	gets            int64
	lists           int64
	errors          int64
	reconnections   int64
}

// MCPStateManagerConfig holds configuration for the state manager
type MCPStateManagerConfig struct {
	RedisClient *redis.Client
	TTL         time.Duration
	Logger      *logging.Logger
}

// NewMCPStateManager creates a new MCP state manager
func NewMCPStateManager(config MCPStateManagerConfig) (*MCPStateManager, error) {
	if config.RedisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}

	logger := config.Logger
	if logger == nil {
		logger = logging.GetLogger("redis.mcp_state")
	}

	manager := &MCPStateManager{
		client:  config.RedisClient,
		logger:  logger,
		ttl:     ttl,
		metrics: &StateMetrics{},
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := manager.client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info("MCP state manager initialized successfully")
	return manager, nil
}

// RegisterMCPClient registers a new MCP client in Redis
func (m *MCPStateManager) RegisterMCPClient(ctx context.Context, sessionID, mcpName, endpoint string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return fmt.Errorf("MCP name is required")
	}
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	state := MCPClientState{
		SessionID:    sessionID,
		MCPName:      mcpName,
		Status:       "active",
		Endpoint:     endpoint,
		LastActivity: time.Now(),
		ToolCount:    0,
		Metadata:     make(map[string]interface{}),
	}

	return m.setStateWithRetry(ctx, sessionID, mcpName, state)
}

// UnregisterMCPClient removes an MCP client from Redis
func (m *MCPStateManager) UnregisterMCPClient(ctx context.Context, sessionID, mcpName string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return fmt.Errorf("MCP name is required")
	}

	key := m.buildKey(sessionID, mcpName)

	err := m.executeWithRetry(ctx, func() error {
		return m.client.Del(ctx, key).Err()
	})

	if err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"mcp_name":   mcpName,
		}).ErrorWithError("failed to unregister MCP client", err)
		return fmt.Errorf("failed to delete MCP client state: %w", err)
	}

	m.incrementUnregistrations()
	m.logger.WithFields(map[string]interface{}{
		"session_id": sessionID,
		"mcp_name":   mcpName,
	}).Info("MCP client unregistered successfully")

	return nil
}

// GetMCPClient retrieves the state of an MCP client
func (m *MCPStateManager) GetMCPClient(ctx context.Context, sessionID, mcpName string) (*MCPClientState, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return nil, fmt.Errorf("MCP name is required")
	}

	key := m.buildKey(sessionID, mcpName)
	var data string

	err := m.executeWithRetry(ctx, func() error {
		result, err := m.client.Get(ctx, key).Result()
		if err != nil {
			return err
		}
		data = result
		return nil
	})

	if err == redis.Nil {
		return nil, fmt.Errorf("MCP client not found")
	}
	if err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"mcp_name":   mcpName,
		}).ErrorWithError("failed to get MCP client state", err)
		return nil, fmt.Errorf("failed to get MCP client state: %w", err)
	}

	var state MCPClientState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"mcp_name":   mcpName,
		}).ErrorWithError("failed to unmarshal MCP client state", err)
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	m.incrementGets()
	return &state, nil
}

// ListMCPClients retrieves all MCP clients for a session
func (m *MCPStateManager) ListMCPClients(ctx context.Context, sessionID string) ([]MCPClientState, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	pattern := fmt.Sprintf("%s:%s:*", KeyPrefix, sessionID)
	var keys []string

	err := m.executeWithRetry(ctx, func() error {
		var cursor uint64
		var scanKeys []string
		var err error

		// Scan for keys matching the pattern
		for {
			scanKeys, cursor, err = m.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}
			keys = append(keys, scanKeys...)
			if cursor == 0 {
				break
			}
		}
		return nil
	})

	if err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
		}).ErrorWithError("failed to scan for MCP clients", err)
		return nil, fmt.Errorf("failed to scan for MCP clients: %w", err)
	}

	if len(keys) == 0 {
		m.incrementLists()
		return []MCPClientState{}, nil
	}

	// Retrieve all client states using pipeline for efficiency
	pipe := m.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	var pipeErr error
	err = m.executeWithRetry(ctx, func() error {
		_, pipeErr = pipe.Exec(ctx)
		return pipeErr
	})

	// Process results even if some keys failed
	states := make([]MCPClientState, 0, len(keys))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			// Key was deleted between scan and get
			continue
		}
		if err != nil {
			m.logger.WithFields(map[string]interface{}{
				"key": keys[i],
			}).Warnf("failed to get key during list operation: %v", err)
			continue
		}

		var state MCPClientState
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			m.logger.WithFields(map[string]interface{}{
				"key": keys[i],
			}).Warnf("failed to unmarshal state: %v", err)
			continue
		}

		states = append(states, state)
	}

	m.incrementLists()
	return states, nil
}

// MarkActive updates the last activity timestamp and sets status to active
func (m *MCPStateManager) MarkActive(ctx context.Context, sessionID, mcpName string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return fmt.Errorf("MCP name is required")
	}

	// Get current state
	state, err := m.GetMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		return err
	}

	// Update state
	state.Status = "active"
	state.LastActivity = time.Now()

	return m.setStateWithRetry(ctx, sessionID, mcpName, *state)
}

// UpdateToolCount updates the tool count for an MCP client
func (m *MCPStateManager) UpdateToolCount(ctx context.Context, sessionID, mcpName string, toolCount int) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return fmt.Errorf("MCP name is required")
	}

	// Get current state
	state, err := m.GetMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		return err
	}

	// Update tool count
	state.ToolCount = toolCount
	state.LastActivity = time.Now()

	return m.setStateWithRetry(ctx, sessionID, mcpName, *state)
}

// CleanupExpiredClients removes clients that haven't been active within maxAge
func (m *MCPStateManager) CleanupExpiredClients(ctx context.Context, maxAge time.Duration) error {
	if maxAge == 0 {
		maxAge = DefaultMaxAge
	}

	pattern := fmt.Sprintf("%s:*:*", KeyPrefix)
	var keys []string

	err := m.executeWithRetry(ctx, func() error {
		var cursor uint64
		var scanKeys []string
		var err error

		for {
			scanKeys, cursor, err = m.client.Scan(ctx, cursor, pattern, 1000).Result()
			if err != nil {
				return err
			}
			keys = append(keys, scanKeys...)
			if cursor == 0 {
				break
			}
		}
		return nil
	})

	if err != nil {
		m.incrementErrors()
		m.logger.ErrorWithError("failed to scan for expired clients", err)
		return fmt.Errorf("failed to scan for expired clients: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	cutoffTime := time.Now().Add(-maxAge)
	expiredKeys := make([]string, 0)

	// Check each key for expiration
	for _, key := range keys {
		data, err := m.client.Get(ctx, key).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			m.logger.WithFields(map[string]interface{}{
				"key": key,
			}).Warnf("failed to get key during cleanup: %v", err)
			continue
		}

		var state MCPClientState
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			m.logger.WithFields(map[string]interface{}{
				"key": key,
			}).Warnf("failed to unmarshal state during cleanup: %v", err)
			continue
		}

		if state.LastActivity.Before(cutoffTime) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Delete expired keys in batches using transaction
	if len(expiredKeys) > 0 {
		err := m.executeWithRetry(ctx, func() error {
			pipe := m.client.Pipeline()
			for _, key := range expiredKeys {
				pipe.Del(ctx, key)
			}
			_, err := pipe.Exec(ctx)
			return err
		})

		if err != nil {
			m.incrementErrors()
			m.logger.ErrorWithError("failed to delete expired clients", err)
			return fmt.Errorf("failed to delete expired clients: %w", err)
		}

		m.logger.WithFields(map[string]interface{}{
			"count":   len(expiredKeys),
			"max_age": maxAge.String(),
		}).Info("cleaned up expired MCP clients")
	}

	return nil
}

// SetMetadata sets custom metadata for an MCP client
func (m *MCPStateManager) SetMetadata(ctx context.Context, sessionID, mcpName string, metadata map[string]interface{}) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if mcpName == "" {
		return fmt.Errorf("MCP name is required")
	}

	// Get current state
	state, err := m.GetMCPClient(ctx, sessionID, mcpName)
	if err != nil {
		return err
	}

	// Update metadata
	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		state.Metadata[k] = v
	}
	state.LastActivity = time.Now()

	return m.setStateWithRetry(ctx, sessionID, mcpName, *state)
}

// RecoverConnection attempts to recover the Redis connection
func (m *MCPStateManager) RecoverConnection(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Test connection
	if err := m.client.Ping(ctx).Err(); err != nil {
		m.incrementErrors()
		m.logger.ErrorWithError("failed to recover Redis connection", err)
		return fmt.Errorf("failed to recover connection: %w", err)
	}

	m.incrementReconnections()
	m.logger.Info("Redis connection recovered successfully")
	return nil
}

// GetMetrics returns current operational metrics
func (m *MCPStateManager) GetMetrics() map[string]int64 {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return map[string]int64{
		"registrations":   m.metrics.registrations,
		"unregistrations": m.metrics.unregistrations,
		"gets":            m.metrics.gets,
		"lists":           m.metrics.lists,
		"errors":          m.metrics.errors,
		"reconnections":   m.metrics.reconnections,
	}
}

// Close closes the state manager (does not close the Redis client)
func (m *MCPStateManager) Close() error {
	m.logger.Info("MCP state manager closed")
	return nil
}

// Helper methods

func (m *MCPStateManager) buildKey(sessionID, mcpName string) string {
	return fmt.Sprintf("%s:%s:%s", KeyPrefix, sessionID, mcpName)
}

func (m *MCPStateManager) setStateWithRetry(ctx context.Context, sessionID, mcpName string, state MCPClientState) error {
	key := m.buildKey(sessionID, mcpName)

	data, err := json.Marshal(state)
	if err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"mcp_name":   mcpName,
		}).ErrorWithError("failed to marshal MCP client state", err)
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = m.executeWithRetry(ctx, func() error {
		return m.client.Set(ctx, key, data, m.ttl).Err()
	})

	if err != nil {
		m.incrementErrors()
		m.logger.WithFields(map[string]interface{}{
			"session_id": sessionID,
			"mcp_name":   mcpName,
		}).ErrorWithError("failed to set MCP client state", err)
		return fmt.Errorf("failed to set state: %w", err)
	}

	m.incrementRegistrations()
	m.logger.WithFields(map[string]interface{}{
		"session_id": sessionID,
		"mcp_name":   mcpName,
		"status":     state.Status,
	}).Debug("MCP client state updated")

	return nil
}

func (m *MCPStateManager) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < DefaultMaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := DefaultRetryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if !m.isRetryableError(lastErr) {
			return lastErr
		}

		m.logger.WithFields(map[string]interface{}{
			"attempt":     attempt + 1,
			"max_retries": DefaultMaxRetries,
		}).Warnf("retrying operation after error: %v", lastErr)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", DefaultMaxRetries, lastErr)
}

func (m *MCPStateManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"i/o timeout",
		"context deadline exceeded",
		"no such host",
		"network is unreachable",
	}

	errLower := strings.ToLower(errStr)
	for _, retryable := range retryableErrors {
		if strings.Contains(errLower, retryable) {
			return true
		}
	}

	return false
}

// Metrics helper methods

func (m *MCPStateManager) incrementRegistrations() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.registrations++
}

func (m *MCPStateManager) incrementUnregistrations() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.unregistrations++
}

func (m *MCPStateManager) incrementGets() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.gets++
}

func (m *MCPStateManager) incrementLists() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.lists++
}

func (m *MCPStateManager) incrementErrors() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.errors++
}

func (m *MCPStateManager) incrementReconnections() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.reconnections++
}

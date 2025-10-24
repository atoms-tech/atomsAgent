package redis

import (
	"context"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/logging"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a test Redis client
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Use a test database
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up the test database
	client.FlushDB(ctx)

	return client
}

func TestNewMCPStateManager(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	tests := []struct {
		name        string
		config      MCPStateManagerConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: MCPStateManagerConfig{
				RedisClient: client,
				TTL:         time.Hour,
				Logger:      logging.GetLogger("test"),
			},
			expectError: false,
		},
		{
			name: "valid config with defaults",
			config: MCPStateManagerConfig{
				RedisClient: client,
			},
			expectError: false,
		},
		{
			name: "nil redis client",
			config: MCPStateManagerConfig{
				TTL:    time.Hour,
				Logger: logging.GetLogger("test"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewMCPStateManager(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				if manager != nil {
					manager.Close()
				}
			}
		})
	}
}

func TestRegisterMCPClient(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		sessionID   string
		mcpName     string
		endpoint    string
		expectError bool
	}{
		{
			name:        "valid registration",
			sessionID:   "session-1",
			mcpName:     "test-mcp",
			endpoint:    "http://localhost:8080",
			expectError: false,
		},
		{
			name:        "empty session ID",
			sessionID:   "",
			mcpName:     "test-mcp",
			endpoint:    "http://localhost:8080",
			expectError: true,
		},
		{
			name:        "empty MCP name",
			sessionID:   "session-1",
			mcpName:     "",
			endpoint:    "http://localhost:8080",
			expectError: true,
		},
		{
			name:        "empty endpoint",
			sessionID:   "session-1",
			mcpName:     "test-mcp",
			endpoint:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterMCPClient(ctx, tt.sessionID, tt.mcpName, tt.endpoint)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMCPClient(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	tests := []struct {
		name        string
		sessionID   string
		mcpName     string
		expectError bool
	}{
		{
			name:        "existing client",
			sessionID:   "session-1",
			mcpName:     "test-mcp",
			expectError: false,
		},
		{
			name:        "non-existent client",
			sessionID:   "session-2",
			mcpName:     "test-mcp",
			expectError: true,
		},
		{
			name:        "empty session ID",
			sessionID:   "",
			mcpName:     "test-mcp",
			expectError: true,
		},
		{
			name:        "empty MCP name",
			sessionID:   "session-1",
			mcpName:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := manager.GetMCPClient(ctx, tt.sessionID, tt.mcpName)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, state)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state)
				assert.Equal(t, tt.sessionID, state.SessionID)
				assert.Equal(t, tt.mcpName, state.MCPName)
				assert.Equal(t, "active", state.Status)
				assert.Equal(t, "http://localhost:8080", state.Endpoint)
			}
		})
	}
}

func TestListMCPClients(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register multiple clients for the same session
	err = manager.RegisterMCPClient(ctx, "session-1", "mcp-1", "http://localhost:8080")
	require.NoError(t, err)
	err = manager.RegisterMCPClient(ctx, "session-1", "mcp-2", "http://localhost:8081")
	require.NoError(t, err)
	err = manager.RegisterMCPClient(ctx, "session-2", "mcp-3", "http://localhost:8082")
	require.NoError(t, err)

	tests := []struct {
		name          string
		sessionID     string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "session with multiple clients",
			sessionID:     "session-1",
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "session with one client",
			sessionID:     "session-2",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "session with no clients",
			sessionID:     "session-3",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "empty session ID",
			sessionID:     "",
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states, err := manager.ListMCPClients(ctx, tt.sessionID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, states, tt.expectedCount)
			}
		})
	}
}

func TestUnregisterMCPClient(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	// Verify it exists
	state, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)
	require.NotNil(t, state)

	// Unregister
	err = manager.UnregisterMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)

	// Verify it no longer exists
	state, err = manager.GetMCPClient(ctx, "session-1", "test-mcp")
	assert.Error(t, err)
	assert.Nil(t, state)
}

func TestMarkActive(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	// Get initial state
	state1, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)
	initialTime := state1.LastActivity

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Mark active
	err = manager.MarkActive(ctx, "session-1", "test-mcp")
	require.NoError(t, err)

	// Get updated state
	state2, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)

	assert.Equal(t, "active", state2.Status)
	assert.True(t, state2.LastActivity.After(initialTime))
}

func TestUpdateToolCount(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	// Update tool count
	err = manager.UpdateToolCount(ctx, "session-1", "test-mcp", 5)
	require.NoError(t, err)

	// Verify tool count
	state, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)
	assert.Equal(t, 5, state.ToolCount)
}

func TestSetMetadata(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	// Set metadata
	metadata := map[string]interface{}{
		"version": "1.0.0",
		"env":     "production",
	}
	err = manager.SetMetadata(ctx, "session-1", "test-mcp", metadata)
	require.NoError(t, err)

	// Verify metadata
	state, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", state.Metadata["version"])
	assert.Equal(t, "production", state.Metadata["env"])
}

func TestCleanupExpiredClients(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register clients
	err = manager.RegisterMCPClient(ctx, "session-1", "mcp-1", "http://localhost:8080")
	require.NoError(t, err)
	err = manager.RegisterMCPClient(ctx, "session-2", "mcp-2", "http://localhost:8081")
	require.NoError(t, err)

	// Manually set an old timestamp for one client
	state, err := manager.GetMCPClient(ctx, "session-1", "mcp-1")
	require.NoError(t, err)
	state.LastActivity = time.Now().Add(-8 * 24 * time.Hour)
	err = manager.setStateWithRetry(ctx, "session-1", "mcp-1", *state)
	require.NoError(t, err)

	// Run cleanup
	err = manager.CleanupExpiredClients(ctx, 7*24*time.Hour)
	require.NoError(t, err)

	// Verify expired client is gone
	state, err = manager.GetMCPClient(ctx, "session-1", "mcp-1")
	assert.Error(t, err)
	assert.Nil(t, state)

	// Verify active client remains
	state, err = manager.GetMCPClient(ctx, "session-2", "mcp-2")
	assert.NoError(t, err)
	assert.NotNil(t, state)
}

func TestGetMetrics(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Perform operations
	err = manager.RegisterMCPClient(ctx, "session-1", "mcp-1", "http://localhost:8080")
	require.NoError(t, err)

	_, err = manager.GetMCPClient(ctx, "session-1", "mcp-1")
	require.NoError(t, err)

	_, err = manager.ListMCPClients(ctx, "session-1")
	require.NoError(t, err)

	err = manager.UnregisterMCPClient(ctx, "session-1", "mcp-1")
	require.NoError(t, err)

	// Get metrics
	metrics := manager.GetMetrics()
	assert.GreaterOrEqual(t, metrics["registrations"], int64(1))
	assert.GreaterOrEqual(t, metrics["gets"], int64(1))
	assert.GreaterOrEqual(t, metrics["lists"], int64(1))
	assert.GreaterOrEqual(t, metrics["unregistrations"], int64(1))
}

func TestConcurrentAccess(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         time.Hour,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Perform concurrent operations
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			sessionID := "session-1"
			mcpName := "mcp-" + string(rune('0'+id))
			endpoint := "http://localhost:8080"

			err := manager.RegisterMCPClient(ctx, sessionID, mcpName, endpoint)
			assert.NoError(t, err)

			_, err = manager.GetMCPClient(ctx, sessionID, mcpName)
			assert.NoError(t, err)

			err = manager.MarkActive(ctx, sessionID, mcpName)
			assert.NoError(t, err)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all clients were registered
	states, err := manager.ListMCPClients(ctx, "session-1")
	require.NoError(t, err)
	assert.Len(t, states, numGoroutines)
}

func TestTTLExpiration(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	// Use a short TTL for testing
	manager, err := NewMCPStateManager(MCPStateManagerConfig{
		RedisClient: client,
		TTL:         100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Register a client
	err = manager.RegisterMCPClient(ctx, "session-1", "test-mcp", "http://localhost:8080")
	require.NoError(t, err)

	// Verify it exists
	state, err := manager.GetMCPClient(ctx, "session-1", "test-mcp")
	require.NoError(t, err)
	require.NotNil(t, state)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify it no longer exists
	state, err = manager.GetMCPClient(ctx, "session-1", "test-mcp")
	assert.Error(t, err)
	assert.Nil(t, state)
}

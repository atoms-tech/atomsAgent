# Redis Session Store

Production-ready Redis-backed session persistence for the AgentAPI session manager.

## Overview

The Redis Session Store provides durable, distributed session persistence with automatic TTL-based cleanup, thread-safe operations, and graceful fallback to in-memory storage when Redis is unavailable.

## Features

- **Persistent Storage**: Sessions survive server restarts and can be shared across multiple instances
- **Automatic TTL**: Sessions automatically expire and are cleaned up by Redis
- **Thread-Safe**: All operations are concurrent-safe with proper locking
- **Fallback Support**: Gracefully degrades to in-memory storage if Redis is unavailable
- **Batch Operations**: Efficient batch store/delete operations
- **Atomic Transactions**: Thread-safe operations with proper error handling
- **Dual Protocol Support**: Works with both native Redis protocol and REST API

## Architecture

```
┌─────────────────────┐
│ SessionManagerV2    │
│                     │
│  ┌──────────────┐   │
│  │  sync.Map    │   │  In-Memory Cache
│  │  (sessions)  │   │
│  └──────────────┘   │
│         │           │
│         ▼           │
│  ┌──────────────┐   │
│  │SessionStore  │   │  Interface
│  └──────────────┘   │
└─────────┬───────────┘
          │
          ├─────────────────────┐
          │                     │
┌─────────▼──────────┐  ┌──────▼────────────┐
│RedisSessionStore   │  │InMemorySessionStore│
│  ┌──────────────┐  │  │   (fallback)       │
│  │ RedisClient  │  │  └────────────────────┘
│  └──────────────┘  │
│         │          │
│    ┌────┴────┐     │
│    ▼         ▼     │
│  Native    REST    │
│  Redis      API    │
└────────────────────┘
```

## Data Structure

### Redis Keys

- **Session Data**: `session:{sessionID}` → JSON-encoded session data
- **User Index**: `user_sessions:{userID}` → JSON array of session IDs

### Session Data Format

```json
{
  "id": "uuid-v4",
  "user_id": "user-123",
  "org_id": "org-456",
  "workspace_path": "/tmp/workspaces/user-123/uuid-v4",
  "mcp_client_ids": ["client-1", "client-2"],
  "system_prompt": "You are a helpful assistant",
  "created_at": "2025-01-15T10:30:00Z",
  "last_active_at": "2025-01-15T11:45:00Z",
  "metadata": {}
}
```

## Usage

### Basic Setup with SessionManager

```go
import (
    "github.com/coder/agentapi/lib/redis"
    "github.com/coder/agentapi/lib/session"
)

// Configure Redis
redisConfig := redis.DefaultConfig()
redisConfig.URL = "redis://localhost:6379"

// Create Redis client
redisClient, err := redis.NewRedisClient(redisConfig)
if err != nil {
    log.Fatal(err)
}
defer redisClient.Close()

// Create session store with 24-hour TTL
sessionStore := redis.NewSessionStore(redisClient, 24*time.Hour)

// Create session manager
sessionManager := session.NewSessionManagerV2("/tmp/workspaces", 100)

// Enable Redis persistence with sync-on-access
if err := sessionManager.SetSessionStore(sessionStore, true); err != nil {
    log.Printf("Warning: Redis unavailable, using in-memory: %v", err)
}

// Create a session (automatically persisted to Redis)
sess, err := sessionManager.CreateSession(ctx, "user-123", "org-456")
if err != nil {
    log.Fatal(err)
}
```

### Direct Session Store Usage

```go
// Create session store
store := redis.NewSessionStore(redisClient, 1*time.Hour)

// Store session
err := store.StoreSession(ctx, session)

// Retrieve session
session, err := store.RetrieveSession(ctx, sessionID)

// Update session
err := store.UpdateSession(ctx, session)

// Delete session
err := store.DeleteSession(ctx, sessionID)

// List sessions for user
sessions, err := store.ListSessions(ctx, userID)

// Check existence
exists, err := store.Exists(ctx, sessionID)

// Update TTL
err := store.SetTTL(ctx, sessionID, 2*time.Hour)
```

### Batch Operations

```go
// Batch store multiple sessions
sessions := []*session.Session{sess1, sess2, sess3}
err := store.BatchStoreSession(ctx, sessions)

// Batch delete multiple sessions
sessionIDs := []string{"sess-1", "sess-2", "sess-3"}
err := store.BatchDeleteSessions(ctx, sessionIDs)
```

### Fallback to In-Memory

```go
// Create session manager (without Redis initially)
sessionManager := session.NewSessionManagerV2("/tmp/workspaces", 100)

// Sessions work immediately using in-memory storage
sess, _ := sessionManager.CreateSession(ctx, "user-123", "org-456")

// Later, add Redis persistence (existing sessions remain in memory)
redisClient, _ := redis.NewRedisClient(redisConfig)
sessionStore := redis.NewSessionStore(redisClient, 24*time.Hour)
sessionManager.SetSessionStore(sessionStore, true)

// New sessions are persisted to Redis
// Existing sessions are lazily synced on access if sync-on-access is enabled
```

## Configuration Options

### Session Store TTL

```go
// Create with custom TTL
store := redis.NewSessionStore(redisClient, 6*time.Hour)

// Update default TTL
store.SetDefaultTTL(12*time.Hour)

// Get current TTL
ttl := store.GetTTL()

// Update specific session TTL
store.SetTTL(ctx, sessionID, 24*time.Hour)
```

### Sync Modes

#### Sync on Access (Recommended for Multi-Instance)

```go
// Enable sync-on-access
sessionManager.SetSessionStore(sessionStore, true)

// Sessions are synced from Redis when:
// - GetSession() is called and session not in memory
// - ListSessions() is called for a user
```

#### No Sync (Best Performance)

```go
// Disable sync-on-access
sessionManager.SetSessionStore(sessionStore, false)

// Sessions are persisted to Redis but not automatically synced
// Use for single-instance deployments where Redis is backup only
```

## Error Handling

The Redis session store implements graceful degradation:

```go
// Redis failure during session creation
sess, err := sessionManager.CreateSession(ctx, userID, orgID)
// ✓ Session created in memory
// ⚠ Redis persistence failed (logged)
// ✓ Redis disabled automatically

// Redis unavailable during setup
err := sessionManager.SetSessionStore(sessionStore, true)
if err != nil {
    // Redis health check failed
    // SessionManager continues with in-memory only
}

// Redis failure during retrieval
sess, err := sessionManager.GetSession(sessionID)
// ✓ Returns from memory if available
// ✓ Tries Redis if sync-on-access enabled
// ✗ Returns ErrSessionNotFound if not found anywhere
```

## Performance Considerations

### Memory vs Redis

- **In-Memory**: Microsecond latency, no network overhead
- **Redis (Native)**: ~1-2ms latency, network overhead
- **Redis (REST)**: ~5-10ms latency, HTTP overhead

### Optimization Strategies

1. **Lazy Sync**: Only sync from Redis when needed (sync-on-access)
2. **Batch Operations**: Use batch store/delete for multiple sessions
3. **TTL-Based Cleanup**: Let Redis handle expiration automatically
4. **Connection Pooling**: Reuse connections (handled by RedisClient)

### Best Practices

```go
// ✓ Good: Create store once, reuse
store := redis.NewSessionStore(redisClient, 24*time.Hour)

// ✗ Bad: Create new store for each operation
for _, sess := range sessions {
    store := redis.NewSessionStore(redisClient, 24*time.Hour)
    store.StoreSession(ctx, sess)
}

// ✓ Good: Use batch operations
store.BatchStoreSession(ctx, sessions)

// ✗ Bad: Individual operations in loop
for _, sess := range sessions {
    store.StoreSession(ctx, sess)
}
```

## Thread Safety

All operations are thread-safe:

```go
// Safe: Concurrent reads
go sessionManager.GetSession(sessionID)
go sessionManager.GetSession(sessionID)

// Safe: Concurrent writes
go sessionManager.CreateSession(ctx, "user-1", "org-1")
go sessionManager.CreateSession(ctx, "user-2", "org-2")

// Safe: Concurrent read/write
go sessionManager.GetSession(sessionID)
go sessionManager.CleanupSession(ctx, sessionID)
```

## Monitoring

### Health Checks

```go
// Check Redis connection health
err := sessionStore.Health()
if err != nil {
    log.Printf("Redis unhealthy: %v", err)
}

// Check if Redis is enabled
if sessionManager.IsUsingRedis() {
    log.Println("Redis persistence active")
} else {
    log.Println("Using in-memory storage only")
}
```

### Metrics to Monitor

- Session creation rate
- Redis connection failures
- Session retrieval latency
- TTL expiration rate
- Memory usage (in-memory cache)
- Redis memory usage

## Testing

### Unit Tests

```bash
# Run session store tests
go test -v ./lib/redis -run TestInMemorySessionStore

# Run with Redis (requires Redis instance)
REDIS_URL=redis://localhost:6379 go test -v ./lib/redis -run TestRedisSessionStore
```

### Integration Tests

```go
// Test with mock Redis
func TestSessionManagerWithRedis(t *testing.T) {
    // Setup
    redisClient := setupTestRedis(t)
    store := redis.NewSessionStore(redisClient, 1*time.Hour)
    manager := session.NewSessionManagerV2("/tmp/test", 10)
    manager.SetSessionStore(store, true)

    // Test session lifecycle
    sess, err := manager.CreateSession(ctx, "user-1", "org-1")
    require.NoError(t, err)

    // Verify persistence
    retrieved, err := store.RetrieveSession(ctx, sess.ID)
    require.NoError(t, err)
    assert.Equal(t, sess.ID, retrieved.ID)

    // Cleanup
    err = manager.CleanupSession(ctx, sess.ID)
    require.NoError(t, err)

    // Verify deletion
    _, err = store.RetrieveSession(ctx, sess.ID)
    assert.Error(t, err)
}
```

## Migration Guide

### From In-Memory to Redis

```go
// Before: In-memory only
manager := session.NewSessionManagerV2("/tmp/workspaces", 100)

// After: Add Redis persistence
redisClient, _ := redis.NewRedisClient(redisConfig)
sessionStore := redis.NewSessionStore(redisClient, 24*time.Hour)
manager.SetSessionStore(sessionStore, true)

// Existing in-memory sessions are preserved
// New sessions are persisted to Redis
```

### Multi-Instance Deployment

```go
// Each instance should:
// 1. Connect to same Redis instance
// 2. Enable sync-on-access
// 3. Use shared workspace storage (e.g., NFS, S3)

manager := session.NewSessionManagerV2("/shared/workspaces", 100)
manager.SetSessionStore(sessionStore, true) // Enable sync
```

## Troubleshooting

### Redis Connection Errors

```
Error: failed to store session in Redis: connection refused
```

**Solution**: Check Redis is running and URL is correct

```bash
redis-cli ping
# Should return: PONG
```

### Session Not Found After Server Restart

```
Error: session not found
```

**Cause**: sync-on-access is disabled and session not in memory

**Solution**: Enable sync-on-access when setting store

```go
manager.SetSessionStore(sessionStore, true) // Enable sync
```

### High Memory Usage

**Cause**: Both in-memory cache and Redis storing sessions

**Solution**: Implement periodic cleanup of in-memory cache

```go
// Periodically cleanup inactive in-memory sessions
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        manager.CleanupInactiveSessions(ctx, 24*time.Hour)
    }
}()
```

## Environment Variables

```bash
# Redis connection URL
REDIS_URL=redis://localhost:6379

# Redis REST API (for fallback)
REDIS_REST_URL=https://redis-api.example.com
REDIS_TOKEN=your-api-token

# Session TTL (optional)
SESSION_TTL=24h
```

## API Reference

### RedisSessionStore

```go
type RedisSessionStore struct {
    // Implements SessionStore interface
}

func NewSessionStore(client *RedisClient, ttl time.Duration) *RedisSessionStore

func (s *RedisSessionStore) StoreSession(ctx context.Context, sess *Session) error
func (s *RedisSessionStore) RetrieveSession(ctx context.Context, sessionID string) (*Session, error)
func (s *RedisSessionStore) UpdateSession(ctx context.Context, sess *Session) error
func (s *RedisSessionStore) DeleteSession(ctx context.Context, sessionID string) error
func (s *RedisSessionStore) ListSessions(ctx context.Context, userID string) ([]*Session, error)
func (s *RedisSessionStore) Exists(ctx context.Context, sessionID string) (bool, error)
func (s *RedisSessionStore) SetTTL(ctx context.Context, sessionID string, ttl time.Duration) error
func (s *RedisSessionStore) Health() error
func (s *RedisSessionStore) Close() error

// Additional methods
func (s *RedisSessionStore) GetTTL() time.Duration
func (s *RedisSessionStore) SetDefaultTTL(ttl time.Duration)
func (s *RedisSessionStore) BatchStoreSession(ctx context.Context, sessions []*Session) error
func (s *RedisSessionStore) BatchDeleteSessions(ctx context.Context, sessionIDs []string) error
func (s *RedisSessionStore) CleanupExpiredSessions(ctx context.Context, userID string) (int, error)
```

### SessionManager Integration

```go
// Set session store
func (sm *SessionManagerV2) SetSessionStore(store SessionStore, syncOnAccess bool) error

// Check if using Redis
func (sm *SessionManagerV2) IsUsingRedis() bool

// Get current store
func (sm *SessionManagerV2) GetSessionStore() SessionStore
```

## License

Same as AgentAPI project license.

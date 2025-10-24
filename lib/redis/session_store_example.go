package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coder/agentapi/lib/session"
)

// ExampleSessionStoreWithRedis demonstrates how to use RedisSessionStore with SessionManagerV2
func ExampleSessionStoreWithRedis() {
	// Configure Redis client
	redisConfig := DefaultConfig()
	redisConfig.URL = os.Getenv("REDIS_URL") // e.g., "redis://localhost:6379"
	redisConfig.RESTBaseURL = os.Getenv("REDIS_REST_URL")
	redisConfig.Token = os.Getenv("REDIS_TOKEN")

	// Create Redis client
	redisClient, err := NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create session store with 24-hour TTL
	sessionStore := NewSessionStore(redisClient, 24*time.Hour)

	// Create session manager
	sessionManager := session.NewSessionManagerV2("/tmp/workspaces", 100)

	// Set the session store with sync-on-access enabled
	if err := sessionManager.SetSessionStore(sessionStore, true); err != nil {
		log.Printf("Warning: Redis not available, using in-memory only: %v", err)
	}

	ctx := context.Background()

	// Create a session
	sess, err := sessionManager.CreateSession(ctx, "user-123", "org-456")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Created session: %s\n", sess.ID)
	fmt.Printf("Using Redis: %v\n", sessionManager.IsUsingRedis())

	// Retrieve the session
	retrieved, err := sessionManager.GetSession(sess.ID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Retrieved session: %s for user: %s\n", retrieved.ID, retrieved.UserID)

	// List sessions for user
	sessions := sessionManager.ListSessions("user-123")
	fmt.Printf("User has %d active sessions\n", len(sessions))

	// Cleanup
	if err := sessionManager.CleanupSession(ctx, sess.ID); err != nil {
		log.Fatalf("Failed to cleanup session: %v", err)
	}

	fmt.Println("Session cleaned up successfully")

	// Output:
	// Created session: ...
	// Using Redis: true
	// Retrieved session: ... for user: user-123
	// User has 1 active sessions
	// Session cleaned up successfully
}

// ExampleSessionStoreWithFallback demonstrates fallback to in-memory when Redis is unavailable
func ExampleSessionStoreWithFallback() {
	// Create session manager
	sessionManager := session.NewSessionManagerV2("/tmp/workspaces", 100)

	// Try to configure Redis (will fail if unavailable)
	redisConfig := DefaultConfig()
	redisConfig.URL = "redis://invalid-host:6379"

	redisClient, err := NewRedisClient(redisConfig)
	if err == nil {
		sessionStore := NewSessionStore(redisClient, 24*time.Hour)
		if err := sessionManager.SetSessionStore(sessionStore, true); err != nil {
			log.Printf("Redis unavailable, using in-memory: %v", err)
		}
	} else {
		log.Printf("Redis client creation failed, using in-memory: %v", err)
	}

	ctx := context.Background()

	// Create a session (works even without Redis)
	sess, err := sessionManager.CreateSession(ctx, "user-123", "org-456")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Created session: %s\n", sess.ID)
	fmt.Printf("Using Redis: %v (fallback to in-memory)\n", sessionManager.IsUsingRedis())

	// Session operations work the same way
	retrieved, err := sessionManager.GetSession(sess.ID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	fmt.Printf("Retrieved session from memory: %s\n", retrieved.ID)

	// Output:
	// Redis unavailable, using in-memory: ...
	// Created session: ...
	// Using Redis: false (fallback to in-memory)
	// Retrieved session from memory: ...
}

// ExampleDirectRedisSessionStore demonstrates direct use of RedisSessionStore
func ExampleDirectRedisSessionStore() {
	// Configure Redis
	redisConfig := DefaultConfig()
	redisConfig.URL = os.Getenv("REDIS_URL")

	redisClient, err := NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create session store
	store := NewSessionStore(redisClient, 1*time.Hour)

	ctx := context.Background()

	// Create a session object
	sess := &session.Session{
		ID:            "test-session-123",
		UserID:        "user-456",
		OrgID:         "org-789",
		WorkspacePath: "/tmp/workspace",
		SystemPrompt:  "You are a helpful assistant",
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
	}

	// Store in Redis
	if err := store.StoreSession(ctx, sess); err != nil {
		log.Fatalf("Failed to store session: %v", err)
	}
	fmt.Println("Session stored in Redis")

	// Check existence
	exists, err := store.Exists(ctx, sess.ID)
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("Session exists: %v\n", exists)

	// Retrieve from Redis
	retrieved, err := store.RetrieveSession(ctx, sess.ID)
	if err != nil {
		log.Fatalf("Failed to retrieve session: %v", err)
	}
	fmt.Printf("Retrieved session: %s for user: %s\n", retrieved.ID, retrieved.UserID)

	// List sessions for user
	sessions, err := store.ListSessions(ctx, "user-456")
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}
	fmt.Printf("User has %d sessions in Redis\n", len(sessions))

	// Update TTL
	if err := store.SetTTL(ctx, sess.ID, 2*time.Hour); err != nil {
		log.Fatalf("Failed to update TTL: %v", err)
	}
	fmt.Println("Session TTL updated to 2 hours")

	// Delete from Redis
	if err := store.DeleteSession(ctx, sess.ID); err != nil {
		log.Fatalf("Failed to delete session: %v", err)
	}
	fmt.Println("Session deleted from Redis")

	// Output:
	// Session stored in Redis
	// Session exists: true
	// Retrieved session: test-session-123 for user: user-456
	// User has 1 sessions in Redis
	// Session TTL updated to 2 hours
	// Session deleted from Redis
}

// ExampleBatchOperations demonstrates batch session operations
func ExampleBatchOperations() {
	// Setup
	redisConfig := DefaultConfig()
	redisConfig.URL = os.Getenv("REDIS_URL")

	redisClient, err := NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	store := NewSessionStore(redisClient, 24*time.Hour)
	ctx := context.Background()

	// Create multiple sessions
	sessions := []*session.Session{
		{
			ID:            "sess-1",
			UserID:        "user-1",
			OrgID:         "org-1",
			WorkspacePath: "/tmp/ws1",
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
		{
			ID:            "sess-2",
			UserID:        "user-1",
			OrgID:         "org-1",
			WorkspacePath: "/tmp/ws2",
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
		{
			ID:            "sess-3",
			UserID:        "user-1",
			OrgID:         "org-1",
			WorkspacePath: "/tmp/ws3",
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
	}

	// Batch store sessions
	if err := store.BatchStoreSession(ctx, sessions); err != nil {
		log.Fatalf("Failed to batch store sessions: %v", err)
	}
	fmt.Printf("Stored %d sessions in batch\n", len(sessions))

	// List all sessions for user
	userSessions, err := store.ListSessions(ctx, "user-1")
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}
	fmt.Printf("User has %d sessions\n", len(userSessions))

	// Batch delete sessions
	sessionIDs := []string{"sess-1", "sess-2", "sess-3"}
	if err := store.BatchDeleteSessions(ctx, sessionIDs); err != nil {
		log.Fatalf("Failed to batch delete sessions: %v", err)
	}
	fmt.Printf("Deleted %d sessions in batch\n", len(sessionIDs))

	// Output:
	// Stored 3 sessions in batch
	// User has 3 sessions
	// Deleted 3 sessions in batch
}

// ExampleCleanupExpiredSessions demonstrates automatic cleanup of expired sessions
func ExampleCleanupExpiredSessions() {
	redisConfig := DefaultConfig()
	redisConfig.URL = os.Getenv("REDIS_URL")

	redisClient, err := NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// Create store with short TTL for demonstration
	store := NewSessionStore(redisClient, 5*time.Second)
	ctx := context.Background()

	// Create a session
	sess := &session.Session{
		ID:            "temp-session",
		UserID:        "user-cleanup",
		OrgID:         "org-cleanup",
		WorkspacePath: "/tmp/temp",
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
	}

	if err := store.StoreSession(ctx, sess); err != nil {
		log.Fatalf("Failed to store session: %v", err)
	}
	fmt.Println("Session stored with 5 second TTL")

	// Wait for expiration (in real usage, this happens automatically)
	time.Sleep(6 * time.Second)

	// Try to retrieve expired session
	_, err = store.RetrieveSession(ctx, sess.ID)
	if err == ErrSessionNotFound {
		fmt.Println("Session expired and automatically cleaned up")
	}

	// Cleanup expired sessions for user
	removed, err := store.CleanupExpiredSessions(ctx, "user-cleanup")
	if err != nil {
		log.Fatalf("Failed to cleanup: %v", err)
	}
	fmt.Printf("Cleaned up %d expired sessions\n", removed)

	// Output:
	// Session stored with 5 second TTL
	// Session expired and automatically cleaned up
	// Cleaned up 0 expired sessions
}

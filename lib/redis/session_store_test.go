package redis

import (
	"context"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/session"
)

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	ctx := context.Background()

	// Create a test session
	sess := &session.Session{
		ID:            "test-session-1",
		UserID:        "user-123",
		OrgID:         "org-456",
		WorkspacePath: "/tmp/workspace",
		MCPClients:    make(map[string]*mcp.Client),
		SystemPrompt:  "Test prompt",
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
	}

	// Test StoreSession
	err := store.StoreSession(ctx, sess)
	if err != nil {
		t.Fatalf("Failed to store session: %v", err)
	}

	// Test Exists
	exists, err := store.Exists(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Errorf("Session should exist")
	}

	// Test RetrieveSession
	retrieved, err := store.RetrieveSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}
	if retrieved.ID != sess.ID {
		t.Errorf("Expected session ID %s, got %s", sess.ID, retrieved.ID)
	}

	// Test ListSessions
	sessions, err := store.ListSessions(ctx, sess.UserID)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Test DeleteSession
	err = store.DeleteSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify deletion
	exists, err = store.Exists(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Failed to check existence after delete: %v", err)
	}
	if exists {
		t.Errorf("Session should not exist after deletion")
	}
}

func TestInMemorySessionStoreMultipleUsers(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	// Create sessions for different users
	user1Sessions := []*session.Session{
		{
			ID:            "sess-1-1",
			UserID:        "user-1",
			OrgID:         "org-1",
			WorkspacePath: "/tmp/ws1",
			MCPClients:    make(map[string]*mcp.Client),
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
		{
			ID:            "sess-1-2",
			UserID:        "user-1",
			OrgID:         "org-1",
			WorkspacePath: "/tmp/ws2",
			MCPClients:    make(map[string]*mcp.Client),
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
	}

	user2Sessions := []*session.Session{
		{
			ID:            "sess-2-1",
			UserID:        "user-2",
			OrgID:         "org-2",
			WorkspacePath: "/tmp/ws3",
			MCPClients:    make(map[string]*mcp.Client),
			CreatedAt:     time.Now(),
			LastActiveAt:  time.Now(),
		},
	}

	// Store all sessions
	for _, sess := range user1Sessions {
		if err := store.StoreSession(ctx, sess); err != nil {
			t.Fatalf("Failed to store session: %v", err)
		}
	}
	for _, sess := range user2Sessions {
		if err := store.StoreSession(ctx, sess); err != nil {
			t.Fatalf("Failed to store session: %v", err)
		}
	}

	// List sessions for user 1
	sessions, err := store.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions for user-1, got %d", len(sessions))
	}

	// List sessions for user 2
	sessions, err = store.ListSessions(ctx, "user-2")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for user-2, got %d", len(sessions))
	}
}

func TestSessionStoreKeyFunctions(t *testing.T) {
	// Test sessionKey function
	sessionID := "test-session-123"
	expectedKey := "session:test-session-123"
	actualKey := sessionKey(sessionID)
	if actualKey != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, actualKey)
	}

	// Test userSessionsKey function
	userID := "user-456"
	expectedUserKey := "user_sessions:user-456"
	actualUserKey := userSessionsKey(userID)
	if actualUserKey != expectedUserKey {
		t.Errorf("Expected key %s, got %s", expectedUserKey, actualUserKey)
	}
}

package session

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/mcp"
)

// mockPromptComposer implements PromptComposer for testing
type mockPromptComposer struct {
	prompt string
	err    error
}

func (m *mockPromptComposer) ComposeSystemPrompt(userID, orgID string, userVariables map[string]any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.prompt, nil
}

// mockAuditLogger implements AuditLogger for testing
type mockAuditLogger struct {
	events []map[string]any
}

func (m *mockAuditLogger) LogSessionEvent(ctx context.Context, userID, orgID, eventType, sessionID string, details map[string]any) {
	m.events = append(m.events, map[string]any{
		"user_id":    userID,
		"org_id":     orgID,
		"event_type": eventType,
		"session_id": sessionID,
		"details":    details,
	})
}

func setupTestManager(t *testing.T) (*SessionManagerV2, string) {
	t.Helper()

	// Create temp directory for workspace
	tempDir, err := os.MkdirTemp("", "session-manager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create session manager
	sm := NewSessionManagerV2(tempDir, 10)

	return sm, tempDir
}

func cleanupTestManager(t *testing.T, tempDir string) {
	t.Helper()
	if err := os.RemoveAll(tempDir); err != nil {
		t.Logf("warning: failed to cleanup temp dir %s: %v", tempDir, err)
	}
}

func TestNewSessionManagerV2(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	if sm == nil {
		t.Fatal("NewSessionManagerV2 returned nil")
	}

	if sm.workspaceRoot != tempDir {
		t.Errorf("expected workspaceRoot %s, got %s", tempDir, sm.workspaceRoot)
	}

	if sm.maxConcurrent != 10 {
		t.Errorf("expected maxConcurrent 10, got %d", sm.maxConcurrent)
	}
}

func TestNewSessionManagerV2WithLogger(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-manager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer cleanupTestManager(t, tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sm := NewSessionManagerV2WithLogger(tempDir, 5, logger)

	if sm == nil {
		t.Fatal("NewSessionManagerV2WithLogger returned nil")
	}

	if sm.maxConcurrent != 5 {
		t.Errorf("expected maxConcurrent 5, got %d", sm.maxConcurrent)
	}
}

func TestCreateSession(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	session, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("CreateSession returned nil session")
	}

	if session.ID == "" {
		t.Error("session ID is empty")
	}

	if session.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, session.UserID)
	}

	if session.OrgID != orgID {
		t.Errorf("expected OrgID %s, got %s", orgID, session.OrgID)
	}

	// Check workspace path
	expectedPath := filepath.Join(tempDir, userID, session.ID)
	if session.WorkspacePath != expectedPath {
		t.Errorf("expected WorkspacePath %s, got %s", expectedPath, session.WorkspacePath)
	}

	// Verify workspace directory exists with correct permissions
	info, err := os.Stat(session.WorkspacePath)
	if err != nil {
		t.Fatalf("workspace directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("workspace path is not a directory")
	}

	// Check permissions (0700)
	mode := info.Mode().Perm()
	if mode != 0700 {
		t.Errorf("expected permissions 0700, got %o", mode)
	}

	// Check timestamps
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if session.LastActiveAt.IsZero() {
		t.Error("LastActiveAt is zero")
	}

	// Check MCP clients initialized
	if session.MCPClients == nil {
		t.Error("MCPClients map is nil")
	}
}

func TestCreateSessionWithInvalidInputs(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()

	tests := []struct {
		name    string
		userID  string
		orgID   string
		wantErr error
	}{
		{
			name:    "empty user ID",
			userID:  "",
			orgID:   "org-123",
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "empty org ID",
			userID:  "user-123",
			orgID:   "",
			wantErr: ErrInvalidOrgID,
		},
		{
			name:    "both empty",
			userID:  "",
			orgID:   "",
			wantErr: ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sm.CreateSession(ctx, tt.userID, tt.orgID)
			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create a session
	session, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Retrieve the session
	retrieved, err := sm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("expected session ID %s, got %s", session.ID, retrieved.ID)
	}

	if retrieved.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, retrieved.UserID)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	_, err := sm.GetSession("nonexistent-session-id")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestCleanupSession(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create a session
	session, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify workspace exists
	if _, err := os.Stat(session.WorkspacePath); err != nil {
		t.Fatalf("workspace directory not found: %v", err)
	}

	// Cleanup the session
	if err := sm.CleanupSession(ctx, session.ID); err != nil {
		t.Fatalf("CleanupSession failed: %v", err)
	}

	// Verify session is removed
	_, err = sm.GetSession(session.ID)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after cleanup, got %v", err)
	}

	// Verify workspace is deleted
	if _, err := os.Stat(session.WorkspacePath); !os.IsNotExist(err) {
		t.Error("workspace directory still exists after cleanup")
	}
}

func TestCountUserSessions(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID1 := "user-123"
	userID2 := "user-456"
	orgID := "org-789"

	// Create sessions for user1
	for i := 0; i < 3; i++ {
		_, err := sm.CreateSession(ctx, userID1, orgID)
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	// Create sessions for user2
	for i := 0; i < 2; i++ {
		_, err := sm.CreateSession(ctx, userID2, orgID)
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	// Count sessions for user1
	count := sm.CountUserSessions(userID1)
	if count != 3 {
		t.Errorf("expected 3 sessions for user1, got %d", count)
	}

	// Count sessions for user2
	count = sm.CountUserSessions(userID2)
	if count != 2 {
		t.Errorf("expected 2 sessions for user2, got %d", count)
	}

	// Count total sessions
	totalCount := sm.CountAllSessions()
	if totalCount != 5 {
		t.Errorf("expected 5 total sessions, got %d", totalCount)
	}
}

func TestListSessions(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create multiple sessions
	sessionIDs := make(map[string]bool)
	for i := 0; i < 3; i++ {
		session, err := sm.CreateSession(ctx, userID, orgID)
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
		sessionIDs[session.ID] = true
	}

	// List sessions
	sessions := sm.ListSessions(userID)
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	// Verify all sessions are returned
	for _, session := range sessions {
		if !sessionIDs[session.ID] {
			t.Errorf("unexpected session ID %s in list", session.ID)
		}
	}
}

func TestMaxConcurrentSessions(t *testing.T) {
	// Create manager with limit of 2
	tempDir, err := os.MkdirTemp("", "session-manager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer cleanupTestManager(t, tempDir)

	sm := NewSessionManagerV2(tempDir, 2)
	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create 2 sessions (should succeed)
	for i := 0; i < 2; i++ {
		_, err := sm.CreateSession(ctx, userID, orgID)
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
	}

	// Try to create a third session (should fail)
	_, err = sm.CreateSession(ctx, userID, orgID)
	if err != ErrMaxSessionsReached {
		t.Errorf("expected ErrMaxSessionsReached, got %v", err)
	}
}

func TestSetPromptComposer(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	composer := &mockPromptComposer{
		prompt: "Test system prompt",
	}

	sm.SetPromptComposer(composer)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	session, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.SystemPrompt != "Test system prompt" {
		t.Errorf("expected system prompt 'Test system prompt', got '%s'", session.SystemPrompt)
	}
}

func TestSetAuditLogger(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	mockLogger := &mockAuditLogger{
		events: make([]map[string]any, 0),
	}

	sm.SetAuditLogger(mockLogger)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	_, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify audit event was logged
	if len(mockLogger.events) != 1 {
		t.Errorf("expected 1 audit event, got %d", len(mockLogger.events))
	}

	event := mockLogger.events[0]
	if event["event_type"] != "session_created" {
		t.Errorf("expected event_type 'session_created', got '%v'", event["event_type"])
	}
}

func TestSessionUpdateSystemPrompt(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	session, err := sm.CreateSession(ctx, "user-123", "org-456")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	newPrompt := "Updated system prompt"
	session.UpdateSystemPrompt(newPrompt)

	retrieved := session.GetSystemPrompt()
	if retrieved != newPrompt {
		t.Errorf("expected system prompt '%s', got '%s'", newPrompt, retrieved)
	}
}

func TestSessionGetWorkspacePath(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	session, err := sm.CreateSession(ctx, "user-123", "org-456")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	path := session.GetWorkspacePath()
	if path != session.WorkspacePath {
		t.Errorf("expected workspace path '%s', got '%s'", session.WorkspacePath, path)
	}
}

func TestSessionGetSessionInfo(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	session, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	info := session.GetSessionInfo()

	if info.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, info.ID)
	}

	if info.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, info.UserID)
	}

	if info.OrgID != orgID {
		t.Errorf("expected OrgID %s, got %s", orgID, info.OrgID)
	}

	if info.WorkspacePath != session.WorkspacePath {
		t.Errorf("expected WorkspacePath %s, got %s", session.WorkspacePath, info.WorkspacePath)
	}
}

func TestCleanupInactiveSessions(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create sessions
	session1, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	session2, err := sm.CreateSession(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Manually set session1 to be inactive for testing
	session1.mu.Lock()
	session1.LastActiveAt = time.Now().Add(-2 * time.Hour)
	session1.mu.Unlock()

	// Cleanup sessions inactive for more than 1 hour
	cleanedIDs := sm.CleanupInactiveSessions(ctx, 1*time.Hour)

	if len(cleanedIDs) != 1 {
		t.Errorf("expected 1 cleaned session, got %d", len(cleanedIDs))
	}

	if cleanedIDs[0] != session1.ID {
		t.Errorf("expected cleaned session ID %s, got %s", session1.ID, cleanedIDs[0])
	}

	// Verify session1 is removed
	_, err = sm.GetSession(session1.ID)
	if err != ErrSessionNotFound {
		t.Error("session1 should have been removed")
	}

	// Verify session2 still exists
	_, err = sm.GetSession(session2.ID)
	if err != nil {
		t.Errorf("session2 should still exist: %v", err)
	}
}

func TestAddMCPClient(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	session, err := sm.CreateSession(ctx, "user-123", "org-456")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create a mock MCP client (note: this will fail connection, but we're testing the add logic)
	client, err := mcp.NewClient("client-1", "Test Client", "stdio", "", nil, nil)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Note: This will fail because we don't have a real MCP server
	// In a real test, you'd mock the Connect method
	err = session.AddMCPClient(ctx, client)
	// We expect this to fail due to connection issues, but not due to session logic
	if err == nil {
		t.Log("AddMCPClient succeeded (unexpected if no MCP server running)")
	}
}

func TestConcurrentSessionOperations(t *testing.T) {
	sm, tempDir := setupTestManager(t)
	defer cleanupTestManager(t, tempDir)

	ctx := context.Background()
	userID := "user-123"
	orgID := "org-456"

	// Create multiple sessions concurrently
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := sm.CreateSession(ctx, userID, orgID)
			done <- err
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent CreateSession failed: %v", err)
		}
	}

	// Verify all sessions were created
	count := sm.CountUserSessions(userID)
	if count != numGoroutines {
		t.Errorf("expected %d sessions, got %d", numGoroutines, count)
	}
}

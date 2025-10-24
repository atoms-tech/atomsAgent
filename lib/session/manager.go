package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/logctx"
	"github.com/coder/agentapi/lib/mcp"
	"github.com/google/uuid"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")
	// ErrMaxSessionsReached is returned when the concurrent session limit is exceeded
	ErrMaxSessionsReached = errors.New("maximum concurrent sessions reached")
	// ErrWorkspaceCreation is returned when workspace directory creation fails
	ErrWorkspaceCreation = errors.New("failed to create workspace directory")
	// ErrInvalidUserID is returned when user ID is empty
	ErrInvalidUserID = errors.New("invalid user ID")
	// ErrInvalidOrgID is returned when org ID is empty
	ErrInvalidOrgID = errors.New("invalid org ID")
)

// SessionManagerV2 manages isolated user sessions with thread-safe operations
// This is the enhanced version with MCP client management and proper isolation
type SessionManagerV2 struct {
	// sessions stores active sessions using sync.Map for concurrent access
	sessions sync.Map

	// workspaceRoot is the base directory for all session workspaces
	workspaceRoot string

	// maxConcurrent limits the total number of concurrent sessions
	maxConcurrent int

	// mu protects session count operations
	mu sync.RWMutex

	// promptComposer composes system prompts for sessions
	promptComposer PromptComposer

	// auditLogger logs session lifecycle events
	auditLogger AuditLogger
}

// Session represents an isolated user session with MCP clients
type Session struct {
	// ID is the unique session identifier (UUID)
	ID string

	// UserID identifies the user who owns this session
	UserID string

	// OrgID identifies the organization context
	OrgID string

	// WorkspacePath is the isolated workspace directory for this session
	WorkspacePath string

	// MCPClients stores initialized MCP clients by client ID
	MCPClients map[string]*mcp.Client

	// SystemPrompt is the composed system prompt for this session
	SystemPrompt string

	// CreatedAt is the session creation timestamp
	CreatedAt time.Time

	// LastActiveAt tracks the last activity timestamp
	LastActiveAt time.Time

	// mu protects session fields during updates
	mu sync.RWMutex
}

// PromptComposer defines the interface for composing system prompts
type PromptComposer interface {
	ComposeSystemPrompt(userID, orgID string, userVariables map[string]any) (string, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogSessionEvent(ctx context.Context, userID, orgID, eventType, sessionID string, details map[string]any)
}

// defaultAuditLogger provides a simple slog-based audit logger
type defaultAuditLogger struct {
	logger *slog.Logger
}

// LogSessionEvent logs a session event using slog
func (l *defaultAuditLogger) LogSessionEvent(ctx context.Context, userID, orgID, eventType, sessionID string, details map[string]any) {
	attrs := []any{
		slog.String("user_id", userID),
		slog.String("org_id", orgID),
		slog.String("event_type", eventType),
		slog.String("session_id", sessionID),
	}

	if details != nil {
		for k, v := range details {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	l.logger.InfoContext(ctx, "session_audit_event", attrs...)
}

// NewSessionManagerV2 creates a new enhanced session manager with the specified configuration
func NewSessionManagerV2(workspaceRoot string, maxConcurrent int) *SessionManagerV2 {
	return &SessionManagerV2{
		workspaceRoot: workspaceRoot,
		maxConcurrent: maxConcurrent,
		auditLogger: &defaultAuditLogger{
			logger: slog.Default(),
		},
	}
}

// NewSessionManagerV2WithLogger creates a session manager with a custom logger
func NewSessionManagerV2WithLogger(workspaceRoot string, maxConcurrent int, logger *slog.Logger) *SessionManagerV2 {
	sm := NewSessionManagerV2(workspaceRoot, maxConcurrent)
	sm.auditLogger = &defaultAuditLogger{logger: logger}
	return sm
}

// SetPromptComposer sets a custom prompt composer for the session manager
func (sm *SessionManagerV2) SetPromptComposer(composer PromptComposer) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.promptComposer = composer
}

// SetAuditLogger sets a custom audit logger for the session manager
func (sm *SessionManagerV2) SetAuditLogger(logger AuditLogger) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.auditLogger = logger
}

// CreateSession creates a new isolated session for a user
func (sm *SessionManagerV2) CreateSession(ctx context.Context, userID, orgID string) (*Session, error) {
	// Validate inputs
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if orgID == "" {
		return nil, ErrInvalidOrgID
	}

	// Check concurrent session limit
	if err := sm.checkSessionLimit(); err != nil {
		sm.auditLogger.LogSessionEvent(ctx, userID, orgID, "session_create_failed", "", map[string]any{
			"error":  err.Error(),
			"reason": "max_sessions_reached",
		})
		return nil, err
	}

	// Generate unique session ID using UUID
	sessionID := uuid.New().String()

	// Create isolated workspace directory with restrictive permissions (0700)
	workspacePath := filepath.Join(sm.workspaceRoot, userID, sessionID)
	if err := os.MkdirAll(workspacePath, 0700); err != nil {
		sm.auditLogger.LogSessionEvent(ctx, userID, orgID, "session_create_failed", sessionID, map[string]any{
			"error":  err.Error(),
			"reason": "workspace_creation_failed",
			"path":   workspacePath,
		})
		return nil, fmt.Errorf("%w: %v", ErrWorkspaceCreation, err)
	}

	// Compose system prompt for this session
	systemPrompt, err := sm.composeSystemPrompt(ctx, userID, orgID)
	if err != nil {
		// Log the error but don't fail session creation
		logger := sm.getLogger(ctx)
		logger.WarnContext(ctx, "failed to compose system prompt",
			slog.String("session_id", sessionID),
			slog.String("error", err.Error()),
		)
		systemPrompt = "" // Use empty prompt as fallback
	}

	// Create session object
	now := time.Now()
	session := &Session{
		ID:            sessionID,
		UserID:        userID,
		OrgID:         orgID,
		WorkspacePath: workspacePath,
		MCPClients:    make(map[string]*mcp.Client),
		SystemPrompt:  systemPrompt,
		CreatedAt:     now,
		LastActiveAt:  now,
	}

	// Store session in concurrent-safe map
	sm.sessions.Store(sessionID, session)

	// Log successful session creation
	sm.auditLogger.LogSessionEvent(ctx, userID, orgID, "session_created", sessionID, map[string]any{
		"workspace_path": workspacePath,
		"created_at":     now,
	})

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManagerV2) GetSession(sessionID string) (*Session, error) {
	value, ok := sm.sessions.Load(sessionID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	session, ok := value.(*Session)
	if !ok {
		return nil, fmt.Errorf("invalid session type in storage")
	}

	// Update last active timestamp
	session.mu.Lock()
	session.LastActiveAt = time.Now()
	session.mu.Unlock()

	return session, nil
}

// CleanupSession removes a session and cleans up all its resources
func (sm *SessionManagerV2) CleanupSession(ctx context.Context, sessionID string) error {
	// Load and remove session atomically
	value, loaded := sm.sessions.LoadAndDelete(sessionID)
	if !loaded {
		return ErrSessionNotFound
	}

	session, ok := value.(*Session)
	if !ok {
		return fmt.Errorf("invalid session type in storage")
	}

	// Cleanup MCP clients
	session.mu.Lock()
	for clientID, client := range session.MCPClients {
		if err := client.Disconnect(); err != nil {
			logger := sm.getLogger(ctx)
			logger.WarnContext(ctx, "failed to disconnect MCP client",
				slog.String("session_id", sessionID),
				slog.String("client_id", clientID),
				slog.String("error", err.Error()),
			)
		}

		// Close the underlying client
		if err := client.Close(); err != nil {
			logger := sm.getLogger(ctx)
			logger.WarnContext(ctx, "failed to close MCP client",
				slog.String("session_id", sessionID),
				slog.String("client_id", clientID),
				slog.String("error", err.Error()),
			)
		}
	}
	session.MCPClients = nil
	session.mu.Unlock()

	// Cleanup workspace directory
	if err := os.RemoveAll(session.WorkspacePath); err != nil {
		// Log error but don't fail the cleanup
		logger := sm.getLogger(ctx)
		logger.WarnContext(ctx, "failed to remove workspace directory",
			slog.String("session_id", sessionID),
			slog.String("workspace_path", session.WorkspacePath),
			slog.String("error", err.Error()),
		)
	}

	// Log successful cleanup
	sm.auditLogger.LogSessionEvent(ctx, session.UserID, session.OrgID, "session_cleaned_up", sessionID, map[string]any{
		"workspace_path":           session.WorkspacePath,
		"session_duration_seconds": time.Since(session.CreatedAt).Seconds(),
	})

	return nil
}

// CountUserSessions returns the number of active sessions for a user
func (sm *SessionManagerV2) CountUserSessions(userID string) int {
	count := 0
	sm.sessions.Range(func(key, value any) bool {
		if session, ok := value.(*Session); ok {
			if session.UserID == userID {
				count++
			}
		}
		return true
	})
	return count
}

// CountAllSessions returns the total number of active sessions
func (sm *SessionManagerV2) CountAllSessions() int {
	count := 0
	sm.sessions.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// ListSessions returns all active sessions for a user
func (sm *SessionManagerV2) ListSessions(userID string) []*Session {
	var sessions []*Session

	sm.sessions.Range(func(key, value any) bool {
		if session, ok := value.(*Session); ok {
			if session.UserID == userID {
				sessions = append(sessions, session)
			}
		}
		return true
	})

	return sessions
}

// checkSessionLimit verifies that creating a new session won't exceed the limit
func (sm *SessionManagerV2) checkSessionLimit() error {
	if sm.maxConcurrent <= 0 {
		return nil // No limit enforced
	}

	count := sm.CountAllSessions()
	if count >= sm.maxConcurrent {
		return ErrMaxSessionsReached
	}

	return nil
}

// composeSystemPrompt generates the system prompt for a session
func (sm *SessionManagerV2) composeSystemPrompt(ctx context.Context, userID, orgID string) (string, error) {
	sm.mu.RLock()
	composer := sm.promptComposer
	sm.mu.RUnlock()

	if composer == nil {
		return "", nil
	}

	// Use empty user variables for now; can be extended later
	userVariables := make(map[string]any)

	return composer.ComposeSystemPrompt(userID, orgID, userVariables)
}

// getLogger retrieves logger from context or returns default
func (sm *SessionManagerV2) getLogger(ctx context.Context) *slog.Logger {
	defer func() {
		if r := recover(); r != nil {
			// If context has no logger, the panic is recovered
		}
	}()

	logger := logctx.From(ctx)
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// AddMCPClient adds an MCP client to the session
func (s *Session) AddMCPClient(ctx context.Context, client *mcp.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.MCPClients == nil {
		s.MCPClients = make(map[string]*mcp.Client)
	}

	// Check if client already exists
	if _, exists := s.MCPClients[client.ID]; exists {
		return fmt.Errorf("MCP client with ID %s already exists in session", client.ID)
	}

	// Connect the client
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect MCP client: %w", err)
	}

	s.MCPClients[client.ID] = client
	s.LastActiveAt = time.Now()

	return nil
}

// RemoveMCPClient removes an MCP client from the session
func (s *Session) RemoveMCPClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.MCPClients[clientID]
	if !exists {
		return fmt.Errorf("MCP client with ID %s not found in session", clientID)
	}

	// Disconnect the client
	if err := client.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect MCP client: %w", err)
	}

	// Close the client
	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to close MCP client: %w", err)
	}

	delete(s.MCPClients, clientID)
	s.LastActiveAt = time.Now()

	return nil
}

// GetMCPClient retrieves an MCP client by ID
func (s *Session) GetMCPClient(clientID string) (*mcp.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.MCPClients[clientID]
	if !exists {
		return nil, fmt.Errorf("MCP client with ID %s not found in session", clientID)
	}

	return client, nil
}

// ListMCPClients returns all MCP clients in the session
func (s *Session) ListMCPClients() []*mcp.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*mcp.Client, 0, len(s.MCPClients))
	for _, client := range s.MCPClients {
		clients = append(clients, client)
	}

	return clients
}

// UpdateSystemPrompt updates the session's system prompt
func (s *Session) UpdateSystemPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SystemPrompt = prompt
	s.LastActiveAt = time.Now()
}

// GetSystemPrompt returns the session's system prompt
func (s *Session) GetSystemPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.SystemPrompt
}

// GetWorkspacePath returns the session's workspace path
func (s *Session) GetWorkspacePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.WorkspacePath
}

// GetSessionInfo returns a snapshot of session information
func (s *Session) GetSessionInfo() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionInfo{
		ID:            s.ID,
		UserID:        s.UserID,
		OrgID:         s.OrgID,
		WorkspacePath: s.WorkspacePath,
		MCPClientIDs:  s.getMCPClientIDs(),
		CreatedAt:     s.CreatedAt,
		LastActiveAt:  s.LastActiveAt,
	}
}

// SessionInfo provides a thread-safe snapshot of session data
type SessionInfo struct {
	ID            string
	UserID        string
	OrgID         string
	WorkspacePath string
	MCPClientIDs  []string
	CreatedAt     time.Time
	LastActiveAt  time.Time
}

// getMCPClientIDs returns a list of MCP client IDs (must be called with lock held)
func (s *Session) getMCPClientIDs() []string {
	ids := make([]string, 0, len(s.MCPClients))
	for id := range s.MCPClients {
		ids = append(ids, id)
	}
	return ids
}

// CleanupInactiveSessions removes sessions that have been inactive for the specified duration
func (sm *SessionManagerV2) CleanupInactiveSessions(ctx context.Context, inactivityThreshold time.Duration) []string {
	var cleanedSessions []string
	now := time.Now()

	sm.sessions.Range(func(key, value any) bool {
		sessionID, ok := key.(string)
		if !ok {
			return true
		}

		session, ok := value.(*Session)
		if !ok {
			return true
		}

		session.mu.RLock()
		lastActive := session.LastActiveAt
		session.mu.RUnlock()

		if now.Sub(lastActive) > inactivityThreshold {
			if err := sm.CleanupSession(ctx, sessionID); err != nil {
				logger := sm.getLogger(ctx)
				logger.WarnContext(ctx, "failed to cleanup inactive session",
					slog.String("session_id", sessionID),
					slog.String("error", err.Error()),
				)
			} else {
				cleanedSessions = append(cleanedSessions, sessionID)
			}
		}

		return true
	})

	return cleanedSessions
}

package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/msgfmt"
	"github.com/coder/agentapi/lib/termexec"
)

// UserSession represents an isolated user session
type UserSession struct {
	ID           string
	UserID       string
	OrgID        string
	AgentType    msgfmt.AgentType
	Workspace    string
	Process      *termexec.Process
	Config       *SessionConfig
	MCPs         []MCPConfig
	SystemPrompt string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Status       SessionStatus
	mutex        sync.RWMutex
}

type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusInactive   SessionStatus = "inactive"
	SessionStatusTerminated SessionStatus = "terminated"
)

type SessionConfig struct {
	AgentType     msgfmt.AgentType  `json:"agentType"`
	TermWidth     uint16            `json:"termWidth"`
	TermHeight    uint16            `json:"termHeight"`
	InitialPrompt string            `json:"initialPrompt,omitempty"`
	Environment   map[string]string `json:"environment"`
	Credentials   map[string]string `json:"credentials"`
}

type MCPConfig struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"` // http, sse, stdio
	Endpoint string            `json:"endpoint"`
	Config   map[string]any    `json:"config"`
	Auth     map[string]string `json:"auth"`
}

// SessionManager manages user sessions with isolation
type SessionManager struct {
	sessions map[string]*UserSession
	mutex    sync.RWMutex
	baseDir  string
}

// NewSessionManager creates a new session manager
func NewSessionManager(baseDir string) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*UserSession),
		baseDir:  baseDir,
	}
}

// CreateSession creates a new isolated user session
func (sm *SessionManager) CreateSession(ctx context.Context, userID, orgID string, agentType msgfmt.AgentType, config *SessionConfig) (*UserSession, error) {
	sessionID := generateSessionID()
	workspace := filepath.Join(sm.baseDir, "workspaces", userID, sessionID)

	// Create isolated workspace directory
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	session := &UserSession{
		ID:        sessionID,
		UserID:    userID,
		OrgID:     orgID,
		AgentType: agentType,
		Workspace: workspace,
		Config:    config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    SessionStatusActive,
	}

	sm.mutex.Lock()
	sm.sessions[sessionID] = session
	sm.mutex.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*UserSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// ListUserSessions returns all sessions for a user
func (sm *SessionManager) ListUserSessions(userID string) []*UserSession {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var userSessions []*UserSession
	for _, session := range sm.sessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

// TerminateSession terminates a session and cleans up resources
func (sm *SessionManager) TerminateSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Terminate the process if running
	if session.Process != nil {
		// Use SIGTERM to gracefully terminate the process
		_ = session.Process.Signal(os.Kill)
	}

	// Update status
	session.Status = SessionStatusTerminated
	session.UpdatedAt = time.Now()

	// Clean up workspace directory
	if err := os.RemoveAll(session.Workspace); err != nil {
		// Log error but don't fail the termination
		fmt.Printf("Warning: failed to clean up workspace %s: %v\n", session.Workspace, err)
	}

	// Remove from active sessions
	delete(sm.sessions, sessionID)

	return nil
}

// SetMCPs sets MCP configurations for a session
func (s *UserSession) SetMCPs(mcps []MCPConfig) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.MCPs = mcps
	s.UpdatedAt = time.Now()
}

// SetSystemPrompt sets the system prompt for a session
func (s *UserSession) SetSystemPrompt(prompt string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.SystemPrompt = prompt
	s.UpdatedAt = time.Now()
}

// GetWorkspacePath returns the isolated workspace path for this session
func (s *UserSession) GetWorkspacePath() string {
	return s.Workspace
}

// IsActive checks if the session is active
func (s *UserSession) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.Status == SessionStatusActive
}

// UpdateStatus updates the session status
func (s *UserSession) UpdateStatus(status SessionStatus) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Status = status
	s.UpdatedAt = time.Now()
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

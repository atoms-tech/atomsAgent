package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/session"
)

const (
	// SessionKeyPrefix is the prefix for all session keys in Redis
	SessionKeyPrefix = "session:"

	// UserSessionsKeyPrefix is the prefix for user session index keys
	UserSessionsKeyPrefix = "user_sessions:"

	// DefaultSessionTTL is the default time-to-live for sessions (24 hours)
	DefaultSessionTTL = 24 * time.Hour
)

var (
	// ErrSessionNotFound is returned when a session is not found in Redis
	ErrSessionNotFound = fmt.Errorf("session not found in Redis")

	// ErrInvalidSession is returned when session data is invalid
	ErrInvalidSession = fmt.Errorf("invalid session data")
)

// SessionStore defines the interface for session persistence
type SessionStore interface {
	// StoreSession stores a session in Redis
	StoreSession(ctx context.Context, sess *session.Session) error

	// RetrieveSession retrieves a session from Redis by ID
	RetrieveSession(ctx context.Context, sessionID string) (*session.Session, error)

	// UpdateSession updates an existing session in Redis
	UpdateSession(ctx context.Context, sess *session.Session) error

	// DeleteSession deletes a session from Redis
	DeleteSession(ctx context.Context, sessionID string) error

	// ListSessions lists all sessions for a user
	ListSessions(ctx context.Context, userID string) ([]*session.Session, error)

	// Exists checks if a session exists in Redis
	Exists(ctx context.Context, sessionID string) (bool, error)

	// SetTTL updates the TTL for a session
	SetTTL(ctx context.Context, sessionID string, ttl time.Duration) error

	// Health checks the health of the Redis connection
	Health() error

	// Close closes the Redis connection
	Close() error
}

// RedisSessionStore implements SessionStore using Redis for persistence
type RedisSessionStore struct {
	client *RedisClient
	ttl    time.Duration
	mu     sync.RWMutex
}

// sessionData represents the serializable session data stored in Redis
type sessionData struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	OrgID         string                 `json:"org_id"`
	WorkspacePath string                 `json:"workspace_path"`
	MCPClientIDs  []string               `json:"mcp_client_ids"`
	SystemPrompt  string                 `json:"system_prompt"`
	CreatedAt     time.Time              `json:"created_at"`
	LastActiveAt  time.Time              `json:"last_active_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewSessionStore creates a new Redis-backed session store
func NewSessionStore(client *RedisClient, ttl time.Duration) *RedisSessionStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}

	return &RedisSessionStore{
		client: client,
		ttl:    ttl,
	}
}

// StoreSession stores a session in Redis with TTL
func (s *RedisSessionStore) StoreSession(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return ErrInvalidSession
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert session to serializable format
	data := s.sessionToData(sess)

	// Serialize to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store in Redis with TTL
	key := sessionKey(sess.ID)
	if err := s.client.Set(ctx, key, string(jsonData), s.ttl); err != nil {
		return fmt.Errorf("failed to store session in Redis: %w", err)
	}

	// Add to user's session index
	if err := s.addToUserIndex(ctx, sess.UserID, sess.ID); err != nil {
		// Log error but don't fail the operation
		// The session is stored, just the index update failed
		return fmt.Errorf("session stored but failed to update user index: %w", err)
	}

	return nil
}

// RetrieveSession retrieves a session from Redis by ID
func (s *RedisSessionStore) RetrieveSession(ctx context.Context, sessionID string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey(sessionID)

	// Get from Redis
	jsonData, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	if jsonData == "" {
		return nil, ErrSessionNotFound
	}

	// Deserialize
	var data sessionData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Convert to session object
	sess := s.dataToSession(&data)

	// Update TTL on access (sliding expiration)
	if err := s.client.Set(ctx, key, jsonData, s.ttl); err != nil {
		// Don't fail on TTL update error
		// The session is still retrieved successfully
	}

	return sess, nil
}

// UpdateSession updates an existing session in Redis
func (s *RedisSessionStore) UpdateSession(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return ErrInvalidSession
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if session exists
	exists, err := s.Exists(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if !exists {
		return ErrSessionNotFound
	}

	// Store updated session (same as StoreSession)
	return s.StoreSession(ctx, sess)
}

// DeleteSession deletes a session from Redis
func (s *RedisSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get session to find userID for index cleanup
	sess, err := s.RetrieveSession(ctx, sessionID)
	if err != nil {
		// If session doesn't exist, consider it a success
		if err == ErrSessionNotFound {
			return nil
		}
		return fmt.Errorf("failed to retrieve session for deletion: %w", err)
	}

	// Delete from Redis
	key := sessionKey(sessionID)
	if err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	// Remove from user's session index
	if err := s.removeFromUserIndex(ctx, sess.UserID, sessionID); err != nil {
		// Log error but don't fail the operation
		// The session is deleted, just the index update failed
	}

	return nil
}

// ListSessions lists all sessions for a user
func (s *RedisSessionStore) ListSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get session IDs from user index
	indexKey := userSessionsKey(userID)
	indexData, err := s.client.Get(ctx, indexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user session index: %w", err)
	}

	if indexData == "" {
		// No sessions for this user
		return []*session.Session{}, nil
	}

	// Parse session IDs
	var sessionIDs []string
	if err := json.Unmarshal([]byte(indexData), &sessionIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session index: %w", err)
	}

	// Retrieve each session
	sessions := make([]*session.Session, 0, len(sessionIDs))
	validSessionIDs := make([]string, 0, len(sessionIDs))

	for _, sessionID := range sessionIDs {
		sess, err := s.RetrieveSession(ctx, sessionID)
		if err != nil {
			// Skip sessions that can't be retrieved (may have expired)
			continue
		}
		sessions = append(sessions, sess)
		validSessionIDs = append(validSessionIDs, sessionID)
	}

	// Update index with valid sessions only
	if len(validSessionIDs) != len(sessionIDs) {
		jsonData, _ := json.Marshal(validSessionIDs)
		s.client.Set(ctx, indexKey, string(jsonData), s.ttl)
	}

	return sessions, nil
}

// Exists checks if a session exists in Redis
func (s *RedisSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey(sessionID)
	return s.client.Exists(ctx, key)
}

// SetTTL updates the TTL for a session
func (s *RedisSessionStore) SetTTL(ctx context.Context, sessionID string, ttl time.Duration) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get current session data
	key := sessionKey(sessionID)
	jsonData, err := s.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if jsonData == "" {
		return ErrSessionNotFound
	}

	// Re-set with new TTL
	return s.client.Set(ctx, key, jsonData, ttl)
}

// Health checks the health of the Redis connection
func (s *RedisSessionStore) Health() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.client.Health()
}

// Close closes the Redis connection
func (s *RedisSessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.client.Close()
}

// GetTTL returns the default TTL for sessions
func (s *RedisSessionStore) GetTTL() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ttl
}

// SetDefaultTTL updates the default TTL for new sessions
func (s *RedisSessionStore) SetDefaultTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ttl > 0 {
		s.ttl = ttl
	}
}

// BatchStoreSessions stores multiple sessions in a single operation
func (s *RedisSessionStore) BatchStoreSession(ctx context.Context, sessions []*session.Session) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Store each session individually
	// In a production environment with pipeline support, this could be optimized
	for _, sess := range sessions {
		if err := s.StoreSession(ctx, sess); err != nil {
			return fmt.Errorf("failed to store session %s: %w", sess.ID, err)
		}
	}

	return nil
}

// BatchDeleteSessions deletes multiple sessions in a single operation
func (s *RedisSessionStore) BatchDeleteSessions(ctx context.Context, sessionIDs []string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Delete each session individually
	// In a production environment with pipeline support, this could be optimized
	for _, sessionID := range sessionIDs {
		if err := s.DeleteSession(ctx, sessionID); err != nil {
			return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
		}
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions for a user
// This is useful for cleanup tasks
func (s *RedisSessionStore) CleanupExpiredSessions(ctx context.Context, userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// List all sessions for user
	sessions, err := s.ListSessions(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	// The ListSessions method already filters out expired sessions
	// and updates the index, so we just return the count of removed sessions

	indexKey := userSessionsKey(userID)
	indexData, _ := s.client.Get(ctx, indexKey)

	var oldSessionIDs []string
	if indexData != "" {
		json.Unmarshal([]byte(indexData), &oldSessionIDs)
	}

	removed := len(oldSessionIDs) - len(sessions)
	if removed < 0 {
		removed = 0
	}

	return removed, nil
}

// Helper functions

func sessionKey(sessionID string) string {
	return SessionKeyPrefix + sessionID
}

func userSessionsKey(userID string) string {
	return UserSessionsKeyPrefix + userID
}

func (s *RedisSessionStore) sessionToData(sess *session.Session) *sessionData {
	info := sess.GetSessionInfo()

	return &sessionData{
		ID:            info.ID,
		UserID:        info.UserID,
		OrgID:         info.OrgID,
		WorkspacePath: info.WorkspacePath,
		MCPClientIDs:  info.MCPClientIDs,
		SystemPrompt:  sess.GetSystemPrompt(),
		CreatedAt:     info.CreatedAt,
		LastActiveAt:  info.LastActiveAt,
	}
}

func (s *RedisSessionStore) dataToSession(data *sessionData) *session.Session {
	// Note: This creates a minimal session object from Redis data
	// MCP clients are not restored from Redis - they need to be reconnected
	sess := &session.Session{
		ID:            data.ID,
		UserID:        data.UserID,
		OrgID:         data.OrgID,
		WorkspacePath: data.WorkspacePath,
		MCPClients:    make(map[string]*mcp.Client),
		SystemPrompt:  data.SystemPrompt,
		CreatedAt:     data.CreatedAt,
		LastActiveAt:  data.LastActiveAt,
	}

	return sess
}

func (s *RedisSessionStore) addToUserIndex(ctx context.Context, userID, sessionID string) error {
	indexKey := userSessionsKey(userID)

	// Get current index
	indexData, err := s.client.Get(ctx, indexKey)
	if err != nil {
		return err
	}

	var sessionIDs []string
	if indexData != "" {
		if err := json.Unmarshal([]byte(indexData), &sessionIDs); err != nil {
			return err
		}
	}

	// Add session ID if not already present
	found := false
	for _, id := range sessionIDs {
		if id == sessionID {
			found = true
			break
		}
	}

	if !found {
		sessionIDs = append(sessionIDs, sessionID)
	}

	// Save updated index
	jsonData, err := json.Marshal(sessionIDs)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, indexKey, string(jsonData), s.ttl)
}

func (s *RedisSessionStore) removeFromUserIndex(ctx context.Context, userID, sessionID string) error {
	indexKey := userSessionsKey(userID)

	// Get current index
	indexData, err := s.client.Get(ctx, indexKey)
	if err != nil {
		return err
	}

	if indexData == "" {
		return nil // Index doesn't exist
	}

	var sessionIDs []string
	if err := json.Unmarshal([]byte(indexData), &sessionIDs); err != nil {
		return err
	}

	// Remove session ID
	filtered := make([]string, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}

	// Save updated index
	if len(filtered) == 0 {
		// Delete index if no sessions left
		return s.client.Delete(ctx, indexKey)
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, indexKey, string(jsonData), s.ttl)
}

// InMemorySessionStore provides a fallback in-memory implementation
type InMemorySessionStore struct {
	sessions  map[string]*session.Session
	userIndex map[string][]string
	mu        sync.RWMutex
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions:  make(map[string]*session.Session),
		userIndex: make(map[string][]string),
	}
}

func (s *InMemorySessionStore) StoreSession(ctx context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sess.ID] = sess

	// Update user index
	userSessions := s.userIndex[sess.UserID]
	found := false
	for _, id := range userSessions {
		if id == sess.ID {
			found = true
			break
		}
	}
	if !found {
		s.userIndex[sess.UserID] = append(userSessions, sess.ID)
	}

	return nil
}

func (s *InMemorySessionStore) RetrieveSession(ctx context.Context, sessionID string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return sess, nil
}

func (s *InMemorySessionStore) UpdateSession(ctx context.Context, sess *session.Session) error {
	return s.StoreSession(ctx, sess)
}

func (s *InMemorySessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, exists := s.sessions[sessionID]
	if !exists {
		return nil
	}

	delete(s.sessions, sessionID)

	// Update user index
	userSessions := s.userIndex[sess.UserID]
	filtered := make([]string, 0, len(userSessions))
	for _, id := range userSessions {
		if id != sessionID {
			filtered = append(filtered, id)
		}
	}
	s.userIndex[sess.UserID] = filtered

	return nil
}

func (s *InMemorySessionStore) ListSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs := s.userIndex[userID]
	sessions := make([]*session.Session, 0, len(sessionIDs))

	for _, id := range sessionIDs {
		if sess, exists := s.sessions[id]; exists {
			sessions = append(sessions, sess)
		}
	}

	return sessions, nil
}

func (s *InMemorySessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.sessions[sessionID]
	return exists, nil
}

func (s *InMemorySessionStore) SetTTL(ctx context.Context, sessionID string, ttl time.Duration) error {
	// No-op for in-memory store
	return nil
}

func (s *InMemorySessionStore) Health() error {
	return nil // Always healthy
}

func (s *InMemorySessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*session.Session)
	s.userIndex = make(map[string][]string)

	return nil
}

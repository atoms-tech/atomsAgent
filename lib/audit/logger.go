package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common error types
var (
	ErrInvalidAction       = errors.New("audit: invalid action name")
	ErrInvalidResourceType = errors.New("audit: invalid resource type")
	ErrDatabaseWrite       = errors.New("audit: database write failed")
	ErrContextCanceled     = errors.New("audit: context canceled")
	ErrNilDatabase         = errors.New("audit: database connection is nil")
)

// Action constants for standardized audit actions
const (
	ActionCreated  = "created"
	ActionUpdated  = "updated"
	ActionDeleted  = "deleted"
	ActionAccessed = "accessed"
	ActionFailed   = "failed"
)

// ResourceType constants for standardized resource types
const (
	ResourceTypeSession = "session"
	ResourceTypeMCP     = "mcp"
	ResourceTypePrompt  = "prompt"
	ResourceTypeAPI     = "api"
	ResourceTypeAuth    = "auth"
	ResourceTypeConfig  = "config"
)

// Context keys for extracting metadata
type contextKey int

const (
	userIDKey contextKey = iota
	orgIDKey
	ipAddressKey
	userAgentKey
	requestIDKey
)

// AuditEntry represents a single audit log entry
// All fields are immutable after creation for compliance
type AuditEntry struct {
	ID           string                 `json:"id" db:"id"`
	Timestamp    time.Time              `json:"timestamp" db:"timestamp"`
	UserID       string                 `json:"user_id" db:"user_id"`
	OrgID        string                 `json:"org_id" db:"org_id"`
	Action       string                 `json:"action" db:"action"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   string                 `json:"resource_id" db:"resource_id"`
	Details      map[string]interface{} `json:"details" db:"details"`
	IPAddress    string                 `json:"ip_address" db:"ip_address"`
	UserAgent    string                 `json:"user_agent" db:"user_agent"`
	RequestID    string                 `json:"request_id,omitempty" db:"request_id"`
}

// AuditFilter represents filters for querying audit logs
type AuditFilter struct {
	UserID       string
	OrgID        string
	Action       string
	ResourceType string
	ResourceID   string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// AuditLogger handles audit logging operations with thread safety and batching
type AuditLogger struct {
	db         *sql.DB
	mu         sync.Mutex
	buffer     []*AuditEntry
	bufferSize int
	flushTimer *time.Timer
	flushDone  chan struct{}
	closed     bool
}

// NewAuditLogger creates a new audit logger with optional buffering
// bufferSize: 0 for immediate writes, >0 for batched writes
func NewAuditLogger(db *sql.DB, bufferSize int) (*AuditLogger, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	logger := &AuditLogger{
		db:         db,
		bufferSize: bufferSize,
		buffer:     make([]*AuditEntry, 0, bufferSize),
		flushDone:  make(chan struct{}),
	}

	// Initialize database schema
	if err := logger.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize audit schema: %w", err)
	}

	// Start periodic flush for buffered mode
	if bufferSize > 0 {
		logger.startPeriodicFlush(30 * time.Second)
	}

	return logger, nil
}

// initSchema creates the audit_logs table if it doesn't exist
func (al *AuditLogger) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL,
		user_id TEXT NOT NULL,
		org_id TEXT NOT NULL,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		details TEXT,
		ip_address TEXT,
		user_agent TEXT,
		request_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for common query patterns (SOC2 compliance)
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(org_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_composite ON audit_logs(org_id, timestamp DESC);
	`

	_, err := al.db.Exec(schema)
	return err
}

// Log creates an audit log entry without context
// Deprecated: Use LogWithContext for better metadata extraction
func (al *AuditLogger) Log(action, resourceType, resourceID string, details map[string]any) error {
	return al.LogWithContext(context.Background(), action, resourceType, resourceID, details)
}

// LogWithContext creates an audit log entry with context metadata
func (al *AuditLogger) LogWithContext(ctx context.Context, action, resourceType, resourceID string, details map[string]any) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ErrContextCanceled
	default:
	}

	// Validate inputs
	if err := validateAction(action); err != nil {
		return err
	}
	if err := validateResourceType(resourceType); err != nil {
		return err
	}

	// Extract metadata from context
	userID := extractUserID(ctx)
	orgID := extractOrgID(ctx)
	ipAddress := extractIPAddress(ctx)
	userAgent := extractUserAgent(ctx)
	requestID := extractRequestID(ctx)

	// Create audit entry
	entry := &AuditEntry{
		ID:           uuid.New().String(),
		Timestamp:    time.Now().UTC(),
		UserID:       userID,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
	}

	// Handle buffered vs immediate writes
	if al.bufferSize > 0 {
		return al.addToBuffer(entry)
	}

	return al.writeEntry(entry)
}

// addToBuffer adds entry to buffer and flushes if full
func (al *AuditLogger) addToBuffer(entry *AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.closed {
		return errors.New("audit: logger is closed")
	}

	al.buffer = append(al.buffer, entry)

	// Flush if buffer is full
	if len(al.buffer) >= al.bufferSize {
		return al.flushBuffer()
	}

	return nil
}

// flushBuffer writes all buffered entries to database
// Must be called with al.mu held
func (al *AuditLogger) flushBuffer() error {
	if len(al.buffer) == 0 {
		return nil
	}

	entries := al.buffer
	al.buffer = make([]*AuditEntry, 0, al.bufferSize)

	// Release lock before database write
	al.mu.Unlock()
	defer al.mu.Lock()

	return al.writeBatch(entries)
}

// writeEntry writes a single audit entry to database
func (al *AuditLogger) writeEntry(entry *AuditEntry) error {
	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal details: %v", ErrDatabaseWrite, err)
	}

	query := `
		INSERT INTO audit_logs (
			id, timestamp, user_id, org_id, action,
			resource_type, resource_id, details, ip_address,
			user_agent, request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = al.db.Exec(
		query,
		entry.ID,
		entry.Timestamp,
		entry.UserID,
		entry.OrgID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		detailsJSON,
		entry.IPAddress,
		entry.UserAgent,
		entry.RequestID,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	return nil
}

// writeBatch writes multiple audit entries in a single transaction
func (al *AuditLogger) writeBatch(entries []*AuditEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := al.db.Begin()
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrDatabaseWrite, err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO audit_logs (
			id, timestamp, user_id, org_id, action,
			resource_type, resource_id, details, ip_address,
			user_agent, request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("%w: failed to prepare statement: %v", ErrDatabaseWrite, err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		detailsJSON, err := json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal details: %v", ErrDatabaseWrite, err)
		}

		_, err = stmt.Exec(
			entry.ID,
			entry.Timestamp,
			entry.UserID,
			entry.OrgID,
			entry.Action,
			entry.ResourceType,
			entry.ResourceID,
			detailsJSON,
			entry.IPAddress,
			entry.UserAgent,
			entry.RequestID,
		)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrDatabaseWrite, err)
	}

	return nil
}

// Query retrieves audit logs based on filters
func (al *AuditLogger) Query(filter AuditFilter) ([]*AuditEntry, error) {
	query := `SELECT id, timestamp, user_id, org_id, action, resource_type,
	          resource_id, details, ip_address, user_agent, request_id
	          FROM audit_logs WHERE 1=1`

	args := make([]interface{}, 0)

	// Build dynamic query based on filters
	// Using ? placeholder for compatibility with both SQLite and PostgreSQL (with appropriate driver)
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.OrgID != "" {
		query += " AND org_id = ?"
		args = append(args, filter.OrgID)
	}
	if filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.ResourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, filter.ResourceID)
	}
	if filter.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.EndTime)
	}

	// Order by timestamp descending (most recent first)
	query += " ORDER BY timestamp DESC"

	// Apply pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 1000" // Default limit for safety
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := al.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	entries := make([]*AuditEntry, 0)
	for rows.Next() {
		var entry AuditEntry
		var detailsJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.UserID,
			&entry.OrgID,
			&entry.Action,
			&entry.ResourceType,
			&entry.ResourceID,
			&detailsJSON,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.RequestID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &entry.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return entries, nil
}

// Cleanup removes audit logs older than the specified duration (retention policy)
// This is the only operation that modifies/deletes logs, for compliance purposes
func (al *AuditLogger) Cleanup(olderThan time.Duration) error {
	cutoffTime := time.Now().UTC().Add(-olderThan)

	result, err := al.db.Exec(
		"DELETE FROM audit_logs WHERE timestamp < ?",
		cutoffTime,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the cleanup operation itself
	_ = al.LogWithContext(
		context.Background(),
		ActionDeleted,
		"audit_log",
		"cleanup",
		map[string]any{
			"cutoff_time":   cutoffTime,
			"rows_deleted":  rowsAffected,
			"retention_days": olderThan.Hours() / 24,
		},
	)

	return nil
}

// Flush manually flushes any buffered entries
func (al *AuditLogger) Flush() error {
	if al.bufferSize == 0 {
		return nil // No buffering enabled
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	return al.flushBuffer()
}

// Close flushes any remaining entries and closes the logger
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	if al.closed {
		al.mu.Unlock()
		return nil
	}
	al.closed = true
	al.mu.Unlock()

	// Stop periodic flush
	if al.flushTimer != nil {
		al.flushTimer.Stop()
		close(al.flushDone)
	}

	// Final flush
	return al.Flush()
}

// startPeriodicFlush starts a goroutine that flushes buffer periodically
func (al *AuditLogger) startPeriodicFlush(interval time.Duration) {
	al.flushTimer = time.NewTimer(interval)

	go func() {
		for {
			select {
			case <-al.flushTimer.C:
				al.mu.Lock()
				if !al.closed {
					_ = al.flushBuffer()
					al.flushTimer.Reset(interval)
				}
				al.mu.Unlock()
			case <-al.flushDone:
				return
			}
		}
	}()
}

// Helper functions for creating audit logs for common operations

// LogSessionCreated logs session creation
func LogSessionCreated(ctx context.Context, al *AuditLogger, sessionID, agentType, workspace string) error {
	return al.LogWithContext(ctx, ActionCreated, ResourceTypeSession, sessionID, map[string]any{
		"agent_type": agentType,
		"workspace":  workspace,
	})
}

// LogSessionTerminated logs session termination
func LogSessionTerminated(ctx context.Context, al *AuditLogger, sessionID string) error {
	return al.LogWithContext(ctx, ActionDeleted, ResourceTypeSession, sessionID, nil)
}

// LogSessionAccessed logs session access
func LogSessionAccessed(ctx context.Context, al *AuditLogger, sessionID string) error {
	return al.LogWithContext(ctx, ActionAccessed, ResourceTypeSession, sessionID, nil)
}

// LogMCPConnected logs MCP connection
func LogMCPConnected(ctx context.Context, al *AuditLogger, mcpName, mcpType string) error {
	return al.LogWithContext(ctx, ActionCreated, ResourceTypeMCP, mcpName, map[string]any{
		"mcp_type": mcpType,
		"status":   "connected",
	})
}

// LogMCPDisconnected logs MCP disconnection
func LogMCPDisconnected(ctx context.Context, al *AuditLogger, mcpName string, reason string) error {
	return al.LogWithContext(ctx, ActionDeleted, ResourceTypeMCP, mcpName, map[string]any{
		"status": "disconnected",
		"reason": reason,
	})
}

// LogPromptModified logs prompt modification
func LogPromptModified(ctx context.Context, al *AuditLogger, promptID string, changes map[string]any) error {
	return al.LogWithContext(ctx, ActionUpdated, ResourceTypePrompt, promptID, changes)
}

// LogPromptCreated logs prompt creation
func LogPromptCreated(ctx context.Context, al *AuditLogger, promptID, promptName string) error {
	return al.LogWithContext(ctx, ActionCreated, ResourceTypePrompt, promptID, map[string]any{
		"prompt_name": promptName,
	})
}

// LogAuthAttempt logs authentication attempt
func LogAuthAttempt(ctx context.Context, al *AuditLogger, userID string, success bool, method string) error {
	action := ActionAccessed
	if !success {
		action = ActionFailed
	}
	return al.LogWithContext(ctx, action, ResourceTypeAuth, userID, map[string]any{
		"auth_method": method,
		"success":     success,
	})
}

// LogAPIRequest logs API request
func LogAPIRequest(ctx context.Context, al *AuditLogger, endpoint, method string, statusCode int) error {
	return al.LogWithContext(ctx, ActionAccessed, ResourceTypeAPI, endpoint, map[string]any{
		"method":      method,
		"status_code": statusCode,
	})
}

// Context helper functions

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithOrgID adds organization ID to context
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, orgIDKey, orgID)
}

// WithIPAddress adds IP address to context
func WithIPAddress(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ipAddressKey, ip)
}

// WithUserAgent adds user agent to context
func WithUserAgent(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, userAgentKey, ua)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

// WithHTTPRequest enriches context with HTTP request metadata
func WithHTTPRequest(ctx context.Context, r *http.Request) context.Context {
	ctx = WithIPAddress(ctx, extractIPFromRequest(r))
	ctx = WithUserAgent(ctx, r.UserAgent())

	// Extract request ID if present in headers
	if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
		ctx = WithRequestID(ctx, reqID)
	}

	return ctx
}

// extractUserID extracts user ID from context
func extractUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return "unknown"
}

// extractOrgID extracts organization ID from context
func extractOrgID(ctx context.Context) string {
	if orgID, ok := ctx.Value(orgIDKey).(string); ok {
		return orgID
	}
	return "unknown"
}

// extractIPAddress extracts IP address from context
func extractIPAddress(ctx context.Context) string {
	if ip, ok := ctx.Value(ipAddressKey).(string); ok {
		return ip
	}
	return ""
}

// extractUserAgent extracts user agent from context
func extractUserAgent(ctx context.Context) string {
	if ua, ok := ctx.Value(userAgentKey).(string); ok {
		return ua
	}
	return ""
}

// extractRequestID extracts request ID from context
func extractRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// extractIPFromRequest extracts IP address from HTTP request
// Handles X-Forwarded-For and X-Real-IP headers for proxy scenarios
func extractIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header first (standard for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (used by some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in format "IP:port", extract just the IP
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// Validation functions

var validActions = map[string]bool{
	ActionCreated:  true,
	ActionUpdated:  true,
	ActionDeleted:  true,
	ActionAccessed: true,
	ActionFailed:   true,
}

var validResourceTypes = map[string]bool{
	ResourceTypeSession: true,
	ResourceTypeMCP:     true,
	ResourceTypePrompt:  true,
	ResourceTypeAPI:     true,
	ResourceTypeAuth:    true,
	ResourceTypeConfig:  true,
}

// validateAction validates that action is one of the allowed values
func validateAction(action string) error {
	if action == "" {
		return fmt.Errorf("%w: action cannot be empty", ErrInvalidAction)
	}
	if !validActions[action] {
		return fmt.Errorf("%w: %s", ErrInvalidAction, action)
	}
	return nil
}

// validateResourceType validates that resource type is one of the allowed values
func validateResourceType(resourceType string) error {
	if resourceType == "" {
		return fmt.Errorf("%w: resource type cannot be empty", ErrInvalidResourceType)
	}
	if !validResourceTypes[resourceType] {
		return fmt.Errorf("%w: %s", ErrInvalidResourceType, resourceType)
	}
	return nil
}

//go:build examples
// +build examples

package audit

// This file contains example integration code for using the audit logger
// with AgentAPI. It is not meant to be used directly but serves as documentation.
// Build with: go build -tags examples

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Example: Initialize audit logger on application startup
func ExampleInitialization() (*AuditLogger, error) {
	// Connect to PostgreSQL (or your preferred database)
	db, err := sql.Open("postgres",
		"host=localhost port=5432 user=agentapi dbname=agentapi_audit sslmode=disable")
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create audit logger with buffering for better performance
	// Buffer size of 100 means it will batch-write every 100 logs or every 30 seconds
	logger, err := NewAuditLogger(db, 100)
	if err != nil {
		return nil, err
	}

	// Start retention policy goroutine (runs daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Keep audit logs for 90 days (adjust based on compliance requirements)
			if err := logger.Cleanup(90 * 24 * time.Hour); err != nil {
				// Log error (use your logging framework)
				println("Audit cleanup failed:", err.Error())
			}
		}
	}()

	return logger, nil
}

// Example: HTTP middleware for automatic audit logging
func ExampleAuditMiddleware(logger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start with base context
			ctx := r.Context()

			// Enrich context with HTTP request metadata
			ctx = WithHTTPRequest(ctx, r)

			// Extract user and org from authentication
			// (Implement these based on your auth system)
			if userID := extractUserIDFromJWT(r); userID != "" {
				ctx = WithUserID(ctx, userID)
			}
			if orgID := extractOrgIDFromJWT(r); orgID != "" {
				ctx = WithOrgID(ctx, orgID)
			}

			// Add request ID for tracing
			if reqID := r.Header.Get("X-Request-ID"); reqID == "" {
				// Generate one if not present
				ctx = WithRequestID(ctx, generateRequestID())
			}

			// Wrap response writer to capture status code
			wrapped := &statusRecorder{ResponseWriter: w, statusCode: 200}

			// Call next handler with enriched context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Log the API request after completion
			go func() {
				// Log asynchronously to not block the response
				_ = LogAPIRequest(ctx, logger, r.URL.Path, r.Method, wrapped.statusCode)
			}()
		})
	}
}

// statusRecorder wraps http.ResponseWriter to record status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Example: Integrating with session management
func ExampleSessionManagement(ctx context.Context, logger *AuditLogger) {
	// When creating a session
	sessionID := "session-123"
	agentType := "claude"
	workspace := "/tmp/workspace-123"

	if err := LogSessionCreated(ctx, logger, sessionID, agentType, workspace); err != nil {
		// Handle error - but don't fail the operation
		println("Failed to log session creation:", err.Error())
	}

	// When accessing a session
	if err := LogSessionAccessed(ctx, logger, sessionID); err != nil {
		println("Failed to log session access:", err.Error())
	}

	// When updating session configuration
	changes := map[string]any{
		"field":     "system_prompt",
		"old_value": "default",
		"new_value": "custom prompt",
	}
	if err := logger.LogWithContext(ctx, ActionUpdated, ResourceTypeSession, sessionID, changes); err != nil {
		println("Failed to log session update:", err.Error())
	}

	// When terminating a session
	if err := LogSessionTerminated(ctx, logger, sessionID); err != nil {
		println("Failed to log session termination:", err.Error())
	}
}

// Example: MCP connection tracking
func ExampleMCPTracking(ctx context.Context, logger *AuditLogger) {
	mcpName := "filesystem-mcp"
	mcpType := "stdio"

	// When connecting to MCP
	if err := LogMCPConnected(ctx, logger, mcpName, mcpType); err != nil {
		println("Failed to log MCP connection:", err.Error())
	}

	// When MCP operation occurs
	operation := map[string]any{
		"operation":  "list_files",
		"path":       "/workspace",
		"file_count": 42,
	}
	if err := logger.LogWithContext(ctx, ActionAccessed, ResourceTypeMCP, mcpName, operation); err != nil {
		println("Failed to log MCP operation:", err.Error())
	}

	// When disconnecting from MCP
	if err := LogMCPDisconnected(ctx, logger, mcpName, "user request"); err != nil {
		println("Failed to log MCP disconnection:", err.Error())
	}

	// When MCP operation fails
	errorDetails := map[string]any{
		"operation": "read_file",
		"error":     "permission denied",
		"path":      "/etc/shadow",
	}
	if err := logger.LogWithContext(ctx, ActionFailed, ResourceTypeMCP, mcpName, errorDetails); err != nil {
		println("Failed to log MCP failure:", err.Error())
	}
}

// Example: Authentication tracking
func ExampleAuthTracking(ctx context.Context, logger *AuditLogger) {
	userID := "user-123"

	// Successful login
	if err := LogAuthAttempt(ctx, logger, userID, true, "oauth"); err != nil {
		println("Failed to log auth success:", err.Error())
	}

	// Failed login attempt
	failedCtx := WithIPAddress(ctx, "203.0.113.1")
	if err := LogAuthAttempt(failedCtx, logger, userID, false, "password"); err != nil {
		println("Failed to log auth failure:", err.Error())
	}

	// Password change
	if err := logger.LogWithContext(ctx, ActionUpdated, ResourceTypeAuth, userID, map[string]any{
		"action": "password_change",
		"method": "reset_link",
	}); err != nil {
		println("Failed to log password change:", err.Error())
	}

	// Permission change
	if err := logger.LogWithContext(ctx, ActionUpdated, ResourceTypeAuth, userID, map[string]any{
		"action":      "permission_change",
		"added":       []string{"admin"},
		"removed":     []string{"user"},
		"modified_by": "admin-user",
	}); err != nil {
		println("Failed to log permission change:", err.Error())
	}
}

// Example: Querying audit logs for compliance reports
func ExampleQueryingLogs(logger *AuditLogger) {
	// Get all failed authentication attempts in the last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	failedAuths, err := logger.Query(AuditFilter{
		ResourceType: ResourceTypeAuth,
		Action:       ActionFailed,
		StartTime:    &since,
		Limit:        1000,
	})
	if err != nil {
		println("Failed to query failed auths:", err.Error())
		return
	}

	println("Failed authentication attempts:", len(failedAuths))
	for _, entry := range failedAuths {
		println("  User:", entry.UserID, "IP:", entry.IPAddress, "Time:", entry.Timestamp)
	}

	// Get all session creations for a specific user
	userSessions, err := logger.Query(AuditFilter{
		UserID:       "user-123",
		ResourceType: ResourceTypeSession,
		Action:       ActionCreated,
		Limit:        100,
	})
	if err != nil {
		println("Failed to query user sessions:", err.Error())
		return
	}

	println("Sessions created by user:", len(userSessions))

	// Get all MCP operations for an organization in date range
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)
	mcpOps, err := logger.Query(AuditFilter{
		OrgID:        "org-456",
		ResourceType: ResourceTypeMCP,
		StartTime:    &startTime,
		EndTime:      &endTime,
		Limit:        10000,
	})
	if err != nil {
		println("Failed to query MCP operations:", err.Error())
		return
	}

	println("MCP operations in January:", len(mcpOps))

	// Pagination example - get all entries in batches
	const batchSize = 100
	offset := 0
	var allEntries []*AuditEntry

	for {
		batch, err := logger.Query(AuditFilter{
			OrgID:  "org-456",
			Limit:  batchSize,
			Offset: offset,
		})
		if err != nil {
			println("Failed to query batch:", err.Error())
			break
		}

		if len(batch) == 0 {
			break // No more entries
		}

		allEntries = append(allEntries, batch...)
		offset += batchSize

		if len(batch) < batchSize {
			break // Last batch
		}
	}

	println("Total entries for org:", len(allEntries))
}

// Example: Graceful shutdown
func ExampleGracefulShutdown(logger *AuditLogger) {
	// Flush any buffered entries before shutdown
	if err := logger.Flush(); err != nil {
		println("Failed to flush audit logs:", err.Error())
	}

	// Close the logger
	if err := logger.Close(); err != nil {
		println("Failed to close audit logger:", err.Error())
	}
}

// Helper functions (implement based on your auth system)

func extractUserIDFromJWT(r *http.Request) string {
	// Extract from JWT token in Authorization header
	// This is just a placeholder - implement based on your auth system
	return ""
}

func extractOrgIDFromJWT(r *http.Request) string {
	// Extract from JWT token in Authorization header
	// This is just a placeholder - implement based on your auth system
	return ""
}

func generateRequestID() string {
	// Generate a unique request ID
	// This is just a placeholder - use your preferred method
	return "req-123"
}

// Example: Custom resource types and actions
func ExampleCustomEvents(ctx context.Context, logger *AuditLogger) {
	// While the package provides standard actions and resources,
	// you may need to log custom events. Be aware that validation
	// will fail for non-standard types.

	// To log custom events, you would need to modify the validation
	// in the audit package, or use the standard types creatively:

	// Option 1: Use generic resource type with descriptive details
	err := logger.LogWithContext(ctx, ActionCreated, ResourceTypeConfig, "custom-setting", map[string]any{
		"setting_type": "notification_preference",
		"setting_name": "email_frequency",
		"value":        "daily",
	})
	if err != nil {
		println("Failed to log custom event:", err.Error())
	}

	// Option 2: Use ActionUpdated for state changes
	err = logger.LogWithContext(ctx, ActionUpdated, ResourceTypeConfig, "feature-flag", map[string]any{
		"flag_name":          "new_ui_enabled",
		"old_value":          false,
		"new_value":          true,
		"rollout_percentage": 50,
	})
	if err != nil {
		println("Failed to log feature flag change:", err.Error())
	}
}

// Example: Monitoring and alerting
func ExampleMonitoring(logger *AuditLogger) {
	// Query for suspicious activity patterns
	since := time.Now().Add(-1 * time.Hour)

	// Check for multiple failed auth attempts
	failedAuths, err := logger.Query(AuditFilter{
		ResourceType: ResourceTypeAuth,
		Action:       ActionFailed,
		StartTime:    &since,
		Limit:        10000,
	})
	if err != nil {
		println("Monitoring error:", err.Error())
		return
	}

	// Group by IP to detect brute force
	ipCounts := make(map[string]int)
	for _, entry := range failedAuths {
		ipCounts[entry.IPAddress]++
	}

	// Alert on IPs with many failures
	for ip, count := range ipCounts {
		if count > 10 {
			println("ALERT: Possible brute force from", ip, "with", count, "attempts")
			// Send alert to security team
		}
	}

	// Check for unusual access patterns
	sessions, err := logger.Query(AuditFilter{
		ResourceType: ResourceTypeSession,
		Action:       ActionCreated,
		StartTime:    &since,
		Limit:        10000,
	})
	if err != nil {
		println("Monitoring error:", err.Error())
		return
	}

	// Group by user
	userCounts := make(map[string]int)
	for _, entry := range sessions {
		userCounts[entry.UserID]++
	}

	// Alert on users creating many sessions
	for userID, count := range userCounts {
		if count > 20 {
			println("ALERT: User", userID, "created", count, "sessions in 1 hour")
			// Investigate potential account compromise
		}
	}
}

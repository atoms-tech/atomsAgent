# SessionManagerV2 Usage Examples

The SessionManagerV2 provides a production-ready session management system for the AgentAPI multi-tenant architecture.

## Basic Usage

### Creating a Session Manager

```go
package main

import (
    "context"
    "log"
    "log/slog"

    "github.com/coder/agentapi/lib/session"
)

func main() {
    // Create session manager with workspace root and max concurrent sessions
    workspaceRoot := "/var/lib/agentapi/workspaces"
    maxConcurrent := 100

    sm := session.NewSessionManagerV2(workspaceRoot, maxConcurrent)

    // Or create with custom logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    sm = session.NewSessionManagerV2WithLogger(workspaceRoot, maxConcurrent, logger)
}
```

### Creating a Session

```go
ctx := context.Background()
userID := "user-123"
orgID := "org-456"

// Create a new isolated session
sess, err := sm.CreateSession(ctx, userID, orgID)
if err != nil {
    if err == session.ErrMaxSessionsReached {
        log.Fatal("Maximum concurrent sessions reached")
    }
    log.Fatalf("Failed to create session: %v", err)
}

log.Printf("Created session %s at %s", sess.ID, sess.WorkspacePath)
```

### Retrieving a Session

```go
sessionID := "some-uuid"

sess, err := sm.GetSession(sessionID)
if err != nil {
    if err == session.ErrSessionNotFound {
        log.Fatal("Session not found")
    }
    log.Fatalf("Failed to get session: %v", err)
}

log.Printf("Retrieved session for user %s", sess.UserID)
```

### Cleaning Up a Session

```go
ctx := context.Background()
sessionID := "some-uuid"

// Cleanup session and all resources
err := sm.CleanupSession(ctx, sessionID)
if err != nil {
    log.Fatalf("Failed to cleanup session: %v", err)
}

log.Printf("Session %s cleaned up successfully", sessionID)
```

## Advanced Usage

### Managing MCP Clients

```go
import "github.com/coder/agentapi/lib/mcp"

// Create an MCP client
client, err := mcp.NewClient(
    "client-1",
    "GitHub MCP",
    "http",
    "https://mcp.github.com",
    map[string]any{"timeout": 30},
    map[string]string{"token": "github-token"},
)
if err != nil {
    log.Fatalf("Failed to create MCP client: %v", err)
}

// Add to session
err = sess.AddMCPClient(ctx, client)
if err != nil {
    log.Fatalf("Failed to add MCP client: %v", err)
}

// Get MCP client
client, err = sess.GetMCPClient("client-1")
if err != nil {
    log.Fatalf("Failed to get MCP client: %v", err)
}

// List all MCP clients
clients := sess.ListMCPClients()
for _, c := range clients {
    log.Printf("MCP client: %s (%s)", c.Name, c.ID)
}

// Remove MCP client
err = sess.RemoveMCPClient("client-1")
if err != nil {
    log.Fatalf("Failed to remove MCP client: %v", err)
}
```

### Custom Prompt Composer

```go
import "github.com/coder/agentapi/lib/prompt"

// Create a custom prompt composer
composer := &prompt.Composer{
    GlobalPrompts: []prompt.SystemPromptConfig{
        {
            ID:       "global-1",
            Name:     "Base Instructions",
            Content:  "You are a helpful AI assistant.",
            IsActive: true,
            Priority: 100,
        },
    },
    Validator: prompt.NewValidator(),
}

// Set it on the session manager
sm.SetPromptComposer(composer)

// Now all new sessions will use this composer
sess, err := sm.CreateSession(ctx, userID, orgID)
if err != nil {
    log.Fatalf("Failed to create session: %v", err)
}

log.Printf("System prompt: %s", sess.SystemPrompt)
```

### Custom Audit Logger

```go
type MyAuditLogger struct {
    db *sql.DB
}

func (l *MyAuditLogger) LogSessionEvent(ctx context.Context, userID, orgID, eventType, sessionID string, details map[string]any) {
    // Insert into database
    _, err := l.db.ExecContext(ctx,
        "INSERT INTO audit_log (user_id, org_id, event_type, session_id, details) VALUES ($1, $2, $3, $4, $5)",
        userID, orgID, eventType, sessionID, details,
    )
    if err != nil {
        log.Printf("Failed to log audit event: %v", err)
    }
}

// Use custom audit logger
myLogger := &MyAuditLogger{db: database}
sm.SetAuditLogger(myLogger)
```

### Session Information

```go
// Get thread-safe snapshot of session data
info := sess.GetSessionInfo()

log.Printf("Session Info:")
log.Printf("  ID: %s", info.ID)
log.Printf("  User: %s", info.UserID)
log.Printf("  Org: %s", info.OrgID)
log.Printf("  Workspace: %s", info.WorkspacePath)
log.Printf("  MCP Clients: %v", info.MCPClientIDs)
log.Printf("  Created: %s", info.CreatedAt)
log.Printf("  Last Active: %s", info.LastActiveAt)
```

### Updating System Prompt

```go
// Update session's system prompt
newPrompt := "You are a specialized code review assistant."
sess.UpdateSystemPrompt(newPrompt)

// Retrieve system prompt
prompt := sess.GetSystemPrompt()
log.Printf("Current prompt: %s", prompt)
```

### Listing User Sessions

```go
// List all sessions for a user
userID := "user-123"
sessions := sm.ListSessions(userID)

log.Printf("User %s has %d active sessions:", userID, len(sessions))
for _, s := range sessions {
    log.Printf("  - %s (created %s)", s.ID, s.CreatedAt)
}
```

### Counting Sessions

```go
// Count sessions for a specific user
userCount := sm.CountUserSessions("user-123")
log.Printf("User has %d active sessions", userCount)

// Count all sessions
totalCount := sm.CountAllSessions()
log.Printf("Total active sessions: %d", totalCount)
```

### Cleanup Inactive Sessions

```go
import "time"

// Cleanup sessions inactive for more than 24 hours
inactivityThreshold := 24 * time.Hour
cleanedIDs := sm.CleanupInactiveSessions(ctx, inactivityThreshold)

log.Printf("Cleaned up %d inactive sessions: %v", len(cleanedIDs), cleanedIDs)
```

## Error Handling

```go
sess, err := sm.CreateSession(ctx, userID, orgID)
if err != nil {
    switch err {
    case session.ErrInvalidUserID:
        log.Fatal("User ID is required")
    case session.ErrInvalidOrgID:
        log.Fatal("Organization ID is required")
    case session.ErrMaxSessionsReached:
        log.Fatal("Maximum concurrent sessions reached")
    case session.ErrWorkspaceCreation:
        log.Fatal("Failed to create workspace directory")
    default:
        log.Fatalf("Unexpected error: %v", err)
    }
}
```

## Best Practices

1. **Always cleanup sessions**: Use `CleanupSession` when a session is no longer needed to free resources.

2. **Set reasonable limits**: Configure `maxConcurrent` based on your system's capacity.

3. **Use audit logging**: Implement custom audit logger for compliance and debugging.

4. **Handle errors**: Always check for specific error types for better error handling.

5. **Workspace isolation**: Each session gets a dedicated workspace with 0700 permissions for security.

6. **Thread safety**: All operations are thread-safe and can be called from multiple goroutines.

7. **Inactive session cleanup**: Run periodic cleanup of inactive sessions to prevent resource leaks.

## Integration Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/coder/agentapi/lib/session"
    "github.com/coder/agentapi/lib/prompt"
)

func main() {
    // Initialize session manager
    sm := session.NewSessionManagerV2("/var/lib/agentapi/workspaces", 100)

    // Setup prompt composer
    composer := &prompt.Composer{
        GlobalPrompts: []prompt.SystemPromptConfig{
            {
                ID:       "global-1",
                Name:     "Base Instructions",
                Content:  "You are a helpful AI assistant.",
                IsActive: true,
                Priority: 100,
            },
        },
        Validator: prompt.NewValidator(),
    }
    sm.SetPromptComposer(composer)

    // Create session
    ctx := context.Background()
    sess, err := sm.CreateSession(ctx, "user-123", "org-456")
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }

    log.Printf("Session created: %s", sess.ID)

    // Use session...

    // Cleanup when done
    defer func() {
        if err := sm.CleanupSession(ctx, sess.ID); err != nil {
            log.Printf("Failed to cleanup session: %v", err)
        }
    }()

    // Start periodic cleanup job
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()

        for range ticker.C {
            cleanedIDs := sm.CleanupInactiveSessions(ctx, 24*time.Hour)
            if len(cleanedIDs) > 0 {
                log.Printf("Cleaned up %d inactive sessions", len(cleanedIDs))
            }
        }
    }()

    // Keep running...
    select {}
}
```

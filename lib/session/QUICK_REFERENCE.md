# SessionManagerV2 Quick Reference

## Import

```go
import "github.com/coder/agentapi/lib/session"
```

## Create Manager

```go
// Basic
sm := session.NewSessionManagerV2("/path/to/workspaces", 100)

// With logger
sm := session.NewSessionManagerV2WithLogger("/path/to/workspaces", 100, logger)
```

## Session Operations

### Create Session
```go
sess, err := sm.CreateSession(ctx, "user-123", "org-456")
```

### Get Session
```go
sess, err := sm.GetSession(sessionID)
```

### Cleanup Session
```go
err := sm.CleanupSession(ctx, sessionID)
```

## Querying

### Count Sessions
```go
userCount := sm.CountUserSessions("user-123")
totalCount := sm.CountAllSessions()
```

### List Sessions
```go
sessions := sm.ListSessions("user-123")
```

### Cleanup Inactive
```go
cleanedIDs := sm.CleanupInactiveSessions(ctx, 24*time.Hour)
```

## Session Methods

### MCP Clients
```go
// Add
err := sess.AddMCPClient(ctx, client)

// Get
client, err := sess.GetMCPClient("client-id")

// List
clients := sess.ListMCPClients()

// Remove
err := sess.RemoveMCPClient("client-id")
```

### System Prompt
```go
// Update
sess.UpdateSystemPrompt("new prompt")

// Get
prompt := sess.GetSystemPrompt()
```

### Information
```go
// Workspace
path := sess.GetWorkspacePath()

// Snapshot
info := sess.GetSessionInfo()
```

## Configuration

### Set Prompt Composer
```go
sm.SetPromptComposer(composer)
```

### Set Audit Logger
```go
sm.SetAuditLogger(logger)
```

## Error Types

```go
session.ErrSessionNotFound
session.ErrMaxSessionsReached
session.ErrWorkspaceCreation
session.ErrInvalidUserID
session.ErrInvalidOrgID
```

## Common Patterns

### Complete Lifecycle
```go
// Create
sess, _ := sm.CreateSession(ctx, userID, orgID)

// Use
client, _ := mcp.NewClient(...)
sess.AddMCPClient(ctx, client)

// Cleanup
defer sm.CleanupSession(ctx, sess.ID)
```

### Error Handling
```go
sess, err := sm.CreateSession(ctx, userID, orgID)
switch err {
case session.ErrMaxSessionsReached:
    // Handle limit
case session.ErrInvalidUserID:
    // Handle invalid user
default:
    // Handle other errors
}
```

### Periodic Cleanup
```go
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        sm.CleanupInactiveSessions(ctx, 24*time.Hour)
    }
}()
```

## Key Features

| Feature | Details |
|---------|---------|
| Session IDs | UUID-based |
| Workspace Permissions | 0700 (owner-only) |
| Concurrency | sync.Map + RWMutex |
| MCP Clients | Full lifecycle management |
| Audit Logging | All operations logged |
| Limits | Configurable max sessions |
| Cleanup | Automatic + periodic |

## Thread Safety

✅ All operations are thread-safe
✅ Safe for concurrent goroutines
✅ No race conditions

## Security

✅ Workspace isolation (0700)
✅ Input validation
✅ Resource limits
✅ Audit trail
✅ Secure cleanup

## Performance

- Lock-free reads (sync.Map)
- RWMutex for read-heavy loads
- Atomic operations
- Minimal locking

## Files

- `manager.go` - Implementation (555 lines)
- `manager_test.go` - Tests (612 lines)
- `README.md` - Full documentation
- `example_usage.md` - Usage examples
- `IMPLEMENTATION.md` - Implementation details

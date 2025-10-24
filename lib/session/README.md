# Session Manager

Production-ready session management system for AgentAPI's multi-tenant architecture.

## Overview

The SessionManagerV2 provides isolated, secure session management with the following features:

- **Thread-safe operations** using sync.Map for concurrent access
- **Isolated workspaces** with restrictive permissions (0700)
- **MCP client management** with automatic cleanup
- **System prompt composition** integration
- **Audit logging** for compliance and debugging
- **Concurrent session limits** to prevent resource exhaustion
- **Automatic cleanup** of inactive sessions
- **UUID-based session IDs** for uniqueness

## Architecture

### SessionManagerV2

The main manager that handles session lifecycle:

```go
type SessionManagerV2 struct {
    sessions      sync.Map          // Thread-safe session storage
    workspaceRoot string            // Base directory for workspaces
    maxConcurrent int               // Session limit
    mu            sync.RWMutex      // Protects manager operations
    promptComposer PromptComposer   // System prompt composition
    auditLogger   AuditLogger       // Audit event logging
}
```

### Session

Individual session with isolation and MCP support:

```go
type Session struct {
    ID            string                    // UUID
    UserID        string                    // Owner
    OrgID         string                    // Organization
    WorkspacePath string                    // Isolated directory (0700)
    MCPClients    map[string]*mcp.Client   // Connected MCP clients
    SystemPrompt  string                    // Composed prompt
    CreatedAt     time.Time                // Creation timestamp
    LastActiveAt  time.Time                // Last activity
    mu            sync.RWMutex             // Thread safety
}
```

## Key Features

### 1. Workspace Isolation

Each session gets a dedicated workspace directory with restrictive permissions:

- **Path pattern**: `{workspaceRoot}/{userID}/{sessionID}`
- **Permissions**: 0700 (owner-only access)
- **Automatic creation** on session creation
- **Automatic cleanup** on session deletion

### 2. MCP Client Management

Sessions can manage multiple MCP clients:

- **Add clients**: `AddMCPClient(ctx, client)`
- **Remove clients**: `RemoveMCPClient(clientID)`
- **Get client**: `GetMCPClient(clientID)`
- **List clients**: `ListMCPClients()`
- **Automatic cleanup**: Disconnects and closes clients on session cleanup

### 3. Thread Safety

All operations are thread-safe:

- Uses `sync.Map` for concurrent session access
- RWMutex for session field protection
- Safe for concurrent goroutines

### 4. Audit Logging

Comprehensive audit trail:

- Session creation
- Session cleanup
- Error events
- Configurable audit logger interface

### 5. Concurrent Session Limits

Prevents resource exhaustion:

- Configurable max concurrent sessions
- Per-manager limit enforcement
- Returns `ErrMaxSessionsReached` when limit hit

### 6. System Prompt Composition

Integrates with prompt composer:

- Composes prompts on session creation
- Supports global, org, and user prompts
- Handles composition errors gracefully

## Error Types

```go
var (
    ErrSessionNotFound    = errors.New("session not found")
    ErrMaxSessionsReached = errors.New("maximum concurrent sessions reached")
    ErrWorkspaceCreation  = errors.New("failed to create workspace directory")
    ErrInvalidUserID      = errors.New("invalid user ID")
    ErrInvalidOrgID       = errors.New("invalid org ID")
)
```

## API Reference

### SessionManagerV2 Methods

#### Constructor Functions

```go
// Create with defaults
NewSessionManagerV2(workspaceRoot string, maxConcurrent int) *SessionManagerV2

// Create with custom logger
NewSessionManagerV2WithLogger(workspaceRoot string, maxConcurrent int, logger *slog.Logger) *SessionManagerV2
```

#### Configuration Methods

```go
SetPromptComposer(composer PromptComposer)
SetAuditLogger(logger AuditLogger)
```

#### Session Management

```go
CreateSession(ctx context.Context, userID, orgID string) (*Session, error)
GetSession(sessionID string) (*Session, error)
CleanupSession(ctx context.Context, sessionID string) error
```

#### Querying Methods

```go
CountUserSessions(userID string) int
CountAllSessions() int
ListSessions(userID string) []*Session
CleanupInactiveSessions(ctx context.Context, inactivityThreshold time.Duration) []string
```

### Session Methods

#### MCP Client Management

```go
AddMCPClient(ctx context.Context, client *mcp.Client) error
RemoveMCPClient(clientID string) error
GetMCPClient(clientID string) (*mcp.Client, error)
ListMCPClients() []*mcp.Client
```

#### Prompt Management

```go
UpdateSystemPrompt(prompt string)
GetSystemPrompt() string
```

#### Information Methods

```go
GetWorkspacePath() string
GetSessionInfo() SessionInfo
```

## Interfaces

### PromptComposer

```go
type PromptComposer interface {
    ComposeSystemPrompt(userID, orgID string, userVariables map[string]any) (string, error)
}
```

### AuditLogger

```go
type AuditLogger interface {
    LogSessionEvent(ctx context.Context, userID, orgID, eventType, sessionID string, details map[string]any)
}
```

## Usage Patterns

### Basic Session Lifecycle

```go
// Create manager
sm := session.NewSessionManagerV2("/var/lib/workspaces", 100)

// Create session
sess, err := sm.CreateSession(ctx, "user-123", "org-456")
if err != nil {
    log.Fatal(err)
}

// Use session...

// Cleanup
err = sm.CleanupSession(ctx, sess.ID)
```

### With MCP Clients

```go
// Create session
sess, err := sm.CreateSession(ctx, userID, orgID)

// Add MCP client
client, _ := mcp.NewClient("id", "name", "http", "endpoint", nil, nil)
err = sess.AddMCPClient(ctx, client)

// Use client...

// Cleanup (automatically disconnects MCP clients)
sm.CleanupSession(ctx, sess.ID)
```

### Periodic Cleanup

```go
// Run cleanup every hour
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        cleanedIDs := sm.CleanupInactiveSessions(ctx, 24*time.Hour)
        log.Printf("Cleaned %d sessions: %v", len(cleanedIDs), cleanedIDs)
    }
}()
```

## Testing

The package includes comprehensive tests:

```bash
go test -v ./lib/session/...
```

Test coverage includes:

- Session creation and deletion
- Concurrent operations
- MCP client management
- Error handling
- Session limits
- Inactive session cleanup
- Custom loggers and composers

## Security Considerations

1. **Workspace Isolation**: Each session has a dedicated directory with 0700 permissions
2. **User Validation**: UserID and OrgID are required and validated
3. **Resource Limits**: Maximum concurrent sessions prevent DoS
4. **Audit Logging**: All operations are logged for compliance
5. **Thread Safety**: Prevents race conditions in concurrent environments

## Performance

- **Concurrent-safe**: Uses sync.Map for lock-free reads in common case
- **Efficient cleanup**: Atomic delete operations
- **Minimal locking**: RWMutex for read-heavy workloads
- **Resource isolation**: Per-session workspaces prevent conflicts

## Migration from SessionManager

The original `SessionManager` in `session.go` is preserved for backward compatibility. New code should use `SessionManagerV2`:

### Differences

| Feature | SessionManager | SessionManagerV2 |
|---------|---------------|------------------|
| Storage | map + RWMutex | sync.Map |
| Session ID | timestamp-based | UUID |
| MCP Clients | Config only | Full client management |
| Permissions | 0755 | 0700 |
| Audit | Printf | Interface-based |
| Limits | No | Yes |
| Cleanup | Manual | Automatic + periodic |

## Examples

See [example_usage.md](./example_usage.md) for detailed examples.

## License

Part of the AgentAPI project.

# System Prompt Composer - Integration Guide

This guide shows how to integrate the System Prompt Composer into AgentAPI for multi-tenant prompt management.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         AgentAPI                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐      ┌──────────────┐                   │
│  │   HTTP API   │─────▶│   Session    │                   │
│  │   Handlers   │      │   Manager    │                   │
│  └──────────────┘      └──────────────┘                   │
│         │                      │                            │
│         │                      ▼                            │
│         │              ┌──────────────┐                    │
│         └─────────────▶│   Prompt     │                    │
│                        │   Composer   │                    │
│                        └──────────────┘                    │
│                               │                             │
│                               ▼                             │
│                        ┌──────────────┐                    │
│                        │   Database   │                    │
│                        │  (Postgres)  │                    │
│                        └──────────────┘                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Step 1: Database Setup

### Initialize Database Schema

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/coder/agentapi/lib/prompt"
    _ "github.com/lib/pq"
)

func initDatabase() (*sql.DB, error) {
    // Connect to PostgreSQL
    db, err := sql.Open("postgres",
        "postgres://user:password@localhost:5432/agentapi?sslmode=disable")
    if err != nil {
        return nil, err
    }

    // Initialize schema
    ctx := context.Background()
    if err := prompt.InitializeSchema(ctx, db); err != nil {
        return nil, err
    }

    log.Println("Database schema initialized")
    return db, nil
}
```

### Run Migrations

```bash
# Apply schema migration
psql -h localhost -U user -d agentapi -f lib/prompt/migrations/001_create_system_prompts.sql

# Seed default prompts
psql -h localhost -U user -d agentapi -f lib/prompt/migrations/002_seed_default_prompts.sql

# To rollback (if needed)
psql -h localhost -U user -d agentapi -f lib/prompt/migrations/001_rollback_system_prompts.sql
```

## Step 2: Update MultiTenantAPI

Modify `/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/lib/api/multitenant.go`:

```go
package api

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/coder/agentapi/lib/prompt"
    "github.com/coder/agentapi/lib/session"
)

// MultiTenantAPI handles multi-tenant operations
type MultiTenantAPI struct {
    SessionManager *session.SessionManager
    PromptComposer *prompt.PromptComposer  // Updated to use database-backed composer
    AuditLogger    *AuditLogger
    db             *sql.DB
}

// NewMultiTenantAPI creates a new multi-tenant API
func NewMultiTenantAPI(sessionManager *session.SessionManager, db *sql.DB) *MultiTenantAPI {
    return &MultiTenantAPI{
        SessionManager: sessionManager,
        PromptComposer: prompt.NewPromptComposer(db),  // Use database-backed composer
        AuditLogger:    NewAuditLogger(),
        db:             db,
    }
}

// CreateSession creates a new user session with composed system prompt
func (api *MultiTenantAPI) CreateSession(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getUserIDFromContext(ctx)
    orgID := getOrgIDFromContext(ctx)

    if userID == "" || orgID == "" {
        http.Error(w, "Missing user or organization context", http.StatusUnauthorized)
        return
    }

    var req CreateSessionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Create session config
    config := &session.SessionConfig{
        AgentType:      req.AgentType,
        TermWidth:      req.TermWidth,
        TermHeight:     req.TermHeight,
        InitialPrompt:  req.InitialPrompt,
        Environment:    req.Environment,
        Credentials:    req.Credentials,
    }

    // Create session
    userSession, err := api.SessionManager.CreateSession(ctx, userID, orgID, req.AgentType, config)
    if err != nil {
        api.AuditLogger.Log(ctx, userID, orgID, "session_create_failed", "session", "",
            map[string]any{"error": err.Error()})
        http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
        return
    }

    // Compose system prompt for this user/org
    systemPrompt, err := api.PromptComposer.ComposePrompt(ctx, userID, orgID)
    if err != nil {
        log.Printf("Warning: Failed to compose system prompt: %v", err)
        // Use default prompt if composition fails
        systemPrompt = "You are a helpful AI assistant."
    }

    // Set the composed system prompt
    userSession.SetSystemPrompt(systemPrompt)

    // Set MCPs if provided
    if len(req.MCPConfigs) > 0 {
        userSession.SetMCPs(req.MCPConfigs)
    }

    // Log successful creation
    api.AuditLogger.Log(ctx, userID, orgID, "session_created", "session", userSession.ID,
        map[string]any{
            "agentType": req.AgentType,
            "workspace": userSession.Workspace,
        })

    // Return response
    response := CreateSessionResponse{
        SessionID: userSession.ID,
        Workspace: userSession.Workspace,
        Status:    string(userSession.Status),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## Step 3: Add Prompt Management Endpoints

Add new HTTP endpoints for managing prompts:

```go
// CreatePrompt handles POST /api/v1/prompts
func (api *MultiTenantAPI) CreatePrompt(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getUserIDFromContext(ctx)
    orgID := getOrgIDFromContext(ctx)

    // Only admins can create prompts
    if !isAdmin(ctx) {
        http.Error(w, "Unauthorized", http.StatusForbidden)
        return
    }

    var req struct {
        ID       string             `json:"id"`
        Scope    prompt.PromptScope `json:"scope"`
        Content  string             `json:"content"`
        Template string             `json:"template,omitempty"`
        Priority int                `json:"priority"`
        Enabled  bool               `json:"enabled"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Create prompt
    p := &prompt.SystemPrompt{
        ID:       req.ID,
        Scope:    req.Scope,
        Content:  req.Content,
        Template: req.Template,
        Priority: req.Priority,
        Enabled:  req.Enabled,
    }

    // Set org/user IDs based on scope
    switch req.Scope {
    case prompt.ScopeOrg:
        p.OrgID = &orgID
    case prompt.ScopeUser:
        p.OrgID = &orgID
        p.UserID = &userID
    }

    if err := api.PromptComposer.CreatePrompt(ctx, p); err != nil {
        http.Error(w, fmt.Sprintf("Failed to create prompt: %v", err),
            http.StatusInternalServerError)
        return
    }

    api.AuditLogger.Log(ctx, userID, orgID, "prompt_created", "prompt", p.ID, nil)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"id": p.ID})
}

// GetPrompt handles GET /api/v1/prompts/:id
func (api *MultiTenantAPI) GetPrompt(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    promptID := getPromptIDFromPath(r.URL.Path)

    p, err := api.PromptComposer.GetPrompt(ctx, promptID)
    if err != nil {
        http.Error(w, "Prompt not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(p)
}

// UpdatePrompt handles PUT /api/v1/prompts/:id
func (api *MultiTenantAPI) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    promptID := getPromptIDFromPath(r.URL.Path)

    if !isAdmin(ctx) {
        http.Error(w, "Unauthorized", http.StatusForbidden)
        return
    }

    var req prompt.SystemPrompt
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if err := api.PromptComposer.UpdatePrompt(ctx, promptID, &req); err != nil {
        http.Error(w, fmt.Sprintf("Failed to update prompt: %v", err),
            http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// DeletePrompt handles DELETE /api/v1/prompts/:id
func (api *MultiTenantAPI) DeletePrompt(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    promptID := getPromptIDFromPath(r.URL.Path)

    if !isAdmin(ctx) {
        http.Error(w, "Unauthorized", http.StatusForbidden)
        return
    }

    if err := api.PromptComposer.DeletePrompt(ctx, promptID); err != nil {
        http.Error(w, fmt.Sprintf("Failed to delete prompt: %v", err),
            http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func getPromptIDFromPath(path string) string {
    parts := strings.Split(path, "/")
    if len(parts) >= 5 {
        return parts[4]
    }
    return ""
}

func isAdmin(ctx context.Context) bool {
    // TODO: Implement admin check based on JWT claims or role
    return true
}
```

## Step 4: Register Routes

Update your HTTP router to include the new endpoints:

```go
func setupRoutes(api *MultiTenantAPI) *http.ServeMux {
    mux := http.NewServeMux()

    // Session endpoints
    mux.HandleFunc("POST /api/v1/sessions", api.CreateSession)
    mux.HandleFunc("GET /api/v1/sessions/{id}", api.GetSession)
    mux.HandleFunc("DELETE /api/v1/sessions/{id}", api.TerminateSession)
    mux.HandleFunc("GET /api/v1/users/sessions", api.ListUserSessions)

    // Prompt management endpoints
    mux.HandleFunc("POST /api/v1/prompts", api.CreatePrompt)
    mux.HandleFunc("GET /api/v1/prompts/{id}", api.GetPrompt)
    mux.HandleFunc("PUT /api/v1/prompts/{id}", api.UpdatePrompt)
    mux.HandleFunc("DELETE /api/v1/prompts/{id}", api.DeletePrompt)

    return mux
}
```

## Step 5: Environment Configuration

Add database configuration to your environment:

```bash
# .env
DATABASE_URL=postgres://user:password@localhost:5432/agentapi?sslmode=disable
PROMPT_CACHE_TTL=5m
```

Load in your application:

```go
import (
    "os"
    "time"
)

func loadConfig() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL is required")
    }

    cacheTTL := os.Getenv("PROMPT_CACHE_TTL")
    if cacheTTL == "" {
        cacheTTL = "5m"
    }

    // Use these values in initialization
}
```

## Step 6: Testing the Integration

### Create a Global Prompt

```bash
curl -X POST http://localhost:8080/api/v1/prompts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "global-custom",
    "scope": "global",
    "content": "You are a specialized AI assistant.",
    "priority": 100,
    "enabled": true
  }'
```

### Create an Organization Prompt

```bash
curl -X POST http://localhost:8080/api/v1/prompts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "org-acme-guidelines",
    "scope": "org",
    "content": "Follow ACME Corporation guidelines and policies.",
    "priority": 50,
    "enabled": true
  }'
```

### Create a User Prompt with Template

```bash
curl -X POST http://localhost:8080/api/v1/prompts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "user-preferences",
    "scope": "user",
    "template": "User {{.UserID}} prefers concise responses. Today is {{.Date}}.",
    "priority": 10,
    "enabled": true
  }'
```

### Create a Session (prompts are automatically composed)

```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "agentType": "code",
    "termWidth": 120,
    "termHeight": 40
  }'
```

## Step 7: Monitoring and Observability

### Add Logging

```go
import "log/slog"

func (api *MultiTenantAPI) CreateSession(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    // Log prompt composition
    slog.Info("Composing system prompt",
        "userID", userID,
        "orgID", orgID,
        "sessionID", userSession.ID)

    systemPrompt, err := api.PromptComposer.ComposePrompt(ctx, userID, orgID)
    if err != nil {
        slog.Error("Failed to compose system prompt",
            "error", err,
            "userID", userID,
            "orgID", orgID)
        // Fallback...
    }

    slog.Info("System prompt composed",
        "userID", userID,
        "orgID", orgID,
        "length", len(systemPrompt))
}
```

### Add Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    promptCompositionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "prompt_composition_duration_seconds",
            Help: "Time spent composing prompts",
        },
        []string{"user_id", "org_id"},
    )

    promptCacheHits = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "prompt_cache_hits_total",
            Help: "Total number of prompt cache hits",
        },
    )
)

func init() {
    prometheus.MustRegister(promptCompositionDuration)
    prometheus.MustRegister(promptCacheHits)
}
```

## Security Considerations

1. **Authentication**: Always verify JWT tokens before allowing prompt operations
2. **Authorization**: Only allow admins to create/update/delete prompts
3. **Scope Validation**: Ensure users can only create prompts in their own scope
4. **Input Validation**: Validate all prompt content before storing
5. **Rate Limiting**: Implement rate limiting on prompt composition
6. **Audit Logging**: Log all prompt operations for compliance

## Performance Optimization

1. **Use Caching**: The composer includes built-in caching (5-minute TTL by default)
2. **Connection Pooling**: Configure database connection pool appropriately
3. **Index Optimization**: Ensure all indexes are created (see migration file)
4. **Async Composition**: Consider composing prompts asynchronously for long sessions
5. **Batch Operations**: Use transactions for multiple prompt operations

## Troubleshooting

### Cache Not Working

```go
// Check cache status
composer.cache.Clear()  // Force cache clear
```

### Database Connection Issues

```go
// Test database connection
err := db.PingContext(ctx)
if err != nil {
    log.Fatal("Database connection failed:", err)
}
```

### Template Errors

```go
// Validate template syntax
_, err := template.New("test").Parse(templateStr)
if err != nil {
    log.Printf("Invalid template: %v", err)
}
```

## Migration from In-Memory Implementation

If you're migrating from the previous in-memory composer:

```go
// Old
composer := &prompt.Composer{
    GlobalPrompts: loadGlobalPrompts(),
    Validator:     prompt.NewValidator(),
}

// New
db := initDatabase()
composer := prompt.NewPromptComposer(db)

// Migrate existing prompts
for _, oldPrompt := range oldGlobalPrompts {
    newPrompt := &prompt.SystemPrompt{
        ID:       oldPrompt.ID,
        Scope:    prompt.ScopeGlobal,
        Content:  oldPrompt.Content,
        Priority: oldPrompt.Priority,
        Enabled:  oldPrompt.IsActive,
    }
    composer.CreatePrompt(ctx, newPrompt)
}
```

## Next Steps

1. Set up monitoring and alerting for prompt operations
2. Implement prompt versioning for audit trails
3. Create UI for prompt management
4. Add prompt preview functionality
5. Implement A/B testing for different prompts

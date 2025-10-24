# AgentAPI Multi-Tenant Architecture - Complete Implementation Plan

## Executive Summary

This document provides the complete implementation architecture for transforming AgentAPI into a multi-tenant, enterprise-grade platform with CCRouter support, FastMCP integration, and SOC2 compliance features.

### Scope

- **CCRouter Integration**: VertexAI model support via Claude Code Router
- **FastMCP Integration**: Advanced MCP client with OAuth 2.1 support
- **Multi-Tenancy**: Organization and user-level isolation with Supabase RLS
- **System Prompt Management**: Hierarchical prompt composition (global → org → user)
- **Security**: SOC2 compliance features with audit logging
- **Containerization**: Production-ready Docker deployment on GCP/Render

### Research Completion Status

✅ **AgentAPI Architecture** - Explored via general-purpose agent
✅ **CCRouter Analysis** - 3 comprehensive docs (1,532 lines, 44KB)
✅ **Droid CLI Analysis** - 5 comprehensive docs (57KB)
✅ **Python-Go Integration** - Detailed comparison of gopy, gRPC, JSON-RPC
✅ **FastMCP Specifications** - Complete technical specs from FastMCP 2.12.5 source

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [CCRouter Integration](#2-ccrouter-integration)
3. [FastMCP Integration Strategy](#3-fastmcp-integration-strategy)
4. [Multi-Tenant Architecture](#4-multi-tenant-architecture)
5. [System Prompt Management](#5-system-prompt-management)
6. [Database Schema](#6-database-schema)
7. [API Design](#7-api-design)
8. [Security & Compliance](#8-security--compliance)
9. [Deployment Strategy](#9-deployment-strategy)
10. [Implementation Roadmap](#10-implementation-roadmap)

---

## 1. Architecture Overview

### 1.1 System Layers

```
┌─────────────────────────────────────────────────────────────┐
│  Frontend (Next.js + Vercel)                                │
│  - Chat UI with OAuth flows                                 │
│  - MCP connection management                                │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTPS/WebSocket
┌─────────────────────────▼───────────────────────────────────┐
│  AgentAPI (Go) - Multi-Tenant API Server                    │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Auth Layer: JWT validation, tenant identification     │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ Session Manager: User workspace isolation             │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ System Prompt Composer: Hierarchical prompt merging   │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ Agent Orchestrator: CCRouter, Droid, Claude, etc.     │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ MCP Client Manager: gRPC → FastMCP Service            │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────┬───────────────────────┬───────────────────────┘
              │                       │
              │ gRPC                  │ PTY/Exec
              ▼                       ▼
┌─────────────────────────┐  ┌──────────────────────────────┐
│ FastMCP Service (Python)│  │ Agent CLIs                   │
│ - OAuth 2.1 flows       │  │ - ccr (CCRouter)             │
│ - HTTP/SSE/stdio MCPs   │  │ - droid (Droid CLI)          │
│ - Progress monitoring   │  │ - claude (Claude Code)       │
│ - Token management      │  │ - goose, aider, etc.         │
└─────────────┬───────────┘  └──────────────────────────────┘
              │
              │ HTTP/SSE/stdio
              ▼
┌─────────────────────────────────────────────────────────────┐
│  MCP Servers (External)                                     │
│  - GitHub MCP (OAuth)                                       │
│  - Google Drive MCP (OAuth)                                 │
│  - Filesystem MCP (stdio)                                   │
│  - Database MCP (stdio)                                     │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Supabase PostgreSQL                                        │
│  - Organizations, Users, Sessions                           │
│  - MCP Configurations, OAuth Tokens                         │
│  - System Prompts, Audit Logs                               │
│  - Row-Level Security (RLS) policies                        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Python-Go Bridge** | gRPC (not gopy) | Async support, scalability, type safety |
| **MVP FastMCP Integration** | JSON-RPC/HTTP | Fastest to market (2 weeks), sufficient perf |
| **Production Migration** | Phased to gRPC | When load > 5K req/s, migrate to gRPC |
| **Multi-Tenant Isolation** | Container-level | SOC2 compliance, resource limits |
| **Database** | Supabase PostgreSQL | RLS support, managed service, OAuth integration |
| **Deployment** | Docker + Kubernetes | GCP (future), Render (MVP) |
| **Agent Architecture** | Current pattern | No changes needed, proven design |

---

## 2. CCRouter Integration

### 2.1 Current Status

✅ **Installed and Running**
- Version: 1.0.58
- Location: `/opt/homebrew/bin/ccr`
- Config: `~/.claude-code-router/config.json`
- Service Port: 3456

### 2.2 Integration Points

**AgentAPI Changes** (Already Implemented):
```go
// lib/msgfmt/msgfmt.go
const AgentTypeCCRouter AgentType = "ccrouter"

// cmd/server/server.go
agentTypeMap := map[string]AgentType{
    "ccrouter": AgentTypeCCRouter,
    "ccr":      AgentTypeCCRouter,
}

// Usage
./agentapi server --type=ccrouter -- ccr code
```

**Frontend Changes** (Already Implemented):
```typescript
// chat/src/components/chat-provider.tsx
export type AgentType = "claude" | "ccrouter" | "ccr" | ...

const agentMetadata = {
    ccrouter: {displayName: "CCRouter"},
    ccr: {displayName: "CCRouter"},
}
```

### 2.3 VertexAI Configuration

**Environment Variables**:
```bash
# CCRouter config
export VERTEX_AI_API_KEY="..."
export VERTEX_AI_PROJECT_ID="your-gcp-project"
export VERTEX_AI_LOCATION="us-central1"

# Override for Claude Code
export ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
```

**CCRouter Config** (`~/.claude-code-router/config.json`):
```json
{
  "transformer": "vertex-gemini",
  "baseURL": "https://us-central1-aiplatform.googleapis.com/v1/projects/${VERTEX_AI_PROJECT_ID}/locations/us-central1/publishers/google/models/",
  "defaultModel": "gemini-1.5-pro",
  "apiKey": "${VERTEX_AI_API_KEY}",
  "models": {
    "gemini-1.5-pro": {
      "maxTokens": 1048576,
      "supportsFunctions": true,
      "supportsStreaming": true
    },
    "gemini-1.5-flash": {
      "maxTokens": 1048576,
      "supportsFunctions": true,
      "supportsStreaming": true
    }
  }
}
```

### 2.4 Multi-Tenant Considerations

**Per-Tenant CCRouter Configuration**:
```go
// lib/ccrouter/config.go
type CCRouterConfig struct {
    TenantID        string
    VertexProjectID string
    VertexLocation  string
    DefaultModel    string
    Transformer     string
}

// Load tenant-specific config from database
func (s *SessionManager) GetCCRouterConfig(tenantID string) (*CCRouterConfig, error)
```

**Session-Scoped Credentials**:
- Store VertexAI credentials in Supabase (encrypted)
- Inject per-session environment variables
- Isolate CCRouter instances per user session

---

## 3. FastMCP Integration Strategy

### 3.1 Phased Approach

**Phase 1: MVP (Weeks 1-2) - JSON-RPC over HTTP**

Use FastAPI + uvicorn for Python service:

```python
# lib/mcp/fastmcp_service.py
from fastapi import FastAPI, HTTPException
from fastmcp import FastMCPClient
import asyncio

app = FastAPI()

# In-memory client storage (will move to Redis in Phase 2)
mcp_clients = {}

@app.post("/mcp/connect")
async def connect_mcp(request: ConnectRequest):
    client = FastMCPClient(
        transport=request.transport,
        oauth_provider=request.oauth_provider if request.auth_type == "oauth" else None
    )
    await client.connect()
    mcp_clients[request.client_id] = client
    return {"status": "connected", "tools": await client.list_tools()}

@app.post("/mcp/call_tool")
async def call_tool(request: ToolCallRequest):
    client = mcp_clients.get(request.client_id)
    if not client:
        raise HTTPException(404, "Client not connected")
    result = await client.call_tool(request.tool_name, request.arguments)
    return {"result": result}
```

**Go Client** (HTTP):
```go
// lib/mcp/fastmcp_client.go
type FastMCPClient struct {
    baseURL string
    httpClient *http.Client
}

func (c *FastMCPClient) ConnectMCP(ctx context.Context, config MCPConfig) error {
    resp, err := c.httpClient.Post(
        c.baseURL+"/mcp/connect",
        "application/json",
        bytes.NewBuffer(configJSON),
    )
    // Handle response...
}
```

**Performance**: ~10ms latency, ~6K req/s (sufficient for MVP)

**Phase 2: Production Hardening (Weeks 3-4)**

Add:
- Redis for client state management
- Circuit breaker pattern (go-resilience)
- Retry with exponential backoff
- Prometheus metrics
- Structured logging
- Health checks

**Phase 3: gRPC Migration (Month 3+, if needed)**

Only if load exceeds 5K req/s. Migrate to gRPC for better performance.

### 3.2 OAuth 2.1 Integration

**Frontend OAuth Flow** (Separate from AgentAPI):

```typescript
// Frontend handles OAuth for MCPs requiring manual login
async function connectMCPWithOAuth(mcpName: string) {
    // 1. Initiate OAuth flow
    const authUrl = await fetch('/api/mcp/oauth/init', {
        method: 'POST',
        body: JSON.stringify({ mcp: mcpName, provider: 'github' })
    }).then(r => r.json())

    // 2. Open OAuth popup
    const popup = window.open(authUrl.url, 'oauth', 'width=600,height=700')

    // 3. Listen for OAuth callback
    window.addEventListener('message', async (event) => {
        if (event.data.type === 'oauth_success') {
            // 4. Exchange code for tokens (backend handles this)
            await fetch('/api/mcp/oauth/callback', {
                method: 'POST',
                body: JSON.stringify({ code: event.data.code, state: event.data.state })
            })
        }
    })
}
```

**Backend OAuth Handler** (Supabase Edge Functions or Vercel API Routes):

```typescript
// Store OAuth tokens in Supabase
export async function POST(req: Request) {
    const { code, state } = await req.json()

    // Verify state (CSRF protection)
    const session = await verifyState(state)

    // Exchange code for tokens
    const tokens = await exchangeCodeForTokens(code)

    // Store in Supabase (encrypted)
    await supabase.from('mcp_oauth_tokens').insert({
        user_id: session.userId,
        mcp_name: session.mcpName,
        access_token: encrypt(tokens.access_token),
        refresh_token: encrypt(tokens.refresh_token),
        expires_at: new Date(Date.now() + tokens.expires_in * 1000)
    })

    return Response.json({ success: true })
}
```

**FastMCP Service** (Uses stored tokens):

```python
@app.post("/mcp/connect_with_oauth")
async def connect_with_oauth(request: OAuthConnectRequest):
    # Fetch tokens from Supabase (via AgentAPI)
    tokens = await fetch_oauth_tokens(request.user_id, request.mcp_name)

    # Create OAuth provider
    oauth_provider = OAuth2Provider(
        client_id=settings.GITHUB_CLIENT_ID,
        client_secret=settings.GITHUB_CLIENT_SECRET,
        token_storage=DatabaseTokenStorage(tokens)
    )

    # Connect MCP with OAuth
    client = FastMCPClient(
        transport=HttpTransport(url=request.mcp_url),
        oauth_provider=oauth_provider
    )
    await client.connect()
    return {"status": "connected"}
```

### 3.3 MCP Server Types Support

**HTTP/SSE MCPs** (GitHub, Google Drive, etc.):
- Use `StreamableHttpTransport` (recommended)
- OAuth 2.1 via frontend flow
- Token refresh handled by FastMCP

**stdio MCPs** (Filesystem, Database, etc.):
- Use `StdioServerParameters` with command/args
- No auth required (local execution)
- Session-scoped process isolation

**Configuration Storage** (Supabase):
```sql
CREATE TABLE mcp_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('http', 'sse', 'stdio')),
    endpoint TEXT,
    command TEXT,
    args JSONB,
    auth_type TEXT CHECK (auth_type IN ('none', 'bearer', 'oauth')),
    bearer_token TEXT,
    oauth_provider TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT mcp_unique_per_scope UNIQUE (organization_id, user_id, name)
);
```

---

## 4. Multi-Tenant Architecture

### 4.1 Session Isolation Strategy

**Container-Level Isolation** (SOC2 compliant):

```yaml
# docker-compose.multitenant.yml
services:
  agentapi:
    image: agentapi:latest
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
    environment:
      - MAX_CONCURRENT_SESSIONS=100
      - SESSION_TIMEOUT=3600
      - WORKSPACE_ROOT=/workspaces
    volumes:
      - workspace_data:/workspaces
```

**Session Manager** (Go):

```go
// lib/session/manager.go
type SessionManager struct {
    sessions sync.Map // map[sessionID]*Session
    workspaceRoot string
    maxConcurrent int
}

type Session struct {
    ID            string
    UserID        string
    OrgID         string
    WorkspacePath string
    MCPClients    map[string]*mcp.Client
    SystemPrompt  string
    CreatedAt     time.Time
    LastActiveAt  time.Time
    mu            sync.RWMutex
}

func (sm *SessionManager) CreateSession(userID, orgID string) (*Session, error) {
    // Check concurrent session limit
    if sm.countUserSessions(userID) >= sm.maxConcurrent {
        return nil, ErrMaxSessionsReached
    }

    // Create isolated workspace
    sessionID := uuid.New().String()
    workspacePath := filepath.Join(sm.workspaceRoot, orgID, userID, sessionID)
    if err := os.MkdirAll(workspacePath, 0700); err != nil {
        return nil, err
    }

    // Compose system prompt (global → org → user)
    systemPrompt, err := sm.promptComposer.ComposePrompt(userID, orgID)
    if err != nil {
        return nil, err
    }

    session := &Session{
        ID:            sessionID,
        UserID:        userID,
        OrgID:         orgID,
        WorkspacePath: workspacePath,
        MCPClients:    make(map[string]*mcp.Client),
        SystemPrompt:  systemPrompt,
        CreatedAt:     time.Now(),
        LastActiveAt:  time.Now(),
    }

    sm.sessions.Store(sessionID, session)

    // Audit log
    sm.auditLogger.Log("session_created", map[string]any{
        "session_id": sessionID,
        "user_id":    userID,
        "org_id":     orgID,
    })

    return session, nil
}

func (sm *SessionManager) CleanupSession(sessionID string) error {
    sessionI, ok := sm.sessions.LoadAndDelete(sessionID)
    if !ok {
        return ErrSessionNotFound
    }
    session := sessionI.(*Session)

    // Disconnect all MCP clients
    for _, client := range session.MCPClients {
        client.Disconnect()
    }

    // Remove workspace (optional, can keep for audit)
    // os.RemoveAll(session.WorkspacePath)

    return nil
}
```

### 4.2 Resource Limits

**Per-Session Limits** (enforced in Docker):
- CPU: 0.5 cores per session
- Memory: 512MB per session
- Disk: 10GB per user workspace
- Network: 10MB/s bandwidth

**Kubernetes Resource Quotas** (future):
```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: agentapi-quota
spec:
  hard:
    requests.cpu: "100"
    requests.memory: 200Gi
    limits.cpu: "200"
    limits.memory: 400Gi
    persistentvolumeclaims: "1000"
```

---

## 5. System Prompt Management

### 5.1 Hierarchical Composition

**Scope Hierarchy**: Global → Organization → User

```go
// lib/prompt/composer.go
type PromptComposer struct {
    db *sql.DB
}

type SystemPrompt struct {
    ID      string
    Scope   string // "global", "organization", "user"
    OrgID   *string
    UserID  *string
    Content string
    Priority int
    Template string
    Enabled bool
}

func (pc *PromptComposer) ComposePrompt(userID, orgID string) (string, error) {
    // Fetch all applicable prompts
    prompts, err := pc.fetchPrompts(userID, orgID)
    if err != nil {
        return "", err
    }

    // Sort by priority (global < org < user)
    sort.Slice(prompts, func(i, j int) bool {
        return prompts[i].Priority < prompts[j].Priority
    })

    // Compose based on strategy
    var composed strings.Builder

    for _, prompt := range prompts {
        if prompt.Template != "" {
            // Render template
            tmpl, err := template.New("prompt").Parse(prompt.Template)
            if err != nil {
                return "", err
            }
            var buf bytes.Buffer
            err = tmpl.Execute(&buf, map[string]any{
                "UserID": userID,
                "OrgID":  orgID,
                "Date":   time.Now().Format("2006-01-02"),
            })
            if err != nil {
                return "", err
            }
            composed.WriteString(buf.String())
        } else {
            composed.WriteString(prompt.Content)
        }
        composed.WriteString("\n\n")
    }

    // Sanitize for prompt injection
    sanitized := pc.sanitize(composed.String())

    return sanitized, nil
}

func (pc *PromptComposer) sanitize(content string) string {
    // Remove dangerous patterns
    patterns := []string{
        `(?i)ignore\s+(all\s+)?previous\s+instructions`,
        `(?i)disregard\s+(all\s+)?prior\s+instructions`,
        `(?i)you\s+are\s+now\s+a`,
        // Add more patterns...
    }

    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        content = re.ReplaceAllString(content, "[REDACTED]")
    }

    // HTML escape
    content = html.EscapeString(content)

    return content
}
```

### 5.2 Database Schema

```sql
CREATE TABLE system_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope TEXT NOT NULL CHECK (scope IN ('global', 'organization', 'user')),
    organization_id UUID REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    content TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    template TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id),

    -- Constraints
    CONSTRAINT global_no_refs CHECK (
        (scope = 'global' AND organization_id IS NULL AND user_id IS NULL) OR
        (scope = 'organization' AND organization_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    )
);

-- Indexes
CREATE INDEX idx_system_prompts_org ON system_prompts(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_system_prompts_user ON system_prompts(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_system_prompts_enabled ON system_prompts(enabled) WHERE enabled = true;

-- RLS Policies
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view their org and user prompts" ON system_prompts
    FOR SELECT USING (
        scope = 'global' OR
        organization_id = auth.uid_org_id() OR
        user_id = auth.uid()
    );

CREATE POLICY "Only admins can manage prompts" ON system_prompts
    FOR ALL USING (auth.uid_is_admin());
```

---

## 6. Database Schema

### 6.1 Complete Schema (Supabase PostgreSQL)

See `database/schema.sql` (created earlier) for full schema including:

- **organizations**: Multi-tenant org management
- **users**: User accounts with org membership
- **user_sessions**: Active session tracking
- **mcp_configurations**: Fixed and dynamic MCP lists
- **mcp_oauth_tokens**: Encrypted OAuth tokens
- **system_prompts**: Hierarchical prompt storage
- **audit_logs**: SOC2 compliance audit trail
- **session_workspaces**: Workspace metadata

### 6.2 RLS Policies Summary

All tables have RLS enabled with policies for:
- **SELECT**: Users can view own org/user scoped data
- **INSERT/UPDATE/DELETE**: Role-based access (admin, user)
- **Tenant Isolation**: All queries filtered by `auth.uid_org_id()`

---

## 7. API Design

### 7.1 Multi-Tenant Endpoints

**Session Management**:
```
POST   /api/v1/sessions                 # Create session
GET    /api/v1/sessions/:id             # Get session details
DELETE /api/v1/sessions/:id             # Cleanup session
GET    /api/v1/sessions                 # List user sessions
```

**MCP Management**:
```
POST   /api/v1/mcp/configurations       # Add MCP
GET    /api/v1/mcp/configurations       # List MCPs (org + user)
PUT    /api/v1/mcp/configurations/:id   # Update MCP
DELETE /api/v1/mcp/configurations/:id   # Remove MCP
POST   /api/v1/mcp/test                 # Test MCP connection
```

**System Prompts**:
```
POST   /api/v1/prompts                  # Create prompt
GET    /api/v1/prompts                  # List prompts
PUT    /api/v1/prompts/:id              # Update prompt
DELETE /api/v1/prompts/:id              # Delete prompt
GET    /api/v1/prompts/preview          # Preview composed prompt
```

**Agent Execution** (existing):
```
POST   /api/v1/chat                     # Send message to agent
GET    /api/v1/chat/stream              # SSE stream
```

### 7.2 Authentication

**JWT Validation** (Supabase):
```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", 401)
            return
        }

        // Validate JWT with Supabase
        claims, err := validateSupabaseJWT(token)
        if err != nil {
            http.Error(w, "Invalid token", 401)
            return
        }

        // Add to context
        ctx := context.WithValue(r.Context(), "user_id", claims.Sub)
        ctx = context.WithValue(ctx, "org_id", claims.OrgID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 8. Security & Compliance

### 8.1 SOC2 Requirements

**Data Encryption**:
- ✅ At rest: Supabase PostgreSQL (AES-256)
- ✅ In transit: TLS 1.3 for all connections
- ✅ OAuth tokens: Encrypted in database

**Access Control**:
- ✅ JWT authentication
- ✅ RLS policies on all tables
- ✅ Role-based access (admin, user)
- ✅ Session isolation

**Audit Logging**:
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details JSONB,
    ip_address INET,
    user_agent TEXT
);

-- Immutable audit logs
CREATE POLICY "Audit logs are immutable" ON audit_logs
    FOR DELETE USING (false);
```

**Compliance Features**:
- Data retention policies
- Right to deletion (GDPR)
- Consent management
- Breach notification procedures

### 8.2 Input Validation

**System Prompt Sanitization** (see Section 5.1)

**MCP Configuration Validation**:
```go
func ValidateMCPConfig(config *MCPConfig) error {
    // URL validation
    if config.Type == "http" || config.Type == "sse" {
        u, err := url.Parse(config.Endpoint)
        if err != nil {
            return fmt.Errorf("invalid endpoint URL: %w", err)
        }
        if u.Scheme != "https" {
            return errors.New("MCP endpoints must use HTTPS")
        }
    }

    // Command injection prevention
    if config.Type == "stdio" {
        if strings.Contains(config.Command, ";") || strings.Contains(config.Command, "|") {
            return errors.New("command contains forbidden characters")
        }
    }

    return nil
}
```

---

## 9. Deployment Strategy

### 9.1 Docker Configuration

**Multi-Stage Dockerfile**:
```dockerfile
# Stage 1: Go builder
FROM golang:1.21-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agentapi ./main.go

# Stage 2: Node builder (chat UI)
FROM node:20-alpine AS node-builder
WORKDIR /app/chat
COPY chat/package*.json ./
RUN npm ci
COPY chat/ ./
RUN npm run build

# Stage 3: Python runtime
FROM python:3.11-slim AS python-deps
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Stage 4: Final image
FROM python:3.11-slim
WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy Go binary
COPY --from=go-builder /app/agentapi /usr/local/bin/

# Copy chat UI
COPY --from=node-builder /app/chat/out /usr/local/share/agentapi/chat

# Copy Python dependencies
COPY --from=python-deps /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages

# Copy FastMCP service
COPY lib/mcp/fastmcp_service.py /app/

# Install agent CLIs
RUN npm install -g @musistudio/claude-code-router

# Create workspace directory
RUN mkdir -p /workspaces && chmod 1777 /workspaces

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3284/health || exit 1

# Expose ports
EXPOSE 3284 8000

# Start both services
CMD ["sh", "-c", "python /app/fastmcp_service.py & agentapi server --port=3284"]
```

### 9.2 Kubernetes Deployment (Future - GCP)

**Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentapi
  template:
    metadata:
      labels:
        app: agentapi
    spec:
      containers:
      - name: agentapi
        image: gcr.io/your-project/agentapi:latest
        ports:
        - containerPort: 3284
        - containerPort: 8000
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: agentapi-secrets
              key: database-url
        - name: SUPABASE_URL
          valueFrom:
            secretKeyRef:
              name: agentapi-secrets
              key: supabase-url
        - name: SUPABASE_ANON_KEY
          valueFrom:
            secretKeyRef:
              name: agentapi-secrets
              key: supabase-anon-key
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi
        volumeMounts:
        - name: workspace-storage
          mountPath: /workspaces
      volumes:
      - name: workspace-storage
        persistentVolumeClaim:
          claimName: agentapi-workspace-pvc
```

**Service**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: agentapi
spec:
  selector:
    app: agentapi
  ports:
  - name: http
    port: 80
    targetPort: 3284
  - name: fastmcp
    port: 8000
    targetPort: 8000
  type: LoadBalancer
```

### 9.3 Render Deployment (MVP)

**render.yaml**:
```yaml
services:
  - type: web
    name: agentapi
    env: docker
    dockerfilePath: ./Dockerfile.multitenant
    envVars:
      - key: DATABASE_URL
        sync: false
      - key: SUPABASE_URL
        sync: false
      - key: SUPABASE_ANON_KEY
        sync: false
      - key: MAX_CONCURRENT_SESSIONS
        value: 100
    healthCheckPath: /health
    disk:
      name: workspaces
      mountPath: /workspaces
      sizeGB: 100
```

---

## 10. Implementation Roadmap

### Phase 1: Foundation (Week 1)
**Goal**: Core multi-tenant infrastructure

- [ ] Implement session manager with workspace isolation
- [ ] Create database schema in Supabase
- [ ] Implement RLS policies
- [ ] Add JWT authentication middleware
- [ ] Create system prompt composer
- [ ] Implement audit logging

**Deliverables**:
- `lib/session/manager.go`
- `lib/prompt/composer.go`
- `lib/auth/middleware.go`
- `database/schema.sql`
- `database/migrations/001_initial.sql`

### Phase 2: FastMCP Integration - MVP (Week 2)
**Goal**: Basic MCP support with JSON-RPC/HTTP

- [ ] Create FastMCP service with FastAPI
- [ ] Implement HTTP client in Go
- [ ] Add MCP configuration API endpoints
- [ ] Implement bearer token authentication for MCPs
- [ ] Create MCP connection testing

**Deliverables**:
- `lib/mcp/fastmcp_service.py`
- `lib/mcp/fastmcp_client.go`
- `lib/api/mcp.go`
- `requirements.txt`

### Phase 3: Frontend OAuth Integration (Week 3)
**Goal**: User-initiated OAuth flows for MCPs

- [ ] Create OAuth initiation endpoint
- [ ] Implement OAuth callback handler
- [ ] Build frontend OAuth popup component
- [ ] Add token storage in Supabase
- [ ] Implement token refresh mechanism

**Deliverables**:
- `api/mcp/oauth/init.ts` (Vercel Edge Function)
- `api/mcp/oauth/callback.ts`
- `chat/components/mcp-oauth-connect.tsx`

### Phase 4: Production Hardening (Week 4)
**Goal**: Production-ready features

- [ ] Add Redis for MCP client state
- [ ] Implement circuit breaker pattern
- [ ] Add Prometheus metrics
- [ ] Implement structured logging
- [ ] Add health check endpoints
- [ ] Create comprehensive error handling

**Deliverables**:
- `lib/resilience/circuit_breaker.go`
- `lib/metrics/prometheus.go`
- `lib/logging/structured.go`

### Phase 5: Deployment (Week 5)
**Goal**: Deploy to production

- [ ] Build Docker images
- [ ] Deploy to Render
- [ ] Configure Supabase production instance
- [ ] Set up monitoring and alerting
- [ ] Run load tests
- [ ] Security audit

**Deliverables**:
- Deployed application
- Monitoring dashboard
- Load test results
- Security audit report

### Phase 6: Evaluation & Optimization (Week 6+)
**Goal**: Optimize based on production metrics

- [ ] Monitor performance under real load
- [ ] Identify bottlenecks
- [ ] Evaluate gRPC migration if needed (load > 5K req/s)
- [ ] Implement additional enterprise features
- [ ] HIPAA/FedRAMP preparation

---

## Appendices

### A. Related Documentation

- `CCROUTER_INDEX.md` - CCRouter navigation guide
- `CCROUTER_QUICK_REFERENCE.md` - CCRouter quick start
- `CCROUTER_COMPLETE_ANALYSIS.md` - CCRouter technical analysis
- `DROID_EXPLORATION_INDEX.md` - Droid CLI documentation index
- `PYTHON_GO_INTEGRATION_RESEARCH.md` - Python-Go integration comparison
- `FASTMCP_TECHNICAL_SPECIFICATIONS.md` - FastMCP detailed specs
- `SOFTWARE_PLANNING_DUMP.md` - Original planning session dump
- `MULTITENANT.md` - Multi-tenant architecture details
- `FASTMCP_INTEGRATION.md` - FastMCP integration guide
- `database/schema.sql` - Complete database schema

### B. Key Performance Metrics

| Metric | Target | Monitoring |
|--------|--------|------------|
| Request Latency (p95) | < 100ms | Prometheus |
| Session Creation Time | < 1s | Application logs |
| MCP Connection Time | < 500ms | FastMCP logs |
| Concurrent Sessions | 1,000+ | Kubernetes metrics |
| Database Query Time (p95) | < 50ms | Supabase dashboard |

### C. Cost Estimates (3-Year TCO)

| Component | MVP (Render) | Production (GCP) |
|-----------|--------------|------------------|
| **Compute** | $150/mo | $500/mo |
| **Database** | $25/mo (Supabase) | $200/mo (Cloud SQL) |
| **Storage** | $10/mo | $50/mo |
| **Network** | $20/mo | $100/mo |
| **Total Annual** | $2,460 | $10,200 |
| **3-Year TCO** | $7,380 | $30,600 |

---

## Conclusion

This implementation architecture provides a complete roadmap for transforming AgentAPI into a multi-tenant, enterprise-grade platform with CCRouter, FastMCP, and SOC2 compliance.

**Key Highlights**:
- ✅ CCRouter integration for VertexAI models
- ✅ FastMCP integration with phased approach (JSON-RPC → gRPC)
- ✅ Multi-tenant architecture with container-level isolation
- ✅ Hierarchical system prompt management
- ✅ OAuth 2.1 support for MCP servers
- ✅ SOC2 compliance features
- ✅ Production-ready deployment strategy

**Next Steps**:
1. Review and approve architecture
2. Begin Phase 1 implementation (Foundation)
3. Set up development environment
4. Create project board for tracking

**Estimated Timeline**: 6 weeks to production MVP
**Estimated Cost**: $7,380 (3-year MVP) to $30,600 (3-year production scale)

---

*Document Version: 1.0*
*Last Updated: 2025-10-23*
*Authors: Claude Code exploration agents*

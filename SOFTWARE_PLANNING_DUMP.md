# Software Planning System Contents

## Multi-tenant Session Isolation Design
**Complexity: 8/10**
**Status: Pending**

Design session isolation architecture for Jira-scale multi-tenancy with MVP focus. Each user session needs isolated filesystem access with user-scoped subdirectories, supporting thousands of concurrent users. Consider process-level vs container-level isolation based on MVP requirements.

**Code Example:**
```go
// Session isolation structure
type UserSession struct {
    UserID      string
    SessionID   string
    Workspace   string  // /workspaces/{userID}/{sessionID}
    Process     *termexec.Process
    AgentType   AgentType
    Config      *UserConfig
    MCPs        []MCPConfig
    SystemPrompt string
}
```

---

## CCRouter Provider Integration
**Complexity: 6/10**
**Status: Pending**

Add CCRouter as a new provider type in agentapi, using CCRouter's built-in VertexAI support as the primary interface. Implement as Option B - direct integration into agentapi as a new agent type, with CCRouter managing model routing and agentapi handling terminal emulation.

**Code Example:**
```go
// CCRouter provider implementation
type CCRouterProvider struct {
    ConfigPath string
    BaseURL    string
    APIKey     string
    Models     []string
    Router     *CCRouterConfig
}

type CCRouterConfig struct {
    Default     string            `json:"default"`
    Background  string            `json:"background"`
    Think       string            `json:"think"`
    LongContext string            `json:"longContext"`
    Providers   []ProviderConfig  `json:"providers"`
}
```

---

## MCP Integration Architecture
**Complexity: 9/10**
**Status: Pending**

Design comprehensive MCP integration with fixed (admin-configurable) and dynamic (org/user CRUD) lists. Support HTTP/SSE MCPs with OAuth flows (DCR/PKCE), API tokens, and stdio MCPs. Implement hierarchical inheritance (user overrides org), validation/security scanning, and approval workflows. Use Supabase PG for storage with optional Redis caching.

**Code Example:**
```go
// MCP configuration structure
type MCPConfig struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Type        MCPType          `json:"type"` // http, sse, stdio
    AuthType    AuthType         `json:"authType"` // bearer, apikey, oauth
    Endpoint    string           `json:"endpoint"`
    Config      map[string]any   `json:"config"`
    Scope       MCPScope         `json:"scope"` // system, org, user
    Status      MCPStatus        `json:"status"` // pending, approved, rejected
    CreatedBy   string           `json:"createdBy"`
    OrgID       string           `json:"orgId"`
}

type MCPType string
const (
    MCPTypeHTTP  MCPType = "http"
    MCPTypeSSE   MCPType = "sse" 
    MCPTypeStdio MCPType = "stdio"
)
```

---

## System Prompt Configuration & Security
**Complexity: 7/10**
**Status: Pending**

Implement hierarchical system prompt configuration (Global/Org/User) with template-based merging and concatenation. Include priority-based selection for rare cases. Store in Supabase with proper validation and sanitization to prevent prompt injection attacks. Design secure prompt composition system.

**Code Example:**
```go
// System prompt configuration
type SystemPromptConfig struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Content     string            `json:"content"`
    Scope       PromptScope       `json:"scope"` // global, org, user
    Priority    int               `json:"priority"`
    Template    string            `json:"template"` // go template
    Variables   map[string]any    `json:"variables"`
    OrgID       string            `json:"orgId"`
    UserID      string            `json:"userId"`
    IsActive    bool              `json:"isActive"`
}

type PromptComposer struct {
    GlobalPrompts []SystemPromptConfig
    OrgPrompts    []SystemPromptConfig
    UserPrompts   []SystemPromptConfig
    Validator     *PromptValidator
}
```

---

## Containerization & Security Architecture
**Complexity: 8/10**
**Status: Pending**

Design single monolithic container with all agents for production simplicity. Implement GCP Secret Manager for credential management with Supabase/Vercel/Render MVP stack. Use separate containers per user session for SOC2 compliance and security isolation. Plan horizontal scaling with load balancer and session affinity.

**Code Example:**
```go
// Container architecture
type AgentContainer struct {
    AgentAPI     *AgentAPIServer
    CCRouter     *CCRouterProvider
    Droid        *DroidProvider
    SecretManager *GCPSecretManager
    SessionManager *SessionManager
}

// Session isolation with separate containers
type UserSessionContainer struct {
    UserID      string
    SessionID   string
    ContainerID string
    AgentType   AgentType
    Workspace   string
    Credentials map[string]string
}
```

---

## MCP Client Wrapper Architecture
**Complexity: 9/10**
**Status: Pending**

Research and implement MCP client wrapper using official Go SDK. Design architecture where MCP client sits as 2nd outermost or outermost layer, requiring reshaping agents as LLMs rather than agents for proper nesting. Handle OAuth flows and credential management through separate frontend client.

**Code Example:**
```go
// MCP client wrapper architecture
type MCPClientWrapper struct {
    Client      *mcp.Client
    Session     *mcp.Session
    Transport   mcp.Transport
    Config      *MCPConfig
    AuthManager *OAuthManager
}

// Reshaped agent as LLM for MCP nesting
type LLMAgent struct {
    Model       string
    MCPClient   *MCPClientWrapper
    SystemPrompt string
    Tools       []Tool
}
```

---

## Frontend OAuth Client for MCP Management
**Complexity: 8/10**
**Status: Pending**

Develop separate frontend client (serverless with Supabase state) for MCP management. Handle OAuth flows (DCR/PKCE), validate connections/tool availability, detect auth needs, collect credentials/refresh tokens, and periodically refresh. Push validated MCP configs to agent processes.

**Code Example:**
```go
// Frontend OAuth client structure
type MCPOAuthClient struct {
    SupabaseClient *supabase.Client
    OAuthFlows     map[string]OAuthFlow
    MCPValidator   *MCPValidator
    StateManager   *StateManager
}

type OAuthFlow struct {
    Provider     string
    ClientID     string
    RedirectURI  string
    Scopes       []string
    PKCEConfig   *PKCEConfig
    TokenStorage *TokenStorage
}
```

---

## Database Schema Design for Multi-tenant System
**Complexity: 7/10**
**Status: Pending**

Design comprehensive Supabase database schema for multi-tenant agentapi system including user sessions, MCP configurations, system prompts, organization management, and audit logging. Include RLS policies for data isolation and security.

**Code Example:**
```sql
-- Core tables for multi-tenant system
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    org_id UUID REFERENCES organizations(id),
    agent_type TEXT NOT NULL,
    workspace_path TEXT NOT NULL,
    container_id TEXT,
    status TEXT DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE mcp_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- http, sse, stdio
    config JSONB NOT NULL,
    scope TEXT NOT NULL, -- global, org, user
    org_id UUID REFERENCES organizations(id),
    user_id UUID,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## API Design for Multi-tenant Agent Management
**Complexity: 6/10**
**Status: Pending**

Design RESTful API endpoints for multi-tenant agent management including session creation/management, MCP CRUD operations, system prompt configuration, and real-time status updates. Include proper authentication, authorization, and rate limiting.

**Code Example:**
```go
// API endpoints structure
type AgentAPI struct {
    // Session management
    POST   /api/v1/sessions                    // Create new session
    GET    /api/v1/sessions/{id}               // Get session details
    DELETE /api/v1/sessions/{id}               // Terminate session
    GET    /api/v1/sessions/{id}/status        // Get session status
    
    // MCP management
    GET    /api/v1/mcps                        // List available MCPs
    POST   /api/v1/mcps                        // Add new MCP
    PUT    /api/v1/mcps/{id}                   // Update MCP config
    DELETE /api/v1/mcps/{id}                   // Remove MCP
    POST   /api/v1/mcps/{id}/validate          // Validate MCP connection
    
    // System prompts
    GET    /api/v1/prompts                     // Get system prompts
    POST   /api/v1/prompts                     // Create system prompt
    PUT    /api/v1/prompts/{id}                // Update system prompt
}
```

---

## Docker Containerization & Orchestration
**Complexity: 8/10**
**Status: Pending**

Create Docker containers for agentapi with all agent types (CCRouter, Droid, etc.) and implement container orchestration for multi-tenant session isolation. Include health checks, resource limits, and scaling strategies for Jira-scale deployment.

**Code Example:**
```dockerfile
# Dockerfile for agentapi with all agents
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o agentapi main.go

FROM node:18-alpine AS chat-builder
WORKDIR /app/chat
COPY chat/ .
RUN npm install && npm run build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agentapi .
COPY --from=chat-builder /app/chat/out ./chat
RUN apk add --no-cache nodejs npm
RUN npm install -g @musistudio/claude-code-router
RUN npm install -g @factory-ai/droid
EXPOSE 3284
CMD ["./agentapi"]
```

---

## Security & Compliance Implementation
**Complexity: 9/10**
**Status: Pending**

Implement SOC2 compliance features including audit logging, data encryption, access controls, and security monitoring. Design for future FedRAMP and HIPAA compliance with proper data isolation and security controls.

**Code Example:**
```go
// Security and compliance structures
type SecurityManager struct {
    AuditLogger    *AuditLogger
    Encryption     *EncryptionManager
    AccessControl  *AccessControlManager
    Monitoring     *SecurityMonitoring
}

type AuditLog struct {
    ID        string    `json:"id"`
    UserID    string    `json:"userId"`
    Action    string    `json:"action"`
    Resource  string    `json:"resource"`
    Timestamp time.Time `json:"timestamp"`
    IPAddress string    `json:"ipAddress"`
    UserAgent string    `json:"userAgent"`
    Details   map[string]any `json:"details"`
}
```

---

## Python-Go Integration Strategy

Given that FastMCP is Python-based, here are the recommended integration approaches:

### Option 1: gopy (Recommended)
Use gopy to create Python bindings that can be called directly from Go.

### Option 2: gRPC Microservice
Create a separate Python gRPC service for FastMCP operations.

### Option 3: HTTP Microservice
Create a Python HTTP service with FastMCP and call it from Go.

### Option 4: CGO with Python C API
Use CGO to directly call Python C API (complex but performant).

### Option 5: Process-based Communication
Use stdin/stdout communication with Python subprocess (current approach, but not ideal for production).

---

## Next Steps

1. Choose Python-Go integration strategy
2. Implement chosen integration approach
3. Complete remaining software planning tasks
4. Test and validate the complete system
5. Deploy to production environment
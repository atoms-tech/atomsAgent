# AgentAPI Architecture Walkthrough

## Overview

Your fork implements a **multi-layered OpenAI-compatible API** that wraps two specialized agents (CCRouter and Droid), both powered by VertexAI, and manages them through an integrated MCP (Model Context Protocol) client.

```
┌─────────────────────────────────────────────────────────────┐
│                   OpenAI-Compatible API                     │
│                    (HTTP Endpoints)                         │
│              POST /v1/chat/completions                      │
│              GET /v1/models                                 │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼─────┐  ┌─────▼────┐  ┌──────▼──────┐
    │ ChatAPI  │  │ ModelAPI │  │  MCP Routes │
    │ Handler  │  │ Handler  │  │  (Manage    │
    │          │  │          │  │   MCPs)     │
    └────┬─────┘  └─────┬────┘  └──────┬──────┘
         │              │              │
         └──────────────┼──────────────┘
                        │
         ┌──────────────▼──────────────┐
         │   Orchestrator Layer        │
         │  (Agent Selection Logic)    │
         │  - Primary: CCRouter        │
         │  - Fallback: Droid          │
         └──────────────┬──────────────┘
                        │
         ┌──────────────┼──────────────┐
         │              │              │
    ┌────▼─────┐  ┌────▼─────┐  ┌────▼──────┐
    │ CCRouter │  │  Droid   │  │ FastMCP   │
    │ Agent    │  │  Agent   │  │ Client    │
    │(VertexAI)│  │(VertexAI)│  │(MCP Mgmt) │
    └────┬─────┘  └────┬─────┘  └────┬──────┘
         │             │             │
         └─────────────┼─────────────┘
                       │
         ┌─────────────▼──────────────┐
         │   VertexAI Backend         │
         │  (Gemini Models)           │
         │  - gemini-1.5-pro          │
         │  - gemini-1.5-flash        │
         └────────────────────────────┘
```

## Layer 1: HTTP API (OpenAI Compatible)

**Location:** `api/v1/`

**Files:**
- `api/v1/chat.go` - Chat completions endpoints
- `api/v1/models.go` - Model listing endpoints

**Key Code: Chat Handler**

```go
type ChatHandler struct {
    orchestrator *Orchestrator
    auditLogger  *AuditLogger
    metrics      *MetricsRegistry
}

func (h *ChatHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
    // 1. Parse OpenAI-compatible request
    var req ChatCompletionRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 2. Route to orchestrator
    response, err := h.orchestrator.ProcessRequest(r.Context(), req)

    // 3. Return OpenAI-compatible response
    json.NewEncoder(w).Encode(response)
}
```

## Layer 2: Orchestrator (Agent Selection)

**Location:** `lib/chat/orchestrator.go`

The orchestrator implements intelligent routing with automatic failover:

```go
type Orchestrator struct {
    primaryAgent   Agent    // CCRouter
    fallbackAgent  Agent    // Droid
    primaryName    string   // "ccrouter"
    fallbackEnabled bool    // true
}

func (o *Orchestrator) ProcessRequest(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
    // Try primary agent first
    response, err := o.primaryAgent.Chat(ctx, req.Messages, req.Model)
    if err == nil {
        return response, nil
    }

    // Fallback to alternate agent
    if o.fallbackEnabled {
        return o.fallbackAgent.Chat(ctx, req.Messages, req.Model)
    }

    return nil, err
}
```

## Layer 3: Agent Implementations

**Location:** `lib/agents/`

### Agent Interface

```go
type Agent interface {
    Chat(ctx context.Context, messages []Message, model string) (*ChatCompletionResponse, error)
    Health(ctx context.Context) error
    Stop() error
}
```

### CCRouter Agent

Specialized reasoning agent using VertexAI:

- Launches as subprocess
- Communicates via stdin/stdout pipes
- Calls VertexAI Gemini models internally
- Specialized for complex reasoning

### Droid Agent

Fallback agent also VertexAI-powered:

- Same subprocess architecture
- Different VertexAI configuration
- Provides redundancy

## Layer 4: VertexAI Backend

**Configuration:**

```env
VERTEX_AI_USE_APPLICATION_DEFAULT=true
VERTEX_AI_PROJECT_ID=serious-mile-462615-a2
VERTEX_AI_LOCATION=us-central1
```

**Supported Models:**
- `gemini-1.5-pro` - Advanced reasoning, long context
- `gemini-1.5-flash` - Fast responses, lower cost
- `gemini-1.0-pro` - Standard reasoning

**Authentication:** Google Cloud Application Default Credentials

```bash
gcloud auth application-default login
```

## Layer 5: MCP Integration

**Location:** `lib/mcp/`

**Files:**
- `lib/mcp/client.go` - Base MCP client
- `lib/mcp/fastmcp_client.go` - FastMCP 2.0 integration
- `lib/mcp/fastmcp_http_client.go` - HTTP-based MCP client

**FastMCP 2.0 Client:**

```go
type FastMCPClient struct {
    baseURL    string
    httpClient *http.Client
}

func (f *FastMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
    // Execute remote MCP tool
}
```

**MCP Routes:**

```
GET  /api/v1/mcp/tools       - List available MCP tools
POST /api/v1/mcp/invoke      - Call an MCP tool
GET  /api/v1/mcp/status      - Check MCP connection
```

## Layer 6: Supporting Systems

### Authentication

**Location:** `lib/auth/`

Priority order:
1. Static API key (environment)
2. Database API key
3. JWT token (WorkOS)

### Audit Logging

**Location:** `lib/audit/`

- Complete request/response logging
- Compliance tracking
- User activity audit trail

### Metrics

**Location:** `lib/metrics/`

- Request count by model
- Latency percentiles
- Error rates
- Token usage
- Prometheus-compatible

## Complete Data Flow

```
1. Client Request
   POST /v1/chat/completions
   Authorization: Bearer {token}

2. Authentication
   ├─ Extract token
   ├─ Try static API key
   ├─ Try database key
   └─ Try JWT ✓

3. ChatHandler
   ├─ Parse OpenAI request
   ├─ Validate format
   └─ Pass to orchestrator

4. Orchestrator
   ├─ Try CCRouter
   └─ Fallback to Droid ✓

5. CCRouter/Droid Agent
   ├─ Call VertexAI
   └─ Get streaming response

6. Handler
   ├─ Convert to OpenAI format
   ├─ Log audit event
   └─ Record metrics

7. Response
   HTTP 200 OK
   {
     "id": "...",
     "choices": [...]
   }
```

## Key Architecture Features

### ✅ OpenAI Compatibility
- 100% compatible API format
- Identical response structure
- Multiple model support

### ✅ Dual Agent Architecture
- Primary: CCRouter (reasoning)
- Fallback: Droid (alternate)
- Automatic failover

### ✅ VertexAI Integration
- Gemini models
- Streaming support
- Function calling

### ✅ MCP Management
- FastMCP 2.0 client
- Tool invocation
- Connection management

### ✅ Enterprise Features
- Multi-tenancy
- Audit logging
- Metrics
- Rate limiting
- Circuit breaker

## Testing

```bash
# Health check
curl http://localhost:3284/health

# List models
curl -H "Authorization: Bearer $STATIC_API_KEY" \
  http://localhost:3284/v1/models

# Chat
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer $STATIC_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ccrouter",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Summary

Your implementation provides:

1. **HTTP API** - OpenAI-compatible endpoints
2. **Orchestrator** - Intelligent agent routing
3. **Dual Agents** - CCRouter + Droid (VertexAI)
4. **VertexAI Backend** - Gemini models
5. **MCP Integration** - Tool management
6. **Enterprise Features** - Auth, audit, metrics
7. **Multi-tenant** - Organization support

This is a **production-ready AI platform** with flexibility, reliability, and scalability.

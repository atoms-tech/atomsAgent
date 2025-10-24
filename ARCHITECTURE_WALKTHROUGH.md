# AgentAPI Multi-Tenant Architecture - Complete Walkthrough

**Date**: October 24, 2025
**Purpose**: Detailed walkthrough of system architecture with environment setup

---

## System Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                         Frontend Layer                             │
│                    (atoms.tech / Next.js)                          │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │ OAuth Components                                           │   │
│  │ - MCPOAuthConnect.tsx (Provider Selection)               │   │
│  │ - MCPOAuthCallback.tsx (Token Exchange)                  │   │
│  │ - useMCPOAuth.ts (State Management)                      │   │
│  │ - oauth.service.ts (API Calls)                           │   │
│  │ - token.service.ts (Encryption/Refresh)                 │   │
│  └────────────────────────────────────────────────────────────┘   │
│  Port: 3000 (localhost)                                            │
└───────────────────────────────┬──────────────────────────────────────┘
                                │
                  HTTPS/JSON (OAuth, REST API)
                                │
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│                      Backend API Layer                             │
│                     (agentapi / Go)                                │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │ API Routes                                                 │   │
│  │ ├─ POST /api/mcp/oauth/init          → OAuth init         │   │
│  │ ├─ POST /api/mcp/oauth/callback      → Token exchange     │   │
│  │ ├─ POST /api/v1/mcp/configurations   → Create MCP config  │   │
│  │ ├─ GET  /api/v1/mcp/configurations   → List configs       │   │
│  │ ├─ PUT  /api/v1/mcp/configurations/:id → Update config    │   │
│  │ ├─ DELETE /api/v1/mcp/configurations/:id → Delete config  │   │
│  │ └─ POST /api/v1/mcp/test             → Test connection    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │ Middleware Layer                                           │   │
│  │ ├─ JWT Authentication (Bearer tokens)                    │   │
│  │ ├─ Rate Limiting (60 req/min, 10 burst)                 │   │
│  │ ├─ CORS Protection                                       │   │
│  │ ├─ Request ID Tracking                                   │   │
│  │ └─ Prometheus Metrics (<1µs overhead)                    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │ Core Libraries                                             │   │
│  │ ├─ lib/session          → Session management              │   │
│  │ ├─ lib/auth             → JWT validation + RBAC           │   │
│  │ ├─ lib/mcp              → MCP client lifecycle            │   │
│  │ ├─ lib/redis            → State persistence               │   │
│  │ ├─ lib/resilience       → Circuit breaker (5x)           │   │
│  │ ├─ lib/ratelimit        → Token bucket algorithm          │   │
│  │ ├─ lib/health           → Health checks                   │   │
│  │ ├─ lib/metrics          → Prometheus metrics              │   │
│  │ ├─ lib/logging          → Structured JSON logging         │   │
│  │ └─ lib/audit            → Security audit tools            │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  Port: 3284 (localhost)                                            │
└────────┬───────────────────────────────┬────────────────────┬─────┘
         │                               │                    │
         │ PostgreSQL                    │ Redis              │ HTTP/JSON-RPC
         │ (data store)                  │ (session state)    │ (FastMCP service)
         │                               │                    │
         ▼                               ▼                    ▼
┌──────────────────┐          ┌──────────────────┐  ┌──────────────────┐
│   Supabase       │          │  Upstash Redis   │  │  FastMCP Service │
│   PostgreSQL     │          │                  │  │  (Python)        │
│                  │          │                  │  │                  │
│ Tables:          │          │ • Sessions       │  │ • MCP clients    │
│ - organizations  │          │ • Tokens         │  │ • OAuth flows    │
│ - users          │          │ • State          │  │ • Async support  │
│ - sessions       │          │ • DLQ            │  │ • Progress track │
│ - mcp_configs    │          │ • Cache          │  │                  │
│ - oauth_tokens   │          │                  │  │ Port: 8000       │
│ - prompts        │          │ Fallback:        │  │                  │
│ - audit_logs     │          │ In-memory store  │  └──────────────────┘
│                  │          │ if unavailable   │        │
│ RLS policies:    │          └──────────────────┘        │
│ - Per org        │                                       │ HTTP/SSE/stdio
│ - Per user       │                                       │
└──────────────────┘                                       ▼
                                                  ┌──────────────────┐
                                                  │   MCP Servers    │
                                                  │                  │
                                                  │ • GitHub         │
                                                  │ • Google Drive   │
                                                  │ • Slack          │
                                                  │ • Database       │
                                                  │ • Filesystem     │
                                                  │ • Custom...      │
                                                  └──────────────────┘
```

---

## Component Breakdown

### 1. Frontend (atoms.tech)

**Location**: `/Users/kooshapari/temp-prodvercel/485/clean/deploy/atoms.tech`

**Key Components**:

#### OAuth Flow Components
```typescript
// src/components/mcp/MCPOAuthConnect.tsx
- Provider selection modal
- GitHub, Google, Azure, Auth0 support
- Initiates OAuth flow via POST /api/mcp/oauth/init

// src/components/mcp/MCPOAuthCallback.tsx
- Handles OAuth callback
- Exchanges authorization code for access token
- Stores encrypted token

// src/hooks/useMCPOAuth.ts
- React hook managing OAuth state
- Auto-refresh on expiry
- Error handling

// src/services/mcp/oauth.service.ts
- HTTP client for OAuth endpoints
- Retry logic with exponential backoff
- Error transformation

// src/services/mcp/token.service.ts
- Secure token storage (localStorage encrypted)
- Token refresh mechanism
- Token validation and expiry checking
```

**Environment Variables** (from atoms.tech/.env.local):
```
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_SUPABASE_URL=https://ydogoylwenufckscqijp.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=[JWT_TOKEN]
SUPABASE_JWT_SECRET=SDqZb9KKouY+7YccWBXxnJdvsGcDLUHrQnZlUo5z8hE0clHn9aArY3RzHra/TCSalpYmcEEW/xmCTbGmVs0CLQ==
SUPABASE_JWKS_URL=https://ydogoylwenufckscqijp.supabase.co/auth/v1/.well-known/jwks.json
```

---

### 2. Backend API (agentapi)

**Location**: `/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi`

#### Authentication Flow

```
Browser (atoms.tech)
    ↓ POST /api/mcp/oauth/init
    ├─ provider: "github"
    ├─ redirect_uri: "http://localhost:3000/oauth/callback"
    └─ state: [32-byte random] + PKCE code_verifier

AgentAPI (port 3284)
    ├─ Generate OAuth URL with PKCE
    ├─ Store state + code_verifier in Redis (10-min TTL)
    └─ Return OAuth URL

Browser
    ├─ Redirect to OAuth provider
    ├─ User authorizes
    └─ Provider redirects to callback with ?code=...&state=...

AgentAPI (POST /api/mcp/oauth/callback)
    ├─ Validate state (CSRF protection)
    ├─ Verify PKCE code_verifier
    ├─ Exchange code for access token
    ├─ Encrypt and store in Redis + Supabase
    └─ Return success + token metadata

Browser
    ├─ Receive encrypted token
    ├─ Store locally
    ├─ Use for MCP operations
    └─ Auto-refresh before expiry
```

#### API Endpoints

**OAuth Endpoints**:
```
POST /api/mcp/oauth/init
  Request:  { provider, redirect_uri, scope? }
  Response: { oauth_url, state }

POST /api/mcp/oauth/callback
  Request:  { code, state, provider }
  Response: { access_token, refresh_token, expires_in }

POST /api/mcp/oauth/refresh
  Request:  { refresh_token, provider }
  Response: { access_token, expires_in }

POST /api/mcp/oauth/revoke
  Request:  { token, provider }
  Response: { success: true }
```

**MCP Configuration Endpoints**:
```
POST /api/v1/mcp/configurations
  Create MCP configuration
  - name, url, provider, auth_token (encrypted)

GET /api/v1/mcp/configurations
  List user's MCP configurations
  - Filters: provider, org_id

GET /api/v1/mcp/configurations/:id
  Get single configuration

PUT /api/v1/mcp/configurations/:id
  Update configuration

DELETE /api/v1/mcp/configurations/:id
  Delete configuration

POST /api/v1/mcp/test
  Test MCP connection
  - Validates URL and credentials
```

**Health Endpoints**:
```
GET /health
  Application health (UP/DOWN)

GET /ready
  Readiness probe (ready for traffic)

GET /live
  Liveness probe (process alive)
```

#### Library Architecture

```
lib/
├── session/
│   ├── manager.go          → Session lifecycle management
│   └── README.md           → Usage guide
│
├── auth/
│   ├── middleware.go       → JWT validation, RBAC
│   ├── claims.go           → JWT claims parsing
│   └── README.md
│
├── mcp/
│   ├── fastmcp_service.py  → Python FastMCP service (HTTP/SSE/stdio)
│   ├── fastmcp_http_client.go → Go HTTP wrapper
│   └── README.md
│
├── redis/
│   ├── client.go           → Dual protocol (native + REST)
│   ├── mcp_state.go        → MCP client state storage
│   ├── session_store.go    → Session persistence
│   ├── token_cache.go      → Encrypted token storage
│   ├── redis_dlq.go        → Dead letter queue
│   └── README.md
│
├── resilience/
│   ├── circuit_breaker.go  → 3-state circuit breaker
│   └── README.md
│
├── ratelimit/
│   ├── limiter.go          → Token bucket algorithm
│   ├── middleware.go       → HTTP middleware
│   └── README.md
│
├── metrics/
│   ├── prometheus.go       → Prometheus client
│   ├── grafana-dashboard.json
│   └── README.md
│
├── logging/
│   ├── structured.go       → JSON structured logging
│   └── README.md
│
├── health/
│   ├── checker.go          → Health check coordinator
│   └── README.md
│
├── security/
│   ├── audit.go            → Security auditor
│   └── README.md
│
└── errors/
    ├── mcp_errors.go       → Error types with retry flags
    └── README.md
```

---

### 3. Data Layer

#### PostgreSQL (Supabase)

**Tables**:

```sql
organizations
  - id (UUID)
  - name (text)
  - created_at (timestamp)
  - RLS: Org members only

users
  - id (UUID)
  - org_id (FK organizations)
  - email (text)
  - role (enum: admin, user)
  - created_at (timestamp)
  - RLS: Self + org admins

user_sessions
  - id (UUID)
  - user_id (FK users)
  - org_id (FK organizations)
  - session_id (text, unique)
  - created_at (timestamp)
  - expires_at (timestamp)
  - RLS: Self only

mcp_configurations
  - id (UUID)
  - org_id (FK organizations)
  - user_id (FK users, nullable)
  - name (text)
  - url (text)
  - provider (enum)
  - auth_token_encrypted (bytea)
  - RLS: Org members + user configs

mcp_oauth_tokens
  - id (UUID)
  - user_id (FK users)
  - org_id (FK organizations)
  - provider (enum)
  - access_token_encrypted (bytea)
  - refresh_token_encrypted (bytea)
  - expires_at (timestamp)
  - RLS: User only

system_prompts
  - id (UUID)
  - scope (enum: global, org, user)
  - org_id (FK organizations, nullable)
  - user_id (FK users, nullable)
  - content (text)
  - RLS: Global readable, org/user scoped

audit_logs
  - id (UUID)
  - user_id (FK users)
  - org_id (FK organizations)
  - action (text)
  - resource (text)
  - details (jsonb)
  - ip_address (text)
  - created_at (timestamp)
  - RLS: Org members only (read)
  - IMMUTABLE: Append-only
```

**Row-Level Security** (RLS):
```
All tables have RLS enabled with policies:
- Organizations: Members can read/write
- Users: Self + admins can manage
- Sessions: User owns their sessions
- MCP Configs: Scoped to org + user
- OAuth Tokens: User owns their tokens
- Audit Logs: Org members can read
```

#### Redis (Upstash)

**Key Patterns**:

```
Sessions:
  session:{session_id}           → Session metadata (JSON)
  session:{session_id}:mcp_state → MCP client state

OAuth:
  oauth:state:{state_id}         → State + PKCE verifier (10min TTL)
  oauth:token:{user_id}:{org}:{provider} → Encrypted token

Rate Limiting:
  ratelimit:{user_id}:{endpoint} → Token count (sliding window)

Dead Letter Queue:
  dlq:failed_operations          → Failed operations log

Cache:
  cache:tool_list:{mcp_id}       → Tool definitions (5min TTL)
  cache:session:{user_id}        → Session metadata (2min TTL)
```

**Fallback**: If Redis unavailable, application falls back to in-memory storage

---

## Environment Setup for Local Testing

### Backend Environment (.env.docker or .env.local)

Create `.env` in agentapi root:

```bash
# ==============================================================================
# Database Configuration
# ==============================================================================
DATABASE_URL=postgresql://agentapi:agentapi@postgres:5432/agentapi?sslmode=disable
POSTGRES_USER=agentapi
POSTGRES_PASSWORD=agentapi
POSTGRES_DB=agentapi
POSTGRES_PORT=5432

# ==============================================================================
# Supabase Configuration (Borrow from atoms.tech)
# ==============================================================================
SUPABASE_URL=https://ydogoylwenufckscqijp.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlkb2dveWx3ZW51ZmNrc2NxaWpwIiwicm9sZSI6ImFub24iLCJpYXQiOjE3MzY3MzUxNjYsImV4cCI6MjA1MjMxMTE2Nn0.Oy0K0aalki4e4b5h8caHYdWxZVKB6IWDDYQ3zvCUu4Y
SUPABASE_SERVICE_ROLE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlkb2dveWx3ZW51ZmNrc2NxaWpwIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTczNjczNTE2NiwiZXhwIjoyMDUyMzExMTY2fQ.fSWnBNuIE3QXU93naKCmbUiWkGg5LVnOQg5uSyLYaNo
SUPABASE_JWT_SECRET=SDqZb9KKouY+7YccWBXxnJdvsGcDLUHrQnZlUo5z8hE0clHn9aArY3RzHra/TCSalpYmcEEW/xmCTbGmVs0CLQ==
SUPABASE_JWKS_URL=https://ydogoylwenufckscqijp.supabase.co/auth/v1/.well-known/jwks.json

# ==============================================================================
# Redis Configuration (Upstash)
# ==============================================================================
UPSTASH_REDIS_REST_URL=https://neat-sloth-35614.upstash.io
UPSTASH_REDIS_REST_TOKEN=AYseAAIncDFhMDY2ZmJlNWNlYzg0ZWNhYmFlYjRmYjliNmQ2NGUwOXAxMzU2MTQ
UPSTASH_REDIS_URL=rediss://default:AYseAAIncDFhMDY2ZmJlNWNlYzg0ZWNhYmFlYjRmYjliNmQ2NGUwOXAxMzU2MTQ@neat-sloth-35614.upstash.io:6379

REDIS_ENABLE=true
REDIS_PROTOCOL=native
REDIS_MAX_POOL_SIZE=10
REDIS_CONNECTION_TIMEOUT=5s

# ==============================================================================
# Application Configuration
# ==============================================================================
NODE_ENV=development
APP_PORT=3284

# CORS and allowed origins (atoms.tech)
AGENTAPI_ALLOWED_HOSTS=*
AGENTAPI_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3284

# ==============================================================================
# Rate Limiting
# ==============================================================================
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10

# ==============================================================================
# Circuit Breaker
# ==============================================================================
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_SUCCESS_THRESHOLD=2
CIRCUIT_BREAKER_TIMEOUT=30s

# ==============================================================================
# Metrics & Logging
# ==============================================================================
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090
LOG_LEVEL=debug

# ==============================================================================
# FastMCP Service
# ==============================================================================
FASTMCP_URL=http://localhost:8000
FASTMCP_TIMEOUT=30s

# ==============================================================================
# Encryption (Generate with: head -c 32 /dev/urandom | base64)
# ==============================================================================
ENCRYPTION_KEY=your-base64-encoded-32-byte-key-here
TOKEN_ENCRYPTION_ALGORITHM=AES-256-GCM
```

### Frontend Environment (.env.local in atoms.tech)

Create/update `.env.local` in atoms.tech:

```bash
# Existing Supabase config (already in atoms.tech)
NEXT_PUBLIC_SUPABASE_URL=https://ydogoylwenufckscqijp.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Inlkb2dveWx3ZW51ZmNrc2NxaWpwIiwicm9sZSI6ImFub24iLCJpYXQiOjE3MzY3MzUxNjYsImV4cCI6MjA1MjMxMTE2Nn0.Oy0K0aalki4e4b5h8caHYdWxZVKB6IWDDYQ3zvCUu4Y
SUPABASE_JWT_SECRET=SDqZb9KKouY+7YccWBXxnJdvsGcDLUHrQnZlUo5z8hE0clHn9aArY3RzHra/TCSalpYmcEEW/xmCTbGmVs0CLQ==
SUPABASE_JWKS_URL=https://ydogoylwenufckscqijp.supabase.co/auth/v1/.well-known/jwks.json
NEXT_PUBLIC_APP_URL=http://localhost:3000

# Add for AgentAPI integration
NEXT_PUBLIC_AGENTAPI_URL=http://localhost:3284
NEXT_PUBLIC_FASTMCP_URL=http://localhost:8000

# OAuth configuration (for testing)
NEXT_PUBLIC_OAUTH_ENABLED=true
NEXT_PUBLIC_OAUTH_PROVIDERS=github,google,azure,auth0

# Feature flags
NEXT_PUBLIC_ENABLE_MCP_OAUTH=true
NEXT_PUBLIC_ENABLE_MCP_MANAGEMENT=true
```

---

## Request/Response Flow Examples

### Example 1: OAuth Token Acquisition

**Frontend Initiates**:
```
POST http://localhost:3284/api/mcp/oauth/init
{
  "provider": "github",
  "redirect_uri": "http://localhost:3000/oauth/callback"
}

Response:
{
  "oauth_url": "https://github.com/login/oauth/authorize?...",
  "state": "abc123xyz...def" (32 bytes)
}
```

**User authorizes on GitHub**, gets redirected:
```
GET http://localhost:3000/oauth/callback?code=abc123&state=abc123xyz...def
```

**Frontend calls backend**:
```
POST http://localhost:3284/api/mcp/oauth/callback
{
  "code": "abc123",
  "state": "abc123xyz...def",
  "provider": "github"
}

Response:
{
  "access_token": "ghu_abc123xyz...",
  "refresh_token": "ghr_refresh123...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**Frontend stores encrypted**:
```javascript
// localStorage
{
  "oauth:github:token": "encrypted[access_token]",
  "oauth:github:refresh": "encrypted[refresh_token]",
  "oauth:github:expires_at": "2025-10-24T09:00:00Z"
}
```

---

### Example 2: MCP Configuration Creation

**Frontend creates configuration**:
```
POST http://localhost:3284/api/v1/mcp/configurations
Authorization: Bearer [jwt-token-from-supabase]
{
  "name": "GitHub MCP",
  "url": "http://localhost:8000/mcp/connect",
  "provider": "github",
  "auth_token": "ghu_abc123xyz..." // Encrypted by backend
}

Response:
{
  "id": "mcp-config-uuid",
  "name": "GitHub MCP",
  "provider": "github",
  "created_at": "2025-10-24T08:00:00Z"
}
```

**List configurations**:
```
GET http://localhost:3284/api/v1/mcp/configurations
Authorization: Bearer [jwt-token]

Response:
[
  {
    "id": "mcp-config-uuid",
    "name": "GitHub MCP",
    "provider": "github",
    "created_by": "current-user",
    "org_scoped": false
  }
]
```

---

### Example 3: Circuit Breaker in Action

**Normal operation** (Closed state):
```
Request 1-5: ✅ Success (fast path)
Request 6: ❌ Fails
Request 7: ❌ Fails
Request 8: ❌ Fails
Request 9: ❌ Fails
Request 10: ❌ Fails (5th failure → Circuit Opens)

Circuit state: OPEN
```

**Circuit open** (Fast-fail):
```
Request 11-15: ❌ Fail immediately (no attempt)
  Error: "Circuit breaker is open for mcp_operations"
  Duration: ~1ms per request (no timeout waiting)
```

**Circuit half-open** (After 30 second timeout):
```
Request 16: ✅ Success (test request)
Request 17-18: ✅ Success

Circuit state: CLOSED (after 2 successes)
Request 19+: ✅ Resume normal operation
```

---

## Monitoring & Debugging

### Prometheus Metrics (Port 9090)

Access `http://localhost:9090/metrics`:

```
# HTTP metrics
http_requests_total{method="GET",endpoint="/health",status="200"} 150
http_request_duration_seconds{endpoint="/api/v1/mcp/configurations",quantile="0.95"} 0.045

# MCP metrics
mcp_connections_active 3
mcp_tools_executed_total 42

# Redis metrics
redis_operations_total{operation="GET"} 1000
redis_operation_duration_seconds{operation="GET",quantile="0.95"} 0.001

# Circuit breaker metrics
circuit_breaker_state{name="mcp_operations"} 0 (closed)
circuit_breaker_failures_total{name="mcp_operations"} 15

# Rate limit metrics
rate_limit_requests_total{user_id="user123"} 60
rate_limit_rejected_total{user_id="user123"} 0
```

### Logs (Structured JSON)

```json
{
  "timestamp": "2025-10-24T08:30:00Z",
  "level": "info",
  "request_id": "req-abc123xyz",
  "component": "mcp.handler",
  "action": "create_configuration",
  "user_id": "user-123",
  "org_id": "org-456",
  "duration_ms": 45,
  "status": "success"
}
```

### Health Checks

```
GET http://localhost:3284/health

{
  "status": "UP",
  "components": {
    "database": "UP",
    "redis": "UP",
    "fastmcp": "UP"
  }
}
```

---

## Security Considerations

### Data Encryption

1. **OAuth Tokens**: AES-256-GCM encrypted in Redis + Supabase
2. **HTTPS**: TLS 1.3 for all connections
3. **JWT**: Signed with Supabase private key, verified with public JWKS

### Access Control

1. **Authentication**: JWT bearer tokens from Supabase
2. **Authorization**: Role-based (admin/user)
3. **Isolation**: Row-Level Security policies in PostgreSQL
4. **Audit**: All actions logged in audit_logs table

### API Security

1. **Input Validation**: All inputs validated and sanitized
2. **Rate Limiting**: 60 req/min per user
3. **CSRF Protection**: State parameter in OAuth flows
4. **SQL Injection**: Parameterized queries via sqlc

---

## Scaling Considerations

### Current Setup (MVP)
- Single agentapi instance (port 3284)
- Single PostgreSQL database
- Single Redis instance
- Success rate: 99%+
- Throughput: 6K req/s

### Phase 2 (Production)
- Kubernetes cluster (3+ replicas)
- PostgreSQL with read replicas
- Redis cluster with HA
- Horizontal Pod Autoscaler
- Expected: 10K+ req/s

### Phase 3 (Enterprise)
- Multi-region deployment
- Global load balancer
- CDN for static assets
- gRPC for high-throughput operations

---

**Ready to begin local testing!**

For detailed testing procedures, see [LOCAL_TESTING_GUIDE.md](./LOCAL_TESTING_GUIDE.md)


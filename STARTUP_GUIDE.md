# AgentAPI - Startup Guide

## Quick Start (Recommended)

### 1. Build the Application
```bash
go build -o chatserver ./cmd/chatserver/main.go
```

### 2. Configure Environment
Edit `.env` with your credentials:
```bash
# Required: AuthKit (WorkOS SSO)
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_YOUR_CLIENT_ID

# Required: Supabase Database
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-jwt-key

# Required: GCP VertexAI
VERTEX_AI_PROJECT_ID=your-gcp-project-id
VERTEX_AI_LOCATION=us-central1

# Optional: Agent Configuration
CCROUTER_PATH=/opt/homebrew/bin/ccr
PRIMARY_AGENT=ccrouter
```

### 3. Start the Server

**Option A: Using start script (Recommended)**
```bash
./start.sh
```

**Option B: Manual environment sourcing**
```bash
set -a; source .env; set +a
./chatserver
```

**Option C: Export variables individually**
```bash
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_YOUR_CLIENT_ID
export SUPABASE_URL=https://your-project.supabase.co
export SUPABASE_SERVICE_ROLE_KEY=your-service-role-jwt-key
export CCROUTER_PATH=/opt/homebrew/bin/ccr
./chatserver
```

---

## Troubleshooting

### ❌ "AUTHKIT_JWKS_URL environment variable is required"

**Cause**: Required environment variables not set

**Solution**:
1. Check `.env` has all required variables
2. Use `./start.sh` which validates env vars automatically
3. Or manually export before starting:
```bash
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_YOUR_CLIENT_ID
```

### ❌ "failed to connect to Supabase"

**Cause**: Supabase credentials invalid or Supabase is unreachable

**Solution**:
1. Verify `SUPABASE_URL` is correct (e.g., `https://your-project.supabase.co`)
2. Verify `SUPABASE_SERVICE_ROLE_KEY` is a valid JWT token
3. Check internet connection to Supabase
4. Check that database tables exist (see Database Setup section)

### ❌ "dial tcp [...]:5432: connect: no route to host"

**Cause**: DATABASE_URL is being used instead of Supabase client (IPv6 issue)

**Solution**:
1. Ensure `DATABASE_URL` is NOT set or commented out in `.env`
2. Ensure `SUPABASE_URL` and `SUPABASE_SERVICE_ROLE_KEY` ARE set
3. Use `./start.sh` which properly sources `.env`

### ❌ "CCRouter agent health check failed" or "no healthy agents available"

**Cause**: CCRouter binary not found or not working

**Solution**:
1. Install CCRouter: `brew install coder/tap/ccr`
2. Verify path: `which ccr`
3. Update `CCROUTER_PATH` in `.env` if needed
4. Test manually: `ccr --version`

### ❌ "listen tcp :3284: bind: address already in use"

**Cause**: Port 3284 already in use by another process

**Solution**:
```bash
# Kill existing process
lsof -i :3284 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Or use different port
export AGENTAPI_PORT=3285
./chatserver
```

---

## Verifying the Server is Running

Once the server starts successfully, you should see:

```
time=2025-10-24T15:59:51.977 level=INFO msg="initializing chat API"
time=2025-10-24T15:59:52.222 level=INFO msg="server listening" port=3284
```

### Test Health Endpoint
```bash
curl http://localhost:3284/health
```

Expected response:
```json
{"status": "ok"}
```

### Test with JWT Token
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/v1/models
```

---

## Environment Variables Reference

### Required
| Variable | Example | Purpose |
|----------|---------|---------|
| `AUTHKIT_JWKS_URL` | `https://api.workos.com/sso/jwks/client_...` | JWT validation |
| `SUPABASE_URL` | `https://project.supabase.co` | Supabase endpoint |
| `SUPABASE_SERVICE_ROLE_KEY` | JWT token | Database auth |
| `VERTEX_AI_PROJECT_ID` | `my-gcp-project` | GCP project for VertexAI |
| `VERTEX_AI_LOCATION` | `us-central1` | GCP region |

### Optional
| Variable | Default | Purpose |
|----------|---------|---------|
| `CCROUTER_PATH` | `/usr/local/bin/ccr` | CCRouter binary location |
| `DROID_PATH` | `/usr/local/bin/droid` | Droid agent binary (if available) |
| `PRIMARY_AGENT` | `ccrouter` | Agent to use by default |
| `FALLBACK_ENABLED` | `true` | Use secondary agent on failure |
| `AGENTAPI_PORT` | `3284` | Server port |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `AUDIT_ENABLED` | `true` | Enable audit logging |
| `LOG_LEVEL` | `info` | Log verbosity |

---

## Database Setup

### Verify Tables Exist
The following tables should be created in your Supabase project:

```sql
SELECT tablename FROM pg_tables WHERE schemaname = 'public'
```

Expected tables:
- `agents` - Agent definitions
- `models` - LLM models catalog
- `chat_sessions` - Conversation sessions
- `chat_messages` - Message history
- `agent_health` - Agent status tracking

### Fix Table Ownership (if needed)
If you see "permission denied" errors, run in Supabase SQL Editor:

```sql
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;
```

---

## Using the API

### 1. Chat Completion

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "max_tokens": 1000,
    "temperature": 0.7
  }'
```

### 2. List Available Models

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/v1/models
```

### 3. Platform Admin Endpoints

```bash
# Get platform stats
curl -H "Authorization: Bearer ADMIN_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/stats

# View audit log
curl -H "Authorization: Bearer ADMIN_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/audit
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Client Application                   │
└────────────────────────┬────────────────────────────────┘
                         │ HTTP/JWT
                         ▼
┌──────────────────────────────────────────────────────────┐
│              AgentAPI Chat Server (Go)                   │
├──────────────────────────────────────────────────────────┤
│  • AuthKit Validator (JWT validation)                    │
│  • Chat Handler (request processing)                     │
│  • Chat Orchestrator (agent routing)                     │
│  • Agent Executors (CCRouter, Droid)                     │
│  • Audit Logger (compliance)                             │
│  • Metrics Registry (observability)                      │
└──────┬─────────────────┬──────────────┬──────────────────┘
       │                 │              │
       ▼                 ▼              ▼
   [Supabase]      [VertexAI]    [Upstash Redis]
   (Database)      (LLM API)      (Cache/Session)
```

### Component Responsibilities

**AuthKit Validator** (`lib/auth/authkit.go`):
- Validates JWT tokens from WorkOS
- Checks platform admin status
- Extracts user/org context

**Chat Handler** (`lib/chat/handler.go`):
- Processes chat completion requests
- Validates input parameters
- Returns responses in OpenAI format

**Chat Orchestrator** (`lib/chat/orchestrator.go`):
- Routes requests to agents
- Handles agent failover
- Manages timeout/retry logic

**Agents**:
- **CCRouter** (`lib/agents/ccrouter.go`): VertexAI/Gemini
- **Droid** (`lib/agents/droid.go`): OpenRouter (multi-model)

---

## Monitoring

### Logs
All output goes to stdout in structured JSON format:
```
time=2025-10-24T15:59:51.977-07:00 level=INFO msg="initializing chat API"
```

Filter by level:
```bash
./start.sh 2>&1 | grep "level=ERROR"
./start.sh 2>&1 | grep "level=WARN"
```

### Prometheus Metrics
If metrics are enabled, metrics available at:
```bash
curl http://localhost:3284/metrics
```

### Audit Log
View audit events:
```bash
curl -H "Authorization: Bearer ADMIN_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/audit
```

---

## Performance Tuning

### Increase Agent Timeout
```bash
export AGENT_TIMEOUT=10m
./start.sh
```

### Adjust Token Limits
```bash
export MAX_TOKENS=8192
export DEFAULT_TEMP=0.5
./start.sh
```

### Disable Metrics (Lighter Load)
```bash
export METRICS_ENABLED=false
./start.sh
```

---

## Next Steps

1. ✅ Verify server starts with `./start.sh`
2. ✅ Test health endpoint with `curl http://localhost:3284/health`
3. ✅ Obtain JWT token from your auth provider
4. ✅ Test chat endpoint with real JWT
5. ✅ Deploy to production using Docker

For deployment, see [PRODUCTION_DEPLOYMENT_GUIDE.md](./PRODUCTION_DEPLOYMENT_GUIDE.md)

---

## Support

- **Coder CCRouter**: https://github.com/coder/ccrouter
- **VertexAI Docs**: https://cloud.google.com/vertex-ai/generative-ai/docs
- **Supabase Docs**: https://supabase.com/docs
- **WorkOS Docs**: https://workos.com/docs


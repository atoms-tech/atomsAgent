# AgentAPI - Quick Start Guide

## Overview
AgentAPI is a production-ready LLM API server with:
- **Multi-agent support**: CCRouter (VertexAI/Gemini) + Droid (OpenRouter)
- **Multi-tenant architecture**: Supabase for data + Upstash for caching
- **Agent orchestration**: Automatic failover and health monitoring
- **Platform administration**: Admin role management and audit logging

---

## Prerequisites

### Install Dependencies
```bash
# Install CCRouter (VertexAI Gemini support)
brew install coder/tap/ccr
# or download from: https://github.com/coder/ccrouter/releases

# Install Go (if not already installed)
brew install go

# GCP Authentication
gcloud auth application-default login
```

### Environment Setup
Copy and configure `.env`:
```bash
cp .env.example .env
```

Edit `.env` with your values:
```bash
# Required: AuthKit (WorkOS SSO)
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_YOUR_CLIENT_ID

# Required: Supabase PostgreSQL
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-jwt-key

# Required: VertexAI GCP
VERTEX_AI_PROJECT_ID=your-gcp-project
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_USE_APPLICATION_DEFAULT=true

# Optional: CCRouter binary path
CCROUTER_PATH=/opt/homebrew/bin/ccr
PRIMARY_AGENT=ccrouter

# Optional: Server port
AGENTAPI_PORT=3284
```

---

## Running the Server

### Build from Source
```bash
go build -o ./bin/chatserver ./cmd/chatserver/main.go
```

### Run the Server
```bash
./bin/chatserver
```

**Expected Output**:
```
time=2025-10-24T15:44:55.436-07:00 level=INFO msg="Starting AgentAPI Chat Server"
time=2025-10-24T15:44:55.483-07:00 level=INFO msg="configuration loaded"
time=2025-10-24T15:44:55.483-07:00 level=INFO msg="initializing chat API"
time=2025-10-24T15:44:55.898-07:00 level=INFO msg="CCRouter agent initialized successfully"
time=2025-10-24T15:44:55.899-07:00 level=INFO msg="server listening" port=3284
```

---

## API Endpoints

### Health Check
```bash
curl http://localhost:3284/health
```

### List Available Models
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/v1/models
```

### Chat Completions (LLM API)
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello, what can you do?"}
    ],
    "max_tokens": 1000,
    "temperature": 0.7
  }'
```

### Platform Admin Endpoints
```bash
# Get platform statistics
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/stats

# View audit log
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/audit

# List platform admins
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:3284/api/v1/platform/admins
```

---

## Available Models

### Via CCRouter (VertexAI)
- `gemini-1.5-pro` - High performance
- `gemini-1.5-flash` - Fast & efficient
- `gemini-2.0-pro` - Latest (preview)

### Via Droid (OpenRouter)
- `claude-3-opus` - Most capable
- `claude-3.5-sonnet` - Best balance
- `gpt-4-turbo` - Open AI latest
- `gpt-4o` - OpenAI omni

---

## Authentication

### Get JWT Token
AgentAPI uses **WorkOS/AuthKit** for authentication.

1. Set up WorkOS organization
2. Configure your JWKS endpoint in `.env`
3. Obtain JWT token from your auth provider
4. Include in all requests:
   ```bash
   Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
   ```

### User Roles
- **Platform Admin**: Can manage all organizations
- **Organization Admin**: Can manage organization members
- **Member**: Can use chat API
- **Viewer**: Read-only access

---

## Database Setup

### Verify Supabase Tables
The following tables are automatically used:

**Core Tables** (created during schema migration):
- `agents` - Agent definitions (CCRouter, Droid)
- `models` - LLM model catalog
- `chat_sessions` - Conversation sessions
- `chat_messages` - Message history
- `agent_health` - Agent status tracking

**Existing Tables** (leveraged from Supabase):
- `profiles` - User profiles
- `organizations` - Organization management
- `organization_members` - User-org relationships
- `mcp_sessions` - Session storage with OAuth

### Fix Table Ownership
If tables show "permission denied" errors:

```sql
-- Run in Supabase SQL Editor
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;
```

---

## Monitoring & Debugging

### Check Server Status
```bash
curl http://localhost:3284/status
```

### View Logs
Logs are output to stdout with JSON format:
- `level=INFO` - Normal operations
- `level=WARN` - Warnings (non-critical)
- `level=ERROR` - Errors (check these!)

### Common Issues

**Issue**: "AUTHKIT_JWKS_URL is required"
```bash
# Solution: Export environment variable
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_YOUR_ID
```

**Issue**: "CCRouter agent health check failed"
```bash
# Solution: Verify binary path
which ccr
# Update CCROUTER_PATH in .env if needed
```

**Issue**: "Connection refused" to Supabase
```bash
# Solution: Check credentials
echo $SUPABASE_URL
echo $SUPABASE_SERVICE_ROLE_KEY
# Verify both are set in .env
```

---

## Configuration Reference

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `AUTHKIT_JWKS_URL` | ✅ Yes | - | JWT validation |
| `SUPABASE_URL` | ✅ Yes | - | Database connection |
| `SUPABASE_SERVICE_ROLE_KEY` | ✅ Yes | - | Database auth |
| `VERTEX_AI_PROJECT_ID` | ✅ Yes | - | GCP project ID |
| `CCROUTER_PATH` | No | `/usr/local/bin/ccr` | Agent binary |
| `PRIMARY_AGENT` | No | `ccrouter` | Primary agent |
| `AGENTAPI_PORT` | No | `3284` | Server port |
| `METRICS_ENABLED` | No | `true` | Telemetry |
| `AUDIT_ENABLED` | No | `true` | Audit logging |

---

## Production Deployment

### Docker (Recommended)
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o chatserver ./cmd/chatserver/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/chatserver /usr/local/bin/
EXPOSE 3284
CMD ["chatserver"]
```

### Environment Variables (Secrets)
Store these securely (e.g., using GitHub Secrets, GCP Secret Manager, etc.):
- `AUTHKIT_JWKS_URL`
- `SUPABASE_SERVICE_ROLE_KEY`
- `VERTEX_AI_PROJECT_ID`
- GCP credentials file

### Health Checks
Configure your orchestrator to monitor:
```bash
GET /health - HTTP 200 = healthy
GET /status - JSON status details
```

### Scaling
- Stateless: Scale horizontally (no session affinity needed)
- Database: Supabase handles scaling
- Cache: Upstash Redis handles caching
- Agents: Native subprocess scaling

---

## Development

### Run Tests
```bash
go test ./...
```

### Build Binary
```bash
go build -o bin/chatserver ./cmd/chatserver/main.go
```

### Format Code
```bash
go fmt ./...
```

### Run with Hot Reload (Optional)
```bash
go install github.com/cosmtrek/air@latest
air
```

---

## Support & Documentation

- **Coder CCRouter**: https://github.com/coder/ccrouter
- **VertexAI Docs**: https://cloud.google.com/vertex-ai/generative-ai/docs
- **Supabase Docs**: https://supabase.com/docs
- **WorkOS Docs**: https://workos.com/docs

---

## Quick Troubleshooting

| Problem | Solution |
|---------|----------|
| Env vars not loading | Use `source .env` before `./bin/chatserver` |
| "No agent binaries found" | Check `CCROUTER_PATH` and run `which ccr` |
| "Connection refused" on 3284 | Change `AGENTAPI_PORT` or kill existing process |
| "Permission denied" on Supabase tables | Run `ALTER TABLE owner` commands in Supabase SQL Editor |
| GCP auth failing | Run `gcloud auth application-default login` |

---

## Next Steps

1. ✅ Set up `.env` with your credentials
2. ✅ Run `./bin/chatserver`
3. ✅ Test health endpoint: `curl http://localhost:3284/health`
4. ✅ Test chat endpoint with valid JWT token
5. ✅ Monitor logs for any errors
6. 📦 Deploy to production

**Happy chatting! 🚀**

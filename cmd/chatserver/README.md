# AgentAPI Chat Server

A standalone HTTP server for the AgentAPI Chat API with multi-agent support (CCRouter and Droid).

## Features

- **Multi-Agent Support**: CCRouter (VertexAI) and Droid (OpenRouter) backends
- **Health Monitoring**: Built-in health and status endpoints
- **Graceful Shutdown**: Proper signal handling and connection draining
- **Environment-Based Configuration**: No config files needed
- **Structured Logging**: slog-based logging for observability

## Quick Start

### Build

```bash
go build -o chatserver ./cmd/chatserver
```

### Run

```bash
# Set required environment variable
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/YOUR_CLIENT_ID"

# Optional: Set agent paths if not in default locations
export CCROUTER_PATH="/usr/local/bin/ccrouter"
export DROID_PATH="/usr/local/bin/droid"

# Start the server
./chatserver
```

The server will start on port 3284 by default.

## Environment Variables

### Required

- **AUTHKIT_JWKS_URL**: AuthKit JWKS URL for JWT validation
  - Example: `https://api.workos.com/sso/jwks/client_123abc`

### Optional

- **CCROUTER_PATH**: Path to CCRouter binary (default: `/usr/local/bin/ccrouter`)
- **DROID_PATH**: Path to Droid binary (default: `/usr/local/bin/droid`)
- **PRIMARY_AGENT**: Primary agent to use: `ccrouter` or `droid` (default: `ccrouter`)
- **FALLBACK_ENABLED**: Enable fallback to secondary agent (default: `true`)
- **PORT**: Server port (default: `3284`)
- **METRICS_ENABLED**: Enable metrics collection (default: `true`)
- **AUDIT_ENABLED**: Enable audit logging (default: `true`)

## API Endpoints

### Health Check

```bash
curl http://localhost:3284/health
```

Response:
```json
{
  "status": "healthy",
  "agents": ["ccrouter", "droid"],
  "primary": "ccrouter"
}
```

### Server Status

```bash
curl http://localhost:3284/status
```

Response:
```json
{
  "status": "running",
  "uptime": "active",
  "configured_agents": {
    "ccrouter": true,
    "droid": true
  }
}
```

### Chat Completions (Coming Soon)

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### List Models (Coming Soon)

```bash
curl http://localhost:3284/v1/models \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Testing

### Unit Tests

```bash
go test ./cmd/chatserver/...
```

### Integration Tests

```bash
# Create mock agent binaries for testing
mkdir -p /tmp/test-agents
touch /tmp/test-agents/ccrouter /tmp/test-agents/droid
chmod +x /tmp/test-agents/*

# Run server with mock agents
AUTHKIT_JWKS_URL=https://example.com/jwks \
CCROUTER_PATH=/tmp/test-agents/ccrouter \
DROID_PATH=/tmp/test-agents/droid \
./chatserver

# In another terminal, test endpoints
curl http://localhost:3284/health
curl http://localhost:3284/status
```

## Configuration Examples

### Development (CCRouter only)

```bash
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/dev_client"
export CCROUTER_PATH="./bin/ccrouter"
export PRIMARY_AGENT="ccrouter"
export PORT="3000"
./chatserver
```

### Production (Both agents with fallback)

```bash
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/prod_client"
export CCROUTER_PATH="/usr/local/bin/ccrouter"
export DROID_PATH="/usr/local/bin/droid"
export PRIMARY_AGENT="ccrouter"
export FALLBACK_ENABLED="true"
export METRICS_ENABLED="true"
export AUDIT_ENABLED="true"
export PORT="3284"
./chatserver
```

## Architecture

The chat server follows a clean architecture pattern:

```
cmd/chatserver/main.go (Entry point)
├── Configuration Loading
├── Agent Validation
├── HTTP Router Setup
├── Endpoint Registration
│   ├── /health (Health check)
│   ├── /status (Server status)
│   ├── /v1/chat/completions (Chat API)
│   └── /v1/models (Model listing)
└── Graceful Shutdown Handler
```

## Known Issues & Roadmap

### Current Limitations

1. **Chat API Not Fully Integrated**: The `/v1/chat/completions` and `/v1/models` endpoints return placeholder responses
2. **API Signature Mismatches**: The integration code in `pkg/server/setup.go` and `lib/chat` has API signature issues that need to be resolved:
   - `audit.Logger` should be `audit.AuditLogger`
   - `metrics.MetricsClient` should be `metrics.MetricsRegistry`
   - `CircuitBreaker.Execute` signature needs correction
3. **Import Cycle Fixed**: The circular dependency between `lib/agents` and `lib/chat` has been resolved by moving `ModelInfo` to `lib/agents/interface.go`

### Next Steps

1. Fix API signature issues in:
   - `pkg/server/setup.go`
   - `lib/chat/handler.go`
   - `lib/chat/orchestrator.go`
2. Complete integration of `SetupChatAPI` function
3. Implement authentication middleware using AuthKit
4. Add metrics collection and export
5. Add audit logging for compliance
6. Implement rate limiting and request validation
7. Add comprehensive integration tests

## Troubleshooting

### "no agent binaries found"

Ensure at least one agent binary exists and is executable:

```bash
which ccrouter  # Should return path
ls -la /usr/local/bin/ccrouter  # Check permissions
```

### "AUTHKIT_JWKS_URL environment variable is required"

Set the required environment variable:

```bash
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/YOUR_CLIENT_ID"
```

### Port already in use

Change the port:

```bash
export PORT="3285"
./chatserver
```

## Contributing

When making changes:

1. Fix import cycles first (✅ Already fixed)
2. Update API signatures to match actual implementations
3. Add tests for new functionality
4. Update this documentation

## License

See the main repository LICENSE file.

# Chat Server Quick Start Guide

## TL;DR

```bash
# Build
go build -o chatserver ./cmd/chatserver

# Run
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/YOUR_CLIENT_ID"
./chatserver

# Test
curl http://localhost:3284/health
```

## Requirements

1. **Go 1.21+** installed
2. **AuthKit JWKS URL** from WorkOS
3. **At least one agent binary**: CCRouter or Droid

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTHKIT_JWKS_URL` | ✅ Yes | - | AuthKit JWKS URL for JWT validation |
| `CCROUTER_PATH` | No | `/usr/local/bin/ccrouter` | Path to CCRouter binary |
| `DROID_PATH` | No | `/usr/local/bin/droid` | Path to Droid binary |
| `PRIMARY_AGENT` | No | `ccrouter` | Primary agent: `ccrouter` or `droid` |
| `PORT` | No | `3284` | Server port |

## Endpoints

| Endpoint | Method | Auth | Status | Description |
|----------|--------|------|--------|-------------|
| `/health` | GET | No | ✅ Working | Health check with agent status |
| `/status` | GET | No | ✅ Working | Server runtime status |
| `/v1/chat/completions` | POST | Yes | ⚠️ Placeholder | Chat completion (needs API fixes) |
| `/v1/models` | GET | Yes | ⚠️ Placeholder | List available models (needs API fixes) |

## Development Setup

### Option 1: With Real Agents

```bash
# Install CCRouter and/or Droid first
# Then run:
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/dev_123"
export CCROUTER_PATH="/path/to/ccrouter"
export DROID_PATH="/path/to/droid"
./chatserver
```

### Option 2: With Mock Agents (Testing)

```bash
# Create mock binaries
mkdir -p /tmp/test-agents
touch /tmp/test-agents/ccrouter /tmp/test-agents/droid
chmod +x /tmp/test-agents/*

# Run server
export AUTHKIT_JWKS_URL="https://example.com/jwks"
export CCROUTER_PATH="/tmp/test-agents/ccrouter"
export DROID_PATH="/tmp/test-agents/droid"
./chatserver
```

## Testing Commands

```bash
# Health check (should return 200)
curl -i http://localhost:3284/health

# Expected response:
# {"status":"healthy","agents":["ccrouter","droid"],"primary":"ccrouter"}

# Status check
curl -i http://localhost:3284/status

# Expected response:
# {"status":"running","uptime":"active","configured_agents":{"ccrouter":true,"droid":true}}
```

## Common Issues

### "AUTHKIT_JWKS_URL environment variable is required"

**Solution**: Export the required variable:
```bash
export AUTHKIT_JWKS_URL="https://api.workos.com/sso/jwks/YOUR_CLIENT_ID"
```

### "no agent binaries found"

**Solution**: Either:
1. Install real agents:
   ```bash
   # Install CCRouter
   go install github.com/example/ccrouter@latest

   # Or install Droid
   go install github.com/example/droid@latest
   ```

2. Use mock agents for testing (see Option 2 above)

3. Specify custom paths:
   ```bash
   export CCROUTER_PATH="/custom/path/to/ccrouter"
   ```

### "bind: address already in use"

**Solution**: Change the port:
```bash
export PORT="3285"
./chatserver
```

## Production Deployment

### Docker Example

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o chatserver ./cmd/chatserver

FROM debian:bookworm-slim
COPY --from=builder /app/chatserver /usr/local/bin/
ENV AUTHKIT_JWKS_URL=""
ENV PORT=3284
EXPOSE 3284
CMD ["chatserver"]
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  chatserver:
    build: .
    ports:
      - "3284:3284"
    environment:
      - AUTHKIT_JWKS_URL=${AUTHKIT_JWKS_URL}
      - CCROUTER_PATH=/usr/local/bin/ccrouter
      - DROID_PATH=/usr/local/bin/droid
      - PRIMARY_AGENT=ccrouter
      - FALLBACK_ENABLED=true
    volumes:
      - ./agents:/usr/local/bin
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3284/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Systemd Service

```ini
[Unit]
Description=AgentAPI Chat Server
After=network.target

[Service]
Type=simple
User=agentapi
WorkingDirectory=/opt/agentapi
Environment="AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID"
Environment="PORT=3284"
ExecStart=/opt/agentapi/chatserver
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Next Steps

1. **Read the full documentation**: `README.md`
2. **Review integration report**: `/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/CHAT_API_INTEGRATION_REPORT.md`
3. **Fix API signatures** to enable full chat functionality
4. **Add authentication** for production use
5. **Enable metrics and audit logging**

## Links

- [Full README](README.md)
- [Integration Report](../../CHAT_API_INTEGRATION_REPORT.md)
- [Example Integration Code](../../pkg/server/example_integration.go)

# Chat API Integration - Quick Reference

## Complete Code Snippets

### 1. Standalone Chat Server (Recommended)

Create `cmd/chatserver/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/agentapi/pkg/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load and validate config
	config, err := server.LoadConfigFromEnv()
	if err != nil {
		logger.Error("config error", "error", err)
		os.Exit(1)
	}

	if err := server.ValidateConfig(config, logger); err != nil {
		logger.Error("validation error", "error", err)
		os.Exit(1)
	}

	// Setup server
	mux := http.NewServeMux()
	components, err := server.SetupChatAPI(mux, logger, config)
	if err != nil {
		logger.Error("setup error", "error", err)
		os.Exit(1)
	}

	server.LogStartupInfo(logger, config, components)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("listening", "port", config.Port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	<-shutdown
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	components.GracefulShutdown(ctx, logger)
	logger.Info("stopped")
}
```

### 2. Modify Existing Server (cmd/server/server.go)

#### Add to imports:
```go
import (
    // ... existing imports ...
    "github.com/coder/agentapi/pkg/server"
)
```

#### Add flag to CreateServerCmd():
```go
// Inside CreateServerCmd(), add this to flagSpecs:
{
    name:         "chat-api",
    shorthand:    "C",
    defaultValue: false,
    usage:        "Enable chat API endpoints",
    flagType:     "bool",
},
```

#### Modify runServer() function:

```go
func runServer(ctx context.Context, logger *slog.Logger, argsToPass []string) error {
    // ... existing server setup code ...

    // NEW: Setup chat API if enabled
    if viper.GetBool("chat-api") {
        if err := initializeChatAPI(logger, srv.GetMux()); err != nil {
            logger.Warn("chat API disabled", "error", err)
        }
    }

    // ... rest of existing code ...
}
```

#### Add helper function:
```go
func initializeChatAPI(logger *slog.Logger, mux *http.ServeMux) error {
    config, err := server.LoadConfigFromEnv()
    if err != nil {
        return err
    }

    if err := server.ValidateConfig(config, logger); err != nil {
        return err
    }

    components, err := server.SetupChatAPI(mux, logger, config)
    if err != nil {
        return err
    }

    server.LogStartupInfo(logger, config, components)
    return nil
}
```

## Environment Variables

### Minimal Setup (.env)
```bash
# Required
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID

# Agent paths (adjust to your installation)
CCROUTER_PATH=/usr/local/bin/ccrouter
DROID_PATH=/usr/local/bin/droid
```

### Full Configuration (.env)
```bash
# Authentication
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID

# Agent Binaries
CCROUTER_PATH=/usr/local/bin/ccrouter
DROID_PATH=/usr/local/bin/droid

# Agent Configuration
PRIMARY_AGENT=ccrouter        # or "droid"
FALLBACK_ENABLED=true

# Observability
METRICS_ENABLED=true
AUDIT_ENABLED=true

# Server
PORT=3284
```

## Test Commands

### 1. Health Check
```bash
curl http://localhost:3284/health
```

### 2. List Models (requires auth)
```bash
curl -H "Authorization: Bearer YOUR_JWT" \
     http://localhost:3284/v1/models
```

### 3. Chat Completion
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [{"role": "user", "content": "Say hello"}],
    "temperature": 0.7
  }'
```

### 4. Streaming Chat
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

## Running the Server

### Option 1: Standalone
```bash
# Set environment variables
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_ID
export CCROUTER_PATH=/path/to/ccrouter
export DROID_PATH=/path/to/droid

# Run
go run cmd/chatserver/main.go
```

### Option 2: With Existing Server
```bash
# Using the modified server command
go run main.go server --chat-api your-agent-here
```

### Option 3: Docker
```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o chatserver cmd/chatserver/main.go

FROM ubuntu:22.04
COPY --from=builder /app/chatserver /usr/local/bin/
COPY ccrouter /usr/local/bin/
COPY droid /usr/local/bin/

ENV AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_ID
ENV CCROUTER_PATH=/usr/local/bin/ccrouter
ENV DROID_PATH=/usr/local/bin/droid

EXPOSE 3284
CMD ["/usr/local/bin/chatserver"]
```

## Troubleshooting

### "no healthy agents available"
```bash
# Check agent binaries exist
ls -l $CCROUTER_PATH
ls -l $DROID_PATH

# Make sure they're executable
chmod +x $CCROUTER_PATH
chmod +x $DROID_PATH

# Test agent directly
$CCROUTER_PATH --version
$DROID_PATH --version
```

### "AUTHKIT_JWKS_URL environment variable is required"
```bash
# Set the variable
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID

# Or use a .env file
echo "AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_ID" > .env
```

### "401 Unauthorized"
1. Get a valid JWT from your AuthKit setup
2. Include it in requests: `-H "Authorization: Bearer YOUR_JWT"`
3. Verify token hasn't expired
4. Check JWKS URL is accessible from server

## Key Files

| File | Purpose |
|------|---------|
| `pkg/server/setup.go` | Main integration code |
| `pkg/server/example_integration.go` | Integration patterns |
| `pkg/server/INTEGRATION_GUIDE.md` | Detailed guide |
| `lib/chat/orchestrator.go` | Agent orchestration |
| `lib/chat/handler.go` | HTTP handlers |
| `lib/agents/ccrouter.go` | CCRouter implementation |
| `lib/agents/droid.go` | Droid implementation |
| `lib/middleware/authkit.go` | Authentication |

## Next Steps

1. ✅ Set `AUTHKIT_JWKS_URL` environment variable
2. ✅ Install agent binaries (ccrouter, droid)
3. ✅ Create standalone server or modify existing
4. ✅ Test health endpoint
5. ✅ Test with valid JWT token
6. ✅ Review logs for any warnings
7. ✅ Set up monitoring/alerting
8. ✅ Deploy to production

## Support

For issues:
1. Check logs for detailed error messages
2. Verify environment variables are set correctly
3. Test agent binaries independently
4. Review the full integration guide
5. Check JWT token is valid and contains required claims

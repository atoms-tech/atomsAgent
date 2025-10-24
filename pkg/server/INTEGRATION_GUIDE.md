# Chat API Server Integration Guide

This guide shows how to integrate the chat API into the existing AgentAPI server.

## Quick Start

### Option 1: Standalone Server (Recommended for Testing)

Create a new file `cmd/chatserver/main.go`:

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
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting AgentAPI Chat Server")

	// Load configuration
	config, err := server.LoadConfigFromEnv()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := server.ValidateConfig(config, logger); err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// Create HTTP router
	mux := http.NewServeMux()

	// Setup chat API
	components, err := server.SetupChatAPI(mux, logger, config)
	if err != nil {
		logger.Error("failed to setup chat API", "error", err)
		os.Exit(1)
	}

	// Log startup info
	server.LogStartupInfo(logger, config, components)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", config.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for shutdown
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-shutdown:
		logger.Info("shutdown initiated")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
		}

		if err := components.GracefulShutdown(ctx, logger); err != nil {
			logger.Error("component shutdown error", "error", err)
		}
	}

	logger.Info("server stopped")
}
```

### Option 2: Integrate into Existing Server

Modify `cmd/server/server.go`:

#### Step 1: Import the server package

```go
import (
    // ... existing imports ...
    "github.com/coder/agentapi/pkg/server"
)
```

#### Step 2: Update CreateServerCmd function

Add a new flag for enabling chat API:

```go
func CreateServerCmd() *cobra.Command {
    serverCmd := &cobra.Command{
        // ... existing code ...
        Run: func(cmd *cobra.Command, args []string) {
            // ... existing code ...

            // NEW: Setup chat API if enabled
            setupChatAPI := viper.GetBool("chat-api-enabled")
            if setupChatAPI {
                if err := setupChatAPIRoutes(logger); err != nil {
                    logger.Warn("chat API setup failed", "error", err)
                }
            }

            // ... rest of existing code ...
        },
    }

    // Add chat API flag
    serverCmd.Flags().Bool("chat-api-enabled", false, "Enable chat API endpoints")
    viper.BindPFlag("chat-api-enabled", serverCmd.Flags().Lookup("chat-api-enabled"))

    return serverCmd
}
```

#### Step 3: Add setupChatAPIRoutes helper

```go
func setupChatAPIRoutes(logger *slog.Logger) error {
    // Load chat API config
    chatConfig, err := server.LoadConfigFromEnv()
    if err != nil {
        return fmt.Errorf("failed to load chat config: %w", err)
    }

    // Validate config
    if err := server.ValidateConfig(chatConfig, logger); err != nil {
        return fmt.Errorf("invalid chat config: %w", err)
    }

    // Create a new mux for chat routes
    // You'll need to get your server's mux here
    // This depends on your server implementation
    mux := getServerMux() // Implement based on your server

    // Setup chat API
    components, err := server.SetupChatAPI(mux, logger, chatConfig)
    if err != nil {
        return fmt.Errorf("failed to setup chat API: %w", err)
    }

    // Log startup info
    server.LogStartupInfo(logger, chatConfig, components)

    return nil
}
```

## Environment Variables

Create a `.env` file or set these environment variables:

### Required

```bash
# AuthKit JWKS URL for JWT validation
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID
```

### Optional - Agent Paths

```bash
# Path to CCRouter binary (default: /usr/local/bin/ccrouter)
CCROUTER_PATH=/usr/local/bin/ccrouter

# Path to Droid binary (default: /usr/local/bin/droid)
DROID_PATH=/usr/local/bin/droid
```

### Optional - Agent Configuration

```bash
# Primary agent to use (default: ccrouter)
# Options: "ccrouter" or "droid"
PRIMARY_AGENT=ccrouter

# Enable fallback to secondary agent on failure (default: true)
FALLBACK_ENABLED=true
```

### Optional - Observability

```bash
# Enable Prometheus metrics (default: true)
METRICS_ENABLED=true

# Enable audit logging (default: true)
AUDIT_ENABLED=true
```

### Optional - Server

```bash
# Server port (default: 3284)
PORT=3284
```

## Testing the Integration

### 1. Start the server

```bash
# Set required environment variables
export AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID
export CCROUTER_PATH=/path/to/ccrouter
export DROID_PATH=/path/to/droid

# Run the server
go run cmd/chatserver/main.go
```

### 2. Check health endpoint

```bash
curl http://localhost:3284/health
```

Expected response:
```json
{
  "status": "healthy",
  "agents": ["ccrouter", "droid"]
}
```

### 3. List available models

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     http://localhost:3284/v1/models
```

### 4. Send a chat completion request

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

## Troubleshooting

### Agent Not Available

If you see warnings like:
```
WARN CCRouter binary not found, agent disabled path=/usr/local/bin/ccrouter
```

Solutions:
1. Set the correct path: `export CCROUTER_PATH=/actual/path/to/ccrouter`
2. Install the agent binary
3. Use the other agent by setting `PRIMARY_AGENT=droid`

### Authentication Errors

If you get `401 Unauthorized`:
1. Verify `AUTHKIT_JWKS_URL` is correct
2. Check your JWT token is valid and not expired
3. Ensure the token includes required claims: `sub`, `org`

### JWKS Loading Errors

If you see errors loading JWKS keys:
1. Check network connectivity to JWKS URL
2. Verify the URL format is correct
3. Check firewall rules allow outbound HTTPS

## Production Deployment Checklist

- [ ] Set `AUTHKIT_JWKS_URL` environment variable
- [ ] Install agent binaries (CCRouter and/or Droid)
- [ ] Set correct paths for `CCROUTER_PATH` and `DROID_PATH`
- [ ] Configure `PRIMARY_AGENT` based on your preference
- [ ] Enable metrics and audit logging for production
- [ ] Set up proper logging (structured logs, log aggregation)
- [ ] Configure health check monitoring
- [ ] Set appropriate timeouts for your use case
- [ ] Test graceful shutdown handling
- [ ] Set up alerts for agent health failures
- [ ] Document available models for your users

## Backward Compatibility

The chat API integration is designed to be **fully backward compatible**:

1. **Existing routes unchanged**: All existing routes continue to work
2. **Opt-in activation**: Chat API is only enabled when configured
3. **Independent middleware**: Tiered access middleware doesn't affect existing routes
4. **Graceful degradation**: If no agents are available, server still runs but chat endpoints return errors

## Architecture Overview

```
HTTP Request
    |
    v
[TieredAccessMiddleware]  <-- Validates JWT, checks access level
    |
    v
[ChatHandler]             <-- Parses request, validates input
    |
    v
[Orchestrator]            <-- Selects agent, handles fallback
    |
    +---> [CCRouterAgent] <-- Executes via ccrouter CLI
    |
    +---> [DroidAgent]    <-- Executes via droid CLI
    |
    v
Response
```

## Available Endpoints

After integration, these endpoints are available:

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | Public | Health check |
| `/v1/models` | GET | AuthKit | List available models |
| `/v1/chat/completions` | POST | AuthKit | Chat completion (streaming & non-streaming) |

## Next Steps

1. Review `pkg/server/setup.go` for implementation details
2. Check `pkg/server/example_integration.go` for integration patterns
3. Read `lib/chat/handler.go` for request/response formats
4. See `lib/agents/*.go` for agent implementations
5. Review `lib/middleware/authkit.go` for authentication details

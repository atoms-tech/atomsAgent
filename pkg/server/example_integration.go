package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ExampleServerIntegration shows how to integrate the chat API into your server
//
// This example can be adapted for the main.go or cmd/server/server.go file
func ExampleServerIntegration() {
	// 1. Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting AgentAPI server with chat API")

	// 2. Load configuration from environment
	config, err := LoadConfigFromEnv()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Validate configuration
	if err := ValidateConfig(config, logger); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// 4. Create HTTP router
	mux := http.NewServeMux()

	// 5. Setup chat API (this wires everything up)
	components, err := SetupChatAPI(mux, logger, config)
	if err != nil {
		logger.Error("failed to setup chat API", "error", err)
		os.Exit(1)
	}

	// 6. Log startup information
	LogStartupInfo(logger, config, components)

	// 7. Register any additional routes (backward compatibility)
	// Example: existing routes from your codebase
	// mux.Handle("/api/v1/mcp", yourMCPHandler)
	// mux.Handle("/api/v1/oauth", yourOAuthHandler)

	// 8. Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 9. Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// 10. Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", config.Port)
		serverErrors <- srv.ListenAndServe()
	}()

	// 11. Wait for shutdown signal or error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)

		// Give outstanding requests 15 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Shutdown HTTP server
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			if err := srv.Close(); err != nil {
				logger.Error("force close failed", "error", err)
			}
		}

		// Shutdown chat API components
		if err := components.GracefulShutdown(ctx, logger); err != nil {
			logger.Error("component shutdown failed", "error", err)
		}

		logger.Info("server stopped")
	}
}

// Example: How to integrate into existing cmd/server/server.go
//
// BEFORE:
// ```go
// func runServer(ctx context.Context, logger *slog.Logger, argsToPass []string) error {
//     // ... existing code ...
//     srv, err := httpapi.NewServer(ctx, httpapi.ServerConfig{...})
//     // ...
// }
// ```
//
// AFTER:
// ```go
// func runServer(ctx context.Context, logger *slog.Logger, argsToPass []string) error {
//     // Load chat API config
//     chatConfig, err := server.LoadConfigFromEnv()
//     if err != nil {
//         logger.Warn("chat API disabled", "error", err)
//         chatConfig = nil
//     }
//
//     // Existing server setup
//     srv, err := httpapi.NewServer(ctx, httpapi.ServerConfig{...})
//     if err != nil {
//         return xerrors.Errorf("failed to create server: %w", err)
//     }
//
//     // Get the server's HTTP mux
//     mux := srv.GetMux() // Assuming your server exposes the mux
//
//     // Setup chat API if config is valid
//     var chatComponents *server.ChatAPIComponents
//     if chatConfig != nil {
//         if err := server.ValidateConfig(chatConfig, logger); err == nil {
//             chatComponents, err = server.SetupChatAPI(mux, logger, chatConfig)
//             if err != nil {
//                 logger.Error("failed to setup chat API", "error", err)
//             } else {
//                 server.LogStartupInfo(logger, chatConfig, chatComponents)
//             }
//         }
//     }
//
//     // Start server (existing code)
//     logger.Info("Starting server on port", "port", port)
//     // ... rest of existing code ...
//
//     // On shutdown, cleanup chat components
//     if chatComponents != nil {
//         if err := chatComponents.GracefulShutdown(ctx, logger); err != nil {
//             logger.Error("Failed to shutdown chat API", "error", err)
//         }
//     }
//
//     return nil
// }
// ```

// Example environment variables for .env file:
//
// # Required
// AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/YOUR_CLIENT_ID
//
// # Optional - Agent Paths
// CCROUTER_PATH=/usr/local/bin/ccrouter
// DROID_PATH=/usr/local/bin/droid
//
// # Optional - Agent Configuration
// PRIMARY_AGENT=ccrouter  # or "droid"
// FALLBACK_ENABLED=true
//
// # Optional - Observability
// METRICS_ENABLED=true
// AUDIT_ENABLED=true
//
// # Optional - Server
// PORT=3284

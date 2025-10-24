package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/coder/agentapi/pkg/server"
)

func main() {
	// 1. Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting AgentAPI Chat Server")

	// 2. Check for required environment variables
	authKitJWKSURL := os.Getenv("AUTHKIT_JWKS_URL")
	if authKitJWKSURL == "" {
		logger.Error("AUTHKIT_JWKS_URL environment variable is required")
		fmt.Fprintln(os.Stderr, "\nConfiguration error: Missing required environment variables")
		fmt.Fprintln(os.Stderr, "\nRequired environment variables:")
		fmt.Fprintln(os.Stderr, "  AUTHKIT_JWKS_URL - AuthKit JWKS URL for authentication")
		fmt.Fprintln(os.Stderr, "\nOptional environment variables:")
		fmt.Fprintln(os.Stderr, "  DATABASE_URL     - PostgreSQL database URL for platform admin features")
		fmt.Fprintln(os.Stderr, "  CCROUTER_PATH    - Path to CCRouter binary (default: /usr/local/bin/ccrouter)")
		fmt.Fprintln(os.Stderr, "  DROID_PATH       - Path to Droid binary (default: /usr/local/bin/droid)")
		fmt.Fprintln(os.Stderr, "  PRIMARY_AGENT    - Primary agent to use: 'ccrouter' or 'droid' (default: ccrouter)")
		fmt.Fprintln(os.Stderr, "  FALLBACK_ENABLED - Enable fallback to secondary agent (default: true)")
		fmt.Fprintln(os.Stderr, "  PORT             - Server port (default: 3284)")
		fmt.Fprintln(os.Stderr, "  METRICS_ENABLED  - Enable metrics collection (default: true)")
		fmt.Fprintln(os.Stderr, "  AUDIT_ENABLED    - Enable audit logging (default: true)")
		os.Exit(1)
	}

	// 3. Load configuration with defaults
	databaseURL := os.Getenv("DATABASE_URL")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseServiceRoleKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	ccRouterPath := getEnvOrDefault("CCROUTER_PATH", "/usr/local/bin/ccrouter")
	droidPath := getEnvOrDefault("DROID_PATH", "/usr/local/bin/droid")
	primaryAgent := getEnvOrDefault("PRIMARY_AGENT", "ccrouter")
	port := getEnvOrDefault("PORT", "3284")

	logger.Info("configuration loaded",
		"authkit_jwks_url", authKitJWKSURL,
		"database_url", databaseURL != "",
		"supabase_url", supabaseURL != "",
		"ccrouter_path", ccRouterPath,
		"droid_path", droidPath,
		"primary_agent", primaryAgent,
		"port", port,
	)

	// 4. Validate agent paths exist
	ccrAvailable := fileExists(ccRouterPath)
	droidAvailable := fileExists(droidPath)

	if !ccrAvailable && !droidAvailable {
		logger.Error("no agent binaries found",
			"ccrouter_path", ccRouterPath,
			"droid_path", droidPath,
		)
		fmt.Fprintln(os.Stderr, "\nError: At least one agent (CCRouter or Droid) must be available")
		fmt.Fprintf(os.Stderr, "  CCRouter path: %s (exists: %v)\n", ccRouterPath, ccrAvailable)
		fmt.Fprintf(os.Stderr, "  Droid path: %s (exists: %v)\n", droidPath, droidAvailable)
		os.Exit(1)
	}

	logger.Info("agents status",
		"ccrouter_available", ccrAvailable,
		"droid_available", droidAvailable,
	)

	// 5. Parse port
	_, err := strconv.Atoi(port)
	if err != nil {
		logger.Error("invalid port number", "port", port, "error", err)
		os.Exit(1)
	}

	// 6. Parse fallback enabled
	fallbackEnabled := getEnvOrDefault("FALLBACK_ENABLED", "true") == "true"
	metricsEnabled := getEnvOrDefault("METRICS_ENABLED", "true") == "true"
	auditEnabled := getEnvOrDefault("AUDIT_ENABLED", "true") == "true"

	// 7. Create server configuration
	config := &server.Config{
		AuthKitJWKSURL:         authKitJWKSURL,
		DatabaseURL:            databaseURL,
		SupabaseURL:            supabaseURL,
		SupabaseServiceRoleKey: supabaseServiceRoleKey,
		CCRouterPath:           ccRouterPath,
		DroidPath:              droidPath,
		PrimaryAgent:           primaryAgent,
		FallbackEnabled:        fallbackEnabled,
		AgentTimeout:           30 * time.Second,
		MaxTokens:              4000,
		DefaultTemp:            0.7,
		MetricsEnabled:         metricsEnabled,
		AuditEnabled:           auditEnabled,
	}

	// 8. Create HTTP router
	mux := http.NewServeMux()

	// 9. Setup chat API
	logger.Info("setting up chat API")
	components, err := server.SetupChatAPI(mux, logger, config)
	if err != nil {
		logger.Error("failed to setup chat API", "error", err)
		os.Exit(1)
	}

	// 10. Register health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		agents := []string{}
		if ccrAvailable {
			agents = append(agents, "ccrouter")
		}
		if droidAvailable {
			agents = append(agents, "droid")
		}
		fmt.Fprintf(w, `{"status":"healthy","agents":%q,"primary":"%s"}`, agents, primaryAgent)
	})

	// 11. Register status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"running","uptime":"active","configured_agents":{"ccrouter":%v,"droid":%v}}`, ccrAvailable, droidAvailable)
	})

	// 12. Register chat API routes
	server.RegisterChatRoutes(mux, logger, components)

	// 13. Create HTTP server with reasonable timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 10. Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// 11. Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening",
			"port", port,
			"address", fmt.Sprintf("http://localhost:%s", port),
		)
		logger.Info("available endpoints",
			"health", "/health",
			"status", "/status",
			"chat_completions", "/v1/chat/completions",
			"models", "/v1/models",
			"platform_stats", "/api/v1/platform/stats",
			"platform_admins", "/api/v1/platform/admins",
			"platform_audit", "/api/v1/platform/audit",
		)
		serverErrors <- srv.ListenAndServe()
	}()

	// 12. Wait for shutdown signal or error
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
		logger.Info("shutting down HTTP server")
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			if err := srv.Close(); err != nil {
				logger.Error("force close failed", "error", err)
			}
		}

		logger.Info("server stopped successfully")
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

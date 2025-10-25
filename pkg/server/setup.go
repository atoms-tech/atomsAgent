package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/coder/agentapi/api/v1"
	"github.com/coder/agentapi/lib/admin"
	"github.com/coder/agentapi/lib/agents"
	"github.com/coder/agentapi/lib/audit"
	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/chat"
	"github.com/coder/agentapi/lib/metrics"
	"github.com/coder/agentapi/lib/middleware"
	"github.com/supabase-community/supabase-go"
	_ "github.com/lib/pq"
)

// Config holds server configuration
type Config struct {
	// Authentication
	AuthKitJWKSURL string

	// Database
	DatabaseURL string

	// Supabase Configuration
	SupabaseURL           string
	SupabaseServiceRoleKey string

	// Agent paths
	CCRouterPath string
	DroidPath    string

	// Agent configuration
	PrimaryAgent    string // "ccrouter" or "droid"
	FallbackEnabled bool
	AgentTimeout    time.Duration

	// Chat API
	MaxTokens   int
	DefaultTemp float32

	// Observability
	MetricsEnabled bool
	AuditEnabled   bool

	// Server
	Port int
}

// ChatAPIComponents holds all initialized chat API components
type ChatAPIComponents struct {
	Orchestrator           *chat.Orchestrator
	Handler                *chat.ChatHandler
	TieredAccessMiddleware *middleware.TieredAccessMiddleware
	CCRouterAgent          agents.Agent
	DroidAgent             agents.Agent
	AuthKitValidator       *auth.AuthKitValidator
	AuditLogger            *audit.AuditLogger
	MetricsClient          *metrics.MetricsRegistry
	SupabaseClient         *supabase.Client
	DB                     *sql.DB
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{
		// Defaults
		PrimaryAgent:    "ccrouter",
		FallbackEnabled: true,
		AgentTimeout:    5 * time.Minute,
		MaxTokens:       4096,
		DefaultTemp:     0.7,
		Port:            3284,
		MetricsEnabled:  true,
		AuditEnabled:    true,
	}

	// Required: AuthKit JWKS URL
	config.AuthKitJWKSURL = os.Getenv("AUTHKIT_JWKS_URL")
	if config.AuthKitJWKSURL == "" {
		return nil, fmt.Errorf("AUTHKIT_JWKS_URL environment variable is required")
	}

	// Supabase Configuration
	config.SupabaseURL = os.Getenv("SUPABASE_URL")
	config.SupabaseServiceRoleKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	// For backward compatibility, also support DATABASE_URL
	config.DatabaseURL = os.Getenv("DATABASE_URL")

	// Optional: Agent paths
	config.CCRouterPath = os.Getenv("CCROUTER_PATH")
	if config.CCRouterPath == "" {
		config.CCRouterPath = "/usr/local/bin/ccrouter" // Default path
	}

	config.DroidPath = os.Getenv("DROID_PATH")
	if config.DroidPath == "" {
		config.DroidPath = "/usr/local/bin/droid" // Default path
	}

	// Optional: Primary agent
	if primary := os.Getenv("PRIMARY_AGENT"); primary != "" {
		if primary != "ccrouter" && primary != "droid" {
			return nil, fmt.Errorf("PRIMARY_AGENT must be 'ccrouter' or 'droid', got: %s", primary)
		}
		config.PrimaryAgent = primary
	}

	// Optional: Fallback
	if fallback := os.Getenv("FALLBACK_ENABLED"); fallback == "false" {
		config.FallbackEnabled = false
	}

	// Optional: Metrics
	if metricsEnabled := os.Getenv("METRICS_ENABLED"); metricsEnabled == "false" {
		config.MetricsEnabled = false
	}

	// Optional: Audit
	if auditEnabled := os.Getenv("AUDIT_ENABLED"); auditEnabled == "false" {
		config.AuditEnabled = false
	}

	return config, nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config, logger *slog.Logger) error {
	if config.AuthKitJWKSURL == "" {
		return fmt.Errorf("AuthKit JWKS URL is required")
	}

	// Check if at least one agent is available
	ccrAvailable := fileExists(config.CCRouterPath)
	droidAvailable := fileExists(config.DroidPath)

	if !ccrAvailable && !droidAvailable {
		logger.Warn("no agent binaries found",
			"ccrouter_path", config.CCRouterPath,
			"droid_path", config.DroidPath,
		)
		return fmt.Errorf("at least one agent (CCRouter or Droid) must be available")
	}

	// Warn if primary agent is not available
	if config.PrimaryAgent == "ccrouter" && !ccrAvailable {
		logger.Warn("primary agent CCRouter not found, will use Droid",
			"ccrouter_path", config.CCRouterPath,
		)
		config.PrimaryAgent = "droid"
	}

	if config.PrimaryAgent == "droid" && !droidAvailable {
		logger.Warn("primary agent Droid not found, will use CCRouter",
			"droid_path", config.DroidPath,
		)
		config.PrimaryAgent = "ccrouter"
	}

	return nil
}

// SetupChatAPI initializes and wires up all chat API components
func SetupChatAPI(router *http.ServeMux, logger *slog.Logger, config *Config) (*ChatAPIComponents, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	logger.Info("initializing chat API",
		"primary_agent", config.PrimaryAgent,
		"fallback_enabled", config.FallbackEnabled,
	)

	components := &ChatAPIComponents{}

	// 1. Initialize database connection
	var db *sql.DB
	if config.SupabaseURL != "" && config.SupabaseServiceRoleKey != "" {
		logger.Info("initializing Supabase connection",
			"url", config.SupabaseURL,
		)

		// Initialize Supabase client
		var err error
		components.SupabaseClient, err = supabase.NewClient(config.SupabaseURL, config.SupabaseServiceRoleKey, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Supabase client: %w", err)
		}

		// Test connection by querying a simple table
		// Use PostgREST client to test connectivity
		var testResult []map[string]interface{}
		_, err = components.SupabaseClient.From("agents").Select("id", "", false).ExecuteTo(&testResult)
		if err != nil {
			// Log the error but don't block startup - RLS or permissions may need manual fixes
			logger.Warn("Supabase table access test failed (RLS or permissions may need fixing)",
				"error", err,
				"note", "Server will start but database operations may fail until resolved",
			)
		} else {
			logger.Info("Supabase connection established")
		}

		// For components that still need sql.DB, create a connection pool
		// This uses DATABASE_URL as fallback or builds from Supabase credentials
		var dbURL string
		if config.DatabaseURL != "" {
			dbURL = config.DatabaseURL
		} else if config.SupabaseURL != "" {
			// Extract credentials from Supabase URL if available
			logger.Warn("DATABASE_URL not provided, using Supabase connection URL only",
				"note", "legacy sql.DB support may not work optimally",
			)
		}

		if dbURL != "" {
			var err error
			db, err = sql.Open("postgres", dbURL)
			if err != nil {
				logger.Error("failed to create connection pool", "error", err)
				// Don't fail here - Supabase client is primary
			} else {
				ctxPing, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := db.PingContext(ctxPing); err != nil {
					logger.Warn("connection pool ping failed", "error", err)
					// Don't fail - Supabase client is working
				} else {
					logger.Info("connection pool established")
				}
			}
		}
	} else if config.DatabaseURL != "" {
		logger.Info("initializing database connection via DATABASE_URL")
		var err error
		db, err = sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		// Test database connection
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
		logger.Info("database connection established")
	} else {
		logger.Warn("no database connection configured (neither SUPABASE_URL nor DATABASE_URL)")
	}

	// Store DB reference in components for graceful shutdown
	components.DB = db

	// 2. Initialize AuthKit validator
	logger.Info("initializing AuthKit validator", "jwks_url", config.AuthKitJWKSURL)
	components.AuthKitValidator = auth.NewAuthKitValidator(logger, config.AuthKitJWKSURL, db)

	// Validate JWKS URL is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test.test" // Dummy token for key fetch
	_, err := components.AuthKitValidator.ValidateToken(ctx, testToken)
	if err != nil {
		// Expected to fail validation, but keys should be loaded
		logger.Debug("initial JWKS key fetch completed", "error", err.Error())
	}

	// 2. Initialize audit logger (if database is available)
	if config.AuditEnabled && db != nil {
		logger.Info("initializing audit logger")
		components.AuditLogger, err = audit.NewAuditLogger(db, 1000)
		if err != nil {
			// Log but don't fail - audit logging is optional
			logger.Warn("failed to initialize audit logger",
				"error", err,
				"note", "Audit logging will be disabled",
			)
		}
	} else if !config.AuditEnabled {
		logger.Info("audit logging disabled by configuration")
	} else {
		logger.Warn("audit logging disabled - database connection not available")
	}

	// 3. Initialize metrics client
	if config.MetricsEnabled {
		logger.Info("initializing metrics client")
		components.MetricsClient = metrics.NewMetricsRegistry()
	} else {
		logger.Info("metrics disabled")
	}

	// 4. Initialize agents
	var agentsInitialized []string

	// Initialize CCRouter agent if available
	if fileExists(config.CCRouterPath) {
		logger.Info("initializing CCRouter agent", "path", config.CCRouterPath)
		components.CCRouterAgent = agents.NewCCRouterAgent(
			logger,
			config.CCRouterPath,
			config.AgentTimeout,
		)

		if components.CCRouterAgent.IsHealthy(context.Background()) {
			agentsInitialized = append(agentsInitialized, "ccrouter")
			logger.Info("CCRouter agent initialized successfully")
		} else {
			logger.Warn("CCRouter agent health check failed")
			components.CCRouterAgent = nil
		}
	} else {
		logger.Warn("CCRouter binary not found, agent disabled", "path", config.CCRouterPath)
	}

	// Initialize Droid agent if available
	if fileExists(config.DroidPath) {
		logger.Info("initializing Droid agent", "path", config.DroidPath)
		components.DroidAgent = agents.NewDroidAgent(
			logger,
			config.DroidPath,
			config.AgentTimeout,
		)

		if components.DroidAgent.IsHealthy(context.Background()) {
			agentsInitialized = append(agentsInitialized, "droid")
			logger.Info("Droid agent initialized successfully")
		} else {
			logger.Warn("Droid agent health check failed")
			components.DroidAgent = nil
		}
	} else {
		logger.Warn("Droid binary not found, agent disabled", "path", config.DroidPath)
	}

	// Ensure at least one agent is available
	if components.CCRouterAgent == nil && components.DroidAgent == nil {
		return nil, fmt.Errorf("no healthy agents available")
	}

	logger.Info("agents initialized", "count", len(agentsInitialized), "agents", agentsInitialized)

	// 5. Create orchestrator
	logger.Info("creating chat orchestrator")
	components.Orchestrator, err = chat.NewOrchestrator(
		logger,
		components.CCRouterAgent,
		components.DroidAgent,
		config.PrimaryAgent,
		config.FallbackEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// 6. Initialize platform admin service
	var adminService *admin.PlatformAdminService
	if db != nil {
		logger.Info("initializing platform admin service")
		adminService = admin.NewPlatformAdminService(db, logger)
	} else {
		logger.Info("platform admin service disabled (no database)")
	}

	// 7. Create chat handler
	logger.Info("creating chat handler")
	components.Handler = chat.NewChatHandler(
		logger,
		components.Orchestrator,
		components.AuditLogger,
		components.MetricsClient,
		config.MaxTokens,
		config.DefaultTemp,
		adminService,
	)

	// 7. Initialize tiered access middleware
	logger.Info("initializing tiered access middleware")
	components.TieredAccessMiddleware = middleware.NewTieredAccessMiddleware(
		logger,
		components.AuthKitValidator,
		components.AuditLogger,
	)

	// 8. Register chat routes with middleware
	logger.Info("registering chat API routes")

	// Chat completion endpoint - requires authentication
	router.Handle("/v1/chat/completions",
		components.TieredAccessMiddleware.Handler(
			http.HandlerFunc(components.Handler.HandleChatCompletion),
		),
	)

	// Models list endpoint - requires authentication
	router.Handle("/v1/models",
		components.TieredAccessMiddleware.Handler(
			http.HandlerFunc(components.Handler.HandleListModels),
		),
	)

	// Platform admin endpoints are registered in api/v1/chat.go

	router.Handle("/api/v1/platform/audit",
		components.TieredAccessMiddleware.Handler(
			http.HandlerFunc(components.Handler.HandleGetAuditLog),
		),
	)

	// Health check endpoint is registered in main.go

	logger.Info("chat API setup complete",
		"endpoints", []string{
			"/v1/chat/completions",
			"/v1/models",
			"/health",
		},
		"available_agents", agentsInitialized,
	)

	return components, nil
}

// RegisterChatRoutes registers all chat API routes
func RegisterChatRoutes(router *http.ServeMux, logger *slog.Logger, components *ChatAPIComponents) {
	// Register chat API routes
	v1.RegisterChatAPI(router, logger, components.Handler, components.TieredAccessMiddleware)
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GracefulShutdown handles graceful shutdown of agents
func (c *ChatAPIComponents) GracefulShutdown(ctx context.Context, logger *slog.Logger) error {
	logger.Info("shutting down chat API components")

	// Close database connection
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			logger.Error("failed to close database connection", "error", err)
		}
	}

	// Close audit logger
	if c.AuditLogger != nil {
		if err := c.AuditLogger.Close(); err != nil {
			logger.Error("failed to close audit logger", "error", err)
		}
	}

	// Close metrics client
	if c.MetricsClient != nil {
		// c.MetricsClient.Close() // TODO: Add Close method to MetricsRegistry
	}

	// Supabase client doesn't require explicit close
	// (HTTP client cleanup is handled internally)

	logger.Info("chat API shutdown complete")
	return nil
}

// LogStartupInfo logs detailed information about the setup
func LogStartupInfo(logger *slog.Logger, config *Config, components *ChatAPIComponents) {
	logger.Info("=== Chat API Configuration ===")
	logger.Info("authentication", "jwks_url", config.AuthKitJWKSURL)
	logger.Info("primary agent", "agent", config.PrimaryAgent)
	logger.Info("fallback", "enabled", config.FallbackEnabled)
	logger.Info("timeouts", "agent_timeout", config.AgentTimeout)
	logger.Info("chat defaults", "max_tokens", config.MaxTokens, "temperature", config.DefaultTemp)

	logger.Info("=== Available Agents ===")
	if components.CCRouterAgent != nil {
		models := components.CCRouterAgent.GetAvailableModels(context.Background())
		logger.Info("CCRouter",
			"status", "available",
			"models", len(models),
			"path", config.CCRouterPath,
		)
	} else {
		logger.Info("CCRouter", "status", "unavailable")
	}

	if components.DroidAgent != nil {
		models := components.DroidAgent.GetAvailableModels(context.Background())
		logger.Info("Droid",
			"status", "available",
			"models", len(models),
			"path", config.DroidPath,
		)
	} else {
		logger.Info("Droid", "status", "unavailable")
	}

	logger.Info("=== Middleware ===")
	logger.Info("tiered access", "enabled", true)
	logger.Info("audit logging", "enabled", config.AuditEnabled)
	logger.Info("metrics", "enabled", config.MetricsEnabled)

	logger.Info("=== Chat API Ready ===")
}

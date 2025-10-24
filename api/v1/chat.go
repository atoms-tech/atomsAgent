package v1

import (
	"log/slog"
	"net/http"

	"github.com/coder/agentapi/lib/admin"
	"github.com/coder/agentapi/lib/agents"
	"github.com/coder/agentapi/lib/audit"
	"github.com/coder/agentapi/lib/chat"
	"github.com/coder/agentapi/lib/metrics"
	"github.com/coder/agentapi/lib/middleware"
)

// ChatRouter registers chat-related API routes
func ChatRouter(
	router *http.ServeMux,
	logger *slog.Logger,
	chatHandler *chat.ChatHandler,
	authKitMiddleware *middleware.TieredAccessMiddleware,
) {
	// POST /v1/chat/completions - Chat completions endpoint
	router.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleChatCompletion(w, r)
		})).ServeHTTP(w, r)
	})

	// GET /v1/models - List available models
	router.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleListModels(w, r)
		})).ServeHTTP(w, r)
	})

	// Platform Admin Routes
	// GET /api/v1/platform/stats - Platform statistics
	router.HandleFunc("GET /api/v1/platform/stats", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandlePlatformStats(w, r)
		})).ServeHTTP(w, r)
	})

	// GET /api/v1/platform/admins - List platform admins
	router.HandleFunc("GET /api/v1/platform/admins", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleListPlatformAdmins(w, r)
		})).ServeHTTP(w, r)
	})

	// POST /api/v1/platform/admins - Add platform admin
	router.HandleFunc("POST /api/v1/platform/admins", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleAddPlatformAdmin(w, r)
		})).ServeHTTP(w, r)
	})

	// DELETE /api/v1/platform/admins/{email} - Remove platform admin
	router.HandleFunc("DELETE /api/v1/platform/admins/", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleRemovePlatformAdmin(w, r)
		})).ServeHTTP(w, r)
	})

	// GET /api/v1/platform/audit - Get audit log
	router.HandleFunc("GET /api/v1/platform/audit", func(w http.ResponseWriter, r *http.Request) {
		authKitMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chatHandler.HandleGetAuditLog(w, r)
		})).ServeHTTP(w, r)
	})
}

// SetupChatAPI initializes the chat API layer
func SetupChatAPI(
	logger *slog.Logger,
	ccRouterAgent agents.Agent,
	droidAgent agents.Agent,
	auditLogger *audit.AuditLogger,
	metricsClient *metrics.MetricsRegistry,
	adminService *admin.PlatformAdminService,
) (*chat.ChatHandler, error) {
	// Create orchestrator
	orchestrator, err := chat.NewOrchestrator(
		logger,
		ccRouterAgent,
		droidAgent,
		"ccrouter", // Primary agent
		true,       // Enable fallback
	)
	if err != nil {
		return nil, err
	}

	// Create chat handler
	chatHandler := chat.NewChatHandler(
		logger,
		orchestrator,
		auditLogger,
		metricsClient,
		4000, // max tokens
		0.7,  // default temperature
		adminService,
	)

	return chatHandler, nil
}

// RegisterChatAPI registers all chat API routes
func RegisterChatAPI(router *http.ServeMux, logger *slog.Logger, chatHandler *chat.ChatHandler, authKitMiddleware *middleware.TieredAccessMiddleware) {
	ChatRouter(router, logger, chatHandler, authKitMiddleware)
}

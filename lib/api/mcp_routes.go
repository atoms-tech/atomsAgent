package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterMCPRoutes registers all MCP-related routes
func RegisterMCPRoutes(router chi.Router, handler *MCPHandler) {
	// MCP configuration management routes
	router.Route("/api/v1/mcp", func(r chi.Router) {
		// Apply authentication middleware
		r.Use(AuthMiddleware)
		r.Use(TenantIsolationMiddleware)

		// Configuration CRUD endpoints
		r.Post("/configurations", handler.CreateMCPConfiguration)
		r.Get("/configurations", handler.ListMCPConfigurations)
		r.Get("/configurations/{id}", handler.GetMCPConfiguration)
		r.Put("/configurations/{id}", handler.UpdateMCPConfiguration)
		r.Delete("/configurations/{id}", handler.DeleteMCPConfiguration)

		// Test connection endpoint
		r.Post("/test", handler.TestMCPConnection)
	})
}

// AuthMiddleware validates authentication tokens
// This is a placeholder - implement with your actual auth mechanism (JWT, session, etc.)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement actual authentication
		// For now, this is a placeholder that sets context values

		// Example:
		// token := r.Header.Get("Authorization")
		// claims, err := validateJWT(token)
		// if err != nil {
		//     http.Error(w, "Unauthorized", http.StatusUnauthorized)
		//     return
		// }
		//
		// ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		// ctx = context.WithValue(ctx, "orgID", claims.OrgID)
		// r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// TenantIsolationMiddleware ensures proper tenant isolation
func TenantIsolationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant information from context
		// Validate that the user has access to the requested organization

		// This middleware should verify:
		// 1. User belongs to the organization they're claiming
		// 2. User has appropriate permissions for the operation
		// 3. Cross-tenant access is prevented

		next.ServeHTTP(w, r)
	})
}

// Example usage in main application setup:
//
// func SetupServer(db *sql.DB) (*chi.Mux, error) {
//     router := chi.NewRouter()
//
//     // Create dependencies
//     fastmcpClient, err := mcp.NewFastMCPClient()
//     if err != nil {
//         return nil, err
//     }
//
//     sessionMgr := session.NewSessionManager("/var/lib/agentapi")
//     auditLogger := NewAuditLogger()
//     encryptionKey := os.Getenv("MCP_ENCRYPTION_KEY") // Store securely!
//
//     // Create MCP handler
//     mcpHandler, err := NewMCPHandler(db, fastmcpClient, sessionMgr, auditLogger, encryptionKey)
//     if err != nil {
//         return nil, err
//     }
//
//     // Register routes
//     RegisterMCPRoutes(router, mcpHandler)
//
//     return router, nil
// }

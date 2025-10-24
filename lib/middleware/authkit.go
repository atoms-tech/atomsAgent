package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/agentapi/lib/audit"
	"github.com/coder/agentapi/lib/auth"
)

// AccessLevel defines access control levels
type AccessLevel string

const (
	Public        AccessLevel = "public"         // No authentication required
	Authenticated AccessLevel = "authenticated"  // AuthKit token required
	OrgAdmin      AccessLevel = "org_admin"      // Organization admin required
	PlatformAdmin AccessLevel = "platform_admin" // Platform admin required
	AdminOnly     AccessLevel = "admin"          // Admin role required (backward compatibility)
)

// AuthKitMiddleware enforces AuthKit authentication with tiered access control
type AuthKitMiddleware struct {
	logger      *slog.Logger
	validator   *auth.AuthKitValidator
	auditLogger *audit.AuditLogger
	accessLevel AccessLevel
}

// NewAuthKitMiddleware creates a new AuthKit middleware
func NewAuthKitMiddleware(
	logger *slog.Logger,
	validator *auth.AuthKitValidator,
	auditLogger *audit.AuditLogger,
	accessLevel AccessLevel,
) *AuthKitMiddleware {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	return &AuthKitMiddleware{
		logger:      logger,
		validator:   validator,
		auditLogger: auditLogger,
		accessLevel: accessLevel,
	}
}

// Handler wraps an HTTP handler with authentication middleware
func (am *AuthKitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public endpoints don't require authentication
		if am.accessLevel == Public {
			next.ServeHTTP(w, r)
			return
		}

		// Extract and validate token
		authHeader := r.Header.Get("Authorization")
		tokenString, err := am.validator.ExtractBearerToken(authHeader)
		if err != nil {
			am.logger.Warn("missing or invalid authorization header",
				"path", r.URL.Path,
				"error", err.Error(),
			)
			http.Error(w, "unauthorized: invalid authorization header", http.StatusUnauthorized)
			return
		}

		// Validate token
		user, err := am.validator.ValidateToken(r.Context(), tokenString)
		if err != nil {
			am.logger.Warn("token validation failed",
				"path", r.URL.Path,
				"error", err.Error(),
			)
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Check access level
		if !am.checkAccess(user, am.accessLevel) {
			am.logger.Warn("user lacks required access level",
				"user_id", user.ID,
				"path", r.URL.Path,
				"role", user.Role,
				"required_level", am.accessLevel,
				"is_org_admin", user.IsOrgAdmin(),
				"is_platform_admin", user.IsPlatformAdmin(),
			)
			http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), "authkit_user", user)
		ctx = context.WithValue(ctx, "user_id", user.ID)
		ctx = context.WithValue(ctx, "org_id", user.OrgID)

		// Log authentication
		am.logger.Debug("user authenticated",
			"user_id", user.ID,
			"org_id", user.OrgID,
			"path", r.URL.Path,
			"method", r.Method,
		)

		// Call next handler with authenticated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkAccess verifies if user has the required access level
func (am *AuthKitMiddleware) checkAccess(user *auth.AuthKitUser, accessLevel AccessLevel) bool {
	if user == nil {
		return accessLevel == Public
	}

	switch accessLevel {
	case Public:
		return true
	case Authenticated:
		return true
	case OrgAdmin:
		return user.IsOrgAdmin() || user.IsPlatformAdmin()
	case PlatformAdmin:
		return user.IsPlatformAdmin()
	case AdminOnly: // Backward compatibility
		return user.IsOrgAdmin() || user.IsPlatformAdmin()
	default:
		return false
	}
}

// TieredAccessMiddleware provides different access control for different endpoints
type TieredAccessMiddleware struct {
	logger      *slog.Logger
	validator   *auth.AuthKitValidator
	auditLogger *audit.AuditLogger
	routes      map[string]AccessLevel // path -> access level
}

// NewTieredAccessMiddleware creates a new tiered access middleware
func NewTieredAccessMiddleware(
	logger *slog.Logger,
	validator *auth.AuthKitValidator,
	auditLogger *audit.AuditLogger,
) *TieredAccessMiddleware {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	return &TieredAccessMiddleware{
		logger:      logger,
		validator:   validator,
		auditLogger: auditLogger,
		routes: map[string]AccessLevel{
			// Public endpoints
			"/health":  Public,
			"/ready":   Public,
			"/live":    Public,
			"/version": Public,

			// Authenticated endpoints (require token)
			"/v1/chat/completions": Authenticated,
			"/v1/models":           Authenticated,

			// Organization admin endpoints
			"/api/v1/org/settings": OrgAdmin,
			"/api/v1/org/members":  OrgAdmin,
			"/api/v1/org/billing":  OrgAdmin,

			// Platform admin endpoints
			"/api/v1/platform/admins": PlatformAdmin,
			"/api/v1/platform/stats":  PlatformAdmin,
			"/api/v1/platform/orgs":   PlatformAdmin,
			"/api/v1/platform/audit":  PlatformAdmin,

			// Legacy admin endpoints (backward compatibility)
			"/api/v1/mcp":       AdminOnly,
			"/api/v1/mcp/oauth": Authenticated, // OAuth is authenticated but not admin
		},
	}
}

// RegisterRoute registers access level for a route
func (tam *TieredAccessMiddleware) RegisterRoute(path string, accessLevel AccessLevel) {
	tam.routes[path] = accessLevel
}

// Handler wraps an HTTP handler with tiered access control
func (tam *TieredAccessMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Determine access level for this path
		accessLevel := tam.getAccessLevel(r.URL.Path)

		// Apply middleware based on access level
		middleware := NewAuthKitMiddleware(tam.logger, tam.validator, tam.auditLogger, accessLevel)
		middleware.Handler(next).ServeHTTP(w, r)
	})
}

// getAccessLevel determines the required access level for a route
func (tam *TieredAccessMiddleware) getAccessLevel(path string) AccessLevel {
	// Check exact match first
	if level, ok := tam.routes[path]; ok {
		return level
	}

	// Check prefix match
	for routePath, level := range tam.routes {
		if strings.HasPrefix(path, routePath) {
			return level
		}
	}

	// Default to authenticated
	return Authenticated
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// WriteUnauthorized writes a 401 response
func WriteUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"error": "unauthorized", "message": "%s", "code": 401}`, message)
}

// WriteForbidden writes a 403 response
func WriteForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{"error": "forbidden", "message": "%s", "code": 403}`, message)
}

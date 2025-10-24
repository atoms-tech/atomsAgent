package auth

// This file provides example integration code for using the JWT authentication middleware
// in your AgentAPI server. This is for reference only and not meant to be executed directly.

/*
Example 1: Basic Server Setup with Authentication

package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Create authentication middleware
	authMiddleware, err := auth.NewAuthMiddleware(auth.AuthConfig{
		JWKSUrl:  "", // Uses SUPABASE_URL from environment
		Issuer:   os.Getenv("SUPABASE_URL") + "/auth/v1",
		Audience: "authenticated",
		Logger:   logger,
		SkipPaths: []string{
			"/health",
			"/public",
			"/docs",
			"/chat", // Public chat interface
		},
		RefreshPeriod: 5 * time.Minute,
	})
	if err != nil {
		log.Fatal("Failed to create auth middleware:", err)
	}

	// Create router
	router := chi.NewRouter()

	// Apply middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(authMiddleware.Middleware)

	// Public endpoints (no auth required due to SkipPaths)
	router.Get("/health", healthHandler)
	router.Get("/public/info", publicInfoHandler)

	// Protected endpoints (auth required)
	router.Get("/api/sessions", listSessionsHandler)
	router.Post("/api/sessions", createSessionHandler)
	router.Delete("/api/sessions/{id}", deleteSessionHandler)

	// Admin-only endpoints
	router.Group(func(r chi.Router) {
		r.Use(auth.RequireAdminRole(logger))
		r.Get("/api/admin/users", listAllUsersHandler)
		r.Get("/api/admin/stats", adminStatsHandler)
	})

	// Start server
	logger.Info("Starting server on :3284")
	if err := http.ListenAndServe(":3284", router); err != nil {
		log.Fatal("Server error:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func publicInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"version":"1.0.0","status":"online"}`))
}

func listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user context
	userID, orgID, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get full claims for additional info
	claims, _ := auth.GetClaimsFromContext(r.Context())

	// Your business logic here
	response := map[string]interface{}{
		"user_id": userID,
		"org_id":  orgID,
		"email":   claims.Email,
		"sessions": []string{}, // Fetch from database
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createSessionHandler(w http.ResponseWriter, r *http.Request) {
	userID, orgID, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Create session logic here
	slog.Info("Creating session", "user_id", userID, "org_id", orgID)

	w.WriteHeader(http.StatusCreated)
}

func deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID, _, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify user owns the session before deleting
	// Delete session logic here
	slog.Info("Deleting session", "session_id", sessionID, "user_id", userID)

	w.WriteHeader(http.StatusNoContent)
}

func listAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Only admins can reach this handler due to RequireAdminRole middleware
	claims, _ := auth.GetClaimsFromContext(r.Context())

	slog.Info("Admin accessing user list", "admin_id", claims.Sub)

	// Fetch all users from database
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"users":[]}`))
}

func adminStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Admin-only statistics endpoint
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"total_users":0,"total_sessions":0}`))
}

// Example 2: Integrating with Existing HTTP API Server

package httpapi

import (
	"context"
	"log/slog"

	"github.com/coder/agentapi/lib/auth"
	"github.com/go-chi/chi/v5"
	"golang.org/x/xerrors"
)

func NewServer(ctx context.Context, config ServerConfig) (*Server, error) {
	router := chi.NewMux()
	logger := logctx.From(ctx)

	// ... existing middleware setup ...

	// Add authentication middleware
	authMiddleware, err := auth.NewAuthMiddleware(auth.AuthConfig{
		Logger: logger,
		SkipPaths: []string{
			"/health",
			"/chat",    // Public chat interface
			"/docs",    // API documentation
			"/status",  // Status endpoint
		},
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to create auth middleware: %w", err)
	}

	// Apply auth middleware before API routes
	router.Use(authMiddleware.Middleware)

	// ... rest of server setup ...

	s := &Server{
		router: router,
		// ... other fields ...
	}

	return s, nil
}

// Example 3: Custom Handler with Role-Based Logic

func hybridHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.GetClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Different logic based on role
	if auth.IsAdmin(r.Context()) {
		// Admin sees all sessions
		handleAdminView(w, r, claims)
	} else {
		// Users see only their sessions
		handleUserView(w, r, claims)
	}
}

func handleAdminView(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	// Fetch all sessions from all organizations
	allSessions := fetchAllSessions() // Your database query

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"admin":    true,
		"sessions": allSessions,
		"viewer":   claims.Email,
	})
}

func handleUserView(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	// Fetch only user's sessions
	userSessions := fetchUserSessions(claims.Sub) // Your database query

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"admin":    false,
		"sessions": userSessions,
		"user_id":  claims.Sub,
	})
}

// Example 4: Testing with Mock JWT

package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/agentapi/lib/auth"
)

func TestProtectedEndpoint(t *testing.T) {
	// Create test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, orgID, err := auth.GetUserFromContext(r.Context())
		if err != nil {
			http.Error(w, "No auth", http.StatusUnauthorized)
			return
		}

		w.Write([]byte(userID + ":" + orgID))
	})

	// Apply auth middleware
	authMiddleware, _ := auth.NewAuthMiddleware(auth.AuthConfig{
		JWKSUrl: "https://test.supabase.co/auth/v1/jwks",
	})

	protectedHandler := authMiddleware.Middleware(handler)

	// Test without token
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}

	// Test with valid token
	// (In real tests, you'd generate a valid JWT token)
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid-jwt-token")
	rec = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)

	// Assertions...
}

// Example 5: Audit Logging Integration

package api

import (
	"context"
	"log/slog"

	"github.com/coder/agentapi/lib/auth"
)

type AuditLogger struct {
	logger *slog.Logger
}

func (al *AuditLogger) Log(ctx context.Context, action, resourceType, resourceID string, details map[string]any) {
	// Extract user info from context
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		al.logger.Warn("Audit log without auth context", "action", action)
		return
	}

	// Log with user context
	al.logger.Info("Audit event",
		"user_id", claims.Sub,
		"org_id", claims.OrgID,
		"email", claims.Email,
		"role", claims.Role,
		"action", action,
		"resource_type", resourceType,
		"resource_id", resourceID,
		"details", details,
	)

	// Also save to database for compliance
	saveAuditLogToDB(ctx, claims, action, resourceType, resourceID, details)
}

func saveAuditLogToDB(ctx context.Context, claims *auth.Claims, action, resourceType, resourceID string, details map[string]any) {
	// INSERT INTO audit_logs (user_id, org_id, action, resource_type, resource_id, details)
	// VALUES ($1, $2, $3, $4, $5, $6)
}

// Example 6: Multi-tenant Session Isolation

package session

import (
	"context"
	"errors"

	"github.com/coder/agentapi/lib/auth"
)

type SessionManager struct {
	sessions map[string]*Session
}

func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Get user context
	userID, orgID, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, errors.New("authentication required")
	}

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}

	// Verify user owns this session (multi-tenant isolation)
	if session.UserID != userID {
		// Admins can access any session in their org
		if auth.IsAdmin(ctx) && session.OrgID == orgID {
			return session, nil
		}
		return nil, errors.New("access denied")
	}

	return session, nil
}

func (sm *SessionManager) CreateSession(ctx context.Context, config *SessionConfig) (*Session, error) {
	// Automatically associate session with authenticated user
	userID, orgID, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, errors.New("authentication required")
	}

	session := &Session{
		ID:     generateID(),
		UserID: userID,
		OrgID:  orgID,
		Config: config,
	}

	sm.sessions[session.ID] = session
	return session, nil
}

// Example 7: Frontend Integration

// JavaScript/TypeScript example for calling authenticated APIs

async function createSession() {
	// Get JWT token from Supabase
	const { data: { session } } = await supabase.auth.getSession();

	if (!session) {
		console.error('Not authenticated');
		return;
	}

	// Call API with Bearer token
	const response = await fetch('http://localhost:3284/api/sessions', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'Authorization': `Bearer ${session.access_token}`,
		},
		body: JSON.stringify({
			agentType: 'claude',
			initialPrompt: 'Hello',
		}),
	});

	if (response.status === 401) {
		console.error('Unauthorized - token may be expired');
		// Refresh token or redirect to login
		return;
	}

	if (response.status === 403) {
		console.error('Forbidden - insufficient permissions');
		return;
	}

	const data = await response.json();
	console.log('Session created:', data);
}

// React Hook example
function useAuthenticatedFetch() {
	const { session } = useSession(); // Your Supabase session hook

	const authenticatedFetch = async (url, options = {}) => {
		if (!session?.access_token) {
			throw new Error('Not authenticated');
		}

		const response = await fetch(url, {
			...options,
			headers: {
				...options.headers,
				'Authorization': `Bearer ${session.access_token}`,
			},
		});

		if (response.status === 401) {
			// Token expired, trigger refresh
			await refreshSession();
			// Retry request
			return authenticatedFetch(url, options);
		}

		return response;
	};

	return authenticatedFetch;
}

*/

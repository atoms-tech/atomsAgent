package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/agentapi/lib/msgfmt"
	"github.com/coder/agentapi/lib/session"
	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/prompt"
)

// MultiTenantAPI handles multi-tenant operations
type MultiTenantAPI struct {
	SessionManager *session.SessionManager
	MCPManager     *MCPManager
	PromptComposer *prompt.Composer
	AuditLogger    *AuditLogger
}

// NewMultiTenantAPI creates a new multi-tenant API
func NewMultiTenantAPI(sessionManager *session.SessionManager) *MultiTenantAPI {
	return &MultiTenantAPI{
		SessionManager: sessionManager,
		MCPManager:     NewMCPManager(),
		PromptComposer: &prompt.Composer{Validator: prompt.NewValidator()},
		AuditLogger:    NewAuditLogger(),
	}
}

// CreateSessionRequest represents a request to create a new session
type CreateSessionRequest struct {
	AgentType      msgfmt.AgentType `json:"agentType"`
	TermWidth      uint16           `json:"termWidth,omitempty"`
	TermHeight     uint16           `json:"termHeight,omitempty"`
	InitialPrompt  string           `json:"initialPrompt,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	Credentials    map[string]string `json:"credentials,omitempty"`
	MCPConfigs     []mcp.MCPConfig  `json:"mcpConfigs,omitempty"`
	SystemPromptID string           `json:"systemPromptId,omitempty"`
}

// CreateSessionResponse represents the response for session creation
type CreateSessionResponse struct {
	SessionID string `json:"sessionId"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
}

// CreateSession creates a new user session
func (api *MultiTenantAPI) CreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)
	
	if userID == "" || orgID == "" {
		http.Error(w, "Missing user or organization context", http.StatusUnauthorized)
		return
	}
	
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Set defaults
	if req.TermWidth == 0 {
		req.TermWidth = 80
	}
	if req.TermHeight == 0 {
		req.TermHeight = 1000
	}
	
	// Create session config
	config := &session.SessionConfig{
		AgentType:      req.AgentType,
		TermWidth:      req.TermWidth,
		TermHeight:     req.TermHeight,
		InitialPrompt:  req.InitialPrompt,
		Environment:    req.Environment,
		Credentials:    req.Credentials,
	}
	
	// Create session
	userSession, err := api.SessionManager.CreateSession(ctx, userID, orgID, req.AgentType, config)
	if err != nil {
		api.AuditLogger.Log(ctx, userID, orgID, "session_create_failed", "session", "", map[string]any{"error": err.Error()})
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Set MCPs if provided
	if len(req.MCPConfigs) > 0 {
		userSession.SetMCPs(req.MCPConfigs)
	}
	
	// Set system prompt if provided
	if req.SystemPromptID != "" {
		// TODO: Load system prompt from database and set it
		// For now, we'll use a placeholder
		userSession.SetSystemPrompt("System prompt placeholder")
	}
	
	// Log successful creation
	api.AuditLogger.Log(ctx, userID, orgID, "session_created", "session", userSession.ID, map[string]any{
		"agentType": req.AgentType,
		"workspace": userSession.Workspace,
	})
	
	// Return response
	response := CreateSessionResponse{
		SessionID: userSession.ID,
		Workspace: userSession.Workspace,
		Status:    string(userSession.Status),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSession retrieves session details
func (api *MultiTenantAPI) GetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	sessionID := getSessionIDFromPath(r.URL.Path)
	
	if userID == "" {
		http.Error(w, "Missing user context", http.StatusUnauthorized)
		return
	}
	
	session, exists := api.SessionManager.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	
	// Check if user owns this session
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	// Return session details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// TerminateSession terminates a session
func (api *MultiTenantAPI) TerminateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	sessionID := getSessionIDFromPath(r.URL.Path)
	
	if userID == "" {
		http.Error(w, "Missing user context", http.StatusUnauthorized)
		return
	}
	
	session, exists := api.SessionManager.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	
	// Check if user owns this session
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	// Terminate session
	if err := api.SessionManager.TerminateSession(sessionID); err != nil {
		api.AuditLogger.Log(ctx, userID, session.OrgID, "session_terminate_failed", "session", sessionID, map[string]any{"error": err.Error()})
		http.Error(w, fmt.Sprintf("Failed to terminate session: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Log successful termination
	api.AuditLogger.Log(ctx, userID, session.OrgID, "session_terminated", "session", sessionID, nil)
	
	w.WriteHeader(http.StatusNoContent)
}

// ListUserSessions returns all sessions for a user
func (api *MultiTenantAPI) ListUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	
	if userID == "" {
		http.Error(w, "Missing user context", http.StatusUnauthorized)
		return
	}
	
	sessions := api.SessionManager.ListUserSessions(userID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// MCPManager handles MCP configurations
type MCPManager struct {
	clients map[string]*mcp.Client
}

// NewMCPManager creates a new MCP manager
func NewMCPManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]*mcp.Client),
	}
}

// AuditLogger handles audit logging
type AuditLogger struct {
	// TODO: Implement actual logging to database
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// Log logs an audit event
func (al *AuditLogger) Log(ctx context.Context, userID, orgID, action, resourceType, resourceID string, details map[string]any) {
	// TODO: Implement actual logging
	fmt.Printf("AUDIT: user=%s org=%s action=%s resource=%s:%s details=%+v\n", 
		userID, orgID, action, resourceType, resourceID, details)
}

// Helper functions for extracting context from requests
func getUserIDFromContext(ctx context.Context) string {
	// TODO: Extract from JWT token or session
	return "user-123" // Placeholder
}

func getOrgIDFromContext(ctx context.Context) string {
	// TODO: Extract from JWT token or session
	return "org-456" // Placeholder
}

func getSessionIDFromPath(path string) string {
	// Extract session ID from URL path like /api/v1/sessions/{id}
	parts := strings.Split(path, "/")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}
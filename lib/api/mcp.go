package api

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/resilience"
	"github.com/coder/agentapi/lib/session"
)

// MCPHandler handles MCP configuration endpoints
type MCPHandler struct {
	db            *sql.DB
	fastmcpClient *mcp.FastMCPClient
	sessionMgr    *session.SessionManager
	auditLogger   *AuditLogger
	encryptionKey []byte // 32 bytes for AES-256

	// Circuit breakers for different MCP operations
	breakers *mcpCircuitBreakers
}

// mcpCircuitBreakers holds circuit breakers for each MCP operation type
type mcpCircuitBreakers struct {
	connect        *resilience.CircuitBreaker
	callTool       *resilience.CircuitBreaker
	listTools      *resilience.CircuitBreaker
	disconnect     *resilience.CircuitBreaker
	testConnection *resilience.CircuitBreaker
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(db *sql.DB, fastmcpClient *mcp.FastMCPClient, sessionMgr *session.SessionManager, auditLogger *AuditLogger, encryptionKey string) (*MCPHandler, error) {
	// Ensure encryption key is 32 bytes for AES-256
	key := []byte(encryptionKey)
	if len(key) < 32 {
		// Pad the key if too short
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		// Truncate if too long
		key = key[:32]
	}

	// Initialize circuit breakers for MCP operations
	breakers, err := initCircuitBreakers()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize circuit breakers: %w", err)
	}

	return &MCPHandler{
		db:            db,
		fastmcpClient: fastmcpClient,
		sessionMgr:    sessionMgr,
		auditLogger:   auditLogger,
		encryptionKey: key,
		breakers:      breakers,
	}, nil
}

// initCircuitBreakers initializes all circuit breakers for MCP operations
func initCircuitBreakers() (*mcpCircuitBreakers, error) {
	// Circuit breaker configuration
	config := resilience.CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 100,
		OnStateChange:         onCircuitBreakerStateChange,
	}

	// Create circuit breakers for each operation type
	connect, err := resilience.NewCircuitBreaker("mcp_connect", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connect breaker: %w", err)
	}

	callTool, err := resilience.NewCircuitBreaker("mcp_call_tool", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create call_tool breaker: %w", err)
	}

	listTools, err := resilience.NewCircuitBreaker("mcp_list_tools", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create list_tools breaker: %w", err)
	}

	disconnect, err := resilience.NewCircuitBreaker("mcp_disconnect", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create disconnect breaker: %w", err)
	}

	testConnection, err := resilience.NewCircuitBreaker("mcp_test_connection", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create test_connection breaker: %w", err)
	}

	return &mcpCircuitBreakers{
		connect:        connect,
		callTool:       callTool,
		listTools:      listTools,
		disconnect:     disconnect,
		testConnection: testConnection,
	}, nil
}

// onCircuitBreakerStateChange is called when a circuit breaker changes state
func onCircuitBreakerStateChange(name string, from resilience.State, to resilience.State) {
	log.Printf("[MCP Circuit Breaker] %s: State changed from %s to %s", name, from.String(), to.String())

	// TODO: Export to metrics system (Prometheus)
	// metrics.RecordCircuitBreakerStateChange(name, to.String())
}

// Request/Response types

// CreateMCPRequest represents a request to create a new MCP configuration
type CreateMCPRequest struct {
	Name        string         `json:"name" validate:"required,min=1,max=255"`
	Type        string         `json:"type" validate:"required,oneof=http sse stdio"`
	Endpoint    string         `json:"endpoint,omitempty"` // URL for http/sse, command for stdio
	Command     string         `json:"command,omitempty"`  // Command to execute for stdio type
	Args        []string       `json:"args,omitempty"`     // Command arguments for stdio type
	AuthType    string         `json:"auth_type" validate:"required,oneof=none bearer oauth api_key"`
	AuthToken   string         `json:"auth_token,omitempty"`  // Bearer token or API key
	AuthHeader  string         `json:"auth_header,omitempty"` // Custom header name for auth
	Config      map[string]any `json:"config,omitempty"`      // Additional configuration
	Scope       string         `json:"scope" validate:"required,oneof=org user"`
	Enabled     bool           `json:"enabled"`
	Description string         `json:"description,omitempty" validate:"max=1000"`
}

// UpdateMCPRequest represents a request to update an MCP configuration
type UpdateMCPRequest struct {
	Name        *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Type        *string         `json:"type,omitempty" validate:"omitempty,oneof=http sse stdio"`
	Endpoint    *string         `json:"endpoint,omitempty"`
	Command     *string         `json:"command,omitempty"`
	Args        *[]string       `json:"args,omitempty"`
	AuthType    *string         `json:"auth_type,omitempty" validate:"omitempty,oneof=none bearer oauth api_key"`
	AuthToken   *string         `json:"auth_token,omitempty"`
	AuthHeader  *string         `json:"auth_header,omitempty"`
	Config      *map[string]any `json:"config,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Description *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// MCPConfiguration represents an MCP configuration with metadata
type MCPConfiguration struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Endpoint    string         `json:"endpoint,omitempty"`
	Command     string         `json:"command,omitempty"`
	Args        []string       `json:"args,omitempty"`
	AuthType    string         `json:"auth_type"`
	AuthHeader  string         `json:"auth_header,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Scope       string         `json:"scope"`
	OrgID       string         `json:"org_id,omitempty"`
	UserID      string         `json:"user_id,omitempty"`
	Enabled     bool           `json:"enabled"`
	Description string         `json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by"`
}

// TestConnectionRequest represents a request to test MCP connection
type TestConnectionRequest struct {
	Name       string         `json:"name" validate:"required"`
	Type       string         `json:"type" validate:"required,oneof=http sse stdio"`
	Endpoint   string         `json:"endpoint,omitempty"`
	Command    string         `json:"command,omitempty"`
	Args       []string       `json:"args,omitempty"`
	AuthType   string         `json:"auth_type" validate:"required,oneof=none bearer oauth api_key"`
	AuthToken  string         `json:"auth_token,omitempty"`
	AuthHeader string         `json:"auth_header,omitempty"`
	Config     map[string]any `json:"config,omitempty"`
}

// TestConnectionResponse represents the response from testing an MCP connection
type TestConnectionResponse struct {
	Success   bool     `json:"success"`
	Error     string   `json:"error,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Prompts   []string `json:"prompts,omitempty"`
}

// ListMCPsResponse represents the response for listing MCPs
type ListMCPsResponse struct {
	Configurations []MCPConfiguration `json:"configurations"`
	Total          int                `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// Circuit Breaker Helper Methods

// handleCircuitBreakerError converts circuit breaker errors to HTTP responses
func (h *MCPHandler) handleCircuitBreakerError(w http.ResponseWriter, err error, operation string) {
	switch err {
	case resilience.ErrCircuitOpen:
		h.sendCircuitOpenResponse(w, operation)
	case resilience.ErrTooManyRequests:
		h.sendTooManyRequestsResponse(w, operation)
	default:
		// Other errors are handled normally
		h.sendError(w, err.Error(), http.StatusInternalServerError, "operation_failed")
	}
}

// sendCircuitOpenResponse sends a 503 Service Unavailable response when circuit is open
func (h *MCPHandler) sendCircuitOpenResponse(w http.ResponseWriter, operation string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "30") // Circuit timeout is 30 seconds
	w.WriteHeader(http.StatusServiceUnavailable)

	response := map[string]any{
		"error":   "Service temporarily unavailable",
		"code":    "circuit_breaker_open",
		"message": fmt.Sprintf("The %s operation is currently unavailable due to repeated failures. Please try again in 30 seconds.", operation),
		"details": map[string]any{
			"operation":           operation,
			"circuit_state":       "open",
			"retry_after_seconds": 30,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// sendTooManyRequestsResponse sends a 429 Too Many Requests response when breaker rejects
func (h *MCPHandler) sendTooManyRequestsResponse(w http.ResponseWriter, operation string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "5")
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]any{
		"error":   "Too many requests",
		"code":    "circuit_breaker_half_open",
		"message": fmt.Sprintf("The %s operation is recovering and cannot accept more requests at this time. Please try again shortly.", operation),
		"details": map[string]any{
			"operation":           operation,
			"circuit_state":       "half-open",
			"retry_after_seconds": 5,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// getDegradedServiceResponse returns a fallback response for degraded service
func (h *MCPHandler) getDegradedServiceResponse(operation string) map[string]any {
	return map[string]any{
		"status":    "degraded",
		"message":   fmt.Sprintf("The %s operation is currently degraded. Using cached or fallback data.", operation),
		"operation": operation,
		"timestamp": time.Now().UTC(),
	}
}

// logCircuitBreakerMetrics logs circuit breaker statistics
func (h *MCPHandler) logCircuitBreakerMetrics(ctx context.Context, operation string, breaker *resilience.CircuitBreaker) {
	stats := breaker.Stats()

	log.Printf("[Circuit Breaker Metrics] Operation: %s, State: %s, Total Requests: %d, Successes: %d, Failures: %d, Consecutive Failures: %d",
		operation,
		stats.State.String(),
		stats.TotalRequests,
		stats.TotalSuccesses,
		stats.TotalFailures,
		stats.ConsecutiveFailures,
	)

	// TODO: Export to Prometheus
	// h.metrics.RecordCircuitBreakerStats(operation, stats)
}

// Validation helpers

// validateURL validates and sanitizes URL input
func (h *MCPHandler) validateURL(urlStr string) error {
	if urlStr == "" {
		return nil // Optional for some types
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: only http and https are allowed")
	}

	// Validate hostname is not empty
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a valid host")
	}

	return nil
}

// validateCommand validates command for potential injection attacks
func (h *MCPHandler) validateCommand(command string) error {
	if command == "" {
		return nil // Optional for some types
	}

	// Check for suspicious characters that might indicate command injection
	dangerousPatterns := []string{
		";", "&&", "||", "|", "`", "$(",
		"\n", "\r", "<", ">", ">>",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("command contains potentially dangerous characters: %s", pattern)
		}
	}

	// Ensure command doesn't start with - (to prevent flag injection)
	if strings.HasPrefix(command, "-") {
		return fmt.Errorf("command cannot start with dash character")
	}

	// Validate command path (basic check)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9/_\.-]+$`, command)
	if !matched {
		return fmt.Errorf("command contains invalid characters")
	}

	return nil
}

// validateCommandArgs validates command arguments
func (h *MCPHandler) validateCommandArgs(args []string) error {
	for i, arg := range args {
		// Check for suspicious patterns
		if strings.Contains(arg, ";") || strings.Contains(arg, "&&") || strings.Contains(arg, "||") {
			return fmt.Errorf("argument %d contains potentially dangerous characters", i)
		}
	}
	return nil
}

// Encryption/Decryption helpers

// encrypt encrypts sensitive data using AES-256-GCM
func (h *MCPHandler) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(h.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts sensitive data using AES-256-GCM
func (h *MCPHandler) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(h.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HTTP Handlers

// CreateMCPConfiguration handles POST /api/v1/mcp/configurations
func (h *MCPHandler) CreateMCPConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	var req CreateMCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "invalid_json")
		return
	}

	// Validate request
	if err := h.validateCreateRequest(&req); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
		return
	}

	// Encrypt sensitive fields
	encryptedToken := ""
	if req.AuthToken != "" {
		var err error
		encryptedToken, err = h.encrypt(req.AuthToken)
		if err != nil {
			h.auditLogger.Log(ctx, userID, orgID, "mcp_create_failed", "mcp", "", map[string]any{
				"error": "encryption_failed",
			})
			h.sendError(w, "Failed to encrypt credentials", http.StatusInternalServerError, "encryption_error")
			return
		}
	}

	// Serialize config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		h.sendError(w, "Invalid configuration format", http.StatusBadRequest, "invalid_config")
		return
	}

	// Serialize args to JSON
	argsJSON, err := json.Marshal(req.Args)
	if err != nil {
		h.sendError(w, "Invalid args format", http.StatusBadRequest, "invalid_args")
		return
	}

	// Generate ID
	mcpID := generateMCPID()

	// Insert into database
	now := time.Now()
	scopeUserID := sql.NullString{}
	scopeOrgID := sql.NullString{}

	if req.Scope == "user" {
		scopeUserID = sql.NullString{String: userID, Valid: true}
	} else {
		scopeOrgID = sql.NullString{String: orgID, Valid: true}
	}

	query := `
		INSERT INTO mcp_configurations (
			id, name, type, endpoint, command, args, auth_type, auth_token,
			auth_header, config, scope, org_id, user_id, enabled, description,
			created_at, updated_at, created_by, updated_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = h.db.ExecContext(ctx, query,
		mcpID, req.Name, req.Type, req.Endpoint, req.Command, string(argsJSON),
		req.AuthType, encryptedToken, req.AuthHeader, string(configJSON),
		req.Scope, scopeOrgID, scopeUserID, req.Enabled, req.Description,
		now, now, userID, userID,
	)

	if err != nil {
		h.auditLogger.Log(ctx, userID, orgID, "mcp_create_failed", "mcp", mcpID, map[string]any{
			"error": err.Error(),
		})
		h.sendError(w, "Failed to create MCP configuration", http.StatusInternalServerError, "database_error")
		return
	}

	// Log successful creation
	h.auditLogger.Log(ctx, userID, orgID, "mcp_created", "mcp", mcpID, map[string]any{
		"name":  req.Name,
		"type":  req.Type,
		"scope": req.Scope,
	})

	// Build response
	config := MCPConfiguration{
		ID:          mcpID,
		Name:        req.Name,
		Type:        req.Type,
		Endpoint:    req.Endpoint,
		Command:     req.Command,
		Args:        req.Args,
		AuthType:    req.AuthType,
		AuthHeader:  req.AuthHeader,
		Config:      req.Config,
		Scope:       req.Scope,
		Enabled:     req.Enabled,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	if req.Scope == "org" {
		config.OrgID = orgID
	} else {
		config.UserID = userID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(config)
}

// ListMCPConfigurations handles GET /api/v1/mcp/configurations
func (h *MCPHandler) ListMCPConfigurations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	// Parse query parameters
	mcpType := r.URL.Query().Get("type")
	enabledStr := r.URL.Query().Get("enabled")

	// Build query
	query := `
		SELECT id, name, type, endpoint, command, args, auth_type, auth_token,
			   auth_header, config, scope, org_id, user_id, enabled, description,
			   created_at, updated_at, created_by, updated_by
		FROM mcp_configurations
		WHERE (org_id = ? OR user_id = ?)
	`
	args := []any{orgID, userID}

	if mcpType != "" {
		query += " AND type = ?"
		args = append(args, mcpType)
	}

	if enabledStr != "" {
		enabled := enabledStr == "true"
		query += " AND enabled = ?"
		args = append(args, enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		h.sendError(w, "Failed to query configurations", http.StatusInternalServerError, "database_error")
		return
	}
	defer rows.Close()

	var configurations []MCPConfiguration

	for rows.Next() {
		var config MCPConfiguration
		var encryptedToken string
		var configJSON, argsJSON string
		var scopeOrgID, scopeUserID sql.NullString

		err := rows.Scan(
			&config.ID, &config.Name, &config.Type, &config.Endpoint, &config.Command,
			&argsJSON, &config.AuthType, &encryptedToken, &config.AuthHeader,
			&configJSON, &config.Scope, &scopeOrgID, &scopeUserID, &config.Enabled,
			&config.Description, &config.CreatedAt, &config.UpdatedAt,
			&config.CreatedBy, &config.UpdatedBy,
		)

		if err != nil {
			h.sendError(w, "Failed to scan configuration", http.StatusInternalServerError, "database_error")
			return
		}

		// Deserialize config
		if configJSON != "" {
			if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
				// Log error but continue
				config.Config = make(map[string]any)
			}
		}

		// Deserialize args
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &config.Args); err != nil {
				// Log error but continue
				config.Args = []string{}
			}
		}

		// Set org/user IDs based on scope
		if scopeOrgID.Valid {
			config.OrgID = scopeOrgID.String
		}
		if scopeUserID.Valid {
			config.UserID = scopeUserID.String
		}

		// Note: We do NOT return the decrypted auth token for security
		// Tokens are only decrypted when actually needed for connection

		configurations = append(configurations, config)
	}

	if err = rows.Err(); err != nil {
		h.sendError(w, "Error reading configurations", http.StatusInternalServerError, "database_error")
		return
	}

	response := ListMCPsResponse{
		Configurations: configurations,
		Total:          len(configurations),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMCPConfiguration handles GET /api/v1/mcp/configurations/:id
func (h *MCPHandler) GetMCPConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	mcpID := extractIDFromPath(r.URL.Path, "/api/v1/mcp/configurations/")

	if mcpID == "" {
		h.sendError(w, "Invalid MCP ID", http.StatusBadRequest, "invalid_id")
		return
	}

	query := `
		SELECT id, name, type, endpoint, command, args, auth_type, auth_token,
			   auth_header, config, scope, org_id, user_id, enabled, description,
			   created_at, updated_at, created_by, updated_by
		FROM mcp_configurations
		WHERE id = ? AND (org_id = ? OR user_id = ?)
	`

	var config MCPConfiguration
	var encryptedToken string
	var configJSON, argsJSON string
	var scopeOrgID, scopeUserID sql.NullString

	err := h.db.QueryRowContext(ctx, query, mcpID, orgID, userID).Scan(
		&config.ID, &config.Name, &config.Type, &config.Endpoint, &config.Command,
		&argsJSON, &config.AuthType, &encryptedToken, &config.AuthHeader,
		&configJSON, &config.Scope, &scopeOrgID, &scopeUserID, &config.Enabled,
		&config.Description, &config.CreatedAt, &config.UpdatedAt,
		&config.CreatedBy, &config.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		h.sendError(w, "MCP configuration not found", http.StatusNotFound, "not_found")
		return
	}

	if err != nil {
		h.sendError(w, "Failed to retrieve configuration", http.StatusInternalServerError, "database_error")
		return
	}

	// Deserialize config
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config.Config); err != nil {
			config.Config = make(map[string]any)
		}
	}

	// Deserialize args
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &config.Args); err != nil {
			config.Args = []string{}
		}
	}

	// Set org/user IDs
	if scopeOrgID.Valid {
		config.OrgID = scopeOrgID.String
	}
	if scopeUserID.Valid {
		config.UserID = scopeUserID.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateMCPConfiguration handles PUT /api/v1/mcp/configurations/:id
func (h *MCPHandler) UpdateMCPConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	mcpID := extractIDFromPath(r.URL.Path, "/api/v1/mcp/configurations/")

	if mcpID == "" {
		h.sendError(w, "Invalid MCP ID", http.StatusBadRequest, "invalid_id")
		return
	}

	var req UpdateMCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "invalid_json")
		return
	}

	// Check ownership
	var scope string
	var scopeOrgID, scopeUserID sql.NullString
	checkQuery := `SELECT scope, org_id, user_id FROM mcp_configurations WHERE id = ?`
	err := h.db.QueryRowContext(ctx, checkQuery, mcpID).Scan(&scope, &scopeOrgID, &scopeUserID)

	if err == sql.ErrNoRows {
		h.sendError(w, "MCP configuration not found", http.StatusNotFound, "not_found")
		return
	}

	if err != nil {
		h.sendError(w, "Failed to check ownership", http.StatusInternalServerError, "database_error")
		return
	}

	// Verify tenant isolation
	hasAccess := false
	if scope == "org" && scopeOrgID.Valid && scopeOrgID.String == orgID {
		hasAccess = true
	} else if scope == "user" && scopeUserID.Valid && scopeUserID.String == userID {
		hasAccess = true
	}

	if !hasAccess {
		h.auditLogger.Log(ctx, userID, orgID, "mcp_update_forbidden", "mcp", mcpID, map[string]any{
			"reason": "insufficient_permissions",
		})
		h.sendError(w, "Insufficient permissions", http.StatusForbidden, "forbidden")
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []any{}

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}

	if req.Type != nil {
		if err := h.validateMCPType(*req.Type); err != nil {
			h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
			return
		}
		updates = append(updates, "type = ?")
		args = append(args, *req.Type)
	}

	if req.Endpoint != nil {
		if err := h.validateURL(*req.Endpoint); err != nil {
			h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
			return
		}
		updates = append(updates, "endpoint = ?")
		args = append(args, *req.Endpoint)
	}

	if req.Command != nil {
		if err := h.validateCommand(*req.Command); err != nil {
			h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
			return
		}
		updates = append(updates, "command = ?")
		args = append(args, *req.Command)
	}

	if req.Args != nil {
		if err := h.validateCommandArgs(*req.Args); err != nil {
			h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
			return
		}
		argsJSON, err := json.Marshal(*req.Args)
		if err != nil {
			h.sendError(w, "Invalid args format", http.StatusBadRequest, "invalid_args")
			return
		}
		updates = append(updates, "args = ?")
		args = append(args, string(argsJSON))
	}

	if req.AuthType != nil {
		updates = append(updates, "auth_type = ?")
		args = append(args, *req.AuthType)
	}

	if req.AuthToken != nil {
		encryptedToken, err := h.encrypt(*req.AuthToken)
		if err != nil {
			h.sendError(w, "Failed to encrypt credentials", http.StatusInternalServerError, "encryption_error")
			return
		}
		updates = append(updates, "auth_token = ?")
		args = append(args, encryptedToken)
	}

	if req.AuthHeader != nil {
		updates = append(updates, "auth_header = ?")
		args = append(args, *req.AuthHeader)
	}

	if req.Config != nil {
		configJSON, err := json.Marshal(*req.Config)
		if err != nil {
			h.sendError(w, "Invalid configuration format", http.StatusBadRequest, "invalid_config")
			return
		}
		updates = append(updates, "config = ?")
		args = append(args, string(configJSON))
	}

	if req.Enabled != nil {
		updates = append(updates, "enabled = ?")
		args = append(args, *req.Enabled)
	}

	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}

	if len(updates) == 0 {
		h.sendError(w, "No fields to update", http.StatusBadRequest, "no_updates")
		return
	}

	// Add updated timestamp and user
	updates = append(updates, "updated_at = ?", "updated_by = ?")
	args = append(args, time.Now(), userID)

	// Add WHERE clause parameters
	args = append(args, mcpID)

	query := fmt.Sprintf("UPDATE mcp_configurations SET %s WHERE id = ?", strings.Join(updates, ", "))

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		h.auditLogger.Log(ctx, userID, orgID, "mcp_update_failed", "mcp", mcpID, map[string]any{
			"error": err.Error(),
		})
		h.sendError(w, "Failed to update configuration", http.StatusInternalServerError, "database_error")
		return
	}

	// Log successful update
	h.auditLogger.Log(ctx, userID, orgID, "mcp_updated", "mcp", mcpID, map[string]any{
		"fields_updated": len(updates) - 2, // Exclude updated_at and updated_by
	})

	w.WriteHeader(http.StatusNoContent)
}

// DeleteMCPConfiguration handles DELETE /api/v1/mcp/configurations/:id
func (h *MCPHandler) DeleteMCPConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	mcpID := extractIDFromPath(r.URL.Path, "/api/v1/mcp/configurations/")

	if mcpID == "" {
		h.sendError(w, "Invalid MCP ID", http.StatusBadRequest, "invalid_id")
		return
	}

	// Check ownership before deletion
	var scope string
	var scopeOrgID, scopeUserID sql.NullString
	checkQuery := `SELECT scope, org_id, user_id FROM mcp_configurations WHERE id = ?`
	err := h.db.QueryRowContext(ctx, checkQuery, mcpID).Scan(&scope, &scopeOrgID, &scopeUserID)

	if err == sql.ErrNoRows {
		h.sendError(w, "MCP configuration not found", http.StatusNotFound, "not_found")
		return
	}

	if err != nil {
		h.sendError(w, "Failed to check ownership", http.StatusInternalServerError, "database_error")
		return
	}

	// Verify tenant isolation
	hasAccess := false
	if scope == "org" && scopeOrgID.Valid && scopeOrgID.String == orgID {
		hasAccess = true
	} else if scope == "user" && scopeUserID.Valid && scopeUserID.String == userID {
		hasAccess = true
	}

	if !hasAccess {
		h.auditLogger.Log(ctx, userID, orgID, "mcp_delete_forbidden", "mcp", mcpID, map[string]any{
			"reason": "insufficient_permissions",
		})
		h.sendError(w, "Insufficient permissions", http.StatusForbidden, "forbidden")
		return
	}

	// Disconnect active clients if connected (with circuit breaker protection)
	if h.fastmcpClient.IsConnected(mcpID) {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		disconnectErr := h.breakers.disconnect.Execute(disconnectCtx, func() error {
			return h.fastmcpClient.DisconnectMCP(disconnectCtx, mcpID)
		})

		if disconnectErr != nil {
			// Log warning but continue with deletion
			h.auditLogger.Log(ctx, userID, orgID, "mcp_disconnect_warning", "mcp", mcpID, map[string]any{
				"warning": "failed to disconnect before deletion",
				"error":   disconnectErr.Error(),
			})
		}
	}

	// Delete from database
	deleteQuery := `DELETE FROM mcp_configurations WHERE id = ?`
	_, err = h.db.ExecContext(ctx, deleteQuery, mcpID)

	if err != nil {
		h.auditLogger.Log(ctx, userID, orgID, "mcp_delete_failed", "mcp", mcpID, map[string]any{
			"error": err.Error(),
		})
		h.sendError(w, "Failed to delete configuration", http.StatusInternalServerError, "database_error")
		return
	}

	// Log successful deletion
	h.auditLogger.Log(ctx, userID, orgID, "mcp_deleted", "mcp", mcpID, nil)

	w.WriteHeader(http.StatusNoContent)
}

// TestMCPConnection handles POST /api/v1/mcp/test
func (h *MCPHandler) TestMCPConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserIDFromContext(ctx)
	orgID := getOrgIDFromContext(ctx)

	if userID == "" || orgID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "missing_auth")
		return
	}

	var req TestConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "invalid_json")
		return
	}

	// Validate the test request
	if err := h.validateTestRequest(&req); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "validation_error")
		return
	}

	// Create temporary MCP config for testing
	testID := fmt.Sprintf("test_%d", time.Now().UnixNano())

	authConfig := make(map[string]string)
	if req.AuthToken != "" {
		if req.AuthHeader != "" {
			authConfig[req.AuthHeader] = req.AuthToken
		} else {
			authConfig["token"] = req.AuthToken
		}
	}

	mcpConfig := mcp.MCPConfig{
		ID:       testID,
		Name:     req.Name,
		Type:     req.Type,
		Endpoint: req.Endpoint,
		AuthType: req.AuthType,
		Config:   req.Config,
		Auth:     authConfig,
	}

	// Try to connect with circuit breaker protection
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var response TestConnectionResponse

	// Wrap connection in circuit breaker
	err := h.breakers.testConnection.Execute(testCtx, func() error {
		connectErr := h.fastmcpClient.ConnectMCP(testCtx, mcpConfig)
		if connectErr != nil {
			return connectErr
		}

		// Connection successful - try to list tools, resources, and prompts
		defer h.fastmcpClient.DisconnectMCP(context.Background(), testID)

		response.Success = true

		// List tools with circuit breaker
		if listErr := h.breakers.listTools.Execute(testCtx, func() error {
			tools, toolsErr := h.fastmcpClient.ListTools(testCtx, testID)
			if toolsErr == nil {
				response.Tools = make([]string, len(tools))
				for i, tool := range tools {
					response.Tools[i] = tool.Name
				}
			}
			return toolsErr
		}); listErr != nil {
			// Log error but continue
			log.Printf("Failed to list tools during test: %v", listErr)
		}

		// List resources
		resources, resErr := h.fastmcpClient.ListResources(testCtx, testID)
		if resErr == nil {
			response.Resources = make([]string, len(resources))
			for i, res := range resources {
				response.Resources[i] = res.Name
			}
		}

		// List prompts
		prompts, promptErr := h.fastmcpClient.ListPrompts(testCtx, testID)
		if promptErr == nil {
			response.Prompts = make([]string, len(prompts))
			for i, prompt := range prompts {
				response.Prompts[i] = prompt.Name
			}
		}

		return nil
	})

	if err != nil {
		// Check if it's a circuit breaker error
		if err == resilience.ErrCircuitOpen || err == resilience.ErrTooManyRequests {
			h.handleCircuitBreakerError(w, err, "test_connection")
			return
		}

		// Regular connection failure
		h.auditLogger.Log(ctx, userID, orgID, "mcp_test_failed", "mcp", testID, map[string]any{
			"error": err.Error(),
		})

		response = TestConnectionResponse{
			Success: false,
			Error:   err.Error(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Log circuit breaker metrics
	h.logCircuitBreakerMetrics(ctx, "test_connection", h.breakers.testConnection)

	h.auditLogger.Log(ctx, userID, orgID, "mcp_test_success", "mcp", testID, map[string]any{
		"tools_count":     len(response.Tools),
		"resources_count": len(response.Resources),
		"prompts_count":   len(response.Prompts),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper methods

func (h *MCPHandler) validateCreateRequest(req *CreateMCPRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(req.Name) > 255 {
		return fmt.Errorf("name must be less than 255 characters")
	}

	if err := h.validateMCPType(req.Type); err != nil {
		return err
	}

	// Validate endpoint/command based on type
	if req.Type == "http" || req.Type == "sse" {
		if req.Endpoint == "" {
			return fmt.Errorf("endpoint is required for %s type", req.Type)
		}
		if err := h.validateURL(req.Endpoint); err != nil {
			return err
		}
	} else if req.Type == "stdio" {
		if req.Command == "" {
			return fmt.Errorf("command is required for stdio type")
		}
		if err := h.validateCommand(req.Command); err != nil {
			return err
		}
		if err := h.validateCommandArgs(req.Args); err != nil {
			return err
		}
	}

	// Validate auth type
	validAuthTypes := map[string]bool{"none": true, "bearer": true, "oauth": true, "api_key": true}
	if !validAuthTypes[req.AuthType] {
		return fmt.Errorf("invalid auth_type: must be one of none, bearer, oauth, api_key")
	}

	// Validate scope
	if req.Scope != "org" && req.Scope != "user" {
		return fmt.Errorf("invalid scope: must be either 'org' or 'user'")
	}

	if req.Description != "" && len(req.Description) > 1000 {
		return fmt.Errorf("description must be less than 1000 characters")
	}

	return nil
}

func (h *MCPHandler) validateTestRequest(req *TestConnectionRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if err := h.validateMCPType(req.Type); err != nil {
		return err
	}

	if req.Type == "http" || req.Type == "sse" {
		if req.Endpoint == "" {
			return fmt.Errorf("endpoint is required for %s type", req.Type)
		}
		if err := h.validateURL(req.Endpoint); err != nil {
			return err
		}
	} else if req.Type == "stdio" {
		if req.Command == "" {
			return fmt.Errorf("command is required for stdio type")
		}
		if err := h.validateCommand(req.Command); err != nil {
			return err
		}
		if err := h.validateCommandArgs(req.Args); err != nil {
			return err
		}
	}

	validAuthTypes := map[string]bool{"none": true, "bearer": true, "oauth": true, "api_key": true}
	if !validAuthTypes[req.AuthType] {
		return fmt.Errorf("invalid auth_type: must be one of none, bearer, oauth, api_key")
	}

	return nil
}

func (h *MCPHandler) validateMCPType(mcpType string) error {
	validTypes := map[string]bool{"http": true, "sse": true, "stdio": true}
	if !validTypes[mcpType] {
		return fmt.Errorf("invalid type: must be one of http, sse, stdio")
	}
	return nil
}

func (h *MCPHandler) sendError(w http.ResponseWriter, message string, statusCode int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func extractIDFromPath(path, prefix string) string {
	// Extract ID from path like /api/v1/mcp/configurations/{id}
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

func generateMCPID() string {
	return fmt.Sprintf("mcp_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// MCPConfig is used by FastMCP client
type MCPConfig struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Endpoint string            `json:"endpoint"`
	AuthType string            `json:"auth_type"`
	Config   map[string]any    `json:"config"`
	Auth     map[string]string `json:"auth"`
}

// Circuit Breaker-Protected MCP Operations
// These methods wrap FastMCP client calls with circuit breaker protection

// ConnectMCPWithBreaker connects to an MCP server with circuit breaker protection
func (h *MCPHandler) ConnectMCPWithBreaker(ctx context.Context, config mcp.MCPConfig) error {
	return h.breakers.connect.Execute(ctx, func() error {
		return h.fastmcpClient.ConnectMCP(ctx, config)
	})
}

// DisconnectMCPWithBreaker disconnects from an MCP server with circuit breaker protection
func (h *MCPHandler) DisconnectMCPWithBreaker(ctx context.Context, mcpID string) error {
	return h.breakers.disconnect.Execute(ctx, func() error {
		return h.fastmcpClient.DisconnectMCP(ctx, mcpID)
	})
}

// ListToolsWithBreaker lists tools from an MCP server with circuit breaker protection
func (h *MCPHandler) ListToolsWithBreaker(ctx context.Context, mcpID string) ([]mcp.Tool, error) {
	var tools []mcp.Tool
	err := h.breakers.listTools.Execute(ctx, func() error {
		var listErr error
		tools, listErr = h.fastmcpClient.ListTools(ctx, mcpID)
		return listErr
	})
	return tools, err
}

// CallToolWithBreaker calls an MCP tool with circuit breaker protection
func (h *MCPHandler) CallToolWithBreaker(ctx context.Context, mcpID, toolName string, args map[string]any) (any, error) {
	var result any
	err := h.breakers.callTool.Execute(ctx, func() error {
		var callErr error
		result, callErr = h.fastmcpClient.CallTool(ctx, mcpID, toolName, args)
		return callErr
	})
	return result, err
}

// GetCircuitBreakerStats returns statistics for all circuit breakers
func (h *MCPHandler) GetCircuitBreakerStats() map[string]resilience.CBStats {
	return map[string]resilience.CBStats{
		"connect":         h.breakers.connect.Stats(),
		"call_tool":       h.breakers.callTool.Stats(),
		"list_tools":      h.breakers.listTools.Stats(),
		"disconnect":      h.breakers.disconnect.Stats(),
		"test_connection": h.breakers.testConnection.Stats(),
	}
}

// GetCircuitBreakerState returns the state of all circuit breakers
func (h *MCPHandler) GetCircuitBreakerState() map[string]string {
	return map[string]string{
		"connect":         h.breakers.connect.State(),
		"call_tool":       h.breakers.callTool.State(),
		"list_tools":      h.breakers.listTools.State(),
		"disconnect":      h.breakers.disconnect.State(),
		"test_connection": h.breakers.testConnection.State(),
	}
}

// ResetCircuitBreakers resets all circuit breakers to closed state
func (h *MCPHandler) ResetCircuitBreakers() {
	h.breakers.connect.Reset()
	h.breakers.callTool.Reset()
	h.breakers.listTools.Reset()
	h.breakers.disconnect.Reset()
	h.breakers.testConnection.Reset()
	log.Println("[MCP Circuit Breakers] All circuit breakers have been reset")
}

// HealthCheck returns health status including circuit breaker states
func (h *MCPHandler) HealthCheck() map[string]any {
	states := h.GetCircuitBreakerState()
	stats := h.GetCircuitBreakerStats()

	health := map[string]any{
		"status": "healthy",
		"circuit_breakers": map[string]any{
			"states": states,
			"stats":  stats,
		},
	}

	// Check if any circuit breaker is open
	for _, state := range states {
		if state == "open" {
			health["status"] = "degraded"
			break
		}
	}

	return health
}

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/agentapi/lib/mcp"
	"github.com/coder/agentapi/lib/session"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create table
	schema := `
		CREATE TABLE mcp_configurations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			endpoint TEXT,
			command TEXT,
			args TEXT,
			auth_type TEXT NOT NULL,
			auth_token TEXT,
			auth_header TEXT,
			config TEXT,
			scope TEXT NOT NULL,
			org_id TEXT,
			user_id TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			description TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// setupTestHandler creates a test MCP handler
func setupTestHandler(t *testing.T) (*MCPHandler, *sql.DB) {
	db := setupTestDB(t)

	// Note: In real tests, you'd mock the FastMCPClient
	// For this example, we'll skip it since it requires external process
	sessionMgr := session.NewSessionManager("/tmp/test")
	auditLogger := NewAuditLogger()
	encryptionKey := "test-encryption-key-32-bytes!!"

	handler, err := NewMCPHandler(db, nil, sessionMgr, auditLogger, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	return handler, db
}

// mockContext creates a mock context with user and org IDs
func mockContext(r *http.Request, userID, orgID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDContextKey, userID)
	ctx = context.WithValue(ctx, orgIDContextKey, orgID)
	return r.WithContext(ctx)
}

type contextKey string

const (
	userIDContextKey contextKey = "userID"
	orgIDContextKey  contextKey = "orgID"
)

// Override context extraction for tests
func init() {
	// This would be replaced with actual implementation
}

func TestCreateMCPConfiguration(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	tests := []struct {
		name           string
		request        CreateMCPRequest
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid HTTP MCP",
			request: CreateMCPRequest{
				Name:     "Test HTTP MCP",
				Type:     "http",
				Endpoint: "https://api.example.com/mcp",
				AuthType: "bearer",
				AuthToken: "test-token",
				Scope:    "org",
				Enabled:  true,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var config MCPConfiguration
				if err := json.Unmarshal(body, &config); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if config.Name != "Test HTTP MCP" {
					t.Errorf("Expected name 'Test HTTP MCP', got '%s'", config.Name)
				}
			},
		},
		{
			name: "valid stdio MCP",
			request: CreateMCPRequest{
				Name:     "Test Stdio MCP",
				Type:     "stdio",
				Command:  "/usr/bin/python3",
				Args:     []string{"-m", "mcp_server"},
				AuthType: "none",
				Scope:    "user",
				Enabled:  true,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid type",
			request: CreateMCPRequest{
				Name:     "Invalid MCP",
				Type:     "invalid",
				AuthType: "none",
				Scope:    "org",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing endpoint for HTTP",
			request: CreateMCPRequest{
				Name:     "Missing Endpoint",
				Type:     "http",
				AuthType: "none",
				Scope:    "org",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "command injection attempt",
			request: CreateMCPRequest{
				Name:     "Injection Test",
				Type:     "stdio",
				Command:  "/usr/bin/python3; rm -rf /",
				AuthType: "none",
				Scope:    "user",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/configurations", bytes.NewBuffer(body))
			req = mockContext(req, "user-123", "org-456")

			w := httptest.NewRecorder()
			handler.CreateMCPConfiguration(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkResponse != nil && w.Code == http.StatusCreated {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestListMCPConfigurations(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	// Insert test data
	testData := []struct {
		id     string
		name   string
		scope  string
		orgID  string
		userID string
	}{
		{"mcp1", "Org MCP 1", "org", "org-456", ""},
		{"mcp2", "Org MCP 2", "org", "org-456", ""},
		{"mcp3", "User MCP 1", "user", "", "user-123"},
		{"mcp4", "Other Org MCP", "org", "org-999", ""},
	}

	for _, td := range testData {
		orgID := sql.NullString{}
		userID := sql.NullString{}
		if td.orgID != "" {
			orgID = sql.NullString{String: td.orgID, Valid: true}
		}
		if td.userID != "" {
			userID = sql.NullString{String: td.userID, Valid: true}
		}

		_, err := db.Exec(`
			INSERT INTO mcp_configurations
			(id, name, type, auth_type, scope, org_id, user_id, enabled, created_at, updated_at, created_by, updated_by)
			VALUES (?, ?, 'http', 'none', ?, ?, ?, 1, datetime('now'), datetime('now'), 'test', 'test')
		`, td.id, td.name, td.scope, orgID, userID)

		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/configurations", nil)
	req = mockContext(req, "user-123", "org-456")

	w := httptest.NewRecorder()
	handler.ListMCPConfigurations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ListMCPsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should return 3 configs (2 org + 1 user), not the other org's config
	if response.Total != 3 {
		t.Errorf("Expected 3 configurations, got %d", response.Total)
	}
}

func TestUpdateMCPConfiguration(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	// Insert test MCP
	mcpID := "test-mcp-1"
	_, err := db.Exec(`
		INSERT INTO mcp_configurations
		(id, name, type, auth_type, scope, org_id, enabled, created_at, updated_at, created_by, updated_by)
		VALUES (?, 'Original Name', 'http', 'none', 'org', 'org-456', 1, datetime('now'), datetime('now'), 'user-123', 'user-123')
	`, mcpID)

	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	newName := "Updated Name"
	updateReq := UpdateMCPRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mcp/configurations/"+mcpID, bytes.NewBuffer(body))
	req = mockContext(req, "user-123", "org-456")

	w := httptest.NewRecorder()
	handler.UpdateMCPConfiguration(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify update
	var name string
	err = db.QueryRow("SELECT name FROM mcp_configurations WHERE id = ?", mcpID).Scan(&name)
	if err != nil {
		t.Fatalf("Failed to query updated record: %v", err)
	}

	if name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", name)
	}
}

func TestDeleteMCPConfiguration(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	// Insert test MCP
	mcpID := "test-mcp-delete"
	_, err := db.Exec(`
		INSERT INTO mcp_configurations
		(id, name, type, auth_type, scope, org_id, enabled, created_at, updated_at, created_by, updated_by)
		VALUES (?, 'To Delete', 'http', 'none', 'org', 'org-456', 1, datetime('now'), datetime('now'), 'user-123', 'user-123')
	`, mcpID)

	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mcp/configurations/"+mcpID, nil)
	req = mockContext(req, "user-123", "org-456")

	w := httptest.NewRecorder()
	handler.DeleteMCPConfiguration(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify deletion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM mcp_configurations WHERE id = ?", mcpID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query deleted record: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected record to be deleted, but it still exists")
	}
}

func TestTenantIsolation(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	// Insert MCP for different org
	mcpID := "other-org-mcp"
	_, err := db.Exec(`
		INSERT INTO mcp_configurations
		(id, name, type, auth_type, scope, org_id, enabled, created_at, updated_at, created_by, updated_by)
		VALUES (?, 'Other Org MCP', 'http', 'none', 'org', 'org-999', 1, datetime('now'), datetime('now'), 'user-999', 'user-999')
	`, mcpID)

	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Try to access with different org
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/configurations/"+mcpID, nil)
	req = mockContext(req, "user-123", "org-456")

	w := httptest.NewRecorder()
	handler.GetMCPConfiguration(w, req)

	// Should return 404 due to tenant isolation
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d (tenant isolation), got %d", http.StatusNotFound, w.Code)
	}
}

func TestEncryptionDecryption(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	testCases := []string{
		"simple-token",
		"complex!@#$%^&*()token",
		"very-long-token-with-many-characters-to-test-encryption-of-larger-strings",
	}

	for _, tc := range testCases {
		encrypted, err := handler.encrypt(tc)
		if err != nil {
			t.Errorf("Failed to encrypt '%s': %v", tc, err)
			continue
		}

		decrypted, err := handler.decrypt(encrypted)
		if err != nil {
			t.Errorf("Failed to decrypt '%s': %v", tc, err)
			continue
		}

		if decrypted != tc {
			t.Errorf("Encryption/decryption mismatch. Expected '%s', got '%s'", tc, decrypted)
		}
	}
}

func TestValidation(t *testing.T) {
	handler, db := setupTestHandler(t)
	defer db.Close()

	t.Run("URL validation", func(t *testing.T) {
		validURLs := []string{
			"http://example.com",
			"https://api.example.com/path",
			"http://localhost:8080/api",
		}

		for _, url := range validURLs {
			if err := handler.validateURL(url); err != nil {
				t.Errorf("Valid URL rejected: %s - %v", url, err)
			}
		}

		invalidURLs := []string{
			"ftp://example.com",
			"javascript:alert(1)",
			"not-a-url",
		}

		for _, url := range invalidURLs {
			if err := handler.validateURL(url); err == nil {
				t.Errorf("Invalid URL accepted: %s", url)
			}
		}
	})

	t.Run("Command validation", func(t *testing.T) {
		validCommands := []string{
			"/usr/bin/python3",
			"/bin/sh",
			"node",
		}

		for _, cmd := range validCommands {
			if err := handler.validateCommand(cmd); err != nil {
				t.Errorf("Valid command rejected: %s - %v", cmd, err)
			}
		}

		invalidCommands := []string{
			"/usr/bin/python3; rm -rf /",
			"node && malicious",
			"$(evil)",
			"`backdoor`",
		}

		for _, cmd := range invalidCommands {
			if err := handler.validateCommand(cmd); err == nil {
				t.Errorf("Invalid command accepted: %s", cmd)
			}
		}
	})
}

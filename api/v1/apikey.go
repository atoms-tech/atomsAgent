package v1

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/middleware"
)

// APIKeyHandler handles API key management endpoints
type APIKeyHandler struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(logger *slog.Logger, db *sql.DB) *APIKeyHandler {
	return &APIKeyHandler{
		logger: logger,
		db:     db,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ExpiresInDays *int `json:"expires_in_days"`
}

// APIKeyResponse represents the response when creating an API key
type APIKeyResponse struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"` // Only returned on creation
	KeyHash     string    `json:"key_hash"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// ListAPIKeysResponse represents a list of API keys
type ListAPIKeysResponse struct {
	Keys []APIKeyResponse `json:"keys"`
	Count int             `json:"count"`
}

// GenerateAPIKey generates a random API key
func (h *APIKeyHandler) GenerateAPIKey() (string, string) {
	// Generate random key (32 bytes = 256 bits)
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	key := "sk_" + hex.EncodeToString(randomBytes)[:32] // 32 char suffix after sk_

	// Hash the key
	hash := sha256.Sum256([]byte(key))
	keyHash := fmt.Sprintf("%x", hash)

	return key, keyHash
}

// HandleCreateAPIKey handles POST /api/v1/api-keys
func (h *APIKeyHandler) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request, authUser *auth.AuthKitUser) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate API key
	key, keyHash := h.GenerateAPIKey()

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expires := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &expires
	}

	// Insert into database
	var id string
	var createdAt time.Time
	err := h.db.QueryRow(`
		INSERT INTO api_keys (user_id, organization_id, key_hash, name, description, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, true, $6)
		RETURNING id, created_at
	`, authUser.ID, authUser.OrgID, keyHash, req.Name, req.Description, expiresAt).
		Scan(&id, &createdAt)

	if err != nil {
		h.logger.Error("failed to create API key", "error", err)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Return response (only return the plaintext key once)
	response := APIKeyResponse{
		ID:          id,
		Key:         key, // Only returned here
		KeyHash:     keyHash,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("API key created", "key_id", id, "user_id", authUser.ID, "org_id", authUser.OrgID)
}

// HandleListAPIKeys handles GET /api/v1/api-keys
func (h *APIKeyHandler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request, authUser *auth.AuthKitUser) {
	rows, err := h.db.Query(`
		SELECT id, key_hash, name, description, is_active, expires_at, created_at, last_used_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, authUser.ID)

	if err != nil {
		h.logger.Error("failed to list API keys", "error", err)
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var keys []APIKeyResponse
	for rows.Next() {
		var id, keyHash, name, description string
		var isActive bool
		var expiresAt, createdAt, lastUsedAt sql.NullTime

		err := rows.Scan(&id, &keyHash, &name, &description, &isActive, &expiresAt, &createdAt, &lastUsedAt)
		if err != nil {
			h.logger.Error("failed to scan API key", "error", err)
			continue
		}

		key := APIKeyResponse{
			ID:          id,
			KeyHash:     keyHash,
			Name:        name,
			Description: description,
			IsActive:    isActive,
			CreatedAt:   createdAt.Time,
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		keys = append(keys, key)
	}

	response := ListAPIKeysResponse{
		Keys:  keys,
		Count: len(keys),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDeleteAPIKey handles DELETE /api/v1/api-keys/{id}
func (h *APIKeyHandler) HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request, authUser *auth.AuthKitUser) {
	keyID := r.PathValue("id")
	if keyID == "" {
		http.Error(w, "Key ID is required", http.StatusBadRequest)
		return
	}

	// Soft delete: mark as inactive
	result, err := h.db.Exec(`
		UPDATE api_keys
		SET is_active = false
		WHERE id = $1 AND user_id = $2
	`, keyID, authUser.ID)

	if err != nil {
		h.logger.Error("failed to delete API key", "error", err)
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "API key deleted successfully",
		"id":      keyID,
	})

	h.logger.Info("API key deleted", "key_id", keyID, "user_id", authUser.ID)
}

// HandleRevokeAPIKey handles POST /api/v1/api-keys/{id}/revoke
func (h *APIKeyHandler) HandleRevokeAPIKey(w http.ResponseWriter, r *http.Request, authUser *auth.AuthKitUser) {
	keyID := r.PathValue("id")
	if keyID == "" {
		http.Error(w, "Key ID is required", http.StatusBadRequest)
		return
	}

	// Mark as inactive
	result, err := h.db.Exec(`
		UPDATE api_keys
		SET is_active = false
		WHERE id = $1 AND user_id = $2
	`, keyID, authUser.ID)

	if err != nil {
		h.logger.Error("failed to revoke API key", "error", err)
		http.Error(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "API key revoked successfully",
		"id":      keyID,
	})

	h.logger.Info("API key revoked", "key_id", keyID, "user_id", authUser.ID)
}

// RegisterAPIKeyRoutes registers API key management routes
func RegisterAPIKeyRoutes(
	router *http.ServeMux,
	logger *slog.Logger,
	db *sql.DB,
	authMiddleware *middleware.TieredAccessMiddleware,
) {
	handler := NewAPIKeyHandler(logger, db)

	// POST /api/v1/api-keys - Create API key
	router.HandleFunc("POST /api/v1/api-keys", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser := r.Context().Value("auth_user").(*auth.AuthKitUser)
			handler.HandleCreateAPIKey(w, r, authUser)
		})).ServeHTTP(w, r)
	})

	// GET /api/v1/api-keys - List API keys
	router.HandleFunc("GET /api/v1/api-keys", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser := r.Context().Value("auth_user").(*auth.AuthKitUser)
			handler.HandleListAPIKeys(w, r, authUser)
		})).ServeHTTP(w, r)
	})

	// DELETE /api/v1/api-keys/{id} - Delete API key
	router.HandleFunc("DELETE /api/v1/api-keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser := r.Context().Value("auth_user").(*auth.AuthKitUser)
			handler.HandleDeleteAPIKey(w, r, authUser)
		})).ServeHTTP(w, r)
	})

	// POST /api/v1/api-keys/{id}/revoke - Revoke API key
	router.HandleFunc("POST /api/v1/api-keys/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser := r.Context().Value("auth_user").(*auth.AuthKitUser)
			handler.HandleRevokeAPIKey(w, r, authUser)
		})).ServeHTTP(w, r)
	})

	logger.Info("API key management routes registered")
}

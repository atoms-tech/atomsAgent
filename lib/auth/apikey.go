package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log/slog"
)

// APIKeyValidator validates API keys for service-to-service authentication
type APIKeyValidator struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewAPIKeyValidator creates a new API key validator
func NewAPIKeyValidator(logger *slog.Logger, db *sql.DB) *APIKeyValidator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}
	return &APIKeyValidator{
		logger: logger,
		db:     db,
	}
}

// ValidateAPIKey validates an API key and returns user information
// API keys are hashed with SHA256 and stored in the database
func (v *APIKeyValidator) ValidateAPIKey(ctx context.Context, apiKey string) (*AuthKitUser, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("missing API key")
	}

	if v.db == nil {
		return nil, fmt.Errorf("database connection required for API key validation")
	}

	// Hash the API key (SHA256)
	keyHash := hashAPIKey(apiKey)

	// Query database for API key
	var userID, orgID, email, name string
	var isPlatformAdmin bool

	err := v.db.QueryRowContext(
		ctx,
		`SELECT
			api_keys.user_id,
			api_keys.organization_id,
			users.email,
			users.name,
			CASE WHEN platform_admins.user_id IS NOT NULL THEN true ELSE false END as is_platform_admin
		FROM api_keys
		LEFT JOIN users ON api_keys.user_id = users.id
		LEFT JOIN platform_admins ON api_keys.user_id = platform_admins.workos_user_id
		WHERE api_keys.key_hash = $1
		AND api_keys.is_active = true
		AND (api_keys.expires_at IS NULL OR api_keys.expires_at > NOW())
		LIMIT 1`,
		keyHash,
	).Scan(&userID, &orgID, &email, &name, &isPlatformAdmin)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid or expired API key")
	} else if err != nil {
		v.logger.Error("failed to query API key", "error", err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return &AuthKitUser{
		ID:                   userID,
		OrgID:                orgID,
		Email:                email,
		Name:                 name,
		IsPlatformAdminFlag:  isPlatformAdmin,
		AuthenticationMethod: "api_key",
	}, nil
}

// hashAPIKey hashes an API key using SHA256
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash)
}

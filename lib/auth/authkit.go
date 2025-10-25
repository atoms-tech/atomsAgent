package auth

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthKitClaims represents WorkOS/AuthKit JWT claims
type AuthKitClaims struct {
	Sub           string   `json:"sub"`    // User ID
	Org           string   `json:"org"`    // Organization ID (WorkOS)
	OrgID         string   `json:"org_id"` // Organization ID (Supabase)
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	Picture       string   `json:"picture"`
	Role          string   `json:"role"` // admin, member, viewer, etc.
	Permissions   []string `json:"permissions"`
	Iss           string   `json:"iss"` // Issuer (for identifying token type)
	jwt.RegisteredClaims
}

// AuthKitValidator validates WorkOS/AuthKit and Supabase JWT tokens
type AuthKitValidator struct {
	logger            *slog.Logger
	jwksURL           string // WorkOS JWKS URL
	supabaseJWKSURL   string // Supabase JWKS URL (optional, for hybrid auth)
	publicKeys        map[string]*rsa.PublicKey
	supabaseKeys      map[string]*rsa.PublicKey
	mu                sync.RWMutex
	keyExpiry         time.Time
	supabaseKeyExpiry time.Time
	keyRefreshTTL     time.Duration
	db                *sql.DB // Database connection for platform admin checks
}

// JWKSResponse from WorkOS
type JWKSResponse struct {
	Keys []AuthKitJWK `json:"keys"`
}

// AuthKitJWK represents a JSON Web Key for AuthKit
type AuthKitJWK struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// AuthKitUser represents authenticated user information
type AuthKitUser struct {
	ID          string
	OrgID       string
	Email       string
	Name        string
	Role        string
	Permissions []string

	// Authorization levels
	IsOrgAdminFlag      bool // true if Role == "admin" (from WorkOS)
	IsPlatformAdminFlag bool // true if in platform_admins table (from DB)

	Token string
}

// NewAuthKitValidator creates a new AuthKit JWT validator
func NewAuthKitValidator(logger *slog.Logger, jwksURL string, db *sql.DB) *AuthKitValidator {
	return NewAuthKitValidatorWithSupabase(logger, jwksURL, "", db)
}

// NewAuthKitValidatorWithSupabase creates a new AuthKit JWT validator with optional Supabase support
func NewAuthKitValidatorWithSupabase(logger *slog.Logger, jwksURL, supabaseJWKSURL string, db *sql.DB) *AuthKitValidator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	// Build Supabase JWKS URL from environment if not provided
	if supabaseJWKSURL == "" {
		supabaseURL := os.Getenv("SUPABASE_URL")
		if supabaseURL != "" {
			supabaseJWKSURL = fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", strings.TrimSuffix(supabaseURL, "/"))
		}
	}

	return &AuthKitValidator{
		logger:          logger,
		jwksURL:         jwksURL,
		supabaseJWKSURL: supabaseJWKSURL,
		publicKeys:      make(map[string]*rsa.PublicKey),
		supabaseKeys:    make(map[string]*rsa.PublicKey),
		keyRefreshTTL:   24 * time.Hour,
		db:              db,
	}
}

// ValidateToken validates a JWT token (either WorkOS or Supabase) and returns user info
func (av *AuthKitValidator) ValidateToken(ctx context.Context, tokenString string) (*AuthKitUser, error) {
	// Parse JWT without verification first to get claims and identify issuer
	unverifiedClaims := &AuthKitClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, unverifiedClaims)
	if err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Debug logging to see what issuer we got
	av.logger.Debug("token inspection",
		"issuer", unverifiedClaims.Iss,
		"has_org", unverifiedClaims.Org != "",
		"has_org_id", unverifiedClaims.OrgID != "",
		"has_sub", unverifiedClaims.Sub != "")

	// Determine token type based on structure
	// WorkOS tokens ALWAYS have 'org' claim (workspace/organization ID)
	// Supabase tokens have 'org_id' claim but NOT 'org' claim
	// If token lacks 'org' claim, treat it as Supabase (try Supabase validation)
	// This handles cases where issuer claim may be misleading
	isSupabase := unverifiedClaims.Org == ""

	if isSupabase {
		// Validate as Supabase token
		av.logger.Info("routing to supabase validator",
			"reason", "missing 'org' claim (Supabase marker)",
			"has_org_id", unverifiedClaims.OrgID != "")
		return av.validateSupabaseToken(ctx, tokenString, unverifiedClaims)
	}

	// Validate as WorkOS token (has 'org' claim)
	av.logger.Info("routing to workos validator", "has_org_claim", true)
	return av.validateWorkOSToken(ctx, tokenString, unverifiedClaims)
}

// validateWorkOSToken validates a WorkOS/AuthKit JWT token
func (av *AuthKitValidator) validateWorkOSToken(ctx context.Context, tokenString string, claims *AuthKitClaims) (*AuthKitUser, error) {
	// Ensure JWKS keys are loaded and fresh
	if err := av.ensureKeysLoaded(ctx); err != nil {
		return nil, fmt.Errorf("failed to load JWKS keys: %w", err)
	}

	// Parse and verify JWT
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Get public key from WorkOS keys
		av.mu.RLock()
		key, ok := av.publicKeys[kid]
		av.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("key not found: %s", kid)
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Validate required claims
	if claims.Sub == "" {
		return nil, fmt.Errorf("missing 'sub' claim")
	}

	if claims.Org == "" {
		return nil, fmt.Errorf("missing 'org' claim")
	}

	// Validate expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate issued at
	if claims.IssuedAt != nil && claims.IssuedAt.After(time.Now().Add(5*time.Minute)) {
		return nil, fmt.Errorf("token issued in the future")
	}

	// Check if user is platform admin
	var isPlatformAdmin bool
	if av.db != nil {
		err := av.db.QueryRowContext(
			ctx,
			"SELECT true FROM platform_admins WHERE workos_user_id = $1 AND is_active = true LIMIT 1",
			claims.Sub,
		).Scan(&isPlatformAdmin)

		if err == sql.ErrNoRows {
			isPlatformAdmin = false
		} else if err != nil {
			av.logger.Error("failed to check platform admin status", "error", err)
			isPlatformAdmin = false
		}
	}

	return &AuthKitUser{
		ID:                  claims.Sub,
		OrgID:               claims.Org,
		Email:               claims.Email,
		Name:                claims.Name,
		Role:                claims.Role,
		Permissions:         claims.Permissions,
		IsOrgAdminFlag:      claims.Role == "admin",
		IsPlatformAdminFlag: isPlatformAdmin,
		Token:               tokenString,
	}, nil
}

// validateSupabaseToken validates a Supabase JWT token
func (av *AuthKitValidator) validateSupabaseToken(ctx context.Context, tokenString string, claims *AuthKitClaims) (*AuthKitUser, error) {
	// Load Supabase JWKS if available
	if av.supabaseJWKSURL == "" {
		return nil, fmt.Errorf("Supabase authentication not configured")
	}

	if err := av.ensureSupabaseKeysLoaded(ctx); err != nil {
		return nil, fmt.Errorf("failed to load Supabase JWKS keys: %w", err)
	}

	// Parse and verify JWT with Supabase keys
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Get public key from Supabase keys
		av.mu.RLock()
		key, ok := av.supabaseKeys[kid]
		av.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("key not found: %s", kid)
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Validate required claims for Supabase
	if claims.Sub == "" {
		return nil, fmt.Errorf("missing 'sub' claim")
	}

	// Use org_id if present, otherwise use a default org
	orgID := claims.OrgID
	if orgID == "" {
		orgID = "default"
	}

	// Validate expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate issued at
	if claims.IssuedAt != nil && claims.IssuedAt.After(time.Now().Add(5*time.Minute)) {
		return nil, fmt.Errorf("token issued in the future")
	}

	// Check if user is platform admin (using Supabase user ID)
	var isPlatformAdmin bool
	if av.db != nil {
		err := av.db.QueryRowContext(
			ctx,
			"SELECT true FROM platform_admins WHERE supabase_user_id = $1 AND is_active = true LIMIT 1",
			claims.Sub,
		).Scan(&isPlatformAdmin)

		if err == sql.ErrNoRows {
			isPlatformAdmin = false
		} else if err != nil {
			av.logger.Error("failed to check platform admin status", "error", err)
			isPlatformAdmin = false
		}
	}

	return &AuthKitUser{
		ID:                  claims.Sub,
		OrgID:               orgID,
		Email:               claims.Email,
		Name:                claims.Name,
		Role:                claims.Role,
		Permissions:         claims.Permissions,
		IsOrgAdminFlag:      claims.Role == "admin",
		IsPlatformAdminFlag: isPlatformAdmin,
		Token:               tokenString,
	}, nil
}

// ExtractBearerToken extracts JWT token from Authorization header
func (av *AuthKitValidator) ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// ensureKeysLoaded loads JWKS keys if they're not cached or expired
func (av *AuthKitValidator) ensureKeysLoaded(ctx context.Context) error {
	av.mu.RLock()
	if len(av.publicKeys) > 0 && time.Now().Before(av.keyExpiry) {
		av.mu.RUnlock()
		return nil
	}
	av.mu.RUnlock()

	// Load JWKS
	req, err := http.NewRequestWithContext(ctx, "GET", av.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwksResp JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	// Convert JWKs to RSA public keys
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range jwksResp.Keys {
		if key.Alg != "RS256" || key.Kty != "RSA" {
			continue // Skip non-RS256 keys
		}

		pubKey, err := av.jwkToRSAPublicKey(&key)
		if err != nil {
			av.logger.Warn("failed to convert JWK to RSA key",
				"kid", key.Kid,
				"error", err.Error(),
			)
			continue
		}

		keys[key.Kid] = pubKey
	}

	if len(keys) == 0 {
		return fmt.Errorf("no valid RS256 keys found in JWKS")
	}

	av.mu.Lock()
	av.publicKeys = keys
	av.keyExpiry = time.Now().Add(av.keyRefreshTTL)
	av.mu.Unlock()

	av.logger.Info("loaded JWKS keys", "count", len(keys))
	return nil
}

// ensureSupabaseKeysLoaded loads Supabase JWKS keys if they're not cached or expired
func (av *AuthKitValidator) ensureSupabaseKeysLoaded(ctx context.Context) error {
	av.mu.RLock()
	if len(av.supabaseKeys) > 0 && time.Now().Before(av.supabaseKeyExpiry) {
		av.mu.RUnlock()
		return nil
	}
	av.mu.RUnlock()

	// Load JWKS from Supabase with retry logic
	var lastErr error
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, etc.
			backoff := time.Duration(100*(attempt)) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", av.supabaseJWKSURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Retry on 5xx errors, but not on 4xx errors
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("Supabase JWKS endpoint returned status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries)
			av.logger.Debug("Supabase JWKS fetch failed, will retry", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Supabase JWKS endpoint returned status %d", resp.StatusCode)
		}

		// Successfully got response, decode it
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read Supabase JWKS response: %w", err)
		}

		var jwksResp JWKSResponse
		if err := json.Unmarshal(body, &jwksResp); err != nil {
			return fmt.Errorf("failed to decode Supabase JWKS response: %w", err)
		}

		return av.processSupabaseKeys(&jwksResp)
	}

	// All retries exhausted
	return fmt.Errorf("failed to load Supabase JWKS after %d attempts: %w", maxRetries, lastErr)
}

// processSupabaseKeys converts JWKS keys and stores them in cache
func (av *AuthKitValidator) processSupabaseKeys(jwksResp *JWKSResponse) error {
	// Convert JWKs to RSA public keys
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range jwksResp.Keys {
		if key.Alg != "RS256" || key.Kty != "RSA" {
			continue // Skip non-RS256 keys
		}

		pubKey, err := av.jwkToRSAPublicKey(&key)
		if err != nil {
			av.logger.Warn("failed to convert Supabase JWK to RSA key",
				"kid", key.Kid,
				"error", err.Error(),
			)
			continue
		}

		keys[key.Kid] = pubKey
	}

	if len(keys) == 0 {
		return fmt.Errorf("no valid RS256 keys found in Supabase JWKS")
	}

	av.mu.Lock()
	av.supabaseKeys = keys
	av.supabaseKeyExpiry = time.Now().Add(av.keyRefreshTTL)
	av.mu.Unlock()

	av.logger.Info("loaded Supabase JWKS keys", "count", len(keys))
	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func (av *AuthKitValidator) jwkToRSAPublicKey(jwk *AuthKitJWK) (*rsa.PublicKey, error) {
	// Decode N (modulus) and E (exponent)
	nBytes, err := decodeBase64URL(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := decodeBase64URL(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert big-endian bytes to big.Int
	var nBigInt, eBigInt big.Int
	nBigInt.SetBytes(nBytes)
	eBigInt.SetBytes(eBytes)

	// Create RSA public key
	return &rsa.PublicKey{
		N: &nBigInt,
		E: int(eBigInt.Int64()),
	}, nil
}

// Helper function to decode base64url
func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	return base64.URLEncoding.DecodeString(s)
}

// IsOrgAdmin checks if user is an organization admin
func (user *AuthKitUser) IsOrgAdmin() bool {
	return user.IsOrgAdminFlag // Organization-level admin
}

// IsPlatformAdmin checks if user is a platform admin
func (user *AuthKitUser) IsPlatformAdmin() bool {
	return user.IsPlatformAdminFlag // Platform-level admin
}

// IsAnyAdmin checks if user has any admin privileges
func (user *AuthKitUser) IsAnyAdmin() bool {
	return user.IsOrgAdmin() || user.IsPlatformAdmin()
}

// IsAdmin checks if user has admin role (backward compatibility)
func (user *AuthKitUser) IsAdmin() bool {
	return user.IsOrgAdmin()
}

// CanManageOrg checks if user can manage a specific organization
func (user *AuthKitUser) CanManageOrg(orgID string) bool {
	if user.IsPlatformAdmin() {
		return true // Platform admins can manage any org
	}
	return user.Role == "admin" && user.OrgID == orgID // Org admins manage only their org
}

// HasPermission checks if user has specific permission
func (user *AuthKitUser) HasPermission(permission string) bool {
	for _, p := range user.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// String returns user representation for logging
func (user *AuthKitUser) String() string {
	return fmt.Sprintf("AuthKitUser{ID:%s, Org:%s, Email:%s, Role:%s}",
		user.ID, user.OrgID, user.Email, user.Role)
}

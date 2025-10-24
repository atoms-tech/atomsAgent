package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Context keys for storing user information
type contextKey string

const (
	// ContextKeyClaims is the key for storing JWT claims in context
	ContextKeyClaims contextKey = "claims"
)

// UserRole defines the user role type
type UserRole string

const (
	// RoleAdmin represents an administrator user
	RoleAdmin UserRole = "admin"
	// RoleUser represents a standard user
	RoleUser UserRole = "user"
)

// Claims represents the JWT claims structure
type Claims struct {
	Sub   string   `json:"sub"`           // User ID (subject)
	Email string   `json:"email"`         // User email
	OrgID string   `json:"org_id"`        // Organization ID
	Role  UserRole `json:"role"`          // User role
	Exp   int64    `json:"exp"`           // Expiration time
	Iat   int64    `json:"iat"`           // Issued at
	Aud   string   `json:"aud,omitempty"` // Audience
	Iss   string   `json:"iss,omitempty"` // Issuer
	jwt.RegisteredClaims
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSet represents a set of JSON Web Keys
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// KeyManager handles JWT verification keys
type KeyManager struct {
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	jwksURL     string
	lastRefresh time.Time
	logger      *slog.Logger
}

// NewKeyManager creates a new key manager
func NewKeyManager(jwksURL string, logger *slog.Logger) *KeyManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &KeyManager{
		keys:    make(map[string]*rsa.PublicKey),
		jwksURL: jwksURL,
		logger:  logger,
	}
}

// RefreshKeys fetches and updates the public keys from JWKS endpoint
func (km *KeyManager) RefreshKeys() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Don't refresh more than once per minute
	if time.Since(km.lastRefresh) < time.Minute {
		return nil
	}

	resp, err := http.Get(km.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Parse and store the keys
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		key, err := km.parseRSAKey(jwk)
		if err != nil {
			km.logger.Warn("Failed to parse JWK", "kid", jwk.Kid, "error", err)
			continue
		}

		km.keys[jwk.Kid] = key
	}

	km.lastRefresh = time.Now()
	km.logger.Info("Refreshed JWT keys", "count", len(km.keys))
	return nil
}

// parseRSAKey converts a JWK to an RSA public key
func (km *KeyManager) parseRSAKey(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: eInt,
	}, nil
}

// GetKey retrieves a public key by kid
func (km *KeyManager) GetKey(kid string) (*rsa.PublicKey, error) {
	km.mu.RLock()
	key, ok := km.keys[kid]
	km.mu.RUnlock()

	if !ok {
		// Try refreshing keys
		if err := km.RefreshKeys(); err != nil {
			return nil, fmt.Errorf("failed to refresh keys: %w", err)
		}

		km.mu.RLock()
		key, ok = km.keys[kid]
		km.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("key not found: %s", kid)
		}
	}

	return key, nil
}

// AuthConfig holds the authentication middleware configuration
type AuthConfig struct {
	JWKSUrl       string        // Supabase JWKS URL
	Issuer        string        // Expected JWT issuer
	Audience      string        // Expected JWT audience
	Logger        *slog.Logger  // Logger instance
	SkipPaths     []string      // Paths to skip authentication
	RefreshPeriod time.Duration // Key refresh period
}

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	config     AuthConfig
	keyManager *KeyManager
	logger     *slog.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config AuthConfig) (*AuthMiddleware, error) {
	if config.JWKSUrl == "" {
		// Try to get from environment
		supabaseURL := os.Getenv("SUPABASE_URL")
		if supabaseURL == "" {
			return nil, errors.New("SUPABASE_URL environment variable not set")
		}
		config.JWKSUrl = fmt.Sprintf("%s/auth/v1/jwks", strings.TrimSuffix(supabaseURL, "/"))
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	if config.RefreshPeriod == 0 {
		config.RefreshPeriod = 5 * time.Minute
	}

	keyManager := NewKeyManager(config.JWKSUrl, config.Logger)

	// Initial key fetch
	if err := keyManager.RefreshKeys(); err != nil {
		config.Logger.Warn("Failed to fetch initial keys, will retry on first request", "error", err)
	}

	am := &AuthMiddleware{
		config:     config,
		keyManager: keyManager,
		logger:     config.Logger,
	}

	// Start background key refresh
	go am.startKeyRefresh()

	return am, nil
}

// startKeyRefresh periodically refreshes the JWT verification keys
func (am *AuthMiddleware) startKeyRefresh() {
	ticker := time.NewTicker(am.config.RefreshPeriod)
	defer ticker.Stop()

	for range ticker.C {
		if err := am.keyManager.RefreshKeys(); err != nil {
			am.logger.Error("Failed to refresh JWT keys", "error", err)
		}
	}
}

// shouldSkipAuth checks if the path should skip authentication
func (am *AuthMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range am.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// Middleware returns the HTTP middleware handler
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for configured paths
		if am.shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.logger.Debug("Missing Authorization header", "path", r.URL.Path)
			am.respondError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			am.logger.Debug("Invalid Authorization header format", "path", r.URL.Path)
			am.respondError(w, http.StatusUnauthorized, "Invalid Authorization header format. Expected: Bearer <token>")
			return
		}

		tokenString := parts[1]

		// Validate the JWT
		claims, err := am.ValidateSupabaseJWT(tokenString)
		if err != nil {
			am.logger.Debug("JWT validation failed", "error", err, "path", r.URL.Path)
			am.respondError(w, http.StatusUnauthorized, fmt.Sprintf("Invalid token: %s", err.Error()))
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)

		// Log successful authentication
		am.logger.Debug("User authenticated",
			"user_id", claims.Sub,
			"org_id", claims.OrgID,
			"role", claims.Role,
			"path", r.URL.Path,
		)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateSupabaseJWT validates a Supabase JWT token and returns the claims
func (am *AuthMiddleware) ValidateSupabaseJWT(tokenString string) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the kid from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Get the verification key
		key, err := am.keyManager.GetKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get verification key: %w", err)
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, errors.New("token has expired")
	}

	// Validate issuer if configured
	if am.config.Issuer != "" && claims.Iss != am.config.Issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", am.config.Issuer, claims.Iss)
	}

	// Validate audience if configured
	if am.config.Audience != "" && claims.Aud != am.config.Audience {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", am.config.Audience, claims.Aud)
	}

	// Validate required fields
	if claims.Sub == "" {
		return nil, errors.New("missing sub claim")
	}

	if claims.Email == "" {
		return nil, errors.New("missing email claim")
	}

	if claims.OrgID == "" {
		return nil, errors.New("missing org_id claim")
	}

	if claims.Role == "" {
		return nil, errors.New("missing role claim")
	}

	// Validate role
	if claims.Role != RoleAdmin && claims.Role != RoleUser {
		return nil, fmt.Errorf("invalid role: %s", claims.Role)
	}

	return claims, nil
}

// respondError sends a JSON error response
func (am *AuthMiddleware) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   http.StatusText(status),
		"message": message,
	})
}

// GetUserFromContext extracts user ID and org ID from context
func GetUserFromContext(ctx context.Context) (userID string, orgID string, err error) {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	if !ok {
		return "", "", errors.New("no claims found in context")
	}

	return claims.Sub, claims.OrgID, nil
}

// GetClaimsFromContext extracts the full claims from context
func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	if !ok {
		return nil, errors.New("no claims found in context")
	}

	return claims, nil
}

// IsAdmin checks if the user in the context is an admin
func IsAdmin(ctx context.Context) bool {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	if !ok {
		return false
	}

	return claims.Role == RoleAdmin
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(requiredRole UserRole, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ContextKeyClaims).(*Claims)
			if !ok {
				logger.Debug("No claims in context", "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Unauthorized",
					"message": "Authentication required",
				})
				return
			}

			if claims.Role != requiredRole {
				logger.Debug("Insufficient permissions",
					"user_id", claims.Sub,
					"required_role", requiredRole,
					"user_role", claims.Role,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Forbidden",
					"message": fmt.Sprintf("Role '%s' required", requiredRole),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminRole is a convenience middleware that requires admin role
func RequireAdminRole(logger *slog.Logger) func(http.Handler) http.Handler {
	return RequireRole(RoleAdmin, logger)
}

// RequireUserRole is a convenience middleware that requires user role (or higher)
func RequireUserRole(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ContextKeyClaims).(*Claims)
			if !ok {
				if logger == nil {
					logger = slog.Default()
				}
				logger.Debug("No claims in context", "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Unauthorized",
					"message": "Authentication required",
				})
				return
			}

			// Allow both user and admin roles
			if claims.Role != RoleUser && claims.Role != RoleAdmin {
				logger.Debug("Insufficient permissions",
					"user_id", claims.Sub,
					"user_role", claims.Role,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Forbidden",
					"message": "User role required",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateSupabaseJWT is a standalone function for JWT validation
// This can be used outside of the middleware context
func ValidateSupabaseJWT(tokenString string) (*Claims, error) {
	// Get Supabase URL from environment
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, errors.New("SUPABASE_URL environment variable not set")
	}

	jwksURL := fmt.Sprintf("%s/auth/v1/jwks", strings.TrimSuffix(supabaseURL, "/"))

	// Create a temporary key manager
	km := NewKeyManager(jwksURL, slog.Default())
	if err := km.RefreshKeys(); err != nil {
		return nil, fmt.Errorf("failed to fetch verification keys: %w", err)
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the kid from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Get the verification key
		key, err := km.GetKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get verification key: %w", err)
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, errors.New("token has expired")
	}

	// Validate required fields
	if claims.Sub == "" {
		return nil, errors.New("missing sub claim")
	}

	if claims.Email == "" {
		return nil, errors.New("missing email claim")
	}

	return claims, nil
}

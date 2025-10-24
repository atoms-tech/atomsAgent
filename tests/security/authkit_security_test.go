package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test RSA keys (DO NOT use in production)
var (
	testPrivateKey *rsa.PrivateKey
	testPublicKey  *rsa.PublicKey
	testKid        = "test-key-1"
)

func init() {
	// Generate test RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	testPrivateKey = key
	testPublicKey = &key.PublicKey
}

// mockJWKSServer creates a mock JWKS server for testing
func mockJWKSServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Encode public key components as base64url
		nBytes := testPublicKey.N.Bytes()
		eBytes := big.NewInt(int64(testPublicKey.E)).Bytes()

		n := base64.RawURLEncoding.EncodeToString(nBytes)
		e := base64.RawURLEncoding.EncodeToString(eBytes)

		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kid": testKid,
					"alg": "RS256",
					"kty": "RSA",
					"use": "sig",
					"n":   n,
					"e":   e,
				},
			},
		}

		json.NewEncoder(w).Encode(jwks)
	}))
}

// createTestToken creates a valid test JWT token
func createTestToken(claims *auth.AuthKitClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKid

	return token.SignedString(testPrivateKey)
}

// createValidClaims creates valid test claims
func createValidClaims(userID, orgID, role string) *auth.AuthKitClaims {
	now := time.Now()
	return &auth.AuthKitClaims{
		Sub:           userID,
		Org:           orgID,
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Role:          role,
		Permissions:   []string{"read", "write"},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
}

// setupTestLogger creates a test logger
func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

// ========================================
// 1. Authentication Tests
// ========================================

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuth_InvalidBearerFormat(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	testCases := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "eyJhbGciOiJIUzI1NiJ9.test.sig"},
		{"Wrong prefix", "Basic dXNlcjpwYXNz"},
		{"Multiple spaces", "Bearer  token  here"},
		{"Only Bearer", "Bearer"},
		{"Empty after Bearer", "Bearer "},
		{"Invalid characters", "Bearer token\nwith\nnewlines"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ExtractBearerToken(tc.header)
			assert.Error(t, err, "Should reject: %s", tc.header)
		})
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	// Create expired token
	claims := createValidClaims("user-123", "org-456", "member")
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))

	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	_, err = validator.ValidateToken(context.Background(), tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAuth_MalformedJWT(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	testCases := []struct {
		name  string
		token string
	}{
		{"Not a JWT", "not-a-jwt-token"},
		{"Missing parts", "header.payload"},
		{"Invalid base64", "!!!.!!!.!!!"},
		{"Empty string", ""},
		{"Only dots", "..."},
		{"Random string", "completely-random-string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateToken(context.Background(), tc.token)
			assert.Error(t, err, "Should reject malformed token: %s", tc.name)
		})
	}
}

func TestAuth_ValidToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user context
		userID := r.Context().Value("user_id")
		orgID := r.Context().Value("org_id")
		authkitUser := r.Context().Value("authkit_user")

		assert.Equal(t, "user-123", userID)
		assert.Equal(t, "org-456", orgID)
		assert.NotNil(t, authkitUser)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestAuth_TokenWithMissingClaims(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	testCases := []struct {
		name        string
		modifyClaim func(*auth.AuthKitClaims)
	}{
		{
			name: "Missing sub claim",
			modifyClaim: func(c *auth.AuthKitClaims) {
				c.Sub = ""
			},
		},
		{
			name: "Missing org claim",
			modifyClaim: func(c *auth.AuthKitClaims) {
				c.Org = ""
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := createValidClaims("user-123", "org-456", "member")
			tc.modifyClaim(claims)

			tokenString, err := createTestToken(claims)
			require.NoError(t, err)

			_, err = validator.ValidateToken(context.Background(), tokenString)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing")
		})
	}
}

func TestAuth_CaseInsensitiveBearerPrefix(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	// Test that "Bearer" is case-sensitive (as per spec)
	// The implementation requires exactly "Bearer"
	testCases := []struct {
		name      string
		prefix    string
		shouldErr bool
	}{
		{"Correct Bearer", "Bearer", false},
		{"Lowercase bearer", "bearer", true}, // Should fail
		{"UPPERCASE BEARER", "BEARER", true}, // Should fail
		{"Mixed BeArEr", "BeArEr", true},     // Should fail
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.prefix + " token123"
			_, err := validator.ExtractBearerToken(header)

			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ========================================
// 2. Authorization Tests (Tiered Access)
// ========================================

func TestAuth_PublicEndpointsNoToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	publicEndpoints := []string{"/health", "/ready", "/live"}

	for _, endpoint := range publicEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Public)

			handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			}))

			req := httptest.NewRequest("GET", endpoint, nil)
			// No Authorization header
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "ok", w.Body.String())
		})
	}
}

func TestAuth_AuthenticatedEndpointsWithoutToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_AuthenticatedEndpointsWithValidToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

func TestAuth_AdminEndpointsWithMemberRole(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.AdminOnly)

	// Create token with "member" role
	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestAuth_AdminEndpointsWithAdminRole(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.AdminOnly)

	// Create token with "admin" role
	claims := createValidClaims("user-123", "org-456", "admin")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin access granted"))
	}))

	req := httptest.NewRequest("POST", "/api/v1/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "admin access granted", w.Body.String())
}

func TestAuth_OAuthEndpointsWithMemberToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	// OAuth endpoints require authentication but not admin
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("oauth access granted"))
	}))

	req := httptest.NewRequest("POST", "/api/v1/mcp/oauth", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "oauth access granted", w.Body.String())
}

// ========================================
// 3. User Context Tests
// ========================================

func TestAuth_VerifyUserIDInContext(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	expectedUserID := "user-abc-123"
	claims := createValidClaims(expectedUserID, "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		assert.Equal(t, expectedUserID, userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_VerifyOrgIDInContext(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	expectedOrgID := "org-xyz-789"
	claims := createValidClaims("user-123", expectedOrgID, "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value("org_id")
		assert.Equal(t, expectedOrgID, orgID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_VerifyRoleExtraction(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	testRoles := []string{"admin", "member", "viewer", "custom-role"}

	for _, role := range testRoles {
		t.Run("Role_"+role, func(t *testing.T) {
			claims := createValidClaims("user-123", "org-456", role)
			tokenString, err := createTestToken(claims)
			require.NoError(t, err)

			handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user := r.Context().Value("authkit_user").(*auth.AuthKitUser)
				assert.Equal(t, role, user.Role)
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAuth_VerifyPermissionsExtraction(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	claims.Permissions = []string{"read", "write", "delete", "admin"}
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	user, err := validator.ValidateToken(context.Background(), tokenString)
	require.NoError(t, err)

	assert.Equal(t, 4, len(user.Permissions))
	assert.Contains(t, user.Permissions, "read")
	assert.Contains(t, user.Permissions, "write")
	assert.Contains(t, user.Permissions, "delete")
	assert.Contains(t, user.Permissions, "admin")

	// Test HasPermission method
	assert.True(t, user.HasPermission("read"))
	assert.True(t, user.HasPermission("admin"))
	assert.False(t, user.HasPermission("nonexistent"))
}

// ========================================
// 4. Token Validation Tests
// ========================================

func TestAuth_JWKSKeyLoadingAndCaching(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	// First validation - should load keys
	user1, err := validator.ValidateToken(context.Background(), tokenString)
	require.NoError(t, err)
	assert.Equal(t, "user-123", user1.ID)

	// Second validation - should use cached keys
	user2, err := validator.ValidateToken(context.Background(), tokenString)
	require.NoError(t, err)
	assert.Equal(t, "user-123", user2.ID)
}

func TestAuth_InvalidSignatureDetection(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	// Tamper with the signature
	parts := strings.Split(tokenString, ".")
	require.Len(t, parts, 3)

	// Change the last character of the signature
	tamperedSig := parts[2][:len(parts[2])-1] + "X"
	tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSig

	_, err = validator.ValidateToken(context.Background(), tamperedToken)
	assert.Error(t, err)
}

func TestAuth_KeyIDMismatch(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Set wrong kid
	token.Header["kid"] = "wrong-key-id"

	tokenString, err := token.SignedString(testPrivateKey)
	require.NoError(t, err)

	_, err = validator.ValidateToken(context.Background(), tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestAuth_AlgorithmMismatch(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")

	// Try to use HS256 instead of RS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = testKid

	tokenString, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)

	_, err = validator.ValidateToken(context.Background(), tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signing method")
}

// ========================================
// 5. Input Validation Tests
// ========================================

func TestAuth_XSSAttemptsInBearerToken(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	xssPayloads := []string{
		"Bearer <script>alert('xss')</script>",
		"Bearer javascript:alert(1)",
		"Bearer <img src=x onerror=alert(1)>",
		"Bearer ';DROP TABLE users;--",
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			_, err := validator.ExtractBearerToken(payload)
			// Should either error on extraction or validation
			if err == nil {
				// If extraction succeeds, validation should fail
				parts := strings.Fields(payload)
				if len(parts) == 2 {
					_, err = validator.ValidateToken(context.Background(), parts[1])
					assert.Error(t, err)
				}
			}
		})
	}
}

func TestAuth_SQLInjectionInHeaders(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	sqlPayloads := []string{
		"Bearer ' OR '1'='1",
		"Bearer '; DROP TABLE tokens;--",
		"Bearer admin'--",
		"Bearer ' UNION SELECT * FROM users--",
	}

	for _, payload := range sqlPayloads {
		t.Run(payload, func(t *testing.T) {
			token, err := validator.ExtractBearerToken(payload)
			if err == nil {
				_, err = validator.ValidateToken(context.Background(), token)
				assert.Error(t, err, "SQL injection payload should not validate")
			}
		})
	}
}

func TestAuth_OversizedAuthorizationHeader(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	// Create an oversized token (100KB)
	oversizedToken := "Bearer " + strings.Repeat("A", 100*1024)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", oversizedToken)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should be rejected during validation
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========================================
// 6. Concurrent Access Tests
// ========================================

func TestAuth_MultipleConcurrentRequestsWithDifferentTokens(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	// Create multiple tokens for different users
	numUsers := 10
	tokens := make([]string, numUsers)
	userIDs := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("user-%d", i)
		userIDs[i] = userID
		claims := createValidClaims(userID, "org-456", "member")
		token, err := createTestToken(claims)
		require.NoError(t, err)
		tokens[i] = token
	}

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(userID))
	}))

	// Run concurrent requests
	var wg sync.WaitGroup
	results := make([]string, numUsers)
	errors := make([]error, numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokens[idx])
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				results[idx] = w.Body.String()
			} else {
				errors[idx] = fmt.Errorf("request failed with status %d", w.Code)
			}
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded with correct user context
	for i := 0; i < numUsers; i++ {
		assert.NoError(t, errors[i])
		assert.Equal(t, userIDs[i], results[i])
	}
}

func TestAuth_ConcurrentTokenValidation(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	// Validate the same token concurrently
	numGoroutines := 100
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := validator.ValidateToken(context.Background(), tokenString)
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// All validations should succeed
	for i, err := range errors {
		assert.NoError(t, err, "Validation %d failed", i)
	}
}

// ========================================
// 7. Error Response Tests
// ========================================

func TestAuth_ErrorMessagesDoNotLeakSensitiveInfo(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with invalid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Should not leak internal details
	body := w.Body.String()
	assert.NotContains(t, body, "panic")
	assert.NotContains(t, body, "stack trace")
	assert.NotContains(t, body, "internal error")
	assert.NotContains(t, body, "JWKS")
}

func TestAuth_ConsistentErrorResponseFormat(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	testCases := []struct {
		name   string
		token  string
		status int
	}{
		{"Missing token", "", http.StatusUnauthorized},
		{"Invalid format", "InvalidFormat", http.StatusUnauthorized},
		{"Malformed JWT", "Bearer not.a.jwt", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Code)

			// All errors should return plain text (current implementation)
			// or JSON (if changed) - verify format is consistent
			body := w.Body.String()
			assert.NotEmpty(t, body)
		})
	}
}

func TestAuth_ProperHTTPStatusCodes(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	testCases := []struct {
		name         string
		accessLevel  middleware.AccessLevel
		token        string
		expectedCode int
	}{
		{
			name:         "No token on authenticated endpoint",
			accessLevel:  middleware.Authenticated,
			token:        "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Valid member token on admin endpoint",
			accessLevel:  middleware.AdminOnly,
			token:        "member",
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Valid admin token on admin endpoint",
			accessLevel:  middleware.AdminOnly,
			token:        "admin",
			expectedCode: http.StatusOK,
		},
		{
			name:         "No token on public endpoint",
			accessLevel:  middleware.Public,
			token:        "",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, tc.accessLevel)

			handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)

			if tc.token != "" {
				var role string
				if tc.token == "admin" {
					role = "admin"
				} else {
					role = "member"
				}

				claims := createValidClaims("user-123", "org-456", role)
				tokenString, err := createTestToken(claims)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+tokenString)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

// ========================================
// 8. Advanced Security Tests
// ========================================

func TestAuth_TokenIssuedInFuture(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	// Token issued 1 hour in the future
	claims.IssuedAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))

	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	_, err = validator.ValidateToken(context.Background(), tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestAuth_IsAdminMethod(t *testing.T) {
	testCases := []struct {
		role    string
		isAdmin bool
	}{
		{"admin", true},
		{"member", false},
		{"viewer", false},
		{"", false},
		{"ADMIN", false}, // Case sensitive
		{"administrator", false},
	}

	for _, tc := range testCases {
		t.Run("Role_"+tc.role, func(t *testing.T) {
			user := &auth.AuthKitUser{
				ID:    "user-123",
				OrgID: "org-456",
				Role:  tc.role,
			}

			assert.Equal(t, tc.isAdmin, user.IsAdmin())
		})
	}
}

func TestAuth_TieredAccessMiddleware(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	tieredMiddleware := middleware.NewTieredAccessMiddleware(logger, validator, nil)

	// Create tokens
	memberClaims := createValidClaims("member-123", "org-456", "member")
	memberToken, err := createTestToken(memberClaims)
	require.NoError(t, err)

	adminClaims := createValidClaims("admin-123", "org-456", "admin")
	adminToken, err := createTestToken(adminClaims)
	require.NoError(t, err)

	handler := tieredMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	testCases := []struct {
		name         string
		path         string
		token        string
		expectedCode int
	}{
		{"Public health endpoint no token", "/health", "", http.StatusOK},
		{"Chat endpoint no token", "/v1/chat/completions", "", http.StatusUnauthorized},
		{"Chat endpoint with member token", "/v1/chat/completions", memberToken, http.StatusOK},
		{"MCP admin endpoint with member token", "/api/v1/mcp", memberToken, http.StatusForbidden},
		{"MCP admin endpoint with admin token", "/api/v1/mcp", adminToken, http.StatusOK},
		{"MCP OAuth endpoint with member token", "/api/v1/mcp/oauth", memberToken, http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code, "Path: %s", tc.path)
		})
	}
}

func TestAuth_ContextIsolation(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)
	authMiddleware := middleware.NewAuthKitMiddleware(logger, validator, nil, middleware.Authenticated)

	// Create two different user tokens
	claims1 := createValidClaims("user-1", "org-1", "member")
	token1, err := createTestToken(claims1)
	require.NoError(t, err)

	claims2 := createValidClaims("user-2", "org-2", "member")
	token2, err := createTestToken(claims2)
	require.NoError(t, err)

	handler := authMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		orgID := r.Context().Value("org_id").(string)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s:%s", userID, orgID)
	}))

	// Request 1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Request 2
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	// Verify context isolation
	assert.Equal(t, "user-1:org-1", w1.Body.String())
	assert.Equal(t, "user-2:org-2", w2.Body.String())
}

func TestAuth_EmptyPermissionsArray(t *testing.T) {
	logger := setupTestLogger()
	jwksServer := mockJWKSServer(t)
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	claims := createValidClaims("user-123", "org-456", "member")
	claims.Permissions = []string{} // Empty permissions
	tokenString, err := createTestToken(claims)
	require.NoError(t, err)

	user, err := validator.ValidateToken(context.Background(), tokenString)
	require.NoError(t, err)

	assert.Equal(t, 0, len(user.Permissions))
	assert.False(t, user.HasPermission("any-permission"))
}

func TestAuth_UserStringRepresentation(t *testing.T) {
	user := &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
		Email: "test@example.com",
		Role:  "member",
	}

	str := user.String()

	assert.Contains(t, str, "user-123")
	assert.Contains(t, str, "org-456")
	assert.Contains(t, str, "test@example.com")
	assert.Contains(t, str, "member")

	// Should not contain sensitive token
	assert.NotContains(t, str, "Bearer")
}

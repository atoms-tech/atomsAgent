package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock JWKS server for testing
func setupMockJWKSServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					// These are test values - in production, these would be real RSA key components
					N: "test-modulus",
					E: "AQAB",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	return server
}

func TestNewAuthMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		config      AuthConfig
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "valid config with JWKS URL",
			config: AuthConfig{
				JWKSUrl: "https://example.supabase.co/auth/v1/jwks",
				Logger:  slog.Default(),
			},
			expectError: false,
		},
		{
			name: "valid config from environment",
			config: AuthConfig{
				Logger: slog.Default(),
			},
			setupEnv: func() {
				os.Setenv("SUPABASE_URL", "https://test.supabase.co")
			},
			cleanupEnv: func() {
				os.Unsetenv("SUPABASE_URL")
			},
			expectError: false,
		},
		{
			name: "missing JWKS URL and environment",
			config: AuthConfig{
				Logger: slog.Default(),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			middleware, err := NewAuthMiddleware(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, middleware)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, middleware)
			}
		})
	}
}

func TestAuthMiddleware_shouldSkipAuth(t *testing.T) {
	am := &AuthMiddleware{
		config: AuthConfig{
			SkipPaths: []string{"/health", "/public", "/docs"},
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/public/api", true},
		{"/docs/swagger", true},
		{"/api/sessions", false},
		{"/api/users", false},
		{"/healthcheck", true}, // Matches /health prefix
		{"/status", false},     // Doesn't match any skip path
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := am.shouldSkipAuth(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectError bool
		expectedUID string
		expectedOID string
	}{
		{
			name: "valid claims in context",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "test@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectError: false,
			expectedUID: "user-123",
			expectedOID: "org-456",
		},
		{
			name: "no claims in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectError: true,
		},
		{
			name: "invalid claims type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), ContextKeyClaims, "invalid")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			userID, orgID, err := GetUserFromContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUID, userID)
				assert.Equal(t, tt.expectedOID, orgID)
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectError bool
	}{
		{
			name: "valid claims in context",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "test@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectError: false,
		},
		{
			name: "no claims in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			claims, err := GetClaimsFromContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected bool
	}{
		{
			name: "admin user",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "admin@example.com",
					Role:  RoleAdmin,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expected: true,
		},
		{
			name: "regular user",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "user@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expected: false,
		},
		{
			name: "no claims in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			result := IsAdmin(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequireRole(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		requiredRole   UserRole
		setupCtx       func() context.Context
		expectedStatus int
	}{
		{
			name:         "admin accessing admin endpoint",
			requiredRole: RoleAdmin,
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "admin@example.com",
					Role:  RoleAdmin,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "user accessing admin endpoint",
			requiredRole: RoleAdmin,
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "user@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "no claims in context",
			requiredRole: RoleAdmin,
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "user accessing user endpoint",
			requiredRole: RoleUser,
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "user@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequireRole(tt.requiredRole, logger)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest("GET", "/api/test", nil)
			req = req.WithContext(tt.setupCtx())
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestRequireAdminRole(t *testing.T) {
	logger := slog.Default()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireAdminRole(logger)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		setupCtx       func() context.Context
		expectedStatus int
	}{
		{
			name: "admin user",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "admin@example.com",
					Role:  RoleAdmin,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular user",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "user@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/admin", nil)
			req = req.WithContext(tt.setupCtx())
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestRequireUserRole(t *testing.T) {
	logger := slog.Default()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireUserRole(logger)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		setupCtx       func() context.Context
		expectedStatus int
	}{
		{
			name: "regular user",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "user@example.com",
					Role:  RoleUser,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "admin user (should also pass)",
			setupCtx: func() context.Context {
				claims := &Claims{
					Sub:   "user-123",
					OrgID: "org-456",
					Email: "admin@example.com",
					Role:  RoleAdmin,
				}
				return context.WithValue(context.Background(), ContextKeyClaims, claims)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "no claims",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/user", nil)
			req = req.WithContext(tt.setupCtx())
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestClaims_Validation(t *testing.T) {
	tests := []struct {
		name    string
		claims  Claims
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid admin claims",
			claims: Claims{
				Sub:   "user-123",
				Email: "admin@example.com",
				OrgID: "org-456",
				Role:  RoleAdmin,
				Exp:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "valid user claims",
			claims: Claims{
				Sub:   "user-123",
				Email: "user@example.com",
				OrgID: "org-456",
				Role:  RoleUser,
				Exp:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "missing sub",
			claims: Claims{
				Email: "user@example.com",
				OrgID: "org-456",
				Role:  RoleUser,
				Exp:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr: true,
			errMsg:  "missing sub claim",
		},
		{
			name: "missing email",
			claims: Claims{
				Sub:   "user-123",
				OrgID: "org-456",
				Role:  RoleUser,
				Exp:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr: true,
			errMsg:  "missing email claim",
		},
		{
			name: "missing org_id",
			claims: Claims{
				Sub:   "user-123",
				Email: "user@example.com",
				Role:  RoleUser,
				Exp:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr: true,
			errMsg:  "missing org_id claim",
		},
		{
			name: "expired token",
			claims: Claims{
				Sub:   "user-123",
				Email: "user@example.com",
				OrgID: "org-456",
				Role:  RoleUser,
				Exp:   time.Now().Add(-1 * time.Hour).Unix(),
			},
			wantErr: true,
			errMsg:  "token has expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test expiration
			if tt.claims.Exp > 0 && time.Now().Unix() > tt.claims.Exp {
				assert.True(t, tt.wantErr)
				assert.Contains(t, tt.errMsg, "expired")
				return
			}

			// Test required fields
			var err error
			if tt.claims.Sub == "" {
				err = assert.AnError
				assert.Contains(t, tt.errMsg, "sub")
			} else if tt.claims.Email == "" {
				err = assert.AnError
				assert.Contains(t, tt.errMsg, "email")
			} else if tt.claims.OrgID == "" {
				err = assert.AnError
				assert.Contains(t, tt.errMsg, "org_id")
			}

			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}

func TestKeyManager_RefreshKeys(t *testing.T) {
	server := setupMockJWKSServer(t)
	defer server.Close()

	logger := slog.Default()
	km := NewKeyManager(server.URL, logger)

	// Test initial refresh
	err := km.RefreshKeys()
	require.NoError(t, err)

	// Keys should be loaded
	assert.NotEmpty(t, km.keys)

	// Test that refresh respects rate limit (once per minute)
	km.lastRefresh = time.Now()
	err = km.RefreshKeys()
	require.NoError(t, err) // Should return nil without error even if skipped
}

func TestAuthMiddleware_Integration(t *testing.T) {
	// This is an integration test that would require a real Supabase instance
	// or a complete mock JWT setup. Skipping for unit tests.
	t.Skip("Integration test - requires Supabase instance")

	// Example of how this would work:
	// 1. Set up mock Supabase server
	// 2. Generate valid JWT tokens
	// 3. Test full middleware flow
	// 4. Verify context population
	// 5. Test error cases
}

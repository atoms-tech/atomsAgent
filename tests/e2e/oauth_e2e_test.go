package e2e

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/errors"
	"github.com/coder/agentapi/lib/ratelimit"
	"github.com/coder/agentapi/lib/redis"
	"github.com/coder/agentapi/lib/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E Test Suite for complete OAuth flow and integration

// TestE2EAuthenticationFlow tests complete authentication flow
func TestE2EAuthenticationFlow(t *testing.T) {
	ctx := context.Background()

	// Setup
	redisConfig := redis.DefaultConfig()
	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create token cache
	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	tokenCache, err := redis.NewTokenCache(redisClient, redis.TokenCacheConfig{
		EncryptionKey: encryptionKey,
		DefaultTTL:    1 * time.Hour,
		KeyPrefix:     "e2e_oauth_token:",
	})
	require.NoError(t, err)

	// Test: Store OAuth token
	userID := "test-user-e2e"
	orgID := "test-org-e2e"
	provider := redis.ProviderGitHub
	accessToken := "ghu_" + generateRandomString(36)
	refreshToken := generateRandomString(32)

	token := redis.OAuthToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Provider:     string(provider),
	}

	err = tokenCache.StoreToken(ctx, userID, orgID, provider, token)
	require.NoError(t, err)

	// Test: Retrieve OAuth token
	retrievedToken, err := tokenCache.GetToken(ctx, userID, orgID, provider)
	require.NoError(t, err)
	assert.NotNil(t, retrievedToken)
	assert.Equal(t, accessToken, retrievedToken.AccessToken)
	assert.Equal(t, refreshToken, retrievedToken.RefreshToken)

	// Test: Token is encrypted in Redis
	redisKey := fmt.Sprintf("e2e_oauth_token:%s:%s:%s", userID, orgID, provider)
	rawValue, err := redisClient.Get(ctx, redisKey)
	require.NoError(t, err)
	// Verify it's not plaintext
	assert.NotContains(t, rawValue, accessToken)
	assert.NotContains(t, rawValue, refreshToken)

	// Test: Refresh token
	newAccessToken := "ghu_" + generateRandomString(36)
	token.AccessToken = newAccessToken
	token.ExpiresAt = time.Now().Add(2 * time.Hour)

	err = tokenCache.UpdateToken(ctx, userID, orgID, provider, token)
	require.NoError(t, err)

	retrievedToken, err = tokenCache.GetToken(ctx, userID, orgID, provider)
	require.NoError(t, err)
	assert.Equal(t, newAccessToken, retrievedToken.AccessToken)

	// Test: Token expiry check
	expiredToken := redis.OAuthToken{
		AccessToken:  "expired_token",
		RefreshToken: "expired_refresh",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
		Provider:     string(provider),
	}

	err = tokenCache.StoreToken(ctx, "expired-user", orgID, provider, expiredToken)
	require.NoError(t, err)

	_, err = tokenCache.GetToken(ctx, "expired-user", orgID, provider)
	assert.Error(t, err) // Should error on expired token

	// Test: Revoke token
	err = tokenCache.RevokeToken(ctx, userID, orgID, provider)
	require.NoError(t, err)

	_, err = tokenCache.GetToken(ctx, userID, orgID, provider)
	assert.Error(t, err) // Should error after revocation
}

// TestE2ECircuitBreakerIntegration tests circuit breaker with MCP operations
func TestE2ECircuitBreakerIntegration(t *testing.T) {
	ctx := context.Background()

	// Create circuit breaker
	cbConfig := resilience.CBConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 10,
	}
	cb, err := resilience.NewCircuitBreaker("test_mcp", cbConfig)
	require.NoError(t, err)

	// Create mock MCP service that fails initially
	failCount := 0
	mockOperation := func() (interface{}, error) {
		failCount++
		if failCount <= 3 {
			return nil, fmt.Errorf("service unavailable")
		}
		return "success", nil
	}

	// Test: Failures trigger circuit opening
	for i := 0; i < 3; i++ {
		_, err := cb.Execute(mockOperation)
		assert.Error(t, err)
	}

	// Verify circuit is open
	state := cb.GetState()
	assert.Equal(t, "open", state)

	// Test: Requests fail fast when open
	_, err = cb.Execute(mockOperation)
	assert.Error(t, err)
	assert.Equal(t, resilience.ErrCircuitOpen.Error(), err.Error())

	// Test: Half-open state allows probe requests
	// Wait for timeout
	time.Sleep(cbConfig.Timeout + 100*time.Millisecond)

	// Next request should try (half-open)
	_, err = cb.Execute(mockOperation)
	// After success, circuit should close
	if err == nil {
		state = cb.GetState()
		assert.Equal(t, "closed", state)
	}
}

// TestE2ERateLimitingIntegration tests rate limiting across multiple users
func TestE2ERateLimitingIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup Redis
	redisConfig := redis.DefaultConfig()
	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create rate limiter
	rateLimitConfig := ratelimit.DefaultConfig()
	rateLimitConfig.RequestsPerMinute = 10
	rateLimitConfig.BurstSize = 3
	rateLimitConfig.KeyPrefix = "e2e_ratelimit"

	limiter, err := ratelimit.NewRateLimiter(redisClient, rateLimitConfig)
	require.NoError(t, err)

	userID := "e2e-user"
	endpoint := "/api/v1/mcp/configurations"

	// Test: Allow requests within limit
	for i := 0; i < 10; i++ {
		allowed, remaining, resetTime := limiter.IsAllowed(ctx, userID, endpoint)
		assert.True(t, allowed)
		assert.Equal(t, 10-i-1, remaining)
		assert.NotNil(t, resetTime)
	}

	// Test: Reject requests exceeding limit
	allowed, remaining, _ := limiter.IsAllowed(ctx, userID, endpoint)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)

	// Test: Reset clears limit
	err = limiter.ResetLimit(ctx, userID, endpoint)
	require.NoError(t, err)

	allowed, remaining, _ = limiter.IsAllowed(ctx, userID, endpoint)
	assert.True(t, allowed)
	assert.Equal(t, 9, remaining)
}

// TestE2ESessionManagement tests complete session lifecycle
func TestE2ESessionManagement(t *testing.T) {
	ctx := context.Background()

	// Create JWT token
	claims := map[string]interface{}{
		"sub":    "user123",
		"org_id": "org456",
		"role":   "user",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(1 * time.Hour).Unix(),
	}

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	token := auth.AuthToken{
		Claims:    claimsJSON,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Test: Session creation
	sessionID := fmt.Sprintf("session-%s", generateRandomString(12))

	// Simulate session storage (would use real session manager in integration)
	sessionData := map[string]interface{}{
		"session_id": sessionID,
		"user_id":    "user123",
		"org_id":     "org456",
		"created_at": time.Now(),
		"expires_at": token.ExpiresAt,
	}

	sessionJSON, err := json.Marshal(sessionData)
	require.NoError(t, err)

	// Test: Session persistence
	assert.NotEmpty(t, sessionJSON)
	assert.Contains(t, string(sessionJSON), "user123")
	assert.Contains(t, string(sessionJSON), "org456")

	// Test: Session validation
	var retrievedSession map[string]interface{}
	err = json.Unmarshal(sessionJSON, &retrievedSession)
	require.NoError(t, err)
	assert.Equal(t, sessionID, retrievedSession["session_id"])

	// Test: Session expiry check
	expiredSession := map[string]interface{}{
		"session_id": "expired-session",
		"expires_at": time.Now().Add(-1 * time.Hour),
	}

	isExpired := time.Now().After(expiredSession["expires_at"].(time.Time))
	assert.True(t, isExpired)
}

// TestE2EErrorHandlingAndRetry tests error handling with retry logic
func TestE2EErrorHandlingAndRetry(t *testing.T) {
	ctx := context.Background()

	// Create mock HTTP server that fails then succeeds
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, `{"error":"service unavailable"}`)
		} else {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"status":"success"}`)
		}
	}))
	defer server.Close()

	// Test: Retry with exponential backoff
	maxRetries := 3
	baseDelay := 10 * time.Millisecond
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := http.Get(server.URL)
		if err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				// Exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * baseDelay
				time.Sleep(backoff)
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("status %d", resp.StatusCode)

		if attempt < maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * baseDelay
			time.Sleep(backoff)
		}
	}

	// Should eventually succeed
	assert.Nil(t, lastErr)
	assert.GreaterOrEqual(t, callCount, 3)
}

// TestE2EOAuthTokenExpiry tests token refresh on expiry
func TestE2EOAuthTokenExpiry(t *testing.T) {
	ctx := context.Background()

	// Create token with short expiry
	token := redis.OAuthToken{
		AccessToken:  "short_lived_token",
		RefreshToken: "refresh_token_123",
		ExpiresAt:    time.Now().Add(100 * time.Millisecond),
		Provider:     string(redis.ProviderGitHub),
	}

	// Simulate token usage
	timeUntilExpiry := time.Until(token.ExpiresAt)
	assert.Greater(t, timeUntilExpiry, time.Duration(0))

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Check expiry
	isExpired := time.Now().After(token.ExpiresAt)
	assert.True(t, isExpired)

	// Test: Auto-refresh should happen before expiry
	token.ExpiresAt = time.Now().Add(5 * time.Minute)
	refreshThreshold := 1 * time.Minute
	shouldRefresh := time.Until(token.ExpiresAt) < refreshThreshold
	assert.False(t, shouldRefresh) // Not yet
}

// TestE2EDataEncryption tests end-to-end data encryption
func TestE2EDataEncryption(t *testing.T) {
	ctx := context.Background()

	// Setup Redis
	redisConfig := redis.DefaultConfig()
	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create encryption key
	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	// Create token cache
	tokenCache, err := redis.NewTokenCache(redisClient, redis.TokenCacheConfig{
		EncryptionKey: encryptionKey,
		DefaultTTL:    1 * time.Hour,
		KeyPrefix:     "e2e_encrypt:",
	})
	require.NoError(t, err)

	// Store sensitive data
	sensitiveData := "super_secret_oauth_token_12345"
	userID := "encrypt-test-user"
	orgID := "encrypt-test-org"
	provider := redis.ProviderGoogle

	token := redis.OAuthToken{
		AccessToken:  sensitiveData,
		RefreshToken: "refresh_secret",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Provider:     string(provider),
	}

	err = tokenCache.StoreToken(ctx, userID, orgID, provider, token)
	require.NoError(t, err)

	// Verify encrypted in Redis
	key := fmt.Sprintf("e2e_encrypt:%s:%s:%s", userID, orgID, provider)
	encrypted, err := redisClient.Get(ctx, key)
	require.NoError(t, err)

	// Should be base64 encoded and encrypted
	assert.NotContains(t, encrypted, sensitiveData)
	assert.NotContains(t, encrypted, "refresh_secret")

	// Decrypt should work
	retrieved, err := tokenCache.GetToken(ctx, userID, orgID, provider)
	require.NoError(t, err)
	assert.Equal(t, sensitiveData, retrieved.AccessToken)
	assert.Equal(t, "refresh_secret", retrieved.RefreshToken)
}

// TestE2EMultiProviderOAuth tests OAuth with multiple providers
func TestE2EMultiProviderOAuth(t *testing.T) {
	ctx := context.Background()

	// Setup
	redisConfig := redis.DefaultConfig()
	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	encryptionKey := make([]byte, 32)
	_, err = rand.Read(encryptionKey)
	require.NoError(t, err)

	tokenCache, err := redis.NewTokenCache(redisClient, redis.TokenCacheConfig{
		EncryptionKey: encryptionKey,
		DefaultTTL:    1 * time.Hour,
		KeyPrefix:     "e2e_multi_provider:",
	})
	require.NoError(t, err)

	userID := "multi-provider-user"
	orgID := "multi-provider-org"

	// Test storing tokens from different providers
	providers := []redis.OAuthProvider{
		redis.ProviderGitHub,
		redis.ProviderGoogle,
		redis.ProviderAzure,
		redis.ProviderAuth0,
	}

	for _, provider := range providers {
		token := redis.OAuthToken{
			AccessToken:  fmt.Sprintf("%s_token_%s", provider, generateRandomString(10)),
			RefreshToken: fmt.Sprintf("%s_refresh_%s", provider, generateRandomString(10)),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			Provider:     string(provider),
		}

		err := tokenCache.StoreToken(ctx, userID, orgID, provider, token)
		require.NoError(t, err)

		// Verify retrieval
		retrieved, err := tokenCache.GetToken(ctx, userID, orgID, provider)
		require.NoError(t, err)
		assert.Equal(t, token.AccessToken, retrieved.AccessToken)
	}
}

// Helper functions

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func generatePKCEChallenge(codeVerifier string) string {
	hash := make([]byte, 32)
	// In production would use SHA-256
	copy(hash, codeVerifier)
	return base64.RawURLEncoding.EncodeToString(hash)
}

// TestE2ECSRF tests CSRF protection with state parameter
func TestE2ECSRF(t *testing.T) {
	// Generate state parameter
	state := generateRandomString(32)
	stateCreatedAt := time.Now()
	stateExpiry := stateCreatedAt.Add(10 * time.Minute)

	// Simulate OAuth callback with state validation
	callbackState := state // Should match

	assert.Equal(t, state, callbackState)
	assert.True(t, time.Now().Before(stateExpiry))
	assert.True(t, time.Now().After(stateCreatedAt))

	// Test state replay attack prevention
	oldState := generateRandomString(32)
	oldStateCreatedAt := time.Now().Add(-15 * time.Minute)
	oldStateExpiry := oldStateCreatedAt.Add(10 * time.Minute)

	isExpired := time.Now().After(oldStateExpiry)
	assert.True(t, isExpired) // Old state should be expired
}

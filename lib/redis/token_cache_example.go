package redis

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"
)

// ExampleTokenCacheUsage demonstrates how to use the TokenCache
func ExampleTokenCacheUsage() {
	// 1. Create Redis client
	redisConfig := DefaultConfig()
	redisConfig.URL = os.Getenv("REDIS_URL") // e.g., "rediss://default:password@host:port"

	redisClient, err := NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// 2. Generate or load encryption key (32 bytes for AES-256)
	// IMPORTANT: In production, load this from environment or secrets manager
	encryptionKey := loadEncryptionKey()

	// 3. Create token cache
	tokenCache, err := NewTokenCache(redisClient, TokenCacheConfig{
		EncryptionKey:     encryptionKey,
		DefaultTTL:        1 * time.Hour,
		KeyPrefix:         "oauth_token:",
		EnableAutoRefresh: true,
		RefreshThreshold:  5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create token cache: %v", err)
	}
	defer tokenCache.Close()

	ctx := context.Background()

	// 4. Cache a token
	token := &Token{
		AccessToken:  "ya29.a0AfH6SMB...",
		RefreshToken: "1//0gHxxx...",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Provider:     ProviderGoogle,
		Scope:        "openid email profile",
		TokenType:    "Bearer",
	}

	userID := "user123"
	if err := tokenCache.CacheToken(ctx, userID, ProviderGoogle, token, 0); err != nil {
		log.Fatalf("Failed to cache token: %v", err)
	}
	fmt.Println("Token cached successfully")

	// 5. Retrieve a token
	retrievedToken, err := tokenCache.GetToken(ctx, userID, ProviderGoogle)
	if err != nil {
		log.Fatalf("Failed to retrieve token: %v", err)
	}
	fmt.Printf("Retrieved token: %s (expires at: %s)\n",
		retrievedToken.AccessToken[:20]+"...",
		retrievedToken.ExpiresAt.Format(time.RFC3339))

	// 6. Check if token is expiring soon
	if tokenCache.IsExpiringSoon(retrievedToken) {
		fmt.Println("Token is expiring soon, consider refreshing")

		// Refresh the token with new credentials
		newToken := &Token{
			AccessToken:  "ya29.a0AfH6SMC...",
			RefreshToken: "1//0gHyyy...",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			Provider:     ProviderGoogle,
			Scope:        "openid email profile",
			TokenType:    "Bearer",
		}

		if err := tokenCache.RefreshToken(ctx, userID, ProviderGoogle, newToken); err != nil {
			log.Fatalf("Failed to refresh token: %v", err)
		}
		fmt.Println("Token refreshed successfully")
	}

	// 7. Get all tokens for a user
	allTokens, err := tokenCache.GetAllTokens(ctx, userID)
	if err != nil {
		log.Fatalf("Failed to get all tokens: %v", err)
	}
	fmt.Printf("User has %d tokens cached\n", len(allTokens))
	for provider, tok := range allTokens {
		fmt.Printf("  - %s: expires at %s\n", provider, tok.ExpiresAt.Format(time.RFC3339))
	}

	// 8. Get token statistics
	stats, err := tokenCache.GetStats(ctx, userID)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	fmt.Printf("Token stats: Total=%d, Expired=%d, ExpiringSoon=%d\n",
		stats.TotalTokens, stats.ExpiredTokens, stats.ExpiringSoon)

	// 9. Revoke a token
	if err := tokenCache.RevokeToken(ctx, userID, ProviderGoogle); err != nil {
		log.Fatalf("Failed to revoke token: %v", err)
	}
	fmt.Println("Token revoked successfully")

	// 10. Verify token is gone
	_, err = tokenCache.GetToken(ctx, userID, ProviderGoogle)
	if err == ErrTokenNotFound {
		fmt.Println("Token successfully removed")
	}
}

// loadEncryptionKey loads or generates an encryption key
// In production, this should come from environment variables or a secrets manager
func loadEncryptionKey() []byte {
	keyStr := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if keyStr != "" {
		// Decode base64-encoded key from environment
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err == nil && len(key) == 32 {
			return key
		}
		log.Println("Warning: Invalid encryption key in environment, generating new one")
	}

	// Generate a new random key (for development only)
	// NEVER do this in production - use a consistent key from secure storage
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}

	// In development, you can print this to set as environment variable
	encodedKey := base64.StdEncoding.EncodeToString(key)
	log.Printf("Generated encryption key (set TOKEN_ENCRYPTION_KEY): %s", encodedKey)

	return key
}

// ExampleMultiProviderTokenManagement demonstrates managing tokens for multiple OAuth providers
func ExampleMultiProviderTokenManagement() {
	// Setup (same as above)
	redisClient, _ := NewRedisClient(DefaultConfig())
	defer redisClient.Close()

	tokenCache, _ := NewTokenCache(redisClient, TokenCacheConfig{
		EncryptionKey: loadEncryptionKey(),
		DefaultTTL:    1 * time.Hour,
	})
	defer tokenCache.Close()

	ctx := context.Background()
	userID := "user456"

	// Cache tokens for multiple providers
	providers := []struct {
		provider    OAuthProvider
		accessToken string
		scope       string
	}{
		{ProviderGoogle, "google-access-token-xxx", "email profile"},
		{ProviderGitHub, "github-access-token-xxx", "repo user"},
		{ProviderMicrosoft, "microsoft-access-token-xxx", "User.Read Mail.Read"},
		{ProviderSlack, "slack-access-token-xxx", "channels:read chat:write"},
	}

	for _, p := range providers {
		token := &Token{
			AccessToken: p.accessToken,
			ExpiresAt:   time.Now().Add(1 * time.Hour),
			Provider:    p.provider,
			Scope:       p.scope,
			TokenType:   "Bearer",
		}

		if err := tokenCache.CacheToken(ctx, userID, p.provider, token, 0); err != nil {
			log.Printf("Failed to cache %s token: %v", p.provider, err)
			continue
		}
		fmt.Printf("Cached %s token\n", p.provider)
	}

	// Retrieve all tokens
	allTokens, _ := tokenCache.GetAllTokens(ctx, userID)
	fmt.Printf("\nUser has tokens for %d providers:\n", len(allTokens))
	for provider, token := range allTokens {
		ttl := tokenCache.GetTokenTTL(token)
		fmt.Printf("  %s: TTL=%v, Scope=%s\n", provider, ttl.Round(time.Minute), token.Scope)
	}
}

// ExampleTokenRefreshWorkflow demonstrates a typical token refresh workflow
func ExampleTokenRefreshWorkflow() {
	redisClient, _ := NewRedisClient(DefaultConfig())
	defer redisClient.Close()

	tokenCache, _ := NewTokenCache(redisClient, TokenCacheConfig{
		EncryptionKey:    loadEncryptionKey(),
		RefreshThreshold: 5 * time.Minute,
	})
	defer tokenCache.Close()

	ctx := context.Background()
	userID := "user789"

	// Initial token (expires soon)
	initialToken := &Token{
		AccessToken:  "initial-access-token",
		RefreshToken: "refresh-token-123",
		ExpiresAt:    time.Now().Add(3 * time.Minute), // Expires in 3 minutes
		Provider:     ProviderGoogle,
		Scope:        "openid email",
		TokenType:    "Bearer",
	}

	tokenCache.CacheToken(ctx, userID, ProviderGoogle, initialToken, 0)

	// Check if refresh is needed
	token, _ := tokenCache.GetToken(ctx, userID, ProviderGoogle)
	if tokenCache.IsExpiringSoon(token) {
		fmt.Println("Token is expiring soon, initiating refresh...")

		// Simulate OAuth refresh (in real code, call OAuth provider's refresh endpoint)
		newToken := &Token{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token-456",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			Provider:     ProviderGoogle,
			Scope:        "openid email",
			TokenType:    "Bearer",
		}

		// Update the cached token
		if err := tokenCache.RefreshToken(ctx, userID, ProviderGoogle, newToken); err != nil {
			log.Fatalf("Failed to refresh token: %v", err)
		}

		fmt.Println("Token refreshed successfully")

		// Verify new token
		updatedToken, _ := tokenCache.GetToken(ctx, userID, ProviderGoogle)
		fmt.Printf("New token expires at: %s\n", updatedToken.ExpiresAt.Format(time.RFC3339))
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	redisClient, _ := NewRedisClient(DefaultConfig())
	defer redisClient.Close()

	tokenCache, _ := NewTokenCache(redisClient, TokenCacheConfig{
		EncryptionKey: loadEncryptionKey(),
	})
	defer tokenCache.Close()

	ctx := context.Background()

	// Try to get non-existent token
	_, err := tokenCache.GetToken(ctx, "nonexistent", ProviderGoogle)
	if err == ErrTokenNotFound {
		fmt.Println("Token not found (expected)")
	}

	// Try invalid user ID
	err = tokenCache.CacheToken(ctx, "", ProviderGoogle, &Token{}, 0)
	if err == ErrInvalidUserID {
		fmt.Println("Invalid user ID (expected)")
	}

	// Try invalid provider
	err = tokenCache.CacheToken(ctx, "user123", "", &Token{}, 0)
	if err == ErrInvalidProvider {
		fmt.Println("Invalid provider (expected)")
	}

	// Try nil token
	err = tokenCache.CacheToken(ctx, "user123", ProviderGoogle, nil, 0)
	if err == ErrInvalidToken {
		fmt.Println("Invalid token (expected)")
	}

	// Validate token
	invalidToken := &Token{
		// Missing required fields
	}
	err = tokenCache.ValidateToken(invalidToken)
	if err != nil {
		fmt.Printf("Token validation failed: %v (expected)\n", err)
	}
}

// ExampleTokenCacheHealthCheck demonstrates token cache health checking
func ExampleTokenCacheHealthCheck() {
	redisClient, _ := NewRedisClient(DefaultConfig())
	defer redisClient.Close()

	tokenCache, _ := NewTokenCache(redisClient, TokenCacheConfig{
		EncryptionKey: loadEncryptionKey(),
	})
	defer tokenCache.Close()

	ctx := context.Background()

	// Perform health check
	if err := tokenCache.Health(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("Token cache is healthy")
	}
}

// ExampleTokenValidation demonstrates token validation
func ExampleTokenValidation() {
	tokenCache, _ := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: loadEncryptionKey(),
	})
	defer tokenCache.Close()

	// Valid token
	validToken := &Token{
		AccessToken: "valid-token",
		Provider:    ProviderGoogle,
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		IssuedAt:    time.Now(),
		Scope:       "openid email",
		TokenType:   "Bearer",
	}

	if err := tokenCache.ValidateToken(validToken); err != nil {
		log.Fatalf("Valid token failed validation: %v", err)
	}
	fmt.Println("Token validation passed")

	// Invalid token (expired)
	expiredToken := &Token{
		AccessToken: "expired-token",
		Provider:    ProviderGoogle,
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
		IssuedAt:    time.Now().Add(-2 * time.Hour),
	}

	if err := tokenCache.ValidateToken(expiredToken); err != nil {
		fmt.Printf("Expired token validation failed: %v (expected)\n", err)
	}
}

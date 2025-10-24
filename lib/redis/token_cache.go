package redis

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Token cache errors
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrInvalidToken       = errors.New("invalid token data")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrInvalidProvider    = errors.New("invalid provider")
	ErrInvalidUserID      = errors.New("invalid user ID")
	ErrEncryptionKeyEmpty = errors.New("encryption key is empty")
)

// OAuthProvider represents supported OAuth providers
type OAuthProvider string

const (
	ProviderGoogle    OAuthProvider = "google"
	ProviderGitHub    OAuthProvider = "github"
	ProviderMicrosoft OAuthProvider = "microsoft"
	ProviderSlack     OAuthProvider = "slack"
	ProviderCustom    OAuthProvider = "custom"
)

// Token represents an OAuth token with encrypted credentials
type Token struct {
	// Encrypted fields (stored encrypted in Redis)
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`

	// Metadata fields (not encrypted)
	ExpiresAt time.Time     `json:"expires_at"`
	Provider  OAuthProvider `json:"provider"`
	Scope     string        `json:"scope,omitempty"`
	TokenType string        `json:"token_type,omitempty"`
	IssuedAt  time.Time     `json:"issued_at"`
}

// encryptedToken represents the encrypted token structure stored in Redis
type encryptedToken struct {
	AccessToken  string        `json:"access_token"`            // Base64 encoded encrypted data
	RefreshToken string        `json:"refresh_token,omitempty"` // Base64 encoded encrypted data
	ExpiresAt    time.Time     `json:"expires_at"`
	Provider     OAuthProvider `json:"provider"`
	Scope        string        `json:"scope,omitempty"`
	TokenType    string        `json:"token_type,omitempty"`
	IssuedAt     time.Time     `json:"issued_at"`
}

// TokenCacheConfig holds configuration for the token cache
type TokenCacheConfig struct {
	// EncryptionKey is a 32-byte key for AES-256-GCM encryption
	// Should be loaded from a secure source (environment variable, secrets manager, etc.)
	EncryptionKey []byte

	// DefaultTTL is the default time-to-live for tokens if not specified
	DefaultTTL time.Duration

	// KeyPrefix is the prefix for Redis keys (default: "oauth_token:")
	KeyPrefix string

	// EnableAutoRefresh enables automatic token refresh before expiration
	EnableAutoRefresh bool

	// RefreshThreshold is the duration before expiration to trigger auto-refresh
	RefreshThreshold time.Duration
}

// TokenCache provides encrypted token caching with Redis
type TokenCache struct {
	client *RedisClient
	config TokenCacheConfig

	// AES-GCM cipher for encryption/decryption
	gcm cipher.AEAD

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewTokenCache creates a new token cache with encryption support
func NewTokenCache(client *RedisClient, config TokenCacheConfig) (*TokenCache, error) {
	if client == nil {
		return nil, errors.New("redis client cannot be nil")
	}

	if len(config.EncryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256, got %d bytes", len(config.EncryptionKey))
	}

	if config.DefaultTTL == 0 {
		config.DefaultTTL = 1 * time.Hour
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = "oauth_token:"
	}

	if config.RefreshThreshold == 0 {
		config.RefreshThreshold = 5 * time.Minute
	}

	// Create AES cipher block
	block, err := aes.NewCipher(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	return &TokenCache{
		client: client,
		config: config,
		gcm:    gcm,
	}, nil
}

// CacheToken stores an encrypted OAuth token in Redis
func (tc *TokenCache) CacheToken(ctx context.Context, userID string, provider OAuthProvider, token *Token, ttl time.Duration) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	if provider == "" {
		return ErrInvalidProvider
	}

	if token == nil {
		return ErrInvalidToken
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Set issued timestamp if not already set
	if token.IssuedAt.IsZero() {
		token.IssuedAt = time.Now()
	}

	// Set provider
	token.Provider = provider

	// Encrypt sensitive fields
	encryptedAccess, err := tc.encrypt(token.AccessToken)
	if err != nil {
		return fmt.Errorf("%w: failed to encrypt access token: %v", ErrEncryptionFailed, err)
	}

	var encryptedRefresh string
	if token.RefreshToken != "" {
		encryptedRefresh, err = tc.encrypt(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("%w: failed to encrypt refresh token: %v", ErrEncryptionFailed, err)
		}
	}

	// Create encrypted token structure
	encToken := encryptedToken{
		AccessToken:  encryptedAccess,
		RefreshToken: encryptedRefresh,
		ExpiresAt:    token.ExpiresAt,
		Provider:     token.Provider,
		Scope:        token.Scope,
		TokenType:    token.TokenType,
		IssuedAt:     token.IssuedAt,
	}

	// Serialize to JSON
	data, err := json.Marshal(encToken)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Calculate TTL
	if ttl == 0 {
		if !token.ExpiresAt.IsZero() {
			ttl = time.Until(token.ExpiresAt)
			if ttl < 0 {
				return errors.New("token has already expired")
			}
		} else {
			ttl = tc.config.DefaultTTL
		}
	}

	// Store in Redis with atomic SET operation
	key := tc.buildKey(userID, provider)
	if err := tc.client.Set(ctx, key, string(data), ttl); err != nil {
		return fmt.Errorf("failed to store token in Redis: %w", err)
	}

	return nil
}

// GetToken retrieves and decrypts an OAuth token from Redis
func (tc *TokenCache) GetToken(ctx context.Context, userID string, provider OAuthProvider) (*Token, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	if provider == "" {
		return nil, ErrInvalidProvider
	}

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Retrieve from Redis
	key := tc.buildKey(userID, provider)
	data, err := tc.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token from Redis: %w", err)
	}

	if data == "" {
		return nil, ErrTokenNotFound
	}

	// Deserialize from JSON
	var encToken encryptedToken
	if err := json.Unmarshal([]byte(data), &encToken); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Decrypt sensitive fields
	accessToken, err := tc.decrypt(encToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decrypt access token: %v", ErrDecryptionFailed, err)
	}

	var refreshToken string
	if encToken.RefreshToken != "" {
		refreshToken, err = tc.decrypt(encToken.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decrypt refresh token: %v", ErrDecryptionFailed, err)
		}
	}

	// Create decrypted token
	token := &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    encToken.ExpiresAt,
		Provider:     encToken.Provider,
		Scope:        encToken.Scope,
		TokenType:    encToken.TokenType,
		IssuedAt:     encToken.IssuedAt,
	}

	// Check if token is expired
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		// Token is expired, remove it
		_ = tc.client.Delete(ctx, key)
		return nil, errors.New("token has expired")
	}

	return token, nil
}

// RefreshToken updates an existing token with new credentials (atomic operation)
func (tc *TokenCache) RefreshToken(ctx context.Context, userID string, provider OAuthProvider, newToken *Token) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	if provider == "" {
		return ErrInvalidProvider
	}

	if newToken == nil {
		return ErrInvalidToken
	}

	// Verify token exists before refreshing
	key := tc.buildKey(userID, provider)
	exists, err := tc.client.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check token existence: %w", err)
	}

	if !exists {
		return ErrTokenNotFound
	}

	// Cache the new token (this is atomic via Redis SET)
	ttl := time.Duration(0)
	if !newToken.ExpiresAt.IsZero() {
		ttl = time.Until(newToken.ExpiresAt)
	}

	return tc.CacheToken(ctx, userID, provider, newToken, ttl)
}

// RevokeToken removes a token from the cache
func (tc *TokenCache) RevokeToken(ctx context.Context, userID string, provider OAuthProvider) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	if provider == "" {
		return ErrInvalidProvider
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.buildKey(userID, provider)
	if err := tc.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// GetAllTokens retrieves all tokens for a user across all providers
func (tc *TokenCache) GetAllTokens(ctx context.Context, userID string) (map[string]*Token, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// List of providers to check
	providers := []OAuthProvider{
		ProviderGoogle,
		ProviderGitHub,
		ProviderMicrosoft,
		ProviderSlack,
		ProviderCustom,
	}

	tokens := make(map[string]*Token)

	// Check each provider
	for _, provider := range providers {
		// Temporarily unlock for the get operation
		tc.mu.RUnlock()
		token, err := tc.GetToken(ctx, userID, provider)
		tc.mu.RLock()

		if err == nil && token != nil {
			tokens[string(provider)] = token
		}
		// Ignore errors for non-existent tokens
	}

	return tokens, nil
}

// IsExpiringSoon checks if a token will expire within the refresh threshold
func (tc *TokenCache) IsExpiringSoon(token *Token) bool {
	if token == nil || token.ExpiresAt.IsZero() {
		return false
	}

	return time.Until(token.ExpiresAt) <= tc.config.RefreshThreshold
}

// GetTokenTTL returns the remaining time-to-live for a token
func (tc *TokenCache) GetTokenTTL(token *Token) time.Duration {
	if token == nil || token.ExpiresAt.IsZero() {
		return 0
	}

	ttl := time.Until(token.ExpiresAt)
	if ttl < 0 {
		return 0
	}

	return ttl
}

// encrypt encrypts plaintext using AES-256-GCM
func (tc *TokenCache) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create nonce
	nonce := make([]byte, tc.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := tc.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-256-GCM
func (tc *TokenCache) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Extract nonce
	nonceSize := tc.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]

	// Decrypt and verify
	plaintext, err := tc.gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// buildKey constructs a Redis key for a user's provider token
func (tc *TokenCache) buildKey(userID string, provider OAuthProvider) string {
	// Format: oauth_token:{userID}:{provider}
	return fmt.Sprintf("%s%s:%s", tc.config.KeyPrefix, userID, provider)
}

// parseKey extracts userID and provider from a Redis key
func (tc *TokenCache) parseKey(key string) (userID string, provider OAuthProvider, err error) {
	// Remove prefix
	if !strings.HasPrefix(key, tc.config.KeyPrefix) {
		return "", "", errors.New("invalid key format: missing prefix")
	}

	parts := strings.Split(strings.TrimPrefix(key, tc.config.KeyPrefix), ":")
	if len(parts) != 2 {
		return "", "", errors.New("invalid key format: expected 2 parts")
	}

	return parts[0], OAuthProvider(parts[1]), nil
}

// ValidateToken performs comprehensive token validation
func (tc *TokenCache) ValidateToken(token *Token) error {
	if token == nil {
		return ErrInvalidToken
	}

	if token.AccessToken == "" {
		return errors.New("access token is required")
	}

	if token.Provider == "" {
		return ErrInvalidProvider
	}

	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return errors.New("token has expired")
	}

	if token.IssuedAt.IsZero() {
		return errors.New("issued_at timestamp is required")
	}

	return nil
}

// Health checks the health of the token cache (delegates to Redis client)
func (tc *TokenCache) Health(ctx context.Context) error {
	return tc.client.Health()
}

// Close gracefully shuts down the token cache
func (tc *TokenCache) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Clear the GCM cipher (security best practice)
	tc.gcm = nil

	// Note: We don't close the Redis client as it may be shared
	// The caller should manage the Redis client lifecycle

	return nil
}

// Stats returns statistics about token cache usage
type TokenCacheStats struct {
	TotalTokens      int
	ExpiredTokens    int
	ExpiringSoon     int
	TokensByProvider map[string]int
}

// GetStats returns statistics about cached tokens for a user
// Note: This is an expensive operation and should be used sparingly
func (tc *TokenCache) GetStats(ctx context.Context, userID string) (*TokenCacheStats, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	tokens, err := tc.GetAllTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	stats := &TokenCacheStats{
		TotalTokens:      len(tokens),
		TokensByProvider: make(map[string]int),
	}

	now := time.Now()
	for provider, token := range tokens {
		stats.TokensByProvider[provider]++

		if !token.ExpiresAt.IsZero() {
			if now.After(token.ExpiresAt) {
				stats.ExpiredTokens++
			} else if tc.IsExpiringSoon(token) {
				stats.ExpiringSoon++
			}
		}
	}

	return stats, nil
}

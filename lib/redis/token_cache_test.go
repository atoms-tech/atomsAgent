package redis

import (
	"context"
	"crypto/rand"
	"testing"
	"time"
)

// generateEncryptionKey generates a random 32-byte encryption key for testing
func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

func TestNewTokenCache(t *testing.T) {
	tests := []struct {
		name        string
		client      *RedisClient
		config      TokenCacheConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil client",
			client:      nil,
			config:      TokenCacheConfig{EncryptionKey: generateEncryptionKey()},
			expectError: true,
			errorMsg:    "redis client cannot be nil",
		},
		{
			name:   "invalid encryption key size",
			client: &RedisClient{},
			config: TokenCacheConfig{
				EncryptionKey: []byte("short"),
			},
			expectError: true,
			errorMsg:    "encryption key must be exactly 32 bytes",
		},
		{
			name:   "valid configuration",
			client: &RedisClient{},
			config: TokenCacheConfig{
				EncryptionKey: generateEncryptionKey(),
				DefaultTTL:    1 * time.Hour,
			},
			expectError: false,
		},
		{
			name:   "default values applied",
			client: &RedisClient{},
			config: TokenCacheConfig{
				EncryptionKey: generateEncryptionKey(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewTokenCache(tt.client, tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					if len(err.Error()) < 30 || len(tt.errorMsg) < 30 {
						t.Errorf("expected error message %q but got %q", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cache == nil {
					t.Error("cache should not be nil")
				}

				// Verify defaults
				if cache.config.DefaultTTL == 0 {
					t.Error("DefaultTTL should have a default value")
				}
				if cache.config.KeyPrefix == "" {
					t.Error("KeyPrefix should have a default value")
				}
				if cache.config.RefreshThreshold == 0 {
					t.Error("RefreshThreshold should have a default value")
				}
			}
		})
	}
}

func TestTokenCache_EncryptDecrypt(t *testing.T) {
	key := generateEncryptionKey()
	cache, err := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: key,
	})
	if err != nil {
		t.Fatalf("failed to create token cache: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "test-token-123",
		},
		{
			name:      "complex token",
			plaintext: "ya29.a0AfH6SMBxxx...verylongtoken...xyz",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "special characters",
			plaintext: "token!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := cache.encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Empty strings should return empty
			if tt.plaintext == "" && encrypted != "" {
				t.Error("expected empty encrypted string for empty plaintext")
			}

			// Encrypted should be different from plaintext
			if tt.plaintext != "" && encrypted == tt.plaintext {
				t.Error("encrypted text should differ from plaintext")
			}

			// Decrypt
			decrypted, err := cache.decrypt(encrypted)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// Should match original
			if decrypted != tt.plaintext {
				t.Errorf("expected %q but got %q", tt.plaintext, decrypted)
			}
		})
	}
}

func TestTokenCache_EncryptDecrypt_DifferentNonces(t *testing.T) {
	key := generateEncryptionKey()
	cache, err := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: key,
	})
	if err != nil {
		t.Fatalf("failed to create token cache: %v", err)
	}

	plaintext := "test-token"

	// Encrypt same plaintext twice
	encrypted1, err := cache.encrypt(plaintext)
	if err != nil {
		t.Fatalf("first encryption failed: %v", err)
	}

	encrypted2, err := cache.encrypt(plaintext)
	if err != nil {
		t.Fatalf("second encryption failed: %v", err)
	}

	// Should produce different ciphertexts (due to different nonces)
	if encrypted1 == encrypted2 {
		t.Error("encrypting same plaintext twice should produce different ciphertexts")
	}

	// Both should decrypt to same plaintext
	decrypted1, _ := cache.decrypt(encrypted1)
	decrypted2, _ := cache.decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("both encrypted values should decrypt to original plaintext")
	}
}

func TestTokenCache_BuildKey(t *testing.T) {
	cache := &TokenCache{
		config: TokenCacheConfig{
			KeyPrefix:     "oauth_token:",
			EncryptionKey: generateEncryptionKey(),
		},
	}

	tests := []struct {
		name     string
		userID   string
		provider OAuthProvider
		expected string
	}{
		{
			name:     "google provider",
			userID:   "user123",
			provider: ProviderGoogle,
			expected: "oauth_token:user123:google",
		},
		{
			name:     "github provider",
			userID:   "user456",
			provider: ProviderGitHub,
			expected: "oauth_token:user456:github",
		},
		{
			name:     "custom provider",
			userID:   "user789",
			provider: ProviderCustom,
			expected: "oauth_token:user789:custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cache.buildKey(tt.userID, tt.provider)
			if key != tt.expected {
				t.Errorf("expected key %q but got %q", tt.expected, key)
			}
		})
	}
}

func TestTokenCache_ParseKey(t *testing.T) {
	cache := &TokenCache{
		config: TokenCacheConfig{
			KeyPrefix:     "oauth_token:",
			EncryptionKey: generateEncryptionKey(),
		},
	}

	tests := []struct {
		name             string
		key              string
		expectedUserID   string
		expectedProvider OAuthProvider
		expectError      bool
	}{
		{
			name:             "valid key",
			key:              "oauth_token:user123:google",
			expectedUserID:   "user123",
			expectedProvider: ProviderGoogle,
			expectError:      false,
		},
		{
			name:        "invalid prefix",
			key:         "wrong_prefix:user123:google",
			expectError: true,
		},
		{
			name:        "missing parts",
			key:         "oauth_token:user123",
			expectError: true,
		},
		{
			name:        "too many parts",
			key:         "oauth_token:user123:google:extra",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, provider, err := cache.parseKey(tt.key)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if userID != tt.expectedUserID {
					t.Errorf("expected userID %q but got %q", tt.expectedUserID, userID)
				}
				if provider != tt.expectedProvider {
					t.Errorf("expected provider %q but got %q", tt.expectedProvider, provider)
				}
			}
		})
	}
}

func TestTokenCache_ValidateToken(t *testing.T) {
	cache := &TokenCache{
		config: TokenCacheConfig{
			EncryptionKey: generateEncryptionKey(),
		},
	}

	now := time.Now()

	tests := []struct {
		name        string
		token       *Token
		expectError bool
	}{
		{
			name:        "nil token",
			token:       nil,
			expectError: true,
		},
		{
			name: "missing access token",
			token: &Token{
				Provider: ProviderGoogle,
				IssuedAt: now,
			},
			expectError: true,
		},
		{
			name: "missing provider",
			token: &Token{
				AccessToken: "token123",
				IssuedAt:    now,
			},
			expectError: true,
		},
		{
			name: "expired token",
			token: &Token{
				AccessToken: "token123",
				Provider:    ProviderGoogle,
				ExpiresAt:   now.Add(-1 * time.Hour),
				IssuedAt:    now.Add(-2 * time.Hour),
			},
			expectError: true,
		},
		{
			name: "missing issued at",
			token: &Token{
				AccessToken: "token123",
				Provider:    ProviderGoogle,
				ExpiresAt:   now.Add(1 * time.Hour),
			},
			expectError: true,
		},
		{
			name: "valid token",
			token: &Token{
				AccessToken: "token123",
				Provider:    ProviderGoogle,
				ExpiresAt:   now.Add(1 * time.Hour),
				IssuedAt:    now,
			},
			expectError: false,
		},
		{
			name: "valid token without expiry",
			token: &Token{
				AccessToken: "token123",
				Provider:    ProviderGoogle,
				IssuedAt:    now,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.ValidateToken(tt.token)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTokenCache_IsExpiringSoon(t *testing.T) {
	cache := &TokenCache{
		config: TokenCacheConfig{
			EncryptionKey:    generateEncryptionKey(),
			RefreshThreshold: 5 * time.Minute,
		},
	}

	now := time.Now()

	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name:     "nil token",
			token:    nil,
			expected: false,
		},
		{
			name: "no expiry",
			token: &Token{
				AccessToken: "token123",
			},
			expected: false,
		},
		{
			name: "expires in 10 minutes",
			token: &Token{
				AccessToken: "token123",
				ExpiresAt:   now.Add(10 * time.Minute),
			},
			expected: false,
		},
		{
			name: "expires in 3 minutes",
			token: &Token{
				AccessToken: "token123",
				ExpiresAt:   now.Add(3 * time.Minute),
			},
			expected: true,
		},
		{
			name: "already expired",
			token: &Token{
				AccessToken: "token123",
				ExpiresAt:   now.Add(-1 * time.Minute),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.IsExpiringSoon(tt.token)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestTokenCache_GetTokenTTL(t *testing.T) {
	cache := &TokenCache{
		config: TokenCacheConfig{
			EncryptionKey: generateEncryptionKey(),
		},
	}

	now := time.Now()

	tests := []struct {
		name   string
		token  *Token
		minTTL time.Duration
		maxTTL time.Duration
	}{
		{
			name:   "nil token",
			token:  nil,
			minTTL: 0,
			maxTTL: 0,
		},
		{
			name: "no expiry",
			token: &Token{
				AccessToken: "token123",
			},
			minTTL: 0,
			maxTTL: 0,
		},
		{
			name: "expires in 1 hour",
			token: &Token{
				AccessToken: "token123",
				ExpiresAt:   now.Add(1 * time.Hour),
			},
			minTTL: 59 * time.Minute, // Allow some time variance
			maxTTL: 61 * time.Minute,
		},
		{
			name: "already expired",
			token: &Token{
				AccessToken: "token123",
				ExpiresAt:   now.Add(-1 * time.Hour),
			},
			minTTL: 0,
			maxTTL: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := cache.GetTokenTTL(tt.token)

			if ttl < tt.minTTL || ttl > tt.maxTTL {
				t.Errorf("expected TTL between %v and %v but got %v", tt.minTTL, tt.maxTTL, ttl)
			}
		})
	}
}

func TestTokenCache_CacheToken_Validation(t *testing.T) {
	cache, err := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: generateEncryptionKey(),
	})
	if err != nil {
		t.Fatalf("failed to create token cache: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name        string
		userID      string
		provider    OAuthProvider
		token       *Token
		expectError bool
	}{
		{
			name:        "empty user ID",
			userID:      "",
			provider:    ProviderGoogle,
			token:       &Token{AccessToken: "token123"},
			expectError: true,
		},
		{
			name:        "empty provider",
			userID:      "user123",
			provider:    "",
			token:       &Token{AccessToken: "token123"},
			expectError: true,
		},
		{
			name:        "nil token",
			userID:      "user123",
			provider:    ProviderGoogle,
			token:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.CacheToken(ctx, tt.userID, tt.provider, tt.token, time.Hour)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	// Test expired token
	t.Run("expired token", func(t *testing.T) {
		expiredToken := &Token{
			AccessToken: "token123",
			ExpiresAt:   now.Add(-1 * time.Hour),
			IssuedAt:    now.Add(-2 * time.Hour),
		}

		err := cache.CacheToken(ctx, "user123", ProviderGoogle, expiredToken, 0)
		if err == nil || err.Error() != "token has already expired" {
			t.Errorf("expected 'token has already expired' error but got: %v", err)
		}
	})
}

func TestOAuthProvider_Constants(t *testing.T) {
	// Verify provider constants are set correctly
	if ProviderGoogle != "google" {
		t.Errorf("ProviderGoogle should be 'google' but got %q", ProviderGoogle)
	}
	if ProviderGitHub != "github" {
		t.Errorf("ProviderGitHub should be 'github' but got %q", ProviderGitHub)
	}
	if ProviderMicrosoft != "microsoft" {
		t.Errorf("ProviderMicrosoft should be 'microsoft' but got %q", ProviderMicrosoft)
	}
	if ProviderSlack != "slack" {
		t.Errorf("ProviderSlack should be 'slack' but got %q", ProviderSlack)
	}
	if ProviderCustom != "custom" {
		t.Errorf("ProviderCustom should be 'custom' but got %q", ProviderCustom)
	}
}

func TestTokenCache_Close(t *testing.T) {
	cache, err := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: generateEncryptionKey(),
	})
	if err != nil {
		t.Fatalf("failed to create token cache: %v", err)
	}

	if err := cache.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify GCM was cleared
	if cache.gcm != nil {
		t.Error("GCM should be nil after Close()")
	}
}

// Benchmark tests
func BenchmarkTokenCache_Encrypt(b *testing.B) {
	cache, _ := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: generateEncryptionKey(),
	})

	plaintext := "ya29.a0AfH6SMBxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.encrypt(plaintext)
	}
}

func BenchmarkTokenCache_Decrypt(b *testing.B) {
	cache, _ := NewTokenCache(&RedisClient{}, TokenCacheConfig{
		EncryptionKey: generateEncryptionKey(),
	})

	plaintext := "ya29.a0AfH6SMBxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	ciphertext, _ := cache.encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.decrypt(ciphertext)
	}
}

# Redis Token Cache

Production-ready OAuth token cache with AES-256-GCM encryption for secure token storage in Redis.

## Overview

The `TokenCache` provides a secure, encrypted storage solution for OAuth tokens in Redis. It uses AES-256-GCM authenticated encryption to protect sensitive token data at rest, while supporting multiple OAuth providers and automatic token expiration.

## Features

- **AES-256-GCM Encryption**: All access and refresh tokens are encrypted using authenticated encryption
- **Multi-Provider Support**: Built-in support for Google, GitHub, Microsoft, Slack, and custom OAuth providers
- **Automatic Expiration**: TTL-based token expiration aligned with OAuth token lifetime
- **Atomic Operations**: Thread-safe operations with proper locking
- **Token Refresh**: Safe token refresh with atomic updates
- **Statistics**: Token usage statistics and health monitoring
- **Error Recovery**: Comprehensive error handling and validation

## Installation

The token cache is part of the `lib/redis` package:

```go
import "github.com/coder/agentapi/lib/redis"
```

## Quick Start

### 1. Setup

```go
package main

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "log"
    "os"
    "time"

    "github.com/coder/agentapi/lib/redis"
)

func main() {
    // Create Redis client
    redisConfig := redis.DefaultConfig()
    redisConfig.URL = os.Getenv("REDIS_URL")

    redisClient, err := redis.NewRedisClient(redisConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer redisClient.Close()

    // Generate encryption key (32 bytes for AES-256)
    encryptionKey := generateEncryptionKey()

    // Create token cache
    tokenCache, err := redis.NewTokenCache(redisClient, redis.TokenCacheConfig{
        EncryptionKey:     encryptionKey,
        DefaultTTL:        1 * time.Hour,
        KeyPrefix:         "oauth_token:",
        EnableAutoRefresh: true,
        RefreshThreshold:  5 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer tokenCache.Close()

    // Use the cache...
}

func generateEncryptionKey() []byte {
    // Load from environment in production
    keyStr := os.Getenv("TOKEN_ENCRYPTION_KEY")
    if keyStr != "" {
        key, _ := base64.StdEncoding.DecodeString(keyStr)
        return key
    }

    // Generate for development
    key := make([]byte, 32)
    rand.Read(key)
    return key
}
```

### 2. Cache a Token

```go
ctx := context.Background()

token := &redis.Token{
    AccessToken:  "ya29.a0AfH6SMB...",
    RefreshToken: "1//0gHxxx...",
    ExpiresAt:    time.Now().Add(1 * time.Hour),
    Provider:     redis.ProviderGoogle,
    Scope:        "openid email profile",
    TokenType:    "Bearer",
}

err := tokenCache.CacheToken(ctx, "user123", redis.ProviderGoogle, token, 0)
if err != nil {
    log.Fatal(err)
}
```

### 3. Retrieve a Token

```go
token, err := tokenCache.GetToken(ctx, "user123", redis.ProviderGoogle)
if err == redis.ErrTokenNotFound {
    // Token doesn't exist or has expired
    log.Println("Token not found")
    return
}
if err != nil {
    log.Fatal(err)
}

// Use the decrypted token
fmt.Println("Access token:", token.AccessToken)
```

### 4. Refresh a Token

```go
// Check if token needs refresh
if tokenCache.IsExpiringSoon(token) {
    // Call OAuth provider to refresh
    newToken := &redis.Token{
        AccessToken:  "ya29.a0AfH6SMC...",
        RefreshToken: "1//0gHyyy...",
        ExpiresAt:    time.Now().Add(1 * time.Hour),
        Provider:     redis.ProviderGoogle,
        Scope:        "openid email profile",
        TokenType:    "Bearer",
    }

    err := tokenCache.RefreshToken(ctx, "user123", redis.ProviderGoogle, newToken)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 5. Revoke a Token

```go
err := tokenCache.RevokeToken(ctx, "user123", redis.ProviderGoogle)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### TokenCacheConfig

```go
type TokenCacheConfig struct {
    // EncryptionKey is a 32-byte key for AES-256-GCM encryption (required)
    EncryptionKey []byte

    // DefaultTTL for tokens when not specified (default: 1 hour)
    DefaultTTL time.Duration

    // KeyPrefix for Redis keys (default: "oauth_token:")
    KeyPrefix string

    // EnableAutoRefresh enables automatic token refresh (default: false)
    EnableAutoRefresh bool

    // RefreshThreshold is duration before expiration to trigger auto-refresh (default: 5 minutes)
    RefreshThreshold time.Duration
}
```

### Encryption Key Management

**IMPORTANT**: The encryption key must be:
- Exactly 32 bytes (256 bits) for AES-256
- Securely generated using cryptographic random number generator
- Stored securely (environment variables, secrets manager, etc.)
- Consistent across application restarts (to decrypt existing tokens)

Example encryption key generation:

```bash
# Generate a random key
openssl rand -base64 32

# Set as environment variable
export TOKEN_ENCRYPTION_KEY="your-base64-encoded-key"
```

## API Reference

### Methods

#### CacheToken
```go
func (tc *TokenCache) CacheToken(ctx context.Context, userID string, provider OAuthProvider, token *Token, ttl time.Duration) error
```
Stores an encrypted OAuth token in Redis.

**Parameters:**
- `ctx`: Context for cancellation and deadlines
- `userID`: Unique identifier for the user
- `provider`: OAuth provider (Google, GitHub, etc.)
- `token`: Token to cache
- `ttl`: Time-to-live (0 = use token expiry or default)

#### GetToken
```go
func (tc *TokenCache) GetToken(ctx context.Context, userID string, provider OAuthProvider) (*Token, error)
```
Retrieves and decrypts an OAuth token from Redis.

**Returns:**
- Decrypted token
- `ErrTokenNotFound` if token doesn't exist or has expired

#### RefreshToken
```go
func (tc *TokenCache) RefreshToken(ctx context.Context, userID string, provider OAuthProvider, newToken *Token) error
```
Updates an existing token with new credentials (atomic operation).

#### RevokeToken
```go
func (tc *TokenCache) RevokeToken(ctx context.Context, userID string, provider OAuthProvider) error
```
Removes a token from the cache.

#### GetAllTokens
```go
func (tc *TokenCache) GetAllTokens(ctx context.Context, userID string) (map[string]*Token, error)
```
Retrieves all cached tokens for a user across all providers.

**Returns:**
- Map of provider name to token

#### ValidateToken
```go
func (tc *TokenCache) ValidateToken(token *Token) error
```
Validates token structure and expiration.

#### IsExpiringSoon
```go
func (tc *TokenCache) IsExpiringSoon(token *Token) bool
```
Checks if token will expire within the refresh threshold.

#### GetTokenTTL
```go
func (tc *TokenCache) GetTokenTTL(token *Token) time.Duration
```
Returns the remaining time-to-live for a token.

#### GetStats
```go
func (tc *TokenCache) GetStats(ctx context.Context, userID string) (*TokenCacheStats, error)
```
Returns statistics about cached tokens for a user.

#### Health
```go
func (tc *TokenCache) Health(ctx context.Context) error
```
Performs a health check on the token cache.

### Types

#### Token
```go
type Token struct {
    AccessToken  string        // Encrypted access token
    RefreshToken string        // Encrypted refresh token (optional)
    ExpiresAt    time.Time     // Token expiration timestamp
    Provider     OAuthProvider // OAuth provider
    Scope        string        // OAuth scopes (optional)
    TokenType    string        // Token type (e.g., "Bearer")
    IssuedAt     time.Time     // Token issue timestamp
}
```

#### OAuthProvider
```go
type OAuthProvider string

const (
    ProviderGoogle    OAuthProvider = "google"
    ProviderGitHub    OAuthProvider = "github"
    ProviderMicrosoft OAuthProvider = "microsoft"
    ProviderSlack     OAuthProvider = "slack"
    ProviderCustom    OAuthProvider = "custom"
)
```

## Error Handling

Common errors:

```go
var (
    ErrTokenNotFound      = errors.New("token not found")
    ErrInvalidToken       = errors.New("invalid token data")
    ErrEncryptionFailed   = errors.New("encryption failed")
    ErrDecryptionFailed   = errors.New("decryption failed")
    ErrInvalidProvider    = errors.New("invalid provider")
    ErrInvalidUserID      = errors.New("invalid user ID")
    ErrEncryptionKeyEmpty = errors.New("encryption key is empty")
)
```

Example error handling:

```go
token, err := tokenCache.GetToken(ctx, userID, provider)
if err == redis.ErrTokenNotFound {
    // Token doesn't exist - redirect to OAuth flow
    return redirectToOAuth(provider)
}
if err != nil {
    // Other error - log and return
    log.Printf("Failed to get token: %v", err)
    return err
}

// Validate token
if err := tokenCache.ValidateToken(token); err != nil {
    // Token is invalid - refresh it
    return refreshToken(userID, provider)
}

// Token is valid - use it
return useToken(token)
```

## Redis Key Format

Tokens are stored in Redis with the following key format:

```
oauth_token:{userID}:{provider}
```

Examples:
- `oauth_token:user123:google`
- `oauth_token:user456:github`
- `oauth_token:user789:microsoft`

## Security Considerations

1. **Encryption Key Security**
   - Never hardcode encryption keys in source code
   - Use environment variables or secrets management systems
   - Rotate keys periodically (requires re-encryption of existing tokens)
   - Use different keys for different environments (dev, staging, prod)

2. **Token Storage**
   - Access and refresh tokens are encrypted with AES-256-GCM
   - Metadata (expiry, provider, scope) is stored in plaintext
   - Keys use a consistent prefix for easy identification

3. **Network Security**
   - Always use TLS/SSL for Redis connections (rediss://)
   - Enable Redis authentication
   - Use private networks for Redis access

4. **Expiration**
   - Tokens automatically expire based on TTL
   - Expired tokens are automatically removed on access
   - Use IsExpiringSoon() to proactively refresh tokens

## Performance

- **Encryption**: ~5-10µs per token (benchmarked)
- **Decryption**: ~5-10µs per token (benchmarked)
- **Cache Hit**: Single Redis GET operation
- **Cache Miss**: Returns immediately
- **Atomic Operations**: All updates are atomic via Redis

## Testing

Run the test suite:

```bash
# Run all token cache tests
go test ./lib/redis -v -run TestTokenCache

# Run specific test
go test ./lib/redis -v -run TestTokenCache_EncryptDecrypt

# Run benchmarks
go test ./lib/redis -bench=BenchmarkTokenCache -benchmem
```

## Examples

See `token_cache_example.go` for complete examples:

- Basic usage
- Multi-provider token management
- Token refresh workflow
- Error handling
- Health checking
- Token validation

## Migration Guide

### From Plain Text Storage

If migrating from plain text token storage:

1. Set up encryption key in environment
2. Create TokenCache instance
3. Read existing tokens from database/cache
4. Encrypt and store using CacheToken()
5. Update application code to use GetToken()
6. Remove old storage system

### Encryption Key Rotation

To rotate encryption keys:

1. Create new TokenCache with new key
2. For each user:
   - Retrieve token with old cache
   - Store token with new cache
3. Update environment variable
4. Restart application
5. Old tokens will fail to decrypt (users will re-authenticate)

## Troubleshooting

### Token Decryption Fails

**Cause**: Encryption key mismatch or corrupted data

**Solution**:
- Verify encryption key is correct
- Check key hasn't changed since token was encrypted
- Revoke and re-issue token

### Tokens Expire Too Quickly

**Cause**: TTL misconfiguration

**Solution**:
- Use token's ExpiresAt for TTL calculation
- Increase DefaultTTL if tokens don't have expiry
- Check RefreshThreshold for auto-refresh

### Redis Connection Issues

**Cause**: Network or authentication problems

**Solution**:
- Verify Redis URL is correct
- Check Redis is running and accessible
- Verify authentication credentials
- Use Health() method to test connection

## License

See main project LICENSE file.

## Support

For issues and questions:
- GitHub Issues: [github.com/coder/agentapi/issues]
- Documentation: See main README.md

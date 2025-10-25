# Static API Key Configuration Guide

## Overview

Static API key support allows both ChatServer and the frontend to authenticate with each other using a simple shared API key, without requiring JWT tokens. This is ideal for development, testing, and simple deployments.

## Quick Start

### 1. Configure ChatServer (.env)

Add these environment variables to `/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/.env`:

```bash
# Static API Key Configuration
STATIC_API_KEY=test-api-key-development
STATIC_API_USER_ID=static-test-user
STATIC_API_ORG_ID=static-test-org
STATIC_API_EMAIL=test@agentapi.local
STATIC_API_NAME=Test User
```

### 2. Configure Frontend (.env.local)

Add this to `/Users/kooshapari/temp-PRODVERCEL/485/clean/deploy/atoms.tech/.env.local`:

```bash
NEXT_PUBLIC_STATIC_API_KEY=test-api-key-development
```

**Important**: The `NEXT_PUBLIC_STATIC_API_KEY` value **must match** the `STATIC_API_KEY` value in ChatServer's `.env` file.

### 3. Test the Connection

```bash
# Test static API key validation on ChatServer
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer test-api-key-development" \
  -H "Content-Type: application/json" \
  -d '{"model":"ccrouter","messages":[{"role":"user","content":"Hello"}]}'
```

## How It Works

### ChatServer Authentication Flow

When a request arrives at ChatServer with an API key:

```
Request → Authorization header extracted
    ↓
Key validator receives "test-api-key-development"
    ↓
Check priority order:
  1. Try static API key validation (env var)
  2. Try database API key validation
  3. Try JWT token validation
    ↓
Static key validated successfully
    ↓
Return AuthKitUser with auth method="static_api_key"
    ↓
Request processed with authenticated context
```

### Frontend Authentication Flow

When the frontend makes a request to ChatServer:

```
AgentAPIClient initialized
    ↓
Check for static API key:
  - If NEXT_PUBLIC_STATIC_API_KEY env var exists
  - Use it for Authorization header
    ↓
Otherwise:
  - Try getToken() callback for JWT
  - Fall back to no auth
    ↓
Request sent with "Authorization: Bearer {key}"
```

## Code Changes

### ChatServer (lib/auth/apikey.go)

New method added to validate static API keys:

```go
// ValidateStaticAPIKey validates a static API key configured via environment variable
func (v *APIKeyValidator) ValidateStaticAPIKey(ctx context.Context, providedKey string) (*AuthKitUser, error) {
    // Get static key from env
    staticKey := os.Getenv("STATIC_API_KEY")

    // Compare with provided key
    if providedKey != staticKey {
        return nil, fmt.Errorf("invalid API key")
    }

    // Return authenticated user
    return &AuthKitUser{
        ID:                   os.Getenv("STATIC_API_USER_ID"),
        OrgID:                os.Getenv("STATIC_API_ORG_ID"),
        Email:                os.Getenv("STATIC_API_EMAIL"),
        Name:                 os.Getenv("STATIC_API_NAME"),
        IsPlatformAdminFlag:  true,
        AuthenticationMethod: "static_api_key",
    }, nil
}
```

### ChatServer (lib/auth/authkit.go)

Updated ValidateToken() to check static keys first:

```go
func (av *AuthKitValidator) ValidateToken(ctx context.Context, tokenString string) (*AuthKitUser, error) {
    // Try static API key validation first
    if av.apiKeyValidator != nil {
        user, err := av.apiKeyValidator.ValidateStaticAPIKey(ctx, tokenString)
        if err == nil {
            return user, nil
        }
    }

    // Try database API key validation
    if av.apiKeyValidator != nil {
        user, err := av.apiKeyValidator.ValidateAPIKey(ctx, tokenString)
        if err == nil {
            return user, nil
        }
    }

    // Finally, try JWT token validation
    return av.validateWorkOSToken(ctx, tokenString, unverifiedClaims)
}
```

### Frontend (src/lib/api/agentapi.ts)

Enhanced AgentAPIClient with static key support:

```typescript
constructor(config: AgentAPIConfig = {}) {
    // Priority: provided apiKey > static API key from env > getToken function
    if (config.apiKey) {
        this.apiKey = config.apiKey;
    } else if (config.useStaticApiKey && process.env.NEXT_PUBLIC_STATIC_API_KEY) {
        this.apiKey = process.env.NEXT_PUBLIC_STATIC_API_KEY;
    }
    this.getToken = config.getToken;
}

private async getAuthHeader(): Promise<string | null> {
    // Use static API key if available
    if (this.apiKey) {
        return `Bearer ${this.apiKey}`;
    }
    // Fall back to getToken callback for JWT
    if (this.getToken) {
        const token = await this.getToken();
        return token ? `Bearer ${token}` : null;
    }
    return null;
}
```

### Frontend (src/lib/services/agentapi.ts)

Updated service to detect and use static keys:

```typescript
private constructor() {
    // Check if static API key is configured
    this.useStaticKey = !!process.env.NEXT_PUBLIC_STATIC_API_KEY;

    this.client = new AgentAPIClient({
        baseURL: AGENTAPI_URL,
        useStaticApiKey: this.useStaticKey,
        getToken: async () => {
            // Skip getToken if using static API key
            if (this.useStaticKey) {
                return null;
            }
            if (this.tokenGetter) {
                return this.tokenGetter();
            }
            return null;
        },
    });
}
```

## Testing

### Unit Tests (ChatServer)

Test static API key validation:

```bash
# Test valid static key
go test -v ./lib/auth -run TestValidateStaticAPIKey
```

### Integration Tests

```bash
# Test end-to-end authentication
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer test-api-key-development" \
  -H "Content-Type: application/json" \
  -d '{
    "model":"ccrouter",
    "messages":[
      {"role":"user","content":"Hello, what can you do?"}
    ]
  }'
```

## Security Considerations

### ✅ DO

- Use static API keys **only for development and testing**
- Change the key regularly in development
- Use strong, random keys (32+ characters) in any shared environment
- Store keys in `.env` files (not in version control)
- Log which authentication method was used (for debugging)
- Keep static keys separate from production deployments

### ❌ DON'T

- Use static API keys in production (use JWT tokens instead)
- Commit `.env` files to version control
- Use simple/predictable keys
- Share keys across different projects
- Log or expose the actual key value

## Production Setup

For production deployments:

1. **Remove static API key configuration** from `.env`
2. **Use JWT tokens** via WorkOS/AuthKit
3. **Use environment-specific secrets** for any needed keys
4. **Implement key rotation** for long-lived keys
5. **Monitor and audit** API usage

## Troubleshooting

### Issue: "invalid or expired API key"

**Cause**: Key mismatch between frontend and backend

**Solution**: Ensure `NEXT_PUBLIC_STATIC_API_KEY` matches `STATIC_API_KEY`:

```bash
# ChatServer
echo $STATIC_API_KEY

# Frontend
echo $NEXT_PUBLIC_STATIC_API_KEY

# They should match exactly
```

### Issue: Static key not being used

**Cause**: Environment variable not loaded

**Solution**: Restart both ChatServer and frontend development servers after setting env vars:

```bash
# Kill processes
pkill -f chatserver
pkill -f "next dev"

# Clear build cache
rm -rf .next out

# Restart with new env vars
./chatserver
npm run dev  # or yarn dev, bun dev
```

### Issue: Falling back to JWT instead of static key

**Cause**: Static key config not detected

**Solution**: Check that `NEXT_PUBLIC_STATIC_API_KEY` is properly set:

```typescript
// In browser console
console.log(process.env.NEXT_PUBLIC_STATIC_API_KEY)

// Should print the key value, not undefined
```

## Summary

Static API key support provides a simple development authentication mechanism that:

- ✅ Works out of the box with environment variables
- ✅ Requires no database configuration
- ✅ Falls back gracefully to JWT tokens
- ✅ Prioritizes static keys for testing
- ✅ Maintains backward compatibility

**Key takeaway**: Use for development/testing only, switch to JWT for production.


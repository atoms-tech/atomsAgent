# OAuth Implementation Guide

Complete guide for implementing and deploying the OAuth token exchange system for MCP authentication.

## Architecture Overview

The OAuth implementation consists of four main components:

1. **Initiate Endpoint** (`/api/mcp/oauth/initiate.ts`) - Starts OAuth flow
2. **Callback Handler** (`/api/mcp/oauth/callback.ts`) - Handles provider callbacks
3. **Token Refresh** (`/api/mcp/oauth/refresh.ts`) - Refreshes expired tokens
4. **Token Revocation** (`/api/mcp/oauth/revoke.ts`) - Revokes and deletes tokens
5. **Helper Functions** (`/api/mcp/oauth/helpers.ts`) - Shared utilities

## Security Features

### 1. PKCE (Proof Key for Code Exchange)

Prevents authorization code interception attacks:

```
Client                          Authorization Server
  |                                     |
  |-- (1) Generate code_verifier ------>|
  |                                     |
  |<- (2) Create code_challenge ------->|
  |                                     |
  |-- (3) Auth request + challenge ---->|
  |                                     |
  |<- (4) Authorization code -----------|
  |                                     |
  |-- (5) Token request + verifier ---->|
  |                                     |
  |<- (6) Verify & issue tokens --------|
```

**Implementation:**
- Code verifier: 43-128 character random string
- Code challenge: SHA256(code_verifier) in base64url
- Challenge sent during authorization
- Verifier used during token exchange

### 2. CSRF Protection

State parameter prevents cross-site request forgery:

```typescript
// Generate random state
const state = crypto.randomBytes(32).toString('base64url');

// Store in database with user context
await storeOAuthState(userId, provider, mcpName, state, ...);

// Include in authorization URL
const authUrl = `${provider_auth_url}?state=${state}&...`;

// Validate on callback
const storedState = await getOAuthState(state);
if (!storedState || storedState.user_id !== userId) {
  throw new Error('Invalid state');
}
```

### 3. Token Encryption

AES-256-GCM encryption for stored tokens:

```typescript
// Encryption format: <iv>:<auth_tag>:<encrypted_data>
function encrypt(text: string): string {
  const iv = crypto.randomBytes(16);
  const key = Buffer.from(ENCRYPTION_KEY, 'hex');
  const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);

  let encrypted = cipher.update(text, 'utf8', 'hex');
  encrypted += cipher.final('hex');

  const authTag = cipher.getAuthTag();

  return iv.toString('hex') + ':' +
         authTag.toString('hex') + ':' +
         encrypted;
}
```

**Key Management:**
- Generate: `openssl rand -hex 32`
- Store securely in environment variables
- Never commit to version control
- Rotate periodically for security

### 4. Row Level Security (RLS)

Database-level access control:

```sql
-- Users can only access their own tokens
CREATE POLICY "oauth_tokens_user_policy" ON mcp_oauth_tokens
  FOR ALL USING (user_id = auth.uid());

-- Users can only access their own OAuth states
CREATE POLICY "oauth_states_user_policy" ON oauth_states
  FOR ALL USING (user_id = auth.uid());
```

## Database Setup

### 1. Run Migrations

```bash
# Connect to Supabase
psql "postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres"

# Run base schema
\i database/schema.sql

# Run OAuth tables migration
\i database/migrations/002_oauth_tables.sql
```

### 2. Verify Tables

```sql
-- Check oauth_states table
SELECT * FROM oauth_states LIMIT 1;

-- Check mcp_oauth_tokens table
SELECT * FROM mcp_oauth_tokens LIMIT 1;

-- Verify RLS policies
SELECT schemaname, tablename, policyname
FROM pg_policies
WHERE tablename IN ('oauth_states', 'mcp_oauth_tokens');
```

### 3. Setup Cleanup Jobs

```sql
-- Create cleanup function (already in migration)
SELECT cleanup_expired_oauth_states();
SELECT cleanup_expired_oauth_tokens();

-- Schedule with pg_cron (if available)
SELECT cron.schedule(
  'cleanup-oauth-states',
  '*/5 * * * *', -- Every 5 minutes
  'SELECT cleanup_expired_oauth_states();'
);

SELECT cron.schedule(
  'cleanup-oauth-tokens',
  '0 0 * * *', -- Daily at midnight
  'SELECT cleanup_expired_oauth_tokens();'
);
```

## Provider Configuration

### GitHub OAuth

1. **Create OAuth App:**
   - Go to https://github.com/settings/developers
   - Click "New OAuth App"
   - Set Application name
   - Set Homepage URL: `https://your-app.com`
   - Set Authorization callback URL: `https://your-app.com/api/mcp/oauth/callback`

2. **Configure Environment:**
   ```bash
   GITHUB_CLIENT_ID=your_github_client_id
   GITHUB_CLIENT_SECRET=your_github_client_secret
   ```

3. **Scopes:**
   - `repo`: Full control of private repositories
   - `user`: Read user profile data
   - `read:org`: Read organization membership

### Google OAuth

1. **Create OAuth Client:**
   - Go to https://console.cloud.google.com/apis/credentials
   - Click "Create Credentials" → "OAuth client ID"
   - Application type: "Web application"
   - Add authorized redirect URIs: `https://your-app.com/api/mcp/oauth/callback`

2. **Configure Environment:**
   ```bash
   GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your_client_secret
   ```

3. **Scopes:**
   - `openid`: OpenID Connect
   - `email`: Email address
   - `profile`: Basic profile info

### Azure AD OAuth

1. **Register Application:**
   - Go to https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/RegisteredApps
   - Click "New registration"
   - Set name and supported account types
   - Add redirect URI: `https://your-app.com/api/mcp/oauth/callback`

2. **Create Client Secret:**
   - Go to "Certificates & secrets"
   - Click "New client secret"
   - Copy the secret value

3. **Configure Environment:**
   ```bash
   AZURE_CLIENT_ID=your_application_id
   AZURE_CLIENT_SECRET=your_client_secret
   AZURE_TENANT_ID=your_tenant_id_or_common
   AZURE_TOKEN_ENDPOINT=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
   AZURE_AUTH_ENDPOINT=https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
   ```

4. **Scopes:**
   - `openid`: OpenID Connect
   - `email`: Email address
   - `profile`: Basic profile info
   - `offline_access`: Refresh tokens

### Auth0 OAuth

1. **Create Application:**
   - Go to https://manage.auth0.com/dashboard
   - Click "Applications" → "Create Application"
   - Select "Regular Web Application"
   - Go to "Settings" tab

2. **Configure URLs:**
   - Allowed Callback URLs: `https://your-app.com/api/mcp/oauth/callback`
   - Allowed Web Origins: `https://your-app.com`

3. **Configure Environment:**
   ```bash
   AUTH0_CLIENT_ID=your_client_id
   AUTH0_CLIENT_SECRET=your_client_secret
   AUTH0_DOMAIN=your-tenant.auth0.com
   AUTH0_TOKEN_ENDPOINT=https://your-tenant.auth0.com/oauth/token
   AUTH0_AUTH_ENDPOINT=https://your-tenant.auth0.com/authorize
   AUTH0_AUDIENCE=https://your-api-identifier
   ```

## Deployment

### 1. Install Dependencies

```bash
cd api
npm install

# Or with pnpm
pnpm install
```

### 2. Set Environment Variables

**Local Development (.env.local):**
```bash
cp .env.example .env.local
# Edit .env.local with your values
```

**Vercel Production:**
```bash
# Via CLI
vercel env add OAUTH_ENCRYPTION_KEY
vercel env add SUPABASE_URL
vercel env add SUPABASE_SERVICE_ROLE_KEY
# ... add all variables

# Or via Dashboard
# Go to Project Settings → Environment Variables
```

### 3. Generate Encryption Key

```bash
# Generate a secure 256-bit key
openssl rand -hex 32

# Example output:
# a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456

# Set as environment variable
export OAUTH_ENCRYPTION_KEY="a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
```

### 4. Deploy to Vercel

```bash
# Deploy to production
vercel --prod

# Or via GitHub integration
git push origin main
```

### 5. Verify Deployment

```bash
# Test initiate endpoint
curl -X POST https://your-app.vercel.app/api/mcp/oauth/initiate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"github","mcp_name":"test","scopes":["repo"]}'

# Should return authorization URL
```

## Frontend Integration

### React Hook Example

```typescript
// hooks/useOAuth.ts
import { useState, useCallback } from 'react';
import { useSupabase } from './useSupabase';

export function useOAuth() {
  const { user, session } = useSupabase();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const initiateOAuth = useCallback(
    async (provider: string, mcpName: string, scopes?: string[]) => {
      if (!session?.access_token) {
        throw new Error('Not authenticated');
      }

      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/mcp/oauth/initiate', {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${session.access_token}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            provider,
            mcp_name: mcpName,
            scopes,
          }),
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.message || 'Failed to initiate OAuth');
        }

        const { authorization_url } = await response.json();

        // Open in popup
        const popup = window.open(
          authorization_url,
          'oauth',
          'width=600,height=700,left=200,top=100'
        );

        // Listen for success message
        return new Promise((resolve, reject) => {
          const handleMessage = (event: MessageEvent) => {
            if (event.origin !== window.location.origin) return;

            if (event.data.type === 'oauth_success') {
              window.removeEventListener('message', handleMessage);
              popup?.close();
              resolve(event.data);
            } else if (event.data.type === 'oauth_error') {
              window.removeEventListener('message', handleMessage);
              popup?.close();
              reject(new Error(event.data.error));
            }
          };

          window.addEventListener('message', handleMessage);

          // Timeout after 5 minutes
          setTimeout(() => {
            window.removeEventListener('message', handleMessage);
            popup?.close();
            reject(new Error('OAuth timeout'));
          }, 5 * 60 * 1000);
        });
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        setError(errorMessage);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [session]
  );

  return { initiateOAuth, loading, error };
}
```

### Success Page Component

```typescript
// pages/oauth/success.tsx
import { useEffect } from 'react';
import { useRouter } from 'next/router';

export default function OAuthSuccess() {
  const router = useRouter();
  const { mcp, provider } = router.query;

  useEffect(() => {
    // Send message to parent window
    if (window.opener) {
      window.opener.postMessage(
        {
          type: 'oauth_success',
          mcp,
          provider,
        },
        window.location.origin
      );

      // Close window after short delay
      setTimeout(() => {
        window.close();
      }, 2000);
    }
  }, [mcp, provider]);

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="text-center">
        <h1 className="text-2xl font-bold mb-4">Authentication Successful!</h1>
        <p className="text-gray-600 mb-2">
          Successfully connected to {provider}
        </p>
        <p className="text-gray-600">
          MCP: {mcp}
        </p>
        <p className="text-sm text-gray-500 mt-4">
          You can close this window.
        </p>
      </div>
    </div>
  );
}
```

### Usage in Component

```typescript
// components/MCPConnectButton.tsx
import { useOAuth } from '@/hooks/useOAuth';
import { useState } from 'react';

export function MCPConnectButton({
  provider,
  mcpName
}: {
  provider: string;
  mcpName: string;
}) {
  const { initiateOAuth, loading } = useOAuth();
  const [connected, setConnected] = useState(false);

  const handleConnect = async () => {
    try {
      await initiateOAuth(provider, mcpName, ['repo', 'user']);
      setConnected(true);
      alert(`Successfully connected ${mcpName}!`);
    } catch (err) {
      console.error('OAuth failed:', err);
      alert('Failed to connect. Please try again.');
    }
  };

  return (
    <button
      onClick={handleConnect}
      disabled={loading || connected}
      className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
    >
      {loading ? 'Connecting...' : connected ? 'Connected' : `Connect ${provider}`}
    </button>
  );
}
```

## Testing

### Unit Tests

```typescript
// __tests__/oauth.test.ts
import { encrypt, decrypt } from '../api/mcp/oauth/helpers';

describe('OAuth Encryption', () => {
  beforeAll(() => {
    process.env.OAUTH_ENCRYPTION_KEY = 'a'.repeat(64);
  });

  it('should encrypt and decrypt correctly', () => {
    const plaintext = 'test_access_token_12345';
    const encrypted = encrypt(plaintext);
    const decrypted = decrypt(encrypted);

    expect(decrypted).toBe(plaintext);
  });

  it('should produce different ciphertext each time', () => {
    const plaintext = 'test_token';
    const encrypted1 = encrypt(plaintext);
    const encrypted2 = encrypt(plaintext);

    expect(encrypted1).not.toBe(encrypted2);
  });
});
```

### Integration Tests

```bash
# Test OAuth initiation
curl -X POST http://localhost:3000/api/mcp/oauth/initiate \
  -H "Authorization: Bearer ${TEST_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "github",
    "mcp_name": "test-mcp",
    "scopes": ["repo"]
  }'

# Verify response contains authorization_url
```

## Monitoring

### Audit Logs Query

```sql
-- Recent OAuth events
SELECT
  action,
  details->>'provider' as provider,
  details->>'mcp_name' as mcp_name,
  created_at
FROM audit_logs
WHERE resource_type = 'oauth_token'
  AND created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- Failed OAuth attempts
SELECT
  user_id,
  action,
  details,
  created_at
FROM audit_logs
WHERE action LIKE '%oauth%failed%'
ORDER BY created_at DESC;
```

### Metrics to Track

1. **OAuth Success Rate**
   ```sql
   SELECT
     COUNT(*) FILTER (WHERE action = 'oauth_token_exchanged') as success,
     COUNT(*) FILTER (WHERE action = 'oauth_token_exchange_failed') as failed
   FROM audit_logs
   WHERE created_at > NOW() - INTERVAL '24 hours';
   ```

2. **Token Refresh Rate**
   ```sql
   SELECT
     COUNT(*) as refresh_count
   FROM audit_logs
   WHERE action = 'oauth_token_refreshed'
     AND created_at > NOW() - INTERVAL '24 hours';
   ```

3. **Active OAuth Connections**
   ```sql
   SELECT
     provider,
     COUNT(*) as connection_count
   FROM mcp_oauth_tokens
   WHERE expires_at > NOW()
   GROUP BY provider;
   ```

## Troubleshooting

### Common Issues

1. **"Invalid encryption key" Error**
   ```
   Solution: Ensure OAUTH_ENCRYPTION_KEY is exactly 64 hex characters
   Generate: openssl rand -hex 32
   ```

2. **"State parameter expired" Error**
   ```
   Solution: User must complete OAuth within 5 minutes
   Check: SELECT * FROM oauth_states WHERE state = 'STATE_VALUE';
   ```

3. **"Token exchange failed" Error**
   ```
   Solutions:
   - Verify provider credentials (client ID/secret)
   - Check redirect URI matches exactly
   - Ensure PKCE is enabled for provider
   - Check provider-specific requirements
   ```

4. **RLS Policy Denial**
   ```
   Solution: Ensure user is authenticated and JWT is valid
   Check: SELECT auth.uid(); -- Should return user ID
   ```

### Debug Mode

```typescript
// Enable debug logging in helpers.ts
const DEBUG = process.env.DEBUG === 'true';

function log(...args: any[]) {
  if (DEBUG) {
    console.log('[OAuth Debug]', ...args);
  }
}
```

Set environment variable:
```bash
DEBUG=true
```

## Security Best Practices

1. **Never log sensitive data**
   ```typescript
   // Bad
   console.log('Token:', accessToken);

   // Good
   console.log('Token received:', accessToken ? 'YES' : 'NO');
   ```

2. **Always use HTTPS in production**
   ```typescript
   const isProduction = process.env.NODE_ENV === 'production';
   const protocol = isProduction ? 'https' : 'http';
   ```

3. **Rotate encryption keys periodically**
   ```bash
   # Generate new key
   NEW_KEY=$(openssl rand -hex 32)

   # Update environment
   vercel env add OAUTH_ENCRYPTION_KEY_NEW production

   # Deploy migration script to re-encrypt tokens
   ```

4. **Monitor for suspicious activity**
   ```sql
   -- Multiple failed attempts from same user
   SELECT
     user_id,
     COUNT(*) as failed_attempts
   FROM audit_logs
   WHERE action = 'oauth_token_exchange_failed'
     AND created_at > NOW() - INTERVAL '1 hour'
   GROUP BY user_id
   HAVING COUNT(*) > 3;
   ```

## Support

For issues or questions:
- Check the README.md in `/api/mcp/oauth/`
- Review audit logs for error details
- Contact: support@example.com

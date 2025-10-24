# MCP OAuth Authentication Endpoints

This directory contains Vercel Edge Functions for handling OAuth authentication flows for MCP (Model Context Protocol) integrations.

## Overview

The OAuth flow enables users to securely connect their MCP configurations with third-party OAuth providers (GitHub, Google, Azure, Auth0).

## Endpoints

### POST /api/mcp/oauth/init

Initializes an OAuth authentication flow for an MCP configuration.

#### Request Headers

```
Authorization: Bearer <user-jwt-token>
Content-Type: application/json
```

#### Request Body

```json
{
  "mcp_name": "github",
  "provider": "github",
  "redirect_uri": "https://yourapp.com/oauth/callback"
}
```

**Parameters:**

- `mcp_name` (string, required): The name/identifier of the MCP configuration
- `provider` (string, required): OAuth provider type. Supported values:
  - `github` - GitHub OAuth
  - `google` - Google OAuth
  - `azure` - Microsoft Azure AD
  - `auth0` - Auth0
- `redirect_uri` (string, required): The URL where the user will be redirected after OAuth consent. Must be a valid absolute URL.

#### Response

**Success (200 OK):**

```json
{
  "success": true,
  "auth_url": "https://github.com/login/oauth/authorize?client_id=...&redirect_uri=...&state=...&code_challenge=..."
}
```

**Error (400 Bad Request):**

```json
{
  "success": false,
  "error": "Invalid or missing provider"
}
```

**Error (401 Unauthorized):**

```json
{
  "success": false,
  "error": "Invalid or expired token"
}
```

**Error (500 Internal Server Error):**

```json
{
  "success": false,
  "error": "Failed to initialize OAuth flow. Please try again."
}
```

## Security Features

### PKCE (Proof Key for Code Exchange)

The endpoint implements PKCE (RFC 7636) for enhanced security:

1. **Code Verifier**: A cryptographically random string (128 characters) is generated
2. **Code Challenge**: SHA256 hash of the code verifier, base64url-encoded
3. The code verifier is stored securely in the database
4. The code challenge is sent to the OAuth provider
5. During token exchange, the original verifier is used to prove the request came from the same client

### CSRF Protection

1. **State Parameter**: A random 64-character hex string is generated
2. Stored in the `oauth_state` table with the user ID and expiration time
3. Included in the authorization URL
4. Verified during the OAuth callback to prevent CSRF attacks

### Database Security

OAuth state is stored in the `oauth_state` table:

```sql
CREATE TABLE oauth_state (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    state TEXT NOT NULL UNIQUE,
    code_verifier TEXT NOT NULL,
    provider TEXT NOT NULL,
    mcp_name TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP,
    expires_at TIMESTAMP,
    used BOOLEAN DEFAULT false
);
```

- States expire after 10 minutes
- Row-Level Security (RLS) ensures users can only access their own states
- States are marked as "used" after successful token exchange to prevent replay attacks

## Environment Variables

The following environment variables must be configured:

### Supabase

```env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key
```

### GitHub OAuth

```env
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

**Scopes requested**: `read:user`, `user:email`

**Authorization endpoint**: `https://github.com/login/oauth/authorize`

**Token endpoint**: `https://github.com/login/oauth/access_token`

### Google OAuth

```env
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret
```

**Scopes requested**: `openid`, `email`, `profile`

**Authorization endpoint**: `https://accounts.google.com/o/oauth2/v2/auth`

**Token endpoint**: `https://oauth2.googleapis.com/token`

**Additional parameters**: `access_type=offline`, `prompt=consent` (for refresh token)

### Azure OAuth

```env
AZURE_CLIENT_ID=your-azure-client-id
AZURE_CLIENT_SECRET=your-azure-client-secret
AZURE_TENANT_ID=common
```

**Scopes requested**: `openid`, `email`, `profile`

**Authorization endpoint**: `https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize`

**Token endpoint**: `https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token`

### Auth0 OAuth

```env
AUTH0_CLIENT_ID=your-auth0-client-id
AUTH0_CLIENT_SECRET=your-auth0-client-secret
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_AUDIENCE=https://your-api-identifier
```

**Scopes requested**: `openid`, `email`, `profile`

**Authorization endpoint**: `https://{domain}/authorize`

**Token endpoint**: `https://{domain}/oauth/token`

## Usage Example

### Frontend Implementation

```typescript
async function initiateMCPOAuth(mcpName: string, provider: string) {
  const redirectUri = `${window.location.origin}/oauth/callback`;

  const response = await fetch('/api/mcp/oauth/init', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userToken}`,
    },
    body: JSON.stringify({
      mcp_name: mcpName,
      provider: provider,
      redirect_uri: redirectUri,
    }),
  });

  const data = await response.json();

  if (data.success) {
    // Open OAuth popup or redirect
    window.location.href = data.auth_url;
  } else {
    console.error('OAuth initialization failed:', data.error);
  }
}
```

### OAuth Callback Flow

After the user authorizes the application:

1. OAuth provider redirects to `redirect_uri` with `code` and `state` parameters
2. Frontend sends `code` and `state` to `/api/mcp/oauth/callback` endpoint
3. Callback endpoint:
   - Verifies the `state` matches stored value
   - Exchanges `code` for access token using stored `code_verifier`
   - Stores encrypted access token in database
   - Returns success to frontend

## Error Handling

The endpoint implements comprehensive error handling:

1. **Input Validation**: Validates all required fields and formats
2. **Provider Validation**: Ensures provider is supported and configured
3. **Authentication**: Verifies user JWT token
4. **Database Errors**: Catches and logs database insertion failures
5. **Generic Errors**: Returns safe error messages to clients while logging details

## Testing

### Manual Testing

```bash
# Set environment variables
export SUPABASE_URL="https://your-project.supabase.co"
export SUPABASE_SERVICE_ROLE_KEY="your-key"
export GITHUB_CLIENT_ID="your-client-id"
export GITHUB_CLIENT_SECRET="your-client-secret"

# Test with curl
curl -X POST http://localhost:3000/api/mcp/oauth/init \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "mcp_name": "my-github-mcp",
    "provider": "github",
    "redirect_uri": "http://localhost:3000/oauth/callback"
  }'
```

Expected response:
```json
{
  "success": true,
  "auth_url": "https://github.com/login/oauth/authorize?client_id=..."
}
```

## Deployment

### Vercel Deployment

1. Install dependencies:
   ```bash
   npm install
   ```

2. Set environment variables in Vercel dashboard or `.env.local`

3. Deploy:
   ```bash
   vercel --prod
   ```

### Environment Setup

Ensure all required environment variables are set in your Vercel project settings:

- Navigate to Project Settings > Environment Variables
- Add all variables from `.env.example`
- Set appropriate values for each environment (Development, Preview, Production)

## Database Migration

Run the database migration to create the `oauth_state` table:

```sql
-- See database/schema.sql for the complete migration
CREATE TABLE oauth_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL UNIQUE,
    code_verifier TEXT NOT NULL,
    provider TEXT NOT NULL,
    mcp_name TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT false
);

-- Indexes
CREATE INDEX idx_oauth_state_state ON oauth_state(state);
CREATE INDEX idx_oauth_state_user_id ON oauth_state(user_id);
CREATE INDEX idx_oauth_state_expires_at ON oauth_state(expires_at);

-- Row-Level Security
ALTER TABLE oauth_state ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can manage their own OAuth states" ON oauth_state
    FOR ALL USING (user_id = auth.uid());
```

## Cleanup

Consider implementing a cleanup job to remove expired OAuth states:

```sql
-- Delete expired states older than 1 hour
DELETE FROM oauth_state
WHERE expires_at < NOW() - INTERVAL '1 hour';
```

This can be run as a periodic job (e.g., via pg_cron or an external scheduler).

## Next Steps

1. Implement the OAuth callback endpoint (`/api/mcp/oauth/callback`)
2. Implement token refresh logic
3. Add token encryption before storage
4. Implement token revocation endpoint
5. Add comprehensive logging and monitoring

## References

- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [PKCE RFC 7636](https://tools.ietf.org/html/rfc7636)
- [Vercel Edge Functions](https://vercel.com/docs/functions/edge-functions)
- [Supabase Auth](https://supabase.com/docs/guides/auth)

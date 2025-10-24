# OAuth Setup Guide for MCP Authentication

This guide explains how to set up and configure OAuth authentication for MCP (Model Context Protocol) integrations in AgentAPI.

## Overview

The OAuth authentication system allows users to securely connect their MCP configurations with third-party services like GitHub, Google, Azure, and Auth0. The implementation uses:

- **PKCE (Proof Key for Code Exchange)** for enhanced security
- **CSRF protection** via state parameters
- **Encrypted token storage** in Supabase
- **Row-Level Security (RLS)** for multi-tenant isolation

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌──────────────┐
│   Frontend  │─────▶│  Edge API    │─────▶│   Supabase   │
│             │      │  /api/mcp/   │      │   Database   │
│             │      │  oauth/init  │      │              │
└─────────────┘      └──────────────┘      └──────────────┘
       │                     │
       │                     ▼
       │             Generate PKCE & State
       │                     │
       ▼                     ▼
┌─────────────┐      ┌──────────────┐
│   OAuth     │◀─────│  Return      │
│   Provider  │      │  auth_url    │
│   (GitHub,  │      │              │
│   Google)   │      └──────────────┘
└─────────────┘
       │
       │ User authorizes
       │
       ▼
┌─────────────┐      ┌──────────────┐
│  Frontend   │─────▶│  Edge API    │
│  Callback   │      │  /api/mcp/   │
│             │      │  oauth/      │
│             │      │  callback    │
└─────────────┘      └──────────────┘
                             │
                             ▼
                     Exchange code for token
                             │
                             ▼
                     Store encrypted token
```

## Quick Start

### 1. Prerequisites

- Node.js 18+
- Supabase account and project
- OAuth provider applications set up (see below)

### 2. Installation

```bash
# Install dependencies
npm install

# Copy environment template
cp .env.example .env
```

### 3. Database Setup

Run the database migration to create the required tables:

```bash
# Apply the schema to your Supabase project
psql $SUPABASE_DB_URL < database/schema.sql
```

This creates:
- `oauth_state` table for CSRF protection and PKCE
- Indexes for performance
- Row-Level Security policies

### 4. Configure Environment Variables

Edit `.env` with your credentials:

```env
# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# GitHub OAuth (optional)
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

# Google OAuth (optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Azure OAuth (optional)
AZURE_CLIENT_ID=your-azure-client-id
AZURE_CLIENT_SECRET=your-azure-client-secret
AZURE_TENANT_ID=common

# Auth0 OAuth (optional)
AUTH0_CLIENT_ID=your-auth0-client-id
AUTH0_CLIENT_SECRET=your-auth0-client-secret
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_AUDIENCE=https://your-api-identifier
```

### 5. Deploy

```bash
# Deploy to Vercel
vercel --prod

# Or run locally
vercel dev
```

## OAuth Provider Setup

### GitHub OAuth

1. Go to GitHub Settings > Developer settings > OAuth Apps
2. Click "New OAuth App"
3. Fill in the details:
   - **Application name**: Your app name
   - **Homepage URL**: `https://yourapp.com`
   - **Authorization callback URL**: `https://yourapp.com/oauth/callback`
4. Save and copy the Client ID and Client Secret
5. Add to `.env`:
   ```env
   GITHUB_CLIENT_ID=your_client_id
   GITHUB_CLIENT_SECRET=your_client_secret
   ```

**Scopes requested**: `read:user`, `user:email`

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google+ API
4. Go to Credentials > Create Credentials > OAuth Client ID
5. Configure OAuth consent screen
6. Choose "Web application" as application type
7. Add authorized redirect URIs: `https://yourapp.com/oauth/callback`
8. Copy Client ID and Client Secret
9. Add to `.env`:
   ```env
   GOOGLE_CLIENT_ID=your_client_id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your_client_secret
   ```

**Scopes requested**: `openid`, `email`, `profile`

### Azure OAuth

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to Azure Active Directory > App registrations
3. Click "New registration"
4. Fill in the details:
   - **Name**: Your app name
   - **Supported account types**: Choose appropriate option
   - **Redirect URI**: `https://yourapp.com/oauth/callback`
5. Go to "Certificates & secrets" > Create new client secret
6. Copy Application (client) ID, Directory (tenant) ID, and Client Secret
7. Add to `.env`:
   ```env
   AZURE_CLIENT_ID=your_client_id
   AZURE_CLIENT_SECRET=your_client_secret
   AZURE_TENANT_ID=your_tenant_id
   ```

**Scopes requested**: `openid`, `email`, `profile`

### Auth0 OAuth

1. Go to [Auth0 Dashboard](https://manage.auth0.com/)
2. Create a new application
3. Choose "Regular Web Application"
4. Go to Settings
5. Add allowed callback URLs: `https://yourapp.com/oauth/callback`
6. Copy Domain, Client ID, and Client Secret
7. Add to `.env`:
   ```env
   AUTH0_CLIENT_ID=your_client_id
   AUTH0_CLIENT_SECRET=your_client_secret
   AUTH0_DOMAIN=your-tenant.auth0.com
   AUTH0_AUDIENCE=https://your-api-identifier
   ```

**Scopes requested**: `openid`, `email`, `profile`

## API Usage

### Initialize OAuth Flow

```typescript
// Frontend code
async function connectMCP(mcpName: string, provider: string) {
  const response = await fetch('/api/mcp/oauth/init', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userToken}`,
    },
    body: JSON.stringify({
      mcp_name: mcpName,
      provider: provider,
      redirect_uri: `${window.location.origin}/oauth/callback`,
    }),
  });

  const data = await response.json();

  if (data.success) {
    // Redirect user to OAuth provider
    window.location.href = data.auth_url;
  } else {
    console.error('OAuth init failed:', data.error);
  }
}
```

### Handle OAuth Callback

The callback endpoint (to be implemented) will:
1. Verify the state parameter
2. Exchange the authorization code for an access token
3. Store the encrypted token in the database
4. Redirect the user back to the application

## Security Considerations

### PKCE Implementation

PKCE prevents authorization code interception attacks:

1. **Code Verifier**: 128-character random string
2. **Code Challenge**: SHA256 hash of verifier, base64url-encoded
3. Verifier stored securely in database
4. Challenge sent to OAuth provider
5. Original verifier used during token exchange

### State Parameter

Protects against CSRF attacks:

1. 64-character random hex string generated
2. Stored with user_id and expiration (10 minutes)
3. Sent to OAuth provider
4. Verified on callback
5. Marked as "used" to prevent replay

### Token Encryption

Access tokens are encrypted before storage:

1. AES-256-GCM encryption
2. Unique IV for each token
3. Authentication tag for integrity
4. Encryption key from environment variable
5. Row-Level Security prevents unauthorized access

### Row-Level Security (RLS)

All OAuth-related tables use RLS:

```sql
CREATE POLICY "Users can manage their own OAuth states" ON oauth_state
    FOR ALL USING (user_id = auth.uid());
```

This ensures users can only access their own OAuth data.

## Database Schema

### oauth_state Table

```sql
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
```

**Indexes**:
- `idx_oauth_state_state` - Fast state lookup
- `idx_oauth_state_user_id` - User's states
- `idx_oauth_state_expires_at` - Cleanup expired states

## Monitoring and Debugging

### Logs

The endpoint logs important events:

```typescript
console.error('OAuth initialization error:', error);
console.error('Failed to store OAuth state:', insertError);
```

View logs in Vercel dashboard or use:

```bash
vercel logs
```

### Common Issues

**1. "Missing Supabase configuration"**
- Check `SUPABASE_URL` and `SUPABASE_SERVICE_ROLE_KEY` are set

**2. "Provider is not properly configured"**
- Verify OAuth provider credentials are set in environment variables
- Check client ID and secret are correct

**3. "Invalid or expired token"**
- User's JWT token has expired
- Request new token from Supabase auth

**4. "Failed to store OAuth state"**
- Database connection issue
- Check `oauth_state` table exists
- Verify RLS policies are correct

### Testing

Run the test suite:

```bash
# Install test dependencies
npm install --save-dev vitest @types/node

# Run tests
npx vitest

# Run with coverage
npx vitest --coverage
```

See `api/mcp/oauth/init.test.ts` for example tests.

## Maintenance

### Cleanup Expired States

Set up a periodic job to clean expired OAuth states:

```sql
-- Run daily via pg_cron or external scheduler
DELETE FROM oauth_state
WHERE expires_at < NOW() - INTERVAL '1 hour';
```

### Token Refresh

Implement token refresh logic for providers that support it:

1. Check token expiration before use
2. Use refresh token to get new access token
3. Update database with new tokens
4. Handle refresh token expiration

## Next Steps

1. **Implement OAuth Callback** (`/api/mcp/oauth/callback`)
2. **Add Token Refresh** for providers that support it
3. **Implement Token Revocation** endpoint
4. **Add Monitoring** with Prometheus/Grafana
5. **Set up Alerting** for OAuth failures

## Resources

- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [PKCE RFC 7636](https://tools.ietf.org/html/rfc7636)
- [Vercel Edge Functions](https://vercel.com/docs/functions/edge-functions)
- [Supabase Auth](https://supabase.com/docs/guides/auth)
- [GitHub OAuth](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [Google OAuth](https://developers.google.com/identity/protocols/oauth2)
- [Azure OAuth](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-auth-code-flow)
- [Auth0 OAuth](https://auth0.com/docs/authorization/flows/authorization-code-flow)

## Support

For issues or questions:
1. Check the [README](./api/mcp/oauth/README.md) in the oauth directory
2. Review the implementation plan
3. Open an issue on GitHub
4. Contact the development team

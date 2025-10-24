# OAuth Quick Reference Card

## File Structure
```
/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/
├── api/mcp/oauth/
│   ├── callback.ts          ★ Main OAuth callback handler (445 lines)
│   ├── initiate.ts          ★ OAuth flow initiator (155 lines)
│   ├── refresh.ts           ★ Token refresh endpoint (113 lines)
│   ├── revoke.ts            ★ Token revocation endpoint (107 lines)
│   ├── helpers.ts           ★ Shared utilities (413 lines)
│   ├── .env.example         ★ Environment template
│   ├── README.md              Existing API docs
│   └── __tests__/
│       └── oauth.test.ts    ★ Test suite (450+ lines)
├── database/migrations/
│   └── 002_oauth_tables.sql ★ OAuth database schema
├── OAUTH_IMPLEMENTATION_GUIDE.md  ★ Complete guide
├── OAUTH_FILES_SUMMARY.md         ★ Detailed summary
└── OAUTH_QUICK_REFERENCE.md       ★ This file

★ = New files created
Total: ~2,400 lines of TypeScript code
```

## Core Components

### 1. Callback Handler (`callback.ts`)
**Purpose:** Handle OAuth provider callbacks and exchange codes for tokens

**Key Functions:**
- `handler(req, res)`: Main Vercel API Route handler
- `getOAuthState(state)`: Retrieve and validate state
- `exchangeCodeForTokens()`: Exchange code with provider
- `storeTokens()`: Encrypt and save tokens
- `encrypt/decrypt()`: AES-256-GCM encryption

**Flow:**
1. Receive callback with `code` and `state`
2. Validate `state` (CSRF protection)
3. Retrieve `code_verifier` from database
4. Exchange `code` + `code_verifier` for tokens (PKCE)
5. Encrypt tokens with AES-256-GCM
6. Store in `mcp_oauth_tokens` table
7. Redirect to success page

### 2. Initiate Endpoint (`initiate.ts`)
**Purpose:** Start OAuth flow

**Flow:**
1. Generate PKCE parameters
2. Generate random state
3. Store in database
4. Build authorization URL
5. Return to frontend

### 3. Helper Functions (`helpers.ts`)
**Key Exports:**
- `encrypt(text)` - Encrypt tokens
- `decrypt(encryptedData)` - Decrypt tokens
- `generatePKCE()` - Generate PKCE parameters
- `generateState()` - Generate CSRF state
- `refreshAccessToken()` - Refresh expired tokens
- `getValidAccessToken()` - Get token, refresh if needed
- `revokeOAuthTokens()` - Revoke and delete tokens
- `buildAuthorizationUrl()` - Build provider auth URL

## Environment Variables

```bash
# Required
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_SERVICE_ROLE_KEY=xxx
OAUTH_ENCRYPTION_KEY=$(openssl rand -hex 32)
FRONTEND_URL=http://localhost:3000

# GitHub
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx

# Google
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx

# Azure
AZURE_CLIENT_ID=xxx
AZURE_CLIENT_SECRET=xxx
AZURE_TENANT_ID=common
AZURE_TOKEN_ENDPOINT=https://login.microsoftonline.com/common/oauth2/v2.0/token
AZURE_AUTH_ENDPOINT=https://login.microsoftonline.com/common/oauth2/v2.0/authorize

# Auth0
AUTH0_CLIENT_ID=xxx
AUTH0_CLIENT_SECRET=xxx
AUTH0_DOMAIN=xxx.auth0.com
AUTH0_TOKEN_ENDPOINT=https://xxx.auth0.com/oauth/token
AUTH0_AUTH_ENDPOINT=https://xxx.auth0.com/authorize
```

## Database Tables

### oauth_states (Temporary)
```sql
id           UUID PRIMARY KEY
state        TEXT UNIQUE NOT NULL
provider     TEXT NOT NULL
mcp_name     TEXT NOT NULL
user_id      UUID NOT NULL
code_verifier TEXT (encrypted)
redirect_uri TEXT NOT NULL
created_at   TIMESTAMP
```
**Retention:** 5 minutes

### mcp_oauth_tokens (Persistent)
```sql
id            UUID PRIMARY KEY
user_id       UUID NOT NULL
mcp_name      TEXT NOT NULL
provider      TEXT NOT NULL
access_token  TEXT NOT NULL (encrypted)
refresh_token TEXT (encrypted)
expires_at    TIMESTAMP NOT NULL
token_type    TEXT DEFAULT 'Bearer'
scope         TEXT
created_at    TIMESTAMP
updated_at    TIMESTAMP
```
**Retention:** Until revoked

## API Endpoints

### POST /api/mcp/oauth/initiate
**Headers:** `Authorization: Bearer <token>`

**Request:**
```json
{
  "provider": "github",
  "mcp_name": "my-mcp",
  "scopes": ["repo", "user"]
}
```

**Response:**
```json
{
  "success": true,
  "authorization_url": "https://...",
  "state": "xxx",
  "provider": "github",
  "mcp_name": "my-mcp"
}
```

### GET /api/mcp/oauth/callback
**Query Params:** `code=xxx&state=xxx`

**Response:** Redirects to:
- Success: `/oauth/success?mcp=xxx&provider=xxx`
- Error: `/oauth/callback?error=xxx&error_description=xxx`

### POST /api/mcp/oauth/refresh
**Headers:** `Authorization: Bearer <token>`

**Request:**
```json
{
  "mcp_name": "my-mcp",
  "provider": "github"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Token refreshed",
  "expires_at": "2024-12-31T23:59:59Z",
  "token_type": "Bearer"
}
```

### POST /api/mcp/oauth/revoke
**Headers:** `Authorization: Bearer <token>`

**Request:**
```json
{
  "mcp_name": "my-mcp",
  "provider": "github"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Tokens revoked"
}
```

## Security Features

### PKCE (RFC 7636)
- **Code Verifier:** 43-128 chars (base64url)
- **Code Challenge:** SHA256(verifier) in base64url
- **Method:** S256

### CSRF Protection
- Random 32-byte state
- 5-minute expiration
- One-time use
- Database validation

### Token Encryption (AES-256-GCM)
- **Key:** 256 bits (64 hex chars)
- **IV:** Random 16 bytes per encryption
- **Auth Tag:** 16 bytes for integrity
- **Format:** `<iv>:<tag>:<encrypted>` (hex)

### Row Level Security (RLS)
```sql
CREATE POLICY "oauth_user_policy" ON mcp_oauth_tokens
  FOR ALL USING (user_id = auth.uid());
```

## Provider Endpoints

| Provider | Auth URL | Token URL |
|----------|----------|-----------|
| GitHub | `https://github.com/login/oauth/authorize` | `https://github.com/login/oauth/access_token` |
| Google | `https://accounts.google.com/o/oauth2/v2/auth` | `https://oauth2.googleapis.com/token` |
| Azure | `https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize` | `https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token` |
| Auth0 | `https://{domain}/authorize` | `https://{domain}/oauth/token` |

## Common Commands

### Generate Encryption Key
```bash
openssl rand -hex 32
```

### Run Database Migration
```bash
psql "postgresql://..." -f database/migrations/002_oauth_tables.sql
```

### Test Initiate Endpoint
```bash
curl -X POST http://localhost:3000/api/mcp/oauth/initiate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"github","mcp_name":"test","scopes":["repo"]}'
```

### Check OAuth States
```sql
SELECT * FROM oauth_states WHERE user_id = 'xxx';
```

### Check Stored Tokens
```sql
SELECT
  mcp_name,
  provider,
  expires_at,
  created_at
FROM mcp_oauth_tokens
WHERE user_id = 'xxx';
```

### View Audit Logs
```sql
SELECT
  action,
  details->>'provider' as provider,
  created_at
FROM audit_logs
WHERE resource_type = 'oauth_token'
  AND user_id = 'xxx'
ORDER BY created_at DESC;
```

### Cleanup Expired States
```sql
SELECT cleanup_expired_oauth_states();
```

## Frontend Integration

### React Hook
```typescript
const { initiateOAuth } = useOAuth();

const handleConnect = async () => {
  await initiateOAuth('github', 'my-mcp', ['repo', 'user']);
};
```

### Success Page
```typescript
// pages/oauth/success.tsx
useEffect(() => {
  if (window.opener) {
    window.opener.postMessage({
      type: 'oauth_success',
      mcp, provider
    }, window.location.origin);
  }
}, []);
```

## Error Codes

| Code | Error | Solution |
|------|-------|----------|
| 400 | Missing parameters | Check request body |
| 401 | Invalid state | State expired or invalid |
| 401 | Token exchange failed | Check provider credentials |
| 500 | Database error | Check Supabase connection |
| 500 | Encryption error | Check OAUTH_ENCRYPTION_KEY |

## Testing

### Run Tests
```bash
cd api
npm test
```

### Test Coverage
- ✅ Encryption/Decryption
- ✅ PKCE generation
- ✅ State generation
- ✅ URL building
- ✅ Security (tamper detection)
- ✅ Error handling

## Deployment Checklist

- [ ] Generate encryption key
- [ ] Set environment variables
- [ ] Run database migration
- [ ] Configure OAuth providers
- [ ] Deploy to Vercel
- [ ] Test OAuth flow
- [ ] Verify token encryption
- [ ] Check audit logs

## Quick Debug

### Check encryption key
```bash
echo $OAUTH_ENCRYPTION_KEY | wc -c
# Should output: 65 (64 chars + newline)
```

### Check state expiration
```sql
SELECT
  state,
  created_at,
  NOW() - created_at as age
FROM oauth_states
WHERE state = 'xxx';
```

### Check token validity
```sql
SELECT
  mcp_name,
  expires_at,
  expires_at > NOW() as is_valid
FROM mcp_oauth_tokens
WHERE user_id = 'xxx';
```

## Support Files

1. **README.md** - API documentation
2. **OAUTH_IMPLEMENTATION_GUIDE.md** - Complete guide (3000+ lines)
3. **OAUTH_FILES_SUMMARY.md** - Detailed file summary
4. **OAUTH_QUICK_REFERENCE.md** - This file

## Provider Setup Links

- **GitHub:** https://github.com/settings/developers
- **Google:** https://console.cloud.google.com/apis/credentials
- **Azure:** https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/RegisteredApps
- **Auth0:** https://manage.auth0.com/dashboard

## Key Timestamps

- State expiration: 5 minutes
- Token refresh buffer: 5 minutes before expiry
- Audit log retention: Configurable
- Cleanup frequency: Every 5 minutes (states), Daily (expired tokens)

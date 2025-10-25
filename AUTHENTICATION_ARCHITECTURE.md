# Authentication Architecture - AgentAPI ChatServer

## Overview

AgentAPI ChatServer now supports multiple authentication methods to handle different use cases:

1. **WorkOS/AuthKit JWT** (Primary) - For authenticated users via the atoms.tech frontend
2. **API Key Authentication** (Fallback) - For service-to-service communication and programmatic access
3. **mTLS** (Planned) - For internal service-to-service communication with client certificates

---

## Current Implementation Status

### ✅ COMPLETED

#### 1. WorkOS/AuthKit JWT Authentication
- **Location**: `lib/auth/authkit.go` - `ValidateToken()` and `validateWorkOSToken()` methods
- **Status**: Fully implemented and tested
- **Latest Fix**: Commit `5bf908a` - Made org claim optional with fallback to org_id or default-org
- **Features**:
  - RS256 signature verification against WorkOS JWKS endpoint
  - JWT claim validation (sub, email, name, role, permissions)
  - Token expiration and issued-at time validation
  - Platform admin status checking from database
  - Graceful handling of missing org claims

**How it works**:
```
Frontend (atoms.tech) 
  ↓ (withAuth() gets WorkOS token from session)
Chat API Route Handler
  ↓ (extracts accessToken from withAuth())
AgentAPI Service setTokenGetter()
  ↓ (stores token getter function)
AgentAPI Client
  ↓ (calls getAuthHeader() to get Bearer token)
ChatServer /v1/chat/completions endpoint
  ↓ (extracts Bearer token from Authorization header)
AuthKit Validator ValidateToken()
  ↓ (parses JWT, loads JWKS keys, verifies signature)
Token Valid → AuthKitUser object with user identity
```

#### 2. API Key Authentication
- **Location**: `lib/auth/apikey.go` - New `APIKeyValidator` class
- **Status**: Code implemented, awaiting database setup
- **Features**:
  - SHA256-based key hashing (never stores plaintext keys)
  - Database-backed validation (queries api_keys table)
  - Support for key expiration
  - Support for key deactivation (soft delete)
  - Last-used timestamp tracking for audit logging
  - Row-Level Security (RLS) policies for data protection

**Database Schema** (Ready to migrate):
```sql
CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  organization_id TEXT NOT NULL,
  key_hash TEXT NOT NULL UNIQUE,
  name TEXT,
  description TEXT,
  is_active BOOLEAN DEFAULT true,
  expires_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  last_used_at TIMESTAMP NULL
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX idx_api_keys_active ON api_keys(is_active) WHERE is_active = true;
```

See **API_KEY_SETUP.md** for complete setup instructions.

### ⏳ IN PROGRESS / PENDING

#### 3. API Key Endpoints (Not Yet Implemented)
These HTTP endpoints need to be created for API key management:

- `POST /api/v1/api-keys` - Create new API key
- `GET /api/v1/api-keys` - List user's API keys
- `GET /api/v1/api-keys/{id}` - Get specific API key details
- `DELETE /api/v1/api-keys/{id}` - Revoke/delete API key
- `POST /api/v1/api-keys/{id}/rotate` - Rotate existing key

#### 4. mTLS Authentication (Planned)
- Designed but not yet implemented
- Will use client certificates for internal service-to-service communication
- Useful for service meshes and microservices deployments

---

## Architecture Decisions

### 1. Why WorkOS-Only (No More Supabase)?
**Problem**: 
- Supabase JWKS endpoint was returning permanent 503 errors
- Infrastructure unavailability made authentication impossible
- Needed reliable, production-grade authentication

**Solution**:
- Switched to WorkOS/AuthKit as primary (and only) JWT authentication method
- WorkOS has 99.99% SLA and dedicated support for authentication
- All user authentication now flows through WorkOS exclusively

**Migration Path**:
```
Previous: Frontend (Supabase JWT) → ChatServer (validate with Supabase JWKS)
Current:  Frontend (WorkOS JWT) → ChatServer (validate with WorkOS JWKS)
```

### 2. Why Fallback to API Keys?
**Problem**:
- Some clients may not be able to use OAuth/OIDC flows
- Service-to-service communication needs different auth mechanism
- Internal tools need programmatic access

**Solution**:
- API keys stored in Supabase database (using service-role access, not JWKS)
- SHA256 hashing prevents key compromise from database breaches
- Supports expiration, deactivation, and audit logging

### 3. Unified AuthKitUser Response
All authentication methods return the same `AuthKitUser` struct:
```go
type AuthKitUser struct {
    ID                   string   // User ID
    OrgID                string   // Organization ID
    Email                string   // User email
    Name                 string   // User name
    Role                 string   // Role (admin, member, viewer, etc.)
    Permissions          []string // List of permissions
    IsOrgAdminFlag       bool     // Is org admin? (from WorkOS role)
    IsPlatformAdminFlag  bool     // Is platform admin? (from DB)
    AuthenticationMethod string   // "jwt" or "api_key"
    Token                string   // Original token (for audit/replay)
}
```

This allows downstream code to work with any authentication method without changes.

---

## Files Modified / Created

### Modified Files
1. **lib/auth/authkit.go** (4 commits)
   - Removed all Supabase JWT validation logic
   - Simplified `ValidateToken()` to WorkOS-only
   - Made org claim optional with intelligent fallback
   - Added `AuthenticationMethod` field to track auth source

2. **cmd/chatserver/main.go**
   - Added godotenv auto-loading for `.env` file
   - Improved configuration logging

3. **.env**
   - Updated to use WorkOS configuration
   - Removed Supabase JWKS validation references

### New Files
1. **lib/auth/apikey.go** (NEW)
   - Complete API key validation implementation
   - SHA256 hashing and database queries
   - Platform admin status checking

2. **API_KEY_SETUP.md** (NEW)
   - Complete step-by-step setup guide
   - SQL migrations with correct PostgreSQL syntax
   - Code examples in Python, JavaScript, Go, cURL
   - Security best practices
   - Testing procedures
   - Troubleshooting guide

3. **AUTHENTICATION_ARCHITECTURE.md** (THIS FILE)
   - High-level architecture overview
   - Implementation status and decisions
   - Integration points

---

## Integration Points

### Frontend (atoms.tech)
- **Auth Source**: WorkOS AuthKit (`withAuth()` hook)
- **Token Type**: JWT from WorkOS
- **Token Destination**: Passed via `Authorization: Bearer {token}` header
- **Service**: `lib/services/agentapi.ts` → `AgentAPIClient.getAuthHeader()`

### ChatServer
- **Auth Endpoints**:
  - `ValidateToken()` - For WorkOS JWT tokens
  - `ValidateAPIKey()` - For API key authentication
- **Middleware**: Authentication happens at handler level in `pkg/server/`
- **Response**: `AuthKitUser` object with authenticated user info

### Supabase Database
- **JWT Validation**: NO LONGER USED
- **API Key Storage**: YES - Uses service-role access (bypass JWT)
- **Platform Admin Lookup**: YES - Queries `platform_admins` table

### Environment Configuration
```bash
# Required
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_01K4CGW2J1FGWZYZJDMVWGQZBD

# Optional (for Supabase DB access, not JWT validation)
DATABASE_URL=postgresql://...  # For API key validation
SUPABASE_URL=...               # Alternative: use Supabase client
SUPABASE_SERVICE_ROLE_KEY=...  # For database access
```

---

## Testing the Authentication

### 1. WorkOS JWT Flow (Frontend Integration)
```bash
# This happens automatically when user logs in via atoms.tech
# Frontend obtains token from WorkOS
# Frontend passes token to ChatServer in Authorization header
# ChatServer validates and creates authenticated session
```

### 2. API Key Flow (Programmatic Access)
```bash
# Create API key (requires endpoint not yet implemented)
curl -X POST http://localhost:3284/api/v1/api-keys \
  -H "Authorization: Bearer {workos-token}" \
  -H "Content-Type: application/json" \
  -d '{"name": "Integration Key", "expires_in_days": 90}'

# Use API key
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer sk_prod_xyz123abc456" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

### 3. Health Check
```bash
curl http://localhost:3284/health
# Response: {"status":"healthy","agents":["ccrouter"],"primary":"ccrouter"}
```

---

## Security Considerations

### ✅ Security Measures in Place
1. **JWT Signature Verification**: RS256 signature validated against JWKS
2. **Key Hashing**: API keys hashed with SHA256 before storage
3. **Expiration Checking**: Both JWT and API keys support expiration
4. **Soft Deletes**: API keys can be deactivated without deletion
5. **Audit Trail**: `last_used_at` field tracks API key usage
6. **RLS Policies**: Database row-level security controls access
7. **Token Isolation**: Tokens never logged in plaintext
8. **Platform Admin Verification**: Database check prevents privilege escalation

### ⚠️ To Do Before Production
1. [ ] Implement rate limiting on token validation
2. [ ] Add request signing/HMAC for API key usage
3. [ ] Implement API key rotation mechanism
4. [ ] Set up audit logging for all authentication events
5. [ ] Configure CORS policies for frontend integration
6. [ ] Enable HTTPS/TLS for all connections
7. [ ] Implement token refresh mechanisms if needed
8. [ ] Add IP whitelisting for API keys (optional)

---

## Recent Commits

1. **5bf908a** - `fix: Make WorkOS token org claim optional with fallback`
   - Resolves "missing 'org' claim" errors
   - Allows flexible WorkOS token structure

2. **3256d1a** - `feat: Add API key authentication as fallback auth method`
   - New `APIKeyValidator` class
   - SHA256 key hashing
   - Database-backed validation

3. **4468e20** - `refactor: Simplify authentication to WorkOS-only mode`
   - Removed Supabase JWT validation routing
   - Simplified `ValidateToken()` method
   - Only uses WorkOS JWKS

4. **beab2a2** - `feat: Add automatic .env file loading to ChatServer`
   - Environment variable auto-loading
   - No need to manually export variables

---

## Next Steps

### Phase 1: API Key Infrastructure (Ready Now)
1. Create `api_keys` table in Supabase (SQL provided in API_KEY_SETUP.md)
2. Implement HTTP endpoints for key management
3. Test end-to-end API key flow

### Phase 2: mTLS Support (Future)
1. Design certificate generation and distribution
2. Implement mTLS validator
3. Set up certificate management
4. Test service-to-service authentication

### Phase 3: Advanced Features (Future)
1. API key rotation workflows
2. Advanced audit logging
3. Request signing with HMAC
4. IP whitelisting per key
5. Rate limiting per key

---

## Configuration Reference

### ChatServer Environment Variables
```bash
# Authentication
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/...  # Required

# Database (for API key validation and platform admin checks)
DATABASE_URL=postgresql://...        # PostgreSQL connection
# OR
SUPABASE_URL=...                      # Supabase project URL
SUPABASE_SERVICE_ROLE_KEY=...         # Service role key for DB access

# Server
PORT=3284                             # HTTP port
CCROUTER_PATH=/usr/local/bin/ccrouter
PRIMARY_AGENT=ccrouter
FALLBACK_ENABLED=false

# Features
METRICS_ENABLED=true
AUDIT_ENABLED=true
```

### Frontend Environment Variables
```bash
# AgentAPI Server
NEXT_PUBLIC_AGENTAPI_URL=http://localhost:3284

# WorkOS
WORKOS_CLIENT_ID=client_01K4CGW2J1FGWZYZJDMVWGQZBD
WORKOS_API_KEY=sk_test_...

# For backward compatibility only (JWT auth disabled)
NEXT_PUBLIC_SUPABASE_URL=https://...
NEXT_PUBLIC_SUPABASE_ANON_KEY=...
```

---

## Troubleshooting

### Error: "missing 'org' claim"
**Cause**: WorkOS token doesn't include `org` field
**Solution**: Already fixed in commit 5bf908a - uses `org_id` or default fallback

### Error: "invalid token format"
**Cause**: Authorization header missing Bearer token or malformed JWT
**Solution**: Check token is sent as `Authorization: Bearer {token}`

### Error: "invalid or expired API key"
**Cause**: API key not found, inactive, or expired
**Solution**: Check key exists in database and is active/not expired

### ChatServer won't start
**Cause**: Missing AUTHKIT_JWKS_URL environment variable
**Solution**: See Configuration Reference above

---

## Questions or Issues?

Refer to the specific setup guides:
- API Key Authentication: See **API_KEY_SETUP.md**
- JWT Validation: Check **lib/auth/authkit.go**
- Client Integration: See **agentapi-service.ts** and **agentapi.ts**

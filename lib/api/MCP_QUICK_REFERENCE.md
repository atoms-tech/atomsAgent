# MCP API Quick Reference

## File Overview

### Core Files
- **`mcp.go`** - Main MCP handler with CRUD operations, validation, encryption
- **`mcp_routes.go`** - Route registration and middleware
- **`mcp_test.go`** - Comprehensive test suite
- **`mcp_example_integration.go`** - Integration examples

### Database
- **`migrations/001_create_mcp_configurations.sql`** - Database schema

### Documentation
- **`MCP_API_README.md`** - Complete API documentation
- **`MCP_QUICK_REFERENCE.md`** - This file

## Quick Start

### 1. Setup Database

```bash
# SQLite example
sqlite3 agentapi.db < migrations/001_create_mcp_configurations.sql
```

### 2. Set Environment Variables

```bash
export MCP_ENCRYPTION_KEY="your-32-byte-key-here-12345678"
export DATABASE_URL="sqlite:///path/to/agentapi.db"
```

### 3. Initialize Handler

```go
import (
    "database/sql"
    "github.com/coder/agentapi/lib/api"
    "github.com/coder/agentapi/lib/mcp"
    "github.com/coder/agentapi/lib/session"
)

// Create dependencies
db, _ := sql.Open("sqlite3", "/path/to/agentapi.db")
fastmcpClient, _ := mcp.NewFastMCPClient()
sessionMgr := session.NewSessionManager("/var/lib/agentapi")
auditLogger := api.NewAuditLogger()

// Create handler
mcpHandler, err := api.NewMCPHandler(
    db,
    fastmcpClient,
    sessionMgr,
    auditLogger,
    os.Getenv("MCP_ENCRYPTION_KEY"),
)
```

### 4. Register Routes

```go
import "github.com/go-chi/chi/v5"

router := chi.NewRouter()
api.RegisterMCPRoutes(router, mcpHandler)
```

## API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/mcp/configurations` | Create MCP config |
| GET | `/api/v1/mcp/configurations` | List MCP configs |
| GET | `/api/v1/mcp/configurations/:id` | Get single MCP |
| PUT | `/api/v1/mcp/configurations/:id` | Update MCP |
| DELETE | `/api/v1/mcp/configurations/:id` | Delete MCP |
| POST | `/api/v1/mcp/test` | Test connection |

## Request Examples

### Create HTTP MCP
```bash
curl -X POST http://localhost:8080/api/v1/mcp/configurations \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My HTTP MCP",
    "type": "http",
    "endpoint": "https://api.example.com/mcp",
    "auth_type": "bearer",
    "auth_token": "secret",
    "scope": "org",
    "enabled": true
  }'
```

### Create Stdio MCP
```bash
curl -X POST http://localhost:8080/api/v1/mcp/configurations \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Python MCP",
    "type": "stdio",
    "command": "/usr/bin/python3",
    "args": ["-m", "mcp_server"],
    "auth_type": "none",
    "scope": "user",
    "enabled": true
  }'
```

### Test Connection
```bash
curl -X POST http://localhost:8080/api/v1/mcp/test \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test",
    "type": "http",
    "endpoint": "https://api.example.com/mcp",
    "auth_type": "bearer",
    "auth_token": "test-token"
  }'
```

### List MCPs (with filters)
```bash
# All MCPs
curl -X GET http://localhost:8080/api/v1/mcp/configurations \
  -H "Authorization: Bearer TOKEN"

# Only HTTP MCPs
curl -X GET "http://localhost:8080/api/v1/mcp/configurations?type=http" \
  -H "Authorization: Bearer TOKEN"

# Only enabled MCPs
curl -X GET "http://localhost:8080/api/v1/mcp/configurations?enabled=true" \
  -H "Authorization: Bearer TOKEN"
```

### Update MCP
```bash
curl -X PUT http://localhost:8080/api/v1/mcp/configurations/mcp_123 \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false,
    "description": "Disabled for maintenance"
  }'
```

### Delete MCP
```bash
curl -X DELETE http://localhost:8080/api/v1/mcp/configurations/mcp_123 \
  -H "Authorization: Bearer TOKEN"
```

## MCP Types

### HTTP
```json
{
  "type": "http",
  "endpoint": "https://api.example.com/mcp",
  "auth_type": "bearer",
  "auth_token": "token"
}
```

### SSE (Server-Sent Events)
```json
{
  "type": "sse",
  "endpoint": "https://api.example.com/mcp/stream",
  "auth_type": "api_key",
  "auth_token": "key",
  "auth_header": "X-API-Key"
}
```

### Stdio (Local Process)
```json
{
  "type": "stdio",
  "command": "/usr/bin/python3",
  "args": ["-m", "my_mcp_server", "--port", "8080"],
  "auth_type": "none"
}
```

## Auth Types

| Type | Description | Required Fields |
|------|-------------|-----------------|
| `none` | No authentication | - |
| `bearer` | Bearer token in Authorization header | `auth_token` |
| `oauth` | OAuth 2.0 (future) | `auth_token` |
| `api_key` | Custom header API key | `auth_token`, `auth_header` |

## Scope Types

| Scope | Description | Access |
|-------|-------------|--------|
| `org` | Organization-level | All users in org |
| `user` | User-level | Only the creating user |

## Security Features

### Input Validation
- ✅ URL validation (http/https only)
- ✅ Command injection prevention
- ✅ SQL injection prevention (parameterized queries)
- ✅ Length limits (name: 255, description: 1000)

### Encryption
- ✅ AES-256-GCM for auth tokens
- ✅ Unique nonce per encryption
- ✅ Base64 encoding for storage

### Tenant Isolation
- ✅ Row-level security (org_id, user_id)
- ✅ Query filtering by tenant
- ✅ Ownership verification on update/delete

### Audit Logging
- ✅ All CRUD operations logged
- ✅ User ID, Org ID, timestamp
- ✅ Action details and metadata

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `missing_auth` | 401 | No authentication provided |
| `invalid_json` | 400 | Malformed JSON body |
| `validation_error` | 400 | Input validation failed |
| `not_found` | 404 | Resource not found |
| `forbidden` | 403 | Insufficient permissions |
| `database_error` | 500 | Database operation failed |
| `encryption_error` | 500 | Encryption/decryption failed |

## Testing

### Run Tests
```bash
cd lib/api
go test -v -run TestMCP
```

### Test Coverage
```bash
go test -cover ./lib/api
```

### Specific Tests
```bash
# Test creation
go test -v -run TestCreateMCPConfiguration

# Test tenant isolation
go test -v -run TestTenantIsolation

# Test encryption
go test -v -run TestEncryptionDecryption
```

## Common Patterns

### Pattern 1: Test Before Create
```go
// Test connection first
testResult := testConnection(config)
if testResult.Success {
    // Create configuration
    createMCP(config)
}
```

### Pattern 2: Load MCPs for Session
```go
// Load all enabled MCPs for user/org
mcps := listMCPs(userID, orgID, enabled=true)

// Connect to each
for _, mcp := range mcps {
    connectMCP(mcp)
}
```

### Pattern 3: Rotate Credentials
```go
// Update auth token
updateMCP(mcpID, {
    auth_token: newToken
})
```

### Pattern 4: Disable Instead of Delete
```go
// Disable temporarily
updateMCP(mcpID, {
    enabled: false,
    description: "Temporarily disabled"
})

// Re-enable later
updateMCP(mcpID, {
    enabled: true
})
```

## Troubleshooting

### Issue: Connection Test Fails
**Cause:** Invalid endpoint or credentials
**Solution:**
1. Verify endpoint URL is correct
2. Check auth token is valid
3. Ensure MCP server is running
4. Check network connectivity

### Issue: Permission Denied on Update
**Cause:** Trying to modify org-scoped MCP as regular user
**Solution:**
1. Check scope of MCP (`GET /configurations/:id`)
2. Verify user has org-level permissions
3. Use user-scoped MCPs for personal configs

### Issue: Encryption Error
**Cause:** Invalid or missing encryption key
**Solution:**
1. Verify `MCP_ENCRYPTION_KEY` env var is set
2. Ensure key is exactly 32 bytes
3. Check key hasn't changed (can't decrypt old data)

### Issue: Command Injection Warning
**Cause:** Command contains dangerous characters
**Solution:**
1. Remove special characters: `;`, `&&`, `||`, etc.
2. Use absolute paths
3. Pass arguments via `args` array, not in command

## Best Practices

1. **Always test before creating** - Use `/test` endpoint
2. **Use appropriate scope** - org for shared, user for personal
3. **Descriptive names** - Help identify purpose
4. **Rotate credentials** - Update tokens regularly
5. **Monitor audit logs** - Track who does what
6. **Disable vs Delete** - Can re-enable disabled configs
7. **Use HTTPS** - Never use HTTP endpoints in production
8. **Environment-specific configs** - Separate dev/staging/prod

## Integration Checklist

- [ ] Database migration run
- [ ] Environment variables set
- [ ] Handler initialized with all dependencies
- [ ] Routes registered
- [ ] Authentication middleware configured
- [ ] Audit logging enabled
- [ ] Error handling implemented
- [ ] Tests passing
- [ ] Documentation updated
- [ ] Security review completed

## Support

For issues or questions:
1. Check the full documentation in `MCP_API_README.md`
2. Review integration examples in `mcp_example_integration.go`
3. Run tests to verify setup: `go test ./lib/api`
4. Check audit logs for operation history

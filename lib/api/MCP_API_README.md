# MCP Management REST API

This document describes the REST API endpoints for managing Model Context Protocol (MCP) server configurations in the AgentAPI system.

## Overview

The MCP Management API provides endpoints to create, read, update, delete, and test MCP server configurations. It includes:

- **Tenant Isolation**: Org-level and user-level configurations
- **Security**: Encrypted credential storage, input validation, command injection prevention
- **Audit Logging**: All operations are logged for compliance
- **Connection Testing**: Test configurations before saving

## Architecture

### Components

1. **MCPHandler**: Main handler struct containing:
   - `db *sql.DB`: Database connection
   - `fastmcpClient *mcp.FastMCPClient`: FastMCP client for testing connections
   - `sessionMgr *session.SessionManager`: Session management
   - `auditLogger *AuditLogger`: Audit logging
   - `encryptionKey []byte`: AES-256 encryption key for sensitive data

2. **Database**: SQLite/MySQL/PostgreSQL table `mcp_configurations`
3. **Middleware**: Auth and tenant isolation
4. **Encryption**: AES-256-GCM for auth tokens

## API Endpoints

### Base URL
```
/api/v1/mcp
```

### Authentication
All endpoints require authentication. Include a valid JWT token or session cookie in requests.

---

### 1. Create MCP Configuration

**Endpoint:** `POST /api/v1/mcp/configurations`

**Description:** Create a new MCP server configuration.

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "name": "My MCP Server",
  "type": "http|sse|stdio",
  "endpoint": "https://api.example.com/mcp",  // For http/sse types
  "command": "/usr/bin/python3",              // For stdio type
  "args": ["-m", "mcp_server"],               // For stdio type
  "auth_type": "none|bearer|oauth|api_key",
  "auth_token": "secret-token",               // Optional, encrypted
  "auth_header": "X-API-Key",                 // Optional, custom header name
  "config": {                                 // Optional, additional config
    "timeout": 30,
    "retry": 3
  },
  "scope": "org|user",
  "enabled": true,
  "description": "Optional description"
}
```

**Response:** `201 Created`
```json
{
  "id": "mcp_1234567890_123456",
  "name": "My MCP Server",
  "type": "http",
  "endpoint": "https://api.example.com/mcp",
  "auth_type": "bearer",
  "auth_header": "Authorization",
  "config": {
    "timeout": 30,
    "retry": 3
  },
  "scope": "org",
  "org_id": "org-456",
  "enabled": true,
  "description": "Optional description",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "created_by": "user-123",
  "updated_by": "user-123"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid input, validation errors
- `401 Unauthorized`: Missing or invalid authentication
- `500 Internal Server Error`: Database or encryption errors

---

### 2. List MCP Configurations

**Endpoint:** `GET /api/v1/mcp/configurations`

**Description:** List all MCP configurations accessible to the user (org-level + user-level).

**Query Parameters:**
- `type` (optional): Filter by type (`http`, `sse`, `stdio`)
- `enabled` (optional): Filter by enabled status (`true`, `false`)

**Request Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "configurations": [
    {
      "id": "mcp_1234567890_123456",
      "name": "Org MCP Server",
      "type": "http",
      "endpoint": "https://api.example.com/mcp",
      "auth_type": "bearer",
      "scope": "org",
      "org_id": "org-456",
      "enabled": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "mcp_1234567891_654321",
      "name": "Personal MCP",
      "type": "stdio",
      "command": "/usr/bin/python3",
      "args": ["-m", "my_mcp"],
      "auth_type": "none",
      "scope": "user",
      "user_id": "user-123",
      "enabled": true,
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  ],
  "total": 2
}
```

**Note:** Auth tokens are NEVER returned in list/get responses for security.

---

### 3. Get MCP Configuration

**Endpoint:** `GET /api/v1/mcp/configurations/:id`

**Description:** Get a single MCP configuration by ID.

**Request Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "id": "mcp_1234567890_123456",
  "name": "My MCP Server",
  "type": "http",
  "endpoint": "https://api.example.com/mcp",
  "auth_type": "bearer",
  "auth_header": "Authorization",
  "config": {
    "timeout": 30
  },
  "scope": "org",
  "org_id": "org-456",
  "enabled": true,
  "description": "Production MCP",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "created_by": "user-123",
  "updated_by": "user-123"
}
```

**Error Responses:**
- `404 Not Found`: Configuration not found or not accessible
- `401 Unauthorized`: Missing or invalid authentication

---

### 4. Update MCP Configuration

**Endpoint:** `PUT /api/v1/mcp/configurations/:id`

**Description:** Update an existing MCP configuration. Only fields provided will be updated.

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:** (all fields optional)
```json
{
  "name": "Updated Name",
  "enabled": false,
  "description": "New description",
  "auth_token": "new-token"
}
```

**Response:** `204 No Content`

**Error Responses:**
- `400 Bad Request`: Invalid input, validation errors
- `403 Forbidden`: Insufficient permissions (can't modify org config as regular user)
- `404 Not Found`: Configuration not found
- `401 Unauthorized`: Missing or invalid authentication

**Permissions:**
- Org-scoped configs: Can be updated by users in that organization
- User-scoped configs: Can only be updated by the owner

---

### 5. Delete MCP Configuration

**Endpoint:** `DELETE /api/v1/mcp/configurations/:id`

**Description:** Delete an MCP configuration. Active connections will be disconnected.

**Request Headers:**
```
Authorization: Bearer <token>
```

**Response:** `204 No Content`

**Error Responses:**
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Configuration not found
- `401 Unauthorized`: Missing or invalid authentication

**Note:** Deleting a configuration will:
1. Disconnect any active MCP connections
2. Remove the configuration from the database
3. Log the deletion in audit logs

---

### 6. Test MCP Connection

**Endpoint:** `POST /api/v1/mcp/test`

**Description:** Test an MCP configuration without saving it. Useful for validating settings before creation.

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "name": "Test MCP",
  "type": "http",
  "endpoint": "https://api.example.com/mcp",
  "auth_type": "bearer",
  "auth_token": "test-token",
  "config": {}
}
```

**Response:** `200 OK` (Success)
```json
{
  "success": true,
  "tools": ["search", "analyze", "summarize"],
  "resources": ["docs", "database"],
  "prompts": ["code_review", "documentation"]
}
```

**Response:** `200 OK` (Failure)
```json
{
  "success": false,
  "error": "Connection timeout: failed to connect to https://api.example.com/mcp"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid configuration
- `401 Unauthorized`: Missing or invalid authentication

---

## Security Features

### 1. Input Validation

**URL Validation:**
- Only `http` and `https` schemes allowed
- Host must be present
- Prevents malformed URLs

**Command Validation:**
- Checks for dangerous characters: `;`, `&&`, `||`, `|`, `` ` ``, `$(`, newlines, redirects
- Prevents flag injection (commands starting with `-`)
- Validates against regex: `^[a-zA-Z0-9/_\.-]+$`

**Command Arguments Validation:**
- Checks each argument for dangerous patterns
- Prevents shell injection through arguments

### 2. Encryption

**AES-256-GCM Encryption:**
- All `auth_token` values are encrypted at rest
- Uses authenticated encryption (GCM mode)
- Unique nonce per encryption
- Base64 encoded for storage

**Key Management:**
- Encryption key should be 32 bytes (AES-256)
- Store in environment variable or secrets manager
- Never commit to source control

### 3. Tenant Isolation

**Scope Types:**
- `org`: Accessible to all users in the organization
- `user`: Only accessible to the creating user

**Isolation Enforcement:**
- Database constraints ensure proper scope assignment
- Queries filter by both `org_id` and `user_id`
- Update/delete operations verify ownership

### 4. Audit Logging

All operations are logged with:
- User ID and Organization ID
- Action type (create, update, delete, test)
- Resource type and ID
- Operation details
- Timestamp
- IP address (if implemented)

## Database Schema

```sql
CREATE TABLE mcp_configurations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('http', 'sse', 'stdio')),
    endpoint TEXT,
    command TEXT,
    args TEXT,  -- JSON array
    auth_type VARCHAR(50) NOT NULL,
    auth_token TEXT,  -- Encrypted
    auth_header VARCHAR(255),
    config TEXT,  -- JSON object
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('org', 'user')),
    org_id VARCHAR(255),
    user_id VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255) NOT NULL,

    -- Constraints
    CONSTRAINT chk_org_scope CHECK (
        (scope = 'org' AND org_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL AND org_id IS NULL)
    )
);
```

## Usage Examples

### Example 1: Create HTTP MCP with Bearer Token

```bash
curl -X POST https://api.example.com/api/v1/mcp/configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "Render API MCP",
    "type": "http",
    "endpoint": "https://api.render.com/mcp",
    "auth_type": "bearer",
    "auth_token": "rnd_xxxxxxxxxxxx",
    "scope": "org",
    "enabled": true,
    "description": "Render API integration"
  }'
```

### Example 2: Create Stdio MCP (Local Process)

```bash
curl -X POST https://api.example.com/api/v1/mcp/configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "Local Python MCP",
    "type": "stdio",
    "command": "/usr/bin/python3",
    "args": ["-m", "my_mcp_server"],
    "auth_type": "none",
    "scope": "user",
    "enabled": true
  }'
```

### Example 3: Test Connection Before Creating

```bash
curl -X POST https://api.example.com/api/v1/mcp/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "Test Connection",
    "type": "http",
    "endpoint": "https://api.example.com/mcp",
    "auth_type": "api_key",
    "auth_token": "test-key",
    "auth_header": "X-API-Key"
  }'
```

### Example 4: List All Enabled HTTP MCPs

```bash
curl -X GET "https://api.example.com/api/v1/mcp/configurations?type=http&enabled=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Example 5: Update MCP Configuration

```bash
curl -X PUT https://api.example.com/api/v1/mcp/configurations/mcp_123456789_123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "enabled": false,
    "description": "Temporarily disabled for maintenance"
  }'
```

### Example 6: Delete MCP Configuration

```bash
curl -X DELETE https://api.example.com/api/v1/mcp/configurations/mcp_123456789_123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Integration with FastMCP

The API uses the FastMCP client to:

1. **Test Connections**: Validate credentials and connectivity
2. **List Capabilities**: Discover tools, resources, and prompts
3. **Manage Lifecycle**: Connect/disconnect MCP servers

When testing a configuration:
- Temporary connection is established
- Tools, resources, and prompts are enumerated
- Connection is immediately closed
- Results are returned to the user

## Error Handling

### Error Response Format

```json
{
  "error": "Human-readable error message",
  "code": "error_code",
  "details": {}  // Optional additional context
}
```

### Common Error Codes

- `missing_auth`: Authentication required
- `invalid_json`: Request body is not valid JSON
- `validation_error`: Input validation failed
- `not_found`: Resource not found or not accessible
- `forbidden`: Insufficient permissions
- `database_error`: Database operation failed
- `encryption_error`: Failed to encrypt/decrypt data

## Best Practices

1. **Always test connections** before creating configurations
2. **Use org-scope** for shared MCPs, user-scope for personal ones
3. **Rotate auth tokens** regularly
4. **Monitor audit logs** for security
5. **Set appropriate descriptions** for easier management
6. **Disable unused MCPs** instead of deleting (can re-enable later)
7. **Use HTTPS endpoints** for security
8. **Validate MCP responses** in your application

## Deployment

### Environment Variables

```bash
# Required
MCP_ENCRYPTION_KEY=your-32-byte-encryption-key-here
DATABASE_URL=sqlite:///path/to/db.sqlite

# Optional
AUDIT_LOG_LEVEL=info
MCP_CONNECTION_TIMEOUT=30
```

### Database Migration

Run the migration SQL to create tables:

```bash
sqlite3 /path/to/db.sqlite < migrations/001_create_mcp_configurations.sql
```

### Route Registration

```go
import (
    "database/sql"
    "github.com/coder/agentapi/lib/api"
    "github.com/coder/agentapi/lib/mcp"
    "github.com/go-chi/chi/v5"
)

func setupRoutes() *chi.Mux {
    router := chi.NewRouter()

    // Setup dependencies
    db, _ := sql.Open("sqlite3", "/path/to/db.sqlite")
    fastmcpClient, _ := mcp.NewFastMCPClient()
    sessionMgr := session.NewSessionManager("/var/lib/agentapi")
    auditLogger := api.NewAuditLogger()

    // Create handler
    mcpHandler, _ := api.NewMCPHandler(
        db,
        fastmcpClient,
        sessionMgr,
        auditLogger,
        os.Getenv("MCP_ENCRYPTION_KEY"),
    )

    // Register routes
    api.RegisterMCPRoutes(router, mcpHandler)

    return router
}
```

## Future Enhancements

- [ ] Rate limiting per user/org
- [ ] MCP server health checks
- [ ] Automatic reconnection on failure
- [ ] Configuration versioning
- [ ] Bulk operations
- [ ] Export/import configurations
- [ ] Webhook notifications for connection status
- [ ] OAuth2 flow support
- [ ] Certificate-based authentication

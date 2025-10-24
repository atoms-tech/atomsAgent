# AgentAPI Database Schema

This directory contains the complete multi-tenant database schema for AgentAPI.

## Files

- `schema.sql` - Complete PostgreSQL schema with all tables, indexes, RLS policies, and functions

## Schema Overview

### Extensions
- `uuid-ossp` - UUID generation
- `pgcrypto` - Encryption for sensitive data (tokens)

### Custom Types
- `user_role` - User roles (admin, user)
- `mcp_type` - MCP connection types (http, sse, stdio)
- `auth_type` - Authentication types (none, bearer, oauth)
- `prompt_scope` - System prompt scopes (global, organization, user)

### Tables

1. **organizations** - Multi-tenant organization records
   - Primary key: `id` (UUID)
   - Unique constraint on `slug`
   - Metadata stored as JSONB

2. **users** - Users belonging to organizations
   - Primary key: `id` (UUID)
   - Foreign key to `organizations`
   - Role-based access control (admin/user)
   - Email validation constraint

3. **user_sessions** - Active user sessions
   - Workspace isolation per session
   - Optional expiration time
   - System prompt override capability
   - Composite foreign key ensures user belongs to organization

4. **mcp_configurations** - MCP server configurations
   - Supports org-wide and user-specific configs
   - Multiple connection types (HTTP, SSE, stdio)
   - Multiple auth types (none, bearer, OAuth)
   - Encrypted bearer tokens
   - Conditional constraints based on type/auth

5. **mcp_oauth_tokens** - OAuth tokens for MCP servers
   - User-scoped OAuth credentials
   - Encrypted access and refresh tokens
   - Expiration tracking

6. **system_prompts** - Hierarchical system prompts
   - Three scopes: global, organization, user
   - Priority-based ordering
   - Optional template support
   - Enabled/disabled flag

7. **audit_logs** - Immutable audit trail
   - All actions tracked
   - Append-only (no updates/deletes)
   - Includes IP address and user agent
   - Success/failure tracking

### Row-Level Security (RLS)

All tables have RLS enabled with policies enforcing:
- Users can only view data in their organization
- Admins can manage org-level resources
- Users can manage their own user-level resources
- Audit logs are append-only

### Indexes

Performance indexes on:
- Foreign keys
- Frequently queried fields
- Organization/user scoping
- Timestamp-based queries
- Enabled/active flags

### Functions

1. `update_updated_at_column()` - Automatic timestamp updates
2. `encrypt_token(token, key)` - Token encryption using pgcrypto
3. `decrypt_token(encrypted_token, key)` - Token decryption
4. `log_audit_event(...)` - Structured audit logging
5. `cleanup_expired_sessions()` - Remove expired sessions
6. `get_effective_system_prompts(user_id, org_id)` - Get prioritized prompts

### Views

1. `active_user_sessions` - Sessions that haven't expired
2. `effective_mcp_configurations` - Enabled configs with org/user info

## Deployment

### Local Development

```bash
# Start PostgreSQL with Docker
docker-compose -f docker-compose.multitenant.yml up -d postgres

# Apply schema
psql -h localhost -U agentapi -d agentapi -f database/schema.sql
```

### Production

1. Ensure PostgreSQL 14+ with pgcrypto extension
2. Create database and application user
3. Apply schema: `psql -f database/schema.sql`
4. Configure database grants (see GRANTS section in schema.sql)
5. Set up encryption key for token encryption
6. Configure RLS by setting `app.user_id` session variable

## Usage

### Setting RLS Context

Before queries, set the current user:

```sql
SET app.user_id = 'user-uuid-here';
```

### Encrypting Tokens

```sql
UPDATE mcp_configurations
SET bearer_token = encrypt_token('my-secret-token', 'encryption-key')
WHERE id = 'config-uuid';
```

### Decrypting Tokens

```sql
SELECT decrypt_token(bearer_token, 'encryption-key')
FROM mcp_configurations
WHERE id = 'config-uuid';
```

### Logging Audit Events

```sql
SELECT log_audit_event(
    'user-uuid',
    'org-uuid',
    'CREATE',
    'mcp_configuration',
    'resource-id',
    '{"config_name": "my-mcp"}'::jsonb,
    '192.168.1.1'::inet,
    'Mozilla/5.0...'
);
```

### Getting Effective Prompts

```sql
SELECT * FROM get_effective_system_prompts(
    'user-uuid',
    'org-uuid'
);
```

## Security Considerations

1. **Encryption Keys**: Store encryption keys securely (environment variables, secrets manager)
2. **RLS Context**: Always set `app.user_id` before queries
3. **Connection Pooling**: Use connection pooling with proper RLS context switching
4. **Audit Logs**: Monitor audit logs for suspicious activity
5. **Token Rotation**: Implement regular token rotation for OAuth tokens
6. **Session Cleanup**: Run `cleanup_expired_sessions()` periodically

## Maintenance

### Regular Tasks

1. Clean up expired sessions:
   ```sql
   SELECT cleanup_expired_sessions();
   ```

2. Vacuum and analyze:
   ```sql
   VACUUM ANALYZE;
   ```

3. Review audit logs:
   ```sql
   SELECT * FROM audit_logs
   WHERE timestamp > NOW() - INTERVAL '7 days'
   ORDER BY timestamp DESC;
   ```

### Optional Setup

- Enable pg_cron for automatic session cleanup
- Set up replication for high availability
- Configure backup schedules
- Monitor table sizes and index usage


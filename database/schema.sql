-- ============================================================================
-- AgentAPI Multi-Tenant Database Schema
-- ============================================================================
-- This schema implements a complete multi-tenant architecture with:
-- - Row-Level Security (RLS) for data isolation
-- - Encrypted sensitive data (tokens)
-- - Audit logging
-- - Hierarchical permissions (org/user scopes)
-- - MCP configuration management
-- ============================================================================

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable encryption functions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- CUSTOM TYPES
-- ============================================================================

-- User roles within an organization
CREATE TYPE user_role AS ENUM ('admin', 'user');

-- MCP server connection types
CREATE TYPE mcp_type AS ENUM ('http', 'sse', 'stdio');

-- MCP authentication types
CREATE TYPE auth_type AS ENUM ('none', 'bearer', 'oauth');

-- System prompt scope levels
CREATE TYPE prompt_scope AS ENUM ('global', 'organization', 'user');

-- ============================================================================
-- TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Organizations Table
-- ----------------------------------------------------------------------------
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT organizations_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT organizations_slug_not_empty CHECK (length(trim(slug)) > 0)
);

-- ----------------------------------------------------------------------------
-- Users Table
-- ----------------------------------------------------------------------------
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'user',
    metadata JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_email_not_empty CHECK (length(trim(email)) > 0)
);

-- ----------------------------------------------------------------------------
-- User Sessions Table
-- ----------------------------------------------------------------------------
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    workspace_path TEXT NOT NULL,
    system_prompt TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_metadata JSONB DEFAULT '{}'::jsonb,

    -- Constraints
    CONSTRAINT user_sessions_workspace_not_empty CHECK (length(trim(workspace_path)) > 0),
    CONSTRAINT user_sessions_expires_after_creation CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- Create composite unique constraint for user_org_match
CREATE UNIQUE INDEX idx_users_id_organization_id ON users(id, organization_id);

-- Add foreign key constraint
ALTER TABLE user_sessions
    ADD CONSTRAINT user_sessions_user_org_match
    FOREIGN KEY (user_id, organization_id)
    REFERENCES users(id, organization_id)
    ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- MCP Configurations Table
-- ----------------------------------------------------------------------------
CREATE TABLE mcp_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type mcp_type NOT NULL,
    endpoint TEXT,
    command TEXT,
    args JSONB DEFAULT '[]'::jsonb,
    env JSONB DEFAULT '{}'::jsonb,
    auth_type auth_type NOT NULL DEFAULT 'none',
    bearer_token TEXT, -- Encrypted
    oauth_provider TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT mcp_configurations_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT mcp_configurations_unique_name UNIQUE(organization_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), name),
    CONSTRAINT mcp_configurations_http_has_endpoint CHECK (
        type != 'http' OR (endpoint IS NOT NULL AND length(trim(endpoint)) > 0)
    ),
    CONSTRAINT mcp_configurations_sse_has_endpoint CHECK (
        type != 'sse' OR (endpoint IS NOT NULL AND length(trim(endpoint)) > 0)
    ),
    CONSTRAINT mcp_configurations_stdio_has_command CHECK (
        type != 'stdio' OR (command IS NOT NULL AND length(trim(command)) > 0)
    ),
    CONSTRAINT mcp_configurations_bearer_has_token CHECK (
        auth_type != 'bearer' OR bearer_token IS NOT NULL
    ),
    CONSTRAINT mcp_configurations_oauth_has_provider CHECK (
        auth_type != 'oauth' OR oauth_provider IS NOT NULL
    )
);

-- ----------------------------------------------------------------------------
-- MCP OAuth Tokens Table
-- ----------------------------------------------------------------------------
CREATE TABLE mcp_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mcp_name TEXT NOT NULL,
    oauth_provider TEXT NOT NULL,
    access_token TEXT NOT NULL, -- Encrypted
    refresh_token TEXT, -- Encrypted
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT mcp_oauth_tokens_mcp_name_not_empty CHECK (length(trim(mcp_name)) > 0),
    CONSTRAINT mcp_oauth_tokens_provider_not_empty CHECK (length(trim(oauth_provider)) > 0),
    CONSTRAINT mcp_oauth_tokens_unique_user_mcp UNIQUE(user_id, mcp_name, oauth_provider)
);

-- Add foreign key constraint for oauth tokens
ALTER TABLE mcp_oauth_tokens
    ADD CONSTRAINT mcp_oauth_tokens_user_org_match
    FOREIGN KEY (user_id, organization_id)
    REFERENCES users(id, organization_id)
    ON DELETE CASCADE;

-- ----------------------------------------------------------------------------
-- OAuth State Table (for CSRF protection and PKCE)
-- ----------------------------------------------------------------------------
CREATE TABLE oauth_state (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL UNIQUE,
    code_verifier TEXT NOT NULL,
    provider TEXT NOT NULL,
    mcp_name TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,

    -- Constraints
    CONSTRAINT oauth_state_state_not_empty CHECK (length(trim(state)) > 0),
    CONSTRAINT oauth_state_verifier_not_empty CHECK (length(trim(code_verifier)) > 0),
    CONSTRAINT oauth_state_provider_not_empty CHECK (length(trim(provider)) > 0),
    CONSTRAINT oauth_state_mcp_name_not_empty CHECK (length(trim(mcp_name)) > 0),
    CONSTRAINT oauth_state_redirect_uri_not_empty CHECK (length(trim(redirect_uri)) > 0),
    CONSTRAINT oauth_state_expires_after_creation CHECK (expires_at > created_at)
);

-- ----------------------------------------------------------------------------
-- System Prompts Table
-- ----------------------------------------------------------------------------
CREATE TABLE system_prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scope prompt_scope NOT NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    template TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Constraints
    CONSTRAINT system_prompts_content_not_empty CHECK (length(trim(content)) > 0),
    CONSTRAINT system_prompts_global_no_refs CHECK (
        scope != 'global' OR (organization_id IS NULL AND user_id IS NULL)
    ),
    CONSTRAINT system_prompts_org_has_org_id CHECK (
        scope != 'organization' OR (organization_id IS NOT NULL AND user_id IS NULL)
    ),
    CONSTRAINT system_prompts_user_has_user_id CHECK (
        scope != 'user' OR (organization_id IS NOT NULL AND user_id IS NOT NULL)
    )
);

-- ----------------------------------------------------------------------------
-- Audit Logs Table (Immutable)
-- ----------------------------------------------------------------------------
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details JSONB DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,

    -- Constraints
    CONSTRAINT audit_logs_action_not_empty CHECK (length(trim(action)) > 0),
    CONSTRAINT audit_logs_resource_type_not_empty CHECK (length(trim(resource_type)) > 0)
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Organizations indexes
CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_created_at ON organizations(created_at DESC);

-- Users indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_organization_role ON users(organization_id, role);
CREATE INDEX idx_users_is_active ON users(is_active);

-- User sessions indexes
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_organization_id ON user_sessions(organization_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX idx_user_sessions_last_active ON user_sessions(last_active_at DESC);

-- MCP configurations indexes
CREATE INDEX idx_mcp_configs_organization_id ON mcp_configurations(organization_id);
CREATE INDEX idx_mcp_configs_user_id ON mcp_configurations(user_id);
CREATE INDEX idx_mcp_configs_org_enabled ON mcp_configurations(organization_id, enabled);
CREATE INDEX idx_mcp_configs_user_enabled ON mcp_configurations(user_id, enabled) WHERE user_id IS NOT NULL;
CREATE INDEX idx_mcp_configs_name ON mcp_configurations(name);

-- MCP OAuth tokens indexes
CREATE INDEX idx_mcp_oauth_tokens_user_id ON mcp_oauth_tokens(user_id);
CREATE INDEX idx_mcp_oauth_tokens_organization_id ON mcp_oauth_tokens(organization_id);
CREATE INDEX idx_mcp_oauth_tokens_expires_at ON mcp_oauth_tokens(expires_at);

-- OAuth state indexes
CREATE INDEX idx_oauth_state_state ON oauth_state(state);
CREATE INDEX idx_oauth_state_user_id ON oauth_state(user_id);
CREATE INDEX idx_oauth_state_expires_at ON oauth_state(expires_at);
CREATE INDEX idx_oauth_state_used ON oauth_state(used);

-- System prompts indexes
CREATE INDEX idx_system_prompts_organization_id ON system_prompts(organization_id);
CREATE INDEX idx_system_prompts_user_id ON system_prompts(user_id);
CREATE INDEX idx_system_prompts_scope ON system_prompts(scope);
CREATE INDEX idx_system_prompts_enabled ON system_prompts(enabled);
CREATE INDEX idx_system_prompts_priority ON system_prompts(priority DESC);
CREATE INDEX idx_system_prompts_org_enabled_priority ON system_prompts(organization_id, enabled, priority DESC);

-- Audit logs indexes
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_org_timestamp ON audit_logs(organization_id, timestamp DESC);

-- ============================================================================
-- ROW LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- RLS POLICIES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Organizations Policies
-- ----------------------------------------------------------------------------

-- Users can view their own organization
CREATE POLICY organizations_view_own ON organizations
    FOR SELECT
    USING (
        id IN (SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid)
    );

-- Only admins can create organizations (via application logic)
CREATE POLICY organizations_admin_insert ON organizations
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM users
            WHERE users.id = current_setting('app.user_id', true)::uuid
            AND users.role = 'admin'
        )
    );

-- Only admins can update their own organization
CREATE POLICY organizations_admin_update ON organizations
    FOR UPDATE
    USING (
        id IN (
            SELECT organization_id FROM users
            WHERE users.id = current_setting('app.user_id', true)::uuid
            AND users.role = 'admin'
        )
    );

-- No one can delete organizations (use application logic)
CREATE POLICY organizations_no_delete ON organizations
    FOR DELETE
    USING (false);

-- ----------------------------------------------------------------------------
-- Users Policies
-- ----------------------------------------------------------------------------

-- Users can view users in their organization
CREATE POLICY users_view_org ON users
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
    );

-- Only admins can create users in their organization
CREATE POLICY users_admin_insert ON users
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        )
    );

-- Admins can update users in their org, users can update themselves
CREATE POLICY users_update ON users
    FOR UPDATE
    USING (
        id = current_setting('app.user_id', true)::uuid
        OR EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        )
    );

-- Only admins can delete users in their organization
CREATE POLICY users_admin_delete ON users
    FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        )
    );

-- ----------------------------------------------------------------------------
-- User Sessions Policies
-- ----------------------------------------------------------------------------

-- Users can view their own sessions
CREATE POLICY user_sessions_view_own ON user_sessions
    FOR SELECT
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can create their own sessions
CREATE POLICY user_sessions_insert_own ON user_sessions
    FOR INSERT
    WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

-- Users can update their own sessions
CREATE POLICY user_sessions_update_own ON user_sessions
    FOR UPDATE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can delete their own sessions
CREATE POLICY user_sessions_delete_own ON user_sessions
    FOR DELETE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- ----------------------------------------------------------------------------
-- MCP Configurations Policies
-- ----------------------------------------------------------------------------

-- Users can view org-wide configs and their own user-specific configs
CREATE POLICY mcp_configs_view ON mcp_configurations
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
        AND (user_id IS NULL OR user_id = current_setting('app.user_id', true)::uuid)
    );

-- Users can create their own configs, admins can create org-wide configs
CREATE POLICY mcp_configs_insert ON mcp_configurations
    FOR INSERT
    WITH CHECK (
        -- User creating own config
        (user_id = current_setting('app.user_id', true)::uuid)
        OR
        -- Admin creating org-wide config
        (user_id IS NULL AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        ))
    );

-- Users can update their own configs, admins can update org configs
CREATE POLICY mcp_configs_update ON mcp_configurations
    FOR UPDATE
    USING (
        user_id = current_setting('app.user_id', true)::uuid
        OR (user_id IS NULL AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        ))
    );

-- Users can delete their own configs, admins can delete org configs
CREATE POLICY mcp_configs_delete ON mcp_configurations
    FOR DELETE
    USING (
        user_id = current_setting('app.user_id', true)::uuid
        OR (user_id IS NULL AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        ))
    );

-- ----------------------------------------------------------------------------
-- MCP OAuth Tokens Policies
-- ----------------------------------------------------------------------------

-- Users can only view their own OAuth tokens
CREATE POLICY mcp_oauth_tokens_view_own ON mcp_oauth_tokens
    FOR SELECT
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only create their own OAuth tokens
CREATE POLICY mcp_oauth_tokens_insert_own ON mcp_oauth_tokens
    FOR INSERT
    WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only update their own OAuth tokens
CREATE POLICY mcp_oauth_tokens_update_own ON mcp_oauth_tokens
    FOR UPDATE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only delete their own OAuth tokens
CREATE POLICY mcp_oauth_tokens_delete_own ON mcp_oauth_tokens
    FOR DELETE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- ----------------------------------------------------------------------------
-- OAuth State Policies
-- ----------------------------------------------------------------------------

-- Users can only view their own OAuth states
CREATE POLICY oauth_state_view_own ON oauth_state
    FOR SELECT
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only create their own OAuth states
CREATE POLICY oauth_state_insert_own ON oauth_state
    FOR INSERT
    WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only update their own OAuth states
CREATE POLICY oauth_state_update_own ON oauth_state
    FOR UPDATE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- Users can only delete their own OAuth states
CREATE POLICY oauth_state_delete_own ON oauth_state
    FOR DELETE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- ----------------------------------------------------------------------------
-- System Prompts Policies
-- ----------------------------------------------------------------------------

-- Users can view global, org, and their own prompts
CREATE POLICY system_prompts_view ON system_prompts
    FOR SELECT
    USING (
        scope = 'global'
        OR (scope = 'organization' AND organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        ))
        OR (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
    );

-- Admins can create global and org prompts, users can create their own
CREATE POLICY system_prompts_insert ON system_prompts
    FOR INSERT
    WITH CHECK (
        -- User creating own prompt
        (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
        OR
        -- Admin creating org or global prompt
        (scope IN ('organization', 'global') AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND (scope = 'global' OR u.organization_id = organization_id)
        ))
    );

-- Same rules for updates
CREATE POLICY system_prompts_update ON system_prompts
    FOR UPDATE
    USING (
        (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
        OR
        (scope IN ('organization', 'global') AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND (scope = 'global' OR u.organization_id = organization_id)
        ))
    );

-- Same rules for deletes
CREATE POLICY system_prompts_delete ON system_prompts
    FOR DELETE
    USING (
        (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
        OR
        (scope IN ('organization', 'global') AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND (scope = 'global' OR u.organization_id = organization_id)
        ))
    );

-- ----------------------------------------------------------------------------
-- Audit Log Policies (Append-Only)
-- ----------------------------------------------------------------------------

-- Anyone can insert audit logs (via application)
CREATE POLICY audit_logs_append_only ON audit_logs
    FOR INSERT
    WITH CHECK (true);

-- Users can view audit logs for their organization
CREATE POLICY audit_logs_view_org ON audit_logs
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
        OR organization_id IS NULL
    );

-- No updates allowed
CREATE POLICY audit_logs_no_update ON audit_logs
    FOR UPDATE
    USING (false);

-- No deletes allowed (immutable)
CREATE POLICY audit_logs_no_delete ON audit_logs
    FOR DELETE
    USING (false);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Function to update updated_at timestamp
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ----------------------------------------------------------------------------
-- Function to encrypt sensitive data
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION encrypt_token(token TEXT, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(pgp_sym_encrypt(token, key), 'base64');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ----------------------------------------------------------------------------
-- Function to decrypt sensitive data
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION decrypt_token(encrypted_token TEXT, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(decode(encrypted_token, 'base64'), key);
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ----------------------------------------------------------------------------
-- Function to log audit events
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION log_audit_event(
    p_user_id UUID,
    p_organization_id UUID,
    p_action TEXT,
    p_resource_type TEXT,
    p_resource_id TEXT DEFAULT NULL,
    p_details JSONB DEFAULT '{}'::jsonb,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_success BOOLEAN DEFAULT true,
    p_error_message TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_audit_id UUID;
BEGIN
    INSERT INTO audit_logs (
        user_id,
        organization_id,
        action,
        resource_type,
        resource_id,
        details,
        ip_address,
        user_agent,
        success,
        error_message
    ) VALUES (
        p_user_id,
        p_organization_id,
        p_action,
        p_resource_type,
        p_resource_id,
        p_details,
        p_ip_address,
        p_user_agent,
        p_success,
        p_error_message
    ) RETURNING id INTO v_audit_id;

    RETURN v_audit_id;
END;
$$ LANGUAGE plpgsql;

-- ----------------------------------------------------------------------------
-- Function to clean up expired sessions
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions
    WHERE expires_at IS NOT NULL AND expires_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ----------------------------------------------------------------------------
-- Function to get effective system prompts for a user
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION get_effective_system_prompts(
    p_user_id UUID,
    p_organization_id UUID
)
RETURNS TABLE (
    id UUID,
    scope prompt_scope,
    content TEXT,
    priority INTEGER,
    template TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        sp.id,
        sp.scope,
        sp.content,
        sp.priority,
        sp.template
    FROM system_prompts sp
    WHERE sp.enabled = true
    AND (
        -- Global prompts
        (sp.scope = 'global')
        OR
        -- Organization prompts
        (sp.scope = 'organization' AND sp.organization_id = p_organization_id)
        OR
        -- User-specific prompts
        (sp.scope = 'user' AND sp.user_id = p_user_id)
    )
    ORDER BY
        -- Order by scope priority: user > organization > global
        CASE sp.scope
            WHEN 'user' THEN 1
            WHEN 'organization' THEN 2
            WHEN 'global' THEN 3
        END,
        -- Then by explicit priority (higher first)
        sp.priority DESC,
        -- Then by creation time (newer first)
        sp.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update updated_at timestamp on updates
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mcp_configurations_updated_at BEFORE UPDATE ON mcp_configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mcp_oauth_tokens_updated_at BEFORE UPDATE ON mcp_oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_prompts_updated_at BEFORE UPDATE ON system_prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- ----------------------------------------------------------------------------
-- View for active user sessions
-- ----------------------------------------------------------------------------
CREATE OR REPLACE VIEW active_user_sessions AS
SELECT
    us.*,
    u.email,
    u.role,
    o.name as organization_name,
    o.slug as organization_slug
FROM user_sessions us
JOIN users u ON us.user_id = u.id
JOIN organizations o ON us.organization_id = o.id
WHERE (us.expires_at IS NULL OR us.expires_at > NOW());

-- ----------------------------------------------------------------------------
-- View for effective MCP configurations (org + user)
-- ----------------------------------------------------------------------------
CREATE OR REPLACE VIEW effective_mcp_configurations AS
SELECT
    mc.*,
    o.name as organization_name,
    o.slug as organization_slug,
    u.email as user_email,
    CASE
        WHEN mc.user_id IS NULL THEN 'organization'
        ELSE 'user'
    END as config_scope
FROM mcp_configurations mc
JOIN organizations o ON mc.organization_id = o.id
LEFT JOIN users u ON mc.user_id = u.id
WHERE mc.enabled = true;

-- ============================================================================
-- COMMENTS
-- ============================================================================

-- Table comments
COMMENT ON TABLE organizations IS 'Multi-tenant organizations';
COMMENT ON TABLE users IS 'Users belonging to organizations';
COMMENT ON TABLE user_sessions IS 'Active user sessions with workspace context';
COMMENT ON TABLE mcp_configurations IS 'MCP server configurations (org-wide or user-specific)';
COMMENT ON TABLE mcp_oauth_tokens IS 'OAuth tokens for MCP servers';
COMMENT ON TABLE system_prompts IS 'Hierarchical system prompts (global/org/user)';
COMMENT ON TABLE audit_logs IS 'Immutable audit trail of all actions';

-- Column comments for sensitive data
COMMENT ON COLUMN mcp_configurations.bearer_token IS 'Encrypted bearer token using pgcrypto';
COMMENT ON COLUMN mcp_oauth_tokens.access_token IS 'Encrypted OAuth access token';
COMMENT ON COLUMN mcp_oauth_tokens.refresh_token IS 'Encrypted OAuth refresh token';

-- Function comments
COMMENT ON FUNCTION encrypt_token IS 'Encrypt sensitive token data using pgcrypto';
COMMENT ON FUNCTION decrypt_token IS 'Decrypt sensitive token data using pgcrypto';
COMMENT ON FUNCTION log_audit_event IS 'Insert an audit log entry';
COMMENT ON FUNCTION cleanup_expired_sessions IS 'Delete expired user sessions';
COMMENT ON FUNCTION get_effective_system_prompts IS 'Get effective system prompts for a user in priority order';

-- ============================================================================
-- INITIAL DATA (Optional)
-- ============================================================================

-- Create a default admin organization and user for initial setup
-- Note: In production, this should be done through application logic
-- Uncomment to enable:

/*
-- Default organization
INSERT INTO organizations (id, name, slug, metadata)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'System Admin',
    'system-admin',
    '{"is_system": true}'::jsonb
);

-- Default admin user (change email and create proper authentication)
INSERT INTO users (id, email, organization_id, role, metadata)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@agentapi.local',
    '00000000-0000-0000-0000-000000000001',
    'admin',
    '{"is_system": true}'::jsonb
);

-- Default global system prompt
INSERT INTO system_prompts (scope, content, priority)
VALUES (
    'global',
    'You are a helpful AI assistant powered by AgentAPI. Respond professionally and accurately.',
    100
);
*/

-- ============================================================================
-- GRANTS (Configure based on your application user)
-- ============================================================================

-- Example grants for an application user 'agentapi_app'
-- Uncomment and modify as needed:

/*
GRANT CONNECT ON DATABASE your_database TO agentapi_app;
GRANT USAGE ON SCHEMA public TO agentapi_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO agentapi_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO agentapi_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO agentapi_app;
*/

-- ============================================================================
-- PLATFORM ADMIN TABLES
-- ============================================================================

-- Table for platform-wide admins
CREATE TABLE platform_admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workos_user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    name TEXT,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    added_by UUID REFERENCES platform_admins(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table for audit logging of admin actions
CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID NOT NULL REFERENCES platform_admins(id),
    action TEXT NOT NULL, -- 'added_admin', 'removed_admin', 'accessed_stats', etc.
    target_org_id TEXT,
    target_user_id TEXT,
    details JSONB,
    ip_address TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_platform_admins_workos_user_id ON platform_admins(workos_user_id);
CREATE INDEX idx_platform_admins_email ON platform_admins(email);
CREATE INDEX idx_platform_admins_is_active ON platform_admins(is_active);
CREATE INDEX idx_admin_audit_log_admin_id ON admin_audit_log(admin_id);
CREATE INDEX idx_admin_audit_log_action ON admin_audit_log(action);
CREATE INDEX idx_admin_audit_log_created_at ON admin_audit_log(created_at);

-- Add RLS policies for platform admin tables
ALTER TABLE platform_admins ENABLE ROW LEVEL SECURITY;
ALTER TABLE admin_audit_log ENABLE ROW LEVEL SECURITY;

-- Platform admins can read all platform admin records
CREATE POLICY platform_admins_select ON platform_admins
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can insert new platform admin records
CREATE POLICY platform_admins_insert ON platform_admins
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can update platform admin records
CREATE POLICY platform_admins_update ON platform_admins
    FOR UPDATE
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can read all audit log records
CREATE POLICY admin_audit_log_select ON admin_audit_log
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- Platform admins can insert audit log records
CREATE POLICY admin_audit_log_insert ON admin_audit_log
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- ============================================================================
-- MAINTENANCE
-- ============================================================================

-- Schedule cleanup of expired sessions (example using pg_cron extension)
-- Uncomment if you have pg_cron enabled:

/*
SELECT cron.schedule(
    'cleanup-expired-sessions',
    '0 * * * *', -- Every hour
    'SELECT cleanup_expired_sessions();'
);
*/

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

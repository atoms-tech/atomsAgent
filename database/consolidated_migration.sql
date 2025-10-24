-- ============================================================================
-- AgentAPI Consolidated Database Schema Migration
-- ============================================================================
-- This migration merges three schema files:
-- 1. database/schema.sql - Core multi-tenant schema
-- 2. database/migrations/002_oauth_tables.sql - OAuth management
-- 3. database/agent_system_schema.sql - Agent system tables
--
-- Date: October 24, 2025
-- Status: Production-ready
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
-- CORE MULTI-TENANT TABLES
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

-- ============================================================================
-- OAUTH TABLES (Enhanced version from 002_oauth_tables.sql)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- OAuth State Table (for CSRF protection and PKCE)
-- ----------------------------------------------------------------------------
CREATE TABLE oauth_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL UNIQUE,
    code_verifier TEXT NOT NULL, -- Encrypted PKCE code_verifier
    provider TEXT NOT NULL,
    mcp_name TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,

    -- Constraints
    CONSTRAINT oauth_states_state_not_empty CHECK (length(trim(state)) > 0),
    CONSTRAINT oauth_states_verifier_not_empty CHECK (length(trim(code_verifier)) > 0),
    CONSTRAINT oauth_states_provider_not_empty CHECK (length(trim(provider)) > 0),
    CONSTRAINT oauth_states_mcp_name_not_empty CHECK (length(trim(mcp_name)) > 0),
    CONSTRAINT oauth_states_redirect_uri_not_empty CHECK (length(trim(redirect_uri)) > 0),
    CONSTRAINT oauth_states_expires_after_creation CHECK (expires_at > created_at)
);

-- ----------------------------------------------------------------------------
-- MCP OAuth Tokens Table (Enhanced version)
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
    token_type TEXT DEFAULT 'Bearer',
    scope TEXT DEFAULT '',
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

-- ----------------------------------------------------------------------------
-- Platform Admin Tables
-- ----------------------------------------------------------------------------
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

CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id UUID NOT NULL REFERENCES platform_admins(id),
    action TEXT NOT NULL,
    target_org_id TEXT,
    target_user_id TEXT,
    details JSONB,
    ip_address TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- AGENT SYSTEM TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Agents Table - Stores agent configurations
-- ----------------------------------------------------------------------------
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL, -- 'ccrouter', 'droid', 'custom'
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    config JSONB, -- Agent-specific configuration
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Models Table - Available LLM models
-- ----------------------------------------------------------------------------
CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    provider VARCHAR(100), -- 'gemini', 'openrouter', etc
    model_id VARCHAR(255), -- Provider-specific model ID
    enabled BOOLEAN DEFAULT true,
    config JSONB, -- Model-specific settings (temperature, max_tokens, etc)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, name)
);

-- ----------------------------------------------------------------------------
-- Chat Sessions Table - Conversation history
-- ----------------------------------------------------------------------------
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    org_id VARCHAR(255) NOT NULL,
    model_id UUID REFERENCES models(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    title VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_message_at TIMESTAMP WITH TIME ZONE
);

-- ----------------------------------------------------------------------------
-- Chat Messages Table - Individual messages in a session
-- ----------------------------------------------------------------------------
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    tokens_in INTEGER,
    tokens_out INTEGER,
    tokens_total INTEGER,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Agent Execution History Table - Track agent responses
-- ----------------------------------------------------------------------------
CREATE TABLE agent_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    model_id UUID REFERENCES models(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL, -- 'pending', 'running', 'success', 'failed'
    input_tokens INTEGER,
    output_tokens INTEGER,
    total_tokens INTEGER,
    latency_ms INTEGER, -- Execution time in milliseconds
    error_message TEXT,
    response_content TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- ----------------------------------------------------------------------------
-- Agent Health Table - Track agent availability
-- ----------------------------------------------------------------------------
CREATE TABLE agent_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- 'healthy', 'degraded', 'unhealthy'
    last_check TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    consecutive_failures INTEGER DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
);

-- ----------------------------------------------------------------------------
-- Agent Metrics Table - Performance metrics
-- ----------------------------------------------------------------------------
CREATE TABLE agent_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_requests INTEGER DEFAULT 0,
    successful_requests INTEGER DEFAULT 0,
    failed_requests INTEGER DEFAULT 0,
    avg_latency_ms DECIMAL(10, 2),
    avg_tokens_in INTEGER,
    avg_tokens_out INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, date)
);

-- ----------------------------------------------------------------------------
-- Circuit Breaker State Table
-- ----------------------------------------------------------------------------
CREATE TABLE circuit_breaker_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL, -- 'closed', 'open', 'half_open'
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_failure_time TIMESTAMP WITH TIME ZONE,
    last_success_time TIMESTAMP WITH TIME ZONE,
    opened_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
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
CREATE INDEX idx_mcp_oauth_tokens_mcp_name ON mcp_oauth_tokens(mcp_name);

-- OAuth state indexes
CREATE INDEX idx_oauth_states_state ON oauth_states(state);
CREATE INDEX idx_oauth_states_user_id ON oauth_states(user_id);
CREATE INDEX idx_oauth_states_expires_at ON oauth_states(expires_at);
CREATE INDEX idx_oauth_states_used ON oauth_states(used);
CREATE INDEX idx_oauth_states_created_at ON oauth_states(created_at);

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

-- Platform admin indexes
CREATE INDEX idx_platform_admins_workos_user_id ON platform_admins(workos_user_id);
CREATE INDEX idx_platform_admins_email ON platform_admins(email);
CREATE INDEX idx_platform_admins_is_active ON platform_admins(is_active);
CREATE INDEX idx_admin_audit_log_admin_id ON admin_audit_log(admin_id);
CREATE INDEX idx_admin_audit_log_action ON admin_audit_log(action);
CREATE INDEX idx_admin_audit_log_created_at ON admin_audit_log(created_at);

-- Chat Sessions indexes
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_org_id ON chat_sessions(org_id);
CREATE INDEX idx_chat_sessions_created_at ON chat_sessions(created_at DESC);

-- Chat Messages indexes
CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at DESC);

-- Agent Executions indexes
CREATE INDEX idx_agent_executions_session_id ON agent_executions(session_id);
CREATE INDEX idx_agent_executions_agent_id ON agent_executions(agent_id);
CREATE INDEX idx_agent_executions_status ON agent_executions(status);
CREATE INDEX idx_agent_executions_created_at ON agent_executions(created_at DESC);

-- Models indexes
CREATE INDEX idx_models_agent_id ON models(agent_id);
CREATE INDEX idx_models_enabled ON models(enabled);

-- Agent Metrics indexes
CREATE INDEX idx_agent_metrics_agent_id ON agent_metrics(agent_id);
CREATE INDEX idx_agent_metrics_date ON agent_metrics(date DESC);

-- ============================================================================
-- ROW LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_admins ENABLE ROW LEVEL SECURITY;
ALTER TABLE admin_audit_log ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- RLS POLICIES
-- ============================================================================

-- Organizations Policies
CREATE POLICY organizations_view_own ON organizations
    FOR SELECT
    USING (
        id IN (SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid)
    );

CREATE POLICY organizations_admin_insert ON organizations
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM users
            WHERE users.id = current_setting('app.user_id', true)::uuid
            AND users.role = 'admin'
        )
    );

CREATE POLICY organizations_admin_update ON organizations
    FOR UPDATE
    USING (
        id IN (
            SELECT organization_id FROM users
            WHERE users.id = current_setting('app.user_id', true)::uuid
            AND users.role = 'admin'
        )
    );

CREATE POLICY organizations_no_delete ON organizations
    FOR DELETE
    USING (false);

-- Users Policies
CREATE POLICY users_view_org ON users
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
    );

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

-- User Sessions Policies
CREATE POLICY user_sessions_view_own ON user_sessions
    FOR SELECT
    USING (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY user_sessions_insert_own ON user_sessions
    FOR INSERT
    WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY user_sessions_update_own ON user_sessions
    FOR UPDATE
    USING (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY user_sessions_delete_own ON user_sessions
    FOR DELETE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- MCP Configurations Policies
CREATE POLICY mcp_configs_view ON mcp_configurations
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
        AND (user_id IS NULL OR user_id = current_setting('app.user_id', true)::uuid)
    );

CREATE POLICY mcp_configs_insert ON mcp_configurations
    FOR INSERT
    WITH CHECK (
        (user_id = current_setting('app.user_id', true)::uuid)
        OR
        (user_id IS NULL AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND u.organization_id = organization_id
        ))
    );

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

-- MCP OAuth Tokens Policies
CREATE POLICY mcp_oauth_tokens_view_own ON mcp_oauth_tokens
    FOR SELECT
    USING (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY mcp_oauth_tokens_insert_own ON mcp_oauth_tokens
    FOR INSERT
    WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY mcp_oauth_tokens_update_own ON mcp_oauth_tokens
    FOR UPDATE
    USING (user_id = current_setting('app.user_id', true)::uuid);

CREATE POLICY mcp_oauth_tokens_delete_own ON mcp_oauth_tokens
    FOR DELETE
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- OAuth States Policies
CREATE POLICY oauth_states_manage_own ON oauth_states
    FOR ALL
    USING (user_id = current_setting('app.user_id', true)::uuid);

-- System Prompts Policies
CREATE POLICY system_prompts_view ON system_prompts
    FOR SELECT
    USING (
        scope = 'global'
        OR (scope = 'organization' AND organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        ))
        OR (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
    );

CREATE POLICY system_prompts_insert ON system_prompts
    FOR INSERT
    WITH CHECK (
        (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
        OR
        (scope IN ('organization', 'global') AND EXISTS (
            SELECT 1 FROM users u
            WHERE u.id = current_setting('app.user_id', true)::uuid
            AND u.role = 'admin'
            AND (scope = 'global' OR u.organization_id = organization_id)
        ))
    );

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

-- Audit Log Policies
CREATE POLICY audit_logs_append_only ON audit_logs
    FOR INSERT
    WITH CHECK (true);

CREATE POLICY audit_logs_view_org ON audit_logs
    FOR SELECT
    USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
        OR organization_id IS NULL
    );

CREATE POLICY audit_logs_no_update ON audit_logs
    FOR UPDATE
    USING (false);

CREATE POLICY audit_logs_no_delete ON audit_logs
    FOR DELETE
    USING (false);

-- Platform Admins Policies
CREATE POLICY platform_admins_select ON platform_admins
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

CREATE POLICY platform_admins_insert ON platform_admins
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

CREATE POLICY platform_admins_update ON platform_admins
    FOR UPDATE
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

CREATE POLICY admin_audit_log_select ON admin_audit_log
    FOR SELECT
    USING (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

CREATE POLICY admin_audit_log_insert ON admin_audit_log
    FOR INSERT
    WITH CHECK (auth.uid()::text IN (SELECT workos_user_id FROM platform_admins WHERE is_active = true));

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to encrypt sensitive data
CREATE OR REPLACE FUNCTION encrypt_token(token TEXT, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(pgp_sym_encrypt(token, key), 'base64');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to decrypt sensitive data
CREATE OR REPLACE FUNCTION decrypt_token(encrypted_token TEXT, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(decode(encrypted_token, 'base64'), key);
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to log audit events
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

-- Function to clean up expired sessions
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

-- Function to get effective system prompts for a user
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
        (sp.scope = 'global')
        OR
        (sp.scope = 'organization' AND sp.organization_id = p_organization_id)
        OR
        (sp.scope = 'user' AND sp.user_id = p_user_id)
    )
    ORDER BY
        CASE sp.scope
            WHEN 'user' THEN 1
            WHEN 'organization' THEN 2
            WHEN 'global' THEN 3
        END,
        sp.priority DESC,
        sp.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired OAuth states
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states()
RETURNS void AS $$
BEGIN
    DELETE FROM oauth_states
    WHERE created_at < NOW() - INTERVAL '5 minutes';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to clean up expired OAuth tokens
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM mcp_oauth_tokens
    WHERE expires_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

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

-- Active user sessions view
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

-- Effective MCP configurations view
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

-- Recent sessions with latest messages
CREATE OR REPLACE VIEW v_recent_sessions AS
SELECT
    cs.id,
    cs.user_id,
    cs.org_id,
    cs.title,
    m.name as model_name,
    a.name as agent_name,
    cs.created_at,
    cs.updated_at,
    COUNT(cm.id) as message_count
FROM chat_sessions cs
LEFT JOIN models m ON cs.model_id = m.id
LEFT JOIN agents a ON cs.agent_id = a.id
LEFT JOIN chat_messages cm ON cs.id = cm.session_id
GROUP BY cs.id, m.name, a.name;

-- Agent status summary
CREATE OR REPLACE VIEW v_agent_status AS
SELECT
    a.id,
    a.name,
    a.type,
    a.enabled,
    ah.status as health_status,
    COUNT(DISTINCT m.id) as model_count,
    COALESCE(am.successful_requests, 0) as requests_today,
    COALESCE(am.avg_latency_ms, 0) as avg_latency_ms
FROM agents a
LEFT JOIN agent_health ah ON a.id = ah.agent_id
LEFT JOIN models m ON a.id = m.agent_id AND m.enabled = true
LEFT JOIN agent_metrics am ON a.id = am.agent_id AND am.date = CURRENT_DATE
GROUP BY a.id, ah.status, am.successful_requests, am.avg_latency_ms;

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Insert default agents
INSERT INTO agents (name, type, description, enabled, config)
VALUES
    ('ccrouter', 'ccrouter', 'VertexAI/Gemini routing agent', true, '{"provider": "vertex-ai", "location": "us-central1"}'),
    ('droid', 'droid', 'Multi-model Droid agent via OpenRouter', true, '{"provider": "openrouter"}')
ON CONFLICT (name) DO NOTHING;

-- Insert default models for CCRouter (VertexAI)
INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gemini-1.5-pro', 'Gemini 1.5 Pro', 'Latest Google Gemini model', 'gemini', 'gemini-1.5-pro', true
FROM agents WHERE name = 'ccrouter'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gemini-1.5-flash', 'Gemini 1.5 Flash', 'Fast Google Gemini model', 'gemini', 'gemini-1.5-flash', true
FROM agents WHERE name = 'ccrouter'
ON CONFLICT (agent_id, name) DO NOTHING;

-- Insert default models for Droid (OpenRouter)
INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'claude-3-opus', 'Claude 3 Opus', 'Anthropic Claude 3 Opus', 'openrouter', 'anthropic/claude-3-opus', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gpt-4', 'GPT-4', 'OpenAI GPT-4', 'openrouter', 'openai/gpt-4', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE organizations IS 'Multi-tenant organizations';
COMMENT ON TABLE users IS 'Users belonging to organizations';
COMMENT ON TABLE user_sessions IS 'Active user sessions with workspace context';
COMMENT ON TABLE mcp_configurations IS 'MCP server configurations (org-wide or user-specific)';
COMMENT ON TABLE mcp_oauth_tokens IS 'OAuth tokens for MCP servers';
COMMENT ON TABLE oauth_states IS 'Temporary storage for OAuth state parameters (CSRF protection)';
COMMENT ON TABLE system_prompts IS 'Hierarchical system prompts (global/org/user)';
COMMENT ON TABLE audit_logs IS 'Immutable audit trail of all actions';
COMMENT ON TABLE agents IS 'Agent configurations for CCRouter and Droid';
COMMENT ON TABLE models IS 'Available LLM models per agent';
COMMENT ON TABLE chat_sessions IS 'User conversation sessions';
COMMENT ON TABLE chat_messages IS 'Individual messages in chat sessions';
COMMENT ON TABLE agent_executions IS 'Agent execution history and metrics';
COMMENT ON TABLE agent_health IS 'Agent health status tracking';
COMMENT ON TABLE agent_metrics IS 'Daily agent performance metrics';
COMMENT ON TABLE circuit_breaker_state IS 'Circuit breaker state for fault tolerance';

-- ============================================================================
-- END OF CONSOLIDATED MIGRATION
-- ============================================================================

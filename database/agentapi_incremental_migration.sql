-- ============================================================================
-- AgentAPI Incremental Migration
-- ============================================================================
-- This migration adds AgentAPI-specific tables to an existing Supabase database
-- that already contains: organizations, platform_admins, admin_audit_log, audit_logs
--
-- Date: October 24, 2025
-- Status: Production-ready
-- Target: Supabase project ydogoylwenufckscqijp
-- ============================================================================

-- ============================================================================
-- EXTENSIONS (Safe to run if already exist)
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- CUSTOM TYPES (Only create if they don't exist)
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('admin', 'user');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE mcp_type AS ENUM ('http', 'sse', 'stdio');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE auth_type AS ENUM ('none', 'bearer', 'oauth');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE prompt_scope AS ENUM ('global', 'organization', 'user');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ============================================================================
-- AGENTAPI CORE TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Users Table (references existing organizations table)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'user',
    metadata JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,

    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_email_not_empty CHECK (length(trim(email)) > 0)
);

-- ----------------------------------------------------------------------------
-- User Sessions Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    workspace_path TEXT NOT NULL,
    system_prompt TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT user_sessions_workspace_not_empty CHECK (length(trim(workspace_path)) > 0),
    CONSTRAINT user_sessions_expires_after_creation CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- Create composite unique constraint for user_org_match
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_id_organization_id ON users(id, organization_id);

-- Add foreign key constraint
DO $$ BEGIN
    ALTER TABLE user_sessions
        ADD CONSTRAINT user_sessions_user_org_match
        FOREIGN KEY (user_id, organization_id)
        REFERENCES users(id, organization_id)
        ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ----------------------------------------------------------------------------
-- MCP Configurations Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mcp_configurations (
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
    bearer_token TEXT,
    oauth_provider TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

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
-- OAUTH TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- OAuth State Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oauth_states (
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

    CONSTRAINT oauth_states_state_not_empty CHECK (length(trim(state)) > 0),
    CONSTRAINT oauth_states_verifier_not_empty CHECK (length(trim(code_verifier)) > 0),
    CONSTRAINT oauth_states_provider_not_empty CHECK (length(trim(provider)) > 0),
    CONSTRAINT oauth_states_mcp_name_not_empty CHECK (length(trim(mcp_name)) > 0),
    CONSTRAINT oauth_states_redirect_uri_not_empty CHECK (length(trim(redirect_uri)) > 0),
    CONSTRAINT oauth_states_expires_after_creation CHECK (expires_at > created_at)
);

-- ----------------------------------------------------------------------------
-- MCP OAuth Tokens Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mcp_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mcp_name TEXT NOT NULL,
    oauth_provider TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ,
    token_type TEXT DEFAULT 'Bearer',
    scope TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT mcp_oauth_tokens_mcp_name_not_empty CHECK (length(trim(mcp_name)) > 0),
    CONSTRAINT mcp_oauth_tokens_provider_not_empty CHECK (length(trim(oauth_provider)) > 0),
    CONSTRAINT mcp_oauth_tokens_unique_user_mcp UNIQUE(user_id, mcp_name, oauth_provider)
);

-- Add foreign key constraint for oauth tokens
DO $$ BEGIN
    ALTER TABLE mcp_oauth_tokens
        ADD CONSTRAINT mcp_oauth_tokens_user_org_match
        FOREIGN KEY (user_id, organization_id)
        REFERENCES users(id, organization_id)
        ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- ----------------------------------------------------------------------------
-- System Prompts Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS system_prompts (
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

-- ============================================================================
-- AGENT SYSTEM TABLES
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Agents Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Models Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    provider VARCHAR(100),
    model_id VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, name)
);

-- ----------------------------------------------------------------------------
-- Chat Sessions Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chat_sessions (
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
-- Chat Messages Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    tokens_in INTEGER,
    tokens_out INTEGER,
    tokens_total INTEGER,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ----------------------------------------------------------------------------
-- Agent Execution History Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    model_id UUID REFERENCES models(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL,
    input_tokens INTEGER,
    output_tokens INTEGER,
    total_tokens INTEGER,
    latency_ms INTEGER,
    error_message TEXT,
    response_content TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- ----------------------------------------------------------------------------
-- Agent Health Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    last_check TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    consecutive_failures INTEGER DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
);

-- ----------------------------------------------------------------------------
-- Agent Metrics Table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_metrics (
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
CREATE TABLE IF NOT EXISTS circuit_breaker_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL,
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
-- INDEXES
-- ============================================================================

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_organization_role ON users(organization_id, role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- User sessions indexes
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_organization_id ON user_sessions(organization_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_last_active ON user_sessions(last_active_at DESC);

-- MCP configurations indexes
CREATE INDEX IF NOT EXISTS idx_mcp_configs_organization_id ON mcp_configurations(organization_id);
CREATE INDEX IF NOT EXISTS idx_mcp_configs_user_id ON mcp_configurations(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_configs_org_enabled ON mcp_configurations(organization_id, enabled);
CREATE INDEX IF NOT EXISTS idx_mcp_configs_user_enabled ON mcp_configurations(user_id, enabled) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_mcp_configs_name ON mcp_configurations(name);

-- OAuth tokens indexes
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_user_id ON mcp_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_organization_id ON mcp_oauth_tokens(organization_id);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_expires_at ON mcp_oauth_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_mcp_name ON mcp_oauth_tokens(mcp_name);

-- OAuth states indexes
CREATE INDEX IF NOT EXISTS idx_oauth_states_state ON oauth_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth_states_user_id ON oauth_states(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_states_used ON oauth_states(used);
CREATE INDEX IF NOT EXISTS idx_oauth_states_created_at ON oauth_states(created_at);

-- System prompts indexes
CREATE INDEX IF NOT EXISTS idx_system_prompts_organization_id ON system_prompts(organization_id);
CREATE INDEX IF NOT EXISTS idx_system_prompts_user_id ON system_prompts(user_id);
CREATE INDEX IF NOT EXISTS idx_system_prompts_scope ON system_prompts(scope);
CREATE INDEX IF NOT EXISTS idx_system_prompts_enabled ON system_prompts(enabled);
CREATE INDEX IF NOT EXISTS idx_system_prompts_priority ON system_prompts(priority DESC);
CREATE INDEX IF NOT EXISTS idx_system_prompts_org_enabled_priority ON system_prompts(organization_id, enabled, priority DESC);

-- Chat sessions indexes
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_org_id ON chat_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_created_at ON chat_sessions(created_at DESC);

-- Chat messages indexes
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at DESC);

-- Agent executions indexes
CREATE INDEX IF NOT EXISTS idx_agent_executions_session_id ON agent_executions(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_id ON agent_executions(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_status ON agent_executions(status);
CREATE INDEX IF NOT EXISTS idx_agent_executions_created_at ON agent_executions(created_at DESC);

-- Models indexes
CREATE INDEX IF NOT EXISTS idx_models_agent_id ON models(agent_id);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);

-- Agent metrics indexes
CREATE INDEX IF NOT EXISTS idx_agent_metrics_agent_id ON agent_metrics(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_metrics_date ON agent_metrics(date DESC);

-- ============================================================================
-- RLS POLICIES (Enable RLS on new tables only)
-- ============================================================================

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;

-- Users policies
DO $$ BEGIN
    CREATE POLICY users_view_org ON users FOR SELECT USING (
        organization_id IN (
            SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid
        )
    );
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- User sessions policies
DO $$ BEGIN
    CREATE POLICY user_sessions_view_own ON user_sessions FOR SELECT USING (user_id = current_setting('app.user_id', true)::uuid);
EXCEPTION WHEN duplicate_object THEN null; END $$;

DO $$ BEGIN
    CREATE POLICY user_sessions_insert_own ON user_sessions FOR INSERT WITH CHECK (user_id = current_setting('app.user_id', true)::uuid);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- MCP configurations policies
DO $$ BEGIN
    CREATE POLICY mcp_configs_view ON mcp_configurations FOR SELECT USING (
        organization_id IN (SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid)
        AND (user_id IS NULL OR user_id = current_setting('app.user_id', true)::uuid)
    );
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- OAuth tokens policies
DO $$ BEGIN
    CREATE POLICY mcp_oauth_tokens_view_own ON mcp_oauth_tokens FOR SELECT USING (user_id = current_setting('app.user_id', true)::uuid);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- OAuth states policies
DO $$ BEGIN
    CREATE POLICY oauth_states_manage_own ON oauth_states FOR ALL USING (user_id = current_setting('app.user_id', true)::uuid);
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- System prompts policies
DO $$ BEGIN
    CREATE POLICY system_prompts_view ON system_prompts FOR SELECT USING (
        scope = 'global'
        OR (scope = 'organization' AND organization_id IN (SELECT organization_id FROM users WHERE users.id = current_setting('app.user_id', true)::uuid))
        OR (scope = 'user' AND user_id = current_setting('app.user_id', true)::uuid)
    );
EXCEPTION WHEN duplicate_object THEN null; END $$;

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions WHERE expires_at IS NOT NULL AND expires_at < NOW();
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states()
RETURNS void AS $$
BEGIN
    DELETE FROM oauth_states WHERE created_at < NOW() - INTERVAL '5 minutes';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION cleanup_expired_oauth_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM mcp_oauth_tokens WHERE expires_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_mcp_configurations_updated_at ON mcp_configurations;
CREATE TRIGGER update_mcp_configurations_updated_at BEFORE UPDATE ON mcp_configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_mcp_oauth_tokens_updated_at ON mcp_oauth_tokens;
CREATE TRIGGER update_mcp_oauth_tokens_updated_at BEFORE UPDATE ON mcp_oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_system_prompts_updated_at ON system_prompts;
CREATE TRIGGER update_system_prompts_updated_at BEFORE UPDATE ON system_prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS
-- ============================================================================

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

INSERT INTO agents (name, type, description, enabled, config)
VALUES
    ('ccrouter', 'ccrouter', 'VertexAI/Gemini routing agent', true, '{"provider": "vertex-ai", "location": "us-central1"}'),
    ('droid', 'droid', 'Multi-model Droid agent via OpenRouter', true, '{"provider": "openrouter"}')
ON CONFLICT (name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gemini-1.5-pro', 'Gemini 1.5 Pro', 'Latest Google Gemini model', 'gemini', 'gemini-1.5-pro', true
FROM agents WHERE name = 'ccrouter'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gemini-1.5-flash', 'Gemini 1.5 Flash', 'Fast Google Gemini model', 'gemini', 'gemini-1.5-flash', true
FROM agents WHERE name = 'ccrouter'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'claude-3-opus', 'Claude 3 Opus', 'Anthropic Claude 3 Opus', 'openrouter', 'anthropic/claude-3-opus', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gpt-4', 'GPT-4', 'OpenAI GPT-4', 'openrouter', 'openai/gpt-4', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

-- ============================================================================
-- END OF INCREMENTAL MIGRATION
-- ============================================================================

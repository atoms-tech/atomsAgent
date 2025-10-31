-- ============================================================================
-- COMPLETE DATABASE FIX FOR CHATSERVER
-- Run this after running inspection script to see what's missing
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. USERS TABLE - Core user authentication and profiles
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    org_id TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Constraints
    CONSTRAINT users_org_id_check CHECK (org_id IS NOT NULL AND org_id != '')
);

-- 2. API_KEYS TABLE - For API key management
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    user_id TEXT NOT NULL,
    org_id TEXT NOT NULL,
    name TEXT NOT NULL,
    permissions JSONB DEFAULT '[]'::jsonb,
    rate_limit INTEGER DEFAULT 100,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT api_keys_user_id_check CHECK (user_id IS NOT NULL),
    CONSTRAINT api_keys_org_id_check CHECK (org_id IS NOT NULL),
    CONSTRAINT api_keys_rate_limit_check CHECK (rate_limit > 0)
);

-- 3. USER_SETTINGS TABLE - User preferences and model settings
CREATE TABLE IF NOT EXISTS user_settings (
    user_id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL,
    default_model TEXT DEFAULT 'claude-4.5-haiku',
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4000,
    top_p DECIMAL(3,2) DEFAULT 1.0,
    preferences JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT user_settings_org_id_check CHECK (org_id IS NOT NULL),
    CONSTRAINT user_settings_temp_check CHECK (temperature >= 0.0 AND temperature <= 2.0),
    CONSTRAINT user_settings_tokens_check CHECK (max_tokens > 0 AND max_tokens <= 8192),
    CONSTRAINT user_settings_top_p_check CHECK (top_p >= 0.0 AND top_p <= 1.0)
);

-- 4. SYSTEM_PROMPTS TABLE - Custom and system prompts
CREATE TABLE IF NOT EXISTS system_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    org_id TEXT NOT NULL,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    category TEXT DEFAULT 'custom',
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Constraints
    CONSTRAINT system_prompts_unique_user_name UNIQUE(user_id, name),
    CONSTRAINT system_prompts_org_id_check CHECK (org_id IS NOT NULL)
);

-- 5. MCP_CONFIGURATIONS TABLE - MCP server configurations
CREATE TABLE IF NOT EXISTS mcp_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    org_id TEXT NOT NULL,
    name TEXT NOT NULL,
    mcp_type TEXT NOT NULL,
    config JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT mcp_configs_unique_user_name UNIQUE(user_id, name),
    CONSTRAINT mcp_configs_type_check CHECK (mcp_type IN ('http', 'sse', 'stdio', 'fastmcp'))
);

-- 6. AGENTS TABLE - Available agent configurations
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    backend_type TEXT,
    config JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT agents_type_check CHECK (type IN ('claude', 'ccrouter', 'droid', 'vertexai'))
);

-- 7. AUDIT_LOGS TABLE - Security and access auditing
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id TEXT NOT NULL,
    org_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    request_id TEXT,
    status_code INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT audit_logs_action_check CHECK (action IN ('created', 'updated', 'deleted', 'accessed', 'failed', 'login', 'logout'))
);

-- 8. CHAT_SESSIONS TABLE - Active chat session tracking
CREATE TABLE IF NOT EXISTS chat_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    org_id TEXT,
    agent_type TEXT,
    title TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    message_count INTEGER DEFAULT 0,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    tokens_total INTEGER DEFAULT 0,
    archived BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    message_index INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    tokens_in INTEGER,
    tokens_out INTEGER,
    tokens_total INTEGER,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_messages_session_index ON chat_messages(session_id, message_index);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at);

-- API Keys indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);

-- User Settings indexes
CREATE INDEX IF NOT EXISTS idx_user_settings_org_id ON user_settings(org_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_default_model ON user_settings(default_model);

-- System Prompts indexes
CREATE INDEX IF NOT EXISTS idx_system_prompts_user_id ON system_prompts(user_id);
CREATE INDEX IF NOT EXISTS idx_system_prompts_org_id ON system_prompts(org_id);
CREATE INDEX IF NOT EXISTS idx_system_prompts_category ON system_prompts(category);
CREATE INDEX IF NOT EXISTS idx_system_prompts_public ON system_prompts(is_public);

-- MCP Configurations indexes
CREATE INDEX IF NOT EXISTS idx_mcp_configs_user_id ON mcp_configurations(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_configs_org_id ON mcp_configurations(org_id);
CREATE INDEX IF NOT EXISTS idx_mcp_configs_active ON mcp_configurations(is_active);

-- Audit Logs indexes
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);

-- Chat Sessions indexes
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_org_id ON chat_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_last_message ON chat_sessions(last_message_at DESC NULLS LAST);

-- ============================================================================
-- ENABLE ROW LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE chat_sessions ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- RLS POLICIES
-- ============================================================================

-- Users policies
CREATE POLICY "Users can view their own profile" ON users
    FOR SELECT USING (id = current_setting('app.current_user_id', true));

CREATE POLICY "Users can insert their own profile" ON users
    FOR INSERT WITH CHECK (id = current_setting('app.current_user_id', true));

CREATE POLICY "Users can update their own profile" ON users
    FOR UPDATE USING (id = current_setting('app.current_user_id', true));

-- API Keys policies
CREATE POLICY "Users can view their own API keys" ON api_keys
    FOR SELECT USING (auth.uid()::text = user_id);

CREATE POLICY "Users can insert their own API keys" ON api_keys
    FOR INSERT WITH CHECK (auth.uid()::text = user_id);

CREATE POLICY "Users can update their own API keys" ON api_keys
    FOR UPDATE USING (auth.uid()::text = user_id);

CREATE POLICY "Users can delete their own API keys" ON api_keys
    FOR DELETE USING (auth.uid()::text = user_id);

-- User Settings policies
CREATE POLICY "Users can view their own settings" ON user_settings
    FOR SELECT USING (user_id = current_setting('app.current_user_id', true));

CREATE POLICY "Users can update their own settings" ON user_settings
    FOR UPDATE USING (user_id = current_setting('app.current_user_id', true));

CREATE POLICY "Users can insert their own settings" ON user_settings
    FOR INSERT WITH CHECK (user_id = current_setting('app.current_user_id', true));

-- System Prompts policies
CREATE POLICY "Users can view their own prompts" ON system_prompts
    FOR SELECT USING (auth.uid()::text = user_id);

CREATE POLICY "Users can insert their own prompts" ON system_prompts
    FOR INSERT WITH CHECK (auth.uid()::text = user_id);

CREATE POLICY "Users can update their own prompts" ON system_prompts
    FOR UPDATE USING (auth.uid()::text = user_id);

CREATE POLICY "Users can delete their own prompts" ON system_prompts
    FOR DELETE USING (auth.uid()::text = user_id);

-- Public prompts can be viewed by anyone in organization
CREATE POLICY "Public prompts visible to org" ON system_prompts
    FOR SELECT USING (is_public = true AND org_id = current_setting('app.current_org_id', true));

-- MCP Configurations policies
CREATE POLICY "Users can view their own MCP configs" ON mcp_configurations
    FOR SELECT USING (auth.uid()::text = user_id);

CREATE POLICY "Users can manage their own MCP configs" ON mcp_configurations
    FOR ALL USING (auth.uid()::text = user_id);

-- Service role policies (for admin operations)
CREATE POLICY "Service role full access to users" ON users
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access to api_keys" ON api_keys
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access to user_settings" ON user_settings
    FOR ALL USING (auth.role() = 'service_role');

CREATE POLICY "Service role full access to audit_logs" ON audit_logs
    FOR ALL USING (auth.role() = 'service_role');

-- ============================================================================
-- VERIFICATION
-- ============================================================================

SELECT 'Database fix completed successfully!' as result,
       now() as completion_time;

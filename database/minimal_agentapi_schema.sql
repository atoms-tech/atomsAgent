-- ============================================================================
-- AgentAPI Minimal Schema Migration
-- ============================================================================
-- Only 5 tables needed - leverages existing Supabase infrastructure
--
-- Existing tables we reuse:
-- - profiles (for user identification)
-- - organizations (for org management)
-- - organization_members (for user-org relationships)
-- - mcp_sessions (for session storage)
--
-- Date: October 24, 2025
-- Status: Production-ready and minimal
-- Target: Supabase project ydogoylwenufckscqijp
-- ============================================================================

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. AGENTS TABLE - Agent configurations
-- ============================================================================

CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,  -- 'ccrouter', 'droid', 'custom'
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    config JSONB,  -- Provider-specific config: {provider, location, api_key, etc}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_enabled ON agents(enabled);
CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);

-- ============================================================================
-- 2. MODELS TABLE - Available LLM models per agent
-- ============================================================================

CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,  -- e.g., 'gemini-1.5-pro', 'claude-3-opus'
    display_name VARCHAR(255),   -- e.g., 'Gemini 1.5 Pro'
    description TEXT,
    provider VARCHAR(100),        -- e.g., 'gemini', 'openrouter', 'openai'
    model_id VARCHAR(255),        -- Provider-specific ID
    enabled BOOLEAN DEFAULT true,
    config JSONB,                 -- Model-specific settings: {temperature, max_tokens, etc}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, name)
);

CREATE INDEX IF NOT EXISTS idx_models_agent_id ON models(agent_id);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);
CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider);

-- ============================================================================
-- 3. CHAT_SESSIONS TABLE - Conversation sessions
-- ============================================================================

CREATE TABLE IF NOT EXISTS chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    model_id UUID REFERENCES models(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    title VARCHAR(255),
    metadata JSONB,  -- Session metadata: {system_prompt, temperature, etc}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_message_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_org_id ON chat_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_agent_id ON chat_sessions(agent_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_created_at ON chat_sessions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_org ON chat_sessions(user_id, org_id);

-- ============================================================================
-- 4. CHAT_MESSAGES TABLE - Individual messages in a session
-- ============================================================================

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,   -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    tokens_in INTEGER,           -- Input tokens consumed
    tokens_out INTEGER,          -- Output tokens generated
    tokens_total INTEGER,        -- Total tokens
    metadata JSONB,              -- Message metadata: {model_used, latency, etc}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_role ON chat_messages(role);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at DESC);

-- ============================================================================
-- 5. AGENT_HEALTH TABLE - Agent status tracking (optional, can use Redis)
-- ============================================================================

CREATE TABLE IF NOT EXISTS agent_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL UNIQUE REFERENCES agents(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,  -- 'healthy', 'degraded', 'unhealthy'
    last_check TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    consecutive_failures INTEGER DEFAULT 0,
    metadata JSONB,               -- Additional health data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_health_status ON agent_health(status);
CREATE INDEX IF NOT EXISTS idx_agent_health_updated_at ON agent_health(updated_at DESC);

-- ============================================================================
-- TRIGGERS FOR AUTO-UPDATING TIMESTAMPS
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_models_updated_at ON models;
CREATE TRIGGER update_models_updated_at BEFORE UPDATE ON models
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_chat_sessions_updated_at ON chat_sessions;
CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_agent_health_updated_at ON agent_health;
CREATE TRIGGER update_agent_health_updated_at BEFORE UPDATE ON agent_health
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

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
    COUNT(cm.id) as message_count,
    MAX(cm.created_at) as last_message_at
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
    ah.last_check,
    ah.consecutive_failures,
    COUNT(DISTINCT m.id) as model_count
FROM agents a
LEFT JOIN agent_health ah ON a.id = ah.agent_id
LEFT JOIN models m ON a.id = m.agent_id AND m.enabled = true
GROUP BY a.id, ah.status, ah.last_check, ah.consecutive_failures;

-- ============================================================================
-- INITIAL DATA - Seed Default Agents and Models
-- ============================================================================

INSERT INTO agents (name, type, description, enabled, config)
VALUES
    ('ccrouter', 'ccrouter', 'VertexAI/Gemini routing agent', true,
     '{"provider": "vertex-ai", "location": "us-central1", "use_application_default": true}'),
    ('droid', 'droid', 'Multi-model Droid agent via OpenRouter', true,
     '{"provider": "openrouter"}')
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

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gemini-2.0-pro', 'Gemini 2.0 Pro', 'Advanced Google Gemini model', 'gemini', 'gemini-2.0-pro', true
FROM agents WHERE name = 'ccrouter'
ON CONFLICT (agent_id, name) DO NOTHING;

-- Insert default models for Droid (OpenRouter)
INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'claude-3-opus', 'Claude 3 Opus', 'Anthropic Claude 3 Opus', 'openrouter', 'anthropic/claude-3-opus', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'claude-3-5-sonnet', 'Claude 3.5 Sonnet', 'Anthropic Claude 3.5 Sonnet', 'openrouter', 'anthropic/claude-3.5-sonnet', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gpt-4-turbo', 'GPT-4 Turbo', 'OpenAI GPT-4 Turbo', 'openrouter', 'openai/gpt-4-turbo', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

INSERT INTO models (agent_id, name, display_name, description, provider, model_id, enabled)
SELECT id, 'gpt-4o', 'GPT-4o', 'OpenAI GPT-4o', 'openrouter', 'openai/gpt-4o', true
FROM agents WHERE name = 'droid'
ON CONFLICT (agent_id, name) DO NOTHING;

-- Initialize health status for each agent
INSERT INTO agent_health (agent_id, status, last_check, consecutive_failures)
SELECT id, 'healthy', NOW(), 0 FROM agents
ON CONFLICT (agent_id) DO NOTHING;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE agents IS 'Agent configurations - CCRouter (VertexAI) and Droid (OpenRouter)';
COMMENT ON TABLE models IS 'Available LLM models per agent';
COMMENT ON TABLE chat_sessions IS 'Chat conversation sessions - references profiles and organizations';
COMMENT ON TABLE chat_messages IS 'Individual messages in chat sessions';
COMMENT ON TABLE agent_health IS 'Agent health status tracking - can be replaced with Redis for ephemeral data';

COMMENT ON COLUMN agents.config IS 'Provider-specific configuration: {provider, location, api_key}';
COMMENT ON COLUMN models.config IS 'Model-specific settings: {temperature, max_tokens, top_p}';
COMMENT ON COLUMN chat_sessions.metadata IS 'Session metadata: {system_prompt, temperature, max_tokens}';
COMMENT ON COLUMN chat_messages.metadata IS 'Message metadata: {model_used, latency_ms, cost}';

-- ============================================================================
-- END OF MINIMAL AGENTAPI SCHEMA
-- ============================================================================

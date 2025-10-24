-- ============================================================================
-- AgentAPI Agent System Schema
-- ============================================================================
-- Tables for managing agents, models, and agent execution state
-- ============================================================================

-- ============================================================================
-- 1. Agents Table - Stores agent configurations
-- ============================================================================
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL, -- 'ccrouter', 'droid', 'custom'
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    config JSONB, -- Agent-specific configuration
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 2. Models Table - Available LLM models
-- ============================================================================
CREATE TABLE IF NOT EXISTS models (
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

-- ============================================================================
-- 3. Chat Sessions Table - Conversation history
-- ============================================================================
CREATE TABLE IF NOT EXISTS chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    org_id VARCHAR(255) NOT NULL,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE SET NULL,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE SET NULL,
    title VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_message_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- 4. Chat Messages Table - Individual messages in a session
-- ============================================================================
CREATE TABLE IF NOT EXISTS chat_messages (
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

-- ============================================================================
-- 5. Agent Execution History Table - Track agent responses
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE SET NULL,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE SET NULL,
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

-- ============================================================================
-- 6. Agent Health Table - Track agent availability
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_health (
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

-- ============================================================================
-- 7. Agent Metrics Table - Performance metrics
-- ============================================================================
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

-- ============================================================================
-- 8. Circuit Breaker State Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS circuit_breaker_state (
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
-- Indexes for Performance
-- ============================================================================

-- Chat Sessions indexes
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_org_id ON chat_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_created_at ON chat_sessions(created_at DESC);

-- Chat Messages indexes
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at DESC);

-- Agent Executions indexes
CREATE INDEX IF NOT EXISTS idx_agent_executions_session_id ON agent_executions(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_id ON agent_executions(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_status ON agent_executions(status);
CREATE INDEX IF NOT EXISTS idx_agent_executions_created_at ON agent_executions(created_at DESC);

-- Models indexes
CREATE INDEX IF NOT EXISTS idx_models_agent_id ON models(agent_id);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);

-- Agent Metrics indexes
CREATE INDEX IF NOT EXISTS idx_agent_metrics_agent_id ON agent_metrics(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_metrics_date ON agent_metrics(date DESC);

-- ============================================================================
-- Initial Data - Default Agents
-- ============================================================================

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
-- Views for Common Queries
-- ============================================================================

-- View: Recent sessions with latest messages
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

-- View: Agent status summary
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
-- End of Schema
-- ============================================================================

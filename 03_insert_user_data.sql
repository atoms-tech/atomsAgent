-- ============================================================================
-- USER DATA INSERTION SCRIPT
-- Run this after creating tables
-- ============================================================================

-- 1. Insert the user from your logs
INSERT INTO users (
    id,
    email,
    org_id,
    first_name,
    last_name,
    metadata
) VALUES (
    'user_01K6EV07KR2MNMDQ60BC03ZM1A',
    'kooshapari@example.com',  -- Replace with actual email
    'default-org',
    'Kush',
    'Pari',
    '{
        "source": "chatserver_logs",
        "created_at": "2025-10-26",
        "tier": "pro"
    }'::jsonb
) ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    org_id = EXCLUDED.org_id,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    updated_at = NOW(),
    metadata = users.metadata || EXCLUDED.metadata;

-- 2. Insert default user settings
INSERT INTO user_settings (
    user_id,
    org_id,
    default_model,
    temperature,
    max_tokens,
    top_p,
    preferences
) VALUES (
    'user_01K6EV07KR2MNMDQ60BC03ZM1A',
    'default-org',
    'claude-4.5-haiku',
    0.7,
    4000,
    1.0,
    '{
        "theme": "dark",
        "notifications": true,
        "auto_save": true,
        "show_line_numbers": true,
        "code_highlighting": true
    }'::jsonb
) ON CONFLICT (user_id) DO UPDATE SET
    default_model = EXCLUDED.default_model,
    temperature = EXCLUDED.temperature,
    max_tokens = EXCLUDED.max_tokens,
    top_p = EXCLUDED.top_p,
    preferences = user_settings.preferences || EXCLUDED.preferences,
    updated_at = NOW();

-- 3. Insert sample API key (if needed)
INSERT INTO api_keys (
    key_hash,
    key_prefix,
    user_id,
    org_id,
    name,
    permissions,
    rate_limit,
    is_active
) VALUES (
    'sha256:sample_hash_for_testing_only',
    'sk_test_',
    'user_01K6EV07KR2MNMDQ60BC03ZM1A',
    'default-org',
    'Development API Key',
    '["read", "write"]'::jsonb,
    100,
    true
) ON CONFLICT (key_hash) DO NOTHING;

-- 4. Insert default system prompts
INSERT INTO system_prompts (
    user_id,
    org_id,
    name,
    content,
    category,
    is_public,
    metadata
) VALUES 
    ('user_01K6EV07KR2MNMDQ60BC03ZM1A', 'default-org', 'Default Assistant', 
     'You are a helpful AI assistant. Be concise and accurate.', 
     'system', false, '{"purpose": "default", "language": "en"}'::jsonb),
    ('user_01K6EV07KR2MNMDQ60BC03ZM1A', 'default-org', 'Code Review Assistant',
     'You are a code review expert. Focus on security, performance, and best practices.',
     'system', false, '{"purpose": "code_review", "language": "en"}'::jsonb)
ON CONFLICT (user_id, name) DO NOTHING;

-- 5. Insert sample MCP configurations
INSERT INTO mcp_configurations (
    user_id,
    org_id,
    name,
    mcp_type,
    config,
    is_active
) VALUES 
    ('user_01K6EV07KR2MNMDQ60BC03ZM1A', 'default-org', 'File System MCP',
     'stdio', '{"command": "npx", "args": ["@modelcontextprotocol/server-filesystem", "/tmp"]}'::jsonb, true),
    ('user_01K6EV07KR2MNMDQ60BC03ZM1A', 'default-org', 'GitHub MCP',
     'http', '{"url": "https://api.github.com", "headers": {"Authorization": "token GITHUB_TOKEN"}}'::jsonb, true)
ON CONFLICT (user_id, name) DO NOTHING;

-- 6. Insert available agents
INSERT INTO agents (name, type, backend_type, config, is_active) VALUES 
    ('Claude 4.5 Sonnet', 'claude', 'anthropic', 
     '{"model": "claude-4.5-sonnet", "max_tokens": 8192, "supports_tools": true}'::jsonb, true),
    ('Claude 4.5 Haiku', 'claude', 'anthropic',
     '{"model": "claude-4.5-haiku", "max_tokens": 8192, "supports_tools": true}'::jsonb, true),
    ('Gemini 2.5 Pro', 'vertexai', 'google',
     '{"model": "gemini-2.5-pro", "max_tokens": 8000, "supports_tools": true}'::jsonb, true),
    ('Gemini 2.5 Flash', 'vertexai', 'google',
     '{"model": "gemini-2.5-flash", "max_tokens": 8000, "supports_tools": true}'::jsonb, true)
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- Verify user was inserted
SELECT 'User Record:' as info, id, email, org_id, created_at 
FROM users 
WHERE id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A';

-- Verify user settings
SELECT 'User Settings:' as info, user_id, default_model, temperature, max_tokens 
FROM user_settings 
WHERE user_id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A';

-- Verify API keys
SELECT 'API Keys:' as info, COUNT(*) as count 
FROM api_keys 
WHERE user_id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A';

-- Verify agents
SELECT 'Available Agents:' as info, name, type, is_active 
FROM agents 
WHERE is_active = true;

-- Check RLS is enabled
SELECT 'RLS Status:' as info, tablename, rowsecurity 
FROM pg_tables 
WHERE schemaname = 'public' 
    AND tablename IN ('users', 'user_settings', 'api_keys', 'system_prompts', 'mcp_configurations');

SELECT 'User data insertion completed successfully!' as result;

-- Multi-tenant AgentAPI Database Schema
-- Designed for Supabase PostgreSQL with RLS

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Users table (integrated with Supabase auth)
CREATE TABLE users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    name TEXT,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User sessions for multi-tenant isolation
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_type TEXT NOT NULL,
    workspace_path TEXT NOT NULL,
    container_id TEXT,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'terminated')),
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- MCP configurations
CREATE TABLE mcp_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('http', 'sse', 'stdio')),
    auth_type TEXT NOT NULL CHECK (auth_type IN ('bearer', 'apikey', 'oauth', 'none')),
    endpoint TEXT,
    config JSONB NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'org', 'user')),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'active')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure proper scoping
    CONSTRAINT mcp_scope_check CHECK (
        (scope = 'global' AND org_id IS NULL AND user_id IS NULL) OR
        (scope = 'org' AND org_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND org_id IS NOT NULL AND user_id IS NOT NULL)
    )
);

-- System prompt configurations
CREATE TABLE system_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    template TEXT, -- Go template for dynamic content
    variables JSONB DEFAULT '{}',
    scope TEXT NOT NULL CHECK (scope IN ('global', 'org', 'user')),
    priority INTEGER DEFAULT 0,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure proper scoping
    CONSTRAINT prompt_scope_check CHECK (
        (scope = 'global' AND org_id IS NULL AND user_id IS NULL) OR
        (scope = 'org' AND org_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND org_id IS NOT NULL AND user_id IS NOT NULL)
    )
);

-- Audit logs for compliance
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_org_id ON user_sessions(org_id);
CREATE INDEX idx_user_sessions_status ON user_sessions(status);
CREATE INDEX idx_mcp_configs_scope ON mcp_configs(scope, org_id, user_id);
CREATE INDEX idx_mcp_configs_status ON mcp_configs(status);
CREATE INDEX idx_system_prompts_scope ON system_prompts(scope, org_id, user_id);
CREATE INDEX idx_system_prompts_active ON system_prompts(is_active);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_org_id ON audit_logs(org_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Row Level Security (RLS) Policies
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies for organizations
CREATE POLICY "Users can view their own organization" ON organizations
    FOR SELECT USING (
        id IN (SELECT org_id FROM users WHERE id = auth.uid())
    );

-- RLS Policies for users
CREATE POLICY "Users can view users in their organization" ON users
    FOR SELECT USING (
        org_id IN (SELECT org_id FROM users WHERE id = auth.uid())
    );

CREATE POLICY "Users can update their own profile" ON users
    FOR UPDATE USING (id = auth.uid());

-- RLS Policies for user_sessions
CREATE POLICY "Users can manage their own sessions" ON user_sessions
    FOR ALL USING (user_id = auth.uid());

-- RLS Policies for mcp_configs
CREATE POLICY "Users can view global MCP configs" ON mcp_configs
    FOR SELECT USING (scope = 'global');

CREATE POLICY "Users can view org MCP configs" ON mcp_configs
    FOR SELECT USING (
        scope = 'org' AND org_id IN (SELECT org_id FROM users WHERE id = auth.uid())
    );

CREATE POLICY "Users can view their own MCP configs" ON mcp_configs
    FOR SELECT USING (
        scope = 'user' AND user_id = auth.uid()
    );

CREATE POLICY "Users can create MCP configs" ON mcp_configs
    FOR INSERT WITH CHECK (
        created_by = auth.uid() AND
        (scope = 'user' AND user_id = auth.uid()) OR
        (scope = 'org' AND org_id IN (SELECT org_id FROM users WHERE id = auth.uid()))
    );

-- RLS Policies for system_prompts
CREATE POLICY "Users can view global system prompts" ON system_prompts
    FOR SELECT USING (scope = 'global');

CREATE POLICY "Users can view org system prompts" ON system_prompts
    FOR SELECT USING (
        scope = 'org' AND org_id IN (SELECT org_id FROM users WHERE id = auth.uid())
    );

CREATE POLICY "Users can view their own system prompts" ON system_prompts
    FOR SELECT USING (
        scope = 'user' AND user_id = auth.uid()
    );

CREATE POLICY "Users can create system prompts" ON system_prompts
    FOR INSERT WITH CHECK (
        created_by = auth.uid() AND
        (scope = 'user' AND user_id = auth.uid()) OR
        (scope = 'org' AND org_id IN (SELECT org_id FROM users WHERE id = auth.uid()))
    );

-- RLS Policies for audit_logs
CREATE POLICY "Users can view their own audit logs" ON audit_logs
    FOR SELECT USING (user_id = auth.uid());

-- Functions for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_sessions_updated_at BEFORE UPDATE ON user_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mcp_configs_updated_at BEFORE UPDATE ON mcp_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_prompts_updated_at BEFORE UPDATE ON system_prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
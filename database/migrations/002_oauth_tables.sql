-- OAuth Token Storage and State Management Tables
-- Migration: 002_oauth_tables.sql
-- Description: Adds tables for OAuth token storage and CSRF state management

-- OAuth state storage (for CSRF protection)
CREATE TABLE IF NOT EXISTS oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state TEXT NOT NULL UNIQUE,
    provider TEXT NOT NULL,
    mcp_name TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_verifier TEXT, -- Encrypted PKCE code_verifier
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Index for quick lookup
    CONSTRAINT oauth_states_unique_state UNIQUE(state)
);

-- OAuth tokens storage (encrypted)
CREATE TABLE IF NOT EXISTS mcp_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mcp_name TEXT NOT NULL,
    provider TEXT NOT NULL,
    access_token TEXT NOT NULL, -- Encrypted
    refresh_token TEXT, -- Encrypted
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    token_type TEXT DEFAULT 'Bearer',
    scope TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Ensure one token set per user per MCP
    CONSTRAINT mcp_oauth_tokens_unique_user_mcp UNIQUE(user_id, mcp_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_oauth_states_user_id ON oauth_states(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_states_created_at ON oauth_states(created_at);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_user_id ON mcp_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_mcp_name ON mcp_oauth_tokens(mcp_name);
CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_expires_at ON mcp_oauth_tokens(expires_at);

-- Row Level Security (RLS) Policies
ALTER TABLE oauth_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;

-- RLS Policies for oauth_states
CREATE POLICY "Users can manage their own OAuth states" ON oauth_states
    FOR ALL USING (user_id = auth.uid());

-- RLS Policies for mcp_oauth_tokens
CREATE POLICY "Users can view their own OAuth tokens" ON mcp_oauth_tokens
    FOR SELECT USING (user_id = auth.uid());

CREATE POLICY "Users can insert their own OAuth tokens" ON mcp_oauth_tokens
    FOR INSERT WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update their own OAuth tokens" ON mcp_oauth_tokens
    FOR UPDATE USING (user_id = auth.uid());

CREATE POLICY "Users can delete their own OAuth tokens" ON mcp_oauth_tokens
    FOR DELETE USING (user_id = auth.uid());

-- Trigger for updated_at on mcp_oauth_tokens
CREATE TRIGGER update_mcp_oauth_tokens_updated_at
    BEFORE UPDATE ON mcp_oauth_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired OAuth states (run periodically via cron)
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states()
RETURNS void AS $$
BEGIN
    DELETE FROM oauth_states
    WHERE created_at < NOW() - INTERVAL '5 minutes';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to clean up expired OAuth tokens (optional, for housekeeping)
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_tokens()
RETURNS void AS $$
BEGIN
    -- Delete tokens that expired more than 7 days ago
    DELETE FROM mcp_oauth_tokens
    WHERE expires_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Grant execute permissions to authenticated users
GRANT EXECUTE ON FUNCTION cleanup_expired_oauth_states() TO authenticated;
GRANT EXECUTE ON FUNCTION cleanup_expired_oauth_tokens() TO authenticated;

-- Comments for documentation
COMMENT ON TABLE oauth_states IS 'Temporary storage for OAuth state parameters (CSRF protection)';
COMMENT ON TABLE mcp_oauth_tokens IS 'Encrypted storage for MCP OAuth access and refresh tokens';
COMMENT ON COLUMN oauth_states.state IS 'Random state parameter for CSRF protection';
COMMENT ON COLUMN oauth_states.code_verifier IS 'Encrypted PKCE code_verifier for enhanced security';
COMMENT ON COLUMN mcp_oauth_tokens.access_token IS 'Encrypted OAuth access token';
COMMENT ON COLUMN mcp_oauth_tokens.refresh_token IS 'Encrypted OAuth refresh token';

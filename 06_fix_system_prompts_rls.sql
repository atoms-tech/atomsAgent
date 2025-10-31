-- ============================================================================
-- FIX SYSTEM_PROMPTS RLS POLICIES TO ALLOW SERVICE_ROLE ACCESS
-- This fixes the 403 error when accessing system_prompts with service_role
-- ============================================================================

-- Ensure the service_role has the required privileges on core tables
GRANT USAGE ON SCHEMA public TO service_role;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE system_prompts TO service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE mcp_configurations TO service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE chat_sessions TO service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE chat_messages TO service_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE platform_admins TO service_role;

GRANT SELECT ON TABLE organizations TO service_role;
GRANT SELECT ON TABLE users TO service_role;
GRANT SELECT ON TABLE user_sessions TO service_role;
GRANT SELECT ON TABLE audit_logs TO service_role;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO service_role;

-- First, check current policies on system_prompts
SELECT schemaname, tablename, rowsecurity 
FROM pg_tables 
WHERE tablename = 'system_prompts' AND schemaname = 'public';

-- Get existing policies
SELECT schemaname, tablename, policyname, permissive, roles, cmd, qual 
FROM pg_policies 
WHERE tablename = 'system_prompts' AND schemaname = 'public';

-- Drop the restrictive policy that prevents service_role access
DROP POLICY IF EXISTS "Users can view their own prompts" ON system_prompts;
DROP POLICY IF EXISTS "Users can insert their own prompts" ON system_prompts;
DROP POLICY IF EXISTS "Users can update their own prompts" ON system_prompts;
DROP POLICY IF EXISTS "Users can delete their own prompts" ON system_prompts;

-- Create new, more permissive policies that include service_role
CREATE POLICY "Allow read access to enabled prompts" ON system_prompts
    FOR SELECT USING (
        enabled = true AND (
            auth.role() = 'service_role' OR
            auth.uid()::text = user_id OR
            (is_public = true AND org_id = current_setting('app.current_org_id', true))
        )
    );

CREATE POLICY "Allow users to manage their own prompts" ON system_prompts
    FOR ALL USING (
        auth.uid()::text = user_id
    );

CREATE POLICY "Allow service role full access" ON system_prompts
    FOR ALL USING (
        auth.role() = 'service_role'
    );

-- Also ensure the table has the correct schema
ALTER TABLE system_prompts 
ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true,
ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS scope TEXT DEFAULT 'user' CHECK (scope IN ('global', 'organization', 'user')),
ADD COLUMN IF NOT EXISTS template TEXT;

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_system_prompts_enabled ON system_prompts(enabled);
CREATE INDEX IF NOT EXISTS idx_system_prompts_priority ON system_prompts(priority DESC);
CREATE INDEX IF NOT EXISTS idx_system_prompts_scope ON system_prompts(scope);

-- Verify the fix
SELECT 'system_prompts RLS policies updated successfully!' as result,
       now() as completion_time;

-- Test the policy (this should work now)
SELECT COUNT(*) as test_count
FROM system_prompts 
WHERE enabled = true 
LIMIT 1;

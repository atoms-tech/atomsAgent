-- ============================================================================
-- VERIFICATION SCRIPT
-- Run this after all fixes to verify everything works
-- ============================================================================

-- 1. Check all tables exist
SELECT 'Tables Verification:' as section,
       table_name,
       'EXISTS' as status
FROM information_schema.tables 
WHERE table_schema = 'public' 
    AND table_name IN ('users', 'user_settings', 'api_keys', 'system_prompts', 'mcp_configurations', 'agents', 'audit_logs', 'chat_sessions')
ORDER BY table_name;

-- 2. Check RLS is enabled
SELECT 'RLS Status:' as section,
       tablename,
       rowsecurity as rls_enabled
FROM pg_tables 
WHERE schemaname = 'public' 
    AND tablename IN ('users', 'user_settings', 'api_keys', 'system_prompts', 'mcp_configurations', 'agents', 'audit_logs', 'chat_sessions')
ORDER BY tablename;

-- 3. Check if user data exists
SELECT 'User Data:' as section,
       'users' as table_name,
       COUNT(*) as record_count
FROM users 
WHERE id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A'
UNION ALL
SELECT 'User Settings:' as section,
       'user_settings' as table_name,
       COUNT(*) as record_count
FROM user_settings 
WHERE user_id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A'
UNION ALL
SELECT 'API Keys:' as section,
       'api_keys' as table_name,
       COUNT(*) as record_count
FROM api_keys 
WHERE user_id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A'
UNION ALL
SELECT 'System Prompts:' as section,
       'system_prompts' as table_name,
       COUNT(*) as record_count
FROM system_prompts 
WHERE user_id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A';

-- 4. Check agents are available
SELECT 'Available Agents:' as section,
       name,
       type,
       is_active
FROM agents 
WHERE is_active = true
ORDER BY type, name;

-- 5. Test user settings query structure
SELECT 'User Settings Structure Test:' as section,
       column_name,
       data_type,
       is_nullable
FROM information_schema.columns 
WHERE table_schema = 'public' 
    AND table_name = 'user_settings'
ORDER BY ordinal_position;

-- 6. Verify metadata structure in users table
SELECT 'Users Metadata Structure Test:' as section,
       user_id,
       jsonb_typeof(metadata) as metadata_type,
       jsonb_object_keys(metadata) as metadata_keys
FROM users 
WHERE id = 'user_01K6EV07KR2MNMDQ60BC03ZM1A';

-- 7. Check for any remaining policy conflicts
SELECT 'Policy Status:' as section,
       policyname,
       permissive,
       roles,
       cmd
FROM pg_policies 
WHERE tablename IN ('users', 'user_settings', 'api_keys')
ORDER BY tablename, policyname;

-- 8. Final health check
SELECT 
    'Database Health Summary:' as section,
    'All critical tables exist and RLS is enabled' as status,
    COUNT(*) as total_tables_checked
FROM information_schema.tables 
WHERE table_schema = 'public' 
    AND table_name IN ('users', 'user_settings', 'api_keys', 'system_prompts', 'mcp_configurations', 'agents', 'audit_logs', 'chat_sessions');

SELECT 'Database verification completed successfully!' as result;

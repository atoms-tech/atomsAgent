-- ============================================================================
-- DATABASE INSPECTION SCRIPT
-- Run this first to see what tables exist and their structure
-- ============================================================================

-- 1. List all tables in public schema
SELECT 
    table_name,
    table_type
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

-- 2. Check if key tables exist
SELECT 
    'api_keys' as table_name,
    COUNT(*) as record_count
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'api_keys'
UNION ALL
SELECT 
    'users' as table_name,
    COUNT(*) as record_count
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'users'
UNION ALL
SELECT 
    'user_settings' as table_name,
    COUNT(*) as record_count
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'user_settings'
UNION ALL
SELECT 
    'system_prompts' as table_name,
    COUNT(*) as record_count
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'system_prompts'
UNION ALL
SELECT 
    'mcp_configurations' as table_name,
    COUNT(*) as record_count
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'mcp_configurations';

-- 3. Check columns in users table (if exists)
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_schema = 'public' AND table_name = 'users'
ORDER BY ordinal_position;

-- 4. Check RLS status for key tables
SELECT 
    schemaname,
    tablename,
    rowsecurity
FROM pg_tables 
WHERE schemaname = 'public' 
    AND tablename IN ('api_keys', 'users', 'user_settings', 'system_prompts', 'mcp_configurations');

-- 5. Check if RLS policies exist
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    roles,
    cmd,
    qual
FROM pg_policies 
WHERE tablename IN ('api_keys', 'users', 'user_settings', 'system_prompts', 'mcp_configurations')
ORDER BY tablename, policyname;

# Supabase Setup & RLS Configuration

## Current Issue

The service role key cannot read tables even though `postgres` owns them. This is because Supabase has **Row-Level Security (RLS)** enabled by default on some tables.

## Solution: Enable Service Role Bypass

The **service_role** key should automatically bypass RLS, but we need to ensure:

1. RLS policies are properly configured
2. Service role has correct grants

### Quick Fix (Run in Supabase SQL Editor)

```sql
-- Check if RLS is enabled
SELECT tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health');

-- If rowsecurity = true, disable RLS for service role access
-- Grant full access to service role
GRANT SELECT, INSERT, UPDATE, DELETE ON agents TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON models TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON chat_sessions TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON chat_messages TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON agent_health TO postgres;

-- If RLS is causing issues, you can:
-- 1. Disable RLS entirely (simpler for private API):
ALTER TABLE agents DISABLE ROW LEVEL SECURITY;
ALTER TABLE models DISABLE ROW LEVEL SECURITY;
ALTER TABLE chat_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE chat_messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_health DISABLE ROW LEVEL SECURITY;

-- 2. Or create policies that allow service role:
-- Example policy for agents table:
CREATE POLICY "Service role full access" ON agents
  FOR ALL
  USING (true)
  WITH CHECK (true)
  TO authenticated;
```

## Steps to Fix

1. **Go to Supabase Dashboard**
   - Project: `ydogoylwenufckscqijp`
   - Select **SQL Editor**

2. **Run First Query** (Check RLS Status)
   ```sql
   SELECT tablename, rowsecurity
   FROM pg_tables
   WHERE schemaname = 'public'
   AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health');
   ```

3. **If RLS is enabled (rowsecurity = true)**

   Option A: Disable RLS entirely (recommended for private API):
   ```sql
   ALTER TABLE agents DISABLE ROW LEVEL SECURITY;
   ALTER TABLE models DISABLE ROW LEVEL SECURITY;
   ALTER TABLE chat_sessions DISABLE ROW LEVEL SECURITY;
   ALTER TABLE chat_messages DISABLE ROW LEVEL SECURITY;
   ALTER TABLE agent_health DISABLE ROW LEVEL SECURITY;
   ```

   Option B: Create policies for service role:
   ```sql
   -- Run for each table:
   CREATE POLICY "Enable all for authenticated" ON agents
     FOR ALL
     USING (true)
     WITH CHECK (true);
   ```

4. **Restart Server**
   ```bash
   ./start.sh
   ```

## Expected Result

```
time=... level=INFO msg="initializing Supabase connection"
time=... level=INFO msg="Supabase connection established"
time=... level=INFO msg="database connection established"
time=... level=INFO msg="server listening" port=3284
```

## Troubleshooting

### Still Getting 42501 Error?

1. **Verify grants**:
   ```sql
   SELECT grantee, privilege_type
   FROM information_schema.role_table_grants
   WHERE table_name IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health');
   ```

2. **Check RLS status**:
   ```sql
   SELECT schemaname, tablename, rowsecurity
   FROM pg_tables
   WHERE tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health');
   ```

3. **Verify service role exists**:
   ```sql
   SELECT usename FROM pg_user WHERE usename = 'postgres';
   ```

## After Setup

Once the server is running:

```bash
# Health check
curl http://localhost:3284/health

# Should return:
# {"status":"healthy","agents":["ccrouter"],"primary":"ccrouter"}
```

## Reference

- **Supabase RLS Docs**: https://supabase.com/docs/guides/auth/row-level-security
- **PostgreSQL RLS**: https://www.postgresql.org/docs/current/ddl-rowsecurity.html


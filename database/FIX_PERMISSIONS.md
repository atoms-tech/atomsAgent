# Fix Table Ownership Issue

## Problem
Tables are owned by `supabase_read_only_user` but you need `postgres` role ownership to modify them.

## Solution 1: Use Supabase SQL Editor (Easiest)

1. Go to **Supabase Dashboard** → Your Project → **SQL Editor**
2. Click **New Query**
3. Paste this SQL:

```sql
-- Transfer table ownership from supabase_read_only_user to postgres
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;
```

4. Click **RUN**
5. Verify with:

```sql
SELECT tablename, tableowner
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health')
ORDER BY tablename;
```

---

## Solution 2: Use psql with postgres password

1. Get your postgres password from Supabase:
   - Dashboard → Settings → Database → Connection Info
   - Copy the password for the `postgres` user

2. Run this command locally:

```bash
psql "postgresql://postgres:[PASSWORD]@db.ydogoylwenufckscqijp.supabase.co:5432/postgres" << EOF
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;

SELECT tablename, tableowner
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health')
ORDER BY tablename;
EOF
```

Replace `[PASSWORD]` with your actual postgres password.

---

## Solution 3: Fix at table creation time (for future tables)

When creating new tables via MCP, they inherit the read-only user's permissions. To prevent this in the future, you may need to:
- Use Supabase SQL Editor directly for DDL operations
- Or recreate tables with proper role handling

---

## Why This Happens

Supabase uses different roles for security:
- `supabase_read_only_user` - Limited read-only access (what the API uses)
- `postgres` - Full superuser access

Tables created through the API connection use the API's role. You need superuser (`postgres`) to change ownership.

---

## Verify It Worked

After running the ownership fix, all tables should show `tableowner: postgres`:

```
     tablename     | tableowner
-------------------+------------
 agent_health      | postgres
 agents            | postgres
 chat_messages     | postgres
 chat_sessions     | postgres
 models            | postgres
```

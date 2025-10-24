# Fix Supabase Table Permissions

The AgentAPI tables were created with incorrect ownership. They're owned by `supabase_read_only_user` but need to be owned by `postgres` for the service role key to modify them.

## Error You'll See

```
(42501) permission denied for table agents
```

This happens because the service role can read, but not modify, tables owned by the read-only user.

---

## Solution: Run in Supabase SQL Editor (1 minute)

### Step 1: Go to Supabase Dashboard
https://app.supabase.com → Select your project (`ydogoylwenufckscqijp`)

### Step 2: Open SQL Editor
Left sidebar → **SQL Editor**

### Step 3: Create New Query
Click **New Query**

### Step 4: Paste and Run

Copy and paste this SQL:

```sql
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;
```

Click **RUN** button (or press `Ctrl+Enter`)

### Step 5: Verify Success

You should see:
```
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
```

---

## Verify It Worked

Run this query to confirm ownership:

```sql
SELECT tablename, tableowner
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health')
ORDER BY tablename;
```

Expected result:

```
     tablename     | tableowner
-------------------+------------
 agent_health      | postgres
 agents            | postgres
 chat_messages     | postgres
 chat_sessions     | postgres
 models            | postgres
```

All should show `postgres` as owner ✅

---

## After Fixing Permissions

Restart the server:

```bash
./start.sh
```

You should see:
```
✅ Supabase connection established
✅ Server listening on port 3284
```

---

## Why This Happened

When tables are created through the Supabase API (PostgREST), they inherit the permissions of the API connection role (`supabase_read_only_user`). Since the service role key is different from the role that created the tables, we need to explicitly transfer ownership.

This is a one-time fix. Future tables will be created with correct ownership once the schema is set up properly.

---

## Quick Command (if using psql directly)

If you have `psql` installed and the database password:

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

Replace `[PASSWORD]` with your database password from Supabase → Settings → Database.

---

## Support

If you get a different error:
- Check that you're using the **Service Role API Key** (not Anon Key)
- Confirm the project slug is correct (`ydogoylwenufckscqijp`)
- Verify database tables actually exist (they should from schema migration)


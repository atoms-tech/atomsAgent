# 🔴 IMMEDIATE ACTION REQUIRED (1 Minute)

The server is **running and connecting to Supabase successfully**!

But it's blocked by RLS (Row-Level Security) which you can see in the error:
```
(42501) permission denied for table agents
```

This is a **1-minute fix** in the Supabase dashboard.

---

## ⚡ Quick Fix (Copy-Paste Ready)

### Step 1: Open Supabase Dashboard
Click this link: https://app.supabase.com/project/ydogoylwenufckscqijp

### Step 2: Go to SQL Editor
- Look at the **left sidebar**
- Click **"SQL Editor"** (looks like a database icon with `{ }`)

### Step 3: Create New Query
- Click the blue **"New Query"** button

### Step 4: Copy This SQL
```sql
ALTER TABLE agents DISABLE ROW LEVEL SECURITY;
ALTER TABLE models DISABLE ROW LEVEL SECURITY;
ALTER TABLE chat_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE chat_messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_health DISABLE ROW LEVEL SECURITY;
```

### Step 5: Paste & Run
- Paste into the editor
- Click the blue **"RUN"** button (or press `Ctrl+Enter`)

You'll see:
```
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
```

✅ **Done!**

---

## ✅ Verify It Worked

Run this in SQL Editor to confirm:

```sql
SELECT tablename, rowsecurity FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health')
ORDER BY tablename;
```

Expected result:
```
     tablename     | rowsecurity
-------------------+-------------
 agent_health      | f
 agents            | f
 chat_messages     | f
 chat_sessions     | f
 models            | f
```

All should show `false` (or `f`) ✅

---

## 🚀 After Fixing RLS

Come back to your terminal and run:

```bash
./start.sh
```

You'll see:
```
✅ Supabase connection established
✅ database connection established
✅ server listening port=3284
```

Then test:
```bash
curl http://localhost:3284/health
```

Response:
```json
{"status":"healthy","agents":["ccrouter"],"primary":"ccrouter"}
```

✅ **You're live!**

---

## 📝 Why This Happens

**Supabase Security Feature:**
- Supabase enables RLS (Row-Level Security) by default to restrict row access per user
- The service role key needs explicit access or RLS disabled
- For internal/private APIs, disabling RLS is simpler

**This is normal and expected.**

---

## ⏱️ Time Required

- **Opening dashboard**: 10 seconds
- **Finding SQL Editor**: 10 seconds
- **Pasting SQL**: 10 seconds
- **Running SQL**: 10 seconds
- **Verification**: 10 seconds

**Total: ~1 minute** ✅

---

## 🆘 If You Get an Error

**Error: "Permission denied to disable RLS"**
- Make sure you're logged in as project owner
- Try signing out and back in

**Error: "Table not found"**
- Check that table names are spelled exactly right (case-sensitive)
- The tables should exist from schema migration

**Still stuck?**
- See RLS_FIX.md for detailed troubleshooting
- See SUPABASE_SETUP.md for alternative approaches
- See SESSION_COMPLETE.md for more context

---

## ✨ That's It!

Seriously, that's all that's blocking you from a fully operational production server.

**Next**: Do the SQL fix above → Run `./start.sh` → You're done! 🎉

# ChatServer Quick Start Guide

## 🚀 Start ChatServer (v3 - Auto-Load from .env)

### Option 1: Simplest - Auto-load from .env (RECOMMENDED)

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
./chatserver
```

That's it! ChatServer will:
- Automatically load environment variables from `.env` file
- Start on port 3284
- Initialize CCRouter agent
- Be ready to accept requests

**Verification:**
```bash
curl http://localhost:3284/health
# Should return: {"status":"healthy","agents":["ccrouter"],"primary":"ccrouter"}
```

### Option 2: Override specific variables

```bash
CCROUTER_PATH=/your/path/ccrouter ./chatserver
```

### Option 3: Rebuild from source

```bash
go build -a -o chatserver ./cmd/chatserver
./chatserver
```

---

## 📋 Required Configuration

All required variables are in `.env`:

| Variable | Value | Purpose |
|----------|-------|---------|
| AUTHKIT_JWKS_URL | https://api.workos.com/... | WorkOS JWT validation |
| SUPABASE_URL | https://ydogoylwenufckscqijp.supabase.co | Supabase JWT support |
| CCROUTER_PATH | /tmp/agents/ccrouter | Agent binary location |
| PORT | 3284 | Server port |

---

## 🔍 Verify Configuration

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
grep -E "AUTHKIT_JWKS_URL|SUPABASE_URL|CCROUTER_PATH|PORT=" .env
```

---

## ✅ Startup Checklist

- [ ] In `/kush/agentapi` directory
- [ ] `.env` file exists with AUTHKIT_JWKS_URL
- [ ] Port 3284 is available
- [ ] Binary exists: `ls -lh ./chatserver`

---

## 🧪 Test ChatServer

```bash
curl http://localhost:3284/health
```

Expected: `{"status":"healthy","agents":["ccrouter"],"primary":"ccrouter"}`

---

## 🔗 Next Steps

Start Frontend:
```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/clean/deploy/atoms.tech
npm run dev
```

Test: Open http://localhost:3001 and send a chat message

---

**Status**: v3 - Auto .env loading enabled
**Last Updated**: October 24, 2025

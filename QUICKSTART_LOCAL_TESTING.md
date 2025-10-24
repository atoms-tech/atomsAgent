# QuickStart: Local Testing AgentAPI + atoms.tech

**Date**: October 24, 2025
**Purpose**: Get agentapi + atoms.tech running locally for testing
**Time**: ~15 minutes setup + auto-start

---

## 30-Second Setup

```bash
# 1. Navigate to agentapi
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi

# 2. Run the automated test script
./scripts/local-test.sh

# 3. Wait for output confirming services are ready
# 4. Open browser and visit http://localhost:3000
```

**That's it! Everything will start automatically.**

---

## What Gets Started

### Backend (agentapi)
- **Port**: 3284
- **Services**:
  - AgentAPI (Go) - REST API
  - PostgreSQL - Database
  - Redis - Cache/session store
  - Nginx - Reverse proxy
- **URLs**:
  - Health: http://localhost:3284/health
  - OAuth: http://localhost:3284/api/mcp/oauth/init
  - MCP API: http://localhost:3284/api/v1/mcp/configurations

### Frontend (atoms.tech)
- **Port**: 3000
- **Branch**: feature/phase2-oauth-integration
- **Services**:
  - Next.js development server with Bun
- **URLs**:
  - Home: http://localhost:3000
  - OAuth Test: http://localhost:3000/mcp/oauth
  - Configurations: http://localhost:3000/mcp/configurations

---

## Manual Testing Steps

### Step 1: Test Backend Health

```bash
# Check backend is running
curl http://localhost:3284/health | jq .

# Expected response:
{
  "status": "UP",
  "components": {
    "database": "UP",
    "redis": "UP",
    "fastmcp": "UP"
  }
}
```

### Step 2: Test OAuth Flow (Recommended: Use GitHub OAuth)

1. **Open browser**: http://localhost:3000/mcp/oauth

2. **Expected**:
   - Provider selection modal appears
   - Options: GitHub, Google, Azure, Auth0

3. **Click "GitHub"**:
   - New popup opens
   - Redirects to GitHub authorization
   - Authorize the application
   - Redirected back to http://localhost:3000/oauth/callback

4. **Verify success**:
   - Callback URL should show: `?code=...&state=...`
   - Token stored in browser local storage (encrypted)
   - Redirected to MCP configuration page

### Step 3: Test MCP Configuration

1. **Navigate to**: http://localhost:3000/mcp/configurations

2. **Create new configuration**:
   - Name: "GitHub MCP"
   - URL: http://localhost:8000/mcp/connect
   - Provider: github
   - Token: Should be auto-populated from OAuth

3. **Click "Save"**:
   - Configuration created
   - Should appear in list below

### Step 4: Test Error Handling

1. **Rate Limiting Test**:
   ```bash
   # Send 65 requests rapidly
   for i in {1..65}; do
     curl -s http://localhost:3284/health > /dev/null
   done
   ```
   - First 60 should succeed
   - Request 61-65 should return 429 (Too Many Requests)

2. **Circuit Breaker Test**:
   ```bash
   # Stop FastMCP service
   docker-compose -f docker-compose.multitenant.yml stop fastmcp

   # Try to access MCP endpoint multiple times
   # Should eventually fail with "Circuit breaker is open"
   ```

---

## Viewing Logs

### Backend Logs

```bash
# All services
docker-compose -f docker-compose.multitenant.yml logs -f

# Just agentapi
docker-compose -f docker-compose.multitenant.yml logs -f agentapi

# Just PostgreSQL
docker-compose -f docker-compose.multitenant.yml logs -f postgres

# Just Redis
docker-compose -f docker-compose.multitenant.yml logs -f redis
```

### Frontend Logs

```bash
# If running in terminal
tail -f /tmp/frontend.log

# Or check browser console
# Open DevTools (F12) → Console tab
# Look for any errors or API calls
```

### Test Logs

```bash
# Go unit tests
tail -f /tmp/go_tests.log

# Package manager install
tail -f /tmp/bun_install.log  # or /tmp/npm_install.log
```

---

## Stopping Services

### Stop Everything

```bash
# Press Ctrl+C in terminal running local-test.sh
# This will:
# 1. Stop frontend server
# 2. Stop Docker containers
# 3. Clean up resources
```

### Or manually:

```bash
# Stop frontend
pkill -f "next dev"

# Stop Docker
docker-compose -f docker-compose.multitenant.yml down

# Stop Redis CLI
redis-cli shutdown
```

---

## Environment Variables Explanation

### From atoms.tech/.env.local (Frontend)

```bash
# Supabase - Shared auth system
NEXT_PUBLIC_SUPABASE_URL=https://ydogoylwenufckscqijp.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=[JWT_TOKEN]

# App URLs
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_AGENTAPI_URL=http://localhost:3284

# OAuth Providers
NEXT_PUBLIC_OAUTH_ENABLED=true
NEXT_PUBLIC_OAUTH_PROVIDERS=github,google,azure,auth0
```

### From agentapi/.env (Backend)

```bash
# Database - Local PostgreSQL
DATABASE_URL=postgresql://agentapi:agentapi@postgres:5432/agentapi?sslmode=disable

# Supabase - Shared auth system (same as frontend)
SUPABASE_URL=https://ydogoylwenufckscqijp.supabase.co
SUPABASE_ANON_KEY=[JWT_TOKEN]
SUPABASE_JWT_SECRET=[SECRET]

# Redis - Upstash (can use local or remote)
UPSTASH_REDIS_URL=rediss://default:[PASSWORD]@[HOST]:6379

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10

# Circuit Breaker
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
```

---

## Architecture at a Glance

```
Browser (localhost:3000)
    ↓ User clicks "Connect OAuth"
    ↓
Frontend (Next.js / atoms.tech)
    ↓ POST /api/mcp/oauth/init
    ↓
Backend (Go / agentapi:3284)
    ├─ Generate OAuth URL
    ├─ Store state in Redis
    └─ Return OAuth URL
    ↓
Browser
    ├─ Redirect to GitHub auth
    ├─ User authorizes
    └─ GitHub redirects back
    ↓
Frontend (/oauth/callback)
    ↓ POST /api/mcp/oauth/callback
    ↓
Backend
    ├─ Validate state (CSRF)
    ├─ Exchange code for token
    ├─ Encrypt and store in Redis + PostgreSQL
    └─ Return success
    ↓
Frontend
    ├─ Store encrypted token
    ├─ Show success message
    └─ Redirect to MCP config
```

---

## Useful URLs

| Component | URL | Purpose |
|-----------|-----|---------|
| Frontend | http://localhost:3000 | Main app |
| OAuth Flow | http://localhost:3000/mcp/oauth | Test OAuth |
| MCP Config | http://localhost:3000/mcp/configurations | Manage MCPs |
| Backend Health | http://localhost:3284/health | Backend status |
| Backend API | http://localhost:3284/api/v1/mcp/configurations | MCP API |
| OAuth Init | http://localhost:3284/api/mcp/oauth/init | OAuth endpoint |
| Metrics | http://localhost:3284:9090/metrics | Prometheus metrics |
| PostgreSQL | localhost:5432 | Database |
| Redis | localhost:6379 | Cache |

---

## Troubleshooting

### Frontend won't start

```bash
# Check if port 3000 is already in use
lsof -i :3000

# Kill existing process
pkill -f "next dev"

# Retry
./scripts/local-test.sh
```

### Backend won't start

```bash
# Check Docker is running
docker ps

# Clean and rebuild
docker-compose -f docker-compose.multitenant.yml down -v
./build-multitenant.sh
./scripts/local-test.sh
```

### Database connection fails

```bash
# Check PostgreSQL is running
docker-compose -f docker-compose.multitenant.yml logs postgres

# Try manual connection
psql -h localhost -U agentapi -d agentapi -c "SELECT 1"

# If fails, reset database
docker-compose -f docker-compose.multitenant.yml down -v
docker-compose -f docker-compose.multitenant.yml up -d postgres
```

### Redis connection fails

```bash
# Check Redis is running
docker-compose -f docker-compose.multitenant.yml logs redis

# Test connection
redis-cli -h localhost PING

# Should return: PONG
```

### OAuth flow fails

```bash
# Check backend logs
docker-compose -f docker-compose.multitenant.yml logs agentapi | tail -50

# Check browser console for errors
# (DevTools → Console tab)

# Verify environment variables in frontend .env.local
cat /Users/kooshapari/temp-prodvercel/485/clean/deploy/atoms.tech/.env.local | grep OAUTH
```

---

## Performance Notes

### Local Performance (Expected)

| Metric | Value | Notes |
|--------|-------|-------|
| Health Check | 10-20ms | Should be very fast |
| OAuth Init | 50-100ms | Includes JWT generation |
| Token Exchange | 200-500ms | Includes OAuth provider call |
| DB Query | 5-20ms | Local PostgreSQL |
| Redis Operation | 1-5ms | Local or Upstash |

### Load Test (if K6 available)

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi

# Run load test against local backend
k6 run tests/load/k6_tests.js -e BASE_URL=http://localhost:3284 --vus 50 --duration 5m

# Expected results:
# - Success rate: >99%
# - p95 latency: <500ms
# - Throughput: 100+ req/s (single instance)
```

---

## Next Steps After Testing

### If Everything Works ✅

1. Document any test results
2. Check all manual test cases pass
3. Verify browser console has no errors
4. Review logs for any warnings
5. Proceed to staging deployment

### If Issues Found ❌

1. Check logs for error messages
2. Verify environment variables
3. Check backend is actually running
4. Check frontend build succeeded
5. Review browser DevTools console
6. Try restarting with fresh Docker containers

---

## Quick Reference

```bash
# Start everything
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
./scripts/local-test.sh

# View backend logs
docker-compose -f docker-compose.multitenant.yml logs -f agentapi

# View frontend logs
tail -f /tmp/frontend.log

# Test backend
curl http://localhost:3284/health | jq .

# Test frontend
curl http://localhost:3000

# Stop everything
# (Press Ctrl+C in terminal where local-test.sh runs)

# Or manually stop
docker-compose -f docker-compose.multitenant.yml down
pkill -f "next dev"
```

---

## Success Criteria

✅ **Backend**
- [ ] Docker containers running
- [ ] Health endpoint returns UP
- [ ] Database connected
- [ ] Redis connected

✅ **Frontend**
- [ ] Server starts on port 3000
- [ ] Page loads in browser
- [ ] No build errors
- [ ] Console has no critical errors

✅ **Integration**
- [ ] OAuth flow completes
- [ ] Token stored encrypted
- [ ] MCP config can be created
- [ ] Connections can be tested

---

**Ready to test! Run the script and let's go! 🚀**

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
./scripts/local-test.sh
```

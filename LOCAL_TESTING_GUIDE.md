# Local Testing Guide - AgentAPI Multi-Tenant Platform

**Date**: October 24, 2025
**Purpose**: Complete local validation before production deployment

---

## Prerequisites

- Docker Desktop installed and running
- Docker Compose v2.0+
- Node.js 18+ (for frontend)
- Bun (for fast package management)
- Go 1.20+ (for running backend tests)
- K6 (for load testing)

---

## Step 1: Local Docker Testing (Backend)

### 1.1 Build Docker Images

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi

# Build the multitenant Docker image
./build-multitenant.sh

# Verify build completed
docker images | grep agentapi
```

**Expected Output**:
```
agentapi    latest    [image-id]    [size]    [time]
```

### 1.2 Start Local Services

```bash
# Start all services in Docker Compose
docker-compose -f docker-compose.multitenant.yml up -d

# Verify all services are running
docker-compose -f docker-compose.multitenant.yml ps
```

**Expected Output**:
```
NAME                    STATUS              PORTS
agentapi                Up (healthy)        3284:3284
postgres                Up (healthy)        5432:5432
redis                   Up (healthy)        6379:6379
nginx                   Up                  80:80, 443:443
```

### 1.3 Verify Database Schema

```bash
# Check PostgreSQL is running
psql -h localhost -U agentapi -d agentapi -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';"

# Expected: 7 tables created (organizations, users, sessions, mcp_configs, oauth_tokens, prompts, audit_logs)
```

### 1.4 Test Health Endpoints

```bash
# Test application health
curl -s http://localhost:3284/health | jq .

# Expected Response:
# {
#   "status": "UP",
#   "components": {
#     "database": "UP",
#     "redis": "UP",
#     "fastmcp": "UP"
#   }
# }

# Test readiness
curl -s http://localhost:3284/ready | jq .

# Test liveness
curl -s http://localhost:3284/live | jq .
```

### 1.5 Test OAuth Endpoints

```bash
# Generate test JWT token (for testing - use Supabase in production)
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJvcmdfaWQiOiJ0ZXN0LW9yZyIsInJvbGUiOiJ1c2VyIiwiaWF0IjoxNjk4MTUyNDAwLCJleHAiOjE2OTgxNTYwMDB9.test"

# Test OAuth initialization
curl -s -X POST http://localhost:3284/api/mcp/oauth/init \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "github",
    "redirect_uri": "http://localhost:3000/oauth/callback"
  }' | jq .
```

### 1.6 Run Backend Tests

```bash
# Run all Go tests
go test ./...

# Run with race detection
go test -race ./...

# Run specific test suite
go test ./tests/integration/... -v

# Run load tests (K6)
k6 run tests/load/k6_tests.js -e BASE_URL=http://localhost:3284
```

### 1.7 View Logs

```bash
# View agentapi logs
docker-compose -f docker-compose.multitenant.yml logs -f agentapi

# View PostgreSQL logs
docker-compose -f docker-compose.multitenant.yml logs -f postgres

# View Redis logs
docker-compose -f docker-compose.multitenant.yml logs -f redis
```

---

## Step 2: Frontend Testing (atoms.tech)

### 2.1 Setup Frontend Repository

```bash
# Navigate to frontend directory
cd /Users/kooshapari/temp-prodvercel/485/clean/deploy/atoms.tech

# Pull latest changes
git pull origin main

# Create new branch for OAuth testing
git checkout -b feature/oauth-integration-test

# Install dependencies
bun install
```

### 2.2 Configure Frontend Environment

Create `.env.local`:

```bash
# Backend API URL
NEXT_PUBLIC_API_URL=http://localhost:3284

# OAuth Configuration
NEXT_PUBLIC_OAUTH_CLIENT_ID=test-client-id
NEXT_PUBLIC_OAUTH_GITHUB_CLIENT_ID=[github-app-client-id]
NEXT_PUBLIC_OAUTH_GOOGLE_CLIENT_ID=[google-oauth-client-id]

# OAuth Redirect
NEXT_PUBLIC_OAUTH_REDIRECT_URI=http://localhost:3000/oauth/callback

# FastMCP URL
NEXT_PUBLIC_FASTMCP_URL=http://localhost:8000

# Feature Flags
NEXT_PUBLIC_ENABLE_MCP_OAUTH=true
```

### 2.3 Start Frontend Development Server

```bash
# Start Next.js development server with Bun
bun dev

# Expected Output:
# ▲ Next.js 14.0.0
# - Local:        http://localhost:3000
# - Environments: .env.local
```

### 2.4 Test OAuth Components

#### Test 1: OAuth Provider Selection

Navigate to `http://localhost:3000/mcp/oauth`

**Expected**:
- OAuth provider selection modal appears
- GitHub option available
- Google option available
- Azure option available
- Auth0 option available

#### Test 2: OAuth Flow Initiation

Click "GitHub" in the OAuth provider selector:

**Expected**:
- OAuth popup opens
- Redirects to GitHub authorization page
- State parameter in URL: `?state=[32-char-random]`
- Code verifier in local storage (PKCE)

#### Test 3: OAuth Callback

After authorizing on GitHub:

**Expected**:
- Callback URL: `http://localhost:3000/oauth/callback?code=[code]&state=[state]`
- Token exchange occurs
- Token stored encrypted in local storage
- Redirect to MCP configuration page
- Success notification appears

#### Test 4: Token Management

In browser DevTools (Application → Local Storage):

**Expected**:
- Token is encrypted (not plaintext)
- Refresh token stored separately
- Token expiry timestamp set
- Provider metadata stored

#### Test 5: Token Refresh

Keep the application open for auto-refresh interval:

**Expected**:
- Token auto-refreshes before expiry
- No user interruption
- New token encrypted and stored
- Expiry timestamp updated

### 2.5 Test MCP Configuration

With OAuth token obtained:

Navigate to `http://localhost:3000/mcp/configurations`

**Expected**:
1. Can list MCP configurations
2. Can create new MCP configuration:
   - Name: "GitHub MCP"
   - URL: `http://localhost:8000/mcp/connect`
   - Provider: "github"
   - Token automatically populated from OAuth
3. Can test connection
4. Can list available tools
5. Can execute tools

### 2.6 Test Error Scenarios

#### Scenario 1: OAuth Provider Error

```bash
# Simulate provider error by using invalid credentials
curl -s http://localhost:3284/api/mcp/oauth/callback \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "code": "invalid_code",
    "state": "test_state",
    "provider": "github"
  }' | jq .
```

**Expected**:
- Error message displayed
- User can retry
- No token stored

#### Scenario 2: Rate Limiting

```bash
# Send 61 requests in 1 minute (limit is 60)
for i in {1..65}; do
  curl -s http://localhost:3284/health > /dev/null
  echo "Request $i"
done
```

**Expected**:
- First 60 requests succeed
- Request 61+ return 429 (Too Many Requests)
- X-RateLimit-Remaining header shows 0
- X-RateLimit-Reset shows retry time

#### Scenario 3: Circuit Breaker

```bash
# Simulate MCP service failure
docker-compose -f docker-compose.multitenant.yml stop fastmcp

# Try to call MCP endpoint
curl -s http://localhost:3284/api/v1/mcp/configurations/test \
  -H "Authorization: Bearer $TOKEN" \
  -X POST
```

**Expected**:
- Circuit breaker opens after 5 failures
- Fast-fail on subsequent requests (no delay)
- Error message: "Circuit breaker is open"
- After 30 seconds, circuit enters half-open state

### 2.7 Run Frontend Tests

```bash
# Run Jest tests
bun test

# Expected: All component tests passing

# Run E2E tests (if using Playwright/Cypress)
bun run test:e2e

# Check build
bun run build

# Expected: No build errors, bundle size acceptable
```

---

## Step 3: Integration Testing

### 3.1 Full OAuth Flow Test

**Automated Test Script**:

```bash
#!/bin/bash
set -e

echo "=== Full OAuth Flow Integration Test ==="

# 1. Start services
echo "Starting services..."
docker-compose -f docker-compose.multitenant.yml up -d

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# 2. Test backend health
echo "Testing backend health..."
curl -s http://localhost:3284/health | jq . || exit 1

# 3. Test OAuth endpoints
echo "Testing OAuth endpoints..."
TOKEN="test_token"

# 4. Test frontend build
echo "Testing frontend build..."
cd /Users/kooshapari/temp-prodvercel/485/clean/deploy/atoms.tech
bun build || exit 1

# 5. Run frontend tests
echo "Running frontend tests..."
bun test || exit 1

echo "=== All Integration Tests Passed ==="
```

Save as `run-integration-tests.sh` and execute:

```bash
chmod +x run-integration-tests.sh
./run-integration-tests.sh
```

### 3.2 Load Testing with Frontend

Use K6 to simulate frontend users:

```bash
k6 run tests/load/k6_tests.js \
  -e BASE_URL=http://localhost:3284 \
  --vus 50 \
  --duration 5m
```

**Expected**:
- Success rate > 99%
- p95 latency < 500ms
- Error rate < 1%

---

## Step 4: Verification Checklist

### Backend (agentapi)
- [ ] Docker image builds successfully
- [ ] All services start without errors
- [ ] Health endpoints respond correctly
- [ ] Database schema created (7 tables)
- [ ] All tests pass (unit, integration, load)
- [ ] No race conditions detected
- [ ] Redis connection working
- [ ] Circuit breaker functional
- [ ] Rate limiting enforced

### Frontend (atoms.tech)
- [ ] Dependencies installed with Bun
- [ ] Development server starts on port 3000
- [ ] OAuth provider selection works
- [ ] OAuth flow completes successfully
- [ ] Token stored encrypted
- [ ] Token refresh works
- [ ] MCP configuration creation works
- [ ] Error scenarios handled gracefully
- [ ] Build completes without errors
- [ ] All tests pass

### Integration
- [ ] Frontend can communicate with backend
- [ ] OAuth token exchange works end-to-end
- [ ] MCP operations execute successfully
- [ ] Rate limiting works across frontend
- [ ] Circuit breaker prevents cascades
- [ ] Error messages display properly
- [ ] Load testing passes (50+ concurrent users)

---

## Troubleshooting

### Issue: Docker Compose fails to start

```bash
# Check Docker daemon is running
docker ps

# Clean up and retry
docker-compose -f docker-compose.multitenant.yml down -v
docker-compose -f docker-compose.multitenant.yml up -d

# Check logs
docker-compose -f docker-compose.multitenant.yml logs
```

### Issue: Database migration fails

```bash
# Check database connection
psql -h localhost -U agentapi -d agentapi -c "\dt"

# Run schema manually if needed
psql -h localhost -U agentapi -d agentapi -f database/schema.sql
```

### Issue: Redis connection fails

```bash
# Check Redis is running
docker-compose -f docker-compose.multitenant.yml logs redis

# Test Redis connection
redis-cli -h localhost PING
```

### Issue: Frontend OAuth not working

```bash
# Check backend is reachable
curl http://localhost:3284/health

# Check environment variables
cat .env.local | grep NEXT_PUBLIC

# Check browser console for CORS errors
# If CORS errors, verify CORS_ORIGINS in backend .env
```

### Issue: Tests failing

```bash
# Run tests with verbose output
go test -v ./...

# Check for race conditions specifically
go test -race ./...

# Frontend tests
bun test --verbose
```

---

## Performance Baseline (Local)

After successful local testing, record these metrics:

```bash
# Response time (should be < 100ms locally)
time curl http://localhost:3284/health

# Throughput (requests per second)
ab -n 1000 -c 100 http://localhost:3284/health

# Load test with K6
k6 run tests/load/k6_tests.js \
  -e BASE_URL=http://localhost:3284 \
  --vus 100 \
  --duration 5m
```

**Expected Local Performance**:
- p50 Latency: 10-50ms
- p95 Latency: 50-150ms
- Throughput: 500+ req/s (single container)
- Success rate: 99.9%+

---

## Cleanup

After testing:

```bash
# Stop all services
docker-compose -f docker-compose.multitenant.yml down

# Remove volumes (to reset database)
docker-compose -f docker-compose.multitenant.yml down -v

# Remove unused Docker images
docker image prune -a --force
```

---

## Next Steps

After successful local testing:

1. ✅ Document any issues found
2. ✅ Record performance metrics
3. ✅ Create test report
4. ✅ Deploy to staging environment
5. ✅ Prepare for production deployment

---

**Local Testing Status**: Ready to begin
**Estimated Time**: 2-3 hours for complete testing
**Success Criteria**: All verification checklist items completed

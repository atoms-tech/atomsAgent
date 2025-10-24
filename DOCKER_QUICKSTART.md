# AgentAPI Docker Quick Start

**Date**: October 24, 2025
**Setup Time**: ~5 minutes
**Providers**: VertexAI (Gemini models) with dynamic model discovery

---

## Prerequisites

- Docker and Docker Compose installed
- Supabase account with PostgreSQL database
- Upstash account with Redis instance
- Google Cloud Platform (GCP) project with VertexAI enabled

---

## 1️⃣ Get Your Credentials

### Supabase
Go to https://supabase.com and get:
- **SUPABASE_URL**: `https://your-project.supabase.co`
- **SUPABASE_ANON_KEY**: Your anon public key
- **SUPABASE_SERVICE_ROLE_KEY**: Your service role key
- **DATABASE_URL**: `postgresql://postgres:password@db.your-project.supabase.co:5432/postgres`

### Upstash
Go to https://console.upstash.com and get:
- **UPSTASH_REDIS_REST_URL**: `https://your-region.upstash.io`
- **UPSTASH_REDIS_REST_TOKEN**: Your REST token

### VertexAI (Google Cloud)
1. Go to https://console.cloud.google.com
2. Enable VertexAI API
3. Create service account with VertexAI permissions
4. Download JSON key and base64 encode it:
   ```bash
   cat service-account-key.json | base64 | tr -d '\n'
   ```
5. Get:
   - **VERTEX_AI_API_KEY**: Your base64-encoded service key
   - **VERTEX_AI_PROJECT_ID**: Your GCP project ID

---

## 2️⃣ Configure Environment

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi

# Copy the example environment file
cp .env.example .env

# Edit with your credentials
nano .env
```

**Minimal `.env` (required fields only):**
```env
# Database (Supabase)
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.YOUR_PROJECT.supabase.co:5432/postgres
SUPABASE_URL=https://YOUR_PROJECT.supabase.co
SUPABASE_ANON_KEY=YOUR_ANON_KEY
SUPABASE_SERVICE_ROLE_KEY=YOUR_SERVICE_ROLE_KEY

# Redis (Upstash)
UPSTASH_REDIS_REST_URL=https://YOUR_REGION.upstash.io
UPSTASH_REDIS_REST_TOKEN=YOUR_REST_TOKEN

# VertexAI (Google Cloud) - ONLY SUPPORTED PROVIDER
VERTEX_AI_API_KEY=YOUR_BASE64_ENCODED_SERVICE_KEY
VERTEX_AI_PROJECT_ID=YOUR_GCP_PROJECT_ID
VERTEX_AI_LOCATION=us-central1  # Or your preferred region

# App Config
NODE_ENV=production
```

---

## 3️⃣ Start AgentAPI

```bash
# Build and start
docker-compose up -d

# Watch logs
docker-compose logs -f agentapi

# Stop
docker-compose down
```

---

## 4️⃣ Verify It's Running

```bash
# Health check
curl http://localhost:3284/health

# Expected response:
{
  "status": "UP",
  "components": {
    "database": "UP",
    "redis": "UP",
    "vertex_ai": "UP"
  }
}

# Check logs
docker-compose logs agentapi | tail -20

# See all containers
docker ps
```

---

## 5️⃣ Access Services

| Service | URL | Notes |
|---------|-----|-------|
| AgentAPI | http://localhost:3284 | REST API |
| FastMCP | http://localhost:8000 | Python MCP service |
| Health | http://localhost:3284/health | Status check |
| Status | http://localhost:3284/status | Detailed status |
| Models | http://localhost:3284/v1/models | List available models |

---

## 6️⃣ List Available Models

Once running, view all dynamically discovered VertexAI models:

```bash
# Without auth
curl http://localhost:3284/v1/models

# With Bearer token (if auth required)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:3284/v1/models

# Expected response:
{
  "object": "list",
  "data": [
    {
      "id": "gemini-1.5-pro",
      "object": "model",
      "created": 1234567890,
      "owned_by": "google"
    },
    {
      "id": "gemini-1.5-flash",
      "object": "model",
      "created": 1234567890,
      "owned_by": "google"
    },
    ...
  ]
}
```

---

## Common Commands

```bash
# View logs in real-time
docker-compose logs -f agentapi

# View last 50 lines
docker-compose logs --tail=50 agentapi

# Stop services (keep data)
docker-compose stop

# Start again
docker-compose start

# Full restart
docker-compose restart

# Remove everything (careful!)
docker-compose down -v

# Check resource usage
docker stats agentapi

# Shell into container
docker exec -it agentapi /bin/sh

# Test VertexAI connection
docker exec agentapi curl \
  -H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
  "https://vertex-ai.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/models"
```

---

## Troubleshooting

### "Connection refused" to Supabase
- Verify DATABASE_URL is correct
- Check Supabase project is running
- Ensure PostgreSQL port (5432) is accessible

### "Connection refused" to Upstash
- Verify UPSTASH_REDIS_REST_URL and token
- Check Upstash console shows instance is running
- Ensure no firewall blocking HTTPS (443)

### VertexAI authentication failed
- Verify VERTEX_AI_API_KEY is base64-encoded correctly
- Check GCP project ID matches VERTEX_AI_PROJECT_ID
- Verify service account has VertexAI permissions
- Test with: `curl -X POST https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/YOUR_SA@YOUR_PROJECT.iam.gserviceaccount.com:generateAccessToken`

### No models returned from /v1/models
- Check VertexAI API is enabled in GCP
- Verify service account has `aiplatform.models.list` permission
- Check VERTEX_AI_LOCATION matches available models
- View container logs: `docker-compose logs agentapi`

### Container exits immediately
```bash
# Check error logs
docker-compose logs agentapi

# Common issues:
# - Missing required env vars (DATABASE_URL, VERTEX_AI_API_KEY, etc.)
# - Invalid base64-encoded API key
# - GCP credentials invalid
# - Port 3284 already in use
```

### Port already in use
```bash
# Use environment variable to change port
export AGENTAPI_PORT=3285
docker-compose up -d
```

---

## Environment Variables Reference

### Database (Supabase)
- `DATABASE_URL`: PostgreSQL connection string (required)
- `SUPABASE_URL`: Supabase project URL
- `SUPABASE_ANON_KEY`: Supabase anon public key
- `SUPABASE_SERVICE_ROLE_KEY`: Supabase service role key

### Redis (Upstash)
- `UPSTASH_REDIS_REST_URL`: REST API endpoint
- `UPSTASH_REDIS_REST_TOKEN`: REST API token
- `REDIS_PROTOCOL`: Set to `rest` for Upstash

### VertexAI Configuration
- `VERTEX_AI_API_KEY`: Base64-encoded GCP service account key (required)
- `VERTEX_AI_PROJECT_ID`: GCP project ID (required)
- `VERTEX_AI_LOCATION`: GCP region (default: us-central1)
- `VERTEX_AI_MODEL_DISCOVERY_ENABLED`: Enable dynamic model discovery (default: true)
- `VERTEX_AI_MODEL_CACHE_TTL`: Model cache TTL (default: 3600s)

### Application
- `NODE_ENV`: `production` or `development`
- `AGENTAPI_PORT`: Server port (default: 3284)
- `AGENTAPI_ALLOWED_HOSTS`: CORS allowed hosts (default: *)
- `AGENTAPI_ALLOWED_ORIGINS`: CORS allowed origins (default: *)
- `LOG_LEVEL`: `info`, `debug`, `warn`, `error`
- `LOG_FORMAT`: `json` or `text`

### Rate Limiting
- `RATE_LIMIT_ENABLED`: Enable rate limiting (default: true)
- `RATE_LIMIT_REQUESTS_PER_MINUTE`: Requests per minute (default: 60)
- `RATE_LIMIT_BURST_SIZE`: Burst size (default: 10)

### Circuit Breaker
- `CIRCUIT_BREAKER_ENABLED`: Enable circuit breaker (default: true)
- `CIRCUIT_BREAKER_FAILURE_THRESHOLD`: Failures before open (default: 5)
- `CIRCUIT_BREAKER_SUCCESS_THRESHOLD`: Successes before closed (default: 2)
- `CIRCUIT_BREAKER_TIMEOUT`: Timeout in open state (default: 30s)

---

## GCP Service Account Setup

### 1. Create Service Account
```bash
gcloud iam service-accounts create agentapi-vertexai \
  --display-name="AgentAPI VertexAI Service Account"
```

### 2. Grant VertexAI Permissions
```bash
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:agentapi-vertexai@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/aiplatform.admin"
```

### 3. Create and Download JSON Key
```bash
gcloud iam service-accounts keys create \
  ~/agentapi-vertexai-key.json \
  --iam-account=agentapi-vertexai@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

### 4. Encode to Base64
```bash
cat ~/agentapi-vertexai-key.json | base64 | tr -d '\n' > ~/vertex-ai-api-key.txt
```

### 5. Use in .env
```bash
# Copy base64 content
VERTEX_AI_API_KEY=$(cat ~/vertex-ai-api-key.txt)

# Add to .env
echo "VERTEX_AI_API_KEY=$VERTEX_AI_API_KEY" >> .env
```

---

## Next Steps

1. **Test the API**: See `AUTHKIT_CHAT_API_IMPLEMENTATION.md` for API examples
2. **Send a request**: Use `/v1/chat/completions` endpoint (OpenAI-compatible)
3. **Frontend Integration**: Connect atoms.tech (see `QUICKSTART_LOCAL_TESTING.md`)
4. **Production Deploy**: Use this same compose file on a server with Docker
5. **Monitoring**: Enable metrics and audit logging (see `ENV_SETUP.md`)

---

## Example API Request

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello, what can you do?"}
    ],
    "temperature": 0.7,
    "max_tokens": 1024
  }'
```

---

## Support

- **VertexAI Docs**: https://cloud.google.com/vertex-ai/docs
- **GCP Console**: https://console.cloud.google.com
- **Supabase Docs**: https://supabase.com/docs
- **Upstash Docs**: https://upstash.com/docs
- **Docker Compose Docs**: https://docs.docker.com/compose/
- **AgentAPI Issues**: Check related documentation

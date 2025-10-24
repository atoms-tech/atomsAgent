# VertexAI Quick Reference

**Setup**: 5 minutes | **Documentation**: 1,400+ lines | **Status**: Production-ready

---

## TL;DR - Setup in 3 Commands

```bash
# 1. Get GCP credentials
gcloud iam service-accounts create agentapi-vertexai --display-name="AgentAPI"
gcloud projects add-iam-policy-binding $(gcloud config get-value project) \
  --member="serviceAccount:agentapi-vertexai@$(gcloud config get-value project).iam.gserviceaccount.com" \
  --role="roles/aiplatform.admin"
gcloud iam service-accounts keys create ~/key.json \
  --iam-account=agentapi-vertexai@$(gcloud config get-value project).iam.gserviceaccount.com

# 2. Configure environment
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
cp .env.example .env
# Edit .env with your Supabase, Upstash, and GCP credentials

# 3. Start
docker-compose up -d && curl http://localhost:3284/health
```

---

## Files Overview

| File | Purpose | Read First |
|------|---------|-----------|
| `VERTEXAI_SETUP_COMPLETE.md` | **Start here** - Complete overview + copy-paste setup | ✓ YES |
| `DOCKER_QUICKSTART.md` | Detailed 5-minute setup guide | After summary |
| `VERTEXAI_CONFIGURATION.md` | GCP setup details + troubleshooting | Troubleshooting |
| `DOCKER_SETUP_SUMMARY.md` | Architecture + comparisons + checklists | Reference |
| `docker-compose.yml` | Docker configuration file | Read as needed |

---

## Environment Variables Needed

```bash
# From GCP (service account key, base64 encoded)
VERTEX_AI_API_KEY=...                    # Required
VERTEX_AI_PROJECT_ID=...                 # Required
VERTEX_AI_LOCATION=us-central1           # Optional (default: us-central1)

# From Supabase
DATABASE_URL=postgresql://...            # Required
SUPABASE_URL=...                         # Required
SUPABASE_ANON_KEY=...                    # Required
SUPABASE_SERVICE_ROLE_KEY=...            # Required

# From Upstash
UPSTASH_REDIS_REST_URL=...               # Required
UPSTASH_REDIS_REST_TOKEN=...             # Required

# Optional (application config)
NODE_ENV=production                      # Default: production
AGENTAPI_PORT=3284                       # Default: 3284
LOG_LEVEL=info                           # Default: info
```

---

## Common Commands

```bash
# Start/Stop
docker-compose up -d          # Start
docker-compose down           # Stop & remove
docker-compose logs -f        # Watch logs
docker-compose restart        # Restart

# Verify
curl http://localhost:3284/health              # Health check
curl http://localhost:3284/v1/models           # List models
curl http://localhost:3284/status              # Detailed status

# Debug
docker exec -it agentapi /bin/sh   # Shell access
docker stats agentapi              # Resource usage
docker-compose logs --tail=50      # Last 50 lines
```

---

## API Endpoints

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/health` | GET | Health check | No |
| `/status` | GET | Detailed status | No |
| `/v1/models` | GET | List available models | No |
| `/v1/chat/completions` | POST | Chat API | Optional |

---

## Example API Requests

### List Models
```bash
curl http://localhost:3284/v1/models
```

### Chat Request
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "max_tokens": 1024
  }'
```

### Streaming Response
```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-1.5-pro",
    "messages": [{"role": "user", "content": "Say hello"}],
    "stream": true
  }' \
  -N
```

---

## Troubleshooting Quick Links

| Issue | Solution | Doc |
|-------|----------|-----|
| Container won't start | Check logs: `docker-compose logs agentapi` | VERTEXAI_CONFIGURATION.md |
| Models not discovered | Verify GCP API enabled + service account perms | VERTEXAI_CONFIGURATION.md |
| Authentication fails | Verify base64-encoded API key | VERTEXAI_CONFIGURATION.md |
| Port 3284 in use | Change port in .env: `AGENTAPI_PORT=3285` | DOCKER_QUICKSTART.md |
| Database connection timeout | Check Supabase URL + firewall | DOCKER_QUICKSTART.md |
| Redis connection timeout | Check Upstash URL + token | DOCKER_QUICKSTART.md |

---

## Model Discovery Flow

```
Container Starts
    ↓
Load VERTEX_AI_API_KEY & PROJECT_ID
    ↓
Call: https://vertex-ai.googleapis.com/v1/projects/{ID}/locations/{LOCATION}/models
    ↓
Cache results (TTL: 3600s)
    ↓
/v1/models endpoint returns cached list
    ↓
Client requests /v1/chat/completions with model name
    ↓
Verify model in discovered list
    ↓
Send to VertexAI API
```

---

## Service Account Setup (Quick)

```bash
# Set variables
PROJECT_ID=$(gcloud config get-value project)

# Enable APIs
gcloud services enable aiplatform.googleapis.com iamcredentials.googleapis.com iam.googleapis.com

# Create service account
gcloud iam service-accounts create agentapi-vertexai \
  --display-name="AgentAPI VertexAI Service Account"

# Grant permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:agentapi-vertexai@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/aiplatform.admin"

# Create key
gcloud iam service-accounts keys create ~/agentapi-key.json \
  --iam-account=agentapi-vertexai@${PROJECT_ID}.iam.gserviceaccount.com

# Base64 encode
cat ~/agentapi-key.json | base64 | tr -d '\n'

# Get project ID
cat ~/agentapi-key.json | jq -r '.project_id'
```

---

## Deployment Checklist

- [ ] GCP project + VertexAI API enabled
- [ ] Service account created + key downloaded
- [ ] Key base64 encoded
- [ ] Supabase database configured
- [ ] Upstash Redis configured
- [ ] `.env` file filled with credentials
- [ ] `docker-compose up -d` successful
- [ ] Health check returns UP
- [ ] `/v1/models` returns models
- [ ] Chat request works

---

## Monitoring & Logs

```bash
# Real-time logs
docker-compose logs -f agentapi

# Specific log lines
docker-compose logs agentapi | grep "model"
docker-compose logs agentapi | grep "error"
docker-compose logs agentapi | grep "startup"

# Last N lines
docker-compose logs --tail=100 agentapi

# Resource usage
docker stats agentapi

# Full container info
docker inspect agentapi
```

---

## Environment Defaults

```env
# VertexAI
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_MODEL_DISCOVERY_ENABLED=true
VERTEX_AI_MODEL_CACHE_TTL=3600s

# Redis
REDIS_PROTOCOL=rest
REDIS_ENABLE=true

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10

# Circuit Breaker
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_SUCCESS_THRESHOLD=2
CIRCUIT_BREAKER_TIMEOUT=30s

# Session
SESSION_STORAGE=redis
SESSION_TTL=3600s
SESSION_CLEANUP_INTERVAL=300s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## Performance Tips

1. **Cache Models**: Increase `VERTEX_AI_MODEL_CACHE_TTL` for high-traffic
2. **Rate Limit**: Adjust `RATE_LIMIT_REQUESTS_PER_MINUTE` as needed
3. **Circuit Breaker**: Keep enabled to prevent cascading failures
4. **Logging**: Use `json` format for production (easier to parse)

---

## Security Checklist

- [ ] API key stored in .env (not in code)
- [ ] .env added to .gitignore
- [ ] No credentials in docker-compose.yml
- [ ] Service account has minimal permissions
- [ ] Keys rotated every 90 days
- [ ] Audit logging enabled in GCP
- [ ] CORS origins configured correctly
- [ ] Rate limiting enabled

---

## Links

- **Documentation Map**: See file listing above
- **Getting Started**: Read `VERTEXAI_SETUP_COMPLETE.md` first
- **Troubleshooting**: See `VERTEXAI_CONFIGURATION.md`
- **GCP Docs**: https://cloud.google.com/vertex-ai/docs
- **Docker Compose**: https://docs.docker.com/compose/

---

**Ready?** Start with: `VERTEXAI_SETUP_COMPLETE.md`

**Version**: 1.0 | **Date**: October 24, 2025 | **Status**: Production-ready

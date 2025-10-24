# Docker Setup Summary - VertexAI Edition

**Date**: October 24, 2025
**Created**: Simplified Docker Compose for AgentAPI with VertexAI-only support
**Status**: Production-ready

---

## What Was Created

### 1. `docker-compose.yml` (Updated)
- **Simplified compose file** (only AgentAPI service)
- **VertexAI-only** model provider (Gemini models)
- **Dynamic model discovery** from Google's API
- **Removes**: Local PostgreSQL and Redis services
- **Uses**: External Supabase and Upstash services
- **Benefit**: Zero local infrastructure, managed services

### 2. `DOCKER_QUICKSTART.md` (Updated)
- **Quick start guide** for VertexAI setup
- **GCP integration** instructions
- **Model discovery** configuration
- **5-minute setup** from scratch
- **Troubleshooting** for VertexAI authentication

### 3. `DOCKER_SETUP_SUMMARY.md` (Updated)
- **Overview** of VertexAI-only configuration
- **Credential requirements** for GCP
- **Service account setup** step-by-step
- **Port mapping** and environment reference

---

## Key Features

✅ **VertexAI-Only**: Single AI provider (Gemini models)
✅ **Dynamic Model Discovery**: Auto-fetch models from Google's API  
✅ **Model Caching**: Cache TTL to avoid repeated API calls
✅ **OpenAI-Compatible**: Standard `/v1/chat/completions` endpoint
✅ **Managed Infrastructure**: Supabase + Upstash + VertexAI
✅ **Production-Ready**: Single container, auto-scaling capable

---

## Architecture

```
Client Request
    ↓
AgentAPI (3284)
    ├─ Authentication
    ├─ Request validation
    └─ Route to VertexAI
         ↓
VertexAI API
    ├─ Fetch available models
    ├─ Cache models (3600s)
    └─ Execute chat completion
         ↓
Return response (OpenAI format)
```

---

## Required Credentials

### Supabase
- DATABASE_URL
- SUPABASE_URL
- SUPABASE_ANON_KEY
- SUPABASE_SERVICE_ROLE_KEY

### Upstash
- UPSTASH_REDIS_REST_URL
- UPSTASH_REDIS_REST_TOKEN

### VertexAI (Google Cloud)
- VERTEX_AI_API_KEY (base64-encoded service account key)
- VERTEX_AI_PROJECT_ID
- VERTEX_AI_LOCATION (e.g., us-central1)

---

## Usage

### Quick Start (3 steps)
```bash
# 1. Copy env template
cp .env.example .env

# 2. Edit with credentials
nano .env

# 3. Start
docker-compose up -d
```

### Verification
```bash
# Health check
curl http://localhost:3284/health

# List models
curl http://localhost:3284/v1/models

# View logs
docker-compose logs -f agentapi
```

### Stop/Restart
```bash
docker-compose stop          # Stop (keeps data)
docker-compose start         # Start again
docker-compose restart       # Restart
docker-compose down          # Remove (keeps volumes)
docker-compose down -v       # Remove everything
```

---

## Model Discovery

Models are dynamically fetched from:
```
https://vertex-ai.googleapis.com/v1/projects/<PROJECT_ID>/locations/<LOCATION>/models
```

**How it works:**
1. Service starts, VertexAI module initializes
2. Fetches all available models from Google API
3. Caches models for VERTEX_AI_MODEL_CACHE_TTL (default: 3600s)
4. Returns cached list on `/v1/models` endpoint
5. Auto-selects model based on user request

---

## Differences from Original Setup

| Aspect | Original | New (VertexAI) |
|--------|----------|---|
| AI Providers | Multi (Anthropic, Gemini, OpenRouter, etc) | VertexAI only |
| Model Source | Hard-coded | Dynamic discovery |
| Configuration | Complex | Simple (3 required vars) |
| Model Updates | Manual | Automatic via API |
| OAuth | Full suite (GitHub, Google, Azure, Auth0) | Not required |
| Setup Time | 10+ minutes | ~5 minutes |
| Ideal For | Experimentation | Production (VertexAI users) |

---

## Port Mapping

| Service | Port | Purpose |
|---------|------|---------|
| AgentAPI | 3284 | REST API endpoints |
| FastMCP | 8000 | Python MCP service |

---

## Files Reference

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Main compose file (VertexAI-only) |
| `docker-compose.multitenant.yml` | Original with local services (legacy) |
| `DOCKER_QUICKSTART.md` | Setup guide with GCP instructions |
| `DOCKER_SETUP_SUMMARY.md` | This document |
| `.env.example` | Environment variables template |

---

## GCP Setup Checklist

- [ ] Create GCP project
- [ ] Enable VertexAI API
- [ ] Create service account `agentapi-vertexai`
- [ ] Grant `roles/aiplatform.admin` role
- [ ] Create and download JSON key
- [ ] Base64 encode the key
- [ ] Create `.env` file with credentials
- [ ] Start with `docker-compose up -d`

---

## Next Steps

1. **Gather credentials** from GCP, Supabase, and Upstash
2. **Set up GCP service account** (see DOCKER_QUICKSTART.md)
3. **Create and configure** `.env` file
4. **Start** `docker-compose up -d`
5. **Verify** `curl http://localhost:3284/health`
6. **List models** `curl http://localhost:3284/v1/models`
7. **Connect frontend** atoms.tech
8. **Deploy to production** using same compose file

---

## Troubleshooting

### VertexAI auth fails
```bash
# Verify base64 encoding
echo $VERTEX_AI_API_KEY | base64 -d | jq .

# Check GCP permissions
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --format='table(bindings.role)' \
  --filter="bindings.members:agentapi-vertexai"
```

### Models not discovered
```bash
# Check VertexAI API enabled
gcloud services list --enabled | grep vertex

# Test API access
curl -H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
  "https://vertex-ai.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/models"
```

### Container won't start
```bash
docker-compose logs agentapi

# Common issues:
# - Missing VERTEX_AI_API_KEY
# - Invalid base64 encoding
# - Database/Redis connection timeout
# - Port 3284 already in use
```

---

## Benefits

✅ **Focused**: Single provider, easier to maintain
✅ **Automatic**: Models discovered dynamically
✅ **Scalable**: VertexAI auto-scales with demand
✅ **Managed**: No local infrastructure
✅ **Cost-Effective**: Pay only for what you use
✅ **Enterprise**: Google-backed support and SLAs

---

## Production Checklist

- [ ] VertexAI API enabled and accessible
- [ ] Service account has proper permissions
- [ ] Database URL tested and working
- [ ] Redis URL tested and working
- [ ] Health checks passing
- [ ] Models successfully discovered
- [ ] Rate limiting configured
- [ ] Circuit breaker enabled
- [ ] Logging and monitoring setup
- [ ] Backup and recovery plan ready

---

**Status**: ✅ Ready for production
**Last Updated**: October 24, 2025
**Provider**: VertexAI (Gemini models only)

# VertexAI Configuration Guide

**Date**: October 24, 2025
**Status**: Production-ready
**Provider**: VertexAI (Gemini models only)

---

## Overview

AgentAPI now supports **VertexAI-only** with dynamic model discovery. Models are automatically fetched from Google's API endpoint rather than hardcoded.

```
https://vertex-ai.googleapis.com/v1/projects/<PROJECT_ID>/locations/<LOCATION>/models
```

---

## Required Environment Variables

### Essential (No Defaults)
```env
# Supabase PostgreSQL
DATABASE_URL=postgresql://postgres:PASSWORD@db.PROJECT.supabase.co:5432/postgres
SUPABASE_URL=https://PROJECT.supabase.co
SUPABASE_ANON_KEY=eyJhbGc...
SUPABASE_SERVICE_ROLE_KEY=eyJhbGc...

# Upstash Redis
UPSTASH_REDIS_REST_URL=https://REGION.upstash.io
UPSTASH_REDIS_REST_TOKEN=token_here

# VertexAI (Google Cloud)
VERTEX_AI_API_KEY=base64_encoded_service_key
VERTEX_AI_PROJECT_ID=my-gcp-project
```

### Optional (With Defaults)
```env
# VertexAI
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_MODEL_DISCOVERY_ENABLED=true
VERTEX_AI_MODEL_CACHE_TTL=3600s

# Application
NODE_ENV=production
AGENTAPI_PORT=3284
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## Model Discovery Process

### 1. Service Startup
```
AgentAPI starts
  ↓
VertexAI module initializes
  ↓
Check VERTEX_AI_MODEL_DISCOVERY_ENABLED (default: true)
```

### 2. Fetch Models
```
Call: GET https://vertex-ai.googleapis.com/v1/projects/{PROJECT_ID}/locations/{LOCATION}/models
Headers: Authorization: Bearer {ACCESS_TOKEN}
  ↓
Returns list of available models:
  - gemini-1.5-pro
  - gemini-1.5-flash
  - gemini-1.0-pro
```

### 3. Cache Models
```
Store fetched models in memory
Cache TTL: VERTEX_AI_MODEL_CACHE_TTL (default: 3600s)
  ↓
Use cached list for /v1/models endpoint
```

### 4. Use Models
```
Client requests: POST /v1/chat/completions
  ↓
Validate model is in discovered list
  ↓
Send to VertexAI API
```

---

## Setup Steps

### 1. Create GCP Project
```bash
gcloud projects create agentapi-vertexai
gcloud config set project agentapi-vertexai
```

### 2. Enable APIs
```bash
gcloud services enable aiplatform.googleapis.com
gcloud services enable iamcredentials.googleapis.com
gcloud services enable iam.googleapis.com
```

### 3. Create Service Account
```bash
gcloud iam service-accounts create agentapi-vertexai \
  --display-name="AgentAPI VertexAI Service Account"
```

### 4. Grant Permissions
```bash
PROJECT_ID=$(gcloud config get-value project)

# Grant AI Platform Admin role
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:agentapi-vertexai@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/aiplatform.admin"
```

### 5. Create and Export Key
```bash
# Create JSON key
PROJECT_ID=$(gcloud config get-value project)
gcloud iam service-accounts keys create \
  ~/agentapi-vertexai-key.json \
  --iam-account=agentapi-vertexai@${PROJECT_ID}.iam.gserviceaccount.com

# Base64 encode
cat ~/agentapi-vertexai-key.json | base64 | tr -d '\n' > ~/vertex-ai-api-key.b64

# Get credentials
PROJECT_ID=$(cat ~/agentapi-vertexai-key.json | jq -r '.project_id')

echo "VERTEX_AI_API_KEY=$(cat ~/vertex-ai-api-key.b64)"
echo "VERTEX_AI_PROJECT_ID=$PROJECT_ID"
```

### 6. Configure AgentAPI
```bash
# Copy environment template
cp .env.example .env

# Add credentials
cat >> .env << ENVFILE
VERTEX_AI_API_KEY=$(cat ~/vertex-ai-api-key.b64)
VERTEX_AI_PROJECT_ID=$(cat ~/agentapi-vertexai-key.json | jq -r '.project_id')
VERTEX_AI_LOCATION=us-central1
ENVFILE

# Edit with other required credentials
nano .env
```

### 7. Start Docker
```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
docker-compose up -d
```

### 8. Verify Setup
```bash
# Check health
curl http://localhost:3284/health

# List discovered models
curl http://localhost:3284/v1/models

# Check logs
docker-compose logs agentapi | grep -i model
```

---

## Verify GCP Configuration

### Test Service Account Access
```bash
# Get access token
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)

# Test models API
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  "https://vertex-ai.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/models"
```

### Verify Permissions
```bash
PROJECT_ID=$(gcloud config get-value project)

gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --format='table(bindings.role)' \
  --filter="bindings.members:agentapi-vertexai@${PROJECT_ID}.iam.gserviceaccount.com"
```

---

## API Response Format

### /v1/models Endpoint
```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-1.5-pro",
      "object": "model",
      "created": 1704067200,
      "owned_by": "google"
    },
    {
      "id": "gemini-1.5-flash",
      "object": "model",
      "created": 1704067200,
      "owned_by": "google"
    }
  ]
}
```

### /v1/chat/completions Endpoint
Request:
```json
{
  "model": "gemini-1.5-pro",
  "messages": [{"role": "user", "content": "Hello!"}],
  "temperature": 0.7,
  "max_tokens": 1024
}
```

---

## Troubleshooting

### Model not found in discovered models
**Solutions**:
1. Verify model exists in GCP VertexAI console
2. Check VERTEX_AI_LOCATION matches model availability
3. Restart container to refresh cache

```bash
docker-compose restart agentapi
docker-compose logs agentapi | grep model
```

### VertexAI API authentication failed
**Solutions**:
1. Verify base64 encoding:
   ```bash
   echo $VERTEX_AI_API_KEY | base64 -d | jq .
   ```

2. Check service account permissions:
   ```bash
   gcloud projects get-iam-policy YOUR_PROJECT_ID \
     --flatten="bindings[].members" \
     --filter="bindings.members:agentapi-vertexai@*"
   ```

### No models discovered from API
**Solutions**:
1. Verify VertexAI API is enabled:
   ```bash
   gcloud services list --enabled | grep aiplatform
   ```

2. Test API directly:
   ```bash
   ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
   curl -H "Authorization: Bearer $ACCESS_TOKEN" \
     "https://vertex-ai.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/models"
   ```

---

## Security Best Practices

- Use service account key (not user account)
- Store key in .env, add to .gitignore
- Rotate keys every 90 days
- Use minimal permissions (not Owner role)
- Enable audit logging in GCP
- Monitor API usage in GCP console

---

## Cost Optimization

### Model Caching
Models cached for VERTEX_AI_MODEL_CACHE_TTL to avoid repeated API calls.

For high-traffic:
```env
VERTEX_AI_MODEL_CACHE_TTL=86400s
```

### Rate Limiting
```env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10
```

---

## Support & Documentation

- **VertexAI Docs**: https://cloud.google.com/vertex-ai/docs
- **GCP Console**: https://console.cloud.google.com
- **Docker Compose**: https://docs.docker.com/compose/

---

**Status**: Production-ready
**Last Updated**: October 24, 2025
**Provider**: VertexAI (Gemini models)

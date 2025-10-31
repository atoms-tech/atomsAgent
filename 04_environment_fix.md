# Environment and Authentication Fix Instructions

## Issue Identified
- `invalid_rapt` error occurs when GCP credentials are not properly loaded by chatserver
- Circuit breaker opens due to repeated authentication failures

## Solution: Proper GCP Credential Setup

### Option 1: Use Service Account Key (Recommended for production)

1. **Create Service Account Key**:
```bash
gcloud iam service-accounts keys create ~/vertexai-service-key.json \
  --iam-account=agentapi-vertexai@serious-mile-462615-a2.iam.gserviceaccount.com
```

2. **Base64 encode the key**:
```bash
export VERTEX_AI_API_KEY=$(base64 -i ~/vertexai-service-key.json)
```

3. **Update your .env file**:
```bash
# Add these to .env
VERTEX_AI_USE_APPLICATION_DEFAULT=false
VERTEX_AI_API_KEY=<base64_encoded_key_from_step_2>
```

### Option 2: Fix Application Default Credentials

1. **Set correct environment variable**:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=$HOME/.config/gcloud/application_default_credentials.json
```

2. **Verify credentials work**:
```bash
gcloud auth application-default print-access-token
```

3. **Restart chatserver with proper env**:
```bash
# Stop current chatserver
pkill -f chatserver

# Start with credentials
export GOOGLE_APPLICATION_CREDENTIALS=$HOME/.config/gcloud/application_default_credentials.json
./chatserver
```

### Option 3: Use Docker Compose with Proper Environment

Update your `docker-compose.yml`:
```yaml
services:
  chatserver:
    # ... existing config ...
    environment:
      - GOOGLE_APPLICATION_CREDENTIALS=/app/config/gcp-credentials.json
    volumes:
      - ~/.config/gcloud/application_default_credentials.json:/app/config/gcp-credentials.json:ro
```

## Verification Steps

1. **Check logs after restart**:
```bash
# Look for these positive signs:
# - "models listed" without errors
# - No "invalid_rapt" errors
# - No "circuit breaker is open" errors
```

2. **Test authentication**:
```bash
# This should succeed without invalid_rapt error
curl http://localhost:3284/v1/models

# Test with proper auth header
curl -H "Authorization: Bearer <your_jwt_token>" \
  -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-4.5-haiku","messages":[{"role":"user","content":"test"}],"max_tokens":10}'
```

## Circuit Breaker Reset

After fixing authentication, the circuit breaker should automatically reset. If it remains open:

1. **Wait for timeout** (default: 60 seconds)
2. **Restart chatserver** (immediate reset):
```bash
pkill -f chatserver
./chatserver
```

## Required GCP Permissions

Ensure your service account has:
- `roles/aiplatform.user` - For model access
- `roles/aiplatform.admin` - For model management
- `roles/iam.serviceAccountUser` - For credential management

```bash
# Verify permissions
gcloud projects get-iam-policy serious-mile-462615-a2 \
  --filter="serviceAccount:agentapi-vertexai@serious-mile-462615-a2.iam.gserviceaccount.com"
```

## Common Issues & Solutions

| Issue | Cause | Solution |
|--------|--------|-----------|
| `invalid_rapt` | Expired/invalid credentials | Refresh or recreate service account key |
| `permission denied` | Missing IAM roles | Add required roles to service account |
| `project not found` | Wrong project ID | Verify `gcloud config get-value project` |
| `circuit breaker open` | Repeated failures | Fix auth then restart service |

## Quick Test Script

```bash
#!/bin/bash
# Test if VertexAI auth is working
echo "Testing VertexAI authentication..."

# Test 1: GCP credentials
if gcloud auth application-default print-access-token > /dev/null 2>&1; then
    echo "✅ GCP credentials valid"
else
    echo "❌ GCP credentials invalid - run fix"
fi

# Test 2: Chat API
RESPONSE=$(curl -s http://localhost:3284/v1/models)
if echo "$RESPONSE" | grep -q "claude-4.5-haiku"; then
    echo "✅ Chat API responding"
else
    echo "❌ Chat API not working - check logs"
fi

# Test 3: Check for errors in logs
if docker logs chatserver 2>&1 | grep -q "invalid_rapt"; then
    echo "❌ invalid_rapt error still present - needs auth fix"
else
    echo "✅ No invalid_rapt errors detected"
fi
```

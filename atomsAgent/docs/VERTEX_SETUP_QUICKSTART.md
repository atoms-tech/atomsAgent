# Vertex AI Setup - Quick Start

## TL;DR - Fix Authentication Issues

If you're seeing `invalid_rapt` errors repeatedly, run this:

```bash
./atomsAgent/scripts/setup-service-account.sh
```

This will create a service account and configure it properly.

## Why Service Accounts?

**User credentials** (what you have now):
- ❌ Expire frequently
- ❌ Require reauthentication
- ❌ Not suitable for servers
- ❌ Cause `invalid_rapt` errors

**Service accounts** (recommended):
- ✅ Never expire
- ✅ No reauthentication needed
- ✅ Designed for servers
- ✅ More secure

## Quick Setup (3 steps)

### 1. Run the setup script

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
./atomsAgent/scripts/setup-service-account.sh
```

### 2. Verify it works

```bash
cd atomsAgent
uv run python -c "
import asyncio
from atomsAgent.dependencies import get_vertex_model_service

async def test():
    service = get_vertex_model_service()
    await service._cache.clear()
    result = await service.list_models()
    print(f'✓ Got {len(result.data)} models')

asyncio.run(test())
"
```

### 3. Start the server

```bash
cd atomsAgent
uv run uvicorn atomsAgent.main:app --host 0.0.0.0 --port 3284
```

Test the endpoint:
```bash
curl http://localhost:3284/v1/models | jq .
```

## What the Script Does

1. Creates a service account: `atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com`
2. Grants it the `roles/aiplatform.user` role
3. Creates a key file at `atomsAgent/config/vertex-service-account.json`
4. Updates `atomsAgent/config/secrets.yml` with the credentials

## Manual Alternative

If you prefer to do it manually:

```bash
# 1. Create service account
gcloud iam service-accounts create atomsagent-vertex \
    --project=serious-mile-462615-a2

# 2. Grant permissions
gcloud projects add-iam-policy-binding serious-mile-462615-a2 \
    --member="serviceAccount:atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"

# 3. Create key
gcloud iam service-accounts keys create atomsAgent/config/vertex-service-account.json \
    --iam-account=atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com

# 4. Update secrets.yml
# Edit atomsAgent/config/secrets.yml and set:
# vertex_credentials_path: "atomsAgent/config/vertex-service-account.json"
```

## Troubleshooting

### "gcloud: command not found"

Install the Google Cloud SDK:
```bash
# macOS
brew install google-cloud-sdk

# Or download from:
# https://cloud.google.com/sdk/docs/install
```

### "Permission denied"

Make sure you're authenticated with gcloud:
```bash
gcloud auth login
gcloud config set project serious-mile-462615-a2
```

### "Service account already exists"

That's fine! The script will use the existing one. Just create a new key:
```bash
gcloud iam service-accounts keys create atomsAgent/config/vertex-service-account.json \
    --iam-account=atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com
```

### Still getting errors?

See the full troubleshooting guide: [VERTEX_AUTH_TROUBLESHOOTING.md](./VERTEX_AUTH_TROUBLESHOOTING.md)

## Security Notes

- ⚠️ The service account key file is sensitive - never commit it to git
- ✅ It's already in `.gitignore`
- ✅ Rotate keys periodically (every 90 days recommended)
- ✅ Use Secret Manager in production environments

## Next Steps

Once authentication is working:

1. Test the `/v1/models` endpoint - it now fetches models dynamically from Vertex AI
2. The models list will always be up-to-date with what's available in Model Garden
3. No more hardcoded model lists!

## Need Help?

- Full troubleshooting guide: [VERTEX_AUTH_TROUBLESHOOTING.md](./VERTEX_AUTH_TROUBLESHOOTING.md)
- Vertex AI docs: https://cloud.google.com/vertex-ai/docs/authentication
- Service accounts: https://cloud.google.com/iam/docs/service-accounts


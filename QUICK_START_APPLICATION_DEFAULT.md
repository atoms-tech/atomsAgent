# QuickStart - Application Default Credentials

**Status**: Ready to use  
**Auth Method**: gcloud CLI (no API key needed)  
**Time**: 5 minutes

---

## Prerequisites

- ✅ gcloud CLI installed
- ✅ Logged into GCP with your account
- ✅ Supabase credentials ready
- ✅ Upstash credentials ready

---

## Step 1: Authenticate with gcloud

```bash
# Login to your Google account
gcloud auth application-default login

# This opens a browser to authenticate
# After auth, credentials are stored locally

# Verify it worked
gcloud auth list
gcloud config get-value project
```

---

## Step 2: Update .env with Supabase & Upstash

```bash
nano .env
```

Find and update these lines:

```env
# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=eyJhbGc...
SUPABASE_SERVICE_ROLE_KEY=eyJhbGc...
DATABASE_URL=postgresql://postgres:PASSWORD@db.your-project.supabase.co:5432/postgres

# Upstash
UPSTASH_REDIS_REST_URL=https://your-region.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-token

# VertexAI (already configured)
VERTEX_AI_USE_APPLICATION_DEFAULT=true
VERTEX_AI_PROJECT_ID=serious-mile-462615-a2
VERTEX_AI_LOCATION=us-central1
```

**Note**: VERTEX_AI_API_KEY should stay empty - we're using gcloud login instead

---

## Step 3: Start Docker

```bash
docker-compose up -d
```

**Important**: Make sure `~/.config/gcloud/` is available (where gcloud stores auth)

---

## Step 4: Verify It's Working

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

# List models
curl http://localhost:3284/v1/models

# View logs
docker-compose logs -f agentapi
```

---

## How This Works

```
You run: gcloud auth application-default login
  ↓
Stores credentials in: ~/.config/gcloud/application_default_credentials.json
  ↓
Docker container can access this file
  ↓
VertexAI API uses your authenticated Google account
  ↓
No need for service account key!
```

---

## Troubleshooting

### "Credentials not found"
```bash
# Make sure you ran:
gcloud auth application-default login

# Verify credentials exist:
cat ~/.config/gcloud/application_default_credentials.json
```

### "Permission denied"
```bash
# Your Google account may need VertexAI permissions
# Add role: roles/aiplatform.admin to your user account
# Go to: https://console.cloud.google.com/iam-admin/iam
```

### Docker can't access gcloud credentials
```bash
# Ensure .config/gcloud is accessible by Docker
# May need to mount the directory:
# -v ~/.config/gcloud:/root/.config/gcloud:ro

# Or set environment variable:
# GOOGLE_APPLICATION_CREDENTIALS=/path/to/application_default_credentials.json
```

---

## Environment Variables

**Minimal setup needed**:
```env
# VertexAI (using Application Default Credentials)
VERTEX_AI_USE_APPLICATION_DEFAULT=true
VERTEX_AI_PROJECT_ID=serious-mile-462615-a2

# Supabase
SUPABASE_URL=...
SUPABASE_ANON_KEY=...
SUPABASE_SERVICE_ROLE_KEY=...
DATABASE_URL=...

# Upstash
UPSTASH_REDIS_REST_URL=...
UPSTASH_REDIS_REST_TOKEN=...
```

---

## Advantages

✅ No API key files to manage  
✅ Uses your Google account credentials  
✅ Avoids GCP policy restrictions  
✅ Works locally on your machine  
✅ Same credentials as `gcloud` CLI  

---

## Next Steps

1. Run: `gcloud auth application-default login`
2. Edit `.env` with Supabase + Upstash creds
3. Run: `docker-compose up -d`
4. Verify: `curl http://localhost:3284/health`

**Done!** AgentAPI is running with VertexAI models 🚀

---

## Reference

- **gcloud docs**: https://cloud.google.com/docs/authentication
- **Application Default Credentials**: https://cloud.google.com/docs/authentication/application-default-credentials
- **VertexAI docs**: https://cloud.google.com/vertex-ai/docs

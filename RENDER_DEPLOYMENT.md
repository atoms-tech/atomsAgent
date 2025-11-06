# Render Deployment Guide for atomsAgent

This guide covers deploying the atomsAgent service to Render with the custom domain `ai.atoms.tech`.

## Prerequisites

- Access to the support@atoms.tech Render account
- Repository pushed to GitHub: https://github.com/KooshaPari/agentapi
- Domain `ai.atoms.tech` DNS access for CNAME configuration

## Deployment Methods

### Method 1: Deploy via Render Dashboard (Recommended)

1. **Login to Render**
   - Go to https://dashboard.render.com
   - Sign in with support@atoms.tech

2. **Create New Web Service from Blueprint**
   - Click "New +" → "Blueprint"
   - Connect your GitHub repository: `KooshaPari/agentapi`
   - Render will automatically detect the `render.yaml` file
   - Click "Apply" to create the service

3. **Add Secret Environment Variables**

   After the service is created, go to the service's Environment settings and add:

   ```bash
   # Vertex AI Credentials (paste the entire JSON content from atcred.json as a single-line string)
   ATOMS_SECRET_VERTEX_CREDENTIALS_JSON='<paste-vertex-credentials-json-here>'

   # Supabase Configuration (get from config/secrets.yml)
   ATOMS_SECRET_SUPABASE_URL='<your-supabase-url>'
   ATOMS_SECRET_SUPABASE_ANON_KEY='<your-supabase-anon-key>'
   ATOMS_SECRET_SUPABASE_SERVICE_KEY='<your-supabase-service-role-key>'

   # Authentication (get from config/secrets.yml)
   ATOMS_SECRET_AUTHKIT_JWKS_URL='<your-authkit-jwks-url>'

   # Token Encryption (Base64 encoded key from config/secrets.yml)
   ATOMS_SECRET_TOKEN_ENCRYPTION_KEY='<your-token-encryption-key>'

   # Static API Key (for testing/development, get from config/secrets.yml)
   ATOMS_SECRET_STATIC_API_KEY='<your-static-api-key>'
   ```

   **Note**: Copy the actual values from your local `config/secrets.yml` and `atcred.json` files.

4. **Trigger Deployment**
   - After adding the secrets, click "Save Changes"
   - This will trigger a new deployment with all environment variables

### Method 2: Deploy via Render CLI

1. **Install Render CLI**
   ```bash
   brew install render  # macOS
   # or
   npm install -g @render-cli/cli
   ```

2. **Login to Render**
   ```bash
   render login
   # Follow the prompts to authenticate with support@atoms.tech
   ```

3. **Deploy from render.yaml**
   ```bash
   cd /path/to/agentapi
   render blueprint launch
   ```

4. **Add secrets via CLI**
   ```bash
   render env set ATOMS_SECRET_VERTEX_CREDENTIALS_JSON='...' --service atoms-agent-api
   render env set ATOMS_SECRET_SUPABASE_URL='...' --service atoms-agent-api
   # ... add all other secrets
   ```

## Configure Custom Domain (ai.atoms.tech)

### Step 1: Add Custom Domain in Render

1. Go to your service in the Render dashboard
2. Navigate to **Settings** → **Custom Domains**
3. Click **Add Custom Domain**
4. Enter: `ai.atoms.tech`
5. Render will provide you with a CNAME target (e.g., `atoms-agent-api.onrender.com`)

### Step 2: Configure DNS

Add a CNAME record in your DNS provider for the domain `atoms.tech`:

```
Type:  CNAME
Name:  ai
Value: atoms-agent-api.onrender.com
TTL:   Auto or 3600
```

**Popular DNS Providers:**

- **Cloudflare**: DNS → Records → Add Record
- **Namecheap**: Domain List → Manage → Advanced DNS → Add New Record
- **Google Domains**: DNS → Custom records → Manage custom records
- **Route53**: Hosted Zones → Create Record

### Step 3: Verify Domain

1. Wait for DNS propagation (usually 5-30 minutes)
2. Check DNS propagation: `dig ai.atoms.tech` or use https://dnschecker.org
3. Once propagated, Render will automatically provision an SSL certificate
4. Your service will be accessible at: https://ai.atoms.tech

## Service Configuration Details

| Setting | Value |
|---------|-------|
| Service Type | Web Service |
| Runtime | Python 3.10.12 |
| Region | Oregon (us-west-2) |
| Plan | Starter ($7/month) |
| Root Directory | `atomsAgent` |
| Build Command | `pip install uv && uv pip install --system .` |
| Start Command | `uvicorn atomsAgent.main:app --host 0.0.0.0 --port $PORT` |
| Health Check | `/health` |
| Auto Deploy | Yes (on git push to main) |

## Testing the Deployment

Once deployed, test the service:

```bash
# Check health endpoint
curl https://ai.atoms.tech/health

# Test the API (if health check is implemented)
curl https://ai.atoms.tech/v1/models

# Test with API key
curl -X POST https://ai.atoms.tech/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-api-key-development" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Monitoring

- **Logs**: View real-time logs in the Render dashboard under the "Logs" tab
- **Metrics**: Monitor CPU, memory, and response times in the "Metrics" tab
- **Events**: Check deployment history and events in the "Events" tab

## Troubleshooting

### Build Failures

If the build fails:
1. Check the build logs in the Render dashboard
2. Verify all environment variables are set correctly
3. Ensure the `rootDir` is set to `atomsAgent`

### Service Won't Start

If the service builds but won't start:
1. Check the service logs for errors
2. Verify the start command is correct
3. Ensure all required environment variables are set

### Domain Not Resolving

If the custom domain isn't working:
1. Verify DNS CNAME record is correct
2. Wait for DNS propagation (up to 48 hours, usually faster)
3. Check DNS propagation with `dig ai.atoms.tech`
4. Ensure SSL certificate is provisioned (check Render dashboard)

## Scaling

To scale the service:
1. Go to **Settings** → **Instance**
2. Upgrade to a higher plan (Standard, Pro, etc.)
3. Or scale horizontally by increasing instance count (Pro plans and above)

## Cost Estimate

- **Starter Plan**: $7/month per instance
- **Bandwidth**: 100 GB/month included, $0.10/GB after
- **Custom Domain**: Free (SSL included)

## Security Notes

- All secrets are stored encrypted in Render
- SSL/TLS is automatically provisioned and renewed
- Service runs in an isolated container
- Network access is controlled via security groups

## Support

For issues with the deployment:
- Render Documentation: https://render.com/docs
- Render Support: support@render.com
- Project Issues: https://github.com/KooshaPari/agentapi/issues

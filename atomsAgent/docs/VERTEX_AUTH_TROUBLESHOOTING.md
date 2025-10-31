# Vertex AI Authentication Troubleshooting

This guide helps resolve authentication issues when using Vertex AI with atomsAgent.

## Common Issue: `invalid_rapt` Error

If you're repeatedly seeing the `invalid_rapt` error, it means your user credentials (OAuth2) require reauthentication due to Google's security policies.

### Why This Happens

- **User credentials** (`authorized_user` type) are meant for interactive use and expire frequently
- They require periodic reauthentication, especially with 2FA enabled
- They're not suitable for server applications

### Recommended Solution: Use Service Account

Service accounts are designed for server-to-server authentication and don't expire like user credentials.

## Setup Options

### Option 1: Automated Setup (Recommended)

Run the provided setup script:

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
./atomsAgent/scripts/setup-service-account.sh
```

This script will:
1. Create a service account named `atomsagent-vertex`
2. Grant it the `roles/aiplatform.user` role
3. Create and download a key file
4. Update your `secrets.yml` configuration

### Option 2: Manual Setup

#### Step 1: Create Service Account

```bash
# Set your project
export PROJECT_ID="serious-mile-462615-a2"

# Create service account
gcloud iam service-accounts create atomsagent-vertex \
    --display-name="AtomAgent Vertex AI Service Account" \
    --project=${PROJECT_ID}

# Grant IAM role
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:atomsagent-vertex@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"
```

#### Step 2: Create and Download Key

```bash
# Create key file
gcloud iam service-accounts keys create ~/atomsagent-vertex-key.json \
    --iam-account=atomsagent-vertex@${PROJECT_ID}.iam.gserviceaccount.com

# Move to config directory
mv ~/atomsagent-vertex-key.json atomsAgent/config/vertex-service-account.json
```

#### Step 3: Update Configuration

Edit `atomsAgent/config/secrets.yml`:

```yaml
vertex_credentials_path: "atomsAgent/config/vertex-service-account.json"
# Remove or comment out vertex_credentials_json if using path
```

Or, to embed the credentials directly in secrets.yml:

```yaml
vertex_credentials_json: |
  {
    "type": "service_account",
    "project_id": "serious-mile-462615-a2",
    "private_key_id": "...",
    "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
    "client_email": "atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com",
    "client_id": "...",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_x509_cert_url": "..."
  }
```

### Option 3: Use Application Default Credentials (ADC)

If you're running on GCP (Cloud Run, GKE, etc.), you can use ADC:

```bash
# Remove credentials from secrets.yml
# The SDK will automatically use the compute engine service account
```

## Verification

Test your setup:

```bash
cd atomsAgent
uv run python -c "
import asyncio
from atomsAgent.dependencies import get_vertex_model_service

async def test():
    service = get_vertex_model_service()
    await service._cache.clear()
    result = await service.list_models()
    print(f'✓ Successfully fetched {len(result.data)} models')
    for model in result.data[:3]:
        print(f'  - {model.id}')

asyncio.run(test())
"
```

Expected output:
```
✓ Successfully fetched 6 models
  - publishers/google/models/gemini-2.0-flash-exp
  - publishers/google/models/gemini-2.0-flash-thinking-exp
  - publishers/google/models/gemini-2.0-pro-exp
```

## Troubleshooting

### Error: "No access token available"

**Cause**: Credentials are not properly configured or the google-auth library is not installed.

**Solution**:
```bash
cd atomsAgent
uv pip install google-auth requests
```

### Error: "Permission denied"

**Cause**: Service account doesn't have the required IAM role.

**Solution**:
```bash
gcloud projects add-iam-policy-binding serious-mile-462615-a2 \
    --member="serviceAccount:atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"
```

### Error: "Service account does not exist"

**Cause**: Service account hasn't been created yet.

**Solution**: Run the setup script or follow the manual setup steps above.

### Error: "Vertex API error: 404"

**Cause**: The Vertex AI API is not enabled or the endpoint is incorrect.

**Solution**:
```bash
gcloud services enable aiplatform.googleapis.com --project=serious-mile-462615-a2
```

## Security Best Practices

1. **Never commit service account keys to version control**
   - Add `*.json` to `.gitignore` in the config directory
   - Use environment variables or secret managers in production

2. **Rotate keys periodically**
   ```bash
   # Delete old key
   gcloud iam service-accounts keys delete KEY_ID \
       --iam-account=atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com
   
   # Create new key
   gcloud iam service-accounts keys create atomsAgent/config/vertex-service-account.json \
       --iam-account=atomsagent-vertex@serious-mile-462615-a2.iam.gserviceaccount.com
   ```

3. **Use least privilege**
   - Only grant `roles/aiplatform.user` (not `roles/owner` or `roles/editor`)
   - Consider creating a custom role with only required permissions:
     - `aiplatform.endpoints.predict`

4. **Use Secret Manager in production**
   ```bash
   # Store key in Secret Manager
   gcloud secrets create vertex-service-account-key \
       --data-file=atomsAgent/config/vertex-service-account.json
   
   # Grant access to your application's service account
   gcloud secrets add-iam-policy-binding vertex-service-account-key \
       --member="serviceAccount:your-app@project.iam.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"
   ```

## Additional Resources

- [Google Cloud Service Accounts](https://cloud.google.com/iam/docs/service-accounts)
- [Vertex AI Authentication](https://cloud.google.com/vertex-ai/docs/authentication)
- [IAM Roles for Vertex AI](https://cloud.google.com/vertex-ai/docs/general/access-control)
- [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials)


#!/bin/bash
set -e

# Configuration
PROJECT_ID="serious-mile-462615-a2"
SERVICE_ACCOUNT_NAME="atomsagent-vertex"
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
KEY_FILE="atomsAgent/config/vertex-service-account.json"

echo "=== Setting up Vertex AI Service Account ==="
echo ""
echo "Project ID: ${PROJECT_ID}"
echo "Service Account: ${SERVICE_ACCOUNT_EMAIL}"
echo ""

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "ERROR: gcloud CLI is not installed."
    echo "Please install it from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Set the project
echo "Setting GCP project..."
gcloud config set project ${PROJECT_ID}

# Check if service account already exists
if gcloud iam service-accounts describe ${SERVICE_ACCOUNT_EMAIL} --project=${PROJECT_ID} &> /dev/null; then
    echo "Service account already exists: ${SERVICE_ACCOUNT_EMAIL}"
else
    echo "Creating service account..."
    gcloud iam service-accounts create ${SERVICE_ACCOUNT_NAME} \
        --display-name="AtomAgent Vertex AI Service Account" \
        --description="Service account for AtomAgent to access Vertex AI models" \
        --project=${PROJECT_ID}
    echo "✓ Service account created"

    # Wait for service account to propagate
    echo "Waiting for service account to propagate..."
    sleep 5
fi

# Grant IAM roles
echo ""
echo "Granting IAM roles..."
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
    --role="roles/aiplatform.user" \
    > /dev/null 2>&1 || echo "Role may already be assigned"
echo "✓ IAM roles granted"

# Create key file
echo ""
echo "Creating service account key..."
if [ -f "${KEY_FILE}" ]; then
    echo "WARNING: Key file already exists at ${KEY_FILE}"
    read -p "Do you want to overwrite it? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Skipping key creation. Using existing key."
        exit 0
    fi
    rm "${KEY_FILE}"
fi

# Retry key creation with exponential backoff
MAX_RETRIES=5
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if gcloud iam service-accounts keys create "${KEY_FILE}" \
        --iam-account="${SERVICE_ACCOUNT_EMAIL}" \
        --project=${PROJECT_ID} 2>/dev/null; then
        echo "✓ Service account key created at: ${KEY_FILE}"
        break
    else
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            WAIT_TIME=$((2 ** RETRY_COUNT))
            echo "Service account not ready yet, waiting ${WAIT_TIME} seconds... (attempt $RETRY_COUNT/$MAX_RETRIES)"
            sleep $WAIT_TIME
        else
            echo "ERROR: Failed to create service account key after $MAX_RETRIES attempts"
            echo "The service account may need more time to propagate."
            echo "Please wait a minute and run this command manually:"
            echo "  gcloud iam service-accounts keys create ${KEY_FILE} --iam-account=${SERVICE_ACCOUNT_EMAIL} --project=${PROJECT_ID}"
            exit 1
        fi
    fi
done

# Update secrets.yml
echo ""
echo "Updating configuration..."
SECRETS_FILE="atomsAgent/config/secrets.yml"

if [ -f "${SECRETS_FILE}" ]; then
    # Read the key file content
    KEY_CONTENT=$(cat "${KEY_FILE}")
    
    # Create a temporary Python script to update the YAML
    python3 << EOF
import yaml
import json

# Read the service account key
with open('${KEY_FILE}', 'r') as f:
    key_data = json.load(f)

# Read existing secrets
try:
    with open('${SECRETS_FILE}', 'r') as f:
        secrets = yaml.safe_load(f) or {}
except FileNotFoundError:
    secrets = {}

# Update vertex credentials
secrets['vertex_credentials_json'] = json.dumps(key_data)
secrets['vertex_credentials_path'] = '${KEY_FILE}'

# Write back
with open('${SECRETS_FILE}', 'w') as f:
    yaml.dump(secrets, f, default_flow_style=False, sort_keys=False)

print("✓ Updated ${SECRETS_FILE}")
EOF
else
    echo "WARNING: ${SECRETS_FILE} not found. Please update it manually."
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Service account credentials have been configured."
echo "You can now use the Vertex AI models API."
echo ""
echo "To test the setup, run:"
echo "  cd atomsAgent && uv run python -c 'import asyncio; from atomsAgent.dependencies import get_vertex_model_service; asyncio.run(get_vertex_model_service().list_models())'"


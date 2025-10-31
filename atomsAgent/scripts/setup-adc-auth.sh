#\!/bin/bash
set -e

echo "=== Setting up Application Default Credentials ==="
echo ""
echo "This will use your user credentials for Vertex AI access."
echo "This is suitable for development but not recommended for production."
echo ""

# Authenticate with gcloud
echo "Authenticating with gcloud..."
gcloud auth application-default login --project=serious-mile-462615-a2

echo ""
echo "✓ Application Default Credentials configured"
echo ""
echo "Now updating atomsAgent configuration..."

# Update secrets.yml to remove explicit credentials
SECRETS_FILE="atomsAgent/config/secrets.yml"

if [ -f "${SECRETS_FILE}" ]; then
    # Create a backup
    cp "${SECRETS_FILE}" "${SECRETS_FILE}.backup"
    
    # Use Python to update the YAML
    python3 << 'PYTHON_EOF'
import yaml
import os

secrets_file = "atomsAgent/config/secrets.yml"

try:
    with open(secrets_file, 'r') as f:
        secrets = yaml.safe_load(f) or {}
except FileNotFoundError:
    secrets = {}

# Remove explicit credentials - will use ADC instead
if 'vertex_credentials_json' in secrets:
    del secrets['vertex_credentials_json']
if 'vertex_credentials_path' in secrets:
    del secrets['vertex_credentials_path']

# Ensure project and location are set
if 'vertex_project_id' not in secrets:
    secrets['vertex_project_id'] = 'serious-mile-462615-a2'
if 'vertex_location' not in secrets:
    secrets['vertex_location'] = 'us-central1'

# Write back
with open(secrets_file, 'w') as f:
    yaml.dump(secrets, f, default_flow_style=False, sort_keys=False)

print(f"✓ Updated {secrets_file}")
print("  - Removed explicit credentials")
print("  - Will use Application Default Credentials")
PYTHON_EOF

    echo ""
    echo "Backup created at: ${SECRETS_FILE}.backup"
else
    echo "WARNING: ${SECRETS_FILE} not found"
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Your credentials are now configured to use Application Default Credentials."
echo "This uses your personal gcloud authentication."
echo ""
echo "To test, run:"
echo "  cd atomsAgent && uv run python -c 'import asyncio; from atomsAgent.dependencies import get_vertex_model_service; asyncio.run(get_vertex_model_service().list_models())'"
echo ""
echo "Note: You'll need to run 'gcloud auth application-default login' again if:"
echo "  - Your credentials expire"
echo "  - You switch to a different machine"
echo "  - You want to use a different account"

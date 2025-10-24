#!/bin/bash
# Setup GCP Service Account for VertexAI
# Run this script to create a service account with proper credentials

set -e

PROJECT_ID="serious-mile-462615-a2"
SERVICE_ACCOUNT_NAME="agentapi-vertexai"
DISPLAY_NAME="AgentAPI VertexAI Service Account"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║          GCP Service Account Setup for AgentAPI                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Project ID: $PROJECT_ID"
echo "Service Account: $SERVICE_ACCOUNT_NAME"
echo ""

# Step 1: Set project
echo "1️⃣  Setting GCP project..."
gcloud config set project $PROJECT_ID

# Step 2: Enable required APIs
echo ""
echo "2️⃣  Enabling VertexAI APIs..."
gcloud services enable aiplatform.googleapis.com
gcloud services enable iamcredentials.googleapis.com
gcloud services enable iam.googleapis.com

# Step 3: Create service account
echo ""
echo "3️⃣  Creating service account..."
gcloud iam service-accounts create $SERVICE_ACCOUNT_NAME \
  --display-name="$DISPLAY_NAME" \
  --description="Service account for AgentAPI VertexAI integration" \
  2>/dev/null || echo "   (Service account may already exist, continuing...)"

# Step 4: Grant permissions
echo ""
echo "4️⃣  Granting VertexAI Admin permissions..."
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/aiplatform.admin" \
  --quiet

# Step 5: Create and download key
echo ""
echo "5️⃣  Creating and downloading service account key..."
KEY_FILE="$HOME/agentapi-vertexai-key.json"
gcloud iam service-accounts keys create "$KEY_FILE" \
  --iam-account="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "   ✅ Key saved to: $KEY_FILE"

# Step 6: Encode to base64
echo ""
echo "6️⃣  Encoding key to base64..."
BASE64_KEY=$(cat "$KEY_FILE" | base64 | tr -d '\n')

echo "   ✅ Base64 encoded (length: ${#BASE64_KEY} chars)"

# Step 7: Display instructions
echo ""
echo "════════════════════════════════════════════════════════════════"
echo "✅ SERVICE ACCOUNT CREATED SUCCESSFULLY"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "📋 NEXT STEPS:"
echo ""
echo "1. Copy your base64 API key:"
echo "   $BASE64_KEY"
echo ""
echo "2. Add to .env file:"
echo "   VERTEX_AI_API_KEY=$BASE64_KEY"
echo ""
echo "3. Or run setup script:"
echo "   bash SETUP_ENV.sh"
echo ""
echo "4. Then start Docker:"
echo "   docker-compose up -d"
echo ""
echo "📁 Files:"
echo "   • Key file: $KEY_FILE"
echo "   • Project: $PROJECT_ID"
echo "   • Service account: ${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
echo ""
echo "════════════════════════════════════════════════════════════════"

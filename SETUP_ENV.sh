#!/bin/bash
# Setup .env file with your credentials
# Run this script and paste your credentials when prompted

set -e

ENV_FILE=".env"

echo "========================================"
echo "AgentAPI VertexAI Setup - Credentials"
echo "========================================"
echo ""

# Read credentials
read -p "Enter VERTEX_AI_API_KEY (base64-encoded GCP service key): " VERTEX_AI_API_KEY
read -p "Enter SUPABASE_URL: " SUPABASE_URL
read -p "Enter SUPABASE_ANON_KEY: " SUPABASE_ANON_KEY
read -p "Enter SUPABASE_SERVICE_ROLE_KEY: " SUPABASE_SERVICE_ROLE_KEY
read -p "Enter DATABASE_URL (PostgreSQL connection string): " DATABASE_URL
read -p "Enter UPSTASH_REDIS_REST_URL: " UPSTASH_REDIS_REST_URL
read -p "Enter UPSTASH_REDIS_REST_TOKEN: " UPSTASH_REDIS_REST_TOKEN

# Update .env file
sed -i '' "s|VERTEX_AI_API_KEY=.*|VERTEX_AI_API_KEY=$VERTEX_AI_API_KEY|" "$ENV_FILE"
sed -i '' "s|SUPABASE_URL=.*|SUPABASE_URL=$SUPABASE_URL|" "$ENV_FILE"
sed -i '' "s|SUPABASE_ANON_KEY=.*|SUPABASE_ANON_KEY=$SUPABASE_ANON_KEY|" "$ENV_FILE"
sed -i '' "s|SUPABASE_SERVICE_ROLE_KEY=.*|SUPABASE_SERVICE_ROLE_KEY=$SUPABASE_SERVICE_ROLE_KEY|" "$ENV_FILE"
sed -i '' "s|DATABASE_URL=.*|DATABASE_URL=$DATABASE_URL|" "$ENV_FILE"
sed -i '' "s|UPSTASH_REDIS_REST_URL=.*|UPSTASH_REDIS_REST_URL=$UPSTASH_REDIS_REST_URL|" "$ENV_FILE"
sed -i '' "s|UPSTASH_REDIS_REST_TOKEN=.*|UPSTASH_REDIS_REST_TOKEN=$UPSTASH_REDIS_REST_TOKEN|" "$ENV_FILE"

echo ""
echo "✅ .env file updated!"
echo ""
echo "Ready to start? Run:"
echo "  docker-compose up -d"
echo ""

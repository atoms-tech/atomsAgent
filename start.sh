#!/bin/bash
# AgentAPI Startup Script
# Loads environment from .env and starts the chatserver

set -a
source .env
set +a

# Ensure key variables are set
if [ -z "$AUTHKIT_JWKS_URL" ]; then
  echo "❌ Error: AUTHKIT_JWKS_URL not set in .env"
  exit 1
fi

if [ -z "$SUPABASE_URL" ]; then
  echo "❌ Error: SUPABASE_URL not set in .env"
  exit 1
fi

if [ -z "$SUPABASE_SERVICE_ROLE_KEY" ]; then
  echo "❌ Error: SUPABASE_SERVICE_ROLE_KEY not set in .env"
  exit 1
fi

echo "🚀 Starting AgentAPI Chat Server..."
echo "   AUTHKIT_JWKS_URL: $AUTHKIT_JWKS_URL"
echo "   SUPABASE_URL: $SUPABASE_URL"
echo "   CCROUTER_PATH: $CCROUTER_PATH"
echo "   PRIMARY_AGENT: $PRIMARY_AGENT"
echo ""

./chatserver

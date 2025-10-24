#!/bin/bash
# Setup AgentAPI schema in Supabase

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}AgentAPI Supabase Schema Setup${NC}"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found${NC}"
    exit 1
fi

# Load DATABASE_URL from .env
DATABASE_URL=$(grep "^DATABASE_URL=" .env | cut -d'=' -f2-)

if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}Error: DATABASE_URL not found in .env${NC}"
    exit 1
fi

echo "Database: $DATABASE_URL"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo -e "${RED}Error: psql not installed${NC}"
    echo "Install with: brew install postgresql"
    exit 1
fi

echo -e "${BLUE}Applying schema...${NC}"
echo ""

# Apply schema
psql "$DATABASE_URL" -f database/agent_system_schema.sql

echo ""
echo -e "${GREEN}✅ Schema applied successfully!${NC}"
echo ""
echo "Created tables:"
echo "  - agents"
echo "  - models"
echo "  - chat_sessions"
echo "  - chat_messages"
echo "  - agent_executions"
echo "  - agent_health"
echo "  - agent_metrics"
echo "  - circuit_breaker_state"
echo ""
echo "Created views:"
echo "  - v_recent_sessions"
echo "  - v_agent_status"
echo ""
echo "Ready to start AgentAPI!"

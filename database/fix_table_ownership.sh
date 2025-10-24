#!/bin/bash
# ============================================================================
# Fix Table Ownership for AgentAPI Tables
# ============================================================================
# This script transfers table ownership from supabase_read_only_user to postgres
#
# Usage:
#   bash fix_table_ownership.sh
#
# The script will use your DATABASE_URL from .env
# ============================================================================

set -e

# Load environment variables
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found"
    exit 1
fi

set -a
source .env
set +a

if [ -z "$DATABASE_URL" ]; then
    echo "❌ Error: DATABASE_URL not found in .env"
    exit 1
fi

echo "🔧 Fixing table ownership for AgentAPI tables..."
echo "Database: $DATABASE_URL"
echo ""

# Create SQL script
SQL_SCRIPT=$(cat <<'EOF'
-- Transfer table ownership from supabase_read_only_user to postgres
ALTER TABLE agents OWNER TO postgres;
ALTER TABLE models OWNER TO postgres;
ALTER TABLE chat_sessions OWNER TO postgres;
ALTER TABLE chat_messages OWNER TO postgres;
ALTER TABLE agent_health OWNER TO postgres;

-- Verify ownership
SELECT
    tablename,
    tableowner
FROM pg_tables
WHERE schemaname = 'public'
AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages', 'agent_health')
ORDER BY tablename;
EOF
)

# Execute the SQL script
echo "$SQL_SCRIPT" | psql "$DATABASE_URL" -v ON_ERROR_STOP=1

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Table ownership fixed successfully!"
    echo ""
    echo "Tables are now owned by: postgres"
else
    echo ""
    echo "❌ Error: Failed to fix table ownership"
    exit 1
fi

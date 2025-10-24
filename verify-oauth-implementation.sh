#!/bin/bash

# OAuth Implementation Verification Script
# Checks that all required files and configurations are in place

set -e

echo "==================================="
echo "OAuth Implementation Verification"
echo "==================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

check_file() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $description"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} $description - Missing: $file"
        FAILED=$((FAILED + 1))
    fi
}

check_directory() {
    local dir=$1
    local description=$2
    
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} $description"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} $description - Missing: $dir"
        FAILED=$((FAILED + 1))
    fi
}

echo "Checking Core API Files..."
echo "-------------------------"
check_file "api/mcp/oauth/callback.ts" "OAuth callback handler"
check_file "api/mcp/oauth/initiate.ts" "OAuth initiation endpoint"
check_file "api/mcp/oauth/refresh.ts" "Token refresh endpoint"
check_file "api/mcp/oauth/revoke.ts" "Token revocation endpoint"
check_file "api/mcp/oauth/helpers.ts" "Helper functions"
echo ""

echo "Checking Configuration Files..."
echo "-------------------------------"
check_file "api/mcp/oauth/.env.example" "Environment template"
check_file "api/package.json" "Package dependencies"
check_file "api/tsconfig.json" "TypeScript configuration"
echo ""

echo "Checking Database Files..."
echo "-------------------------"
check_file "database/schema.sql" "Base database schema"
check_file "database/migrations/002_oauth_tables.sql" "OAuth tables migration"
echo ""

echo "Checking Documentation..."
echo "------------------------"
check_file "api/mcp/oauth/README.md" "API documentation"
check_file "OAUTH_IMPLEMENTATION_GUIDE.md" "Implementation guide"
check_file "OAUTH_FILES_SUMMARY.md" "Files summary"
check_file "OAUTH_QUICK_REFERENCE.md" "Quick reference"
echo ""

echo "Checking Test Files..."
echo "---------------------"
check_directory "api/mcp/oauth/__tests__" "Tests directory"
check_file "api/mcp/oauth/__tests__/oauth.test.ts" "Test suite"
echo ""

echo "Checking File Sizes..."
echo "---------------------"
if [ -f "api/mcp/oauth/callback.ts" ]; then
    LINES=$(wc -l < "api/mcp/oauth/callback.ts")
    if [ "$LINES" -gt 400 ]; then
        echo -e "${GREEN}✓${NC} callback.ts has $LINES lines (expected >400)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} callback.ts has only $LINES lines (expected >400)"
        FAILED=$((FAILED + 1))
    fi
fi

if [ -f "api/mcp/oauth/helpers.ts" ]; then
    LINES=$(wc -l < "api/mcp/oauth/helpers.ts")
    if [ "$LINES" -gt 400 ]; then
        echo -e "${GREEN}✓${NC} helpers.ts has $LINES lines (expected >400)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} helpers.ts has only $LINES lines (expected >400)"
        FAILED=$((FAILED + 1))
    fi
fi
echo ""

echo "Checking for Required Functions..."
echo "----------------------------------"
if grep -q "function encrypt" api/mcp/oauth/helpers.ts 2>/dev/null; then
    echo -e "${GREEN}✓${NC} encrypt() function found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} encrypt() function missing"
    FAILED=$((FAILED + 1))
fi

if grep -q "function decrypt" api/mcp/oauth/helpers.ts 2>/dev/null; then
    echo -e "${GREEN}✓${NC} decrypt() function found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} decrypt() function missing"
    FAILED=$((FAILED + 1))
fi

if grep -q "function generatePKCE" api/mcp/oauth/helpers.ts 2>/dev/null; then
    echo -e "${GREEN}✓${NC} generatePKCE() function found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} generatePKCE() function missing"
    FAILED=$((FAILED + 1))
fi

if grep -q "exchangeCodeForTokens" api/mcp/oauth/callback.ts 2>/dev/null; then
    echo -e "${GREEN}✓${NC} exchangeCodeForTokens() function found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} exchangeCodeForTokens() function missing"
    FAILED=$((FAILED + 1))
fi
echo ""

echo "Checking Database Schema..."
echo "--------------------------"
if grep -q "CREATE TABLE oauth_states" database/migrations/002_oauth_tables.sql 2>/dev/null; then
    echo -e "${GREEN}✓${NC} oauth_states table definition found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} oauth_states table definition missing"
    FAILED=$((FAILED + 1))
fi

if grep -q "CREATE TABLE mcp_oauth_tokens" database/migrations/002_oauth_tables.sql 2>/dev/null; then
    echo -e "${GREEN}✓${NC} mcp_oauth_tokens table definition found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} mcp_oauth_tokens table definition missing"
    FAILED=$((FAILED + 1))
fi

if grep -q "cleanup_expired_oauth_states" database/migrations/002_oauth_tables.sql 2>/dev/null; then
    echo -e "${GREEN}✓${NC} cleanup_expired_oauth_states() function found"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} cleanup_expired_oauth_states() function missing"
    FAILED=$((FAILED + 1))
fi
echo ""

echo "==================================="
echo "Results"
echo "==================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ "$FAILED" -eq 0 ]; then
    echo -e "${GREEN}All checks passed! ✓${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Set environment variables (copy .env.example)"
    echo "2. Generate encryption key: openssl rand -hex 32"
    echo "3. Run database migration: 002_oauth_tables.sql"
    echo "4. Configure OAuth providers"
    echo "5. Deploy to Vercel"
    exit 0
else
    echo -e "${RED}Some checks failed. Please review the output above.${NC}"
    exit 1
fi

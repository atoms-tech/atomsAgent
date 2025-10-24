#!/bin/bash
# Verification script for FastMCP Service setup

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}FastMCP Service - Setup Verification${NC}"
echo "========================================"
echo ""

# Check Python version
echo -n "Checking Python version... "
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 -c 'import sys; print(".".join(map(str, sys.version_info[:2])))')
    if [ "$(printf '%s\n' "3.8" "$PYTHON_VERSION" | sort -V | head -n1)" = "3.8" ]; then
        echo -e "${GREEN}✓${NC} Python $PYTHON_VERSION"
    else
        echo -e "${RED}✗${NC} Python $PYTHON_VERSION (3.8+ required)"
        exit 1
    fi
else
    echo -e "${RED}✗${NC} Python 3 not found"
    exit 1
fi

# Check pip
echo -n "Checking pip... "
if command -v pip &> /dev/null || command -v pip3 &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC} pip not found"
    exit 1
fi

# Check required files
echo ""
echo "Checking required files:"

FILES=(
    "fastmcp_service.py"
    "test_fastmcp_service.py"
    "example_usage.py"
    "requirements.txt"
    "Dockerfile"
    "docker-compose.yml"
    "Makefile"
    "start.sh"
    "FASTMCP_SERVICE_README.md"
    "API_REFERENCE.md"
    "QUICKSTART.md"
)

for file in "${FILES[@]}"; do
    echo -n "  $file... "
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        exit 1
    fi
done

# Check if files are executable
echo ""
echo "Checking executable permissions:"

EXEC_FILES=(
    "fastmcp_service.py"
    "example_usage.py"
    "start.sh"
)

for file in "${EXEC_FILES[@]}"; do
    echo -n "  $file... "
    if [ -x "$file" ]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${YELLOW}⚠${NC} Not executable (fixing...)"
        chmod +x "$file"
        echo -e "    ${GREEN}✓${NC} Fixed"
    fi
done

# Check Python imports
echo ""
echo "Checking Python dependencies:"

MODULES=(
    "fastapi"
    "uvicorn"
    "pydantic"
    "httpx"
    "fastmcp"
)

for module in "${MODULES[@]}"; do
    echo -n "  $module... "
    if python3 -c "import $module" 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${YELLOW}⚠${NC} Not installed"
        MISSING_DEPS=1
    fi
done

if [ -n "$MISSING_DEPS" ]; then
    echo ""
    echo -e "${YELLOW}Some dependencies are missing. Install with:${NC}"
    echo "  pip install -r requirements.txt"
    echo ""
fi

# Check Docker (optional)
echo ""
echo "Checking optional tools:"

echo -n "  Docker... "
if command -v docker &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} Not installed (optional)"
fi

echo -n "  docker-compose... "
if command -v docker-compose &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} Not installed (optional)"
fi

echo -n "  make... "
if command -v make &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} Not installed (optional)"
fi

# Syntax check
echo ""
echo "Performing syntax checks:"

echo -n "  fastmcp_service.py... "
if python3 -m py_compile fastmcp_service.py 2>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC} Syntax error"
    exit 1
fi

echo -n "  test_fastmcp_service.py... "
if python3 -m py_compile test_fastmcp_service.py 2>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC} Syntax error"
    exit 1
fi

echo -n "  example_usage.py... "
if python3 -m py_compile example_usage.py 2>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC} Syntax error"
    exit 1
fi

# Summary
echo ""
echo "========================================"
echo -e "${GREEN}✓ Setup verification complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Install dependencies: pip install -r requirements.txt"
echo "  2. Start the service: python fastmcp_service.py"
echo "  3. Check health: curl http://localhost:8080/health"
echo "  4. View API docs: http://localhost:8080/docs"
echo ""
echo "Quick commands:"
echo "  make install  - Install dependencies"
echo "  make dev      - Run in development mode"
echo "  make test     - Run tests"
echo ""
echo "Documentation:"
echo "  - QUICKSTART.md - Quick start guide"
echo "  - FASTMCP_SERVICE_README.md - Full documentation"
echo "  - API_REFERENCE.md - API reference"
echo ""

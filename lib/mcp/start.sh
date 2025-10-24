#!/bin/bash
# Startup script for FastMCP Service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8080}"
WORKERS="${WORKERS:-4}"
LOG_LEVEL="${LOG_LEVEL:-info}"

echo -e "${GREEN}Starting FastMCP Service${NC}"
echo "================================"
echo "Host: $HOST"
echo "Port: $PORT"
echo "Workers: $WORKERS"
echo "Log Level: $LOG_LEVEL"
echo "================================"

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: Python 3 is not installed${NC}"
    exit 1
fi

# Check Python version
PYTHON_VERSION=$(python3 -c 'import sys; print(".".join(map(str, sys.version_info[:2])))')
REQUIRED_VERSION="3.8"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$PYTHON_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo -e "${RED}Error: Python $REQUIRED_VERSION or higher is required (found $PYTHON_VERSION)${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Python version: $PYTHON_VERSION${NC}"

# Check if dependencies are installed
echo "Checking dependencies..."

if ! python3 -c "import fastapi" 2>/dev/null; then
    echo -e "${YELLOW}Warning: FastAPI not found. Installing dependencies...${NC}"
    pip install -r requirements.txt
fi

if ! python3 -c "import fastmcp" 2>/dev/null; then
    echo -e "${YELLOW}Warning: FastMCP not found. Installing dependencies...${NC}"
    pip install -r requirements.txt
fi

echo -e "${GREEN}✓ Dependencies installed${NC}"

# Create logs directory
mkdir -p logs

# Check if port is available
if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}Warning: Port $PORT is already in use${NC}"
    echo "Do you want to continue? (y/N)"
    read -r response
    if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        exit 1
    fi
fi

# Determine if we should use Gunicorn or Uvicorn directly
if command -v gunicorn &> /dev/null && [ "$WORKERS" -gt 1 ]; then
    echo -e "${GREEN}Starting with Gunicorn ($WORKERS workers)...${NC}"
    exec gunicorn fastmcp_service:app \
        --workers "$WORKERS" \
        --worker-class uvicorn.workers.UvicornWorker \
        --bind "$HOST:$PORT" \
        --log-level "$LOG_LEVEL" \
        --access-logfile logs/access.log \
        --error-logfile logs/error.log
else
    echo -e "${GREEN}Starting with Uvicorn...${NC}"
    exec python3 fastmcp_service.py \
        --host "$HOST" \
        --port "$PORT"
fi

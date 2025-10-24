#!/bin/bash
# AgentAPI Run Script - Supports both Docker and binary modes

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found${NC}"
    echo "Please create .env file first:"
    echo "  cp .env.example .env"
    echo "  nano .env"
    exit 1
fi

# Show usage
if [ "$#" -eq 0 ]; then
    cat << USAGE
${BLUE}AgentAPI Run Script${NC}

Usage: bash run.sh [OPTION]

Options:
  docker    - Run with Docker (RECOMMENDED)
  binary    - Run chatserver binary directly
  help      - Show this help message

Examples:
  bash run.sh docker
  bash run.sh binary

${GREEN}RECOMMENDED:${NC} Use Docker unless you have a specific reason not to

USAGE
    exit 1
fi

case "$1" in
    docker)
        echo -e "${BLUE}Starting AgentAPI with Docker...${NC}"
        echo ""
        docker-compose up -d
        echo ""
        sleep 2
        echo -e "${GREEN}✅ AgentAPI started!${NC}"
        echo ""
        echo "Check status:"
        echo "  docker-compose logs -f agentapi"
        echo ""
        echo "Test endpoints:"
        echo "  curl http://localhost:3284/health"
        echo "  curl http://localhost:3284/v1/models"
        ;;
    
    binary)
        echo -e "${BLUE}Starting AgentAPI binary...${NC}"
        echo ""
        
        # Load environment variables
        echo "Loading .env variables..."
        set -a
        source .env
        set +a
        
        echo -e "${GREEN}✅ Environment loaded${NC}"
        echo ""
        echo "Starting chatserver on port ${AGENTAPI_PORT:-3284}..."
        echo ""
        
        # Check if chatserver binary exists
        if [ ! -f ./chatserver ]; then
            echo -e "${RED}Error: chatserver binary not found${NC}"
            echo "Build it first:"
            echo "  go build -o chatserver ./cmd/chatserver"
            exit 1
        fi
        
        # Run chatserver
        exec ./chatserver
        ;;
    
    help)
        bash run.sh
        ;;
    
    *)
        echo -e "${RED}Unknown option: $1${NC}"
        bash run.sh
        exit 1
        ;;
esac

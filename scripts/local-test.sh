#!/bin/bash

# ============================================================================
# AgentAPI Local Testing Script
# ============================================================================
# Comprehensive local testing of backend + frontend integration
# Usage: ./scripts/local-test.sh
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ATOMS_TECH_ROOT="/Users/kooshapari/temp-prodvercel/485/clean/deploy/atoms.tech"
BACKEND_PORT=3284
FRONTEND_PORT=3000
DOCKER_COMPOSE_FILE="$PROJECT_ROOT/docker-compose.multitenant.yml"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# ============================================================================
# Phase 1: Prerequisites Check
# ============================================================================

print_status "Phase 1: Checking Prerequisites"

# Check Docker
if ! command_exists docker; then
    print_error "Docker is not installed"
    exit 1
fi
print_success "Docker installed"

# Check Docker Compose
if ! command_exists docker-compose; then
    print_error "Docker Compose is not installed"
    exit 1
fi
print_success "Docker Compose installed"

# Check Go
if ! command_exists go; then
    print_warning "Go is not installed (needed for backend tests)"
else
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go $GO_VERSION installed"
fi

# Check Node/Bun
if ! command_exists bun; then
    if ! command_exists npm; then
        print_error "Neither Bun nor npm is installed"
        exit 1
    fi
    print_warning "Using npm instead of bun"
    NODE_PACKAGE_MANAGER="npm"
else
    print_success "Bun installed"
    NODE_PACKAGE_MANAGER="bun"
fi

# Check curl
if ! command_exists curl; then
    print_error "curl is not installed"
    exit 1
fi
print_success "curl installed"

# ============================================================================
# Phase 2: Backend Setup
# ============================================================================

print_status "Phase 2: Setting up Backend"

cd "$PROJECT_ROOT"

# Check .env file
if [ ! -f .env ]; then
    print_warning ".env file not found, copying from .env.docker"
    cp .env.docker .env
    print_status "Created .env from .env.docker"
    print_warning "Please review and update .env with correct values"
fi

print_status "Building Docker image..."
if ./build-multitenant.sh > /dev/null 2>&1; then
    print_success "Docker image built successfully"
else
    print_error "Failed to build Docker image"
    exit 1
fi

print_status "Starting Docker services..."
docker-compose -f "$DOCKER_COMPOSE_FILE" down -v > /dev/null 2>&1 || true
docker-compose -f "$DOCKER_COMPOSE_FILE" up -d > /dev/null 2>&1

# Wait for services to be ready
print_status "Waiting for services to be ready (30 seconds)..."
sleep 30

# Verify services are running
print_status "Verifying services..."
docker-compose -f "$DOCKER_COMPOSE_FILE" ps

# ============================================================================
# Phase 3: Backend Health Checks
# ============================================================================

print_status "Phase 3: Backend Health Checks"

# Check health endpoint
max_attempts=10
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -s "http://localhost:$BACKEND_PORT/health" > /dev/null 2>&1; then
        print_success "Backend health endpoint responding"
        break
    fi
    attempt=$((attempt + 1))
    if [ $attempt -lt $max_attempts ]; then
        sleep 2
    fi
done

if [ $attempt -eq $max_attempts ]; then
    print_error "Backend health endpoint not responding"
    docker-compose -f "$DOCKER_COMPOSE_FILE" logs agentapi | tail -20
    exit 1
fi

# Check readiness
if curl -s "http://localhost:$BACKEND_PORT/ready" > /dev/null 2>&1; then
    print_success "Backend ready probe passing"
else
    print_error "Backend ready probe failing"
fi

# Check liveness
if curl -s "http://localhost:$BACKEND_PORT/live" > /dev/null 2>&1; then
    print_success "Backend liveness probe passing"
else
    print_error "Backend liveness probe failing"
fi

# ============================================================================
# Phase 4: Backend Tests
# ============================================================================

print_status "Phase 4: Running Backend Tests"

if command_exists go; then
    print_status "Running Go unit tests..."
    if go test ./... -timeout 30s > /tmp/go_tests.log 2>&1; then
        # Count tests
        TEST_COUNT=$(grep -c "^=== RUN" /tmp/go_tests.log || echo "0")
        print_success "Go unit tests passed ($TEST_COUNT tests)"
    else
        print_error "Go unit tests failed"
        tail -30 /tmp/go_tests.log
        print_warning "Continuing with remaining tests..."
    fi
else
    print_warning "Skipping Go tests (Go not installed)"
fi

# ============================================================================
# Phase 5: Frontend Setup
# ============================================================================

print_status "Phase 5: Setting up Frontend (atoms.tech)"

if [ ! -d "$ATOMS_TECH_ROOT" ]; then
    print_error "atoms.tech directory not found at $ATOMS_TECH_ROOT"
    exit 1
fi

cd "$ATOMS_TECH_ROOT"

# Check if on correct branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "feature/phase2-oauth-integration" ]; then
    print_status "Switching to feature/phase2-oauth-integration branch..."
    git checkout feature/phase2-oauth-integration
fi
print_success "On branch: $CURRENT_BRANCH"

# Check .env.local
if [ ! -f .env.local ]; then
    print_error ".env.local not found in atoms.tech"
    exit 1
fi
print_success ".env.local exists"

# Install dependencies
print_status "Installing frontend dependencies..."
if [ "$NODE_PACKAGE_MANAGER" = "bun" ]; then
    if bun install > /tmp/bun_install.log 2>&1; then
        print_success "Frontend dependencies installed"
    else
        print_error "Failed to install frontend dependencies"
        tail -20 /tmp/bun_install.log
        exit 1
    fi
else
    if npm install > /tmp/npm_install.log 2>&1; then
        print_success "Frontend dependencies installed"
    else
        print_error "Failed to install frontend dependencies"
        tail -20 /tmp/npm_install.log
        exit 1
    fi
fi

# ============================================================================
# Phase 6: Frontend Build Test
# ============================================================================

print_status "Phase 6: Frontend Build Test"

if [ "$NODE_PACKAGE_MANAGER" = "bun" ]; then
    if bun run build > /tmp/bun_build.log 2>&1; then
        print_success "Frontend build successful"
    else
        print_error "Frontend build failed"
        tail -30 /tmp/bun_build.log
    fi
else
    if npm run build > /tmp/npm_build.log 2>&1; then
        print_success "Frontend build successful"
    else
        print_error "Frontend build failed"
        tail -30 /tmp/npm_build.log
    fi
fi

# ============================================================================
# Phase 7: Integration Test Preparation
# ============================================================================

print_status "Phase 7: Integration Test Information"

print_status "Backend is running on http://localhost:$BACKEND_PORT"
print_status "Frontend will run on http://localhost:$FRONTEND_PORT"
print_status "PostgreSQL on localhost:5432"
print_status "Redis on localhost:6379"

# ============================================================================
# Phase 8: Start Frontend in Background
# ============================================================================

print_status "Phase 8: Starting Frontend Development Server"

if [ "$NODE_PACKAGE_MANAGER" = "bun" ]; then
    # Start backend in background
    cd "$ATOMS_TECH_ROOT"
    bun dev > /tmp/frontend.log 2>&1 &
    FRONTEND_PID=$!
    print_success "Frontend server started (PID: $FRONTEND_PID)"
else
    cd "$ATOMS_TECH_ROOT"
    npm run dev > /tmp/frontend.log 2>&1 &
    FRONTEND_PID=$!
    print_success "Frontend server started (PID: $FRONTEND_PID)"
fi

# Wait for frontend to start
print_status "Waiting for frontend to start..."
sleep 15

# Check frontend is responding
if curl -s "http://localhost:$FRONTEND_PORT" > /dev/null 2>&1; then
    print_success "Frontend responding on port $FRONTEND_PORT"
else
    print_warning "Frontend may not be ready yet"
fi

# ============================================================================
# Phase 9: Manual Testing Instructions
# ============================================================================

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}       LOCAL TESTING ENVIRONMENT READY${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${GREEN}Backend (AgentAPI):${NC}"
echo "  - URL: http://localhost:$BACKEND_PORT"
echo "  - Health: http://localhost:$BACKEND_PORT/health"
echo "  - API Docs: http://localhost:$BACKEND_PORT/swagger"
echo "  - Metrics: http://localhost:$BACKEND_PORT:9090/metrics"
echo ""

echo -e "${GREEN}Frontend (atoms.tech):${NC}"
echo "  - URL: http://localhost:$FRONTEND_PORT"
echo "  - OAuth Test: http://localhost:$FRONTEND_PORT/mcp/oauth"
echo "  - MCP Config: http://localhost:$FRONTEND_PORT/mcp/configurations"
echo ""

echo -e "${GREEN}Database:${NC}"
echo "  - PostgreSQL: localhost:5432"
echo "  - Database: agentapi"
echo "  - User: agentapi"
echo "  - Command: psql -h localhost -U agentapi -d agentapi"
echo ""

echo -e "${GREEN}Cache:${NC}"
echo "  - Redis: localhost:6379"
echo "  - Test: redis-cli -h localhost ping"
echo ""

echo -e "${GREEN}Manual Tests to Run:${NC}"
echo "  1. Visit http://localhost:$FRONTEND_PORT"
echo "  2. Navigate to /mcp/oauth"
echo "  3. Select GitHub provider"
echo "  4. Authorize with OAuth"
echo "  5. Verify token storage"
echo "  6. Create MCP configuration"
echo "  7. Test connection"
echo ""

echo -e "${YELLOW}To stop frontend server:${NC}"
echo "  kill $FRONTEND_PID"
echo ""

echo -e "${YELLOW}To view logs:${NC}"
echo "  Backend:  docker-compose -f $DOCKER_COMPOSE_FILE logs -f agentapi"
echo "  Frontend: tail -f /tmp/frontend.log"
echo "  Tests:    tail -f /tmp/go_tests.log"
echo ""

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo ""

# ============================================================================
# Keep script running
# ============================================================================

print_status "Press Ctrl+C to stop all services"

trap cleanup INT

cleanup() {
    print_status "Cleaning up..."
    kill $FRONTEND_PID 2>/dev/null || true
    docker-compose -f "$DOCKER_COMPOSE_FILE" down
    print_success "Cleanup complete"
    exit 0
}

# Keep script running
while true; do
    sleep 1
done

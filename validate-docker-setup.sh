#!/bin/bash

# ==============================================================================
# Docker Setup Validation Script for AgentAPI Multi-Tenant
# ==============================================================================
# This script validates that your environment is ready for Docker deployment

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
ERRORS=0
WARNINGS=0
PASSED=0

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
}

print_check() {
    echo -ne "${BLUE}Checking $1...${NC} "
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}"
    if [ -n "$1" ]; then
        echo -e "${RED}  → $1${NC}"
    fi
    ((ERRORS++))
}

print_warn() {
    echo -e "${YELLOW}⚠ WARNING${NC}"
    if [ -n "$1" ]; then
        echo -e "${YELLOW}  → $1${NC}"
    fi
    ((WARNINGS++))
}

print_info() {
    echo -e "${BLUE}  ℹ $1${NC}"
}

# Validation functions
check_docker() {
    print_check "Docker installation"
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
        print_pass
        print_info "Docker version: $DOCKER_VERSION"
    else
        print_fail "Docker is not installed"
        print_info "Install from: https://docs.docker.com/get-docker/"
        return 1
    fi
}

check_docker_compose() {
    print_check "Docker Compose installation"
    if docker-compose --version &> /dev/null; then
        COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}' | sed 's/,//')
        print_pass
        print_info "Docker Compose version: $COMPOSE_VERSION"
    else
        print_fail "Docker Compose is not installed"
        print_info "Install from: https://docs.docker.com/compose/install/"
        return 1
    fi
}

check_docker_running() {
    print_check "Docker daemon status"
    if docker info &> /dev/null; then
        print_pass
    else
        print_fail "Docker daemon is not running"
        print_info "Start Docker Desktop or run: sudo systemctl start docker"
        return 1
    fi
}

check_disk_space() {
    print_check "Available disk space"
    REQUIRED_GB=10

    if command -v df &> /dev/null; then
        AVAILABLE_GB=$(df -BG . | tail -1 | awk '{print $4}' | sed 's/G//')
        if [ "$AVAILABLE_GB" -gt "$REQUIRED_GB" ]; then
            print_pass
            print_info "Available: ${AVAILABLE_GB}GB (minimum: ${REQUIRED_GB}GB)"
        else
            print_fail "Insufficient disk space"
            print_info "Available: ${AVAILABLE_GB}GB, Required: ${REQUIRED_GB}GB"
            return 1
        fi
    else
        print_warn "Cannot determine disk space"
    fi
}

check_memory() {
    print_check "Available memory"
    REQUIRED_GB=8

    if command -v free &> /dev/null; then
        AVAILABLE_GB=$(free -g | awk '/^Mem:/{print $2}')
        if [ "$AVAILABLE_GB" -ge "$REQUIRED_GB" ]; then
            print_pass
            print_info "Total: ${AVAILABLE_GB}GB (minimum: ${REQUIRED_GB}GB)"
        else
            print_warn "Low memory detected"
            print_info "Total: ${AVAILABLE_GB}GB, Recommended: ${REQUIRED_GB}GB"
        fi
    elif sysctl hw.memsize &> /dev/null; then
        # macOS
        AVAILABLE_GB=$(( $(sysctl -n hw.memsize) / 1024 / 1024 / 1024 ))
        if [ "$AVAILABLE_GB" -ge "$REQUIRED_GB" ]; then
            print_pass
            print_info "Total: ${AVAILABLE_GB}GB (minimum: ${REQUIRED_GB}GB)"
        else
            print_warn "Low memory detected"
            print_info "Total: ${AVAILABLE_GB}GB, Recommended: ${REQUIRED_GB}GB"
        fi
    else
        print_warn "Cannot determine available memory"
    fi
}

check_ports() {
    print_check "Required ports availability"
    PORTS=(3284 8000 5432 6379)
    PORTS_IN_USE=""

    for port in "${PORTS[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t &> /dev/null; then
            PORTS_IN_USE="$PORTS_IN_USE $port"
        fi
    done

    if [ -z "$PORTS_IN_USE" ]; then
        print_pass
        print_info "Ports 3284, 8000, 5432, 6379 are available"
    else
        print_warn "Some ports are already in use:$PORTS_IN_USE"
        print_info "You can change ports in .env or stop conflicting services"
    fi
}

check_env_file() {
    print_check ".env file"
    if [ -f ".env" ]; then
        print_pass
        print_info ".env file exists"
    else
        print_warn ".env file not found"
        if [ -f ".env.docker" ]; then
            print_info "Run: cp .env.docker .env"
        else
            print_fail "Neither .env nor .env.docker found"
            return 1
        fi
    fi
}

check_env_variables() {
    print_check "Required environment variables"

    if [ ! -f ".env" ]; then
        print_fail ".env file not found"
        return 1
    fi

    source .env 2>/dev/null || true

    MISSING_VARS=""
    REQUIRED_VARS=("SUPABASE_URL" "SUPABASE_ANON_KEY" "SUPABASE_SERVICE_ROLE_KEY" "ANTHROPIC_API_KEY")

    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ] || [ "${!var}" == "your-"* ]; then
            MISSING_VARS="$MISSING_VARS $var"
        fi
    done

    if [ -z "$MISSING_VARS" ]; then
        print_pass
        print_info "All required variables are set"
    else
        print_fail "Missing or invalid variables:$MISSING_VARS"
        print_info "Edit .env and set these variables"
        return 1
    fi
}

check_compose_file() {
    print_check "docker-compose.multitenant.yml"
    if [ -f "docker-compose.multitenant.yml" ]; then
        print_pass

        # Validate syntax
        print_check "docker-compose.yml syntax"
        if docker-compose -f docker-compose.multitenant.yml config &> /dev/null; then
            print_pass
        else
            print_fail "Invalid docker-compose.yml syntax"
            return 1
        fi
    else
        print_fail "docker-compose.multitenant.yml not found"
        return 1
    fi
}

check_dockerfile() {
    print_check "Dockerfile.multitenant"
    if [ -f "Dockerfile.multitenant" ]; then
        print_pass
    else
        print_fail "Dockerfile.multitenant not found"
        return 1
    fi
}

check_directories() {
    print_check "Required directories"

    MISSING_DIRS=""
    REQUIRED_DIRS=("database" "lib/mcp")

    for dir in "${REQUIRED_DIRS[@]}"; do
        if [ ! -d "$dir" ]; then
            MISSING_DIRS="$MISSING_DIRS $dir"
        fi
    done

    if [ -z "$MISSING_DIRS" ]; then
        print_pass
    else
        print_fail "Missing directories:$MISSING_DIRS"
        return 1
    fi

    # Check data directories (will be created if missing)
    print_check "Data directories"
    DATA_DIRS=("data/workspaces" "data/postgres" "data/redis")
    MISSING_DATA=""

    for dir in "${DATA_DIRS[@]}"; do
        if [ ! -d "$dir" ]; then
            MISSING_DATA="$MISSING_DATA $dir"
        fi
    done

    if [ -z "$MISSING_DATA" ]; then
        print_pass
    else
        print_warn "Data directories will be created:$MISSING_DATA"
        print_info "Run: mkdir -p data/workspaces data/postgres data/redis"
    fi
}

check_files() {
    print_check "Required files"

    MISSING_FILES=""
    REQUIRED_FILES=("requirements.txt" "database/schema.sql")

    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$file" ]; then
            MISSING_FILES="$MISSING_FILES $file"
        fi
    done

    if [ -z "$MISSING_FILES" ]; then
        print_pass
    else
        print_warn "Missing optional files:$MISSING_FILES"
    fi
}

check_network_connectivity() {
    print_check "Internet connectivity"
    if ping -c 1 google.com &> /dev/null || ping -c 1 8.8.8.8 &> /dev/null; then
        print_pass
        print_info "Internet connection is available"
    else
        print_warn "Cannot verify internet connection"
        print_info "Internet is required to pull Docker images"
    fi
}

# Main validation
main() {
    print_header "AgentAPI Docker Setup Validation"

    echo -e "${BLUE}Validating Docker environment...${NC}"
    echo ""

    # System checks
    print_header "System Requirements"
    check_docker || true
    check_docker_compose || true
    check_docker_running || true
    check_disk_space || true
    check_memory || true
    check_ports || true
    check_network_connectivity || true

    # File checks
    print_header "Configuration Files"
    check_env_file || true
    check_env_variables || true
    check_compose_file || true
    check_dockerfile || true
    check_directories || true
    check_files || true

    # Summary
    print_header "Validation Summary"
    echo -e "${GREEN}Passed:   $PASSED${NC}"
    echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
    echo -e "${RED}Errors:   $ERRORS${NC}"
    echo ""

    if [ $ERRORS -eq 0 ]; then
        echo -e "${GREEN}✓ Your environment is ready for Docker deployment!${NC}"
        echo ""
        echo -e "${BLUE}Next steps:${NC}"
        echo "  1. Review .env file and set any optional variables"
        echo "  2. Run: make -f Makefile.docker start"
        echo "  3. Or:  ./docker-manage.sh start"
        echo "  4. Or:  docker-compose -f docker-compose.multitenant.yml up -d"
        echo ""
        exit 0
    elif [ $ERRORS -le 2 ] && [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠ Your environment has minor issues but may work${NC}"
        echo -e "${YELLOW}Please review the warnings above${NC}"
        echo ""
        exit 1
    else
        echo -e "${RED}✗ Your environment has critical issues${NC}"
        echo -e "${RED}Please fix the errors above before proceeding${NC}"
        echo ""
        exit 2
    fi
}

# Run validation
main

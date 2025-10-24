#!/bin/bash

# ==============================================================================
# Docker Compose Management Script for AgentAPI Multi-Tenant
# ==============================================================================
# This script provides convenient commands for managing the Docker Compose setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.multitenant.yml"
ENV_FILE=".env"
ENV_EXAMPLE=".env.docker"

# Helper functions
print_info() {
    echo -e "${BLUE}ℹ ${1}${NC}"
}

print_success() {
    echo -e "${GREEN}✓ ${1}${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ ${1}${NC}"
}

print_error() {
    echo -e "${RED}✗ ${1}${NC}"
}

print_header() {
    echo ""
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}${1}${NC}"
    echo -e "${BLUE}================================${NC}"
    echo ""
}

# Check if .env file exists
check_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        print_warning ".env file not found!"
        if [ -f "$ENV_EXAMPLE" ]; then
            print_info "Copying $ENV_EXAMPLE to $ENV_FILE"
            cp "$ENV_EXAMPLE" "$ENV_FILE"
            print_warning "Please edit $ENV_FILE with your actual values before proceeding"
            exit 1
        else
            print_error "Neither $ENV_FILE nor $ENV_EXAMPLE found!"
            exit 1
        fi
    fi
}

# Create required directories
create_directories() {
    print_info "Creating required directories..."
    mkdir -p data/workspaces data/postgres data/redis logs/nginx
    print_success "Directories created"
}

# Build services
build() {
    print_header "Building Services"
    check_env_file
    docker-compose -f "$COMPOSE_FILE" build "$@"
    print_success "Build complete"
}

# Start services
start() {
    print_header "Starting Services"
    check_env_file
    create_directories
    docker-compose -f "$COMPOSE_FILE" up -d "$@"
    print_success "Services started"

    echo ""
    print_info "Waiting for services to be healthy..."
    sleep 5
    docker-compose -f "$COMPOSE_FILE" ps
}

# Stop services
stop() {
    print_header "Stopping Services"
    docker-compose -f "$COMPOSE_FILE" stop "$@"
    print_success "Services stopped"
}

# Restart services
restart() {
    print_header "Restarting Services"
    docker-compose -f "$COMPOSE_FILE" restart "$@"
    print_success "Services restarted"
}

# View logs
logs() {
    print_header "Service Logs"
    docker-compose -f "$COMPOSE_FILE" logs -f "$@"
}

# Show service status
status() {
    print_header "Service Status"
    docker-compose -f "$COMPOSE_FILE" ps
    echo ""

    print_info "Health Checks:"
    echo "AgentAPI: http://localhost:3284/status"
    echo "FastMCP: http://localhost:8000/health"
    echo ""

    # Test endpoints
    if command -v curl &> /dev/null; then
        print_info "Testing endpoints..."

        if curl -s http://localhost:3284/status > /dev/null 2>&1; then
            print_success "AgentAPI is responding"
        else
            print_warning "AgentAPI is not responding"
        fi

        if curl -s http://localhost:8000/health > /dev/null 2>&1; then
            print_success "FastMCP is responding"
        else
            print_warning "FastMCP is not responding"
        fi
    fi
}

# Clean up everything
clean() {
    print_header "Cleaning Up"
    print_warning "This will remove all containers and networks"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker-compose -f "$COMPOSE_FILE" down
        print_success "Cleanup complete"
    else
        print_info "Cleanup cancelled"
    fi
}

# Clean up including volumes
clean_all() {
    print_header "Deep Clean"
    print_error "WARNING: This will remove all containers, networks, AND DATA VOLUMES!"
    print_warning "All database data and user workspaces will be deleted!"
    read -p "Are you ABSOLUTELY sure? (type 'yes' to confirm) " -r
    echo
    if [[ $REPLY == "yes" ]]; then
        docker-compose -f "$COMPOSE_FILE" down -v
        rm -rf data/workspaces data/postgres data/redis logs/nginx
        print_success "Deep clean complete"
    else
        print_info "Deep clean cancelled"
    fi
}

# Backup data
backup() {
    print_header "Creating Backup"

    BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"

    print_info "Backing up workspace data..."
    tar -czf "$BACKUP_DIR/workspaces.tar.gz" data/workspaces/ 2>/dev/null || true

    print_info "Backing up PostgreSQL database..."
    docker-compose -f "$COMPOSE_FILE" exec -T postgres \
        pg_dump -U agentapi agentapi | gzip > "$BACKUP_DIR/postgres.sql.gz" 2>/dev/null || true

    print_info "Backing up Redis data..."
    cp data/redis/dump.rdb "$BACKUP_DIR/redis.rdb" 2>/dev/null || true

    print_success "Backup created in $BACKUP_DIR"
}

# Execute command in service
exec_service() {
    SERVICE=$1
    shift
    docker-compose -f "$COMPOSE_FILE" exec "$SERVICE" "$@"
}

# Database shell
db_shell() {
    print_header "PostgreSQL Shell"
    exec_service postgres psql -U agentapi -d agentapi
}

# Redis shell
redis_shell() {
    print_header "Redis Shell"
    exec_service redis redis-cli
}

# AgentAPI shell
agentapi_shell() {
    print_header "AgentAPI Shell"
    exec_service agentapi sh
}

# Update services
update() {
    print_header "Updating Services"
    print_info "Pulling latest images..."
    docker-compose -f "$COMPOSE_FILE" pull

    print_info "Rebuilding services..."
    docker-compose -f "$COMPOSE_FILE" build --pull

    print_info "Restarting services..."
    docker-compose -f "$COMPOSE_FILE" up -d

    print_success "Update complete"
}

# Show resource usage
stats() {
    print_header "Resource Usage"
    docker stats --no-stream $(docker-compose -f "$COMPOSE_FILE" ps -q)
}

# Show help
show_help() {
    cat << EOF
AgentAPI Docker Compose Management Script

Usage: $0 [command] [options]

Commands:
  build           Build all services
  start           Start all services
  stop            Stop all services
  restart         Restart all services
  logs [service]  View logs (optionally for specific service)
  status          Show service status and health
  clean           Stop and remove containers/networks
  clean-all       Stop and remove everything including volumes (DESTRUCTIVE!)
  backup          Create backup of all data
  update          Update and restart all services
  stats           Show resource usage statistics

Service Shells:
  db              Open PostgreSQL shell
  redis           Open Redis CLI
  shell           Open AgentAPI shell
  exec <service> <command>  Execute command in service

Examples:
  $0 start                 # Start all services
  $0 logs agentapi        # View AgentAPI logs
  $0 restart postgres     # Restart only PostgreSQL
  $0 backup               # Create backup
  $0 db                   # Open PostgreSQL shell
  $0 exec agentapi ls -la # List files in AgentAPI container

Environment:
  - Edit .env file to configure services
  - See .env.docker for all available options
  - See DOCKER_COMPOSE_README.md for detailed documentation

EOF
}

# Main script
main() {
    case "${1:-}" in
        build)
            shift
            build "$@"
            ;;
        start|up)
            shift
            start "$@"
            ;;
        stop|down)
            shift
            stop "$@"
            ;;
        restart)
            shift
            restart "$@"
            ;;
        logs)
            shift
            logs "$@"
            ;;
        status|ps)
            status
            ;;
        clean)
            clean
            ;;
        clean-all|reset)
            clean_all
            ;;
        backup)
            backup
            ;;
        update)
            update
            ;;
        stats)
            stats
            ;;
        db|psql)
            db_shell
            ;;
        redis)
            redis_shell
            ;;
        shell|sh)
            agentapi_shell
            ;;
        exec)
            shift
            exec_service "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: ${1:-}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"

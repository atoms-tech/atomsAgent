#!/bin/bash

################################################################################
# K6 Load Test Runner Script
#
# This script provides a convenient way to run K6 load tests with various
# configurations and options.
#
# Usage:
#   ./run-tests.sh [options]
#
# Options:
#   -e, --env ENV           Environment (local, staging, production)
#   -s, --scenario NAME     Run specific scenario only
#   -d, --docker            Run in Docker container
#   -m, --monitor           Start monitoring stack (InfluxDB + Grafana)
#   -r, --report            Generate HTML report
#   -v, --verbose           Verbose output
#   -h, --help              Show this help message
#
# Examples:
#   ./run-tests.sh                          # Run all tests locally
#   ./run-tests.sh --env staging            # Run against staging
#   ./run-tests.sh --scenario authentication # Run auth scenario only
#   ./run-tests.sh --docker --monitor       # Run in Docker with monitoring
################################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
ENVIRONMENT="local"
SCENARIO=""
USE_DOCKER=false
START_MONITORING=false
GENERATE_REPORT=true
VERBOSE=false

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "${SCRIPT_DIR}/../.." && pwd )"

################################################################################
# Functions
################################################################################

print_usage() {
    cat << EOF
K6 Load Test Runner for AgentAPI

Usage: $0 [options]

Options:
    -e, --env ENV           Environment (local, staging, production) [default: local]
    -s, --scenario NAME     Run specific scenario only
    -d, --docker            Run in Docker container
    -m, --monitor           Start monitoring stack (InfluxDB + Grafana)
    -r, --report            Generate HTML report [default: true]
    -v, --verbose           Verbose output
    -h, --help              Show this help message

Available Scenarios:
    - authentication
    - mcp_connection
    - tool_execution
    - list_tools
    - disconnect
    - mixed_workload

Examples:
    $0                                     # Run all tests locally
    $0 --env staging                       # Run against staging
    $0 --scenario authentication           # Run auth scenario only
    $0 --docker --monitor                  # Run in Docker with monitoring
    $0 --env production --verbose          # Run in production with verbose output

EOF
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."

    if [ "$USE_DOCKER" = false ]; then
        if ! command -v k6 &> /dev/null; then
            log_error "K6 is not installed. Please install it first:"
            echo "  macOS: brew install k6"
            echo "  Linux: See https://k6.io/docs/getting-started/installation"
            echo "  Or use --docker flag to run in container"
            exit 1
        fi
        log_success "K6 found: $(k6 version)"
    else
        if ! command -v docker &> /dev/null; then
            log_error "Docker is not installed. Please install Docker first."
            exit 1
        fi
        log_success "Docker found: $(docker --version)"

        if [ "$START_MONITORING" = true ]; then
            if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
                log_error "Docker Compose is not installed. Please install it first."
                exit 1
            fi
            log_success "Docker Compose found"
        fi
    fi
}

load_env_config() {
    local env_file="${SCRIPT_DIR}/.env.${ENVIRONMENT}"

    if [ -f "$env_file" ]; then
        log_info "Loading environment from ${env_file}"
        set -a
        source "$env_file"
        set +a
    else
        log_warning "Environment file ${env_file} not found, using defaults"
    fi

    # Set defaults if not defined
    export BASE_URL=${BASE_URL:-"http://localhost:3284"}
    export OAUTH_BASE_URL=${OAUTH_BASE_URL:-"http://localhost:3000/api/mcp/oauth"}
}

start_monitoring_stack() {
    log_info "Starting monitoring stack (InfluxDB + Grafana)..."

    cd "${SCRIPT_DIR}"

    if command -v docker compose &> /dev/null; then
        docker compose up -d influxdb grafana
    else
        docker-compose up -d influxdb grafana
    fi

    log_success "Monitoring stack started"
    log_info "Grafana dashboard: http://localhost:3001"
    log_info "Waiting for services to be ready..."
    sleep 5
}

stop_monitoring_stack() {
    log_info "Stopping monitoring stack..."

    cd "${SCRIPT_DIR}"

    if command -v docker compose &> /dev/null; then
        docker compose down
    else
        docker-compose down
    fi

    log_success "Monitoring stack stopped"
}

run_k6_local() {
    log_info "Running K6 tests locally..."

    local k6_args=""

    if [ -n "$SCENARIO" ]; then
        k6_args="$k6_args --scenario $SCENARIO"
        log_info "Running scenario: $SCENARIO"
    fi

    if [ "$VERBOSE" = true ]; then
        k6_args="$k6_args --verbose"
    fi

    if [ "$START_MONITORING" = true ]; then
        k6_args="$k6_args --out influxdb=http://localhost:8086/k6"
    fi

    cd "${SCRIPT_DIR}"

    log_info "Test configuration:"
    log_info "  BASE_URL: $BASE_URL"
    log_info "  OAUTH_BASE_URL: $OAUTH_BASE_URL"
    echo ""

    k6 run $k6_args k6_tests.js

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        log_success "Tests completed successfully!"
    else
        log_error "Tests failed with exit code: $exit_code"
    fi

    if [ "$GENERATE_REPORT" = true ] && [ -f "summary.html" ]; then
        log_success "HTML report generated: ${SCRIPT_DIR}/summary.html"
    fi

    return $exit_code
}

run_k6_docker() {
    log_info "Running K6 tests in Docker..."

    local docker_args="-e BASE_URL=${BASE_URL} -e OAUTH_BASE_URL=${OAUTH_BASE_URL}"

    if [ "$START_MONITORING" = true ]; then
        docker_args="$docker_args -e K6_OUT=influxdb=http://influxdb:8086/k6"
    fi

    local k6_args=""

    if [ -n "$SCENARIO" ]; then
        k6_args="$k6_args --scenario $SCENARIO"
        log_info "Running scenario: $SCENARIO"
    fi

    if [ "$VERBOSE" = true ]; then
        k6_args="$k6_args --verbose"
    fi

    cd "${SCRIPT_DIR}"

    log_info "Test configuration:"
    log_info "  BASE_URL: $BASE_URL"
    log_info "  OAUTH_BASE_URL: $OAUTH_BASE_URL"
    echo ""

    if command -v docker compose &> /dev/null; then
        docker compose run --rm $docker_args k6 k6 run $k6_args /scripts/k6_tests.js
    else
        docker-compose run --rm $docker_args k6 k6 run $k6_args /scripts/k6_tests.js
    fi

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        log_success "Tests completed successfully!"
    else
        log_error "Tests failed with exit code: $exit_code"
    fi

    if [ "$GENERATE_REPORT" = true ] && [ -f "summary.html" ]; then
        log_success "HTML report generated: ${SCRIPT_DIR}/summary.html"
    fi

    return $exit_code
}

cleanup() {
    if [ "$START_MONITORING" = true ]; then
        echo ""
        read -p "Stop monitoring stack? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            stop_monitoring_stack
        else
            log_info "Monitoring stack left running"
            log_info "To stop manually: cd ${SCRIPT_DIR} && docker-compose down"
        fi
    fi
}

################################################################################
# Main
################################################################################

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--env)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -s|--scenario)
                SCENARIO="$2"
                shift 2
                ;;
            -d|--docker)
                USE_DOCKER=true
                shift
                ;;
            -m|--monitor)
                START_MONITORING=true
                shift
                ;;
            -r|--report)
                GENERATE_REPORT=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    # Print banner
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║           K6 Load Test Runner for AgentAPI                ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""

    # Check dependencies
    check_dependencies

    # Load environment configuration
    load_env_config

    # Start monitoring if requested
    if [ "$START_MONITORING" = true ]; then
        start_monitoring_stack
    fi

    # Set trap for cleanup
    trap cleanup EXIT

    # Run tests
    if [ "$USE_DOCKER" = true ]; then
        run_k6_docker
    else
        run_k6_local
    fi

    exit_code=$?

    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                    Test Run Complete                      ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""

    exit $exit_code
}

# Run main function
main "$@"

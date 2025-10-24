#!/bin/bash

# AgentAPI Chat Completions Load Test Runner
# This script provides convenient execution of the K6 chat API load tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
BASE_URL="${BASE_URL:-http://localhost:3284}"
AUTH_TOKEN="${AUTH_TOKEN:-Bearer test-token-12345}"
TEST_FILE="tests/load/k6_chat_api_test.js"

# Helper functions
print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check if k6 is installed
check_k6_installed() {
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed!"
        echo ""
        echo "Install k6:"
        echo "  macOS:   brew install k6"
        echo "  Linux:   See https://k6.io/docs/getting-started/installation/"
        echo "  Windows: choco install k6"
        exit 1
    fi
    print_success "k6 is installed ($(k6 version))"
}

# Check if API is accessible
check_api_health() {
    print_info "Checking API health at ${BASE_URL}..."

    if curl -f -s "${BASE_URL}/status" > /dev/null 2>&1; then
        print_success "API is accessible at ${BASE_URL}"
        return 0
    else
        print_error "Cannot reach API at ${BASE_URL}"
        echo ""
        echo "Make sure the AgentAPI server is running:"
        echo "  agentapi server -- claude"
        exit 1
    fi
}

# Load environment file if it exists
load_env_file() {
    if [ -f "tests/load/.env.chat-test" ]; then
        print_info "Loading configuration from .env.chat-test"
        source tests/load/.env.chat-test
        print_success "Configuration loaded"
    else
        print_warning "No .env.chat-test file found, using defaults"
        print_info "Create one from: tests/load/.env.chat-test.example"
    fi
}

# Display test configuration
show_config() {
    print_header "Test Configuration"
    echo "Base URL:    ${BASE_URL}"
    echo "Auth Token:  ${AUTH_TOKEN:0:20}..." # Show only first 20 chars
    echo "Test File:   ${TEST_FILE}"
    echo ""
}

# Run the test
run_test() {
    local scenario="$1"

    print_header "Running K6 Load Test"

    if [ -n "$scenario" ]; then
        print_info "Running scenario: ${scenario}"
        k6 run \
            --env BASE_URL="${BASE_URL}" \
            --env AUTH_TOKEN="${AUTH_TOKEN}" \
            --include-scenario-in-stats "${scenario}" \
            "${TEST_FILE}"
    else
        print_info "Running all scenarios"
        k6 run \
            --env BASE_URL="${BASE_URL}" \
            --env AUTH_TOKEN="${AUTH_TOKEN}" \
            "${TEST_FILE}"
    fi
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run K6 load tests for AgentAPI chat completions endpoint.

OPTIONS:
    -h, --help              Show this help message
    -u, --url URL           Set base URL (default: http://localhost:3284)
    -t, --token TOKEN       Set auth token
    -s, --scenario NAME     Run specific scenario only
    -c, --check-only        Only check prerequisites, don't run tests
    -l, --list-scenarios    List available test scenarios

SCENARIOS:
    basic_load          Basic load test (10 req/s for 2 min)
    ramp_up             Gradual ramp up (0-100 req/s over 5 min)
    stress_test         Stress test (200 req/s spike for 1 min)
    sustained_load      Sustained load (50 req/s for 10 min)
    streaming_test      Streaming-specific test (1-50 concurrent streams)

EXAMPLES:
    # Run all scenarios with defaults
    $0

    # Run specific scenario
    $0 --scenario basic_load

    # Use custom URL and token
    $0 --url http://api.example.com --token "Bearer xyz123"

    # Check prerequisites only
    $0 --check-only

ENVIRONMENT:
    You can also set environment variables:
    export BASE_URL=http://localhost:3284
    export AUTH_TOKEN="Bearer your-token"
    $0

    Or create a .env.chat-test file (see .env.chat-test.example)

EOF
}

# List available scenarios
list_scenarios() {
    print_header "Available Test Scenarios"
    cat << EOF

1. basic_load
   - Duration: 2 minutes
   - Load: 10 requests/second (constant)
   - Mix: 70% non-streaming, 30% streaming
   - Purpose: Baseline performance validation

2. ramp_up
   - Duration: 5 minutes
   - Load: 0 → 20 → 50 → 100 req/s (gradual increase)
   - Mix: 50% non-streaming, 50% streaming
   - Purpose: Test system under increasing load

3. stress_test
   - Duration: 1 minute
   - Load: 200 requests/second (spike)
   - Mix: 80% non-streaming, 20% streaming
   - Purpose: Identify breaking points

4. sustained_load
   - Duration: 10 minutes
   - Load: 50 requests/second (constant)
   - Mix: 60% non-streaming, 40% streaming
   - Purpose: Endurance and stability testing

5. streaming_test
   - Duration: 9 minutes
   - VUs: 1 → 10 → 30 → 50 concurrent streams
   - Focus: Streaming performance and SSE validation
   - Purpose: Deep streaming analysis

Total test duration (all scenarios): ~28 minutes

EOF
}

# Parse command line arguments
SCENARIO=""
CHECK_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -u|--url)
            BASE_URL="$2"
            shift 2
            ;;
        -t|--token)
            AUTH_TOKEN="$2"
            shift 2
            ;;
        -s|--scenario)
            SCENARIO="$2"
            shift 2
            ;;
        -c|--check-only)
            CHECK_ONLY=true
            shift
            ;;
        -l|--list-scenarios)
            list_scenarios
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo ""
            usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_header "AgentAPI Chat Completions Load Test"
    echo ""

    # Load environment file
    load_env_file

    # Check prerequisites
    check_k6_installed
    check_api_health

    # Show configuration
    show_config

    if [ "$CHECK_ONLY" = true ]; then
        print_success "All prerequisites met!"
        exit 0
    fi

    # Run the test
    run_test "$SCENARIO"

    # Check for output files
    if [ -f "tests/load/chat_api_summary.html" ]; then
        print_success "Test completed!"
        echo ""
        print_info "View results:"
        echo "  HTML Report: tests/load/chat_api_summary.html"
        echo "  JSON Report: tests/load/chat_api_summary.json"
        echo ""

        # Try to open HTML report (macOS)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            read -p "Open HTML report in browser? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                open tests/load/chat_api_summary.html
            fi
        fi
    fi
}

# Run main function
main

#!/bin/bash

# OAuth Integration Test Runner
# This script runs the OAuth integration tests with proper setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}OAuth Integration Test Runner${NC}"
echo "========================================"

# Check if Redis is available
echo -n "Checking Redis availability... "
if command -v redis-cli &> /dev/null && redis-cli -u "${REDIS_URL:-redis://localhost:6379}" ping &> /dev/null; then
    echo -e "${GREEN}✓ Redis available${NC}"
    REDIS_AVAILABLE=true
else
    echo -e "${YELLOW}⚠ Redis not available (tests will use in-memory fallback)${NC}"
    REDIS_AVAILABLE=false
fi

# Set test environment
export REDIS_URL="${REDIS_URL:-redis://localhost:6379/15}"
export GO_TEST_TIMEOUT="${GO_TEST_TIMEOUT:-10m}"

# Parse arguments
VERBOSE=""
RACE=""
COVER=""
RUN_PATTERN=""
SHORT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -race|--race)
            RACE="-race"
            shift
            ;;
        -cover|--cover)
            COVER="-cover -coverprofile=coverage.out"
            shift
            ;;
        -run)
            RUN_PATTERN="-run $2"
            shift 2
            ;;
        -short|--short)
            SHORT="-short"
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose     Verbose test output"
            echo "  -race, --race     Enable race detection"
            echo "  -cover, --cover   Generate coverage report"
            echo "  -run <pattern>    Run specific tests matching pattern"
            echo "  -short, --short   Run tests in short mode"
            echo "  -h, --help        Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 -v                    # Run all tests with verbose output"
            echo "  $0 -v -race              # Run with race detection"
            echo "  $0 -run TestOAuthInit    # Run only OAuth initiation tests"
            echo "  $0 -cover                # Run with coverage report"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Navigate to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

echo ""
echo "Test Configuration:"
echo "  Redis URL: $REDIS_URL"
echo "  Timeout: $GO_TEST_TIMEOUT"
echo "  Verbose: ${VERBOSE:-no}"
echo "  Race Detection: ${RACE:-no}"
echo "  Coverage: ${COVER:-no}"
echo "  Run Pattern: ${RUN_PATTERN:-all tests}"
echo ""

# Run tests
echo -e "${GREEN}Running integration tests...${NC}"
echo "========================================"

# Build test command
TEST_CMD="go test ./tests/integration $VERBOSE $RACE $COVER $RUN_PATTERN $SHORT -timeout $GO_TEST_TIMEOUT"

echo "Command: $TEST_CMD"
echo ""

# Execute tests
if eval $TEST_CMD; then
    echo ""
    echo -e "${GREEN}✓ All tests passed!${NC}"

    # Show coverage if generated
    if [[ -n "$COVER" ]] && [[ -f "coverage.out" ]]; then
        echo ""
        echo -e "${GREEN}Coverage Report:${NC}"
        go tool cover -func=coverage.out | tail -1

        # Generate HTML coverage report
        echo ""
        echo "Generating HTML coverage report..."
        go tool cover -html=coverage.out -o coverage.html
        echo -e "${GREEN}✓ Coverage report saved to coverage.html${NC}"
    fi

    exit 0
else
    echo ""
    echo -e "${RED}✗ Tests failed${NC}"
    exit 1
fi

#!/bin/bash
# Build script for multi-tenant AgentAPI Docker image
# Provides convenient build options and validation

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
IMAGE_NAME="${IMAGE_NAME:-agentapi-multitenant}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
DOCKERFILE="${DOCKERFILE:-Dockerfile.multitenant}"
REGISTRY="${REGISTRY:-}"
PLATFORM="${PLATFORM:-linux/amd64}"
BUILD_ARGS="${BUILD_ARGS:-}"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

# Parse arguments
PUSH=false
NO_CACHE=false
SQUASH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --push)
            PUSH=true
            shift
            ;;
        --no-cache)
            NO_CACHE=true
            shift
            ;;
        --squash)
            SQUASH=true
            shift
            ;;
        --tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --push              Push image to registry after build"
            echo "  --no-cache          Build without using cache"
            echo "  --squash            Squash layers in final image (requires experimental)"
            echo "  --tag TAG           Image tag (default: latest)"
            echo "  --registry URL      Registry URL (e.g., docker.io/username)"
            echo "  --platform PLATFORM Target platform (default: linux/amd64)"
            echo "  --help              Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  IMAGE_NAME          Image name (default: agentapi-multitenant)"
            echo "  IMAGE_TAG           Image tag (default: latest)"
            echo "  DOCKERFILE          Dockerfile path (default: Dockerfile.multitenant)"
            echo "  REGISTRY            Registry URL"
            echo "  PLATFORM            Target platform"
            echo ""
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Construct full image name
if [ -n "$REGISTRY" ]; then
    FULL_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
else
    FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
fi

# Display build configuration
print_header "Multi-Tenant AgentAPI Docker Build"
echo ""
print_info "Build Configuration:"
echo "  Image Name:    $FULL_IMAGE_NAME"
echo "  Dockerfile:    $DOCKERFILE"
echo "  Platform:      $PLATFORM"
echo "  Push:          $PUSH"
echo "  No Cache:      $NO_CACHE"
echo "  Squash:        $SQUASH"
echo ""

# Verify prerequisites
print_info "Verifying prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi
print_success "Docker is installed"

if [ ! -f "$DOCKERFILE" ]; then
    print_error "Dockerfile not found: $DOCKERFILE"
    exit 1
fi
print_success "Dockerfile found"

if [ ! -f "go.mod" ]; then
    print_warning "go.mod not found in current directory"
fi

if [ ! -f "requirements.txt" ]; then
    print_warning "requirements.txt not found"
fi

if [ ! -d "chat" ]; then
    print_warning "chat directory not found"
fi

# Build Docker image
print_header "Building Docker Image"

BUILD_CMD="docker build"

# Add build flags
BUILD_CMD="$BUILD_CMD --platform $PLATFORM"
BUILD_CMD="$BUILD_CMD -t $FULL_IMAGE_NAME"
BUILD_CMD="$BUILD_CMD -f $DOCKERFILE"

if [ "$NO_CACHE" = true ]; then
    BUILD_CMD="$BUILD_CMD --no-cache"
fi

if [ "$SQUASH" = true ]; then
    BUILD_CMD="$BUILD_CMD --squash"
fi

# Add build arguments
if [ -n "$BUILD_ARGS" ]; then
    BUILD_CMD="$BUILD_CMD $BUILD_ARGS"
fi

# Add build context
BUILD_CMD="$BUILD_CMD ."

print_info "Running: $BUILD_CMD"
echo ""

# Execute build
if eval "$BUILD_CMD"; then
    print_success "Docker image built successfully"
else
    print_error "Docker build failed"
    exit 1
fi

# Get image size
IMAGE_SIZE=$(docker images "$FULL_IMAGE_NAME" --format "{{.Size}}" | head -1)
print_success "Image size: $IMAGE_SIZE"

# Tag additional versions if building latest
if [ "$IMAGE_TAG" = "latest" ] && [ -d ".git" ]; then
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")

    if [ -n "$GIT_COMMIT" ] && [ "$GIT_COMMIT" != "unknown" ]; then
        COMMIT_IMAGE_NAME="${IMAGE_NAME}:${GIT_COMMIT}"
        if [ -n "$REGISTRY" ]; then
            COMMIT_IMAGE_NAME="${REGISTRY}/${COMMIT_IMAGE_NAME}"
        fi
        docker tag "$FULL_IMAGE_NAME" "$COMMIT_IMAGE_NAME"
        print_success "Tagged as: $COMMIT_IMAGE_NAME"
    fi

    if [ -n "$GIT_TAG" ]; then
        TAG_IMAGE_NAME="${IMAGE_NAME}:${GIT_TAG}"
        if [ -n "$REGISTRY" ]; then
            TAG_IMAGE_NAME="${REGISTRY}/${TAG_IMAGE_NAME}"
        fi
        docker tag "$FULL_IMAGE_NAME" "$TAG_IMAGE_NAME"
        print_success "Tagged as: $TAG_IMAGE_NAME"
    fi
fi

# Run security scan (if trivy is available)
if command -v trivy &> /dev/null; then
    print_header "Running Security Scan"
    print_info "Scanning image for vulnerabilities..."
    trivy image --severity HIGH,CRITICAL "$FULL_IMAGE_NAME" || true
else
    print_warning "Trivy not installed - skipping security scan"
    print_info "Install trivy for security scanning: https://github.com/aquasecurity/trivy"
fi

# Test image
print_header "Testing Image"
print_info "Starting test container..."

TEST_CONTAINER_NAME="agentapi-test-$$"
if docker run --rm -d --name "$TEST_CONTAINER_NAME" \
    -e AGENTAPI_PORT=3284 \
    -e FASTMCP_PORT=8000 \
    "$FULL_IMAGE_NAME" > /dev/null 2>&1; then

    print_success "Container started"

    # Wait for services to start
    sleep 5

    # Check health
    if docker exec "$TEST_CONTAINER_NAME" curl -f http://localhost:3284/health > /dev/null 2>&1; then
        print_success "AgentAPI health check passed"
    else
        print_warning "AgentAPI health check failed"
    fi

    if docker exec "$TEST_CONTAINER_NAME" curl -f http://localhost:8000/health > /dev/null 2>&1; then
        print_success "FastMCP health check passed"
    else
        print_warning "FastMCP health check failed"
    fi

    # Stop test container
    docker stop "$TEST_CONTAINER_NAME" > /dev/null 2>&1 || true
    print_success "Test complete"
else
    print_warning "Could not start test container"
fi

# Push to registry
if [ "$PUSH" = true ]; then
    if [ -z "$REGISTRY" ]; then
        print_error "Cannot push: REGISTRY not specified"
        exit 1
    fi

    print_header "Pushing to Registry"
    print_info "Pushing $FULL_IMAGE_NAME..."

    if docker push "$FULL_IMAGE_NAME"; then
        print_success "Image pushed successfully"

        # Push additional tags
        if [ "$IMAGE_TAG" = "latest" ] && [ -d ".git" ]; then
            GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "")
            if [ -n "$GIT_COMMIT" ]; then
                COMMIT_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${GIT_COMMIT}"
                docker push "$COMMIT_IMAGE_NAME" && print_success "Pushed: $COMMIT_IMAGE_NAME"
            fi

            GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
            if [ -n "$GIT_TAG" ]; then
                TAG_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${GIT_TAG}"
                docker push "$TAG_IMAGE_NAME" && print_success "Pushed: $TAG_IMAGE_NAME"
            fi
        fi
    else
        print_error "Failed to push image"
        exit 1
    fi
fi

# Summary
print_header "Build Complete"
echo ""
print_success "Successfully built: $FULL_IMAGE_NAME"
echo ""
print_info "To run the image:"
echo "  docker run -p 3284:3284 -p 8000:8000 $FULL_IMAGE_NAME"
echo ""
print_info "To run with docker-compose:"
echo "  docker-compose -f docker-compose.multitenant.yml up"
echo ""

exit 0

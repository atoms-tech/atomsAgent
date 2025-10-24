# Multi-Tenant AgentAPI Docker Configuration

Production-ready Docker configuration for multi-tenant AgentAPI with FastMCP 2.0 support.

## Overview

This Docker configuration provides a complete multi-stage build process that:

- Builds the Go AgentAPI server with optimized binaries
- Builds the Next.js chat UI
- Installs FastMCP 2.0 and Python dependencies
- Creates a minimal production image with security best practices
- Runs both AgentAPI and FastMCP services using supervisord

## Architecture

### Multi-Stage Build

The Dockerfile uses a 4-stage build process:

1. **Go Builder** (golang:1.21-alpine)
   - Builds the agentapi binary
   - Strips debug symbols with ldflags
   - Creates a static binary (CGO_ENABLED=0)
   - Optimizes for size and security

2. **Node.js Builder** (node:20-alpine)
   - Builds Next.js chat UI
   - Supports both npm and pnpm
   - Creates static export for production

3. **Python Dependencies** (python:3.11-slim)
   - Installs FastMCP 2.0 and dependencies
   - Isolates build dependencies
   - Reduces final image size

4. **Final Production Image** (python:3.11-slim)
   - Minimal base image
   - Non-root user (agentapi:1001)
   - Runtime dependencies only
   - Multi-service orchestration with supervisord

## Quick Start

### Build the Image

```bash
# Using the build script (recommended)
./build-multitenant.sh

# Or manually with docker
docker build -f Dockerfile.multitenant -t agentapi-multitenant:latest .
```

### Run the Container

```bash
# Run with default ports
docker run -p 3284:3284 -p 8000:8000 agentapi-multitenant:latest

# Run with custom environment
docker run -p 3284:3284 -p 8000:8000 \
  -e AGENTAPI_PORT=3284 \
  -e FASTMCP_PORT=8000 \
  -e FASTMCP_WORKERS=4 \
  -v ./workspaces:/workspaces \
  agentapi-multitenant:latest
```

### Using Docker Compose

```bash
docker-compose -f docker-compose.multitenant.yml up
```

## Build Script Usage

The `build-multitenant.sh` script provides convenient build options:

```bash
# Basic build
./build-multitenant.sh

# Build with specific tag
./build-multitenant.sh --tag v2.0.0

# Build and push to registry
./build-multitenant.sh --registry docker.io/username --push

# Build without cache
./build-multitenant.sh --no-cache

# Build for different platform
./build-multitenant.sh --platform linux/arm64

# Show help
./build-multitenant.sh --help
```

### Build Script Options

- `--push` - Push image to registry after build
- `--no-cache` - Build without using cache
- `--squash` - Squash layers in final image (experimental)
- `--tag TAG` - Specify image tag (default: latest)
- `--registry URL` - Specify registry URL
- `--platform PLATFORM` - Target platform (default: linux/amd64)

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTAPI_PORT` | `3284` | Port for AgentAPI server |
| `FASTMCP_PORT` | `8000` | Port for FastMCP service |
| `FASTMCP_WORKERS` | `4` | Number of FastMCP workers |
| `PYTHONUNBUFFERED` | `1` | Python output buffering |

### Exposed Ports

- `3284` - AgentAPI HTTP server
- `8000` - FastMCP service

### Volumes

- `/workspaces` - Workspace directory (permissions: 1777)
- `/var/log/agentapi` - Application logs
- `/app` - Application directory

## Architecture Details

### Service Management

The container uses **supervisord** to manage both services:

- **agentapi** - Go server handling API requests
- **fastmcp** - Python FastAPI service for MCP operations

Supervisord configuration:
- Auto-restart on failure
- Separate log files for each service
- Graceful shutdown handling
- Non-root execution

### Directory Structure

```
/
├── app/                          # Application working directory
├── workspaces/                   # Tenant workspaces (1777 permissions)
├── usr/
│   ├── local/
│   │   ├── bin/
│   │   │   ├── agentapi          # Go binary
│   │   │   ├── fastmcp_service.py # FastMCP service
│   │   │   ├── fastmcp_wrapper.py # FastMCP wrapper
│   │   │   └── fastmcp_start.sh  # Startup script
│   │   └── share/
│   │       └── agentapi/
│   │           └── chat/         # Next.js build output
└── var/
    └── log/
        └── agentapi/             # Service logs
            ├── agentapi.log
            ├── agentapi_error.log
            ├── fastmcp.log
            └── fastmcp_error.log
```

### Security Features

1. **Non-root Execution**
   - Services run as `agentapi` user (UID 1001)
   - Minimal privileges
   - Proper file ownership

2. **Minimal Base Image**
   - Based on python:3.11-slim
   - Only runtime dependencies
   - No build tools in final image

3. **Optimized Binaries**
   - Stripped debug symbols (-s -w)
   - Static linking (no external dependencies)
   - Reduced attack surface

4. **Health Checks**
   - HTTP health endpoint monitoring
   - Automatic restart on failure
   - 30-second interval checks

## Health Checks

The container includes health checks for both services:

```bash
# Check AgentAPI
curl -f http://localhost:3284/health

# Check FastMCP
curl -f http://localhost:8000/health
```

Docker health check configuration:
- Interval: 30 seconds
- Timeout: 10 seconds
- Start period: 10 seconds
- Retries: 3

## Logs

Access logs from running container:

```bash
# View all logs
docker logs -f <container_id>

# View AgentAPI logs
docker exec <container_id> tail -f /var/log/agentapi/agentapi.log

# View FastMCP logs
docker exec <container_id> tail -f /var/log/agentapi/fastmcp.log

# View supervisord logs
docker exec <container_id> tail -f /var/log/agentapi/supervisord.log
```

## Development

### Local Testing

```bash
# Build image
./build-multitenant.sh

# Run with volume mounts for development
docker run -p 3284:3284 -p 8000:8000 \
  -v $(pwd)/workspaces:/workspaces \
  -e AGENTAPI_PORT=3284 \
  -e FASTMCP_PORT=8000 \
  agentapi-multitenant:latest
```

### Debugging

```bash
# Enter running container
docker exec -it <container_id> /bin/bash

# Check service status
docker exec <container_id> supervisorctl status

# Restart a service
docker exec <container_id> supervisorctl restart agentapi
docker exec <container_id> supervisorctl restart fastmcp
```

## Optimization

### Image Size Optimization

The multi-stage build significantly reduces image size:

- Stage 1 (Go builder): ~500MB (discarded)
- Stage 2 (Node builder): ~300MB (discarded)
- Stage 3 (Python deps): ~400MB (only packages copied)
- Final image: ~250MB (runtime only)

### Build Cache

For faster builds, leverage Docker layer caching:

```bash
# Build with cache
docker build -f Dockerfile.multitenant -t agentapi-multitenant:latest .

# Clear cache and rebuild
docker build --no-cache -f Dockerfile.multitenant -t agentapi-multitenant:latest .
```

## Production Deployment

### Kubernetes

Example deployment configuration:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi-multitenant
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentapi-multitenant
  template:
    metadata:
      labels:
        app: agentapi-multitenant
    spec:
      containers:
      - name: agentapi
        image: agentapi-multitenant:latest
        ports:
        - containerPort: 3284
          name: agentapi
        - containerPort: 8000
          name: fastmcp
        env:
        - name: AGENTAPI_PORT
          value: "3284"
        - name: FASTMCP_PORT
          value: "8000"
        - name: FASTMCP_WORKERS
          value: "4"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3284
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 3284
          initialDelaySeconds: 5
          periodSeconds: 10
        volumeMounts:
        - name: workspaces
          mountPath: /workspaces
      volumes:
      - name: workspaces
        persistentVolumeClaim:
          claimName: agentapi-workspaces
```

### Docker Swarm

```bash
docker stack deploy -c docker-compose.multitenant.yml agentapi
```

### Resource Requirements

Recommended minimum resources:

- **CPU**: 1 core
- **Memory**: 512MB
- **Storage**: 10GB

Production recommendations:

- **CPU**: 2-4 cores
- **Memory**: 2-4GB
- **Storage**: 50GB+ (depending on workspace usage)

## Monitoring

### Prometheus Metrics

Both services expose Prometheus-compatible metrics:

- AgentAPI: `http://localhost:3284/metrics`
- FastMCP: `http://localhost:8000/metrics`

### Logging Integration

Logs are written to:
- `/var/log/agentapi/agentapi.log` - AgentAPI logs
- `/var/log/agentapi/fastmcp.log` - FastMCP logs
- `/var/log/agentapi/supervisord.log` - Supervisor logs

Configure log drivers for centralized logging:

```bash
docker run -p 3284:3284 -p 8000:8000 \
  --log-driver=json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  agentapi-multitenant:latest
```

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   # Use different ports
   docker run -p 8284:3284 -p 9000:8000 agentapi-multitenant:latest
   ```

2. **Permission denied on /workspaces**
   ```bash
   # Check volume permissions
   docker exec <container_id> ls -la /workspaces
   ```

3. **Service won't start**
   ```bash
   # Check supervisord status
   docker exec <container_id> supervisorctl status

   # View error logs
   docker exec <container_id> cat /var/log/agentapi/agentapi_error.log
   ```

4. **Build fails**
   ```bash
   # Build with verbose output
   docker build --progress=plain -f Dockerfile.multitenant -t agentapi-multitenant:latest .
   ```

## Security Scanning

Run security scans before deploying:

```bash
# Using Trivy
trivy image agentapi-multitenant:latest

# Using Docker Scout
docker scout cves agentapi-multitenant:latest

# Using Snyk
snyk container test agentapi-multitenant:latest
```

## License

See LICENSE file in the repository root.

## Support

For issues and questions:
- GitHub Issues: https://github.com/coder/agentapi/issues
- Documentation: https://github.com/coder/agentapi

## Contributing

See CONTRIBUTING.md for development guidelines.

# Docker Multi-Tenant Quick Reference

Quick command reference for building and running the multi-tenant AgentAPI Docker container.

## Build Commands

```bash
# Quick build (default settings)
./build-multitenant.sh

# Build specific version
./build-multitenant.sh --tag v2.0.0

# Build and push to registry
./build-multitenant.sh --registry docker.io/myuser --push --tag latest

# Clean build (no cache)
./build-multitenant.sh --no-cache

# Manual Docker build
docker build -f Dockerfile.multitenant -t agentapi-multitenant:latest .
```

## Run Commands

```bash
# Basic run
docker run -p 3284:3284 -p 8000:8000 agentapi-multitenant:latest

# Run with custom environment
docker run -p 3284:3284 -p 8000:8000 \
  -e AGENTAPI_PORT=3284 \
  -e FASTMCP_PORT=8000 \
  -e FASTMCP_WORKERS=4 \
  agentapi-multitenant:latest

# Run with volume mount
docker run -p 3284:3284 -p 8000:8000 \
  -v ./workspaces:/workspaces \
  agentapi-multitenant:latest

# Run in background (detached)
docker run -d -p 3284:3284 -p 8000:8000 \
  --name agentapi-multitenant \
  agentapi-multitenant:latest

# Run with Docker Compose
docker-compose -f docker-compose.multitenant.yml up -d
```

## Management Commands

```bash
# View logs
docker logs -f agentapi-multitenant

# View specific service logs
docker exec agentapi-multitenant tail -f /var/log/agentapi/agentapi.log
docker exec agentapi-multitenant tail -f /var/log/agentapi/fastmcp.log

# Check service status
docker exec agentapi-multitenant supervisorctl status

# Restart services
docker exec agentapi-multitenant supervisorctl restart agentapi
docker exec agentapi-multitenant supervisorctl restart fastmcp

# Access container shell
docker exec -it agentapi-multitenant /bin/bash

# Stop container
docker stop agentapi-multitenant

# Remove container
docker rm agentapi-multitenant
```

## Health Checks

```bash
# Check AgentAPI health
curl http://localhost:3284/health

# Check FastMCP health
curl http://localhost:8000/health

# Check from inside container
docker exec agentapi-multitenant curl -f http://localhost:3284/health
docker exec agentapi-multitenant curl -f http://localhost:8000/health
```

## Debugging

```bash
# View all environment variables
docker exec agentapi-multitenant env

# Check running processes
docker exec agentapi-multitenant ps aux

# Check network connectivity
docker exec agentapi-multitenant netstat -tlnp

# View supervisord configuration
docker exec agentapi-multitenant cat /etc/supervisor/conf.d/agentapi.conf

# Manually start a service
docker exec agentapi-multitenant supervisorctl start agentapi
docker exec agentapi-multitenant supervisorctl start fastmcp
```

## Image Management

```bash
# List images
docker images | grep agentapi

# Inspect image
docker inspect agentapi-multitenant:latest

# View image history
docker history agentapi-multitenant:latest

# Check image size
docker images agentapi-multitenant:latest --format "{{.Size}}"

# Remove image
docker rmi agentapi-multitenant:latest

# Prune unused images
docker image prune -a
```

## Registry Operations

```bash
# Tag image for registry
docker tag agentapi-multitenant:latest myregistry.com/agentapi-multitenant:latest

# Push to registry
docker push myregistry.com/agentapi-multitenant:latest

# Pull from registry
docker pull myregistry.com/agentapi-multitenant:latest

# Login to registry
docker login myregistry.com
```

## Security Scanning

```bash
# Scan with Trivy
trivy image agentapi-multitenant:latest

# Scan with Docker Scout
docker scout cves agentapi-multitenant:latest

# Quick vulnerability check
docker scan agentapi-multitenant:latest
```

## Docker Compose

```bash
# Start services
docker-compose -f docker-compose.multitenant.yml up

# Start in background
docker-compose -f docker-compose.multitenant.yml up -d

# Stop services
docker-compose -f docker-compose.multitenant.yml down

# View logs
docker-compose -f docker-compose.multitenant.yml logs -f

# Restart services
docker-compose -f docker-compose.multitenant.yml restart

# Scale services
docker-compose -f docker-compose.multitenant.yml up -d --scale agentapi=3
```

## Ports & Services

| Port | Service | Endpoint |
|------|---------|----------|
| 3284 | AgentAPI | http://localhost:3284 |
| 8000 | FastMCP | http://localhost:8000 |

## Key Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check (AgentAPI) |
| `GET /health` | Health check (FastMCP) |
| `POST /mcp/connect` | Connect to MCP server |
| `POST /mcp/call_tool` | Call MCP tool |
| `GET /mcp/list_tools` | List available tools |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTAPI_PORT` | `3284` | AgentAPI server port |
| `FASTMCP_PORT` | `8000` | FastMCP service port |
| `FASTMCP_WORKERS` | `4` | Number of FastMCP workers |
| `PYTHONUNBUFFERED` | `1` | Python output buffering |

## Volumes

| Path | Purpose | Permissions |
|------|---------|-------------|
| `/workspaces` | Tenant workspaces | 1777 (rwxrwxrwt) |
| `/var/log/agentapi` | Application logs | 755 |
| `/app` | Working directory | 755 |

## Common Issues

### Port Already in Use
```bash
# Use different ports
docker run -p 8284:3284 -p 9000:8000 agentapi-multitenant:latest
```

### Service Not Starting
```bash
# Check logs
docker exec agentapi-multitenant cat /var/log/agentapi/agentapi_error.log

# Check supervisord status
docker exec agentapi-multitenant supervisorctl status
```

### Permission Issues
```bash
# Check file permissions
docker exec agentapi-multitenant ls -la /workspaces

# Check running user
docker exec agentapi-multitenant whoami
```

### Build Failures
```bash
# Clear cache and rebuild
docker build --no-cache -f Dockerfile.multitenant -t agentapi-multitenant:latest .

# Build with verbose output
docker build --progress=plain -f Dockerfile.multitenant -t agentapi-multitenant:latest .
```

## Performance Tuning

```bash
# Increase FastMCP workers
docker run -e FASTMCP_WORKERS=8 -p 3284:3284 -p 8000:8000 agentapi-multitenant:latest

# Set resource limits
docker run --cpus=2 --memory=4g -p 3284:3284 -p 8000:8000 agentapi-multitenant:latest

# Enable performance monitoring
docker stats agentapi-multitenant
```

## Production Deployment

```bash
# Run with restart policy
docker run -d --restart=unless-stopped \
  -p 3284:3284 -p 8000:8000 \
  --name agentapi-multitenant \
  agentapi-multitenant:latest

# Run with health check override
docker run -d \
  --health-cmd="curl -f http://localhost:3284/health || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  -p 3284:3284 -p 8000:8000 \
  agentapi-multitenant:latest
```

## Backup & Restore

```bash
# Backup workspaces
docker run --rm -v agentapi-workspaces:/workspaces -v $(pwd):/backup \
  alpine tar czf /backup/workspaces-backup.tar.gz /workspaces

# Restore workspaces
docker run --rm -v agentapi-workspaces:/workspaces -v $(pwd):/backup \
  alpine tar xzf /backup/workspaces-backup.tar.gz -C /
```

## Network Configuration

```bash
# Create custom network
docker network create agentapi-network

# Run with custom network
docker run -d --network agentapi-network \
  -p 3284:3284 -p 8000:8000 \
  --name agentapi-multitenant \
  agentapi-multitenant:latest
```

## Tips

1. **Always use specific tags in production** (avoid `:latest`)
2. **Monitor resource usage** with `docker stats`
3. **Set up log rotation** for production deployments
4. **Use health checks** for automatic recovery
5. **Regular security scans** with Trivy or similar tools
6. **Backup workspaces** regularly
7. **Use Docker Compose** for easier management
8. **Set resource limits** to prevent resource exhaustion

## Further Reading

- [DOCKER_MULTITENANT.md](./DOCKER_MULTITENANT.md) - Comprehensive documentation
- [docker-compose.multitenant.yml](./docker-compose.multitenant.yml) - Docker Compose configuration
- [Dockerfile.multitenant](./Dockerfile.multitenant) - Dockerfile source

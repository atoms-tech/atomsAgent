# Docker Compose Quick Start Guide

Get AgentAPI running in minutes with Docker Compose!

## Prerequisites

- Docker Desktop (macOS/Windows) or Docker Engine + Docker Compose (Linux)
- At least 8GB RAM and 10GB free disk space
- Required API keys (see Configuration below)

## 5-Minute Setup

### 1. Copy Environment File

```bash
cp .env.docker .env
```

### 2. Configure Essential Variables

Edit `.env` and set these **required** variables:

```bash
# Supabase (Required)
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# AI Provider (Required - at least one)
ANTHROPIC_API_KEY=sk-ant-api03-your-key

# Optional: For Vertex AI support
VERTEX_AI_API_KEY=your-gcp-credentials
VERTEX_AI_PROJECT_ID=your-project-id
```

### 3. Start Services

**Option A: Using Make (Recommended)**
```bash
make -f Makefile.docker init    # First time setup
make -f Makefile.docker start   # Start services
make -f Makefile.docker status  # Check health
```

**Option B: Using Helper Script**
```bash
./docker-manage.sh start
./docker-manage.sh status
```

**Option C: Using Docker Compose Directly**
```bash
mkdir -p data/workspaces data/postgres data/redis
docker-compose -f docker-compose.multitenant.yml up -d
docker-compose -f docker-compose.multitenant.yml ps
```

### 4. Verify Installation

```bash
# Check services are running
curl http://localhost:3284/status  # AgentAPI
curl http://localhost:8000/health  # FastMCP

# Or use the management tools
make -f Makefile.docker status
# or
./docker-manage.sh status
```

You should see:
- AgentAPI running on port 3284
- FastMCP running on port 8000
- PostgreSQL running on port 5432
- Redis running on port 6379

## Services Overview

| Service | Port | Purpose | Required |
|---------|------|---------|----------|
| AgentAPI | 3284 | Main Go backend API | Yes |
| FastMCP | 8000 | Python MCP service | Yes |
| PostgreSQL | 5432 | Database (optional, can use external) | Optional |
| Redis | 6379 | Cache and sessions | Optional |

## Common Operations

### View Logs

```bash
# All services
make -f Makefile.docker logs

# Specific service
make -f Makefile.docker logs-agentapi
docker-compose -f docker-compose.multitenant.yml logs -f agentapi
```

### Restart Services

```bash
# All services
make -f Makefile.docker restart

# Single service
make -f Makefile.docker restart-agentapi
docker-compose -f docker-compose.multitenant.yml restart agentapi
```

### Stop Services

```bash
make -f Makefile.docker stop
# or
docker-compose -f docker-compose.multitenant.yml stop
```

### Database Access

```bash
# PostgreSQL shell
make -f Makefile.docker db-shell
# or
docker-compose -f docker-compose.multitenant.yml exec postgres psql -U agentapi -d agentapi
```

### Redis Access

```bash
# Redis CLI
make -f Makefile.docker redis-shell
# or
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli
```

### Container Shell Access

```bash
# AgentAPI container
make -f Makefile.docker shell
# or
docker-compose -f docker-compose.multitenant.yml exec agentapi sh
```

## Resource Monitoring

```bash
# View resource usage
make -f Makefile.docker stats
# or
docker stats
```

## Backup Data

```bash
# Create full backup
make -f Makefile.docker backup

# Or manually
./docker-manage.sh backup
```

Backups are stored in `./backups/` directory.

## Cleanup

```bash
# Stop and remove containers (keeps data)
make -f Makefile.docker clean

# Remove everything including data (DESTRUCTIVE!)
make -f Makefile.docker clean-all
```

## Configuration Options

### Using External Database

If you want to use an external PostgreSQL database (like Supabase):

1. Edit `.env`:
```bash
DATABASE_URL=postgresql://user:password@external-host:5432/dbname?sslmode=require
```

2. Comment out postgres dependency in `docker-compose.multitenant.yml`:
```yaml
depends_on:
  # postgres:
  #   condition: service_healthy
  redis:
    condition: service_healthy
```

3. Stop local postgres:
```bash
docker-compose -f docker-compose.multitenant.yml stop postgres
```

### Resource Limits

Default resource limits:

**AgentAPI:**
- CPU: 2.0 max, 1.0 reserved
- Memory: 4GB max, 2GB reserved

**PostgreSQL:**
- CPU: 1.0 max, 0.5 reserved
- Memory: 2GB max, 512MB reserved

**Redis:**
- CPU: 0.5 max, 0.25 reserved
- Memory: 512MB max, 256MB reserved

To adjust, edit `docker-compose.multitenant.yml` under each service's `deploy.resources` section.

### Environment Variables

See `.env.docker` for all available configuration options, including:

- Database configuration
- AI provider keys (Anthropic, Vertex AI)
- OAuth providers (GitHub, Google, Azure, Auth0)
- Logging and monitoring
- Feature flags
- Security settings

## Troubleshooting

### Services Won't Start

```bash
# Check logs
docker-compose -f docker-compose.multitenant.yml logs

# Rebuild from scratch
docker-compose -f docker-compose.multitenant.yml down -v
docker-compose -f docker-compose.multitenant.yml build --no-cache
docker-compose -f docker-compose.multitenant.yml up -d
```

### Port Already in Use

```bash
# Find what's using the port
lsof -i :3284

# Change port in .env (example)
AGENTAPI_PORT=3285
```

Then update port mapping in `docker-compose.multitenant.yml`.

### Database Connection Issues

```bash
# Verify postgres is running
docker-compose -f docker-compose.multitenant.yml ps postgres

# Check health
docker-compose -f docker-compose.multitenant.yml exec postgres pg_isready

# Verify connection string
docker-compose -f docker-compose.multitenant.yml exec agentapi env | grep DATABASE_URL
```

### Out of Disk Space

```bash
# Check usage
docker system df

# Clean up
docker system prune -a
docker volume prune
```

### Permission Issues

```bash
# Fix data directory permissions
sudo chown -R $USER:$USER data/
```

## Development Mode

For active development with live logs:

```bash
make -f Makefile.docker dev
```

This builds, starts, and tails logs for all services.

## Production Deployment

See [DOCKER_COMPOSE_README.md](./DOCKER_COMPOSE_README.md) for detailed production deployment guide, including:

- Security hardening
- SSL/TLS configuration
- High availability setup
- Monitoring and observability
- Backup strategies

## Management Tools

Three ways to manage services:

1. **Makefile** (Recommended for frequent use)
   ```bash
   make -f Makefile.docker help
   ```

2. **Helper Script** (Recommended for beginners)
   ```bash
   ./docker-manage.sh help
   ```

3. **Docker Compose** (Direct control)
   ```bash
   docker-compose -f docker-compose.multitenant.yml --help
   ```

## Next Steps

1. **Access the API**: http://localhost:3284
2. **View API Docs**: http://localhost:3284/docs (if available)
3. **Check FastMCP**: http://localhost:8000/docs
4. **Monitor Logs**: `make -f Makefile.docker logs`
5. **Read Full Docs**: [DOCKER_COMPOSE_README.md](./DOCKER_COMPOSE_README.md)

## Support

- Full Documentation: [DOCKER_COMPOSE_README.md](./DOCKER_COMPOSE_README.md)
- Multi-Tenant Guide: [MULTITENANT.md](./MULTITENANT.md)
- CCRouter Setup: [CCROUTER_QUICK_REFERENCE.md](./CCROUTER_QUICK_REFERENCE.md)

## Tips

- Use `make -f Makefile.docker help` to see all available commands
- Always backup data before major updates
- Check logs regularly for errors
- Monitor resource usage with `make -f Makefile.docker stats`
- Keep your `.env` file secure and never commit it to version control

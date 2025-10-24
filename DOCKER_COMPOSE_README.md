# Docker Compose Multi-Tenant Setup Guide

This guide explains how to run the AgentAPI multi-tenant application using Docker Compose for local development and production deployments.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Services](#services)
- [Volume Management](#volume-management)
- [Network Configuration](#network-configuration)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)
- [Maintenance](#maintenance)

## Architecture Overview

The Docker Compose setup includes the following services:

```
┌─────────────────────────────────────────────────┐
│              agentapi-network                   │
│  ┌──────────────────────────────────────────┐  │
│  │  AgentAPI Service                        │  │
│  │  - Go Backend (Port 3284)                │  │
│  │  - Python FastMCP Service (Port 8000)    │  │
│  │  - CCRouter Integration                  │  │
│  │  - Multi-tenant Support                  │  │
│  └──────────────┬───────────────────────────┘  │
│                 │                               │
│  ┌──────────────▼───────────┐  ┌────────────┐  │
│  │  PostgreSQL Database     │  │   Redis    │  │
│  │  - Port 5432             │  │  - Port    │  │
│  │  - Persistent Storage    │  │    6379    │  │
│  └──────────────────────────┘  └────────────┘  │
└─────────────────────────────────────────────────┘
```

## Prerequisites

- **Docker**: Version 20.10 or higher
- **Docker Compose**: Version 2.0 or higher
- **Disk Space**: At least 10GB free space
- **Memory**: At least 8GB RAM recommended
- **CPU**: 2+ cores recommended

### Install Docker

**macOS:**
```bash
brew install --cask docker
```

**Linux:**
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
```

**Windows:**
Download and install [Docker Desktop](https://www.docker.com/products/docker-desktop)

## Quick Start

### 1. Clone and Navigate to Project

```bash
cd /path/to/agentapi
```

### 2. Set Up Environment Variables

Copy the example environment file and configure it:

```bash
cp .env.docker .env
```

Edit `.env` with your actual values:

```bash
# Required: Supabase Configuration
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# Required: AI Provider API Keys
ANTHROPIC_API_KEY=sk-ant-api03-your-key

# Optional: Vertex AI Support
VERTEX_AI_API_KEY=your-gcp-credentials
VERTEX_AI_PROJECT_ID=your-gcp-project
```

### 3. Create Required Directories

```bash
mkdir -p data/workspaces data/postgres data/redis logs/nginx
```

### 4. Build and Start Services

```bash
# Build the images
docker-compose -f docker-compose.multitenant.yml build

# Start all services
docker-compose -f docker-compose.multitenant.yml up -d

# View logs
docker-compose -f docker-compose.multitenant.yml logs -f
```

### 5. Verify Services

Check that all services are running:

```bash
docker-compose -f docker-compose.multitenant.yml ps
```

Test the health endpoints:

```bash
# AgentAPI health check
curl http://localhost:3284/status

# FastMCP service health check
curl http://localhost:8000/health
```

## Configuration

### Environment Variables

All configuration is done through environment variables in the `.env` file.

#### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `SUPABASE_URL` | Supabase project URL | `https://xyz.supabase.co` |
| `SUPABASE_ANON_KEY` | Supabase anonymous key | `eyJhbGc...` |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase service role key | `eyJhbGc...` |
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude | `sk-ant-api03-...` |

#### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VERTEX_AI_API_KEY` | Google Cloud credentials | - |
| `VERTEX_AI_PROJECT_ID` | GCP project ID | - |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://agentapi:agentapi@postgres:5432/agentapi` |
| `REDIS_URL` | Redis connection string | `redis://redis:6379` |
| `NODE_ENV` | Environment mode | `production` |
| `LOG_LEVEL` | Logging level | `info` |

### Database Configuration

#### Using Local PostgreSQL (Default)

The compose file includes a PostgreSQL service for local development:

```yaml
DATABASE_URL=postgresql://agentapi:agentapi@postgres:5432/agentapi?sslmode=disable
POSTGRES_USER=agentapi
POSTGRES_PASSWORD=agentapi
POSTGRES_DB=agentapi
```

#### Using External Database

To use an external database (e.g., Supabase PostgreSQL):

```bash
# In .env file
DATABASE_URL=postgresql://user:password@external-host:5432/dbname?sslmode=require
```

Then comment out the `postgres` service dependency in `docker-compose.multitenant.yml`:

```yaml
depends_on:
  # postgres:
  #   condition: service_healthy
  redis:
    condition: service_healthy
```

## Services

### AgentAPI Service

**Ports:**
- `3284` - Go backend API
- `8000` - Python FastMCP service

**Resource Limits:**
- CPU: 2.0 max, 1.0 reserved
- Memory: 4GB max, 2GB reserved

**Volumes:**
- `/workspaces` - User workspace data
- `/app/database` - Database schema and migrations

**Management Commands:**

```bash
# View logs
docker-compose -f docker-compose.multitenant.yml logs -f agentapi

# Restart service
docker-compose -f docker-compose.multitenant.yml restart agentapi

# Execute commands inside container
docker-compose -f docker-compose.multitenant.yml exec agentapi sh

# Scale service (if needed)
docker-compose -f docker-compose.multitenant.yml up -d --scale agentapi=2
```

### PostgreSQL Service

**Port:** `5432`

**Resource Limits:**
- CPU: 1.0 max, 0.5 reserved
- Memory: 2GB max, 512MB reserved

**Management Commands:**

```bash
# Access PostgreSQL shell
docker-compose -f docker-compose.multitenant.yml exec postgres psql -U agentapi -d agentapi

# Create database backup
docker-compose -f docker-compose.multitenant.yml exec postgres pg_dump -U agentapi agentapi > backup.sql

# Restore database backup
docker-compose -f docker-compose.multitenant.yml exec -T postgres psql -U agentapi agentapi < backup.sql

# View database logs
docker-compose -f docker-compose.multitenant.yml logs -f postgres
```

### Redis Service

**Port:** `6379`

**Resource Limits:**
- CPU: 0.5 max, 0.25 reserved
- Memory: 512MB max, 256MB reserved

**Management Commands:**

```bash
# Access Redis CLI
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli

# Monitor Redis operations
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli MONITOR

# Check Redis info
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli INFO

# Flush all data (use with caution!)
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli FLUSHALL
```

## Volume Management

### Volume Structure

```
data/
├── workspaces/     # User workspace files
├── postgres/       # PostgreSQL data
└── redis/          # Redis cache data

logs/
└── nginx/          # Nginx logs (if using nginx)
```

### Backup Volumes

```bash
# Backup workspace data
tar -czf workspaces-backup-$(date +%Y%m%d).tar.gz data/workspaces/

# Backup PostgreSQL data
docker-compose -f docker-compose.multitenant.yml exec postgres \
  pg_dump -U agentapi agentapi | gzip > postgres-backup-$(date +%Y%m%d).sql.gz

# Backup Redis data
docker-compose -f docker-compose.multitenant.yml exec redis \
  redis-cli --rdb /data/dump.rdb
cp data/redis/dump.rdb redis-backup-$(date +%Y%m%d).rdb
```

### Restore Volumes

```bash
# Restore workspace data
tar -xzf workspaces-backup-YYYYMMDD.tar.gz -C data/

# Restore PostgreSQL data
gunzip < postgres-backup-YYYYMMDD.sql.gz | \
  docker-compose -f docker-compose.multitenant.yml exec -T postgres \
  psql -U agentapi agentapi

# Restore Redis data
docker-compose -f docker-compose.multitenant.yml stop redis
cp redis-backup-YYYYMMDD.rdb data/redis/dump.rdb
docker-compose -f docker-compose.multitenant.yml start redis
```

### Clean Up Volumes

```bash
# Stop all services
docker-compose -f docker-compose.multitenant.yml down

# Remove all volumes (WARNING: This deletes all data!)
docker-compose -f docker-compose.multitenant.yml down -v

# Remove specific volume
docker volume rm agentapi_workspace_data
```

## Network Configuration

### Default Network

The compose file creates a custom bridge network `agentapi-network` with subnet `172.20.0.0/16`.

### Custom Network Subnet

To change the network subnet:

```bash
# In .env file
NETWORK_SUBNET=172.30.0.0/16
```

### Service Communication

Services can communicate using their service names:

```bash
# From agentapi service to postgres
postgresql://postgres:5432/agentapi

# From agentapi service to redis
redis://redis:6379
```

## Production Deployment

### Security Hardening

1. **Change Default Passwords:**

```bash
# Generate strong passwords
openssl rand -base64 32

# Update .env
POSTGRES_PASSWORD=<strong-password>
SESSION_SECRET=<strong-secret>
JWT_SECRET=<strong-secret>
```

2. **Enable SSL/TLS:**

Uncomment the nginx service in `docker-compose.multitenant.yml` and configure SSL certificates:

```bash
# Create SSL directory
mkdir -p nginx/ssl

# Place your certificates
cp your-cert.crt nginx/ssl/
cp your-cert.key nginx/ssl/
```

3. **Restrict Network Access:**

```yaml
# In docker-compose.multitenant.yml, remove port mappings for internal services
postgres:
  # ports:
  #   - "5432:5432"  # Only expose internally
```

4. **Enable Firewall:**

```bash
# Allow only necessary ports
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3284/tcp
sudo ufw enable
```

### Resource Optimization

For production workloads, adjust resource limits based on your needs:

```yaml
deploy:
  resources:
    limits:
      cpus: '4.0'      # Increase for higher load
      memory: 8G       # Increase for more concurrent users
    reservations:
      cpus: '2.0'
      memory: 4G
```

### Monitoring

Add monitoring services:

```bash
# Prometheus for metrics
docker-compose -f docker-compose.multitenant.yml -f docker-compose.monitoring.yml up -d

# View metrics
curl http://localhost:9090/metrics
```

### High Availability

For production HA setup:

1. Use external managed services:
   - Supabase PostgreSQL
   - Redis Cloud or AWS ElastiCache
   - Cloud Load Balancer

2. Scale AgentAPI horizontally:

```bash
docker-compose -f docker-compose.multitenant.yml up -d --scale agentapi=3
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check logs
docker-compose -f docker-compose.multitenant.yml logs agentapi

# Check service health
docker-compose -f docker-compose.multitenant.yml ps

# Rebuild from scratch
docker-compose -f docker-compose.multitenant.yml down -v
docker-compose -f docker-compose.multitenant.yml build --no-cache
docker-compose -f docker-compose.multitenant.yml up -d
```

#### Database Connection Issues

```bash
# Verify database is running
docker-compose -f docker-compose.multitenant.yml exec postgres pg_isready

# Check connection from agentapi
docker-compose -f docker-compose.multitenant.yml exec agentapi \
  wget --spider http://postgres:5432 2>&1 | grep connected

# Verify DATABASE_URL
docker-compose -f docker-compose.multitenant.yml exec agentapi env | grep DATABASE_URL
```

#### Port Already in Use

```bash
# Find process using port
lsof -i :3284

# Kill the process
kill -9 <PID>

# Or change the port in .env
AGENTAPI_PORT=3285
```

#### Out of Disk Space

```bash
# Check disk usage
docker system df

# Clean up unused resources
docker system prune -a

# Remove old images
docker image prune -a

# Remove unused volumes
docker volume prune
```

### Debug Mode

Enable debug logging:

```bash
# In .env
LOG_LEVEL=debug
DEBUG=true

# Restart services
docker-compose -f docker-compose.multitenant.yml restart
```

### Performance Issues

```bash
# Check container stats
docker stats

# View resource usage
docker-compose -f docker-compose.multitenant.yml top

# Increase resource limits in docker-compose.multitenant.yml
```

## Maintenance

### Update Services

```bash
# Pull latest images
docker-compose -f docker-compose.multitenant.yml pull

# Rebuild and restart
docker-compose -f docker-compose.multitenant.yml up -d --build

# View updated containers
docker-compose -f docker-compose.multitenant.yml ps
```

### Log Rotation

Logs are automatically rotated with the json-file driver:

```yaml
logging:
  driver: json-file
  options:
    max-size: "10m"
    max-file: "3"
```

Manual log cleanup:

```bash
# Clear all logs
docker-compose -f docker-compose.multitenant.yml down
rm -rf /var/lib/docker/containers/*/*-json.log
docker-compose -f docker-compose.multitenant.yml up -d
```

### Database Maintenance

```bash
# Vacuum database
docker-compose -f docker-compose.multitenant.yml exec postgres \
  psql -U agentapi -d agentapi -c "VACUUM ANALYZE;"

# Check database size
docker-compose -f docker-compose.multitenant.yml exec postgres \
  psql -U agentapi -d agentapi -c "SELECT pg_size_pretty(pg_database_size('agentapi'));"

# Reindex database
docker-compose -f docker-compose.multitenant.yml exec postgres \
  psql -U agentapi -d agentapi -c "REINDEX DATABASE agentapi;"
```

### Health Checks

All services include health checks. View status:

```bash
# Check all service health
docker-compose -f docker-compose.multitenant.yml ps

# View detailed health
docker inspect --format='{{json .State.Health}}' agentapi | jq
```

## Additional Resources

- [AgentAPI Documentation](./README.md)
- [Multi-Tenant Architecture](./MULTITENANT.md)
- [CCRouter Integration](./CCROUTER_QUICK_REFERENCE.md)
- [FastMCP Service](./lib/mcp/FASTMCP_SERVICE_README.md)

## Support

For issues and questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review logs: `docker-compose -f docker-compose.multitenant.yml logs`
3. Open an issue on GitHub

## License

See [LICENSE](./LICENSE) file for details.

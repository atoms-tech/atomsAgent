# Redis Configuration Guide

This guide explains how to configure Redis for AgentAPI in different deployment scenarios.

## Table of Contents

- [Overview](#overview)
- [Deployment Options](#deployment-options)
  - [Option 1: Upstash Redis (Production)](#option-1-upstash-redis-production)
  - [Option 2: Local Redis (Development)](#option-2-local-redis-development)
- [Environment Variables](#environment-variables)
- [Docker Compose Configuration](#docker-compose-configuration)
- [Health Checks](#health-checks)
- [Redis Insights](#redis-insights)
- [Features Using Redis](#features-using-redis)
- [Troubleshooting](#troubleshooting)

## Overview

AgentAPI uses Redis for:

- **Session Storage**: User sessions and authentication state
- **Rate Limiting**: API rate limiting and throttling
- **Circuit Breaker**: Service resilience and failure detection
- **Caching**: Performance optimization and data caching
- **Multi-tenant State**: Tenant-specific configuration and state

The application supports two Redis deployment modes:

1. **Upstash Redis** (Recommended for production)
2. **Local Redis** (Development and testing)

## Deployment Options

### Option 1: Upstash Redis (Production)

**Recommended for:**
- Production deployments
- Serverless/cloud environments
- No infrastructure management
- Automatic scaling
- High availability

**Setup Steps:**

1. **Create an Upstash Redis Instance**
   - Visit [Upstash Console](https://console.upstash.com)
   - Create a new Redis database
   - Choose your region (closest to your deployment)
   - Select a plan (free tier available)

2. **Get Credentials**
   - Copy the following from Upstash console:
     - REST URL (e.g., `https://your-instance.upstash.io`)
     - REST Token
     - Redis URL (e.g., `rediss://default:password@your-instance.upstash.io:6379`)

3. **Configure Environment Variables**

   Add to your `.env` file:
   ```bash
   # Enable Redis
   REDIS_ENABLE=true
   REDIS_PROTOCOL=native  # or 'rest' for REST API fallback

   # Upstash Redis Credentials
   UPSTASH_REDIS_REST_URL=https://your-instance.upstash.io
   UPSTASH_REDIS_REST_TOKEN=your-rest-token-here
   UPSTASH_REDIS_URL=rediss://default:your-password@your-instance.upstash.io:6379

   # Connection Settings
   REDIS_MAX_POOL_SIZE=10
   REDIS_CONNECTION_TIMEOUT=5s
   ```

4. **Update Docker Compose**

   Comment out the local `redis` service in `docker-compose.multitenant.yml`:
   ```yaml
   # Comment out or remove the entire redis service block
   # redis:
   #   image: redis:7-alpine
   #   ...
   ```

5. **Remove Redis from depends_on**

   In the `agentapi` service, update:
   ```yaml
   depends_on:
     postgres:
       condition: service_healthy
     # Remove redis dependency when using Upstash
   ```

### Option 2: Local Redis (Development)

**Recommended for:**
- Local development
- Testing
- CI/CD pipelines
- Offline development

**Setup Steps:**

1. **Configure Environment Variables**

   Add to your `.env` file:
   ```bash
   # Enable Redis
   REDIS_ENABLE=true
   REDIS_PROTOCOL=native

   # Local Redis Configuration
   REDIS_URL=redis://redis:6379
   REDIS_MAX_POOL_SIZE=10
   REDIS_CONNECTION_TIMEOUT=5s

   # Redis Server Settings
   REDIS_PORT=6379
   REDIS_MAX_MEMORY=256mb
   REDIS_LOG_LEVEL=notice
   REDIS_PASSWORD=  # Optional, leave empty for no password
   ```

2. **Keep Redis Service in Docker Compose**

   The `redis` service in `docker-compose.multitenant.yml` should remain uncommented:
   ```yaml
   redis:
     image: redis:7-alpine
     container_name: agentapi-redis
     ports:
       - "${REDIS_PORT:-6379}:6379"
     # ... rest of configuration
   ```

3. **Ensure Redis is in depends_on**

   In the `agentapi` service:
   ```yaml
   depends_on:
     postgres:
       condition: service_healthy
     redis:
       condition: service_healthy
   ```

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `REDIS_ENABLE` | Enable/disable Redis | `true` or `false` |
| `REDIS_PROTOCOL` | Connection protocol | `native` or `rest` |

### Upstash Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `UPSTASH_REDIS_REST_URL` | Upstash REST API URL | `https://your-instance.upstash.io` |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash REST API token | `AYseAAIncD...` |
| `UPSTASH_REDIS_URL` | Upstash native Redis URL | `rediss://default:pass@host:6379` |

### Local Redis Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_URL` | Local Redis connection URL | `redis://redis:6379` |
| `REDIS_PORT` | Redis server port | `6379` |
| `REDIS_MAX_MEMORY` | Maximum memory allocation | `256mb` |
| `REDIS_LOG_LEVEL` | Redis log level | `notice` |
| `REDIS_PASSWORD` | Redis password (optional) | Empty |

### Connection Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_MAX_POOL_SIZE` | Maximum connection pool size | `10` |
| `REDIS_CONNECTION_TIMEOUT` | Connection timeout | `5s` |

### Rate Limiting

| Variable | Description | Default |
|----------|-------------|---------|
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `true` |
| `RATE_LIMIT_REQUESTS_PER_MINUTE` | Requests per minute limit | `60` |
| `RATE_LIMIT_BURST_SIZE` | Burst size for rate limiting | `10` |

### Circuit Breaker

| Variable | Description | Default |
|----------|-------------|---------|
| `CIRCUIT_BREAKER_ENABLED` | Enable circuit breaker | `true` |
| `CIRCUIT_BREAKER_FAILURE_THRESHOLD` | Failures before opening | `5` |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | Successes before closing | `2` |
| `CIRCUIT_BREAKER_TIMEOUT` | Circuit breaker timeout | `30s` |

### Session Storage

| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_STORAGE` | Session storage backend | `redis` or `memory` |
| `SESSION_TTL` | Session time-to-live | `3600s` |
| `SESSION_CLEANUP_INTERVAL` | Cleanup interval | `300s` |
| `TOKEN_ENCRYPTION_KEY` | 32-byte encryption key | Generate with `openssl rand -hex 32` |

## Docker Compose Configuration

### Full AgentAPI Service with Redis

```yaml
agentapi:
  build:
    context: .
    dockerfile: Dockerfile.multitenant
  environment:
    # Redis Configuration
    - REDIS_ENABLE=${REDIS_ENABLE:-true}
    - REDIS_PROTOCOL=${REDIS_PROTOCOL:-native}

    # Upstash Redis (production)
    - UPSTASH_REDIS_REST_URL=${UPSTASH_REDIS_REST_URL:-}
    - UPSTASH_REDIS_REST_TOKEN=${UPSTASH_REDIS_REST_TOKEN:-}
    - UPSTASH_REDIS_URL=${UPSTASH_REDIS_URL:-}

    # Local Redis (fallback)
    - REDIS_URL=${REDIS_URL:-redis://redis:6379}
    - REDIS_MAX_POOL_SIZE=${REDIS_MAX_POOL_SIZE:-10}
    - REDIS_CONNECTION_TIMEOUT=${REDIS_CONNECTION_TIMEOUT:-5s}

    # Rate Limiting
    - RATE_LIMIT_ENABLED=${RATE_LIMIT_ENABLED:-true}
    - RATE_LIMIT_REQUESTS_PER_MINUTE=${RATE_LIMIT_REQUESTS_PER_MINUTE:-60}
    - RATE_LIMIT_BURST_SIZE=${RATE_LIMIT_BURST_SIZE:-10}

    # Circuit Breaker
    - CIRCUIT_BREAKER_ENABLED=${CIRCUIT_BREAKER_ENABLED:-true}
    - CIRCUIT_BREAKER_FAILURE_THRESHOLD=${CIRCUIT_BREAKER_FAILURE_THRESHOLD:-5}
    - CIRCUIT_BREAKER_SUCCESS_THRESHOLD=${CIRCUIT_BREAKER_SUCCESS_THRESHOLD:-2}
    - CIRCUIT_BREAKER_TIMEOUT=${CIRCUIT_BREAKER_TIMEOUT:-30s}

    # Session Storage
    - SESSION_STORAGE=${SESSION_STORAGE:-redis}
    - SESSION_TTL=${SESSION_TTL:-3600s}
    - SESSION_CLEANUP_INTERVAL=${SESSION_CLEANUP_INTERVAL:-300s}
    - TOKEN_ENCRYPTION_KEY=${TOKEN_ENCRYPTION_KEY}

  depends_on:
    postgres:
      condition: service_healthy
    redis:  # Comment out when using Upstash
      condition: service_healthy
```

### Local Redis Service

```yaml
redis:
  image: redis:7-alpine
  container_name: agentapi-redis
  ports:
    - "${REDIS_PORT:-6379}:6379"

  command: >
    redis-server
    --appendonly yes
    --appendfilename "appendonly.aof"
    --dir /data
    --save 900 1
    --save 300 10
    --save 60 10000
    --maxmemory ${REDIS_MAX_MEMORY:-256mb}
    --maxmemory-policy allkeys-lru
    --loglevel ${REDIS_LOG_LEVEL:-notice}
    --requirepass ${REDIS_PASSWORD:-}

  volumes:
    - redis_data:/data

  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 5s
```

## Health Checks

### AgentAPI Health Check

The AgentAPI service includes a health check that verifies both the application server and Redis connectivity:

```yaml
healthcheck:
  test: ["CMD", "sh", "-c", "wget --no-verbose --tries=1 --spider http://localhost:3284/status || exit 1"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### Redis Health Check

The local Redis service has its own health check:

```yaml
healthcheck:
  test: ["CMD", "redis-cli", "ping"]
  interval: 10s
  timeout: 5s
  retries: 5
  start_period: 5s
```

## Redis Insights

Redis Insights is a GUI tool for visualizing and debugging Redis data.

### Enabling Redis Insights

1. **Start with Development Profile**
   ```bash
   docker-compose --profile development -f docker-compose.multitenant.yml up -d
   ```

2. **Access Redis Insights**
   - Open browser: http://localhost:8001
   - First time: Accept terms and configure
   - Add database connection:
     - Host: `redis`
     - Port: `6379`
     - Name: `AgentAPI Local Redis`

### Use Cases

- **View Cached Data**: Inspect cached API responses and computed values
- **Debug Sessions**: View active user sessions and their data
- **Monitor Rate Limits**: Check rate limit counters for users/IPs
- **Circuit Breaker State**: Inspect circuit breaker states and statistics
- **Query Keys**: Search and filter Redis keys by pattern
- **Memory Analysis**: Analyze memory usage and key distribution

### Configuration

```yaml
redis-insights:
  image: redislabs/redisinsight:latest
  container_name: agentapi-redis-insights
  ports:
    - "8001:8001"
  volumes:
    - redis_insights_data:/db
  depends_on:
    redis:
      condition: service_healthy
  profiles:
    - development
    - debug
```

## Features Using Redis

### 1. Session Storage

- **User Sessions**: Authentication tokens and user state
- **TTL Management**: Automatic session expiration
- **Persistence**: Sessions survive application restarts (with Upstash)

### 2. Rate Limiting

- **Per-User Limits**: Track API usage per user
- **Per-IP Limits**: Protect against abuse
- **Sliding Window**: Accurate rate limiting with burst support
- **Distributed**: Works across multiple application instances

### 3. Circuit Breaker

- **Service Protection**: Prevent cascading failures
- **Failure Detection**: Automatic service health monitoring
- **Recovery**: Gradual service recovery detection
- **Metrics**: Track failure rates and circuit states

### 4. Caching

- **API Response Cache**: Cache frequently accessed data
- **Computed Results**: Cache expensive computations
- **Multi-tenant Data**: Tenant-specific cache isolation
- **Invalidation**: Smart cache invalidation strategies

### 5. Multi-tenant State

- **Tenant Configuration**: Per-tenant settings and preferences
- **Feature Flags**: Tenant-specific feature toggles
- **Usage Tracking**: Per-tenant usage metrics
- **Isolation**: Data isolation between tenants

## Troubleshooting

### Connection Issues

**Problem**: Cannot connect to Redis

**Solutions**:

1. **Check Redis is running**
   ```bash
   docker-compose -f docker-compose.multitenant.yml ps redis
   ```

2. **Verify health check**
   ```bash
   docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping
   ```
   Expected output: `PONG`

3. **Check environment variables**
   ```bash
   docker-compose -f docker-compose.multitenant.yml exec agentapi env | grep REDIS
   ```

4. **View Redis logs**
   ```bash
   docker-compose -f docker-compose.multitenant.yml logs redis
   ```

### Upstash Connection Issues

**Problem**: Cannot connect to Upstash Redis

**Solutions**:

1. **Verify credentials**
   - Check Upstash console for correct URL and token
   - Ensure no trailing spaces in environment variables

2. **Test REST endpoint**
   ```bash
   curl -H "Authorization: Bearer YOUR_REST_TOKEN" \
        https://your-instance.upstash.io/ping
   ```
   Expected output: `{"result":"PONG"}`

3. **Check network connectivity**
   - Ensure firewall allows outbound HTTPS (port 443)
   - Verify DNS resolution

4. **Try REST protocol fallback**
   ```bash
   REDIS_PROTOCOL=rest
   ```

### Performance Issues

**Problem**: Slow Redis operations

**Solutions**:

1. **Check connection pool size**
   ```bash
   # Increase if needed
   REDIS_MAX_POOL_SIZE=20
   ```

2. **Monitor memory usage**
   ```bash
   docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info memory
   ```

3. **Analyze slow queries** (local Redis)
   ```bash
   docker-compose -f docker-compose.multitenant.yml exec redis redis-cli slowlog get 10
   ```

4. **Check network latency** (Upstash)
   - Consider choosing a region closer to your deployment

### Memory Issues

**Problem**: Redis running out of memory

**Solutions**:

1. **Increase max memory** (local Redis)
   ```bash
   REDIS_MAX_MEMORY=512mb
   ```

2. **Check eviction policy**
   - Current policy: `allkeys-lru` (evicts least recently used keys)
   - Adjust if needed in docker-compose.multitenant.yml

3. **Monitor key count**
   ```bash
   docker-compose -f docker-compose.multitenant.yml exec redis redis-cli dbsize
   ```

4. **Clean up old keys**
   - Ensure TTL is set on session keys
   - Verify SESSION_TTL and SESSION_CLEANUP_INTERVAL

### Data Persistence Issues

**Problem**: Data lost after restart

**Solutions**:

1. **Check volume mount** (local Redis)
   ```bash
   docker volume inspect agentapi_redis_data
   ```

2. **Verify persistence settings**
   - Check `appendonly yes` in Redis command
   - Verify save intervals in docker-compose.multitenant.yml

3. **Check disk space**
   ```bash
   df -h
   ```

4. **Upstash automatically persists data**
   - No additional configuration needed

## Best Practices

### Development

- Use local Redis for development
- Enable Redis Insights for debugging
- Use low TTL values for faster testing
- Monitor memory usage regularly

### Production

- Use Upstash Redis for serverless deployment
- Enable rate limiting and circuit breaker
- Set appropriate session TTL (1 hour = 3600s)
- Generate strong TOKEN_ENCRYPTION_KEY
- Monitor Redis metrics and alerts
- Set up backup (Upstash includes automatic backups)

### Security

- Always use TLS in production (Upstash uses `rediss://`)
- Set strong REDIS_PASSWORD for local development
- Rotate TOKEN_ENCRYPTION_KEY periodically
- Limit Redis port exposure (only docker network)
- Use environment variables, never hardcode credentials

### Monitoring

- Track connection pool usage
- Monitor cache hit/miss ratios
- Alert on circuit breaker opens
- Track session count and TTL
- Monitor memory usage trends

## Quick Reference

### Start with Local Redis
```bash
docker-compose -f docker-compose.multitenant.yml up -d
```

### Start with Redis Insights
```bash
docker-compose --profile development -f docker-compose.multitenant.yml up -d
```

### Switch to Upstash
1. Set Upstash environment variables in `.env`
2. Comment out `redis` service in docker-compose.multitenant.yml
3. Remove `redis` from `depends_on` in agentapi service
4. Restart: `docker-compose -f docker-compose.multitenant.yml up -d`

### View Redis Logs
```bash
docker-compose -f docker-compose.multitenant.yml logs -f redis
```

### Connect to Redis CLI
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli
```

### Monitor Redis
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli monitor
```

### Generate Encryption Key
```bash
openssl rand -hex 32
```

## Support

For issues or questions:

- Check [Upstash Documentation](https://docs.upstash.com/redis)
- View [Redis Documentation](https://redis.io/docs/)
- Check application logs: `docker-compose logs agentapi`
- Open an issue in the repository

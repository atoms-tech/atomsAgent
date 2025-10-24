# Docker Redis Setup for AgentAPI

This document describes the Redis integration in the docker-compose.multitenant.yml configuration.

## Overview

The AgentAPI multi-tenant setup supports flexible Redis deployment with:

1. **Upstash Redis** (Production - Managed, serverless Redis)
2. **Local Redis** (Development - Docker-based Redis)
3. **Redis Insights** (Optional debugging tool)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      AgentAPI Service                        │
│  ┌────────────────────────────────────────────────────┐    │
│  │  • Session Management (Redis-backed)                │    │
│  │  • Rate Limiting (Redis-backed)                     │    │
│  │  • Circuit Breaker (Redis-backed)                   │    │
│  │  • Caching Layer (Redis-backed)                     │    │
│  └────────────────────────────────────────────────────┘    │
│                          │                                   │
│                          ▼                                   │
│              ┌─────────────────────┐                        │
│              │  Redis Connection   │                        │
│              │  (Native or REST)   │                        │
│              └─────────────────────┘                        │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴──────────────┐
         │                            │
         ▼                            ▼
┌──────────────────┐        ┌──────────────────┐
│  Upstash Redis   │   OR   │   Local Redis    │
│  (Production)    │        │  (Development)   │
│                  │        │                  │
│ • Serverless     │        │ • Docker Image   │
│ • Auto-scaling   │        │ • redis:7-alpine │
│ • High Avail.    │        │ • Local Storage  │
│ • REST + Native  │        │ • Port 6379      │
└──────────────────┘        └─────────┬────────┘
                                      │
                                      ▼
                            ┌──────────────────┐
                            │ Redis Insights   │
                            │ (Optional GUI)   │
                            │ Port 8001        │
                            └──────────────────┘
```

## Services

### 1. AgentAPI Service

**Redis Environment Variables:**
```yaml
# Core Redis Configuration
REDIS_ENABLE=true                    # Enable/disable Redis
REDIS_PROTOCOL=native                # 'native' or 'rest'

# Upstash Configuration (Production)
UPSTASH_REDIS_REST_URL              # REST API endpoint
UPSTASH_REDIS_REST_TOKEN            # REST API token
UPSTASH_REDIS_URL                   # Native connection string

# Local Redis Configuration (Development)
REDIS_URL=redis://redis:6379        # Local connection
REDIS_MAX_POOL_SIZE=10              # Connection pool
REDIS_CONNECTION_TIMEOUT=5s         # Timeout

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10

# Circuit Breaker
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_SUCCESS_THRESHOLD=2
CIRCUIT_BREAKER_TIMEOUT=30s

# Session Storage
SESSION_STORAGE=redis
SESSION_TTL=3600s
SESSION_CLEANUP_INTERVAL=300s
TOKEN_ENCRYPTION_KEY=<32-byte-hex>
```

**Health Check:**
- Endpoint: `http://localhost:3284/status`
- Verifies both application and Redis connectivity
- Interval: 30s
- Timeout: 10s
- Retries: 3

**Dependencies:**
- PostgreSQL (always required)
- Redis (optional, when using local Redis)

### 2. Redis Service (Local Development)

**Configuration:**
```yaml
Image: redis:7-alpine
Port: 6379 (configurable via REDIS_PORT)
Persistence: AOF + RDB snapshots
Memory: 256MB default (configurable)
Policy: allkeys-lru (evicts least recently used)
```

**Persistence Settings:**
- AOF: Append-only file for durability
- RDB Snapshots:
  - Every 900s if 1+ keys changed
  - Every 300s if 10+ keys changed
  - Every 60s if 10000+ keys changed

**Health Check:**
- Command: `redis-cli ping`
- Interval: 10s
- Timeout: 5s
- Retries: 5

**Resource Limits:**
- CPU: 0.25-0.5 cores
- Memory: 256MB-512MB

**Volume:**
- Path: `./data/redis`
- Persistent across restarts

### 3. Redis Insights (Optional)

**Configuration:**
```yaml
Image: redislabs/redisinsight:latest
Port: 8001
Profile: development, debug
```

**Features:**
- Visual Redis browser
- Key/value viewer
- Command execution
- Performance monitoring
- Memory analysis

**Activation:**
```bash
# Start with development profile
docker-compose --profile development -f docker-compose.multitenant.yml up -d

# Or with debug profile
docker-compose --profile debug -f docker-compose.multitenant.yml up -d
```

**Resource Limits:**
- CPU: 0.1-0.25 cores
- Memory: 128MB-256MB

**Volume:**
- Path: `./data/redis-insights`
- Stores RedisInsight configuration

## Deployment Modes

### Production Mode (Upstash Redis)

**When to use:**
- Production deployments
- Serverless/cloud environments
- Need automatic scaling
- Want managed infrastructure
- Require high availability

**Setup:**

1. Create Upstash account at https://console.upstash.com
2. Create new Redis database
3. Copy credentials to .env:
   ```bash
   UPSTASH_REDIS_REST_URL=https://your-instance.upstash.io
   UPSTASH_REDIS_REST_TOKEN=your-token
   UPSTASH_REDIS_URL=rediss://default:pass@your-instance.upstash.io:6379
   ```

4. Comment out local redis service in docker-compose.multitenant.yml:
   ```yaml
   # redis:
   #   image: redis:7-alpine
   #   ...
   ```

5. Remove redis from depends_on:
   ```yaml
   depends_on:
     postgres:
       condition: service_healthy
     # redis:
     #   condition: service_healthy
   ```

6. Start services:
   ```bash
   docker-compose -f docker-compose.multitenant.yml up -d
   ```

**Benefits:**
- No infrastructure management
- Automatic backups
- Global replication
- REST API fallback
- Pay-per-use pricing

### Development Mode (Local Redis)

**When to use:**
- Local development
- Testing
- Offline work
- CI/CD pipelines
- Learning/experimentation

**Setup:**

1. Configure .env:
   ```bash
   REDIS_ENABLE=true
   REDIS_URL=redis://redis:6379
   ```

2. Keep redis service uncommented in docker-compose.multitenant.yml

3. Keep redis in depends_on:
   ```yaml
   depends_on:
     postgres:
       condition: service_healthy
     redis:
       condition: service_healthy
   ```

4. Start services:
   ```bash
   docker-compose -f docker-compose.multitenant.yml up -d
   ```

**Benefits:**
- No external dependencies
- Fast local access
- Full control
- No cost
- Works offline

## Data Flow

### Session Storage Flow
```
User Login
    ↓
Generate Session Token
    ↓
Encrypt Token (TOKEN_ENCRYPTION_KEY)
    ↓
Store in Redis (key: session:{token})
    ↓
Set TTL (SESSION_TTL)
    ↓
Return to Client
    ↓
Subsequent Requests → Verify Token → Extend TTL
    ↓
Logout or TTL Expires → Delete from Redis
```

### Rate Limiting Flow
```
Incoming Request
    ↓
Extract User/IP Identifier
    ↓
Check Redis Counter (key: ratelimit:{identifier})
    ↓
Counter Exists?
    ├─ Yes → Increment Counter
    │         ↓
    │    Counter > Limit?
    │         ├─ Yes → Reject (429 Too Many Requests)
    │         └─ No → Allow Request
    └─ No → Create Counter with TTL=1 minute
              ↓
         Allow Request
```

### Circuit Breaker Flow
```
External Service Call
    ↓
Check Circuit State (Redis key: circuit:{service})
    ↓
State = OPEN?
    ├─ Yes → Fast Fail (Return Error)
    └─ No → Attempt Call
              ↓
         Success?
              ├─ Yes → Record Success
              │         ↓
              │    Increment Success Counter
              │         ↓
              │    Counter > Threshold?
              │         └─ Yes → Close Circuit
              └─ No → Record Failure
                        ↓
                   Increment Failure Counter
                        ↓
                   Counter > Threshold?
                        └─ Yes → Open Circuit (Set TTL)
```

## Volume Management

### Local Redis Data
```yaml
redis_data:
  driver: local
  device: ./data/redis
```

**Purpose:** Persist Redis data across container restarts

**Backup:**
```bash
# Create backup
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli SAVE
tar -czf redis-backup-$(date +%Y%m%d).tar.gz ./data/redis

# Restore backup
docker-compose -f docker-compose.multitenant.yml stop redis
tar -xzf redis-backup-YYYYMMDD.tar.gz
docker-compose -f docker-compose.multitenant.yml start redis
```

### Redis Insights Data
```yaml
redis_insights_data:
  driver: local
  device: ./data/redis-insights
```

**Purpose:** Store RedisInsight configuration and preferences

## Network Configuration

All services communicate on the `agentapi-network` bridge network:

```yaml
networks:
  agentapi-network:
    driver: bridge
    subnet: 172.20.0.0/16
```

**Service Communication:**
- AgentAPI → Redis: `redis:6379` (internal DNS)
- AgentAPI → Upstash: `https://your-instance.upstash.io` (external)
- Redis Insights → Redis: `redis:6379` (internal DNS)

## Monitoring

### Health Checks

**AgentAPI:**
```bash
docker-compose -f docker-compose.multitenant.yml exec agentapi wget -O- http://localhost:3284/status
```

**Local Redis:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping
```

**Upstash Redis:**
```bash
curl -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN" \
     $UPSTASH_REDIS_REST_URL/ping
```

### Logs

**AgentAPI logs:**
```bash
docker-compose -f docker-compose.multitenant.yml logs -f agentapi
```

**Redis logs:**
```bash
docker-compose -f docker-compose.multitenant.yml logs -f redis
```

**Filter Redis-related logs:**
```bash
docker-compose -f docker-compose.multitenant.yml logs agentapi | grep -i redis
```

### Metrics

**Redis Info:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info
```

**Memory Usage:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info memory
```

**Connection Stats:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info clients
```

**Key Count:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli dbsize
```

## Security

### Local Redis

**Password Protection:**
```yaml
command: >
  redis-server
  --requirepass ${REDIS_PASSWORD}
```

Update .env:
```bash
REDIS_PASSWORD=your-secure-password
REDIS_URL=redis://:your-secure-password@redis:6379
```

**Network Isolation:**
- Redis only accessible within docker network
- Not exposed to host (remove ports mapping)

**Persistence Encryption:**
- Encrypt volume at rest (OS-level encryption)
- Use encrypted backups

### Upstash Redis

**TLS/SSL:**
- Always uses `rediss://` (TLS encrypted)
- REST API uses HTTPS

**Authentication:**
- REST token for API access
- Password for native connections

**Best Practices:**
- Rotate credentials regularly
- Use environment variables
- Never commit secrets
- Enable IP restrictions (Upstash console)

### Token Encryption

**Generate secure key:**
```bash
openssl rand -hex 32
```

**Store in .env:**
```bash
TOKEN_ENCRYPTION_KEY=<generated-key>
```

**Rotate periodically:**
- Generate new key
- Decrypt old tokens
- Re-encrypt with new key
- Update environment variable

## Troubleshooting

### Common Issues

**1. Redis connection refused**

Local Redis:
```bash
# Check if Redis is running
docker-compose -f docker-compose.multitenant.yml ps redis

# Check health
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping

# View logs
docker-compose -f docker-compose.multitenant.yml logs redis
```

Upstash:
```bash
# Test REST endpoint
curl -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN" \
     $UPSTASH_REDIS_REST_URL/ping

# Check credentials
echo $UPSTASH_REDIS_REST_URL
echo $UPSTASH_REDIS_REST_TOKEN
```

**2. Out of memory**

Local Redis:
```bash
# Check memory usage
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info memory

# Increase max memory
# Edit docker-compose.multitenant.yml:
REDIS_MAX_MEMORY=512mb
```

Upstash:
- Upgrade plan or optimize key usage

**3. Slow performance**

Local Redis:
```bash
# Check slow log
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli slowlog get 10

# Monitor commands
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli monitor
```

Upstash:
- Check network latency
- Consider region change
- Use REST protocol fallback

**4. Data loss**

Local Redis:
```bash
# Check persistence
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli config get save
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli config get appendonly

# Manual save
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli SAVE
```

## References

- [Redis Configuration Guide](./REDIS_CONFIGURATION.md) - Full configuration details
- [Redis Quick Start](./REDIS_QUICKSTART.md) - Fast setup guide
- [Upstash Documentation](https://docs.upstash.com/redis)
- [Redis Documentation](https://redis.io/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

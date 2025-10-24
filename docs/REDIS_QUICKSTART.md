# Redis Quick Start Guide

Get Redis up and running with AgentAPI in 5 minutes.

## Choose Your Deployment

### Option A: Local Redis (Development)

Perfect for local development and testing.

**1. Update .env file:**
```bash
# Enable Redis
REDIS_ENABLE=true
REDIS_PROTOCOL=native

# Local Redis (uses docker-compose redis service)
REDIS_URL=redis://redis:6379
```

**2. Start services:**
```bash
docker-compose -f docker-compose.multitenant.yml up -d
```

**3. Verify Redis is running:**
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping
```

Expected output: `PONG`

**Done!** Your application is now using local Redis.

---

### Option B: Upstash Redis (Production)

Perfect for production, serverless, and cloud deployments.

**1. Create Upstash Redis instance:**
- Go to [Upstash Console](https://console.upstash.com)
- Click "Create Database"
- Choose a name and region
- Copy credentials from the console

**2. Update .env file:**
```bash
# Enable Redis
REDIS_ENABLE=true
REDIS_PROTOCOL=native

# Upstash Redis credentials (from Upstash console)
UPSTASH_REDIS_REST_URL=https://your-instance.upstash.io
UPSTASH_REDIS_REST_TOKEN=AYseAAIncD...
UPSTASH_REDIS_URL=rediss://default:password@your-instance.upstash.io:6379
```

**3. Update docker-compose.multitenant.yml:**

Comment out the local Redis service:
```yaml
# Comment out or remove this entire section
# redis:
#   image: redis:7-alpine
#   ...
```

Update the agentapi service dependencies:
```yaml
depends_on:
  postgres:
    condition: service_healthy
  # Remove or comment out redis dependency
  # redis:
  #   condition: service_healthy
```

**4. Start services:**
```bash
docker-compose -f docker-compose.multitenant.yml up -d
```

**Done!** Your application is now using Upstash Redis.

---

## Optional: Enable Redis Insights

Redis Insights provides a GUI for debugging and monitoring Redis.

**1. Start with development profile:**
```bash
docker-compose --profile development -f docker-compose.multitenant.yml up -d
```

**2. Access Redis Insights:**
- Open browser: http://localhost:8001
- Accept terms and conditions
- Add database:
  - Host: `redis`
  - Port: `6379`
  - Name: `AgentAPI Local Redis`

**3. Explore your data:**
- View cached data
- Debug sessions
- Monitor rate limits
- Inspect circuit breaker states

---

## Environment Variables Reference

### Minimal Configuration (Local Redis)
```bash
REDIS_ENABLE=true
REDIS_URL=redis://redis:6379
```

### Minimal Configuration (Upstash Redis)
```bash
REDIS_ENABLE=true
UPSTASH_REDIS_REST_URL=https://your-instance.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-token
UPSTASH_REDIS_URL=rediss://default:password@your-instance.upstash.io:6379
```

### Full Configuration (Optional)
```bash
# Redis Core
REDIS_ENABLE=true
REDIS_PROTOCOL=native
REDIS_MAX_POOL_SIZE=10
REDIS_CONNECTION_TIMEOUT=5s

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
TOKEN_ENCRYPTION_KEY=$(openssl rand -hex 32)
```

---

## Verification

### Check Redis Connection
```bash
# For local Redis
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping

# For Upstash Redis
curl -H "Authorization: Bearer YOUR_REST_TOKEN" \
     https://your-instance.upstash.io/ping
```

### View Application Logs
```bash
docker-compose -f docker-compose.multitenant.yml logs agentapi | grep -i redis
```

### Check Redis Health
```bash
docker-compose -f docker-compose.multitenant.yml ps
```

All services should show "Up (healthy)".

---

## Troubleshooting

### Redis not connecting?

**Local Redis:**
```bash
# Check if Redis is running
docker-compose -f docker-compose.multitenant.yml ps redis

# View Redis logs
docker-compose -f docker-compose.multitenant.yml logs redis

# Restart Redis
docker-compose -f docker-compose.multitenant.yml restart redis
```

**Upstash Redis:**
```bash
# Verify credentials in .env
cat .env | grep UPSTASH

# Test REST endpoint
curl -H "Authorization: Bearer YOUR_REST_TOKEN" \
     https://your-instance.upstash.io/ping

# Try REST protocol fallback
REDIS_PROTOCOL=rest
```

### Application not using Redis?

```bash
# Check environment variables
docker-compose -f docker-compose.multitenant.yml exec agentapi env | grep REDIS

# Verify REDIS_ENABLE is true
# Restart application
docker-compose -f docker-compose.multitenant.yml restart agentapi
```

---

## Next Steps

- Read [Full Redis Configuration Guide](./REDIS_CONFIGURATION.md)
- Configure rate limiting and circuit breaker settings
- Set up monitoring and alerts
- Generate secure TOKEN_ENCRYPTION_KEY for production
- Configure session TTL based on your requirements

---

## Common Commands

```bash
# Start services
docker-compose -f docker-compose.multitenant.yml up -d

# Start with Redis Insights
docker-compose --profile development -f docker-compose.multitenant.yml up -d

# Stop services
docker-compose -f docker-compose.multitenant.yml down

# View logs
docker-compose -f docker-compose.multitenant.yml logs -f

# Restart a service
docker-compose -f docker-compose.multitenant.yml restart agentapi

# Connect to Redis CLI (local only)
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli

# Generate encryption key
openssl rand -hex 32
```

---

## Support

- Full documentation: [REDIS_CONFIGURATION.md](./REDIS_CONFIGURATION.md)
- Upstash docs: https://docs.upstash.com/redis
- Redis docs: https://redis.io/docs/

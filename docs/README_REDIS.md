# Redis Documentation Index

Welcome to the AgentAPI Redis documentation. This directory contains comprehensive guides for configuring and using Redis with AgentAPI.

## Quick Navigation

### 🚀 Getting Started
**Start here if you're new to Redis with AgentAPI**

- **[Redis Quick Start Guide](REDIS_QUICKSTART.md)** (5 min read)
  - Fast setup for development or production
  - Choose between Upstash or local Redis
  - Verification and testing steps

### 📚 Comprehensive Guides

- **[Redis Configuration Guide](REDIS_CONFIGURATION.md)** (15 min read)
  - Complete configuration reference
  - Environment variables documentation
  - Features and use cases
  - Troubleshooting guide
  - Best practices

- **[Docker Redis Setup Guide](DOCKER_REDIS_SETUP.md)** (20 min read)
  - Technical deep dive
  - Architecture diagrams
  - Service configurations
  - Data flows and patterns
  - Security considerations

- **[Redis Architecture Diagram](redis-architecture.txt)** (Visual reference)
  - ASCII architecture diagrams
  - Deployment modes
  - Data flow visualizations
  - Resource allocation

## Documentation Map

```
Choose your path based on your needs:

┌─────────────────────────────────────────────────────┐
│  I want to get started quickly                      │
│  ↓                                                   │
│  Read: REDIS_QUICKSTART.md                          │
│  Time: 5 minutes                                    │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  I need to understand all configuration options     │
│  ↓                                                   │
│  Read: REDIS_CONFIGURATION.md                       │
│  Time: 15 minutes                                   │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  I want to understand the architecture              │
│  ↓                                                   │
│  Read: DOCKER_REDIS_SETUP.md + redis-architecture.txt│
│  Time: 20 minutes                                   │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  I need to troubleshoot an issue                    │
│  ↓                                                   │
│  Read: REDIS_CONFIGURATION.md → Troubleshooting    │
│  Time: 5 minutes                                    │
└─────────────────────────────────────────────────────┘
```

## Document Overview

### REDIS_QUICKSTART.md
**Purpose:** Get Redis running in 5 minutes

**Contents:**
- Quick setup for local Redis
- Quick setup for Upstash Redis
- Redis Insights setup
- Basic verification steps
- Common commands

**Best for:**
- First-time users
- Quick testing
- Rapid prototyping
- Development setup

---

### REDIS_CONFIGURATION.md
**Purpose:** Complete Redis configuration reference

**Contents:**
- Deployment options comparison
- Detailed setup instructions
- Environment variables (full reference)
- Docker Compose examples
- Health checks
- Redis Insights
- Features using Redis
- Troubleshooting
- Best practices
- Quick reference

**Best for:**
- Production deployment
- Understanding all options
- Optimization
- Troubleshooting
- Reference documentation

---

### DOCKER_REDIS_SETUP.md
**Purpose:** Technical deep dive into Docker Redis setup

**Contents:**
- Architecture overview
- Service descriptions
- Deployment modes
- Data flow diagrams
- Volume management
- Network configuration
- Monitoring and metrics
- Security considerations
- Detailed troubleshooting

**Best for:**
- DevOps engineers
- System administrators
- Understanding internals
- Performance optimization
- Security hardening

---

### redis-architecture.txt
**Purpose:** Visual reference for Redis architecture

**Contents:**
- ASCII architecture diagrams
- Production deployment diagram
- Development deployment diagram
- Data flow visualizations
- Service dependencies
- Environment variable flows
- Network architecture
- Volume structure
- Health check flows
- Resource allocation tables

**Best for:**
- Visual learners
- Architecture planning
- Team presentations
- Understanding data flows
- Capacity planning

## Common Use Cases

### Use Case 1: Local Development Setup
**Goal:** Set up Redis for local development

**Steps:**
1. Read: [REDIS_QUICKSTART.md](REDIS_QUICKSTART.md) - Option A
2. Configure .env with local Redis settings
3. Start: `docker-compose -f docker-compose.multitenant.yml up -d`
4. Verify: `docker-compose exec redis redis-cli ping`

**Time:** 5 minutes

---

### Use Case 2: Production Deployment with Upstash
**Goal:** Deploy to production with managed Redis

**Steps:**
1. Read: [REDIS_CONFIGURATION.md](REDIS_CONFIGURATION.md) - Option 1
2. Create Upstash account and database
3. Configure .env with Upstash credentials
4. Update docker-compose.multitenant.yml
5. Deploy and verify

**Time:** 15 minutes

---

### Use Case 3: Understanding Architecture
**Goal:** Learn how Redis integrates with AgentAPI

**Steps:**
1. View: [redis-architecture.txt](redis-architecture.txt)
2. Read: [DOCKER_REDIS_SETUP.md](DOCKER_REDIS_SETUP.md)
3. Review data flow diagrams
4. Understand service dependencies

**Time:** 30 minutes

---

### Use Case 4: Troubleshooting Connection Issues
**Goal:** Fix Redis connection problems

**Steps:**
1. Check: [REDIS_QUICKSTART.md](REDIS_QUICKSTART.md) - Troubleshooting section
2. Detailed help: [REDIS_CONFIGURATION.md](REDIS_CONFIGURATION.md) - Troubleshooting
3. Run verification commands
4. Check logs

**Time:** 10 minutes

---

### Use Case 5: Optimizing Performance
**Goal:** Tune Redis for better performance

**Steps:**
1. Read: [REDIS_CONFIGURATION.md](REDIS_CONFIGURATION.md) - Best Practices
2. Read: [DOCKER_REDIS_SETUP.md](DOCKER_REDIS_SETUP.md) - Monitoring
3. Adjust connection pool size
4. Monitor metrics
5. Optimize resource limits

**Time:** 20 minutes

## Key Topics by Document

### Session Management
- **REDIS_CONFIGURATION.md** - Configuration and setup
- **DOCKER_REDIS_SETUP.md** - Session data flow
- **redis-architecture.txt** - Session management diagram

### Rate Limiting
- **REDIS_CONFIGURATION.md** - Rate limiting configuration
- **DOCKER_REDIS_SETUP.md** - Rate limiting flow
- **redis-architecture.txt** - Rate limiting diagram

### Circuit Breaker
- **REDIS_CONFIGURATION.md** - Circuit breaker setup
- **DOCKER_REDIS_SETUP.md** - Circuit breaker flow
- **redis-architecture.txt** - Circuit breaker diagram

### Caching
- **REDIS_CONFIGURATION.md** - Cache configuration
- **DOCKER_REDIS_SETUP.md** - Cache flow
- **redis-architecture.txt** - Caching diagram

### Security
- **REDIS_CONFIGURATION.md** - Security best practices
- **DOCKER_REDIS_SETUP.md** - Security section
- All documents cover security aspects

### Monitoring
- **REDIS_CONFIGURATION.md** - Basic monitoring
- **DOCKER_REDIS_SETUP.md** - Detailed monitoring
- **redis-architecture.txt** - Health check flows

## Environment Variables

### Quick Reference
For a complete list of environment variables, see:
- [REDIS_CONFIGURATION.md](REDIS_CONFIGURATION.md) - Environment Variables section

### Essential Variables

**Upstash (Production):**
```bash
UPSTASH_REDIS_REST_URL=https://your-instance.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-token
UPSTASH_REDIS_URL=rediss://default:password@host:6379
```

**Local (Development):**
```bash
REDIS_URL=redis://redis:6379
```

**Common:**
```bash
REDIS_ENABLE=true
REDIS_PROTOCOL=native
RATE_LIMIT_ENABLED=true
CIRCUIT_BREAKER_ENABLED=true
SESSION_STORAGE=redis
```

## Docker Compose Profiles

### Default Profile
Includes: AgentAPI, PostgreSQL, Redis

**Start:**
```bash
docker-compose -f docker-compose.multitenant.yml up -d
```

### Development Profile
Includes: All default + Redis Insights

**Start:**
```bash
docker-compose --profile development -f docker-compose.multitenant.yml up -d
```

**Access Redis Insights:** http://localhost:8001

## Quick Commands

### Start Services
```bash
# Default (with local Redis)
docker-compose -f docker-compose.multitenant.yml up -d

# With Redis Insights
docker-compose --profile development -f docker-compose.multitenant.yml up -d
```

### Verify Redis
```bash
# Local Redis
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli ping

# Upstash Redis
curl -H "Authorization: Bearer $UPSTASH_REDIS_REST_TOKEN" \
     $UPSTASH_REDIS_REST_URL/ping
```

### View Logs
```bash
# All services
docker-compose -f docker-compose.multitenant.yml logs -f

# Redis only
docker-compose -f docker-compose.multitenant.yml logs -f redis

# Filter Redis-related logs
docker-compose -f docker-compose.multitenant.yml logs agentapi | grep -i redis
```

### Connect to Redis CLI
```bash
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli
```

### Monitor Redis
```bash
# Real-time monitoring
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli monitor

# Memory info
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli info memory

# Key count
docker-compose -f docker-compose.multitenant.yml exec redis redis-cli dbsize
```

## Support

### Documentation Issues
If you find errors or missing information in the documentation:
1. Check all related documents
2. Review the main project README
3. Open an issue in the repository

### Redis Issues
For Redis-specific problems:
1. Check [REDIS_CONFIGURATION.md](REDIS_CONFIGURATION.md) - Troubleshooting
2. Review logs: `docker-compose logs redis`
3. Verify configuration: `docker-compose config`
4. Test connection manually

### External Resources
- [Upstash Documentation](https://docs.upstash.com/redis)
- [Redis Documentation](https://redis.io/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

## Document Versions

All documents in this directory are current as of **2025-10-23**.

### Version History
- **v1.0** (2025-10-23): Initial Redis integration documentation
  - REDIS_QUICKSTART.md
  - REDIS_CONFIGURATION.md
  - DOCKER_REDIS_SETUP.md
  - redis-architecture.txt

## Contributing

When updating Redis documentation:
1. Update all relevant documents
2. Maintain consistency across files
3. Update this README if adding new documents
4. Test all commands and examples
5. Update version history

## License

This documentation is part of the AgentAPI project.

## Related Documentation

### Main Project Documentation
- **../README.md** - Main project README
- **../docker-compose.multitenant.yml** - Docker Compose configuration
- **../.env.example** - Environment variables reference

### Summary Documents
- **../REDIS_INTEGRATION_SUMMARY.md** - Integration summary
- **../DOCKER_REDIS_UPDATE_COMPLETE.md** - Update completion summary

---

**Last Updated:** 2025-10-23

**Total Documentation Size:** 51+ KB

**Documents:** 4 core documents + this index

**Coverage:** Complete Redis integration for AgentAPI

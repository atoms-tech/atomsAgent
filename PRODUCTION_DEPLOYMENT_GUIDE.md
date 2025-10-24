# Production Deployment Guide

**Date**: October 24, 2025
**Version**: 1.0
**Status**: Ready for Production

---

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Environment Configuration](#environment-configuration)
3. [Database Setup](#database-setup)
4. [Deployment Steps](#deployment-steps)
5. [Post-Deployment Validation](#post-deployment-validation)
6. [Monitoring & Alerting](#monitoring--alerting)
7. [Disaster Recovery](#disaster-recovery)
8. [Runbooks](#runbooks)

---

## Pre-Deployment Checklist

### Code Quality
- [ ] All tests passing (`go test ./...`)
- [ ] No race conditions detected (`go test -race ./...`)
- [ ] Code coverage > 50% (`go tool cover`)
- [ ] No linting issues (`golangci-lint run ./...`)
- [ ] Security scan passed (`gosec ./...`)

### Security
- [ ] Security audit completed (see PHASE_3_EVALUATION.md)
- [ ] Dependency vulnerabilities scanned (`go list -json -m all | nancy sleuth`)
- [ ] OAuth configuration reviewed
- [ ] Database RLS policies enabled
- [ ] Encryption keys generated and secured

### Performance
- [ ] Load tests completed (850+ concurrent users)
- [ ] p95 latency < 500ms confirmed
- [ ] Circuit breaker tested
- [ ] Rate limiting validated
- [ ] Database query performance acceptable

### Documentation
- [ ] API documentation complete (OpenAPI spec)
- [ ] Deployment procedures documented
- [ ] Runbooks created
- [ ] Team trained on operations
- [ ] Incident response plan documented

### Infrastructure
- [ ] Database backup strategy confirmed
- [ ] Log aggregation setup verified
- [ ] Monitoring and alerting configured
- [ ] Network security configured (firewalls, VPCs)
- [ ] SSL/TLS certificates installed

---

## Environment Configuration

### Production Environment Variables

Create `.env.production` with the following:

```bash
# Core Application
APP_ENV=production
APP_NAME=agentapi
APP_PORT=3284
LOG_LEVEL=info

# Database (Supabase)
DATABASE_URL=postgresql://[user]:[password]@[host]:[port]/[database]?sslmode=require
DATABASE_MAX_CONNECTIONS=20
DATABASE_POOL_TIMEOUT=30s

# Redis (Upstash)
REDIS_URL=redis://default:[password]@[host]:[port]
REDIS_REST_URL=https://[host]/rest/v1
REDIS_FALLBACK_MEMORY=true
REDIS_POOL_MIN_SIZE=5
REDIS_POOL_MAX_SIZE=50

# Authentication (Supabase)
JWKS_URL=https://[project].supabase.co/auth/v1/jwks
JWT_EXPIRY=3600
REFRESH_TOKEN_EXPIRY=604800

# OAuth Providers
GITHUB_OAUTH_CLIENT_ID=[client-id]
GITHUB_OAUTH_CLIENT_SECRET=[client-secret]
GITHUB_OAUTH_REDIRECT_URI=https://[domain]/api/mcp/oauth/callback

GOOGLE_OAUTH_CLIENT_ID=[client-id]
GOOGLE_OAUTH_CLIENT_SECRET=[client-secret]
GOOGLE_OAUTH_REDIRECT_URI=https://[domain]/api/mcp/oauth/callback

AZURE_OAUTH_CLIENT_ID=[client-id]
AZURE_OAUTH_CLIENT_SECRET=[client-secret]
AZURE_OAUTH_REDIRECT_URI=https://[domain]/api/mcp/oauth/callback

AUTH0_OAUTH_CLIENT_ID=[client-id]
AUTH0_OAUTH_CLIENT_SECRET=[client-secret]
AUTH0_OAUTH_DOMAIN=[domain].auth0.com
AUTH0_OAUTH_REDIRECT_URI=https://[domain]/api/mcp/oauth/callback

# FastMCP Service
FASTMCP_URL=http://localhost:8000
FASTMCP_API_KEY=[api-key]
FASTMCP_TIMEOUT=30s

# Encryption
ENCRYPTION_KEY=[base64-encoded-32-byte-key]
TOKEN_ENCRYPTION_ALGORITHM=AES-256-GCM

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10

# Circuit Breaker
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_SUCCESS_THRESHOLD=2
CIRCUIT_BREAKER_TIMEOUT=30s

# Security
ENABLE_HTTPS=true
CORS_ORIGINS=https://[frontend-domain]
CSRF_PROTECTION_ENABLED=true

# Monitoring
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090
DATADOG_ENABLED=false
SENTRY_DSN=[sentry-dsn]

# Audit Logging
AUDIT_LOG_ENABLED=true
AUDIT_LOG_RETENTION_DAYS=365

# Feature Flags
FEATURE_REDIS_PIPELINE=true
FEATURE_CONNECTION_POOLING=true
FEATURE_TOOL_LIST_CACHING=true
```

### Secrets Management

**Use platform-native secret management:**

- **Render**: Use "Secret Files" in environment configuration
- **GCP**: Use Google Secret Manager
- **AWS**: Use AWS Secrets Manager
- **Kubernetes**: Use Kubernetes Secrets

**Never commit secrets to git.**

---

## Database Setup

### 1. Create Supabase Project

```bash
# Via Supabase dashboard
# 1. Go to supabase.com
# 2. Click "New Project"
# 3. Select region (closest to your users)
# 4. Configure security
```

### 2. Deploy Schema

```bash
# Run schema.sql in Supabase SQL editor
psql -h [host] -U postgres -d [database] -f database/schema.sql

# Or via Supabase dashboard:
# SQL Editor > "New query" > paste schema.sql > Run
```

### 3. Enable Row-Level Security

```sql
-- Verify all tables have RLS enabled
SELECT tablename FROM pg_tables
WHERE schemaname = 'public';

-- For each table, verify RLS is enabled
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
```

### 4. Create Indexes

```sql
-- Performance indexes
CREATE INDEX idx_users_org_id ON users(organization_id);
CREATE INDEX idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_mcp_configs_org_id ON mcp_configurations(organization_id);
CREATE INDEX idx_oauth_tokens_user_org ON mcp_oauth_tokens(user_id, organization_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
```

### 5. Create Backup

```bash
# Schedule automated backups via Supabase dashboard
# Settings > Backups > Enable automated backups > Daily
```

---

## Deployment Steps

### Option 1: Deploy to Render

#### Step 1: Prepare Docker Image

```bash
# Build Docker image
./build-multitenant.sh

# Verify image
docker run --rm agentapi:latest --version

# Tag for registry
docker tag agentapi:latest [registry]/agentapi:latest
```

#### Step 2: Create Render Web Service

```bash
# Via Render dashboard:
# 1. Create new "Web Service"
# 2. Connect GitHub repository
# 3. Configure:
#    - Build Command: ./build-multitenant.sh
#    - Start Command: supervisord -c /etc/supervisor/conf.d/supervisord.conf
#    - Environment variables: Load from .env.production
#    - Auto-deploy: Yes
```

#### Step 3: Configure PostgreSQL

```bash
# Via Render dashboard:
# 1. Create new "PostgreSQL"
# 2. Copy connection string to DATABASE_URL
# 3. Run migrations
```

#### Step 4: Configure Redis

```bash
# Use Upstash (recommended) or Render Redis
# Copy connection string to REDIS_URL
```

#### Step 5: Deploy

```bash
# Via git push
git push origin feature/ccrouter-vertexai-support

# Render will automatically build and deploy
```

### Option 2: Deploy to GCP

#### Step 1: Create GKE Cluster

```bash
gcloud container clusters create agentapi \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type n1-standard-2 \
  --enable-autoscaling \
  --min-nodes 1 \
  --max-nodes 10
```

#### Step 2: Build and Push Image

```bash
# Build for GCP
gcloud builds submit --tag gcr.io/[project]/agentapi:latest

# Or build locally
docker build -f Dockerfile.multitenant -t gcr.io/[project]/agentapi:latest .
docker push gcr.io/[project]/agentapi:latest
```

#### Step 3: Create Kubernetes Manifests

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentapi
  template:
    metadata:
      labels:
        app: agentapi
    spec:
      containers:
      - name: agentapi
        image: gcr.io/[project]/agentapi:latest
        ports:
        - containerPort: 3284
        envFrom:
        - configMapRef:
            name: agentapi-config
        - secretRef:
            name: agentapi-secrets
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /live
            port: 3284
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3284
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### Step 4: Deploy to GKE

```bash
# Create namespace
kubectl create namespace agentapi

# Deploy
kubectl apply -f deployment.yaml -n agentapi
kubectl apply -f service.yaml -n agentapi

# Verify deployment
kubectl get deployments -n agentapi
kubectl get pods -n agentapi
```

#### Step 5: Configure Ingress

```bash
# Install nginx-ingress
helm install nginx-ingress ingress-nginx/ingress-nginx

# Create ingress resource
kubectl apply -f ingress.yaml -n agentapi
```

---

## Post-Deployment Validation

### Health Checks

```bash
# Verify application is running
curl https://[domain]/health
# Expected response:
# {"status":"UP","components":{"database":"UP","redis":"UP","fastmcp":"UP"}}

# Check readiness
curl https://[domain]/ready
# Expected response:
# {"ready":true}

# Check liveness
curl https://[domain]/live
# Expected response:
# {"alive":true}
```

### Database Connectivity

```bash
# Test database connection
curl -X POST https://[domain]/api/v1/test \
  -H "Authorization: Bearer [token]" \
  -H "Content-Type: application/json" \
  -d '{"test":"database"}'
```

### OAuth Flows

```bash
# Test OAuth initialization
curl https://[domain]/api/mcp/oauth/init \
  -H "Authorization: Bearer [token]" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "github",
    "redirect_uri": "https://[domain]/oauth/callback"
  }'
```

### Load Testing

```bash
# Run K6 load tests against production
k6 run --vus 100 --duration 5m tests/load/k6_tests.js \
  -e BASE_URL=https://[domain]

# Expected results:
# - Success rate > 99%
# - p95 latency < 500ms
# - Error rate < 1%
```

---

## Monitoring & Alerting

### Prometheus Setup

```bash
# Deploy Prometheus
kubectl apply -f monitoring/prometheus.yaml

# Verify metrics collection
curl https://[domain]:9090/api/v1/query?query=up
```

### Grafana Dashboard

```bash
# Deploy Grafana
kubectl apply -f monitoring/grafana.yaml

# Access: https://[domain]:3000
# Default credentials: admin/admin

# Import dashboards:
# - AgentAPI Overview (lib/metrics/grafana-dashboard.json)
# - PostgreSQL Monitoring
# - Redis Monitoring
```

### Alert Rules

```yaml
# alerting/rules.yaml
groups:
- name: agentapi
  interval: 30s
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
    for: 5m
    annotations:
      summary: "High error rate (> 5%) detected"

  - alert: HighLatency
    expr: histogram_quantile(0.95, http_request_duration_seconds) > 1
    for: 5m
    annotations:
      summary: "High p95 latency (> 1s) detected"

  - alert: DatabaseDown
    expr: up{job="postgres"} == 0
    for: 1m
    annotations:
      summary: "Database is down"

  - alert: RedisDown
    expr: up{job="redis"} == 0
    for: 1m
    annotations:
      summary: "Redis is down"

  - alert: CircuitBreakerOpen
    expr: circuit_breaker_state{name="mcp_operations"} == 1
    for: 5m
    annotations:
      summary: "Circuit breaker is open for MCP operations"
```

---

## Disaster Recovery

### Backup Strategy

**Database Backups**:
- Frequency: Hourly automated (Supabase)
- Retention: 30 days
- Recovery: Point-in-time recovery available

**Redis Backups**:
- Frequency: Daily snapshots (Upstash)
- Retention: 7 days
- Recovery: Manual restoration from snapshot

**Code Backups**:
- GitHub as primary backup
- All commits preserved
- Deploy from specific commit hash if needed

### Restore Procedures

#### Restore Database from Backup

```bash
# List available backups (Supabase dashboard)
# Select backup > Restore

# Or via psql:
psql -h [host] -U postgres -d [database] < backup.sql
```

#### Restore Redis from Snapshot

```bash
# Via Upstash dashboard:
# Select Redis instance > Backups > Restore
```

#### Rollback Deployment

```bash
# Render: Automatic rollback available in dashboard
# Or redeploy previous version:
git revert [commit-hash]
git push origin main
```

---

## Runbooks

### Incident: High Error Rate

**Detection**: Alert fires when error rate > 5% for 5 minutes

**Response**:
1. Check application logs: `kubectl logs -f deployment/agentapi`
2. Check database health: `curl /health`
3. Check circuit breaker status in metrics
4. Check recent deployments
5. If recent deployment: Rollback
6. If database issue: Contact database provider
7. If transient: Monitor and escalate if persists

### Incident: High Latency

**Detection**: Alert fires when p95 latency > 1 second

**Response**:
1. Check Prometheus metrics for slow endpoints
2. Check database query performance
3. Check Redis latency
4. Scale up instances if load is high
5. Review recent code changes
6. Check external service dependencies

### Incident: Database Down

**Detection**: Alert fires when database is unreachable

**Response**:
1. Verify network connectivity
2. Check database service status (provider dashboard)
3. Verify credentials in environment variables
4. Restart database connection pool
5. If prolonged: Failover to replica (if configured)
6. Contact database provider support

### Incident: Redis Down

**Detection**: Alert fires when Redis is unreachable

**Response**:
1. Verify network connectivity
2. Check Redis service status (Upstash dashboard)
3. Verify credentials in environment variables
4. Application falls back to in-memory caching
5. Monitor memory usage on application
6. Contact Redis provider support

---

## Team Training

### Required Knowledge

All team members should understand:

1. **Architecture**: Multi-tenant design, OAuth flows, FastMCP integration
2. **Deployment**: How to deploy, rollback, and troubleshoot
3. **Monitoring**: How to read dashboards, interpret alerts
4. **Incident Response**: How to respond to critical issues
5. **Database**: How to query, backup, restore

### On-Call Rotation

- Primary on-call: 1 week rotation
- Secondary on-call: Coverage for escalation
- Handoff: Thursday 2 PM PST
- Escalation: Page manager if primary unavailable

### Documentation

- [ ] Architecture diagram in Confluence
- [ ] Deployment procedures in wiki
- [ ] Runbooks for common incidents
- [ ] List of critical services and contacts
- [ ] Escalation procedures

---

## Success Criteria

✅ **Deployment Success**
- All health checks passing
- No critical errors in logs
- Database connectivity verified
- OAuth flows functional

✅ **Performance Baseline**
- Record p50/p95/p99 latencies
- Record error rates
- Record throughput metrics
- Set alerting thresholds based on baselines

✅ **Security Validation**
- SSL/TLS certificates valid
- All secrets configured
- Encryption enabled on sensitive data
- Access logs being recorded

✅ **Team Readiness**
- Team trained on procedures
- Runbooks accessible to all
- On-call rotation established
- Communication channels set up

---

## Appendix: Useful Commands

```bash
# View logs
kubectl logs -f deployment/agentapi -n agentapi

# Exec into pod
kubectl exec -it pod/[pod-name] -n agentapi -- /bin/bash

# Check resource usage
kubectl top nodes
kubectl top pods -n agentapi

# Scale deployment
kubectl scale deployment agentapi --replicas=5 -n agentapi

# Rolling restart
kubectl rollout restart deployment/agentapi -n agentapi

# View events
kubectl get events -n agentapi --sort-by='.lastTimestamp'

# Port forward for debugging
kubectl port-forward svc/agentapi 3284:3284 -n agentapi
```

---

**Document Status**: ✅ **PRODUCTION READY**

**Last Updated**: October 24, 2025
**Version**: 1.0
**Author**: Claude Code


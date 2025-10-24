# AgentAPI Environment Configuration Guide

Complete guide to configuring environment variables for AgentAPI, including setup instructions for all dependencies.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Environment Variables Overview](#environment-variables-overview)
3. [Required Dependencies](#required-dependencies)
4. [WorkOS AuthKit Setup](#workos-authkit-setup)
5. [Installing CCRouter](#installing-ccrouter)
6. [Installing Droid](#installing-droid)
7. [Database Setup](#database-setup)
8. [Redis Setup](#redis-setup)
9. [Development Setup](#development-setup)
10. [Production Setup](#production-setup)
11. [Security Best Practices](#security-best-practices)
12. [Troubleshooting](#troubleshooting)
13. [Secret Rotation](#secret-rotation)

---

## Quick Start

### For Development

1. Copy the development template:
   ```bash
   cp .env.development .env
   ```

2. Generate required secrets:
   ```bash
   # Generate encryption keys
   openssl rand -hex 32  # TOKEN_ENCRYPTION_KEY
   openssl rand -base64 32  # JWT_SECRET
   openssl rand -base64 32  # SESSION_SECRET
   ```

3. Add your API keys:
   - Get Anthropic API key from https://console.anthropic.com
   - Get WorkOS credentials from https://dashboard.workos.com

4. Install dependencies (see sections below):
   - PostgreSQL
   - Redis
   - CCRouter (optional)
   - Droid (optional)

5. Start the server:
   ```bash
   agentapi server -- claude
   ```

### For Production

1. Copy the example template:
   ```bash
   cp .env.example .env
   ```

2. Fill in all required values (see [Production Setup](#production-setup))

3. Use a secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.)

4. Never commit `.env` to version control

---

## Environment Variables Overview

### Required Variables

These must be set for AgentAPI to function:

| Variable | Type | Description | How to Get |
|----------|------|-------------|------------|
| `DATABASE_URL` | string | PostgreSQL connection URL | [Database Setup](#database-setup) |
| `REDIS_URL` | string | Redis connection URL | [Redis Setup](#redis-setup) |
| `WORKOS_API_KEY` | string | WorkOS API key for authentication | [WorkOS Setup](#workos-authkit-setup) |
| `WORKOS_CLIENT_ID` | string | WorkOS client identifier | [WorkOS Setup](#workos-authkit-setup) |
| `AUTHKIT_JWKS_URL` | string | JWT verification endpoint | [WorkOS Setup](#workos-authkit-setup) |
| `TOKEN_ENCRYPTION_KEY` | string | 32-byte hex for token encryption | `openssl rand -hex 32` |
| `JWT_SECRET` | string | Secret for JWT signing | `openssl rand -base64 32` |
| `SESSION_SECRET` | string | Secret for session encryption | `openssl rand -base64 32` |

### Conditional Variables

Required depending on which features you use:

| Variable | Required When | Description |
|----------|---------------|-------------|
| `ANTHROPIC_API_KEY` | Using Claude models | Anthropic API access |
| `OPENAI_API_KEY` | Using OpenAI models | OpenAI API access |
| `VERTEX_AI_PROJECT_ID` | Using Google/VertexAI | Google Cloud project ID |
| `CCROUTER_PATH` | Using CCRouter agent | Path to ccr binary |
| `DROID_PATH` | Using Droid agent | Path to droid binary |

### Optional Variables

Configuration tuning and features:

- `LOG_LEVEL` - Logging verbosity (debug, info, warn, error)
- `AGENTAPI_PORT` - Server port (default: 3284)
- `RATE_LIMIT_ENABLED` - Enable rate limiting (default: true)
- `CIRCUIT_BREAKER_ENABLED` - Enable circuit breaker (default: true)
- `ENABLE_METRICS` - Enable Prometheus metrics (default: true)

See `.env.example` for the complete list with detailed documentation.

---

## Required Dependencies

### 1. PostgreSQL

**Why:** AgentAPI uses PostgreSQL for persistent data storage including user sessions, chat history, and configuration.

**Installation:**

macOS (Homebrew):
```bash
brew install postgresql@15
brew services start postgresql@15
```

Ubuntu/Debian:
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Setup:**
```bash
# Create database and user
sudo -u postgres psql
```

```sql
CREATE DATABASE agentapi_dev;
CREATE USER agentapi WITH PASSWORD 'agentapi_dev_pass';
GRANT ALL PRIVILEGES ON DATABASE agentapi_dev TO agentapi;
\q
```

**Environment Variable:**
```bash
DATABASE_URL=postgresql://agentapi:agentapi_dev_pass@localhost:5432/agentapi_dev?sslmode=disable
```

### 2. Redis

**Why:** Redis provides fast caching, session storage, rate limiting, and pub/sub capabilities.

**Installation:**

macOS (Homebrew):
```bash
brew install redis
brew services start redis
```

Ubuntu/Debian:
```bash
sudo apt update
sudo apt install redis-server
sudo systemctl start redis-server
```

**Verify:**
```bash
redis-cli ping
# Should return: PONG
```

**Environment Variable:**
```bash
REDIS_URL=redis://localhost:6379/0
```

**Alternative: Upstash Redis (Cloud)**

For production or serverless deployments:

1. Sign up at https://console.upstash.com
2. Create a Redis database
3. Copy credentials:

```bash
UPSTASH_REDIS_REST_URL=https://your-redis.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-token-here
REDIS_PROTOCOL=rest
```

---

## WorkOS AuthKit Setup

**Why:** AgentAPI uses WorkOS AuthKit for enterprise-grade authentication with JWT tokens.

### Steps

1. **Create WorkOS Account**
   - Go to https://dashboard.workos.com
   - Sign up or log in

2. **Create an Environment**
   - For development: Create a "Test" environment
   - For production: Create a "Production" environment

3. **Get API Credentials**
   - Navigate to **API Keys** section
   - Copy the **API Key**:
     ```bash
     WORKOS_API_KEY=sk_test_abc123...
     ```

4. **Configure AuthKit**
   - Go to **AuthKit** section
   - Enable AuthKit if not already enabled
   - Copy the **Client ID**:
     ```bash
     WORKOS_CLIENT_ID=client_test_xyz789...
     ```

5. **Set JWKS URL**
   - The JWKS URL follows this pattern:
     ```bash
     AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/{CLIENT_ID}
     ```
   - Replace `{CLIENT_ID}` with your actual client ID

### Example Configuration

Development:
```bash
WORKOS_API_KEY=sk_test_1234567890abcdef
WORKOS_CLIENT_ID=client_test_abcd1234
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_test_abcd1234
AUTHKIT_ISSUER=https://api.workos.com
```

Production:
```bash
WORKOS_API_KEY=sk_live_1234567890abcdef
WORKOS_CLIENT_ID=client_01H1234567890
AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_01H1234567890
AUTHKIT_ISSUER=https://api.workos.com
```

### Testing Authentication

Create a test user in WorkOS dashboard, then test with:

```bash
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-3-opus","messages":[{"role":"user","content":"Hello"}]}'
```

### Skipping Auth (Development Only)

For local testing, you can skip authentication for specific paths:

```bash
# Not an environment variable, requires code modification
# See lib/middleware/authkit.go for SkipPaths configuration
```

---

## Installing CCRouter

**What:** CCRouter is a multi-model routing CLI that provides access to VertexAI, OpenAI, and Anthropic models with intelligent routing.

**Why:** Use CCRouter for:
- Multi-cloud model access
- Cost optimization via model routing
- Fallback between providers

### Installation

#### macOS (Homebrew)

```bash
# Add tap (if available)
brew tap yourusername/ccrouter

# Install
brew install ccrouter

# Verify installation
which ccr
# Should output: /opt/homebrew/bin/ccr

ccr --version
```

#### From Binary

```bash
# Download latest release
curl -L https://github.com/yourusername/ccrouter/releases/latest/download/ccr-darwin-arm64 -o ccr

# Make executable
chmod +x ccr

# Move to PATH
sudo mv ccr /usr/local/bin/ccr

# Verify
ccr --version
```

#### From Source

```bash
git clone https://github.com/yourusername/ccrouter.git
cd ccrouter
go build -o ccr ./cmd/ccr
sudo mv ccr /usr/local/bin/
```

### Configuration

Set the path in your environment:

```bash
# macOS (Homebrew)
CCROUTER_PATH=/opt/homebrew/bin/ccr

# Linux
CCROUTER_PATH=/usr/local/bin/ccr

# Custom installation
CCROUTER_PATH=~/.local/bin/ccr
```

### Required API Keys for CCRouter

CCRouter needs API keys for the models it routes to:

```bash
# For VertexAI/Gemini models
VERTEX_AI_PROJECT_ID=your-gcp-project-id
VERTEX_AI_LOCATION=us-central1
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# For OpenAI models
OPENAI_API_KEY=sk-...

# For Anthropic models
ANTHROPIC_API_KEY=sk-ant-...
```

### Verify CCRouter

Test that ccr is working:

```bash
ccr code --model vertex-gemini --system "You are helpful" <<< "Say hello"
```

---

## Installing Droid

**What:** Droid is a CLI tool providing access to 14+ AI models through OpenRouter and direct API integrations.

**Why:** Use Droid for:
- Access to diverse model selection
- OpenRouter integration
- Local model experimentation

### Installation

#### Method 1: Installation Script

```bash
curl -fsSL https://droid.ai/install.sh | sh
```

This installs to `~/.droid/bin/droid`.

#### Method 2: Homebrew (if available)

```bash
brew tap droid-ai/droid
brew install droid

# Verify
which droid
# Output: /opt/homebrew/bin/droid
```

#### Method 3: From Binary

```bash
# Download for your platform
curl -L https://github.com/droid-ai/droid/releases/latest/download/droid-darwin-arm64 -o droid

# Make executable
chmod +x droid

# Move to standard location
mkdir -p ~/.droid/bin
mv droid ~/.droid/bin/

# Add to PATH in ~/.zshrc or ~/.bashrc
echo 'export PATH="$HOME/.droid/bin:$PATH"' >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

### Configuration

Set the path:

```bash
# Default installation
DROID_PATH=~/.droid/bin/droid

# Homebrew
DROID_PATH=/opt/homebrew/bin/droid

# Custom
DROID_PATH=/usr/local/bin/droid
```

### Required API Keys for Droid

Droid requires API keys for models you want to use:

```bash
# For Claude models
ANTHROPIC_API_KEY=sk-ant-...

# For GPT models
OPENAI_API_KEY=sk-...

# For OpenRouter models (optional)
OPENROUTER_API_KEY=sk-or-...

# For Google/PaLM models
GOOGLE_API_KEY=...
```

### Verify Droid

Test that droid is working:

```bash
droid claude-3-sonnet --system "You are helpful" <<< "Say hello"
```

---

## Database Setup

### Development Database

1. **Create Database:**
```bash
createdb agentapi_dev
```

2. **Set Environment Variable:**
```bash
DATABASE_URL=postgresql://agentapi:agentapi_dev_pass@localhost:5432/agentapi_dev?sslmode=disable
```

3. **Run Migrations:**
```bash
# If using migrations
agentapi migrate up
```

### Production Database

#### Managed PostgreSQL (Recommended)

**AWS RDS:**
1. Create RDS PostgreSQL instance
2. Configure security groups
3. Get connection string:
```bash
DATABASE_URL=postgresql://username:password@your-rds-endpoint.amazonaws.com:5432/agentapi?sslmode=require
```

**Supabase:**
1. Create project at https://supabase.com
2. Get PostgreSQL connection string from Settings > Database
3. Set environment variable:
```bash
DATABASE_URL=postgresql://postgres:your-password@db.your-project.supabase.co:5432/postgres?sslmode=require
```

**Neon:**
1. Create database at https://neon.tech
2. Copy connection string:
```bash
DATABASE_URL=postgresql://username:password@ep-random-name.region.aws.neon.tech/dbname?sslmode=require
```

### Connection Pool Settings

For production, tune the connection pool:

```bash
DATABASE_POOL_SIZE=20
DATABASE_TIMEOUT=10
```

---

## Redis Setup

### Development Redis

Local Redis is fine for development:

```bash
# Start Redis
redis-server

# Or with Homebrew
brew services start redis

# Set environment
REDIS_URL=redis://localhost:6379/0
```

### Production Redis

#### Upstash Redis (Recommended for Serverless)

1. Sign up at https://console.upstash.com
2. Create a Redis database
3. Choose region close to your app
4. Get connection details:

```bash
UPSTASH_REDIS_REST_URL=https://your-endpoint.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-token-here
REDIS_PROTOCOL=rest
REDIS_ENABLE=true
```

#### AWS ElastiCache

1. Create ElastiCache Redis cluster
2. Configure VPC and security groups
3. Get endpoint:

```bash
REDIS_URL=redis://your-cluster.cache.amazonaws.com:6379/0
```

#### Redis Cloud

1. Sign up at https://redis.com/try-free
2. Create subscription
3. Get connection string:

```bash
REDIS_URL=redis://default:password@redis-12345.c123.us-east-1-1.ec2.cloud.redislabs.com:12345
```

### Redis Configuration

Tune Redis settings:

```bash
REDIS_MAX_POOL_SIZE=20
REDIS_CONNECTION_TIMEOUT=10s
REDIS_PROTOCOL=native
```

---

## Development Setup

### Complete Development Checklist

1. **Install Dependencies**
   - [ ] PostgreSQL installed and running
   - [ ] Redis installed and running
   - [ ] Go 1.21+ installed
   - [ ] CCRouter installed (optional)
   - [ ] Droid installed (optional)

2. **Setup Environment**
   ```bash
   # Copy development template
   cp .env.development .env

   # Generate secrets
   echo "TOKEN_ENCRYPTION_KEY=$(openssl rand -hex 32)" >> .env
   echo "JWT_SECRET=$(openssl rand -base64 32)" >> .env
   echo "SESSION_SECRET=$(openssl rand -base64 32)" >> .env
   ```

3. **Configure Database**
   ```bash
   # Create database
   createdb agentapi_dev

   # Update .env
   DATABASE_URL=postgresql://agentapi:agentapi_dev_pass@localhost:5432/agentapi_dev?sslmode=disable
   ```

4. **Configure Redis**
   ```bash
   # Verify Redis is running
   redis-cli ping

   # Update .env
   REDIS_URL=redis://localhost:6379/0
   ```

5. **Add API Keys**
   ```bash
   # Add to .env
   ANTHROPIC_API_KEY=sk-ant-api03-your-dev-key
   OPENAI_API_KEY=sk-your-openai-key
   ```

6. **Configure WorkOS (Test Environment)**
   ```bash
   WORKOS_API_KEY=sk_test_your_key
   WORKOS_CLIENT_ID=client_test_your_id
   AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_test_your_id
   ```

7. **Set Agent Paths**
   ```bash
   CCROUTER_PATH=/opt/homebrew/bin/ccr
   DROID_PATH=~/.droid/bin/droid
   ```

8. **Start Development Server**
   ```bash
   agentapi server -- claude
   ```

9. **Verify Setup**
   ```bash
   # Health check
   curl http://localhost:3284/health

   # Metrics
   curl http://localhost:9090/metrics
   ```

### Development Tips

- Use `LOG_LEVEL=debug` for verbose output
- Set `CIRCUIT_BREAKER_ENABLED=false` to avoid tripping during debugging
- Set `RATE_LIMIT_ENABLED=false` for unlimited testing
- Use `PRETTY_PRINT_JSON=true` for readable responses
- Enable `DEBUG=true` for additional debug endpoints

---

## Production Setup

### Production Checklist

1. **Security Review**
   - [ ] All secrets generated using cryptographically secure methods
   - [ ] Secrets stored in secrets manager (not .env file)
   - [ ] SSL/TLS enabled for all connections
   - [ ] Rate limiting enabled
   - [ ] Circuit breaker enabled
   - [ ] CORS properly configured

2. **Infrastructure**
   - [ ] Managed PostgreSQL with automated backups
   - [ ] Managed Redis with persistence
   - [ ] Load balancer configured
   - [ ] Auto-scaling enabled
   - [ ] Monitoring and alerting set up

3. **Environment Variables**
   ```bash
   # Use production WorkOS credentials
   WORKOS_API_KEY=sk_live_...
   WORKOS_CLIENT_ID=client_01H...
   AUTHKIT_JWKS_URL=https://api.workos.com/sso/jwks/client_01H...

   # Production database with SSL
   DATABASE_URL=postgresql://user:pass@prod-db.amazonaws.com:5432/agentapi?sslmode=require

   # Production Redis
   REDIS_URL=rediss://default:pass@prod-redis.cloud.redislabs.com:6379

   # Strong secrets (use secrets manager)
   TOKEN_ENCRYPTION_KEY=<64-char-hex>
   JWT_SECRET=<strong-random-string>
   SESSION_SECRET=<strong-random-string>

   # Production settings
   AGENTAPI_ENVIRONMENT=production
   LOG_LEVEL=info
   LOG_FORMAT=json

   # Enable all protections
   RATE_LIMIT_ENABLED=true
   CIRCUIT_BREAKER_ENABLED=true
   ENABLE_METRICS=true
   ENABLE_AUDIT_LOGGING=true

   # Strict CORS
   AGENTAPI_ALLOWED_ORIGINS=https://yourdomain.com
   AGENTAPI_ALLOWED_HOSTS=yourdomain.com
   ```

4. **Agent Configuration**
   ```bash
   # Production paths (Docker or system-wide)
   CCROUTER_PATH=/usr/local/bin/ccr
   DROID_PATH=/usr/local/bin/droid

   # Production timeouts
   CCROUTER_TIMEOUT=300
   DROID_TIMEOUT=300
   CHAT_TIMEOUT=120
   ```

5. **Monitoring**
   ```bash
   # Enable metrics
   ENABLE_METRICS=true
   METRICS_PORT=9090

   # Error tracking
   SENTRY_DSN=https://your-sentry-dsn@sentry.io/project

   # Health checks
   ENABLE_HEALTH_CHECKS=true
   HEALTH_CHECK_PATH=/health
   ```

### Deployment Platforms

#### AWS ECS/Fargate

```bash
# Use AWS Secrets Manager
aws secretsmanager create-secret \
  --name agentapi/production \
  --secret-string file://secrets.json

# Reference in task definition
{
  "secrets": [
    {
      "name": "WORKOS_API_KEY",
      "valueFrom": "arn:aws:secretsmanager:region:account:secret:agentapi/production:WORKOS_API_KEY::"
    }
  ]
}
```

#### Vercel/Netlify

Use environment variables in dashboard:
- Add all required variables
- Use different values for preview vs production
- Never commit .env to Git

#### Docker

```dockerfile
# Use build args for non-sensitive config
ARG AGENTAPI_PORT=3284

# Use secrets for sensitive values
RUN --mount=type=secret,id=env,target=/app/.env \
    cp /app/.env.example /app/.env
```

---

## Security Best Practices

### 1. Secret Generation

Always use cryptographically secure methods:

```bash
# Good
openssl rand -hex 32

# Good
head -c 32 /dev/urandom | base64

# Bad - DO NOT USE
echo "my-secret-key"
```

### 2. Secret Storage

**Never** store secrets in:
- Git repositories
- Unencrypted files
- Application logs
- Client-side code

**Always** use:
- Secrets managers (AWS Secrets Manager, Vault, etc.)
- Environment variables (in secure environments)
- Encrypted configuration (with rotation)

### 3. Access Control

```bash
# Restrict .env file permissions
chmod 600 .env

# Ensure only owner can read
ls -la .env
# Should show: -rw------- 1 user group
```

### 4. SSL/TLS

Always use SSL in production:

```bash
# Database
DATABASE_URL=postgresql://...?sslmode=require

# Redis
REDIS_URL=rediss://...  # Note the 'rediss' with double 's'
```

### 5. API Key Rotation

Regularly rotate all API keys:

```bash
# Set up rotation schedule
# - WorkOS: Every 90 days
# - AI Provider Keys: Every 30 days
# - Encryption Keys: Every 180 days
```

### 6. Rate Limiting

Protect against abuse:

```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST_SIZE=10
```

### 7. CORS Configuration

Be specific about allowed origins:

```bash
# Bad
AGENTAPI_ALLOWED_ORIGINS=*

# Good
AGENTAPI_ALLOWED_ORIGINS=https://yourdomain.com https://app.yourdomain.com
```

---

## Troubleshooting

### Common Issues

#### 1. "WORKOS_API_KEY not found"

**Problem:** WorkOS credentials not set or incorrect.

**Solution:**
```bash
# Verify credentials are set
echo $WORKOS_API_KEY
echo $WORKOS_CLIENT_ID

# Check for typos
cat .env | grep WORKOS

# Test with WorkOS API
curl -H "Authorization: Bearer $WORKOS_API_KEY" \
  https://api.workos.com/organizations
```

#### 2. "Failed to connect to database"

**Problem:** PostgreSQL connection issues.

**Solution:**
```bash
# Test connection
psql $DATABASE_URL

# Check if PostgreSQL is running
pg_isready

# Verify credentials
echo $DATABASE_URL

# Check SSL mode
# Development: sslmode=disable
# Production: sslmode=require
```

#### 3. "Redis connection refused"

**Problem:** Redis not running or wrong URL.

**Solution:**
```bash
# Check if Redis is running
redis-cli ping

# Verify URL
echo $REDIS_URL

# Start Redis
brew services start redis
# or
redis-server
```

#### 4. "CCRouter/Droid not found"

**Problem:** Binary path incorrect or not installed.

**Solution:**
```bash
# Find ccr
which ccr

# Find droid
which droid

# Update .env with correct path
CCROUTER_PATH=$(which ccr)
DROID_PATH=$(which droid)

# Verify binary works
ccr --version
droid --version
```

#### 5. "JWT validation failed"

**Problem:** JWKS URL incorrect or keys not refreshing.

**Solution:**
```bash
# Verify JWKS URL
curl $AUTHKIT_JWKS_URL

# Should return JSON with keys
# Example:
# {"keys":[{"kid":"...","kty":"RSA",...}]}

# Check client ID matches
echo $WORKOS_CLIENT_ID
echo $AUTHKIT_JWKS_URL | grep -o 'client_[^/]*'
```

#### 6. "Rate limit exceeded"

**Problem:** Too many requests or misconfigured limits.

**Solution:**
```bash
# For development, disable rate limiting
RATE_LIMIT_ENABLED=false

# Or increase limits
RATE_LIMIT_REQUESTS_PER_MINUTE=1000

# Check Redis for rate limit keys
redis-cli keys "ratelimit:*"
```

#### 7. "Circuit breaker open"

**Problem:** Too many failures triggered circuit breaker.

**Solution:**
```bash
# For development, disable circuit breaker
CIRCUIT_BREAKER_ENABLED=false

# Or adjust thresholds
CIRCUIT_BREAKER_FAILURE_THRESHOLD=10

# Reset circuit breaker state in Redis
redis-cli del "circuitbreaker:*"
```

### Debugging Tips

#### Enable Debug Logging

```bash
LOG_LEVEL=debug
LOG_FORMAT=text
```

#### Check Health Endpoint

```bash
curl http://localhost:3284/health

# Should return:
# {"status":"healthy","timestamp":"..."}
```

#### View Metrics

```bash
curl http://localhost:9090/metrics

# Look for errors:
# agentapi_errors_total
# agentapi_requests_failed_total
```

#### Test Database Connection

```bash
psql $DATABASE_URL -c "SELECT 1;"
```

#### Test Redis Connection

```bash
redis-cli -u $REDIS_URL ping
```

#### Verify Environment Variables Loaded

```bash
# In your application, log all env vars (development only!)
env | grep AGENTAPI
env | grep WORKOS
env | grep DATABASE
env | grep REDIS
```

---

## Secret Rotation

### Why Rotate Secrets?

- Security best practice
- Limit exposure window if compromised
- Compliance requirements (SOC 2, HIPAA, etc.)

### Rotation Schedule

| Secret Type | Rotation Frequency |
|-------------|-------------------|
| API Keys (AI Providers) | Every 30 days |
| WorkOS API Key | Every 90 days |
| JWT/Session Secrets | Every 180 days |
| Token Encryption Key | Every 180 days |
| Database Passwords | Every 90 days |
| Redis Passwords | Every 90 days |

### Safe Rotation Process

#### 1. API Keys (Zero Downtime)

Most services support multiple active keys:

```bash
# Step 1: Generate new key in provider dashboard
# Step 2: Add new key to environment (keep old one)
ANTHROPIC_API_KEY_NEW=sk-ant-new-key

# Step 3: Update application to try both keys
# Step 4: Monitor for 24 hours
# Step 5: Remove old key from environment
# Step 6: Revoke old key in provider dashboard
```

#### 2. JWT Secret Rotation

Use a grace period:

```bash
# Step 1: Generate new secret
JWT_SECRET_NEW=$(openssl rand -base64 32)

# Step 2: Configure application to verify with both old and new
# Step 3: Start signing new tokens with new secret
# Step 4: Wait for old tokens to expire (check SESSION_TTL)
# Step 5: Remove old secret
```

#### 3. Database Password Rotation

```bash
# Step 1: Create new user with same privileges
CREATE USER agentapi_new WITH PASSWORD 'new-password';
GRANT ALL PRIVILEGES ON DATABASE agentapi TO agentapi_new;

# Step 2: Update DATABASE_URL to use new user
DATABASE_URL=postgresql://agentapi_new:new-password@...

# Step 3: Deploy and verify connections work
# Step 4: Drop old user
DROP USER agentapi;
```

#### 4. Token Encryption Key Rotation

Requires re-encrypting existing tokens:

```bash
# Step 1: Generate new key
TOKEN_ENCRYPTION_KEY_NEW=$(openssl rand -hex 32)

# Step 2: Run migration script to re-encrypt tokens
agentapi migrate reencrypt-tokens \
  --old-key $TOKEN_ENCRYPTION_KEY \
  --new-key $TOKEN_ENCRYPTION_KEY_NEW

# Step 3: Update environment variable
TOKEN_ENCRYPTION_KEY=$TOKEN_ENCRYPTION_KEY_NEW

# Step 4: Restart application
```

### Automated Rotation

For production, consider automating rotation:

**AWS Secrets Manager:**
```bash
aws secretsmanager rotate-secret \
  --secret-id agentapi/production/database \
  --rotation-lambda-arn arn:aws:lambda:...
```

**HashiCorp Vault:**
```bash
vault write database/rotate-role/agentapi
```

### Rotation Checklist

Before rotating:
- [ ] Backup current secrets securely
- [ ] Test rotation in staging environment
- [ ] Plan rollback procedure
- [ ] Schedule during low-traffic period
- [ ] Notify team of planned rotation

During rotation:
- [ ] Monitor application logs
- [ ] Watch error rates in metrics
- [ ] Check authentication success rate
- [ ] Verify database connections

After rotation:
- [ ] Confirm old secrets are revoked
- [ ] Update documentation
- [ ] Securely delete old secret backups (after grace period)
- [ ] Schedule next rotation

---

## Getting Help

### Resources

- **Documentation:** [AgentAPI Docs](https://github.com/coder/agentapi)
- **Issues:** [GitHub Issues](https://github.com/coder/agentapi/issues)
- **Discussions:** [GitHub Discussions](https://github.com/coder/agentapi/discussions)

### Support Checklist

When reporting configuration issues, include:

1. Environment (development, staging, production)
2. Relevant environment variables (redact secrets!)
3. Error messages or logs
4. Steps to reproduce
5. Expected vs actual behavior

### Example Support Request

```
**Environment:** Development (macOS, Homebrew)

**Issue:** CCRouter agent not found

**Environment Variables:**
CCROUTER_PATH=/opt/homebrew/bin/ccr
CCROUTER_TIMEOUT=60
LOG_LEVEL=debug

**Error:**
Failed to initialize CCRouter agent: exec: "/opt/homebrew/bin/ccr": file does not exist

**Steps:**
1. Installed via `brew install ccrouter`
2. Set CCROUTER_PATH in .env
3. Started server: `agentapi server -- ccrouter`

**Expected:** Server starts with CCRouter agent
**Actual:** Error about ccr not found

**Additional Info:**
- `which ccr` returns `/opt/homebrew/bin/ccr`
- `ccr --version` works correctly
- File permissions: -rwxr-xr-x
```

---

## Appendix

### Complete .env Template

See `.env.example` for the complete template with all variables documented.

### Environment Variable Reference

| Category | Variables |
|----------|-----------|
| Auth | WORKOS_API_KEY, WORKOS_CLIENT_ID, AUTHKIT_JWKS_URL |
| Agents | CCROUTER_PATH, DROID_PATH, DEFAULT_AGENT |
| Database | DATABASE_URL, DATABASE_POOL_SIZE, DATABASE_TIMEOUT |
| Redis | REDIS_URL, REDIS_ENABLE, REDIS_PROTOCOL |
| Security | TOKEN_ENCRYPTION_KEY, JWT_SECRET, SESSION_SECRET |
| API | AGENTAPI_PORT, AGENTAPI_HOST, AGENTAPI_ENVIRONMENT |
| Logging | LOG_LEVEL, LOG_FORMAT |
| Features | ENABLE_METRICS, ENABLE_TRACING, ENABLE_AUDIT_LOGGING |

### Quick Reference Commands

```bash
# Generate secrets
openssl rand -hex 32          # TOKEN_ENCRYPTION_KEY
openssl rand -base64 32       # JWT_SECRET / SESSION_SECRET

# Check services
pg_isready                    # PostgreSQL
redis-cli ping                # Redis
which ccr                     # CCRouter
which droid                   # Droid

# Test connections
psql $DATABASE_URL -c "SELECT 1;"
redis-cli -u $REDIS_URL ping
curl http://localhost:3284/health

# View logs
tail -f logs/agentapi.log
docker logs -f agentapi-container
```

---

**Last Updated:** 2025-01-24
**Version:** 1.0.0
**Maintained By:** AgentAPI Team

# Configuration Guide

## Overview

The `atomsAgent` package uses YAML-based configuration files for both application settings and secrets. This approach provides a clean, version-controllable way to manage configuration while keeping secrets secure.

## Configuration Files

### Location

Configuration files are located in the `atomsAgent/config/` directory:

```
atomsAgent/
├── config/
│   ├── config.yml          # Non-sensitive application settings
│   ├── secrets.yml         # Sensitive credentials (gitignored)
│   └── secrets.yml.example # Template for secrets
```

### config.yml

Contains non-sensitive application configuration:

- Application version and feature flags
- Model settings (default model, cache TTL)
- Tool permissions and allowed tools
- Sandbox and workspace paths
- CORS settings

**Example:**
```yaml
app_version: "0.1.0"
enable_docs: true
default_model: "claude-3-5-sonnet-20241022"
default_allowed_tools:
  - "Read"
  - "Write"
  - "Edit"
  - "Bash"
```

### secrets.yml

Contains sensitive credentials and API keys:

- Vertex AI credentials and project ID
- Claude API key
- Supabase URL and service keys
- Redis/Upstash connection URL
- Database connection string
- Authentication configuration
- Static API keys for development

**⚠️ Important:** This file is gitignored and should never be committed to version control.

## Setup Instructions

### 1. Initial Setup

```bash
# Navigate to the atomsAgent directory
cd atomsAgent

# Copy the example secrets file
cp config/secrets.yml.example config/secrets.yml

# Edit with your actual credentials
nano config/secrets.yml  # or use your preferred editor
```

### 2. Configure Your Secrets

Edit `config/secrets.yml` and fill in your actual values:

```yaml
# Vertex AI
vertex_project_id: "your-actual-project-id"
vertex_location: "us-central1"

# Supabase
supabase_url: "https://your-project.supabase.co"
supabase_service_key: "your-actual-service-key"

# Database
database_url: "postgresql://user:pass@host:port/db"
```

### 3. (Optional) Customize Application Settings

Edit `config/config.yml` to customize application behavior:

```yaml
default_model: "claude-3-5-sonnet-20241022"
enable_docs: true
sandbox_root_dir: "/tmp/atomsAgent/sandboxes"
```

## Loading Priority

Configuration is loaded in the following order (later sources override earlier ones):

1. **Default values** - Hardcoded in the Pydantic models
2. **YAML files** - From `config/config.yml` and `config/secrets.yml`
3. **Environment variables** - Prefixed with `ATOMS_` or `ATOMS_SECRET_`

### Environment Variable Overrides

You can override any setting using environment variables:

```bash
# Non-sensitive settings (ATOMS_ prefix)
export ATOMS_DEFAULT_MODEL="claude-3-opus-20240229"
export ATOMS_ENABLE_DOCS=false

# Sensitive settings (ATOMS_SECRET_ prefix)
export ATOMS_SECRET_SUPABASE_URL="https://override.supabase.co"
export ATOMS_SECRET_CLAUDE_API_KEY="sk-ant-..."
```

### Custom File Locations

Override the default config file locations:

```bash
# Use custom config files
export ATOMS_CONFIG_PATH="/path/to/custom/config.yml"
export ATOMS_SECRETS_PATH="/path/to/custom/secrets.yml"
```

## File Search Order

The configuration loader searches for files in this order:

### For secrets.yml:
1. `$ATOMS_SECRETS_PATH` (if set)
2. `atomsAgent/config/secrets.yml` (package config directory)
3. `./secrets.yml` (current working directory)
4. Environment variables only

### For config.yml:
1. `$ATOMS_CONFIG_PATH` (if set)
2. `atomsAgent/config/config.yml` (package config directory)
3. `./config.yml` (current working directory)
4. Defaults and environment variables

## Security Best Practices

1. **Never commit secrets.yml** - It's gitignored by default
2. **Use environment variables in production** - For deployment environments
3. **Rotate credentials regularly** - Especially API keys and service keys
4. **Use minimal permissions** - Grant only necessary access to service accounts
5. **Keep secrets.yml.example updated** - But with placeholder values only

## Debugging

Enable debug output to see which config files are being loaded:

```bash
# Debug config loading
export DEBUG_CONFIG=1

# Debug secrets loading
export DEBUG_SECRETS=1

# Run your application
python -m atomsAgent.cli.main server run
```

This will print the paths being checked and which files were found.

## Development vs Production

### Development

Use YAML files for local development:
- Easy to edit and version control (config.yml)
- Keeps secrets local (secrets.yml)
- Quick iteration

### Production

Use environment variables for production:
- Integrate with secret managers (AWS Secrets Manager, GCP Secret Manager, etc.)
- Better security and audit trails
- Easier to rotate credentials

Example production setup:
```bash
# In your deployment script or container
export ATOMS_SECRET_SUPABASE_URL="${SUPABASE_URL}"
export ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY="${SUPABASE_KEY}"
export ATOMS_SECRET_DATABASE_URL="${DATABASE_URL}"
# ... etc
```

## Troubleshooting

### "Supabase configuration is missing" Error

This means `secrets.yml` wasn't found or doesn't contain the required Supabase settings.

**Solution:**
1. Verify `atomsAgent/config/secrets.yml` exists
2. Check it contains `supabase_url` and `supabase_service_key`
3. Run with `DEBUG_SECRETS=1` to see which paths are being checked

### Config values not loading

**Check:**
1. YAML syntax is correct (use a YAML validator)
2. Field names match exactly (case-sensitive)
3. File is in the correct location
4. No environment variables are overriding the values

### Permission denied errors

**Solution:**
```bash
# Ensure config files are readable
chmod 600 atomsAgent/config/secrets.yml
chmod 644 atomsAgent/config/config.yml
```

### Supabase `permission denied for table system_prompts`

This happens when the Supabase `service_role` does not have the required grants. Run the
`06_fix_system_prompts_rls.sql` migration against your Supabase database with the service
role key:

```bash
psql "$DATABASE_URL" -f 06_fix_system_prompts_rls.sql
```

The script:
- Grants `service_role` usage on the `public` schema
- Grants the required `SELECT/INSERT/UPDATE/DELETE` privileges on `system_prompts`,
  `mcp_configurations`, `chat_sessions`, `chat_messages`, and `platform_admins`
- Grants read access on `organizations`, `users`, `user_sessions`, and `audit_logs`
- Ensures RLS policies allow the service role to fetch enabled prompts

After running the script, restart the atomsAgent server and verify `/health` returns `200`.

### Claude Vertex configuration missing

The Agent SDK expects Vertex AI credentials. Ensure the following are set before running
`atoms-agent server run`:

```bash
export CLAUDE_CODE_USE_VERTEX=1
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
export VERTEX_PROJECT_ID="your-gcp-project"
export VERTEX_LOCATION="us-central1"
```

Alternatively, place `vertex_credentials_path`, `vertex_project_id`, and `vertex_location`
in `config/secrets.yml` / `config.yml`. When these values are missing, the backend now
returns a `503` with a clear error instead of timing out while the SDK waits for
Vertex authentication.

## Migration from .env

If you're migrating from a `.env` file:

1. Copy values from `.env` to `config/secrets.yml`
2. Remove the `ATOMS_SECRET_` prefix from variable names
3. Convert to YAML format
4. Test with `DEBUG_SECRETS=1`

Example:
```bash
# .env (old)
ATOMS_SECRET_SUPABASE_URL=https://example.supabase.co

# secrets.yml (new)
supabase_url: "https://example.supabase.co"
```

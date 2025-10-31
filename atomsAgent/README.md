# atomsAgent Service

This package hosts the FastAPI service that exposes Claude Code (running on Vertex AI) behind an OpenAI-compatible interface. It also provides MCP configuration APIs and prompt orchestration for the SaaS chat UI.

## Key Features
- `/v1/chat/completions` – OpenAI-compatible chat completions backed by Claude Agents SDK with Vertex AI models (Claude 4.5 Sonnet/Haiku, Gemini 2.5 Pro/Flash).
- `/v1/models` – Lists supported models by querying Vertex AI Model Garden and internal configuration.
- `/atoms/mcp` – CRUD endpoints for registering HTTP-based MCP servers with bearer token or OAuth DCR auth flows.
- Prompt orchestration that merges platform, organization, and user-level prompts with workflow metadata.
- Supabase integration using `sb-pydantic` generated Pydantic models.

## Development Setup
1. Change into this directory and install dependencies with [uv](https://github.com/astral-sh/uv) (or your preferred PEP 517 installer):
   ```bash
   cd atomsAgent
   uv pip install -e ".[dev]"
   ```
2. Generate Supabase models (requires `sb-pydantic`):
   ```bash
   python -m atomsAgent.codegen.supabase
   ```
3. Run the service:
   ```bash
   uvicorn atomsAgent.main:app --reload
   ```

4. Execute the fast unit checks:
   ```bash
   pytest tests
   ```

## Command-Line Interface

The project ships a Typer + Rich powered CLI once installed:

```bash
atoms-agent --help
```

Common commands:

| Command | Description |
| --- | --- |
| `atoms-agent vertex models` | List available Vertex AI models (add `--json` for machine-readable output). |
| `atoms-agent supabase generate-models --schema database/schema.sql --output src/atomsAgent/db` | Regenerate Supabase Pydantic models. |
| `atoms-agent mcp list --org <uuid>` | Inspect MCP configurations for an organisation (supports `create`, `update`, `delete`). |
| `atoms-agent prompt show --org <uuid> [--user <uuid>]` | Render the merged prompt stack for a tenant/workflow. |
| `atoms-agent server run --host 0.0.0.0 --port 3284 --reload` | Launch the FastAPI server using Uvicorn. |

Use `--json` on supported commands to emit structured output suitable for scripting.

## Configuration

`atomsAgent` uses YAML configuration files located in the `config/` directory:

### Configuration Files

- **`config/config.yml`** - Non-sensitive application configuration (model settings, tool permissions, etc.)
- **`config/secrets.yml`** - Sensitive credentials (API keys, database URLs, etc.)

Both files are loaded automatically from the package's `config/` directory. You can also override the location using environment variables:
- `ATOMS_CONFIG_PATH` - Path to custom config.yml
- `ATOMS_SECRETS_PATH` - Path to custom secrets.yml

### Setup

1. Copy the example secrets file:
   ```bash
   cp config/secrets.yml.example config/secrets.yml
   ```

2. Edit `config/secrets.yml` with your actual credentials

3. (Optional) Customize `config/config.yml` for your environment

### Environment Variable Overrides

Individual settings can still be overridden via environment variables:

- `ConfigSettings` (prefix `ATOMS_`) for non-sensitive values such as `ATOMS_VERTEX_AI_PROJECT_ID`, `ATOMS_DEFAULT_MODEL`, etc.
- `SecretSettings` (prefix `ATOMS_SECRET_`) for credentials such as `ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY`, `ATOMS_SECRET_REDIS_URL`, etc.

**Priority order**: Environment variables > YAML files > Defaults

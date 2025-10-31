# Design: atomsAgent Typer + Rich CLI

## Goals
- Provide a consolidated CLI (`atoms-agent`) with discoverable subcommands for routine atomsAgent workflows.
- Reuse existing service layers (Supabase repositories, Vertex model service, MCP registry, prompt orchestrator) rather than duplicating logic.
- Offer both human-friendly Rich output and machine-readable JSON when requested.

## Command Topology
```
atoms-agent
├── supabase generate-models [--schema <sql> --output <path>]
├── vertex models [--provider <owner> --json]
├── mcp
│   ├── list --org <uuid> [--user <uuid>] [--json]
│   ├── create --name ... --endpoint ... (--org <uuid> | --platform)
│   ├── update <id> [--name --endpoint --disable/--enable ...]
│   └── delete <id>
├── prompt show --org <uuid> [--user <uuid>] [--workflow <slug>] [--json]
└── server run [--host 0.0.0.0 --port 3284 --reload]
```

## Architecture Choices
- **Typer app composition**: Create module `atomsAgent/cli/main.py` exporting `app = typer.Typer()` and sub-typer instances per domain (`supabase`, `vertex`, `mcp`, `prompt`, `server`). Entry point registered in `pyproject.toml`.
- **Dependency Wiring**: Reuse existing dependency functions (`get_vertex_model_service`, etc.). For CLI context we instantiate dependencies directly (not via FastAPI request), but reuse same constructors to keep configuration consistent.
- **Output Formatting**: Default to Rich tables/progress. Provide `--json` to dump `json.dumps()` of dataclasses/pydantic models. Use `orjson` when available.
- **Error Handling**: Wrap service calls in try/except; print Rich `Console` error messages with non-zero exit codes.
- **Async Integration**: Typer commands run sync. We'll use `asyncio.run()` to invoke coroutine-based services (Vertex, MCP). Provide helper `run_async(coro)` to ensure event loop management.

## Dependencies
- Typer and Rich already present (codegen tool). Ensure they are listed in core dependencies via pyproject update.
- For CLI tests, use `typer.testing.CliRunner`.

## Testing Strategy
- Unit test each command using `CliRunner.invoke` with patched services (monkeypatch the dependency factories to return fakes).
- Test JSON and table output for `vertex models` and `mcp list`.
- For `server run`, mock Uvicorn `run` call to avoid starting server.

## Open Questions / Risks
- Need to ensure CLI commands respect settings (requires environment). We'll load `settings = load_settings()` at module import and pass to services.
- MCP create/update may require secrets (bearer token); handle via options with prompt support (`--bearer-token`, `--bearer-token-file`).

The CLI builds on existing modules, adding minimal new surface area while dramatically improving operator UX.

## Library Adoption Plan
| Library / Repo | Planned Usage | Integration Notes |
| --- | --- | --- |
| [`google-cloud-aiplatform`](https://github.com/googleapis/python-aiplatform) | Replace the bespoke REST plumbing in `VertexModelService` with `ModelServiceAsyncClient` so model discovery, pagination, and auth reuse the official SDK. | Instantiate the client with project/location from settings, reuse async paginator to feed CLI `vertex models` command. |
| [`supabase-py`](https://github.com/supabase-community/supabase-py) | Supersede the handcrafted REST wrapper under `atomsAgent.db` with the maintained Python client for CRUD/RPC calls. | Inject the async Supabase client via dependencies, update repositories to call the SDK (which handles headers, errors, and future schema changes). |
| [`sse-starlette`](https://github.com/sysid/sse-starlette) | Manage Server-Sent Events returned by `/v1/chat/completions` when `stream=True`. | Swap the manual `_serialize_chunk` logic for `EventSourceResponse`, ensuring keep-alive and `[DONE]` framing remain spec compliant. |
| [`PyFilesystem2`](https://github.com/PyFilesystem/pyfilesystem2) | Provide an abstraction for `SandboxManager` so per-session workspaces can target local directories, in-memory FS, or future object storage uniformly. | Replace direct `os`/`shutil` operations with FS operations and ensure permissions are enforced via the backend’s capabilities. |
| [`rich-click`](https://github.com/ewels/rich-click) | Enhance Typer-generated help/output with consistent Rich styling across all CLI subcommands. | Wrap the Typer app with `rich_click.rich_click_cli` so `atoms-agent --help` shows tables, command groups, and examples cleanly. |
| [`Authlib`](https://github.com/lepture/authlib) | Handle OAuth DCR flows and token exchanges for MCP configurations instead of manual HTTP crafting. | Use `authlib.integrations.httpx_client` within `MCPRegistryService` (and CLI commands) to obtain/refresh tokens securely. |
| [`prometheus-fastapi-instrumentator`](https://github.com/trallnag/prometheus-fastapi-instrumentator) | Expose `/metrics` endpoint and gather latency/call counts for chat, model listing, and admin APIs. | Add middleware during FastAPI app creation; configure registry/labels for multi-tenant observability. |
| [`structlog` + `rich.logging.RichHandler`](https://github.com/hynek/structlog) | Standardise structured logging that plays well both in CLI and API contexts, with Rich rendering during interactive runs. | Configure logging in `main.create_app()` and CLI entry to emit JSON (for logs) or Rich-formatted output when `--verbose`. |
| [`watchfiles`](https://github.com/samuelcolvin/watchfiles) | Provide rapid autoreload when running `atoms-agent server run --reload`. | Replace Uvicorn’s default watcher with `watchfiles` for better cross-platform support and faster reload times. |
| [`redis.asyncio`](https://github.com/redis/redis-py) (via `aiocache` backend) | Allow production deployments to use Redis for model/MCP cache instead of in-memory defaults. | Configure `aiocache` to use the Redis backend when `ATOMS_SECRET_REDIS_URL` is present, ensuring CLI commands share cached data with the API. |
| [`typer.testing`](https://typer.tiangolo.com/testing/) | Exercise CLI commands in unit tests without spawning real services. | Use `CliRunner` to assert flag parsing, Rich/JSON outputs, and exit codes for each new subcommand. |

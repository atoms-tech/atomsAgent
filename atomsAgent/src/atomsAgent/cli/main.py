"""Typer CLI entrypoint for atomsAgent developer tooling."""

from __future__ import annotations

import asyncio
import functools
import json
import os
import pathlib
import subprocess
import sys
from collections.abc import Iterable
from datetime import datetime
from typing import TYPE_CHECKING, Any, Literal, cast
from uuid import UUID

import typer
import uvicorn
from pydantic import HttpUrl, TypeAdapter, ValidationError
from rich.console import Console
from rich.json import JSON
from rich.table import Table

from atomsAgent.schemas.mcp import (
    MCPConfiguration,
    MCPCreateRequest,
    MCPMetadata,
    MCPScope,
    MCPUpdateRequest,
)

if TYPE_CHECKING:  # pragma: no cover - import only for type checking
    from atomsAgent.services.mcp_registry import MCPRegistryService
    from atomsAgent.services.prompts import PromptOrchestrator
    from atomsAgent.services.vertex_models import VertexModelService


AuthTypeLiteral = Literal["none", "bearer", "oauth"]
ScopeLiteral = Literal["platform", "organization", "user"]

_HTTP_URL_ADAPTER = TypeAdapter(HttpUrl)
_AUTH_TYPES: set[str] = {"none", "bearer", "oauth"}
_OAUTH_PROVIDERS: set[str] = {"github", "google", "microsoft", "auth0"}


def _normalize_scope(value: str) -> ScopeLiteral:
    normalized = value.lower()
    if normalized not in {"platform", "organization", "user"}:
        raise CommandError("Scope must be one of platform|organization|user")
    return cast(ScopeLiteral, normalized)


console = Console()

app = typer.Typer(
    name="atoms-agent",
    help="Developer CLI for atomsAgent systems",
    no_args_is_help=True,
    rich_markup_mode="markdown",
)

supabase_app = typer.Typer(help="Supabase helpers")
vertex_app = typer.Typer(help="Vertex AI utilities")
mcp_app = typer.Typer(help="Model Context Protocol management")
prompt_app = typer.Typer(help="Prompt orchestration utilities")
server_app = typer.Typer(help="Local server commands")

app.add_typer(supabase_app, name="supabase")
app.add_typer(vertex_app, name="vertex")
app.add_typer(mcp_app, name="mcp")
app.add_typer(prompt_app, name="prompt")
app.add_typer(server_app, name="server")


class CommandError(RuntimeError):
    """Raised when a CLI command should exit with a non-zero code."""


def command_handler(func):
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except CommandError as exc:
            console.print(f"[red]{exc}[/red]")
            raise typer.Exit(1) from exc

    return wrapper


def _project_root() -> pathlib.Path:
    # atomsAgent/src/atomsAgent/cli/main.py → parents[3] == atomsAgent project root
    return pathlib.Path(__file__).resolve().parents[3]


def _repo_root() -> pathlib.Path:
    # atomsAgent project root parent == agentapi mono-repo root
    return pathlib.Path(__file__).resolve().parents[4]


def _default_schema_path() -> pathlib.Path:
    return _repo_root() / "database" / "schema.sql"


def _default_output_dir() -> pathlib.Path:
    return _project_root() / "src" / "atomsAgent" / "db" / "models"


def _discover_database_url(explicit: str | None) -> str | None:
    if explicit:
        return explicit

    for key in (
        "SUPABASE_DB_URL",
        "DATABASE_URL",
        "ATOMS_SECRET_DATABASE_URL",
        "ATOMS_SECRET_DATABASE_URL",
    ):
        value = os.environ.get(key)
        if value:
            return value

    try:  # pragma: no cover - settings may not be wired during unit tests
        from atomsAgent.config import settings

        if getattr(settings, "database_url", None):
            return settings.database_url
    except Exception:
        pass

    candidate = _repo_root().parent / "atoms-mcp-prod" / "config" / "atoms.secrets.yaml"
    if candidate.exists():
        try:
            import yaml  # type: ignore

            with candidate.open(encoding="utf-8") as handle:
                loaded_data = yaml.safe_load(handle)
                data: dict[str, Any] = loaded_data if isinstance(loaded_data, dict) else {}
            return data.get("database_url")
        except Exception:
            return None

    return None


def _ensure_exists(path: pathlib.Path, description: str) -> None:
    if not path.exists():
        raise CommandError(f"{description} not found at {path}")


def _ensure_directory(path: pathlib.Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def _run_command(command: list[str]) -> subprocess.CompletedProcess:
    return subprocess.run(command, check=False)


def _run_async(awaitable):
    return asyncio.run(awaitable)


@supabase_app.command("generate-models")
@command_handler
def supabase_generate_models(
    schema: pathlib.Path = typer.Option(
        None,
        "--schema",
        help="Path to Supabase schema.sql",
        dir_okay=False,
        readable=True,
    ),
    output: pathlib.Path = typer.Option(
        None,
        "--output",
        help="Directory to place generated models",
        file_okay=False,
        writable=True,
    ),
    db_url: str | None = typer.Option(
        None,
        "--db-url",
        help="Override database URL used by supabase-pydantic",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit machine-readable output"),
) -> None:
    schema_path = schema or _default_schema_path()
    _ensure_exists(schema_path, "Schema file")

    output_dir = output or _default_output_dir()
    _ensure_directory(output_dir)

    resolved_db_url = _discover_database_url(db_url)

    command = [
        sys.executable,
        "-m",
        "supabase_pydantic",
        "gen",
        "--type",
        "pydantic",
        "--dir",
        str(output_dir),
    ]
    if resolved_db_url:
        command.extend(["--db-url", resolved_db_url])
    else:
        command.append("--local")

    result = _run_command(command)
    success = result.returncode == 0

    payload = {
        "schema": str(schema_path),
        "output": str(output_dir),
        "db_url": resolved_db_url,
        "returncode": result.returncode,
    }

    if json_output:
        console.print(JSON.from_data({"success": success, **payload}, sort_keys=True))
    else:
        status = "✅" if success else "❌"
        console.print(f"{status} Generated Supabase models -> {output_dir}")
        console.print(f"Schema: {schema_path}")
        if resolved_db_url:
            console.print(f"Database URL: {resolved_db_url}")
        else:
            console.print("Using local Supabase connection (--local)")

    if not success:
        raise typer.Exit(result.returncode or 1)


def _load_vertex_service() -> VertexModelService:  # pragma: no cover - runtime wiring
    from atomsAgent.dependencies import get_vertex_model_service

    return get_vertex_model_service()


@vertex_app.command("models")
@command_handler
def vertex_models(
    provider: str | None = typer.Option(
        None,
        "--provider",
        help="Filter models by provider name",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON instead of table"),
) -> None:
    service = _load_vertex_service()
    response = _run_async(service.list_models())
    models = getattr(response, "data", response)

    if provider:
        provider_lower = provider.lower()
        models = [m for m in models if getattr(m, "provider", "").lower() == provider_lower]

    serialisable = [getattr(model, "model_dump", lambda: model)() for model in models]

    if json_output:
        console.print(JSON.from_data(serialisable, indent=2))
        return

    table = Table(title="Vertex AI Models", show_lines=False)
    table.add_column("ID", overflow="fold")
    table.add_column("Provider")
    table.add_column("Capabilities")

    for item in serialisable:
        table.add_row(
            item.get("id", ""),
            item.get("provider", ""),
            ", ".join(item.get("capabilities", [])),
        )

    console.print(table)


def _load_mcp_service() -> MCPRegistryService:  # pragma: no cover - runtime wiring
    from atomsAgent.dependencies import get_mcp_service

    return get_mcp_service()


def _parse_env_pairs(pairs: Iterable[str] | None) -> dict[str, str]:
    env: dict[str, str] = {}
    if not pairs:
        return env
    for entry in pairs:
        if "=" not in entry:
            raise CommandError(f"Invalid env entry '{entry}'. Expected KEY=VALUE")
        key, value = entry.split("=", 1)
        key = key.strip()
        if not key:
            raise CommandError(f"Invalid env entry '{entry}': empty key")
        env[key] = value.strip()
    return env


def _validate_oauth_provider(provider: str | None, auth_type: str) -> None:
    """Validate OAuth provider if auth_type is oauth."""
    if auth_type == "oauth":
        if not provider:
            raise CommandError("OAuth provider is required when auth type is 'oauth'")
        if provider.lower() not in _OAUTH_PROVIDERS:
            raise CommandError(
                f"Unsupported OAuth provider '{provider}'. "
                f"Supported providers: {', '.join(sorted(_OAUTH_PROVIDERS))}"
            )


def _normalize_auth_type(value: str | None, *, allow_none: bool = False) -> AuthTypeLiteral | None:
    if value is None:
        if allow_none:
            return None
        return "none"
    normalized = value.lower()
    if normalized not in _AUTH_TYPES:
        raise CommandError("Auth type must be one of none|bearer|oauth")
    return cast(AuthTypeLiteral, normalized)


def _parse_http_url(value: str) -> HttpUrl:
    try:
        return _HTTP_URL_ADAPTER.validate_python(value)
    except ValidationError as exc:  # pragma: no cover - validation errors surfaced to user
        raise CommandError(f"Invalid endpoint URL '{value}': {exc.errors()[0]['msg']}") from exc


def _format_scope(config: MCPConfiguration) -> str:
    scope = config.scope
    if scope.type == "platform":
        return "platform"
    if scope.type == "organization" and scope.organization_id:
        return f"org:{scope.organization_id}"
    if scope.type == "user" and scope.organization_id and scope.user_id:
        return f"user:{scope.organization_id}/{scope.user_id}"
    return scope.type


@mcp_app.command("list")
@command_handler
def mcp_list(
    org: UUID = typer.Option(..., "--org", help="Organization ID"),
    user: UUID | None = typer.Option(None, "--user", help="User scope"),
    include_platform: bool = typer.Option(
        True,
        "--include-platform/--no-platform",
        help="Include platform-level MCP servers",
    ),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON output"),
) -> None:
    service = _load_mcp_service()
    response = _run_async(
        service.list(organization_id=org, user_id=user, include_platform=include_platform)
    )
    configs = response.items

    if json_output:
        console.print(JSON.from_data(configs, indent=2))
        return

    table = Table(title="MCP Servers")
    table.add_column("ID", overflow="fold")
    table.add_column("Name")
    table.add_column("Scope")
    table.add_column("Auth")
    table.add_column("Endpoint", overflow="fold")

    for config in configs:
        table.add_row(
            str(config.get("id")),
            config.get("name"),
            config.get("scope", {}).get("type", "unknown"),
            config.get("auth_type"),
            config.get("endpoint"),
        )

    console.print(table)


@mcp_app.command("create")
@command_handler
def mcp_create(
    name: str = typer.Argument(..., help="Display name"),
    endpoint: str = typer.Option(..., "--endpoint", help="HTTP endpoint for MCP server"),
    scope: str = typer.Option(
        "organization",
        "--scope",
        case_sensitive=False,
        help="Scope type",
        show_default=True,
    ),
    org: UUID | None = typer.Option(None, "--org", help="Organization ID"),
    user: UUID | None = typer.Option(None, "--user", help="User ID"),
    auth_type: str = typer.Option(
        "none",
        "--auth",
        case_sensitive=False,
        help="Authentication mode (none|bearer|oauth)",
        show_default=True,
    ),
    bearer_token: str | None = typer.Option(None, "--bearer-token", help="Bearer token"),
    oauth_provider: str | None = typer.Option(None, "--oauth-provider", help="OAuth provider"),
    arg: list[str] | None = typer.Option(None, "--arg", help="Metadata arg (repeatable)"),
    env: list[str] | None = typer.Option(None, "--env", help="Environment variable KEY=VALUE"),
    enabled: bool = typer.Option(True, "--enable/--disable", help="Enable configuration"),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON output"),
) -> None:
    scope_type = _normalize_scope(scope)
    if scope_type in {"organization", "user"} and org is None:
        raise CommandError("--org is required for organization/user scope")
    if scope_type == "user" and user is None:
        raise CommandError("--user is required for user scope")

    service = _load_mcp_service()
    metadata = MCPMetadata(args=arg or [], env=_parse_env_pairs(env))
    scope_model = MCPScope(
        type=scope_type,
        organization_id=org,
        user_id=user,
    )
    auth_normalized = _normalize_auth_type(auth_type)
    endpoint_url: HttpUrl = _parse_http_url(endpoint)

    # auth_normalized should never be None when allow_none=False
    assert auth_normalized is not None

    # Validate OAuth provider if needed
    if oauth_provider is not None:
        _validate_oauth_provider(oauth_provider, auth_normalized)

    request = MCPCreateRequest(
        name=name,
        endpoint=endpoint_url,
        auth_type=auth_normalized,
        bearer_token=bearer_token,
        oauth_provider=oauth_provider,
        metadata=metadata,
        scope=scope_model,
        enabled=enabled,
    )
    config = _run_async(service.create(request))

    if json_output:
        console.print(JSON.from_data(config.model_dump(), indent=2))
    else:
        console.print(f"✅ Created MCP server {config.name} ({config.id})")


@mcp_app.command("update")
@command_handler
def mcp_update(
    config_id: UUID = typer.Argument(..., help="Configuration ID"),
    name: str | None = typer.Option(None, "--name", help="Updated name"),
    endpoint: str | None = typer.Option(None, "--endpoint", help="HTTP endpoint"),
    auth_type: str | None = typer.Option(None, "--auth", help="Authentication mode"),
    bearer_token: str | None = typer.Option(None, "--bearer-token", help="Bearer token"),
    oauth_provider: str | None = typer.Option(None, "--oauth-provider", help="OAuth provider key"),
    arg: list[str] | None = typer.Option(None, "--arg", help="Replace metadata args"),
    env: list[str] | None = typer.Option(None, "--env", help="Replace env vars (KEY=VALUE)"),
    enable: bool | None = typer.Option(None, "--enable/--disable", help="Toggle enabled flag"),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON output"),
) -> None:
    service = _load_mcp_service()
    metadata = None
    if arg is not None or env is not None:
        metadata = MCPMetadata(args=arg or [], env=_parse_env_pairs(env))

    auth_validated: AuthTypeLiteral | None = _normalize_auth_type(auth_type, allow_none=True)
    endpoint_validated: HttpUrl | None = _parse_http_url(endpoint) if endpoint is not None else None

    # Validate OAuth provider if auth type is being updated
    if auth_validated:
        _validate_oauth_provider(oauth_provider, auth_validated)

    request = MCPUpdateRequest(
        name=name,
        endpoint=endpoint_validated,
        auth_type=auth_validated,
        bearer_token=bearer_token,
        oauth_provider=oauth_provider,
        metadata=metadata,
        enabled=enable,
    )
    config = _run_async(service.update(config_id, request))

    if json_output:
        console.print(JSON.from_data(config.model_dump(), indent=2))
    else:
        console.print(f"✅ Updated MCP server {config.name} ({config.id})")


@mcp_app.command("delete")
@command_handler
def mcp_delete(
    config_id: UUID = typer.Argument(..., help="Configuration ID"),
) -> None:
    service = _load_mcp_service()
    _run_async(service.delete(config_id))
    console.print(f"🗑️  Deleted MCP server {config_id}")


@mcp_app.command("test")
@command_handler
def mcp_test(
    config_id: UUID = typer.Argument(..., help="Configuration ID to test"),
    timeout: int = typer.Option(30, "--timeout", help="Connection timeout in seconds"),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON output"),
) -> None:
    """Test connectivity to an MCP server."""
    service = _load_mcp_service()
    config = _run_async(service.get_by_id(config_id))

    import httpx

    headers = {}
    if config.auth_type == "bearer" and config.bearer_token_id:
        # In a real implementation, you'd fetch the token from secure storage
        headers["Authorization"] = "Bearer <TOKEN>"  # Placeholder
    elif config.auth_type == "oauth" and config.oauth_provider:
        headers["Authorization"] = "Bearer <OAUTH_TOKEN>"  # Placeholder

    start_time = datetime.now()
    status = "unknown"
    error_msg = None
    response_time_ms = 0

    try:
        with httpx.Client(timeout=timeout) as client:
            response = client.get(
                str(config.endpoint) + "/health",  # Try common health endpoint
                headers=headers,
            )
            response_time_ms = int((datetime.now() - start_time).total_seconds() * 1000)
            status = "connected" if response.is_success else "error"
            if not response.is_success:
                error_msg = f"HTTP {response.status_code}: {response.text[:100]}"
    except httpx.TimeoutException:
        response_time_ms = timeout * 1000
        status = "timeout"
        error_msg = f"Connection timed out after {timeout}s"
    except Exception as exc:
        response_time_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        status = "error"
        error_msg = str(exc)

    result = {
        "config_id": str(config_id),
        "config_name": config.name,
        "endpoint": str(config.endpoint),
        "auth_type": config.auth_type,
        "status": status,
        "response_time_ms": response_time_ms,
        "error": error_msg,
        "timestamp": datetime.now().isoformat(),
    }

    if json_output:
        console.print(JSON.from_data(result, indent=2))
    else:
        status_icon = {"connected": "✅", "timeout": "⏰", "error": "❌", "unknown": "❓"}.get(
            status, "❓"
        )

        console.print(f"{status_icon} MCP Server Test Results:")
        console.print(f"  Name: {config.name}")
        console.print(f"  Endpoint: {config.endpoint}")
        console.print(f"  Status: {status}")
        console.print(f"  Response Time: {response_time_ms}ms")
        if error_msg:
            console.print(f"  Error: {error_msg}")


@mcp_app.command("export")
@command_handler
def mcp_export(
    org: UUID = typer.Option(..., "--org", help="Organization ID"),
    user: UUID | None = typer.Option(None, "--user", help="User scope"),
    format: str = typer.Option("json", "--format", help="Export format (json|yaml)"),
    output: pathlib.Path = typer.Option(None, "--output", help="Output file (default: stdout)"),
) -> None:
    """Export MCP configurations for backup or migration."""
    if format.lower() not in {"json", "yaml"}:
        raise CommandError("Format must be 'json' or 'yaml'")

    service = _load_mcp_service()
    response = _run_async(service.list(organization_id=org, user_id=user, include_platform=False))

    export_data = {
        "exported_at": datetime.now().isoformat(),
        "organization_id": str(org),
        "user_id": str(user) if user else None,
        "configurations": [config.model_dump() for config in response.items],
    }

    if format.lower() == "yaml":
        import yaml

        content = yaml.dump(export_data, default_flow_style=False, sort_keys=False)
    else:
        import json

        content = json.dumps(export_data, indent=2)

    if output:
        output.write_text(content)
        console.print(f"✅ Exported {len(response.items)} MCP configurations to {output}")
    else:
        console.print(content)


@mcp_app.command("import")
@command_handler
def mcp_import(
    file: pathlib.Path = typer.Argument(..., help="Import file (JSON or YAML)"),
    dry_run: bool = typer.Option(False, "--dry-run", help="Preview changes without applying"),
    merge_strategy: str = typer.Option(
        "skip",
        "--merge",
        help="Strategy for existing configs: skip|replace|update",
    ),
) -> None:
    """Import MCP configurations from backup or migration file."""
    if not file.exists():
        raise CommandError(f"Import file not found: {file}")

    if merge_strategy not in {"skip", "replace", "update"}:
        raise CommandError("Merge strategy must be: skip|replace|update")

    # Parse input file
    content = file.read_text()
    if file.suffix.lower() in [".yaml", ".yml"]:
        import yaml

        try:
            data = yaml.safe_load(content)
        except yaml.YAMLError as exc:
            raise CommandError(f"Invalid YAML: {exc}") from exc
    else:
        try:
            data = json.loads(content)
        except json.JSONDecodeError as exc:
            raise CommandError(f"Invalid JSON: {exc}") from exc

    # Validate structure
    if "configurations" not in data:
        raise CommandError("Import file missing 'configurations' section")

    configs_to_import = data["configurations"]
    if not isinstance(configs_to_import, list):
        raise CommandError("'configurations' must be an array")

    service = _load_mcp_service()
    results: dict[str, Any] = {
        "total": len(configs_to_import),
        "created": 0,
        "updated": 0,
        "skipped": 0,
        "errors": [],
    }

    for config_data in configs_to_import:
        try:
            # Create MCPCreateRequest from imported data
            create_request = MCPCreateRequest(**config_data)

            if dry_run:
                results["created"] += 1
                console.print(f"Would create: {create_request.name}")
                continue

            # Check if exists (by name and scope)
            org_id = create_request.scope.organization_id
            if org_id is None:
                org_id = UUID("00000000-0000-0000-0000-000000000000")
            existing = _run_async(
                service.list(
                    organization_id=org_id,
                    user_id=create_request.scope.user_id,
                    include_platform=False,
                )
            )

            existing_match = next(
                (
                    c
                    for c in existing.items
                    if c.name == create_request.name and c.scope.type == create_request.scope.type
                ),
                None,
            )

            if existing_match:
                if merge_strategy == "skip":
                    results["skipped"] += 1
                    console.print(f"⏭️  Skipped existing: {create_request.name}")
                elif merge_strategy == "replace":
                    _run_async(service.delete(existing_match.id))
                    _run_async(service.create(create_request))
                    results["updated"] += 1
                    console.print(f"🔄 Replaced: {create_request.name}")
                else:  # update
                    update_request = MCPUpdateRequest(
                        name=create_request.name,
                        endpoint=create_request.endpoint,
                        auth_type=create_request.auth_type,
                        oauth_provider=create_request.oauth_provider,
                        metadata=create_request.metadata,
                        enabled=create_request.enabled,
                    )
                    _run_async(service.update(existing_match.id, update_request))
                    results["updated"] += 1
                    console.print(f"🔄 Updated: {create_request.name}")
            else:
                _run_async(service.create(create_request))
                results["created"] += 1
                console.print(f"✅ Created: {create_request.name}")

        except Exception as exc:
            error_msg = f"Failed to import {config_data.get('name', '<unknown>')}: {exc}"
            results["errors"].append(error_msg)
            console.print(f"❌ {error_msg}")

    # Summary
    console.print("\n[bold]Import Summary[/bold]")
    console.print(f"  Total: {results['total']}")
    console.print(f"  Created: {results['created']}")
    console.print(f"  Updated: {results['updated']}")
    console.print(f"  Skipped: {results['skipped']}")
    console.print(f"  Errors: {len(results['errors'])}")

    if results["errors"] and not dry_run:
        console.print("\n[bold red]Errors occurred during import:[/bold red]")
        for error in results["errors"]:
            console.print(f"  • {error}")
        raise typer.Exit(1)


def _load_prompt_orchestrator() -> PromptOrchestrator:  # pragma: no cover - runtime wiring
    from atomsAgent.dependencies import get_prompt_orchestrator

    return get_prompt_orchestrator()


def _parse_key_values(pairs: Iterable[str] | None) -> dict[str, str]:
    return _parse_env_pairs(pairs)


@prompt_app.command("show")
@command_handler
def prompt_show(
    org: UUID | None = typer.Option(None, "--org", help="Organization scope"),
    user: UUID | None = typer.Option(None, "--user", help="User scope"),
    workflow: str | None = typer.Option(None, "--workflow", help="Workflow identifier"),
    var: list[str] | None = typer.Option(None, "--var", help="Template variable KEY=VALUE"),
    json_output: bool = typer.Option(False, "--json", help="Emit JSON output"),
) -> None:
    orchestrator = _load_prompt_orchestrator()
    variables = _parse_key_values(var)
    rendered = _run_async(
        orchestrator.compose_prompt(
            organization_id=str(org) if org else None,
            user_id=str(user) if user else None,
            workflow=workflow,
            variables=variables,
        )
    )

    records = []
    repo = getattr(orchestrator, "_prompt_repository", None)
    if repo is not None:
        records = _run_async(repo.list_prompts(organization_id=org, user_id=user))

    payload = {
        "prompt": rendered,
        "variables": variables,
        "records": [
            {
                "id": getattr(record, "id", None),
                "scope": getattr(record, "scope", None),
                "priority": getattr(record, "priority", None),
            }
            for record in records
        ],
    }

    if json_output:
        console.print(JSON.from_data(payload, indent=2))
    else:
        console.print("[bold]Resolved Prompt[/bold]")
        console.print(rendered or "<empty>")


@server_app.command("run")
def server_run(
    host: str = typer.Option("127.0.0.1", "--host", help="Bind address"),
    port: int = typer.Option(3284, "--port", help="Listen port"),
    reload: bool = typer.Option(False, "--reload/--no-reload", help="Enable auto-reload"),
    log_level: str = typer.Option("info", "--log-level", help="Uvicorn log level"),
) -> None:
    uvicorn.run("atomsAgent.main:app", host=host, port=port, reload=reload, log_level=log_level)


@app.callback(invoke_without_command=True)
def main(ctx: typer.Context) -> None:  # pragma: no cover - guard for bare invocation
    if ctx.invoked_subcommand is None:
        typer.echo(ctx.get_help())


if __name__ == "__main__":  # pragma: no cover
    app()

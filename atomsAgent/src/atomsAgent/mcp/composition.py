"""
MCP Server Composition

Dynamically compose multiple MCP servers based on user/org/project context.
"""

from __future__ import annotations

from typing import Any

from fastmcp import Client, FastMCP

from atomsAgent.mcp.supabase_client import get_supabase_client as _get_supabase_client


def get_supabase_client():
    """Return configured Supabase client or None when credentials are missing."""
    try:
        return _get_supabase_client()
    except RuntimeError:
        return None


async def get_user_mcp_servers(
    user_id: str,
    org_id: str | None = None,
    project_id: str | None = None
) -> list[dict[str, Any]]:
    """Get MCP servers configured for a user/org/project"""
    supabase = get_supabase_client()

    if supabase is None:
        return []

    filters: dict[str, str] = {
        "user_id": f"eq.{user_id}",
        "enabled": "eq.true",
    }
    if org_id:
        filters["organization_id"] = f"eq.{org_id}"

    columns = """
        id,
        server_id,
        user_id,
        organization_id,
        scope,
        enabled,
        config,
        transport_type,
        auth_status,
        server:mcp_servers (
            id,
            name,
            namespace,
            description,
            server_url,
            transport_type,
            auth_type,
            auth_config,
            stdio_config,
            proxy_config,
            scope,
            tier
        )
    """
    result = await supabase.select(
        "user_mcp_servers",
        columns=" ".join(line.strip() for line in columns.strip().splitlines()),
        filters=filters,
    )
    return result.data if result.data else []


async def create_mcp_client(server_config: dict[str, Any]) -> Client:
    """
    Create an MCP client for a remote server.
    
    Args:
        server_config: Server configuration from database
    
    Returns:
        FastMCP Client instance
    """
    server = server_config.get("server", {})
    transport_type = server.get("transport_type", "stdio")
    
    if transport_type == "stdio":
        # STDIO transport
        stdio_config = server.get("stdio_config", {})
        command = stdio_config.get("command")
        args = stdio_config.get("args", [])
        env = stdio_config.get("environmentVariables", {})
        
        if not command:
            raise ValueError(f"STDIO server {server.get('name')} missing command")
        
        return Client(command=command, args=args, env=env)
    
    elif transport_type in ("sse", "http"):
        # HTTP/SSE transport
        server_url = server.get("server_url")
        if not server_url:
            raise ValueError(f"HTTP/SSE server {server.get('name')} missing server_url")
        
        # Handle case where URL might be stored as a JSON string
        if isinstance(server_url, str):
            # Check if it's a JSON string containing url and source
            if server_url.startswith('{"url":') and '"source"' in server_url:
                try:
                    import json
                    url_obj = json.loads(server_url)
                    server_url = url_obj.get("url")
                except (json.JSONDecodeError, TypeError):
                    pass
        
        if not server_url:
            raise ValueError(f"HTTP/SSE server {server.get('name')} has invalid server_url")
        
        # Handle authentication
        auth_type = server.get("auth_type")
        headers = {}
        
        if auth_type == "bearer":
            auth_config = server.get("auth_config", {})
            bearer_token = auth_config.get("bearerToken")
            if bearer_token:
                headers["Authorization"] = f"Bearer {bearer_token}"
        
        elif auth_type == "oauth":
            # OAuth authentication - TODO: Implement
            raise NotImplementedError(
                f"OAuth authentication not yet implemented for server {server.get('name')}"
            )
        
        return Client(url=server_url, headers=headers if headers else None)
    
    else:
        raise ValueError(f"Unsupported transport type: {transport_type}")


async def compose_user_servers(
    base_mcp: FastMCP,
    user_id: str,
    org_id: str | None = None,
    project_id: str | None = None
) -> FastMCP:
    """
    Compose MCP servers for a user/org/project context.
    
    Strategy:
    - User-scoped servers: import_server() (static copy)
    - Organization-scoped servers: mount() (live link)
    - System-scoped servers: mount() (live link)
    
    Args:
        base_mcp: Base FastMCP server with built-in tools
        user_id: User ID
        org_id: Optional organization ID
        project_id: Optional project ID
    
    Returns:
        Composed FastMCP server with all tools
    """
    servers = await get_user_mcp_servers(user_id, org_id, project_id)
    
    for server_config in servers:
        server = server_config.get("server", {})
        scope = server.get("scope", "user")
        name = server.get("name", "unknown")
        
        try:
            client = await create_mcp_client(server_config)
            
            if scope == "user":
                await base_mcp.import_server(client, prefix=f"user_{name}")
            elif scope == "organization":
                await base_mcp.mount(client, prefix=f"org_{name}")
            elif scope == "system":
                await base_mcp.mount(client, prefix=f"system_{name}")
            
        except Exception as e:
            print(f"Error composing server {name}: {e}")
            continue
    
    return base_mcp

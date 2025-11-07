"""
MCP Server Database Operations

This module provides functions for fetching MCP server configurations from Supabase.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import Any

from atomsAgent.mcp.supabase_client import get_supabase_client

logger = logging.getLogger(__name__)


async def get_user_mcp_servers(user_id: str) -> list[dict[str, Any]]:
    """
    Fetch user-specific MCP servers from database.
    
    Args:
        user_id: User ID to fetch servers for
    
    Returns:
        List of MCP server configurations
    """
    try:
        supabase = get_supabase_client()

        result = await supabase.select(
            "mcp_servers",
            filters={
                "scope": "eq.user",
                "user_id": f"eq.{user_id}",
                "enabled": "eq.true",
            },
        )

        servers = result.data or []
        logger.info(f"Found {len(servers)} user MCP servers for user {user_id}")

        return servers
    except Exception as e:
        logger.error(f"Error fetching user MCP servers: {e}")
        return []


async def get_org_mcp_servers(org_id: str) -> list[dict[str, Any]]:
    """
    Fetch organization-specific MCP servers from database.
    
    Args:
        org_id: Organization ID to fetch servers for
    
    Returns:
        List of MCP server configurations
    """
    try:
        supabase = get_supabase_client()

        result = await supabase.select(
            "mcp_servers",
            filters={
                "scope": "eq.organization",
                "organization_id": f"eq.{org_id}",
                "enabled": "eq.true",
            },
        )

        servers = result.data or []
        logger.info(f"Found {len(servers)} organization MCP servers for org {org_id}")

        return servers
    except Exception as e:
        logger.error(f"Error fetching organization MCP servers: {e}")
        return []


async def get_project_mcp_servers(project_id: str) -> list[dict[str, Any]]:
    """
    Fetch project-specific MCP servers from database.

    Args:
        project_id: Project ID to fetch servers for

    Returns:
        List of MCP server configurations
    """
    try:
        supabase = get_supabase_client()

        result = await supabase.select(
            "mcp_servers",
            filters={
                "scope": "eq.project",
                "project_id": f"eq.{project_id}",
                "enabled": "eq.true",
            },
        )

        servers = result.data or []
        logger.info(f"Found {len(servers)} project MCP servers for project {project_id}")

        return servers
    except Exception as e:
        logger.error(f"Error fetching project MCP servers: {e}")
        return []


async def get_active_profile_servers(user_id: str) -> list[dict[str, Any]]:
    """
    Fetch MCP servers from user's active profile.

    This queries the mcp_profiles table and returns servers with tool-level filtering
    based on the active profile configuration.

    Args:
        user_id: User ID to fetch active profile for

    Returns:
        List of MCP server configurations with tool filtering applied
    """
    try:
        supabase = get_supabase_client()

        # Get active profile ID from user preferences
        profile_result = await supabase.select(
            "profiles",
            filters={"id": f"eq.{user_id}"},
            columns="preferences",
        )

        if not profile_result.data:
            logger.debug(f"No profile found for user {user_id}")
            return []

        preferences = profile_result.data[0].get("preferences") or {}
        active_profile_id = preferences.get("activeMcpProfileId")

        if not active_profile_id:
            logger.debug(f"No active MCP profile for user {user_id}")
            return []

        # Fetch the active MCP profile
        mcp_profile_result = await supabase.select(
            "mcp_profiles",
            filters={
                "id": f"eq.{active_profile_id}",
                "user_id": f"eq.{user_id}",
            },
        )

        if not mcp_profile_result.data:
            logger.warning(f"Active MCP profile {active_profile_id} not found for user {user_id}")
            return []

        profile = mcp_profile_result.data[0]
        servers_config = profile.get("servers") or []

        # Extract enabled server IDs
        enabled_server_configs = [s for s in servers_config if s.get("enabled")]
        server_ids = [s.get("serverId") for s in enabled_server_configs if s.get("serverId")]

        if not server_ids:
            logger.info(f"No enabled servers in active profile for user {user_id}")
            return []

        # Fetch actual server details from mcp_servers table
        # Build the filter for multiple IDs
        id_filter = ",".join(f'"{sid}"' for sid in server_ids)
        servers_result = await supabase.select(
            "mcp_servers",
            filters={"id": f"in.({id_filter})"},
        )

        servers = servers_result.data or []

        # Attach profile tool configuration to each server
        for server in servers:
            profile_server = next(
                (s for s in enabled_server_configs if s.get("serverId") == server.get("id")),
                None
            )
            if profile_server:
                # Store tool filtering config
                server["_profile_tools"] = profile_server.get("tools", [])
                server["_profile_enabled_tools"] = [
                    t.get("name") for t in profile_server.get("tools", [])
                    if t.get("enabled")
                ]

        logger.info(f"Found {len(servers)} servers from active profile for user {user_id}")
        return servers

    except Exception as e:
        logger.error(f"Error fetching active profile servers: {e}")
        return []


def convert_db_server_to_mcp_config(
    server: dict[str, Any], 
    *, 
    oauth_token: str | None = None,
    user_token: str | None = None
) -> dict[str, Any]:
    """
    Convert database MCP server record to MCP server configuration format.

    Args:
        server: Database record from mcp_servers table
        oauth_token: Optional OAuth token to use for authentication
        user_token: Optional user token (AuthKit JWT) for internal MCPs

    Returns:
        MCP server configuration dict compatible with Claude Agent SDK
    """
    transport_type = server.get("transport_type")

    if transport_type == "stdio":
        # STDIO transport: command + args
        transport_config = server.get("transport_config") or {}
        config = {
            "command": transport_config.get("command") or server.get("command"),
            "args": transport_config.get("args") or server.get("args", []),
        }

        # Add environment variables if present
        env = server.get("env", {})
        if env:
            config["env"] = env

        return config

    elif transport_type in ("http", "sse"):
        # HTTP/SSE transport: URL from 'url' field (database column name)
        server_url = server.get("url")

        # Handle case where URL might be stored as a JSON string
        if isinstance(server_url, str):
            # Check if it's a JSON string containing url and source
            if server_url.startswith('{"url":') and '"source"' in server_url:
                try:
                    import json
                    url_obj = json.loads(server_url)
                    server_url = url_obj.get("url")
                except (json.JSONDecodeError, TypeError):
                    logger.warning(f"Invalid JSON URL format for server {server.get('id')}")
                    pass

        if not server_url:
            logger.warning(f"No URL configured for HTTP/SSE server {server.get('id')}")
            return {}

        config = {
            "url": server_url,
        }

        # Check if this is an internal MCP (Atoms)
        is_internal = server.get("is_internal", False)
        server_name = server.get("name", "")
        
        # Auto-detect Atoms MCP as internal
        if "atoms-mcp" in server_name.lower() or "atoms_mcp" in server_name.lower():
            is_internal = True
        
        # If internal MCP and user token provided, use it
        if is_internal and user_token:
            logger.info(f"Using user AuthKit JWT for internal MCP: {server_name}")
            config["headers"] = {
                "Authorization": f"Bearer {user_token}"
            }
            return config

        # Add authentication from auth_config
        auth_type = server.get("auth_type")
        auth_config = server.get("auth_config") or {}

        if auth_type == "bearer":
            # Bearer token authentication
            bearer_token = auth_config.get("bearerToken") or auth_config.get("apiKey")
            if bearer_token:
                config["headers"] = {
                    "Authorization": f"Bearer {bearer_token}"
                }

        elif auth_type == "api_key":
            # API key authentication (X-API-Key header)
            api_key = auth_config.get("apiKey")
            if api_key:
                config["headers"] = {
                    "X-API-Key": api_key
                }

        elif auth_type == "oauth":
            # OAuth authentication - prefer externally provided token
            access_token = oauth_token or auth_config.get("accessToken")
            if access_token:
                config["headers"] = {"Authorization": f"Bearer {access_token}"}

        # Add any custom headers
        custom_headers = auth_config.get("customHeaders")
        if custom_headers and isinstance(custom_headers, dict):
            if "headers" not in config:
                config["headers"] = {}
            config["headers"].update(custom_headers)

        return config

    else:
        logger.warning(f"Unknown transport type: {transport_type}")
        return {}


async def update_server_usage(server_id: str) -> None:
    """
    Update server usage statistics.
    
    Args:
        server_id: MCP server ID
    """
    try:
        supabase = get_supabase_client()

        current = await supabase.select(
            "mcp_servers",
            columns="usage_count",
            filters={"id": f"eq.{server_id}"},
            limit=1,
        )
        usage_count = 0
        if current.data:
            raw_value = current.data[0].get("usage_count")
            if isinstance(raw_value, int):
                usage_count = raw_value
            elif isinstance(raw_value, str) and raw_value.isdigit():
                usage_count = int(raw_value)

        await supabase.update(
            "mcp_servers",
            filters={"id": f"eq.{server_id}"},
            payload={
                "usage_count": usage_count + 1,
                "last_used_at": datetime.now(timezone.utc).isoformat(),
            },
        )

    except Exception as e:
        logger.error(f"Error updating server usage: {e}")

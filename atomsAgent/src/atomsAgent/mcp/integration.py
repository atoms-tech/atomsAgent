"""
MCP Integration for Claude Agent SDK

This module integrates our FastMCP server with the Claude Agent SDK,
allowing Claude to use our custom tools (search_requirements, create_requirement, etc.)
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING, Any
from uuid import UUID

from atomsAgent.mcp.server import mcp as atoms_mcp_server
from atomsAgent.services.mcp_oauth import MCPOAuthError

if TYPE_CHECKING:  # pragma: no cover
    from atomsAgent.services.mcp_oauth import MCPOAuthService

logger = logging.getLogger(__name__)


def get_atoms_mcp_server_config() -> dict[str, Any]:
    """
    Get MCP server configuration for Claude Agent SDK.
    
    Returns a configuration dict that can be passed to the Claude Agent SDK's
    mcp_servers parameter.
    
    Returns:
        Dictionary with MCP server configuration
    """
    # The Claude Agent SDK expects MCP servers to be configured as either:
    # 1. Command-based: {"command": "python", "args": ["server.py"]}
    # 2. URL-based: {"url": "http://localhost:8000/mcp"}
    # 3. Direct server object (for in-process servers)
    
    # Since our FastMCP server is in-process, we can pass it directly
    return {
        "atoms-tools": atoms_mcp_server
    }


def get_default_mcp_servers() -> dict[str, Any]:
    """
    Get default MCP servers configuration including atoms-tools.
    
    This can be extended to include other default MCP servers.
    
    Returns:
        Dictionary with all default MCP server configurations
    """
    servers = {}
    
    # NOTE: We do NOT add atoms-tools here for subprocess transport
    # The atoms-tools FastMCP server is in-process and should be handled separately
    # Only add JSON-serializable server configs here
    
    # Future: Add other default servers here
    # servers["github"] = {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-github"]}
    # servers["filesystem"] = {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem"]}
    
    return servers


async def compose_mcp_servers(
    user_id: str | None = None,
    org_id: str | None = None,
    project_id: str | None = None,
    additional_servers: dict[str, Any] | None = None,
    user_token: str | None = None
) -> dict[str, Any]:
    """
    Compose MCP servers based on user/org/project context.
    
    This function will:
    1. Start with default servers (atoms-tools)
    2. Add user-specific servers from database
    3. Add org-specific servers from database
    4. Add project-specific servers from database
    5. Add any additional servers passed in
    
    Args:
        user_id: User ID to fetch user-specific servers
        org_id: Organization ID to fetch org-specific servers
        project_id: Project ID to fetch project-specific servers
        additional_servers: Additional MCP servers to include
    
    Returns:
        Dictionary with all composed MCP server configurations
    """
    from atomsAgent.mcp.database import (
        convert_db_server_to_mcp_config,
        get_active_profile_servers,
        get_org_mcp_servers,
        get_project_mcp_servers,
        get_user_mcp_servers,
    )

    oauth_service: MCPOAuthService | None = None

    def _to_uuid(value: Any) -> UUID | None:
        if not value:
            return None
        try:
            return UUID(str(value))
        except (TypeError, ValueError):
            logger.debug("Invalid UUID value %r when composing MCP servers", value)
            return None

    async def _resolve_oauth_token(record: dict[str, Any]) -> str | None:
        nonlocal oauth_service
        auth_type = (record.get("auth_type") or "").lower()
        if auth_type != "oauth":
            return None

        namespace = (
            record.get("namespace")
            or record.get("registry_namespace")
            or record.get("name")
        )
        if not namespace:
            logger.debug("Skipping OAuth token lookup for server with no namespace")
            return None

        if oauth_service is None:
            from atomsAgent.dependencies import (
                get_mcp_oauth_service,  # local import to avoid circular
            )

            oauth_service = get_mcp_oauth_service()

        # Prefer the record-specific scope, fall back to the compose scope
        record_user = record.get("user_id") or user_id
        record_org = record.get("organization_id") or org_id

        user_uuid = _to_uuid(record_user)
        org_uuid = _to_uuid(record_org)

        token_record = None
        try:
            if user_uuid is not None:
                token_record = await oauth_service.latest_tokens_for_namespace(
                    mcp_namespace=namespace,
                    user_id=user_uuid,
                )
            if token_record is None and org_uuid is not None:
                token_record = await oauth_service.latest_tokens_for_namespace(
                    mcp_namespace=namespace,
                    organization_id=org_uuid,
                )
        except MCPOAuthError as exc:
            logger.warning(
                "Failed to load OAuth token for namespace %s: %s",
                namespace,
                exc,
            )
            return None

        if token_record and token_record.access_token:
            return token_record.access_token

        logger.debug(
            "No stored OAuth token found for namespace %s (user=%s, org=%s)",
            namespace,
            user_uuid,
            org_uuid,
        )
        return None
    
    # Start with default servers
    servers = get_default_mcp_servers()
    
    # Fetch and add user-specific servers
    if user_id:
        logger.info(f"Composing MCP servers for user: {user_id}")
        try:
            # Try to get servers from active profile first
            user_servers_db = await get_active_profile_servers(user_id)

            if user_servers_db:
                logger.info(f"Using {len(user_servers_db)} servers from active MCP profile")
            else:
                # Fallback to all enabled user servers if no profile
                logger.debug("No active profile found, falling back to all user servers")
                user_servers_db = await get_user_mcp_servers(user_id)

            for server_record in user_servers_db:
                server_name = f"user_{server_record['name']}"
                oauth_token = await _resolve_oauth_token(server_record)
                server_config = convert_db_server_to_mcp_config(
                    server_record,
                    oauth_token=oauth_token,
                    user_token=user_token,
                )
                if server_config:
                    servers[server_name] = server_config
                    logger.debug(f"Added user server: {server_name}")
        except Exception as e:
            message = str(e)
            if "Supabase credentials not configured" in message:
                logger.debug("Supabase not configured; skipping user MCP servers")
            else:
                logger.error(f"Error loading user MCP servers: {message}")
    
    # Fetch and add organization-specific servers
    if org_id:
        logger.info(f"Composing MCP servers for org: {org_id}")
        try:
            org_servers_db = await get_org_mcp_servers(org_id)
            for server_record in org_servers_db:
                server_name = f"org_{server_record['name']}"
                oauth_token = await _resolve_oauth_token(server_record)
                server_config = convert_db_server_to_mcp_config(
                    server_record,
                    oauth_token=oauth_token,
                    user_token=user_token,
                )
                if server_config:
                    servers[server_name] = server_config
                    logger.debug(f"Added org server: {server_name}")
        except Exception as e:
            message = str(e)
            if "Supabase credentials not configured" in message:
                logger.debug("Supabase not configured; skipping org MCP servers")
            else:
                logger.error(f"Error loading org MCP servers: {message}")
    
    # Fetch and add project-specific servers
    if project_id:
        logger.info(f"Composing MCP servers for project: {project_id}")
        try:
            project_servers_db = await get_project_mcp_servers(project_id)
            for server_record in project_servers_db:
                server_name = f"proj_{server_record['name']}"
                oauth_token = await _resolve_oauth_token(server_record)
                server_config = convert_db_server_to_mcp_config(
                    server_record,
                    oauth_token=oauth_token,
                    user_token=user_token,
                )
                if server_config:
                    servers[server_name] = server_config
                    logger.debug(f"Added project server: {server_name}")
        except Exception as e:
            message = str(e)
            if "Supabase credentials not configured" in message:
                logger.debug("Supabase not configured; skipping project MCP servers")
            else:
                logger.error(f"Error loading project MCP servers: {message}")
    
    # Add additional servers
    if additional_servers:
        servers.update(additional_servers)
    
    logger.info(f"Composed {len(servers)} MCP servers: {list(servers.keys())}")
    
    return servers


# Export for easy importing
__all__ = [
    "compose_mcp_servers",
    "get_atoms_mcp_server_config",
    "get_default_mcp_servers",
]

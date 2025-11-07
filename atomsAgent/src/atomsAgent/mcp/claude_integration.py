"""
Claude Integration with MCP Composition

Integrates the composed MCP servers with Claude Agent SDK.
"""

from __future__ import annotations

from typing import Any

from fastmcp import FastMCP

from atomsAgent.mcp.integration import compose_mcp_servers
from atomsAgent.mcp.server import mcp as base_mcp


async def get_composed_mcp_for_user(
    user_id: str,
    org_id: str | None = None,
    project_id: str | None = None
) -> FastMCP:
    """
    Get composed MCP server for a user with all their configured tools.
    
    Args:
        user_id: User ID
        org_id: Optional organization ID
        project_id: Optional project ID
    
    Returns:
        Composed FastMCP server with all tools
    """
    # Legacy helper retained for backward compatibility with tests.
    # Prefer compose_mcp_servers for new integrations.
    from atomsAgent.mcp.composition import compose_user_servers

    return await compose_user_servers(
        base_mcp,
        user_id=user_id,
        org_id=org_id,
        project_id=project_id,
    )


async def get_mcp_servers_dict(
    user_id: str,
    org_id: str | None = None,
    project_id: str | None = None,
    custom_servers: dict[str, Any] | None = None,
    user_token: str | None = None
) -> dict[str, FastMCP]:
    """
    Get MCP servers dictionary for Claude SDK.
    
    This function composes all MCP servers for a user and returns them
    in the format expected by Claude Agent SDK.
    
    Args:
        user_id: User ID
        org_id: Optional organization ID
        project_id: Optional project ID
        custom_servers: Optional additional custom servers
    
    Returns:
        Dictionary of MCP servers for Claude SDK
    """
    servers = await compose_mcp_servers(
        user_id=user_id,
        org_id=org_id,
        project_id=project_id,
        additional_servers=custom_servers,
        user_token=user_token,
    )

    return servers

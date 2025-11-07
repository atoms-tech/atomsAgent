"""
MCP (Model Context Protocol) Integration for atomsAgent

This module provides FastMCP server and client implementations for atomsAgent.
"""

from atomsAgent.mcp.integration import (
    compose_mcp_servers,
    get_atoms_mcp_server_config,
    get_default_mcp_servers,
)
from atomsAgent.mcp.server import mcp

__all__ = [
    "compose_mcp_servers",
    "get_atoms_mcp_server_config",
    "get_default_mcp_servers",
    "mcp",
]

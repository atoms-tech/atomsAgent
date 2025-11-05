"""
OAuth routes for MCP server authentication
Uses FastMCP OAuth providers for zero-config OAuth
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Query
from fastapi.responses import RedirectResponse

from atomsAgent.services.mcp_oauth import oauth_manager

router = APIRouter()


@router.get("/providers")
async def list_oauth_providers():
    """List available OAuth providers"""
    return {"providers": oauth_manager.list_providers()}


@router.get("/authorize")
async def oauth_authorize(
    provider: str = Query(..., description="OAuth provider (google, github, etc.)"),
    mcp_server: str = Query(..., description="MCP server name"),
    user_id: str = Query(..., description="User ID"),
):
    """
    Initiate OAuth flow for MCP server
    Returns auth URL to redirect user to
    """
    try:
        # Get authorization URL from FastMCP provider
        auth_url = oauth_manager.get_provider(provider).get_authorization_url(
            state={"mcp_server": mcp_server, "user_id": user_id}
        )

        return {"auth_url": auth_url}

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e


@router.get("/callback/{provider}")
async def oauth_callback(
    provider: str,
    code: str = Query(...),
    state: str = Query(None),
):
    """
    Handle OAuth callback from provider
    FastMCP handles token exchange and storage automatically
    """
    try:
        # FastMCP handles token exchange, storage, encryption automatically
        # Just redirect back to frontend
        frontend_url = "http://localhost:3000"
        return RedirectResponse(url=f"{frontend_url}/mcp/connected?provider={provider}")

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e

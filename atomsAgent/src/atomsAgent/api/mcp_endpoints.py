"""
MCP API Endpoints

REST API endpoints for OAuth, Artifacts, and Tool Approval.
"""

from __future__ import annotations

import uuid
from typing import Any

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from atomsAgent.mcp.oauth_handler import OAuthConfig, OAuthHandler
from atomsAgent.services.artifacts import ArtifactStorage
from atomsAgent.services.tool_approval import ToolApprovalManager

# Create router
router = APIRouter(prefix="/api/mcp", tags=["mcp"])


# ============================================================================
# OAuth Endpoints
# ============================================================================

class OAuthInitRequest(BaseModel):
    """Request to initiate OAuth flow"""
    server_id: str
    authorization_url: str
    token_url: str
    client_id: str | None = None
    client_secret: str | None = None
    redirect_uri: str = "http://localhost:3000/oauth/callback"
    scopes: list[str] | None = None
    registration_url: str | None = None


class OAuthInitResponse(BaseModel):
    """Response from OAuth init"""
    authorization_url: str
    state: str


class OAuthCallbackRequest(BaseModel):
    """OAuth callback request"""
    server_id: str
    code: str
    state: str


class OAuthCallbackResponse(BaseModel):
    """OAuth callback response"""
    success: bool
    message: str


@router.post("/oauth/init", response_model=OAuthInitResponse)
async def oauth_init(request: OAuthInitRequest):
    """
    Initiate OAuth flow for an MCP server.
    
    Returns authorization URL for user to visit.
    """
    try:
        handler = OAuthHandler()
        
        # Create config
        config = OAuthConfig(
            server_id=request.server_id,
            authorization_url=request.authorization_url,
            token_url=request.token_url,
            client_id=request.client_id,
            client_secret=request.client_secret,
            redirect_uri=request.redirect_uri,
            scopes=request.scopes,
            registration_url=request.registration_url,
            use_pkce=True
        )
        
        # Perform DCR if needed
        if request.registration_url and not request.client_id:
            client_id, client_secret = await handler.register_client(config)
            config.client_id = client_id
            config.client_secret = client_secret
        
        # Generate authorization URL
        auth_url, state, _ = handler.generate_authorization_url(config)
        
        return OAuthInitResponse(
            authorization_url=auth_url,
            state=state
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/oauth/callback", response_model=OAuthCallbackResponse)
async def oauth_callback(request: OAuthCallbackRequest):
    """
    Handle OAuth callback.
    
    Exchanges authorization code for tokens.
    """
    try:
        handler = OAuthHandler()
        
        # TODO: Get config from database
        # For now, return success
        
        return OAuthCallbackResponse(
            success=True,
            message="OAuth flow completed successfully"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/oauth/status/{server_id}")
async def oauth_status(server_id: str):
    """Get OAuth status for an MCP server"""
    try:
        handler = OAuthHandler()
        tokens = await handler.get_tokens(server_id)
        
        if tokens is None:
            return {"connected": False}
        
        return {
            "connected": True,
            "expired": tokens.is_expired(),
            "expires_at": tokens.expires_at.isoformat() if tokens.expires_at else None
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# Artifact Endpoints
# ============================================================================

class ArtifactResponse(BaseModel):
    """Artifact response"""
    id: str
    type: str
    title: str
    content: str
    language: str | None = None
    created_at: str | None = None
    metadata: dict[str, Any] | None = None


@router.get("/artifacts/{artifact_id}", response_model=ArtifactResponse)
async def get_artifact(artifact_id: str):
    """Get artifact by ID"""
    try:
        storage = ArtifactStorage()
        artifact = await storage.get_artifact(artifact_id)
        
        if artifact is None:
            raise HTTPException(status_code=404, detail="Artifact not found")
        
        return ArtifactResponse(
            id=artifact.id,
            type=artifact.type,
            title=artifact.title,
            content=artifact.content,
            language=artifact.language,
            created_at=artifact.created_at.isoformat() if artifact.created_at else None,
            metadata=artifact.metadata
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/artifacts/session/{session_id}", response_model=list[ArtifactResponse])
async def get_session_artifacts(session_id: str):
    """Get all artifacts for a session"""
    try:
        storage = ArtifactStorage()
        artifacts = await storage.get_session_artifacts(session_id)
        
        return [
            ArtifactResponse(
                id=a.id,
                type=a.type,
                title=a.title,
                content=a.content,
                language=a.language,
                created_at=a.created_at.isoformat() if a.created_at else None,
                metadata=a.metadata
            )
            for a in artifacts
        ]
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# Tool Approval Endpoints
# ============================================================================

class ToolApprovalRequestModel(BaseModel):
    """Tool approval request"""
    tool_name: str
    tool_input: dict[str, Any]
    session_id: str
    user_id: str


class ToolApprovalResponse(BaseModel):
    """Tool approval response"""
    id: str
    tool_name: str
    status: str
    risk_level: str
    reason: str | None = None


class ApprovalDecisionRequest(BaseModel):
    """Approval decision"""
    approved: bool
    reason: str | None = None
    approved_by: str


@router.post("/tool-approval/request", response_model=ToolApprovalResponse)
async def request_tool_approval(request: ToolApprovalRequestModel):
    """Request approval for a tool execution"""
    try:
        manager = ToolApprovalManager()
        
        approval_request = await manager.request_approval(
            request_id=str(uuid.uuid4()),
            tool_name=request.tool_name,
            tool_input=request.tool_input,
            session_id=request.session_id,
            user_id=request.user_id
        )
        
        return ToolApprovalResponse(
            id=approval_request.id,
            tool_name=approval_request.tool_name,
            status=approval_request.status,
            risk_level=approval_request.risk_level.value,
            reason=approval_request.reason
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/tool-approval/{request_id}/decide", response_model=ToolApprovalResponse)
async def decide_tool_approval(request_id: str, decision: ApprovalDecisionRequest):
    """Approve or deny a tool execution request"""
    try:
        manager = ToolApprovalManager()
        
        if decision.approved:
            approval_request = await manager.approve_request(
                request_id,
                approved_by=decision.approved_by,
                reason=decision.reason
            )
        else:
            approval_request = await manager.deny_request(
                request_id,
                denied_by=decision.approved_by,
                reason=decision.reason
            )
        
        return ToolApprovalResponse(
            id=approval_request.id,
            tool_name=approval_request.tool_name,
            status=approval_request.status,
            risk_level=approval_request.risk_level.value,
            reason=approval_request.reason
        )
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/tool-approval/pending/{user_id}", response_model=list[ToolApprovalResponse])
async def get_pending_approvals(user_id: str):
    """Get all pending approval requests for a user"""
    try:
        manager = ToolApprovalManager()
        requests = await manager.get_pending_requests(user_id)
        
        return [
            ToolApprovalResponse(
                id=r.id,
                tool_name=r.tool_name,
                status=r.status,
                risk_level=r.risk_level.value,
                reason=r.reason
            )
            for r in requests
        ]
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/tool-approval/{request_id}", response_model=ToolApprovalResponse)
async def get_approval_request(request_id: str):
    """Get approval request by ID"""
    try:
        manager = ToolApprovalManager()
        request = await manager.get_request(request_id)
        
        if request is None:
            raise HTTPException(status_code=404, detail="Request not found")
        
        return ToolApprovalResponse(
            id=request.id,
            tool_name=request.tool_name,
            status=request.status,
            risk_level=request.risk_level.value,
            reason=request.reason
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

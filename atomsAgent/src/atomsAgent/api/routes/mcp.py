from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Path, Query, status

from atomsAgent.dependencies import get_mcp_service
from atomsAgent.schemas.mcp import (
    MCPConfiguration,
    MCPCreateRequest,
    MCPListResponse,
    MCPUpdateRequest,
)
from atomsAgent.services import MCPRegistryService

router = APIRouter()


@router.get("", response_model=MCPListResponse)
async def list_mcp_servers(
    organization_id: UUID = Query(..., description="Organization context"),
    user_id: UUID | None = Query(None, description="User context if applicable"),
    include_platform: bool = Query(True, description="Include platform-level MCPs"),
    service: MCPRegistryService = Depends(get_mcp_service),
) -> MCPListResponse:
    return await service.list(
        organization_id=organization_id,
        user_id=user_id,
        include_platform=include_platform,
    )


@router.post("", response_model=MCPConfiguration, status_code=status.HTTP_201_CREATED)
async def create_mcp_server(
    payload: MCPCreateRequest,
    service: MCPRegistryService = Depends(get_mcp_service),
) -> MCPConfiguration:
    try:
        return await service.create(payload)
    except ValueError as exc:  # invalid scope
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc


@router.put("/{mcp_id}", response_model=MCPConfiguration)
async def update_mcp_server(
    payload: MCPUpdateRequest,
    mcp_id: UUID = Path(..., description="MCP configuration identifier"),
    service: MCPRegistryService = Depends(get_mcp_service),
) -> MCPConfiguration:
    try:
        return await service.update(mcp_id, payload)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc


@router.delete("/{mcp_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_mcp_server(
    mcp_id: UUID,
    service: MCPRegistryService = Depends(get_mcp_service),
) -> None:
    await service.delete(mcp_id)

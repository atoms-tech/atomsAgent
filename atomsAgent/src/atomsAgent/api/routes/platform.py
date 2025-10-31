from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Path, Query, status

from atomsAgent.dependencies import get_platform_service
from atomsAgent.schemas.platform import (
    AddAdminRequest,
    AdminListResponse,
    AdminResponse,
    AuditLogResponse,
    PlatformStats,
)
from atomsAgent.services import PlatformService

router = APIRouter()


@router.get("/stats", response_model=PlatformStats)
async def get_platform_stats(
    service: PlatformService = Depends(get_platform_service),
) -> PlatformStats:
    return await service.get_stats()


@router.get("/admins", response_model=AdminListResponse)
async def list_platform_admins(
    service: PlatformService = Depends(get_platform_service),
) -> AdminListResponse:
    return await service.list_admins()


@router.post("/admins", response_model=AdminResponse, status_code=status.HTTP_201_CREATED)
async def create_platform_admin(
    request: AddAdminRequest,
    service: PlatformService = Depends(get_platform_service),
) -> AdminResponse:
    if not request.email or not request.workos_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST, detail="email and workos_id required"
        )
    return await service.add_admin(request, created_by=request.workos_id)


@router.delete("/admins/{email}", response_model=AdminResponse)
async def delete_platform_admin(
    email: str = Path(..., description="Admin email"),
    service: PlatformService = Depends(get_platform_service),
) -> AdminResponse:
    return await service.remove_admin(email)


@router.get("/audit", response_model=AuditLogResponse)
async def get_audit_logs(
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    service: PlatformService = Depends(get_platform_service),
) -> AuditLogResponse:
    return await service.list_audit(limit=limit, offset=offset)

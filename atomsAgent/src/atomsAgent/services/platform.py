from __future__ import annotations

import json
from typing import Any

from atomsAgent.db.repositories import (
    AdminRecord,
    AuditLogRecord,
    PlatformRepository,
)
from atomsAgent.schemas.platform import (
    AddAdminRequest,
    AdminInfo,
    AdminListResponse,
    AdminResponse,
    AuditEntry,
    AuditLogResponse,
    PlatformStats,
    SystemHealth,
)


class PlatformService:
    def __init__(self, repository: PlatformRepository) -> None:
        self._repository = repository

    async def get_stats(self) -> PlatformStats:
        record = await self._repository.get_stats()
        return PlatformStats(
            total_users=record.total_users,
            active_users=record.active_users,
            total_organizations=record.total_organizations,
            total_requests=record.total_requests,
            requests_today=record.requests_today,
            total_tokens=record.total_tokens,
            tokens_today=record.tokens_today,
            total_mcp_servers=record.total_mcp_servers,
            system_health=SystemHealth(
                status="healthy",
                circuit_breaker_status=record.circuit_breaker_status,
                active_agents=record.active_agents,
            ),
        )

    async def list_admins(self) -> AdminListResponse:
        records = await self._repository.list_admins()
        return AdminListResponse(admins=[self._map_admin(r) for r in records], count=len(records))

    async def add_admin(self, request: AddAdminRequest, created_by: str) -> AdminResponse:
        record = await self._repository.add_admin(
            {
                "workos_id": request.workos_id,
                "email": request.email,
                "name": request.name,
                "created_by": created_by,
            }
        )
        return AdminResponse(email=record.email)

    async def remove_admin(self, email: str) -> AdminResponse:
        await self._repository.remove_admin(email)
        return AdminResponse(email=email)

    async def list_audit(self, *, limit: int = 50, offset: int = 0) -> AuditLogResponse:
        records = await self._repository.list_audit_logs(limit=limit, offset=offset)
        return AuditLogResponse(
            entries=[self._map_audit(r) for r in records],
            count=len(records),
            limit=limit,
            offset=offset,
        )

    @staticmethod
    def _map_admin(record: AdminRecord) -> AdminInfo:
        return AdminInfo(
            id=record.id,
            email=record.email,
            name=record.name,
            created_at=record.created_at,
            created_by=record.created_by,
        )

    @staticmethod
    def _map_audit(record: AuditLogRecord) -> AuditEntry:
        details: dict[str, Any] = {}
        if record.details:
            if isinstance(record.details, dict):
                details = record.details  # type: ignore[assignment]
            else:
                try:
                    details = json.loads(record.details)
                except Exception:
                    details = {}
        return AuditEntry(
            id=record.id,
            timestamp=record.timestamp,
            user_id=details.get("user_id"),
            org_id=details.get("organization_id"),
            action=record.action,
            resource=record.resource_type,
            resource_id=record.resource_id,
            metadata=details,
        )

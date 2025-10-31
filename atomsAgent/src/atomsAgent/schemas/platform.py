from __future__ import annotations

from pydantic import BaseModel, Field


class SystemHealth(BaseModel):
    status: str = "healthy"
    circuit_breaker_status: str | None = None
    active_agents: list[str] = Field(default_factory=list)


class PlatformStats(BaseModel):
    total_users: int = 0
    active_users: int = 0
    total_organizations: int = 0
    total_requests: int = 0
    requests_today: int = 0
    total_tokens: int = 0
    tokens_today: int = 0
    total_mcp_servers: int = 0
    system_health: SystemHealth = Field(default_factory=SystemHealth)


class AdminInfo(BaseModel):
    id: str
    email: str
    name: str | None = None
    created_at: str
    created_by: str | None = None


class AuditEntry(BaseModel):
    id: str
    timestamp: str
    user_id: str | None = None
    org_id: str | None = None
    action: str
    resource: str | None = None
    resource_id: str | None = None
    metadata: dict = Field(default_factory=dict)


class AuditLogResponse(BaseModel):
    entries: list[AuditEntry]
    count: int
    limit: int
    offset: int


class AdminListResponse(BaseModel):
    admins: list[AdminInfo]
    count: int


class AddAdminRequest(BaseModel):
    workos_id: str
    email: str
    name: str | None = None


class AdminResponse(BaseModel):
    status: str = "success"
    email: str

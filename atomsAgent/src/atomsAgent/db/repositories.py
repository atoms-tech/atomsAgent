from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from atomsAgent.db.supabase import SupabaseClient


@dataclass
class PromptRecord:
    id: str
    content: str
    priority: int
    scope: str
    organization_id: str | None
    user_id: str | None
    template: str | None


class PromptRepository:
    """Data access helpers for system prompts."""

    def __init__(self, client: SupabaseClient) -> None:
        self._client = client

    async def list_prompts(
        self,
        *,
        organization_id: UUID | None,
        user_id: UUID | None,
    ) -> list[PromptRecord]:
        response = await self._client.select(
            "system_prompts",
            columns="id,content,priority,scope,organization_id,user_id,template,enabled",
            filters={"enabled": "eq.true"},
            order=["priority.desc"],
        )
        records: list[PromptRecord] = []
        org_id_str = str(organization_id) if organization_id else None
        user_id_str = str(user_id) if user_id else None
        for row in response.data:
            scope = row.get("scope")
            if scope == "global":
                records.append(_prompt_record_from_row(row))
            elif (
                scope == "organization" and org_id_str and row.get("organization_id") == org_id_str
            ):
                records.append(_prompt_record_from_row(row))
            elif (
                scope == "user"
                and org_id_str
                and user_id_str
                and row.get("organization_id") == org_id_str
                and row.get("user_id") == user_id_str
            ):
                records.append(_prompt_record_from_row(row))
        return records


@dataclass
class MCPConfigRecord:
    id: str
    org_id: str | None  # Changed from organization_id to match DB
    user_id: str | None
    name: str
    type: str
    endpoint: str | None
    auth_type: str
    auth_token: str | None  # Changed from bearer_token to match DB
    auth_header: str | None  # New field
    config: str | None  # New field (JSON config)
    scope: str | None  # New field (org/user scope)
    description: str | None  # New field
    created_at: str | None
    updated_at: str | None  # New field
    created_by: str | None  # New field
    updated_by: str | None  # New field
    enabled: bool


class MCPRepository:
    """Data access helpers for MCP configurations."""

    def __init__(self, client: SupabaseClient) -> None:
        self._client = client

    async def list_configs(
        self,
        *,
        organization_id: UUID,
        user_id: UUID | None,
        include_platform: bool = True,
    ) -> list[MCPConfigRecord]:
        # Get org-specific configs first
        org_id_str = str(organization_id)
        org_filters = {"enabled": "eq.true", "org_id": f"eq.{org_id_str}"}
        response = await self._client.select(
            "mcp_configurations",
            columns="id,org_id,user_id,name,type,endpoint,auth_type,auth_token,auth_header,config,scope,enabled,description,created_at,updated_at,created_by,updated_by",
            filters=org_filters,
        )
        configs: list[MCPConfigRecord] = []
        user_id_str = str(user_id) if user_id else None
        for row in response.data:
            row_org_id = row.get("org_id")
            row_user_id = row.get("user_id")
            if row_org_id == org_id_str and row_user_id is None:
                configs.append(_mcp_record_from_row(row))
            elif user_id_str and row_org_id == org_id_str and row_user_id == user_id_str:
                configs.append(_mcp_record_from_row(row))

        # If include_platform, get platform configs separately
        if include_platform:
            platform_filters = {"enabled": "eq.true", "org_id": "is.null"}
            platform_response = await self._client.select(
                "mcp_configurations",
                columns="id,org_id,user_id,name,type,endpoint,auth_type,auth_token,auth_header,config,scope,enabled,description,created_at,updated_at,created_by,updated_by",
                filters=platform_filters,
            )
            for row in platform_response.data:
                if row.get("org_id") is None:
                    configs.append(_mcp_record_from_row(row))

        return configs

    async def get_config(self, config_id: UUID) -> MCPConfigRecord:
        response = await self._client.select(
            "mcp_configurations",
            columns="id,org_id,user_id,name,type,endpoint,auth_type,auth_token,auth_header,config,scope,enabled,description,created_at,updated_at,created_by,updated_by",
            filters={"id": f"eq.{config_id}"},
        )
        if not response.data:
            raise ValueError(f"MCP configuration not found: {config_id}")
        return _mcp_record_from_row(response.data[0])

    async def create_config(self, payload: dict[str, Any]) -> MCPConfigRecord:
        response = await self._client.insert("mcp_configurations", payload)
        return _mcp_record_from_row(response.data[0])

    async def update_config(self, config_id: UUID, payload: dict[str, Any]) -> MCPConfigRecord:
        response = await self._client.update(
            "mcp_configurations",
            filters={"id": f"eq.{config_id}"},
            payload=payload,
        )
        return _mcp_record_from_row(response.data[0])

    async def delete_config(self, config_id: UUID) -> None:
        await self._client.delete(
            "mcp_configurations",
            filters={"id": f"eq.{config_id}"},
        )


@dataclass
class PlatformStatsRecord:
    total_users: int
    active_users: int
    total_organizations: int
    total_requests: int
    requests_today: int
    total_tokens: int
    tokens_today: int
    total_mcp_servers: int
    active_agents: list[str]
    circuit_breaker_status: str | None


@dataclass
class AdminRecord:
    id: str
    email: str
    name: str | None
    created_at: str
    created_by: str | None
    workos_id: str | None = None


@dataclass
class AuditLogRecord:
    id: str
    timestamp: str
    action: str
    resource_type: str
    resource_id: str | None
    details: dict[str, Any]
    success: bool


class PlatformRepository:
    def __init__(self, client: SupabaseClient) -> None:
        self._client = client

    async def get_stats(self) -> PlatformStatsRecord:
        orgs = await self._client.select("organizations", columns="id", count=True)
        users = await self._client.select("users", columns="id", count=True)
        sessions = await self._client.select("user_sessions", columns="id", count=True)
        mcps = await self._client.select("mcp_configurations", columns="id", count=True)
        return PlatformStatsRecord(
            total_organizations=orgs.count or 0,
            total_users=users.count or 0,
            active_users=sessions.count or 0,
            total_requests=0,
            requests_today=0,
            total_tokens=0,
            tokens_today=0,
            total_mcp_servers=mcps.count or 0,
            active_agents=["claude", "gemini"],
            circuit_breaker_status="healthy",
        )

    async def list_admins(self) -> list[AdminRecord]:
        response = await self._client.select(
            "platform_admins",
            columns="id,workos_id,email,name,created_at,created_by",
            order=["created_at.desc"],
        )
        return [AdminRecord(**row) for row in response.data]

    async def add_admin(self, payload: dict[str, Any]) -> AdminRecord:
        response = await self._client.insert("platform_admins", payload)
        return AdminRecord(**response.data[0])

    async def remove_admin(self, email: str) -> None:
        await self._client.delete("platform_admins", filters={"email": f"eq.{email}"})

    async def list_audit_logs(self, limit: int = 50, offset: int = 0) -> list[AuditLogRecord]:
        response = await self._client.select(
            "audit_logs",
            columns="id,timestamp,action,resource_type,resource_id,details,success",
            order=["timestamp.desc"],
            limit=limit,
            offset=offset,
        )
        records: list[AuditLogRecord] = []
        for row in response.data:
            details: dict[str, Any] = {}
            raw_details = row.get("details")
            if isinstance(raw_details, dict):
                details = raw_details  # type: ignore[assignment]
            elif isinstance(raw_details, str):
                try:
                    details = json.loads(raw_details)
                except Exception:
                    details = {}
            row["details"] = details
            records.append(AuditLogRecord(**row))
        return records


@dataclass
class ChatSessionRecord:
    id: str
    user_id: str
    organization_id: str | None
    title: str | None
    model: str | None
    agent_type: str | None
    created_at: str
    updated_at: str
    last_message_at: str | None
    message_count: int
    tokens_in: int
    tokens_out: int
    tokens_total: int
    metadata: dict[str, Any]
    archived: bool


@dataclass
class ChatMessageRecord:
    id: str
    session_id: str
    message_index: int
    role: str
    content: str
    metadata: dict[str, Any]
    tokens: int | None
    created_at: str
    updated_at: str | None


class ChatHistoryRepository:
    """Data access helpers for chat history persistence."""

    def __init__(self, client: SupabaseClient) -> None:
        self._client = client

    async def fetch_session(self, session_id: str) -> ChatSessionRecord | None:
        response = await self._client.select(
            "chat_sessions",
            columns="id,user_id,org_id,title,model,agent_type,created_at,updated_at,last_message_at,message_count,tokens_in,tokens_out,tokens_total,metadata,archived",
            filters={"id": f"eq.{session_id}"},
        )
        if not response.data:
            return None
        return _chat_session_from_row(response.data[0])

    async def fetch_session_for_user(
        self, session_id: str, user_id: str
    ) -> ChatSessionRecord | None:
        response = await self._client.select(
            "chat_sessions",
            columns="id,user_id,org_id,title,model,agent_type,created_at,updated_at,last_message_at,message_count,tokens_in,tokens_out,tokens_total,metadata,archived",
            filters={"id": f"eq.{session_id}", "user_id": f"eq.{user_id}"},
        )
        if not response.data:
            return None
        return _chat_session_from_row(response.data[0])

    async def create_session(self, payload: dict[str, Any]) -> ChatSessionRecord:
        response = await self._client.insert("chat_sessions", payload)
        return _chat_session_from_row(response.data[0])

    async def update_session(self, session_id: str, payload: dict[str, Any]) -> ChatSessionRecord:
        response = await self._client.update(
            "chat_sessions",
            filters={"id": f"eq.{session_id}"},
            payload=payload,
        )
        if not response.data:
            raise ValueError(f"chat session not found: {session_id}")
        return _chat_session_from_row(response.data[0])

    async def list_sessions(
        self, user_id: str, *, limit: int, offset: int
    ) -> tuple[list[ChatSessionRecord], int]:
        response = await self._client.select(
            "chat_sessions",
            columns="id,user_id,org_id,title,model,agent_type,created_at,updated_at,last_message_at,message_count,tokens_in,tokens_out,tokens_total,metadata,archived",
            filters={"user_id": f"eq.{user_id}", "archived": "eq.false"},
            order=["last_message_at.desc.nullslast", "created_at.desc"],
            limit=limit,
            offset=offset,
            count=True,
        )
        sessions = [_chat_session_from_row(row) for row in response.data]
        return sessions, response.count or 0

    async def insert_message(self, payload: dict[str, Any]) -> ChatMessageRecord:
        response = await self._client.insert("chat_messages", payload)
        return _chat_message_from_row(response.data[0])

    async def fetch_messages(self, session_id: str) -> list[ChatMessageRecord]:
        response = await self._client.select(
            "chat_messages",
            columns="id,session_id,message_index,role,content,metadata,tokens_in,tokens_out,tokens_total,created_at,updated_at",
            filters={"session_id": f"eq.{session_id}"},
            order=["message_index.asc"],
        )
        return [_chat_message_from_row(row) for row in response.data]


def _prompt_record_from_row(row: dict[str, Any]) -> PromptRecord:
    prompt_id = row.get("id")
    if prompt_id is None:
        raise ValueError("prompt row missing id")
    if not isinstance(prompt_id, str):
        prompt_id = str(prompt_id)

    content = row.get("content", "")
    if not isinstance(content, str):
        content = str(content)

    priority_raw = row.get("priority", 0)
    priority = int(priority_raw) if not isinstance(priority_raw, int) else priority_raw

    scope_value = row.get("scope", "global")
    if not isinstance(scope_value, str):
        scope_value = str(scope_value)

    organization_id = row.get("organization_id")
    if organization_id is not None and not isinstance(organization_id, str):
        organization_id = str(organization_id)

    user_id = row.get("user_id")
    if user_id is not None and not isinstance(user_id, str):
        user_id = str(user_id)

    template_value = row.get("template")
    if template_value is not None and not isinstance(template_value, str):
        template_value = str(template_value)

    return PromptRecord(
        id=prompt_id,
        content=content,
        priority=priority,
        scope=scope_value,
        organization_id=organization_id,
        user_id=user_id,
        template=template_value,
    )


def _mcp_record_from_row(row: dict[str, Any]) -> MCPConfigRecord:
    record_id = row.get("id")
    if record_id is None:
        raise ValueError("MCP configuration row missing id")
    record_id_str = str(record_id)

    # Map org_id from database to org_id field
    org_id = row.get("org_id")
    if org_id is not None and not isinstance(org_id, str):
        org_id = str(org_id)

    user_id = row.get("user_id")
    if user_id is not None and not isinstance(user_id, str):
        user_id = str(user_id)

    name = row.get("name")
    if not isinstance(name, str) or not name:
        raise ValueError("MCP configuration row missing name")

    type_value = row.get("type")
    if not isinstance(type_value, str) or not type_value:
        raise ValueError("MCP configuration row missing type")

    endpoint_value = row.get("endpoint")
    if endpoint_value is not None and not isinstance(endpoint_value, str):
        endpoint_value = str(endpoint_value)

    auth_value = row.get("auth_type")
    if not isinstance(auth_value, str) or not auth_value:
        raise ValueError("MCP configuration row missing auth_type")

    # Map database fields to model fields
    auth_token = row.get("auth_token")
    if auth_token is not None and not isinstance(auth_token, str):
        auth_token = str(auth_token)

    auth_header = row.get("auth_header")
    if auth_header is not None and not isinstance(auth_header, str):
        auth_header = str(auth_header)

    config_field = row.get("config")
    if config_field is not None and not isinstance(config_field, str):
        config_field = str(config_field)

    scope_field = row.get("scope")
    if scope_field is not None and not isinstance(scope_field, str):
        scope_field = str(scope_field)

    description_field = row.get("description")
    if description_field is not None and not isinstance(description_field, str):
        description_field = str(description_field)

    enabled_value = row.get("enabled", True)
    enabled = bool(enabled_value)

    created_at = row.get("created_at")
    if created_at is not None and not isinstance(created_at, str):
        created_at = str(created_at)

    updated_at = row.get("updated_at")
    if updated_at is not None and not isinstance(updated_at, str):
        updated_at = str(updated_at)

    created_by = row.get("created_by")
    if created_by is not None and not isinstance(created_by, str):
        created_by = str(created_by)

    updated_by = row.get("updated_by")
    if updated_by is not None and not isinstance(updated_by, str):
        updated_by = str(updated_by)

    return MCPConfigRecord(
        id=record_id_str,
        org_id=org_id,
        user_id=user_id,
        name=name,
        type=type_value,
        endpoint=endpoint_value,
        auth_type=auth_value,
        auth_token=auth_token,
        auth_header=auth_header,
        config=config_field,
        scope=scope_field,
        description=description_field,
        created_at=created_at,
        updated_at=updated_at,
        created_by=created_by,
        updated_by=updated_by,
        enabled=enabled,
    )


def _chat_session_from_row(row: dict[str, Any]) -> ChatSessionRecord:
    return ChatSessionRecord(
        id=row["id"],
        user_id=row["user_id"],
        organization_id=row.get("organization_id") or row.get("org_id"),
        title=row.get("title"),
        model=row.get("model"),
        agent_type=row.get("agent_type"),
        created_at=str(row.get("created_at", "")),
        updated_at=str(row.get("updated_at", "")),
        last_message_at=str(row.get("last_message_at", "")),
        message_count=row.get("message_count", 0) or 0,
        tokens_in=row.get("tokens_in", 0) or 0,
        tokens_out=row.get("tokens_out", 0) or 0,
        tokens_total=row.get("tokens_total", 0) or 0,
        metadata=row.get("metadata") or {},  # type: ignore[arg-type]
        archived=bool(row.get("archived", False)),
    )


def _chat_message_from_row(row: dict[str, Any]) -> ChatMessageRecord:
    return ChatMessageRecord(
        id=row["id"],
        session_id=row["session_id"],
        message_index=row.get("message_index", 0) or 0,
        role=row.get("role", ""),
        content=row.get("content", ""),
        metadata=dict(row.get("metadata", {}) or {}),  # type: ignore[arg-type]
        tokens=row.get("tokens") or row.get("tokens_total") or row.get("tokens_out"),
        created_at=str(row.get("created_at", "")),
        updated_at=str(row.get("updated_at", "")),
    )

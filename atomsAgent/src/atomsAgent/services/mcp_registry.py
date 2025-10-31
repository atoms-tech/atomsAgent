from __future__ import annotations

from typing import Any, Literal, cast
from uuid import UUID

from pydantic import BaseModel, Field, HttpUrl, TypeAdapter, ValidationError

from atomsAgent.db.repositories import MCPConfigRecord, MCPRepository
from atomsAgent.schemas.mcp import (
    MCPConfiguration,
    MCPCreateRequest,
    MCPListResponse,
    MCPMetadata,
    MCPScope,
    MCPUpdateRequest,
)


# Custom response that accepts string IDs
class MCPListResponseCompat(BaseModel):
    items: list[dict[str, Any]] = Field(default_factory=list)


AuthTypeLiteral = Literal["none", "bearer", "oauth"]
ConfigTypeLiteral = Literal["http"]

_HTTP_URL_ADAPTER = TypeAdapter(HttpUrl)


class MCPRegistryService:
    """Service for managing MCP configurations."""

    def __init__(self, repository: MCPRepository):
        self._repository = repository

    async def list(
        self,
        organization_id: UUID | None = None,
        user_id: UUID | None = None,
        include_platform: bool = False,
    ) -> MCPListResponse:
        # Default organization_id if None
        org_id = (
            organization_id
            if organization_id is not None
            else UUID("00000000-0000-0000-0000-000000000000")
        )
        records = await self._repository.list_configs(
            organization_id=org_id,
            user_id=user_id,
            include_platform=include_platform,
        )
        return MCPListResponse(items=[self._map_record(r) for r in records])

    async def create(self, payload: MCPCreateRequest) -> MCPConfiguration:
        supabase_payload = self._build_payload(payload)
        record = await self._repository.create_config(supabase_payload)
        return self._map_record(record)

    async def update(self, config_id: UUID, payload: MCPUpdateRequest) -> MCPConfiguration:
        supabase_payload = self._build_payload(payload, partial=True)
        record = await self._repository.update_config(config_id, supabase_payload)
        return self._map_record(record)

    async def delete(self, config_id: UUID) -> None:
        await self._repository.delete_config(config_id)

    async def get_by_id(self, config_id: UUID) -> MCPConfiguration:
        record = await self._repository.get_config(config_id)
        return self._map_record(record)

    @staticmethod
    def _map_record(record: MCPConfigRecord) -> MCPConfiguration:
        # Extract metadata from config field or build default
        metadata_dict: dict[str, Any] = {}
        if record.config and record.config != "null":
            try:
                import json

                metadata_dict = (
                    json.loads(record.config) if isinstance(record.config, str) else record.config
                )
            except Exception:
                metadata_dict = {}

        # Use default empty metadata if not found
        args = metadata_dict.get("args", [])
        env = metadata_dict.get("env", {})
        metadata = MCPMetadata(args=args, env=env)

        if record.endpoint is None:
            raise ValueError("MCP configuration missing endpoint")

        try:
            endpoint = _HTTP_URL_ADAPTER.validate_python(record.endpoint)
        except ValidationError as exc:
            raise ValueError("Invalid MCP endpoint stored in database") from exc
        config_type = record.type
        if config_type != "http":
            raise ValueError(f"Unsupported MCP configuration type '{config_type}'")
        auth_value = record.auth_type
        if auth_value not in {"none", "bearer", "oauth", "api_key"}:
            raise ValueError(f"Unsupported MCP auth type '{auth_value}'")

        # Map org_id to organization_id
        organization_id = None
        if record.org_id:
            try:
                organization_id = UUID(record.org_id)
            except ValueError:
                organization_id = None

        user_id = None
        if record.user_id:
            try:
                user_id = UUID(record.user_id)
            except ValueError:
                user_id = None

        # Determine scope from scope field or fall back to org/user presence
        if record.scope == "platform" or (organization_id is None and user_id is None):
            scope = MCPScope(type="platform", organization_id=None, user_id=None)
        elif record.scope == "user" or (organization_id and user_id):
            scope = MCPScope(type="user", organization_id=organization_id, user_id=user_id)
        else:  # record.scope == "org" or org-only
            scope = MCPScope(type="organization", organization_id=organization_id, user_id=None)

        # Create MCPConfiguration object
        return MCPConfiguration(
            id=UUID(record.id),  # Convert to UUID for type compatibility
            name=record.name,
            type=cast(Literal["http"], config_type),
            endpoint=endpoint,  # Already an HttpUrl
            auth_type=cast(Literal["none", "bearer", "oauth"], auth_value),
            enabled=record.enabled,
            metadata=metadata,
            scope=scope,
            created_at=record.created_at,
        )

    @staticmethod
    def _build_payload(
        payload: MCPCreateRequest | MCPUpdateRequest, *, partial: bool = False
    ) -> dict:
        import json

        base: dict = {}
        if getattr(payload, "name", None) is not None:
            base["name"] = payload.name
        if getattr(payload, "endpoint", None) is not None:
            base["endpoint"] = str(payload.endpoint)
        if getattr(payload, "type", None) is not None:
            base["type"] = payload.type
        if getattr(payload, "auth_type", None) is not None:
            base["auth_type"] = payload.auth_type
        if getattr(payload, "bearer_token", None) is not None:
            base["auth_token"] = payload.bearer_token
        if getattr(payload, "oauth_provider", None) is not None:
            base["oauth_provider"] = payload.oauth_provider
        if getattr(payload, "metadata", None) is not None:
            meta_dict = (
                payload.metadata.model_dump()
                if hasattr(payload.metadata, "model_dump")
                else payload.metadata
            )
            base["config"] = json.dumps(meta_dict)
        scope = getattr(payload, "scope", None)
        if scope is not None:
            base["scope"] = scope.type  # type: ignore
            if not partial:
                base["org_id"] = (
                    str(scope.organization_id) if scope.organization_id else None
                )  # type: ignore
                base["user_id"] = str(scope.user_id) if scope.user_id else None  # type: ignore
        if getattr(payload, "enabled", None) is not None:
            base["enabled"] = payload.enabled
            # Skip org/user_id for updates (they're in scope)

        return base

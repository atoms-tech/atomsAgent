from __future__ import annotations

from typing import Literal
from uuid import UUID

from pydantic import BaseModel, Field, HttpUrl

AuthTypeLiteral = Literal["none", "bearer", "oauth", "api_key"]


class MCPScope(BaseModel):
    type: Literal["platform", "organization", "user"]
    organization_id: UUID | None = None
    user_id: UUID | None = None


class MCPMetadata(BaseModel):
    args: list[str] = Field(default_factory=list)
    env: dict[str, str] = Field(default_factory=dict)


class MCPConfiguration(BaseModel):
    id: UUID | str
    name: str
    type: Literal["http"] = Field(default="http")
    endpoint: HttpUrl
    auth_type: AuthTypeLiteral = Field(default="none")
    bearer_token_id: UUID | None = None
    oauth_provider: str | None = None
    enabled: bool = True
    metadata: MCPMetadata = Field(default_factory=MCPMetadata)
    created_at: str | None = None
    scope: MCPScope


class MCPCreateRequest(BaseModel):
    name: str
    type: Literal["http"] = Field(default="http")
    endpoint: HttpUrl
    auth_type: AuthTypeLiteral = Field(default="none")
    bearer_token: str | None = None
    oauth_provider: str | None = None
    enabled: bool = True
    metadata: MCPMetadata = Field(default_factory=MCPMetadata)
    scope: MCPScope


class MCPUpdateRequest(BaseModel):
    name: str | None = None
    type: Literal["http"] | None = None
    endpoint: HttpUrl | None = None
    auth_type: AuthTypeLiteral | None = None
    bearer_token: str | None = None
    oauth_provider: str | None = None
    enabled: bool | None = None
    metadata: MCPMetadata | None = None


class MCPListResponse(BaseModel):
    items: list[MCPConfiguration]

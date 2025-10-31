from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from pydantic import BaseModel, Field


class ChatMessageModel(BaseModel):
    id: UUID
    message_index: int
    role: str
    content: str
    metadata: dict[str, Any] = Field(default_factory=dict)
    tokens: int | None = None
    created_at: datetime
    updated_at: datetime | None = None


class ChatSessionSummary(BaseModel):
    id: UUID
    user_id: UUID
    organization_id: UUID | None = None
    title: str | None = None
    model: str | None = None
    agent_type: str | None = None
    created_at: datetime
    updated_at: datetime
    last_message_at: datetime | None = None
    message_count: int = 0
    tokens_in: int = 0
    tokens_out: int = 0
    tokens_total: int = 0
    metadata: dict[str, Any] = Field(default_factory=dict)
    archived: bool = False


class ChatSessionListResponse(BaseModel):
    sessions: list[ChatSessionSummary]
    total: int
    page: int
    page_size: int
    has_more: bool


class ChatSessionDetailResponse(BaseModel):
    session: ChatSessionSummary
    messages: list[ChatMessageModel]

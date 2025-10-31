from __future__ import annotations

from datetime import datetime, timezone
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query

from atomsAgent.dependencies import get_chat_history_service
from atomsAgent.schemas.chat import (
    ChatMessageModel,
    ChatSessionDetailResponse,
    ChatSessionListResponse,
    ChatSessionSummary,
)
from atomsAgent.services.chat_history import ChatHistoryService

router = APIRouter()


def _parse_datetime(value: str | None) -> datetime | None:
    if not value:
        return None
    value = value.rstrip("Z")
    try:
        return datetime.fromisoformat(value)
    except ValueError:
        return None


def _to_session_summary(record) -> ChatSessionSummary:
    return ChatSessionSummary(
        id=UUID(record.id),
        user_id=UUID(record.user_id),
        organization_id=UUID(record.organization_id) if record.organization_id else None,
        title=record.title,
        model=record.model,
        agent_type=record.agent_type,
        created_at=_parse_datetime(record.created_at) or datetime.now(timezone.utc),
        updated_at=_parse_datetime(record.updated_at) or datetime.now(timezone.utc),
        last_message_at=_parse_datetime(record.last_message_at),
        message_count=record.message_count,
        tokens_in=record.tokens_in,
        tokens_out=record.tokens_out,
        tokens_total=record.tokens_total,
        metadata=record.metadata,
        archived=record.archived,
    )


def _to_message_model(record) -> ChatMessageModel:
    return ChatMessageModel(
        id=UUID(record.id),
        message_index=record.message_index,
        role=record.role,
        content=record.content,
        metadata=record.metadata,
        tokens=record.tokens,
        created_at=_parse_datetime(record.created_at) or datetime.now(timezone.utc),
        updated_at=_parse_datetime(record.updated_at),
    )


@router.get("/sessions", response_model=ChatSessionListResponse)
async def list_chat_sessions(
    *,
    user_id: UUID = Query(..., description="User identifier"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    service: ChatHistoryService = Depends(get_chat_history_service),
) -> ChatSessionListResponse:
    sessions, total = await service.list_sessions(
        user_id=str(user_id), page=page, page_size=page_size
    )
    summaries = [_to_session_summary(session) for session in sessions]
    has_more = (page - 1) * page_size + len(summaries) < total
    return ChatSessionListResponse(
        sessions=summaries,
        total=total,
        page=page,
        page_size=page_size,
        has_more=has_more,
    )


@router.get("/sessions/{session_id}", response_model=ChatSessionDetailResponse)
async def get_chat_session(
    session_id: UUID,
    *,
    user_id: UUID = Query(..., description="User identifier"),
    service: ChatHistoryService = Depends(get_chat_history_service),
) -> ChatSessionDetailResponse:
    try:
        session, messages = await service.get_session_detail(
            session_id=str(session_id), user_id=str(user_id)
        )
    except ValueError as exc:
        raise HTTPException(status_code=404, detail="chat session not found") from exc
    return ChatSessionDetailResponse(
        session=_to_session_summary(session),
        messages=[_to_message_model(message) for message in messages],
    )

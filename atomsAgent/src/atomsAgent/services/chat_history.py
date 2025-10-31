from __future__ import annotations

from collections.abc import Sequence
from datetime import datetime, timezone

from atomsAgent.db.repositories import (
    ChatHistoryRepository,
    ChatMessageRecord,
    ChatSessionRecord,
)


def _utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _derive_title(messages: Sequence[dict[str, str]]) -> str | None:
    for message in messages:
        if message.get("role") == "user":
            content = (message.get("content") or "").strip()
            if content:
                first_line = content.splitlines()[0].strip()
                return first_line[:80]
    return None


class ChatHistoryService:
    """Coordinates chat session persistence and retrieval."""

    def __init__(self, repository: ChatHistoryRepository) -> None:
        self._repository = repository

    async def ensure_session(
        self,
        *,
        session_id: str,
        user_id: str,
        organization_id: str | None,
        model: str | None,
        agent_type: str | None,
        title_seed: str | None = None,
        metadata: dict[str, object] | None = None,
    ) -> ChatSessionRecord:
        existing = await self._repository.fetch_session(session_id)
        if existing is not None:
            return existing
        payload = {
            "id": session_id,
            "user_id": user_id,
            "org_id": organization_id,
            "title": title_seed,
            "model": model,
            "agent_type": agent_type,
            "metadata": metadata or {},
            "message_count": 0,
            "tokens_in": 0,
            "tokens_out": 0,
            "tokens_total": 0,
            "archived": False,
        }
        return await self._repository.create_session(payload)

    async def sync_user_messages(
        self,
        *,
        session_id: str,
        user_id: str,
        messages: Sequence[dict[str, str]],
    ) -> ChatSessionRecord:
        session = await self._repository.fetch_session_for_user(session_id, user_id)
        if session is None:
            raise ValueError("chat session not found or access denied")

        existing_count = session.message_count
        total_messages = len(messages)
        if total_messages < existing_count:
            existing_count = 0  # fallback to avoid negative slicing if history was reset

        new_messages = messages[existing_count:]
        if new_messages:
            for idx, message in enumerate(new_messages, start=existing_count):
                await self._repository.insert_message(
                    {
                        "session_id": session_id,
                        "message_index": idx,
                        "role": message.get("role"),
                        "content": message.get("content"),
                        "metadata": {},
                        "tokens_in": None,
                        "tokens_out": None,
                        "tokens_total": None,
                    }
                )
            now_iso = _utc_now_iso()
            update_payload: dict[str, object] = {
                "message_count": existing_count + len(new_messages),
                "last_message_at": now_iso,
                "updated_at": now_iso,
            }
            if not session.title:
                title = _derive_title(messages)
                if title:
                    update_payload["title"] = title
            session = await self._repository.update_session(session_id, update_payload)
        elif not session.title:
            title = _derive_title(messages)
            if title:
                session = await self._repository.update_session(session_id, {"title": title})

        return session

    async def record_assistant_message(
        self,
        *,
        session_id: str,
        user_id: str,
        content: str,
        prompt_tokens: int,
        completion_tokens: int,
        total_tokens: int,
    ) -> ChatSessionRecord:
        session = await self._repository.fetch_session_for_user(session_id, user_id)
        if session is None:
            raise ValueError("chat session not found or access denied")

        total_tokens = total_tokens or (prompt_tokens + completion_tokens)

        next_index = session.message_count
        await self._repository.insert_message(
            {
                "session_id": session_id,
                "message_index": next_index,
                "role": "assistant",
                "content": content,
                "metadata": {},
                "tokens_in": None,
                "tokens_out": completion_tokens,
                "tokens_total": total_tokens,
            }
        )
        now_iso = _utc_now_iso()
        update_payload = {
            "message_count": next_index + 1,
            "last_message_at": now_iso,
            "updated_at": now_iso,
            "tokens_in": session.tokens_in + prompt_tokens,
            "tokens_out": session.tokens_out + completion_tokens,
            "tokens_total": session.tokens_total + total_tokens,
        }
        return await self._repository.update_session(session_id, update_payload)

    async def list_sessions(
        self,
        *,
        user_id: str,
        page: int,
        page_size: int,
    ) -> tuple[list[ChatSessionRecord], int]:
        if page < 1:
            page = 1
        if page_size <= 0:
            page_size = 20
        offset = (page - 1) * page_size
        return await self._repository.list_sessions(user_id, limit=page_size, offset=offset)

    async def get_session_detail(
        self,
        *,
        session_id: str,
        user_id: str,
    ) -> tuple[ChatSessionRecord, list[ChatMessageRecord]]:
        session = await self._repository.fetch_session_for_user(session_id, user_id)
        if session is None:
            raise ValueError("chat session not found or access denied")
        messages = await self._repository.fetch_messages(session_id)
        return session, messages

from __future__ import annotations

import json
import time
import uuid
from collections.abc import AsyncGenerator
from typing import Any

from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.responses import StreamingResponse

from atomsAgent.dependencies import (
    get_chat_history_service,
    get_claude_client,
    get_prompt_orchestrator,
    get_vertex_model_service,
)
from atomsAgent.schemas.openai import (
    ChatCompletionChoice,
    ChatCompletionRequest,
    ChatCompletionResponse,
    ChatMessage,
    ModelListResponse,
    UsageInfo,
)
from atomsAgent.services import (
    ClaudeAgentClient,
    CompletionChunk,
    PromptOrchestrator,
    VertexModelService,
    default_session_id,
)
from atomsAgent.services.chat_history import ChatHistoryService

router = APIRouter()


@router.post("/chat/completions", response_model=ChatCompletionResponse)
async def create_chat_completion(
    request: ChatCompletionRequest,
    claude_client: ClaudeAgentClient = Depends(get_claude_client),
    prompt_orchestrator: PromptOrchestrator = Depends(get_prompt_orchestrator),
    history_service: ChatHistoryService = Depends(get_chat_history_service),
) -> StreamingResponse | ChatCompletionResponse:
    metadata: dict[str, Any] = request.metadata or {}
    session_id = metadata.get("session_id") or default_session_id()
    workflow = metadata.get("workflow")
    organization_id = metadata.get("organization_id")
    user_id = metadata.get("user_id")
    if user_id is None:
        user_id = ""  # Provide default value to avoid type errors
    variables = metadata.get("variables")
    allowed_tools: list[str] | None = metadata.get("allowed_tools")
    setting_sources: list[str] | None = metadata.get("setting_sources")
    mcp_servers: dict[str, Any] | None = metadata.get("mcp_servers")

    system_prompt = request.system_prompt or await prompt_orchestrator.compose_prompt(
        organization_id=organization_id,
        user_id=user_id,
        workflow=workflow,
        variables=variables,
    )
    if not request.messages:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="messages array must include at least one message",
        )

    if not request.model:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="model field is required - fetch available models from /v1/models endpoint",
        )

    model = request.model
    messages = [msg.model_dump() for msg in request.messages]
    temperature = request.temperature or 1.0
    max_tokens = request.max_tokens

    history_enabled = bool(user_id)
    history_messages = [
        {"role": msg.get("role"), "content": msg.get("content") if isinstance(msg, dict) else ""}
        for msg in messages
    ]

    if history_enabled:
        session_metadata: dict[str, Any] = {
            "workflow": workflow,
            "variables": variables or {},
        }
        await history_service.ensure_session(
            session_id=session_id,
            user_id=user_id,
            organization_id=organization_id,
            model=model,
            agent_type=metadata.get("agent_type"),
            metadata=session_metadata,
        )
        await history_service.sync_user_messages(
            session_id=session_id,
            user_id=user_id,
            messages=history_messages,
        )

    if request.stream:
        stream_id = f"chatcmpl-{uuid.uuid4().hex}"
        created_ts = int(time.time())

        async def event_stream() -> AsyncGenerator[str, None]:
            first_delta = True
            collected_parts: list[str] = []
            final_usage: UsageInfo | None = None
            try:
                async for chunk in claude_client.stream_complete(
                    session_id=session_id,
                    messages=messages,
                    temperature=temperature,
                    max_tokens=max_tokens,
                    model=model,
                    system_prompt=system_prompt,
                    setting_sources=setting_sources,
                    allowed_tools=allowed_tools,
                    mcp_servers=mcp_servers,
                    user_identifier=request.user,
                    top_p=request.top_p,
                ):
                    for payload in _serialize_chunk(
                        chunk=chunk,
                        stream_id=stream_id,
                        created_ts=created_ts,
                        model=model,
                        session_id=session_id,
                        include_role=first_delta,
                    ):
                        yield payload
                    if chunk.delta:
                        collected_parts.append(chunk.delta)
                        first_delta = False
                    if chunk.usage:
                        final_usage = UsageInfo(
                            prompt_tokens=chunk.usage.prompt_tokens,
                            completion_tokens=chunk.usage.completion_tokens,
                            total_tokens=chunk.usage.total_tokens,
                        )
            except (ValueError, RuntimeError) as exc:
                error_payload = {
                    "error": {
                        "message": str(exc),
                        "type": "invalid_request_error"
                        if isinstance(exc, ValueError)
                        else "server_error",
                    }
                }
                yield f"data: {json.dumps(error_payload)}\n\n"
                yield "data: [DONE]\n\n"
                return

            if history_enabled:
                text = "".join(collected_parts)
                usage = final_usage or UsageInfo(
                    prompt_tokens=0, completion_tokens=0, total_tokens=0
                )
                await history_service.record_assistant_message(
                    session_id=session_id,
                    user_id=user_id,
                    content=text,
                    prompt_tokens=usage.prompt_tokens,
                    completion_tokens=usage.completion_tokens,
                    total_tokens=usage.total_tokens,
                )

        return StreamingResponse(event_stream(), media_type="text/event-stream")

    try:
        result = await claude_client.complete(
            session_id=session_id,
            messages=messages,
            temperature=temperature,
            max_tokens=max_tokens,
            model=model,
            system_prompt=system_prompt,
            setting_sources=setting_sources,
            allowed_tools=allowed_tools,
            mcp_servers=mcp_servers,
            user_identifier=request.user,
            top_p=request.top_p,
        )
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc
    except RuntimeError as exc:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE, detail=str(exc)
        ) from exc

    if history_enabled:
        await history_service.record_assistant_message(
            session_id=session_id,
            user_id=user_id,
            content=result.text,
            prompt_tokens=result.usage.prompt_tokens,
            completion_tokens=result.usage.completion_tokens,
            total_tokens=result.usage.total_tokens,
        )

    response = ChatCompletionResponse(
        id=f"chatcmpl-{uuid.uuid4().hex}",
        object="chat.completion",
        created=int(time.time()),
        model=model,
        choices=[
            ChatCompletionChoice(
                index=0,
                message=ChatMessage(role="assistant", content=result.text),
                finish_reason="stop",
            )
        ],
        usage=UsageInfo(
            prompt_tokens=result.usage.prompt_tokens,
            completion_tokens=result.usage.completion_tokens,
            total_tokens=result.usage.total_tokens,
        ),
        system_fingerprint=session_id,
    )
    return response


@router.get("/models", response_model=ModelListResponse)
async def list_models(
    model_service: VertexModelService = Depends(get_vertex_model_service),
) -> ModelListResponse:
    return await model_service.list_models()


def _serialize_chunk(
    *,
    chunk: CompletionChunk,
    stream_id: str,
    created_ts: int,
    model: str,
    session_id: str,
    include_role: bool,
) -> list[str]:
    payloads: list[str] = []

    if chunk.delta:
        delta: dict[str, Any] = {"content": chunk.delta}
        if include_role:
            delta["role"] = "assistant"
        payload = {
            "id": stream_id,
            "object": "chat.completion.chunk",
            "created": created_ts,
            "model": model,
            "choices": [
                {
                    "index": 0,
                    "delta": delta,
                    "finish_reason": None,
                }
            ],
            "system_fingerprint": session_id,
        }
        payloads.append(f"data: {json.dumps(payload)}\n\n")

    if chunk.done:
        done_payload: dict[str, Any] = {
            "id": stream_id,
            "object": "chat.completion.chunk",
            "created": created_ts,
            "model": model,
            "choices": [
                {
                    "index": 0,
                    "delta": {},
                    "finish_reason": "stop",
                }
            ],
            "system_fingerprint": session_id,
        }
        if chunk.usage:
            done_payload["usage"] = {
                "prompt_tokens": chunk.usage.prompt_tokens,
                "completion_tokens": chunk.usage.completion_tokens,
                "total_tokens": chunk.usage.total_tokens,
            }
        payloads.append(f"data: {json.dumps(done_payload)}\n\n")
        payloads.append("data: [DONE]\n\n")

    return payloads

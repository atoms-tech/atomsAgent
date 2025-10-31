from __future__ import annotations

import time
from typing import Any, Literal

from pydantic import BaseModel, Field, field_validator


class MessageContentText(BaseModel):
    type: Literal["text"] = "text"
    text: str


MessageContent = MessageContentText | str


class ChatMessage(BaseModel):
    role: Literal["system", "user", "assistant", "tool"]
    content: str | list[MessageContent]
    name: str | None = None

    @field_validator("content")
    @classmethod
    def normalize_content(cls, value: str | list[MessageContent]) -> str | list[MessageContentText]:
        if isinstance(value, str):
            return value
        normalized: list[MessageContentText] = []
        for part in value:
            if isinstance(part, str):
                normalized.append(MessageContentText(text=part))
            elif isinstance(part, MessageContentText):
                normalized.append(part)
            elif isinstance(part, dict):
                normalized.append(MessageContentText(**part))
        return normalized


class ChatCompletionRequest(BaseModel):
    model: str
    messages: list[ChatMessage]
    temperature: float = Field(default=0.7, ge=0, le=2)
    max_tokens: int = Field(default=4000, ge=1, le=4000)
    top_p: float = Field(default=1.0, ge=0, le=1)
    stream: bool = Field(default=False)
    user: str | None = None
    system_prompt: str | None = None
    metadata: dict[str, Any] | None = None


class ChatCompletionChoice(BaseModel):
    index: int
    message: ChatMessage
    finish_reason: str | None = None


class UsageInfo(BaseModel):
    prompt_tokens: int = Field(default=0)
    completion_tokens: int = Field(default=0)
    total_tokens: int = Field(default=0)


class ChatCompletionResponse(BaseModel):
    id: str
    object: str = Field(default="chat.completion")
    created: int
    model: str
    choices: list[ChatCompletionChoice]
    usage: UsageInfo
    system_fingerprint: str | None = None


class ModelInfo(BaseModel):
    id: str
    object: str = Field(default="model")
    owned_by: str
    created: int = Field(default_factory=lambda: int(time.time()))
    description: str | None = None
    context_length: int | None = None
    provider: str | None = None
    capabilities: list[str] = Field(default_factory=list)


class ModelListResponse(BaseModel):
    data: list[ModelInfo]
    object: str = Field(default="list")

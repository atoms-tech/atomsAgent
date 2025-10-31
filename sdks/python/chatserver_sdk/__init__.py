"""
ChatServer Python SDK

A Python client library for interacting with the ChatServer API.
Provides OpenAI-compatible interface for chat completions with multi-agent backend support.
"""

from .client import ChatServerClient
from .models import (
    Message,
    MessageRole,
    ChatCompletionRequest,
    ChatCompletionResponse,
    ChatCompletionChoice,
    UsageInfo,
    ModelInfo,
    PlatformStats,
    AdminInfo,
    AuditEntry,
    AdminRequest,
    ChatSessionSummary,
    ChatSessionListResponse,
    ChatSessionDetailResponse,
    ChatMessageRecord,
)
from .exceptions import (
    ChatServerError,
    BadRequestError,
    UnauthorizedError,
    ForbiddenError,
    NotFoundError,
    InternalServerError
)

__version__ = "0.10.0"
__all__ = [
    "ChatServerClient",
    "Message",
    "MessageRole", 
    "ChatCompletionRequest",
    "ChatCompletionResponse",
    "ChatCompletionChoice",
    "UsageInfo",
    "ModelInfo",
    "PlatformStats",
    "AdminInfo",
    "AuditEntry",
    "AdminRequest",
    "ChatSessionSummary",
    "ChatSessionListResponse",
    "ChatSessionDetailResponse",
    "ChatMessageRecord",
    "ChatServerError",
    "BadRequestError",
    "UnauthorizedError",
    "ForbiddenError",
    "NotFoundError",
    "InternalServerError"
]

"""Database models for atomsAgent.

This module provides wrapper classes around the generated Supabase models
to add custom methods like `from_row` for compatibility with the repository layer.
"""

from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from pydantic import ConfigDict, Field, field_validator

from atomsAgent.db.generated.fastapi.schema_public_latest import (
    ChatMessageBaseSchema,
    ChatSessionBaseSchema,
    McpConfigurationBaseSchema,
    McpOauthTokenBaseSchema,
    McpOauthTransactionBaseSchema,
    SystemPromptBaseSchema,
)


class SupabaseChatMessage(ChatMessageBaseSchema):
    """Chat message model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    # Override UUID fields to accept any UUID version
    id: UUID = Field(...)  # type: ignore
    session_id: UUID = Field(...)  # type: ignore

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseChatMessage:
        """Create a SupabaseChatMessage instance from a database row."""
        return cls.model_validate(row)


class SupabaseChatSession(ChatSessionBaseSchema):
    """Chat session model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    # Override UUID fields to accept any UUID version
    id: UUID = Field(...)  # type: ignore
    user_id: UUID = Field(...)  # type: ignore
    org_id: UUID | None = Field(default=None)  # type: ignore
    agent_id: UUID | None = Field(default=None)  # type: ignore
    field_model_id: UUID | None = Field(default=None, alias="model_id")  # type: ignore

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseChatSession:
        """Create a SupabaseChatSession instance from a database row."""
        return cls.model_validate(row)


class SupabaseMcpConfiguration(McpConfigurationBaseSchema):
    """MCP configuration model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    # Override UUID fields to accept any UUID version (if any exist)
    # MCP config uses string IDs, so no UUID override needed

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseMcpConfiguration:
        """Create a SupabaseMcpConfiguration instance from a database row."""
        return cls.model_validate(row)


class SupabaseMcpOauthToken(McpOauthTokenBaseSchema):
    """MCP OAuth token model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    # Override UUID fields to accept any UUID version
    id: UUID = Field(...)  # type: ignore
    transaction_id: UUID = Field(...)  # type: ignore
    user_id: UUID | None = Field(default=None)  # type: ignore
    organization_id: UUID | None = Field(default=None)  # type: ignore

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseMcpOauthToken:
        """Create a SupabaseMcpOauthToken instance from a database row."""
        return cls.model_validate(row)


class SupabaseMcpOauthTransaction(McpOauthTransactionBaseSchema):
    """MCP OAuth transaction model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    # Override UUID fields to accept any UUID version
    id: UUID = Field(...)  # type: ignore
    user_id: UUID | None = Field(default=None)  # type: ignore
    organization_id: UUID | None = Field(default=None)  # type: ignore

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseMcpOauthTransaction:
        """Create a SupabaseMcpOauthTransaction instance from a database row."""
        return cls.model_validate(row)


class SupabaseSystemPrompt(SystemPromptBaseSchema):
    """System prompt model with custom methods.

    Uses BaseSchema to avoid circular dependencies with foreign key relationships.
    """

    model_config = ConfigDict(from_attributes=True, arbitrary_types_allowed=True)

    @classmethod
    def from_row(cls, row: dict[str, Any]) -> SupabaseSystemPrompt:
        """Create a SupabaseSystemPrompt instance from a database row.

        Handles missing fields by providing defaults for required fields.
        """
        # Provide defaults for required fields if missing
        now = datetime.now()
        data = dict(row)

        # Set defaults only if fields are missing
        if 'created_at' not in data:
            data['created_at'] = now
        if 'updated_at' not in data:
            data['updated_at'] = now
        if 'name' not in data:
            data['name'] = data.get('id', 'unnamed')

        return cls.model_validate(data)


__all__ = [
    "SupabaseChatMessage",
    "SupabaseChatSession",
    "SupabaseMcpConfiguration",
    "SupabaseMcpOauthToken",
    "SupabaseMcpOauthTransaction",
    "SupabaseSystemPrompt",
]


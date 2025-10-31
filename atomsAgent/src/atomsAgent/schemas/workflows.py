from typing import Literal

from pydantic import BaseModel, Field


class WorkflowMetadata(BaseModel):
    """Metadata attached to OpenAI-style requests to drive workflow behaviors."""

    name: str
    trigger: Literal["chat", "analyze_requirements", "custom"] | None = Field(default="chat")
    context: dict | None = None

"""Chat completion client for AgentAPI.

Provides synchronous and streaming chat completion capabilities
for the atoms-agent dev UI and CLI commands.
"""

from __future__ import annotations

import json
from collections.abc import Iterator
from typing import Any

import httpx
from pydantic import BaseModel

from atomsAgent.config import load_settings

try:
    import sseclient  # type: ignore[import-not-found]
    HAS_SSE = True
except ImportError:
    HAS_SSE = False


class ChatMessage(BaseModel):
    """Chat message model."""

    role: str
    content: str
    name: str | None = None


class ChatCompletionRequest(BaseModel):
    """Chat completion request."""

    model: str
    messages: list[dict[str, Any]]
    temperature: float = 0.7
    max_tokens: int | None = None
    top_p: float | None = None
    stream: bool = False
    system_prompt: str | None = None


class ModelInfo(BaseModel):
    """Model information."""

    id: str
    provider: str | None = None
    context_length: int | None = None
    description: str | None = None


class ChatClient:
    """Client for AgentAPI chat completions."""

    def __init__(
        self,
        base_url: str | None = None,
        api_key: str | None = None,
        timeout: float = 60.0,
    ):
        """Initialize chat client.

        Args:
            base_url: Base URL for AgentAPI (defaults to config)
            api_key: API key for authentication (defaults to config)
            timeout: Request timeout in seconds
        """
        settings = load_settings()
        self.base_url = (base_url or getattr(settings, "agentapi_url", "http://localhost:3284")).rstrip("/")
        self.api_key = api_key or getattr(settings, "agentapi_key", None)
        self.timeout = timeout

        self.client = httpx.Client(timeout=timeout)

    def _get_headers(self) -> dict[str, str]:
        """Get request headers with authentication."""
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def chat(
        self,
        messages: list[dict[str, Any]],
        model: str,
        temperature: float = 0.7,
        max_tokens: int | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Send non-streaming chat completion request.

        Args:
            messages: List of message dicts with 'role' and 'content'
            model: Model ID to use
            temperature: Sampling temperature (0-2)
            max_tokens: Maximum tokens to generate
            **kwargs: Additional parameters

        Returns:
            Chat completion response dict
        """
        payload = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "stream": False,
            **kwargs,
        }
        if max_tokens:
            payload["max_tokens"] = max_tokens

        response = self.client.post(
            f"{self.base_url}/v1/chat/completions",
            headers=self._get_headers(),
            json=payload,
        )
        response.raise_for_status()
        return response.json()

    def stream_chat(
        self,
        messages: list[dict[str, Any]],
        model: str,
        temperature: float = 0.7,
        max_tokens: int | None = None,
        **kwargs: Any,
    ) -> Iterator[str]:
        """Send streaming chat completion request.

        Args:
            messages: List of message dicts with 'role' and 'content'
            model: Model ID to use
            temperature: Sampling temperature (0-2)
            max_tokens: Maximum tokens to generate
            **kwargs: Additional parameters

        Yields:
            Content chunks as they arrive

        Raises:
            RuntimeError: If sseclient-py is not installed
        """
        if not HAS_SSE:
            raise RuntimeError(
                "Streaming requires sseclient-py. Install it with: pip install sseclient-py"
            )

        payload = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "stream": True,
            **kwargs,
        }
        if max_tokens:
            payload["max_tokens"] = max_tokens

        with self.client.stream(
            "POST",
            f"{self.base_url}/v1/chat/completions",
            headers=self._get_headers(),
            json=payload,
        ) as response:
            response.raise_for_status()
            # Decode bytes to strings for SSEClient
            lines = (line.decode('utf-8') if isinstance(line, bytes) else line
                    for line in response.iter_lines())
            client = sseclient.SSEClient(lines)
            for event in client.events():
                if event.data == "[DONE]":
                    break
                try:
                    chunk = json.loads(event.data)
                    delta = chunk.get("choices", [{}])[0].get("delta", {})
                    if content := delta.get("content"):
                        yield content
                except (json.JSONDecodeError, KeyError, IndexError):
                    continue

    def list_models(self) -> list[ModelInfo]:
        """List available models.

        Returns:
            List of available models
        """
        try:
            response = self.client.get(
                f"{self.base_url}/v1/models",
                headers=self._get_headers(),
            )
            response.raise_for_status()
            data = response.json()
            models = data.get("data", [])
            return [ModelInfo(**model) for model in models]
        except Exception:
            # Return default models if API call fails
            return [
                ModelInfo(id="claude-4-5-haiku-20251001", provider="anthropic", context_length=200000),
                ModelInfo(id="claude-3-5-haiku-20241022", provider="anthropic", context_length=200000),
            ]

    def close(self) -> None:
        """Close the HTTP client."""
        self.client.close()

    def __enter__(self) -> ChatClient:
        """Context manager entry."""
        return self

    def __exit__(self, *args: Any) -> None:
        """Context manager exit."""
        self.close()

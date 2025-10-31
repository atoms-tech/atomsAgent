from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, Iterator, List

import pytest

from chatserver_sdk import ChatServerClient


@dataclass
class _FakeResponse:
    payload: Dict[str, Any]
    status_code: int = 200

    def json(self) -> Dict[str, Any]:
        return self.payload


class _FakeEvent:
    def __init__(self, data: str) -> None:
        self.data = data


class _FakeSSEClient:
    def __init__(self, events: List[_FakeEvent]):
        self._events = events

    def events(self) -> Iterator[_FakeEvent]:
        yield from self._events


@pytest.fixture
def client() -> ChatServerClient:
    return ChatServerClient(api_key="test", base_url="http://localhost:3284")


def test_list_sessions_parses_response(monkeypatch: pytest.MonkeyPatch, client: ChatServerClient) -> None:
    expected = {
        "sessions": [
            {
                "id": "session-1",
                "user_id": "user-1",
                "title": "First session",
                "model": "claude-3-haiku",
                "agent_type": "atoms",
                "created_at": "2024-07-12T19:26:01Z",
                "updated_at": "2024-07-12T19:27:11Z",
                "last_message_at": "2024-07-12T19:27:11Z",
                "message_count": 4,
                "tokens_in": 128,
                "tokens_out": 256,
                "tokens_total": 384,
                "metadata": {"workflow": "default"},
                "archived": False,
            }
        ],
        "total": 1,
        "page": 1,
        "page_size": 20,
        "has_more": False,
    }

    def fake_get(url: str, params: Dict[str, Any], timeout: int) -> _FakeResponse:
        assert url.endswith("/atoms/chat/sessions")
        assert params == {"user_id": "user-1", "page": 1, "page_size": 20}
        return _FakeResponse(expected)

    monkeypatch.setattr(client.session, "get", fake_get)

    result = client.list_sessions("user-1")
    assert result.total == 1
    assert not result.has_more
    assert result.sessions[0].metadata["workflow"] == "default"


def test_get_session_returns_transcript(monkeypatch: pytest.MonkeyPatch, client: ChatServerClient) -> None:
    payload = {
        "session": {
            "id": "session-1",
            "user_id": "user-1",
            "title": "First session",
            "model": "claude-3-haiku",
            "agent_type": "atoms",
            "created_at": "2024-07-12T19:26:01Z",
            "updated_at": "2024-07-12T19:27:11Z",
            "last_message_at": "2024-07-12T19:27:11Z",
            "message_count": 2,
            "tokens_in": 64,
            "tokens_out": 32,
            "tokens_total": 96,
            "metadata": {},
            "archived": False,
        },
        "messages": [
            {
                "id": "msg-1",
                "session_id": "session-1",
                "message_index": 0,
                "role": "user",
                "content": "Hello",
                "metadata": {},
                "tokens": None,
                "created_at": "2024-07-12T19:26:01Z",
                "updated_at": None,
            },
            {
                "id": "msg-2",
                "session_id": "session-1",
                "message_index": 1,
                "role": "assistant",
                "content": "Hi there",
                "metadata": {},
                "tokens": 32,
                "created_at": "2024-07-12T19:27:11Z",
                "updated_at": None,
            },
        ],
    }

    def fake_get(url: str, params: Dict[str, Any], timeout: int) -> _FakeResponse:
        assert url.endswith("/atoms/chat/sessions/session-1")
        assert params == {"user_id": "user-1"}
        return _FakeResponse(payload)

    monkeypatch.setattr(client.session, "get", fake_get)

    detail = client.get_session("session-1", user_id="user-1")
    assert detail.session.message_count == 2
    assert detail.messages[1].role == "assistant"
    assert detail.messages[1].tokens == 32


def test_stream_metadata_captures_session_and_usage(monkeypatch: pytest.MonkeyPatch, client: ChatServerClient) -> None:
    events = _FakeSSEClient(
        [
            _FakeEvent('{"choices":[{"delta":{"content":"Hello"}}],"system_fingerprint":"session-xyz"}'),
            _FakeEvent('{"choices":[{"delta":{"content":" world"}}],"system_fingerprint":"session-xyz"}'),
            _FakeEvent('{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}'),
            _FakeEvent('[DONE]'),
        ]
    )

    def fake_post(url: str, json: Dict[str, Any], stream: bool, timeout: Any) -> _FakeResponse:
        assert stream is True
        assert url.endswith('/v1/chat/completions')
        assert json['metadata']['session_id'] == 'session-xyz'
        return _FakeResponse({}, status_code=200)

    monkeypatch.setattr(client.session, "post", fake_post)
    monkeypatch.setattr("chatserver_sdk.client.sseclient.SSEClient", lambda response: events)

    stream = client.create_completion(
        model="claude-3-haiku",
        messages=[{"role": "user", "content": "Ping"}],
        stream=True,
        session_id="session-xyz",
    )

    assert list(stream) == ["Hello", " world"]
    metadata = stream.metadata
    assert metadata["system_fingerprint"] == "session-xyz"
    usage = metadata["usage"]
    assert usage is not None
    assert usage.total_tokens == 13

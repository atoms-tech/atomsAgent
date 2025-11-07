from __future__ import annotations

import asyncio
import dataclasses
from uuid import UUID

import pytest

from atomsAgent.db.repositories import MCPOAuthTokenRecord, MCPOAuthTransactionRecord
from atomsAgent.services.mcp_oauth import MCPOAuthService, MCPOAuthError


class _StubOAuthRepository:
    def __init__(self) -> None:
        self.transactions: dict[str, MCPOAuthTransactionRecord] = {}
        self.tokens: list[MCPOAuthTokenRecord] = []

    async def create_transaction(self, payload: dict[str, str | None]):
        record = MCPOAuthTransactionRecord(
            id=str(payload["id"]),
            user_id=payload.get("user_id"),
            organization_id=payload.get("organization_id"),
            mcp_namespace=str(payload.get("mcp_namespace")),
            provider_key=str(payload.get("provider_key")),
            status=str(payload.get("status", "pending")),
            authorization_url=payload.get("authorization_url"),
            code_challenge=payload.get("code_challenge"),
            code_verifier=payload.get("code_verifier"),
            state=payload.get("state"),
            scopes=list(payload.get("scopes") or []),
            upstream_metadata=payload.get("upstream_metadata") or {},
            error=None,
            created_at="2025-01-01T00:00:00Z",
            updated_at="2025-01-01T00:00:00Z",
            completed_at=None,
        )
        self.transactions[record.id] = record
        return record

    async def update_transaction(self, transaction_id: UUID, payload: dict[str, object]):
        record = self.transactions[str(transaction_id)]
        updated = dataclasses.replace(
            record,
            status=str(payload.get("status", record.status)),
            error=payload.get("error", record.error),
            completed_at=payload.get("completed_at", record.completed_at),
            updated_at="2025-01-01T00:00:01Z",
        )
        self.transactions[str(transaction_id)] = updated
        return updated

    async def get_transaction(self, transaction_id: UUID):
        return self.transactions[str(transaction_id)]

    async def get_transaction_by_state(self, state: str):
        for record in self.transactions.values():
            if record.state == state:
                return record
        raise ValueError("state not found")

    async def list_active_transactions_for_user(self, **kwargs):  # pragma: no cover - unused
        return []

    async def store_tokens(self, payload: dict[str, object]):
        record = MCPOAuthTokenRecord(
            id="token-1",
            transaction_id=str(payload["transaction_id"]),
            user_id=payload.get("user_id"),
            organization_id=payload.get("organization_id"),
            mcp_namespace=str(payload.get("mcp_namespace")),
            provider_key=str(payload.get("provider_key")),
            access_token=payload.get("access_token"),
            refresh_token=payload.get("refresh_token"),
            token_type=payload.get("token_type"),
            scope=payload.get("scope"),
            expires_at=payload.get("expires_at"),
            issued_at="2025-01-01T00:00:02Z",
            upstream_response=payload.get("upstream_response") or {},
        )
        self.tokens.append(record)
        return record

    async def get_latest_tokens(
        self,
        *,
        mcp_namespace: str,
        user_id: UUID | None = None,
        organization_id: UUID | None = None,
    ):
        if not self.tokens:
            return None
        for token in reversed(self.tokens):
            if token.mcp_namespace != mcp_namespace:
                continue
            if user_id is not None and token.user_id == str(user_id):
                return token
            if organization_id is not None and token.organization_id == str(organization_id):
                return token
        return None


def _write_provider_config(tmp_path):
    config = tmp_path / "mcp_oauth.yml"
    config.write_text(
        """
providers:
  test:
    display_name: Test Provider
    authorization_endpoint: https://example.com/auth
    token_endpoint: https://example.com/token
    redirect_uri: https://agent.example.com/callback
    scopes:
      - files.read
    client_id: test-client
    client_secret: test-secret
"""
    )
    return config


def test_start_transaction_generates_pkce(tmp_path):
    repo = _StubOAuthRepository()
    config_path = _write_provider_config(tmp_path)
    service = MCPOAuthService(repository=repo, config_path=config_path, base_url="https://agent")

    user = UUID(int=1)
    record = asyncio.run(
        service.start_transaction(
            provider_key="test",
            user_id=user,
            mcp_namespace="example/server",
        )
    )

    assert record.provider_key == "test"
    assert record.authorization_url and "example.com/auth" in record.authorization_url
    assert repo.transactions
    payload = next(iter(repo.transactions.values()))
    assert payload.code_challenge is not None
    assert payload.code_verifier is not None


class _FakeHTTPResponse:
    def __init__(self, status_code: int = 200):
        self.status_code = status_code

    def json(self):
        return {
            "access_token": "ACCESS",
            "refresh_token": "REFRESH",
            "expires_in": 3600,
            "token_type": "Bearer",
            "scope": "files.read",
        }


class _FakeAsyncClient:
    def __init__(self, *args, **kwargs) -> None:  # pragma: no cover - simple stub
        self.request_log: list[tuple[str, dict]] = []

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb):
        return False

    async def post(self, url: str, data: dict[str, str]):
        self.request_log.append((url, data))
        return _FakeHTTPResponse()


def test_complete_transaction_fetches_tokens(monkeypatch, tmp_path):
    repo = _StubOAuthRepository()
    config_path = _write_provider_config(tmp_path)
    service = MCPOAuthService(repository=repo, config_path=config_path, base_url="https://agent")

    user = UUID(int=2)
    transaction = asyncio.run(
        service.start_transaction(
            provider_key="test",
            user_id=user,
            mcp_namespace="example/server",
        )
    )

    monkeypatch.setattr("httpx.AsyncClient", _FakeAsyncClient)

    token_record = asyncio.run(
        service.complete_transaction(
            UUID(transaction.id),
            code="auth-code",
            state=transaction.state,
        )
    )

    assert token_record.access_token == "ACCESS"
    assert repo.tokens
    stored = repo.tokens[-1]
    assert stored.access_token == "ACCESS"

    latest = asyncio.run(
        service.latest_tokens_for_namespace(
            mcp_namespace="example/server",
            user_id=user,
        )
    )
    assert latest is not None
    assert latest.access_token == "ACCESS"

    # Organization fallback
    latest_org = asyncio.run(
        service.latest_tokens_for_namespace(
            mcp_namespace="example/server",
            organization_id=UUID(int=3),
        )
    )
    assert latest_org is None


def test_latest_tokens_requires_scope(tmp_path):
    service = MCPOAuthService(repository=_StubOAuthRepository(), config_path=_write_provider_config(tmp_path))
    with pytest.raises(MCPOAuthError):
        asyncio.run(service.latest_tokens_for_namespace(mcp_namespace="example/server"))

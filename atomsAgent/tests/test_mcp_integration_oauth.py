from __future__ import annotations

import asyncio
from uuid import UUID

from atomsAgent.db.repositories import MCPOAuthTokenRecord


def test_compose_servers_injects_oauth_header(monkeypatch):
    user_uuid = str(UUID(int=5))

    async def _fake_user_servers(user_id: str):  # pragma: no cover - simple stub
        return [
            {
                "id": "srv-1",
                "name": "drive",
                "transport_type": "http",
                "url": "https://drive.example.com/mcp",
                "requires_auth": True,
                "auth_type": "oauth",
                "namespace": "drive/server",
                "user_id": user_id,
            }
        ]

    async def _fake_org_servers(org_id: str):  # pragma: no cover - unused
        return []

    async def _fake_project_servers(project_id: str):  # pragma: no cover - unused
        return []

    class _FakeOAuthService:
        async def latest_tokens_for_namespace(self, *, mcp_namespace: str, user_id=None, organization_id=None):
            if mcp_namespace == "drive/server":
                return MCPOAuthTokenRecord(
                    id="token-1",
                    transaction_id="txn-1",
                    user_id=str(user_id) if user_id else None,
                    organization_id=str(organization_id) if organization_id else None,
                    mcp_namespace=mcp_namespace,
                    provider_key="test",
                    access_token="ACCESS",
                    refresh_token="REFRESH",
                    token_type="Bearer",
                    scope="files.read",
                    expires_at="2025-01-01T01:00:00Z",
                    issued_at="2025-01-01T00:00:00Z",
                    upstream_response={},
                )
            return None

    monkeypatch.setattr("atomsAgent.mcp.database.get_user_mcp_servers", _fake_user_servers)
    monkeypatch.setattr("atomsAgent.mcp.database.get_org_mcp_servers", _fake_org_servers)
    monkeypatch.setattr("atomsAgent.mcp.database.get_project_mcp_servers", _fake_project_servers)
    monkeypatch.setattr("atomsAgent.dependencies.get_mcp_oauth_service", lambda: _FakeOAuthService())

    from atomsAgent.mcp.integration import compose_mcp_servers

    servers = asyncio.run(compose_mcp_servers(user_id=user_uuid))

    assert "user_drive" in servers
    drive_config = servers["user_drive"]
    assert drive_config["headers"]["Authorization"] == "Bearer ACCESS"

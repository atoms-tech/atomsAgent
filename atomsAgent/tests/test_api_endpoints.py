import asyncio
from types import SimpleNamespace
from uuid import UUID

from pydantic import HttpUrl

from atomsAgent.api.routes.chat import get_chat_session, list_chat_sessions
from atomsAgent.api.routes.mcp import (
    create_mcp_server,
    delete_mcp_server,
    list_mcp_servers,
    update_mcp_server,
)
from atomsAgent.api.routes.openai import create_chat_completion
from atomsAgent.api.routes.platform import (
    create_platform_admin,
    delete_platform_admin,
    get_audit_logs,
    get_platform_stats,
    list_platform_admins,
)
from atomsAgent.schemas.mcp import (
    MCPConfiguration,
    MCPCreateRequest,
    MCPListResponse,
    MCPMetadata,
    MCPScope,
    MCPUpdateRequest,
)
from atomsAgent.schemas.openai import ChatCompletionRequest, ChatMessage
from atomsAgent.schemas.platform import (
    AddAdminRequest,
    AdminInfo,
    AdminListResponse,
    AdminResponse,
    AuditEntry,
    AuditLogResponse,
    PlatformStats,
)
from atomsAgent.services import (
    CompletionChunk,
    CompletionResult,
    PlatformService,
    PromptOrchestrator,
    UsageStats,
)


class FakeHistoryService:
    def __init__(self) -> None:  # pragma: no cover - stub
        pass

    async def ensure_session(self, **kwargs):  # type: ignore[override]
        return SimpleNamespace(
            id=kwargs.get("session_id"),
            user_id=kwargs.get("user_id"),
            organization_id=kwargs.get("organization_id"),
            title=None,
            model=kwargs.get("model"),
            agent_type=kwargs.get("agent_type"),
            created_at="",
            updated_at="",
            last_message_at=None,
            message_count=0,
            tokens_in=0,
            tokens_out=0,
            tokens_total=0,
            metadata=kwargs.get("metadata", {}),
            archived=False,
        )

    async def sync_user_messages(self, **kwargs):  # type: ignore[override]
        return SimpleNamespace(
            message_count=len(kwargs.get("messages", [])),
            title=None,
            tokens_in=0,
            tokens_out=0,
            tokens_total=0,
        )

    async def record_assistant_message(self, **kwargs):  # type: ignore[override]
        return SimpleNamespace(
            message_count=1,
            tokens_in=kwargs.get("prompt_tokens", 0),
            tokens_out=kwargs.get("completion_tokens", 0),
            tokens_total=kwargs.get("total_tokens", 0),
        )

    async def list_sessions(self, **kwargs):  # pragma: no cover - unused stub
        return [], 0

    async def get_session_detail(self, **kwargs):  # pragma: no cover - unused stub
        raise ValueError


class FakeClaudeClient:
    async def complete(self, **kwargs):
        return CompletionResult(
            text="Hello from Claude",
            usage=UsageStats(prompt_tokens=10, completion_tokens=20),
            session_id=kwargs["session_id"],
            raw_messages=[],
        )

    async def stream_complete(self, **kwargs):
        yield CompletionChunk(delta="Hello", done=False)
        yield CompletionChunk(done=True, usage=UsageStats(prompt_tokens=5, completion_tokens=5))


class FakePromptOrchestrator(PromptOrchestrator):
    async def compose_prompt(self, **kwargs) -> str:  # type: ignore[override]
        return "System prompt"


def test_chat_completion_non_streaming():
    request = ChatCompletionRequest(
        model="claude-4.5-haiku",
        messages=[ChatMessage(role="user", content="Say hello")],
        temperature=0.5,
    )

    response = asyncio.run(
        create_chat_completion(
            request,
            FakeClaudeClient(),
            FakePromptOrchestrator(),
            history_service=FakeHistoryService(),
        )
    )
    assert response.choices[0].message.content == "Hello from Claude"
    assert response.usage.total_tokens == 30


def test_chat_completion_streaming():
    async def _run() -> str:
        request = ChatCompletionRequest(
            model="claude-4.5-haiku",
            messages=[ChatMessage(role="user", content="Say hello")],
            stream=True,
        )
        streaming_response = await create_chat_completion(
            request,
            FakeClaudeClient(),
            FakePromptOrchestrator(),
            history_service=FakeHistoryService(),
        )
        chunks = []
        async for chunk in streaming_response.body_iterator:  # type: ignore[attr-defined]
            if isinstance(chunk, str):
                chunk = chunk.encode()
            chunks.append(chunk)
        return b"".join(chunks).decode()

    text = asyncio.run(_run())
    assert "Hello" in text


class FakeMCPService:
    def __init__(self):
        self.list_called_with = None
        self.create_called = None
        self.update_called = None
        self.delete_called = None

    async def list(self, **kwargs):
        self.list_called_with = kwargs
        scope = MCPScope(type="platform")
        item = MCPConfiguration(
            id=UUID("00000000-0000-0000-0000-000000000001"),
            name="example",
            endpoint=HttpUrl("https://example.com/mcp"),
            metadata=MCPMetadata(),
            auth_type="none",
            scope=scope,
            enabled=True,
        )
        return MCPListResponse(items=[item])

    async def create(self, payload: MCPCreateRequest):
        self.create_called = payload
        scope = payload.scope
        return MCPConfiguration(
            id=UUID("00000000-0000-0000-0000-000000000002"),
            name=payload.name,
            endpoint=payload.endpoint,
            metadata=payload.metadata,
            auth_type=payload.auth_type,
            scope=scope,
            enabled=payload.enabled,
        )

    async def update(self, config_id: UUID, payload):
        self.update_called = (config_id, payload)
        scope = MCPScope(
            type="organization", organization_id=UUID("00000000-0000-0000-0000-000000000003")
        )
        return MCPConfiguration(
            id=config_id,
            name=payload.name or "updated",
            endpoint=payload.endpoint or HttpUrl("https://example.com"),
            metadata=payload.metadata or MCPMetadata(),
            auth_type=payload.auth_type or "none",
            scope=scope,
            enabled=payload.enabled if payload.enabled is not None else True,
        )

    async def delete(self, config_id: UUID):
        self.delete_called = config_id


def test_mcp_flows():
    async def _run() -> None:
        service = FakeMCPService()
        org_id = UUID("00000000-0000-0000-0000-000000000004")

        listing = await list_mcp_servers(
            organization_id=org_id,
            user_id=None,
            include_platform=True,
            service=service,
        )
        assert listing.items[0].name == "example"

        payload = MCPCreateRequest(
            name="new",
            endpoint=HttpUrl("https://new.example.com"),
            scope=MCPScope(type="organization", organization_id=org_id),
        )
        created = await create_mcp_server(payload, service=service)
        assert created.name == "new"

        update_payload = MCPUpdateRequest(
            name="other",
            endpoint=HttpUrl("https://other.example.com"),
        )
        updated = await update_mcp_server(
            update_payload, UUID("00000000-0000-0000-0000-000000000005"), service=service
        )
        assert updated.name == "other"

        await delete_mcp_server(UUID("00000000-0000-0000-0000-000000000006"), service=service)
        assert service.delete_called is not None
        assert service.delete_called.hex.endswith("6")

    asyncio.run(_run())


class FakePlatformService(PlatformService):
    async def get_stats(self):  # type: ignore[override]
        return PlatformStats(
            total_users=10,
            active_users=5,
            total_organizations=2,
            total_requests=100,
            requests_today=20,
            total_tokens=2000,
            tokens_today=300,
        )

    async def list_admins(self):  # type: ignore[override]
        return AdminListResponse(
            admins=[
                AdminInfo(
                    id="1",
                    email="admin@example.com",
                    name="Admin",
                    created_at="2024-01-01T00:00:00Z",
                    created_by="system",
                )
            ],
            count=1,
        )

    async def add_admin(self, request: AddAdminRequest, created_by: str):  # type: ignore[override]
        return AdminResponse(email=request.email)

    async def remove_admin(self, email: str):  # type: ignore[override]
        return AdminResponse(email=email)

    async def list_audit(self, *, limit: int = 50, offset: int = 0):  # type: ignore[override]
        return AuditLogResponse(
            entries=[
                AuditEntry(
                    id="1",
                    timestamp="2024-01-01T00:00:00Z",
                    user_id="user-1",
                    org_id="org-1",
                    action="create",
                    resource="mcp",
                    resource_id="res-1",
                    metadata={"key": "value"},
                )
            ],
            count=1,
            limit=limit,
            offset=offset,
        )


def test_platform_endpoints():
    async def _run() -> None:
        service = FakePlatformService(repository=None)  # type: ignore[arg-type]

        stats = await get_platform_stats(service=service)
        assert stats.total_users == 10

        admins = await list_platform_admins(service=service)
        assert admins.count == 1

        add_request = AddAdminRequest(workos_id="workos-1", email="new@example.com")
        add_resp = await create_platform_admin(add_request, service=service)
        assert add_resp.email == "new@example.com"

        remove_resp = await delete_platform_admin("old@example.com", service=service)
        assert remove_resp.email == "old@example.com"

        audit = await get_audit_logs(limit=10, offset=0, service=service)
        assert audit.count == 1

    asyncio.run(_run())


class FakeHistoryServiceForRoutes(FakeHistoryService):
    def __init__(self) -> None:
        super().__init__()
        self.sample_session = SimpleNamespace(
            id=str(UUID("00000000-0000-0000-0000-000000000101")),
            user_id=str(UUID("00000000-0000-0000-0000-000000000102")),
            organization_id=None,
            title="Sample Chat",
            model="claude-4.5-sonnet",
            agent_type="claude",
            created_at="2024-01-01T00:00:00",
            updated_at="2024-01-01T00:00:00",
            last_message_at="2024-01-01T00:05:00",
            message_count=2,
            tokens_in=10,
            tokens_out=20,
            tokens_total=30,
            metadata={},
            archived=False,
        )
        self.sample_messages = [
            SimpleNamespace(
                id=str(UUID("00000000-0000-0000-0000-000000000201")),
                session_id=self.sample_session.id,
                message_index=0,
                role="user",
                content="Hello",
                metadata={},
                tokens=None,
                created_at="2024-01-01T00:00:00",
                updated_at=None,
            ),
            SimpleNamespace(
                id=str(UUID("00000000-0000-0000-0000-000000000202")),
                session_id=self.sample_session.id,
                message_index=1,
                role="assistant",
                content="Hi there",
                metadata={},
                tokens=5,
                created_at="2024-01-01T00:00:05",
                updated_at=None,
            ),
        ]

    async def list_sessions(self, **kwargs):  # type: ignore[override]
        return [self.sample_session], 1

    async def get_session_detail(self, **kwargs):  # type: ignore[override]
        return self.sample_session, self.sample_messages


def test_list_chat_sessions_route():
    async def _run() -> None:
        response = await list_chat_sessions(
            user_id=UUID("00000000-0000-0000-0000-000000000102"),
            page=1,
            page_size=10,
            service=FakeHistoryServiceForRoutes(),
        )
        assert response.total == 1
        assert response.sessions[0].title == "Sample Chat"

    asyncio.run(_run())


def test_get_chat_session_route():
    async def _run() -> None:
        session_id = UUID("00000000-0000-0000-0000-000000000101")
        response = await get_chat_session(
            session_id,
            user_id=UUID("00000000-0000-0000-0000-000000000102"),
            service=FakeHistoryServiceForRoutes(),
        )
        assert response.session.message_count == 2
        assert len(response.messages) == 2

    asyncio.run(_run())

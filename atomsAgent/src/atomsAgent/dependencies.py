from __future__ import annotations

from functools import lru_cache

from aiocache import Cache

from atomsAgent.config import settings
from atomsAgent.db.repositories import (
    ChatHistoryRepository,
    MCPRepository,
    PlatformRepository,
    PromptRepository,
)
from atomsAgent.db.supabase import SupabaseClient
from atomsAgent.services import (
    ClaudeAgentClient,
    ClaudeSessionManager,
    MCPRegistryService,
    PlatformService,
    PromptOrchestrator,
    SandboxManager,
    VertexModelService,
)
from atomsAgent.services.chat_history import ChatHistoryService


@lru_cache
def get_sandbox_manager() -> SandboxManager:
    root_dir = settings.sandbox_root_dir or "/tmp/atomsAgent/sandboxes"
    return SandboxManager(root_path=root_dir)


@lru_cache
def get_supabase_client() -> SupabaseClient:
    if not settings.supabase_url or not settings.supabase_service_key:
        raise RuntimeError(
            "Supabase configuration is missing. Set ATOMS_SECRET_SUPABASE_URL and ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY."
        )
    return SupabaseClient(
        url=settings.supabase_url,
        service_role_key=settings.supabase_service_key,
    )


@lru_cache
def get_session_manager() -> ClaudeSessionManager:
    return ClaudeSessionManager(
        sandbox_manager=get_sandbox_manager(),
        default_allowed_tools=settings.default_allowed_tools,
        default_setting_sources=settings.default_setting_sources or None,
    )


@lru_cache
def get_prompt_orchestrator() -> PromptOrchestrator:
    return PromptOrchestrator(
        prompt_repository=PromptRepository(get_supabase_client()),
        platform_prompt=settings.platform_system_prompt,
        workflow_prompts=settings.workflow_prompt_map,
    )


@lru_cache
def get_vertex_model_service() -> VertexModelService:
    cache = Cache(Cache.MEMORY)
    return VertexModelService(
        project_id=settings.vertex_project_id,
        location=settings.vertex_location,
        cache_ttl=settings.model_cache_ttl_seconds,
        credentials_path=getattr(settings, "vertex_credentials_path", None),
        credentials_json=getattr(settings, "vertex_credentials_json", None),
        cache=cache,
    )


@lru_cache
def get_claude_client() -> ClaudeAgentClient:
    return ClaudeAgentClient(
        session_manager=get_session_manager(),
        default_allowed_tools=settings.default_allowed_tools,
    )


@lru_cache
def get_platform_service() -> PlatformService:
    return PlatformService(repository=PlatformRepository(get_supabase_client()))


@lru_cache
def get_mcp_service() -> MCPRegistryService:
    return MCPRegistryService(repository=MCPRepository(get_supabase_client()))


@lru_cache
def get_chat_history_service() -> ChatHistoryService:
    return ChatHistoryService(repository=ChatHistoryRepository(get_supabase_client()))

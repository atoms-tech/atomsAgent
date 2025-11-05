from fastapi import FastAPI

from atomsAgent.api.routes import chat, mcp, oauth, openai, platform
from atomsAgent.schemas.platform import SystemHealth


def register_routes(app: FastAPI) -> None:
    """Attach all routers to the FastAPI application."""
    app.include_router(openai.router, prefix="/v1", tags=["openai"])
    app.include_router(chat.router, prefix="/atoms/chat", tags=["chat"])
    app.include_router(mcp.router, prefix="/atoms/mcp", tags=["mcp"])
    app.include_router(oauth.router, prefix="/atoms/oauth", tags=["oauth"])
    app.include_router(platform.router, prefix="/api/v1/platform", tags=["platform"])

    @app.get("/health", tags=["health"])
    async def health_check() -> SystemHealth:
        return SystemHealth(status="healthy")

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import ORJSONResponse

from atomsAgent.api import register_routes
from atomsAgent.config import settings


def create_app() -> FastAPI:
    """Create FastAPI application instance configured for atomsAgent."""
    app = FastAPI(
        title="atomsAgent API",
        version=settings.app_version,
        docs_url="/docs" if settings.enable_docs else None,
        redoc_url="/redoc" if settings.enable_docs else None,
        openapi_url="/openapi.json",
        default_response_class=ORJSONResponse,
    )

    if settings.cors_allow_origins:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=settings.cors_allow_origins,
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    register_routes(app)

    return app


app = create_app()

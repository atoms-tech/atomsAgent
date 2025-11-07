"""Route registration helpers."""

from atomsAgent.api.routes import chat, mcp, openai, platform
# from atomsAgent.api.routes import oauth  # Temporarily disabled - needs oauth_manager implementation

__all__ = ["chat", "mcp", "openai", "platform"]  # "oauth" temporarily removed

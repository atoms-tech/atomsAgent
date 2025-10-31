from __future__ import annotations

from dataclasses import dataclass
from typing import ClassVar

from atomsAgent.settings import ConfigSettings, SecretSettings, get_config, get_secrets


@dataclass(frozen=True)
class SettingsProxy:
    """Composite settings object exposing config + secrets namespaces."""

    config: ConfigSettings
    secrets: SecretSettings

    def __getattr__(self, item: str):
        # Check secrets first for sensitive/credential fields
        if hasattr(self.secrets, item):
            secret_value = getattr(self.secrets, item)
            if secret_value is not None:
                return secret_value

        # Check config for non-sensitive fields
        if hasattr(self.config, item):
            config_value = getattr(self.config, item)
            # Only return config value if it's not empty/None or if secrets doesn't have it
            if config_value not in [None, ""] or not hasattr(self.secrets, item):
                return config_value

        # If we get here, raise AttributeError
        raise AttributeError(item)


class UnifiedConfig(ConfigSettings):
    """Configuration with full Agent SDK features enabled."""

    # Permission settings
    permission_mode: str = "default"

    allowed_tools: ClassVar[list[str]] = [
        "Read",
        "Write",
        "Edit",
        "Bash",
        "Grep",
        "Glob",
        "WebSearch",
        "WebFetch",
        "TodoWrite",
        # Add our custom tools
        "atoms_status",
        "session_info",
        "workspace_search",
    ]

    disallowed_tools: ClassVar[list[str]] = []

    # Session settings
    max_turns: int | None = 50
    continue_conversation: bool = True

    # Setting sources for project context
    setting_sources: ClassVar[list[str]] = ["project"]

    # Enhanced features
    enable_hooks: bool = True
    enable_interrupts: bool = True
    enable_streaming: bool = True
    include_partial_messages: bool = False

    # Security settings
    sandbox_isolation: bool = True
    workspace_path: str = "./workspaces"

    # Performance settings
    max_concurrent_sessions: int = 10
    session_timeout: int = 3600

    # Custom tool settings
    enable_custom_tools: bool = True

    # Agent Skills (new feature)
    enable_agent_skills: bool = True
    skills_directory: str = "./.claude/skills"

    # Hooks configuration
    pre_tool_hooks: ClassVar[list[str]] = ["logging", "security"]
    post_tool_hooks: ClassVar[list[str]] = ["logging", "monitoring"]

    # Environment variables for sessions
    session_env: ClassVar[dict[str, str]] = {}

    # CLI args
    extra_cli_args: ClassVar[dict[str, str | None]] = {}

    # Feature flags for progressive rollout
    enable_experimental_features: bool = False
    enable_advanced_permissions: bool = True
    enable_session_analytics: bool = True


def load_settings() -> SettingsProxy:
    """Load settings with enhanced features if available."""
    try:
        # Try to load enhanced config
        config = get_config()
        if hasattr(config, "enable_hooks"):
            # Already enhanced config
            pass
        else:
            # Upgrade to enhanced config while preserving loaded values
            # Get the field values from the loaded config as a dict
            config_dict = config.model_dump() if hasattr(config, "model_dump") else config.__dict__
            enhanced_config = UnifiedConfig(**config_dict)
            config = enhanced_config
    except Exception:
        # Fallback to enhanced config
        config = UnifiedConfig()

    return SettingsProxy(config=config, secrets=get_secrets())


def get_unified_settings() -> UnifiedConfig:
    """Get enhanced configuration instance."""
    try:
        config = get_config()
        if isinstance(config, UnifiedConfig):
            return config
        else:
            # Upgrade to enhanced config
            return UnifiedConfig(**config.__dict__)
    except Exception:
        return UnifiedConfig()


settings = load_settings()

__all__ = [
    "SettingsProxy",
    "UnifiedConfig",
    "get_unified_settings",
    "load_settings",
    "settings",
]

from __future__ import annotations

import pathlib
from functools import lru_cache

import yaml
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class ConfigSettings(BaseSettings):
    """Non-secret application configuration."""

    model_config = SettingsConfigDict(env_prefix="ATOMS_", extra="allow")

    app_version: str = Field(default="0.1.0")
    enable_docs: bool = Field(default=True)
    cors_allow_origins: list[str] = Field(default_factory=list)

    vertex_project_id: str = Field(default="")
    vertex_location: str = Field(default="us-central1")

    model_cache_ttl_seconds: int = Field(default=600)

    platform_prompt_id: str | None = Field(default=None)
    platform_system_prompt: str | None = Field(default=None)
    workflow_prompt_map: dict[str, str] = Field(default_factory=dict)

    sandbox_root_dir: str = Field(default="/tmp/atomsAgent/sandboxes")
    default_allowed_tools: list[str] = Field(
        default_factory=lambda: ["Read", "Write", "Edit", "Bash", "Skill"]
    )
    default_setting_sources: list[str] = Field(default_factory=list)

    @classmethod
    def from_yaml_file(cls, yaml_path: pathlib.Path) -> ConfigSettings:
        """Load config from YAML file."""
        if not yaml_path.exists():
            return cls()  # Return empty instance if file doesn't exist

        with open(yaml_path, encoding="utf-8") as f:
            config_data = yaml.safe_load(f)

        return cls(**config_data)


@lru_cache
def get_config() -> ConfigSettings:
    """Load config from YAML file first, then environment variables.

    Looks for config.yml in the following order:
    1. ATOMS_CONFIG_PATH environment variable
    2. atomsAgent/config/config.yml (package config directory)
    3. Current working directory
    4. Fall back to defaults and environment variables
    """
    import os

    # 1. Check if ATOMS_CONFIG_PATH environment variable is set
    if env_path := os.getenv("ATOMS_CONFIG_PATH"):
        config_yaml_path = pathlib.Path(env_path)
        if config_yaml_path.exists():
            return ConfigSettings.from_yaml_file(config_yaml_path)

    # 2. Check atomsAgent/config/config.yml (package config directory)
    # Path: atomsAgent/src/atomsAgent/settings/config.py → ../../config/config.yml
    # Go up 2 levels: settings → atomsAgent, then into config/
    package_config = pathlib.Path(__file__).resolve().parents[1] / "config" / "config.yml"
    if package_config.exists():
        return ConfigSettings.from_yaml_file(package_config)

    # 3. Check current working directory
    cwd_config = pathlib.Path.cwd() / "config.yml"
    if cwd_config.exists():
        return ConfigSettings.from_yaml_file(cwd_config)

    # 4. Check project root (go up from src/atomsAgent/settings to project root)
    project_root_config = pathlib.Path(__file__).resolve().parents[3] / "config" / "config.yml"
    if project_root_config.exists():
        return ConfigSettings.from_yaml_file(project_root_config)

    # Debug: print the paths we tried if DEBUG_CONFIG is set
    if os.getenv("DEBUG_CONFIG"):
        print("Tried config paths:")
        print(f"  - Package config: {package_config} (exists: {package_config.exists()})")
        print(f"  - CWD: {cwd_config} (exists: {cwd_config.exists()})")

    # Fall back to defaults and environment variables
    return ConfigSettings()

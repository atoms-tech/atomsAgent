from __future__ import annotations

import pathlib
from functools import lru_cache

import yaml
from pydantic_settings import BaseSettings, SettingsConfigDict


class SecretSettings(BaseSettings):
    """Sensitive configuration loaded from YAML and environment variables."""

    model_config = SettingsConfigDict(env_prefix="ATOMS_SECRET_", extra="ignore")

    # Vertex AI Configuration
    vertex_credentials_path: str | None = None
    vertex_credentials_json: str | None = None
    vertex_project_id: str | None = None
    vertex_location: str | None = None

    # Claude API Configuration
    claude_api_key: str | None = None

    # Supabase Configuration
    supabase_url: str | None = None
    supabase_anon_key: str | None = None
    supabase_service_key: str | None = None

    # Redis Configuration
    redis_url: str | None = None

    # Database Configuration
    database_url: str | None = None

    # Authentication Configuration
    authkit_jwks_url: str | None = None

    # Security Configuration
    token_encryption_key: str | None = None

    # Static API Configuration
    static_api_key: str | None = None
    static_api_user_id: str | None = None
    static_api_org_id: str | None = None
    static_api_email: str | None = None
    static_api_name: str | None = None

    @classmethod
    def from_yaml_file(cls, yaml_path: pathlib.Path) -> SecretSettings:
        """Load secrets from YAML file."""
        if not yaml_path.exists():
            return cls()  # Return empty instance if file doesn't exist

        with open(yaml_path, encoding="utf-8") as f:
            secrets_data = yaml.safe_load(f)

        return cls(**secrets_data)


@lru_cache
def get_secrets() -> SecretSettings:
    """Load secrets from YAML file first, then environment variables.

    Looks for secrets.yml in the following order:
    1. ATOMS_SECRETS_PATH environment variable
    2. atomsAgent/config/secrets.yml (package config directory)
    3. Current working directory
    4. Fall back to environment variables only
    """
    import os

    # 1. Check if ATOMS_SECRETS_PATH environment variable is set
    if env_path := os.getenv("ATOMS_SECRETS_PATH"):
        secrets_yaml_path = pathlib.Path(env_path)
        if secrets_yaml_path.exists():
            return SecretSettings.from_yaml_file(secrets_yaml_path)

    # 2. Check atomsAgent/config/secrets.yml (package config directory)
    # Path: atomsAgent/src/atomsAgent/settings/secrets.py → ../../config/secrets.yml
    # Go up 2 levels: settings → atomsAgent, then into config/
    package_config = pathlib.Path(__file__).resolve().parents[1] / "config" / "secrets.yml"
    if package_config.exists():
        return SecretSettings.from_yaml_file(package_config)

    # 3. Check current working directory
    cwd_secrets = pathlib.Path.cwd() / "secrets.yml"
    if cwd_secrets.exists():
        return SecretSettings.from_yaml_file(cwd_secrets)

    # 4. Check project root (go up from src/atomsAgent/settings to project root)
    project_root_secrets = pathlib.Path(__file__).resolve().parents[3] / "config" / "secrets.yml"
    if project_root_secrets.exists():
        return SecretSettings.from_yaml_file(project_root_secrets)

    # Debug: print the paths we tried if DEBUG_SECRETS is set
    if os.getenv("DEBUG_SECRETS"):
        print("Tried secrets paths:")
        print(f"  - Package config: {package_config} (exists: {package_config.exists()})")
        print(f"  - CWD: {cwd_secrets} (exists: {cwd_secrets.exists()})")

    # Fall back to environment variables
    return SecretSettings()

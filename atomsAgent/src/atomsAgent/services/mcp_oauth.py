"""MCP OAuth orchestration service.

This module owns the OAuth handshake used to connect remote MCP servers that require
OAuth or PKCE authentication. The service coordinates provider metadata, creates
transaction records in Supabase, exchanges authorization codes for tokens, and
persists refresh credentials for later use when constructing Claude sessions.
"""

from __future__ import annotations

import base64
import hashlib
import json
import os
import secrets
from collections.abc import Iterable
from dataclasses import dataclass, field
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any
from urllib.parse import urlencode
from uuid import UUID, uuid4

import httpx
import yaml
from mcp.server.auth.provider import OAuthToken

from atomsAgent.config import settings
from atomsAgent.db.models import SupabaseMcpOauthToken, SupabaseMcpOauthTransaction
from atomsAgent.db.repositories import MCPOAuthRepository

_DEFAULT_CONFIG_PATH = Path(__file__).resolve().parents[2] / "config" / "mcp_oauth.yml"


class MCPOAuthError(RuntimeError):
    """Raised when an OAuth operation cannot be completed."""


@dataclass(slots=True)
class OAuthProviderConfig:
    key: str
    display_name: str
    authorization_endpoint: str
    token_endpoint: str
    redirect_uri: str
    scopes: list[str]
    client_id: str
    client_secret: str | None = None
    audience: str | None = None
    access_type: str | None = None
    prompt: str | None = None
    uses_pkce: bool = True
    extra_authorize_params: dict[str, str] = field(default_factory=dict)
    extra_token_params: dict[str, str] = field(default_factory=dict)

    def build_authorization_url(
        self,
        *,
        state: str,
        code_challenge: str | None,
        scopes: Iterable[str] | None = None,
    ) -> str:
        """Compose the authorization URL for the provider."""
        query: dict[str, Any] = {
            "response_type": "code",
            "client_id": self.client_id,
            "redirect_uri": self.redirect_uri,
            "scope": " ".join(scopes or self.scopes),
            "state": state,
        }

        if self.audience:
            query["audience"] = self.audience
        if self.access_type:
            query["access_type"] = self.access_type
        if self.prompt:
            query["prompt"] = self.prompt
        if self.uses_pkce and code_challenge:
            query["code_challenge"] = code_challenge
            query["code_challenge_method"] = "S256"

        query.update(self.extra_authorize_params)

        return f"{self.authorization_endpoint}?{urlencode(query, doseq=True)}"


def _load_yaml_config(config_path: Path) -> dict[str, Any]:
    if not config_path.exists():
        return {}
    with config_path.open("r", encoding="utf-8") as handle:
        return yaml.safe_load(handle) or {}


def _base64url_sha256(value: str) -> str:
    digest = hashlib.sha256(value.encode("utf-8")).digest()
    return base64.urlsafe_b64encode(digest).decode("utf-8").rstrip("=")


def _now_utc() -> datetime:
    return datetime.now(UTC)


class MCPOAuthService:
    """Coordinates OAuth flows for MCP servers."""

    def __init__(
        self,
        repository: MCPOAuthRepository,
        *,
        config_path: Path | None = None,
        base_url: str | None = None,
    ) -> None:
        self._repository = repository
        self._base_url = (base_url or os.getenv("ATOMSAGENT_URL") or "http://localhost:3284").rstrip(
            "/"
        )
        self._providers = self._load_providers(config_path or _DEFAULT_CONFIG_PATH)
        self._dynamic_clients: dict[str, dict[str, Any]] = {}

    # ------------------------------------------------------------------ Providers
    def list_providers(self) -> list[dict[str, Any]]:
        """Return provider metadata for UI consumption."""
        providers: list[dict[str, Any]] = []
        for provider in self._providers.values():
            providers.append(
                {
                    "key": provider.key,
                    "name": provider.display_name,
                    "scopes": provider.scopes,
                    "uses_pkce": provider.uses_pkce,
                    "authorization_endpoint": provider.authorization_endpoint,
                }
            )
        return providers

    def get_provider(self, key: str) -> OAuthProviderConfig:
        try:
            return self._providers[key]
        except KeyError as exc:  # pragma: no cover - simple guard
            raise MCPOAuthError(f"OAuth provider '{key}' is not configured") from exc

    # ------------------------------------------------------------------ Transactions
    async def start_transaction(
        self,
        *,
        provider_key: str,
        user_id: UUID,
        mcp_namespace: str,
        organization_id: UUID | None = None,
        scopes: Iterable[str] | None = None,
        auth_metadata: dict[str, Any] | None = None,
    ) -> SupabaseMcpOauthTransaction:
        provider = await self._ensure_provider(provider_key, auth_metadata)
        if provider.uses_pkce and not provider.client_id:
            raise MCPOAuthError(f"Provider '{provider_key}' is missing client credentials")

        transaction_id = uuid4()
        verifier = secrets.token_urlsafe(64)
        code_challenge = _base64url_sha256(verifier) if provider.uses_pkce else None
        state_nonce = secrets.token_urlsafe(16)
        state = f"{transaction_id}:{state_nonce}"
        requested_scopes = list(scopes) if scopes else list(provider.scopes)
        if not requested_scopes:
            raise MCPOAuthError(f"Provider '{provider_key}' has no scopes configured")

        auth_url = provider.build_authorization_url(
            state=state, code_challenge=code_challenge, scopes=requested_scopes
        )

        payload = {
            "id": str(transaction_id),
            "user_id": str(user_id),
            "organization_id": str(organization_id) if organization_id else None,
            "mcp_namespace": mcp_namespace,
            "provider_key": provider_key,
            "status": "pending",
            "authorization_url": auth_url,
            "code_verifier": verifier if provider.uses_pkce else None,
            "code_challenge": code_challenge,
            "state": state,
            "scopes": requested_scopes,
            "upstream_metadata": json.dumps({"base_url": self._base_url}),
        }

        record = await self._repository.create_transaction(payload)
        return record

    async def get_transaction(self, transaction_id: UUID) -> SupabaseMcpOauthTransaction:
        return await self._repository.get_transaction(transaction_id)

    async def get_transaction_by_state(self, state: str) -> SupabaseMcpOauthTransaction:
        return await self._repository.get_transaction_by_state(state)

    async def mark_transaction_failed(
        self,
        transaction_id: UUID,
        *,
        reason: str,
        details: dict[str, Any] | None = None,
    ) -> SupabaseMcpOauthTransaction:
        payload = {
            "status": "failed",
            "error": json.dumps({"reason": reason, "details": details or {}}),
            "completed_at": _now_utc().isoformat(),
        }
        return await self._repository.update_transaction(transaction_id, payload)

    # ------------------------------------------------------------------ Completion
    async def complete_transaction(
        self,
        transaction_id: UUID,
        *,
        code: str,
        state: str | None,
    ) -> SupabaseMcpOauthToken:
        transaction = await self._repository.get_transaction(transaction_id)
        if transaction.status != "pending":
            raise MCPOAuthError("OAuth transaction already resolved")

        if state and transaction.state and state != transaction.state:
            raise MCPOAuthError("State parameter mismatch during OAuth callback")

        provider = self.get_provider(transaction.provider_key)
        payload = {
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": provider.redirect_uri,
            "client_id": provider.client_id,
        }
        if provider.client_secret:
            payload["client_secret"] = provider.client_secret
        if provider.uses_pkce and transaction.code_verifier:
            payload["code_verifier"] = transaction.code_verifier
        payload.update(provider.extra_token_params)

        async with httpx.AsyncClient(timeout=30.0) as client:
            response = await client.post(provider.token_endpoint, data=payload)
        if response.status_code >= 400:
            raise MCPOAuthError(
                f"OAuth token endpoint error ({response.status_code}): {response.text}"
            )

        token_data = response.json()
        oauth_token = OAuthToken.model_validate(token_data)

        expires_at: datetime | None = None
        if oauth_token.expires_in:
            expires_at = _now_utc() + timedelta(seconds=oauth_token.expires_in)

        token_payload = {
            "transaction_id": str(transaction_id),
            "user_id": transaction.user_id,
            "organization_id": transaction.organization_id,
            "mcp_namespace": transaction.mcp_namespace,
            "provider_key": transaction.provider_key,
            "access_token": oauth_token.access_token,
            "refresh_token": oauth_token.refresh_token,
            "token_type": oauth_token.token_type,
            "scope": oauth_token.scope or " ".join(transaction.scopes or []),
            "expires_at": expires_at.isoformat() if expires_at else None,
            "upstream_response": json.dumps(token_data),
        }

        token_record = await self._repository.store_tokens(token_payload)
        await self._repository.update_transaction(
            transaction_id,
            {
                "status": "authorized",
                "completed_at": _now_utc().isoformat(),
                "error": None,
            },
        )
        return token_record

    async def revoke_tokens(self, transaction_id: UUID) -> None:
        # TODO: implement revocation when upstream providers require it
        await self._repository.update_transaction(
            transaction_id,
            {
                "status": "cancelled",
                "completed_at": _now_utc().isoformat(),
            },
        )

    async def latest_tokens_for_namespace(
        self,
        *,
        mcp_namespace: str,
        user_id: UUID | None = None,
        organization_id: UUID | None = None,
    ) -> SupabaseMcpOauthToken | None:
        if user_id is None and organization_id is None:
            raise MCPOAuthError(
                "Cannot fetch OAuth tokens without a user_id or organization_id"
            )
        return await self._repository.get_latest_tokens(
            mcp_namespace=mcp_namespace,
            user_id=user_id,
            organization_id=organization_id,
        )

    # ------------------------------------------------------------------ Provider loading
    def _load_providers(self, config_path: Path) -> dict[str, OAuthProviderConfig]:
        raw_config = _load_yaml_config(config_path)
        providers_cfg = raw_config.get("providers", {}) or {}
        providers: dict[str, OAuthProviderConfig] = {}

        for key, payload in providers_cfg.items():
            client_id = self._resolve_secret(payload.get("client_id"), payload.get("client_id_env"))
            client_secret = self._resolve_secret(
                payload.get("client_secret"), payload.get("client_secret_env")
            )
            redirect_uri = self._expand_redirect_uri(
                payload.get("redirect_uri"),
                default=f"{self._base_url}/atoms/oauth/callback/{key}",
            )
            scopes = payload.get("scopes") or []

            providers[key] = OAuthProviderConfig(
                key=key,
                display_name=payload.get("display_name", key),
                authorization_endpoint=payload["authorization_endpoint"],
                token_endpoint=payload["token_endpoint"],
                redirect_uri=redirect_uri,
                scopes=scopes,
                client_id=client_id,
                client_secret=client_secret,
                audience=payload.get("audience"),
                access_type=payload.get("access_type"),
                prompt=payload.get("prompt"),
                uses_pkce=payload.get("uses_pkce", True),
                extra_authorize_params=payload.get("extra_authorize_params", {}) or {},
                extra_token_params=payload.get("extra_token_params", {}) or {},
            )

        return providers

    async def _ensure_provider(
        self,
        provider_key: str,
        metadata: dict[str, Any] | None,
    ) -> OAuthProviderConfig:
        provider = self._providers.get(provider_key)
        if provider is not None:
            return provider

        provider = await self._bootstrap_remote_provider(provider_key, metadata)
        if provider is not None:
            self._providers[provider_key] = provider
            return provider

        raise MCPOAuthError(
            f"OAuth provider '{provider_key}' is not configured and remote discovery failed"
        )

    async def _bootstrap_remote_provider(
        self,
        provider_key: str,
        metadata: dict[str, Any] | None,
    ) -> OAuthProviderConfig | None:
        if not metadata:
            return None

        discovery_urls: list[str] = []
        raw = metadata

        for key in ("metadata_url", "wellKnown", "well_known", "configuration_url"):
            value = raw.get(key)
            if isinstance(value, str):
                discovery_urls.append(value)

        issuer = raw.get("issuer") or raw.get("issuer_url")
        if isinstance(issuer, str):
            issuer = issuer.rstrip("/")
            discovery_urls.append(f"{issuer}/.well-known/openid-configuration")
            discovery_urls.append(f"{issuer}/.well-known/oauth-authorization-server")

        metadata_doc: dict[str, Any] = {}
        for url in discovery_urls:
            try:
                async with httpx.AsyncClient(timeout=10.0) as client:
                    response = await client.get(url)
                    if response.status_code < 400:
                        metadata_doc = response.json()
                        break
            except Exception:  # pragma: no cover - best effort discovery
                continue

        authorization_endpoint = (
            metadata_doc.get("authorization_endpoint")
            or raw.get("authorization_endpoint")
        )
        token_endpoint = metadata_doc.get("token_endpoint") or raw.get("token_endpoint")
        registration_endpoint = (
            metadata_doc.get("registration_endpoint")
            or raw.get("registration_endpoint")
            or raw.get("registrationEndpoint")
        )

        if not authorization_endpoint or not token_endpoint or not registration_endpoint:
            return None

        scopes: list[str] = []
        raw_scopes = raw.get("scopes")
        if isinstance(raw_scopes, list):
            scopes.extend(str(scope) for scope in raw_scopes)
        elif isinstance(raw_scopes, str):
            scopes.extend(part.strip() for part in raw_scopes.split())

        if not scopes:
            scopes_supported = metadata_doc.get("scopes_supported")
            if isinstance(scopes_supported, list):
                scopes.extend(str(scope) for scope in scopes_supported)

        redirect_uri = f"{self._base_url}/atoms/oauth/callback/{provider_key}"

        registration_payload = {
            "client_name": f"atomsAgent {provider_key}",
            "application_type": "web",
            "redirect_uris": [redirect_uri],
            "grant_types": ["authorization_code"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none",
        }
        if scopes:
            registration_payload["scope"] = " ".join(scopes)

        try:
            async with httpx.AsyncClient(timeout=15.0) as client:
                registration_response = await client.post(
                    registration_endpoint,
                    json=registration_payload,
                    headers={"Content-Type": "application/json"},
                )
        except Exception as exc:  # pragma: no cover
            raise MCPOAuthError(
                f"Failed to register OAuth client with provider '{provider_key}': {exc}"
            ) from exc

        if registration_response.status_code >= 400:
            raise MCPOAuthError(
                f"Provider '{provider_key}' rejected dynamic registration: "
                f"{registration_response.status_code}"
            )

        client_metadata = registration_response.json()
        client_id = client_metadata.get("client_id")
        if not client_id:
            raise MCPOAuthError(
                f"Provider '{provider_key}' did not return a client_id during registration"
            )

        client_secret = client_metadata.get("client_secret")
        self._dynamic_clients[provider_key] = client_metadata

        display_name = raw.get("name") or raw.get("display_name") or provider_key

        return OAuthProviderConfig(
            key=provider_key,
            display_name=str(display_name),
            authorization_endpoint=str(authorization_endpoint),
            token_endpoint=str(token_endpoint),
            redirect_uri=redirect_uri,
            scopes=scopes or ["openid"],
            client_id=str(client_id),
            client_secret=str(client_secret) if client_secret else None,
            uses_pkce=True,
        )

    @staticmethod
    def _resolve_secret(
        value: str | None,
        env_name: str | None,
    ) -> str:
        if value:
            return value
        if env_name:
            env_value = os.getenv(env_name)
            if env_value:
                return env_value
        raise MCPOAuthError(f"Missing required OAuth credential: {env_name or 'client_id'}")

    def _expand_redirect_uri(self, configured: str | None, *, default: str) -> str:
        if not configured:
            return default
        expanded = os.path.expandvars(configured)
        placeholder = "${ATOMSAGENT_URL:-http://localhost:3284}"
        if placeholder in expanded:
            expanded = expanded.replace(placeholder, self._base_url)
        if "${ATOMSAGENT_URL}" in expanded:
            expanded = expanded.replace("${ATOMSAGENT_URL}", self._base_url)
        return expanded


# Dependency factory ---------------------------------------------------------

def create_mcp_oauth_service(repository: MCPOAuthRepository) -> MCPOAuthService:
    """Factory helper used by FastAPI dependencies."""
    return MCPOAuthService(
        repository=repository,
        base_url=settings.base_url if hasattr(settings, "base_url") else None,
    )

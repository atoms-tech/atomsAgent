"""
OAuth Handler for Remote MCP Servers

Implements OAuth 2.0 + PKCE flow for remote MCP servers.
Handles Dynamic Client Registration (DCR), token storage, and refresh.
"""

from __future__ import annotations

import base64
import hashlib
import secrets
from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import Any
from urllib.parse import urlencode

import httpx


@dataclass
class OAuthConfig:
    """OAuth configuration for an MCP server"""
    
    server_id: str
    authorization_url: str
    token_url: str
    client_id: str | None = None
    client_secret: str | None = None
    redirect_uri: str = "http://localhost:3000/oauth/callback"
    scopes: list[str] | None = None
    
    # DCR endpoints (if supported)
    registration_url: str | None = None
    
    # PKCE
    use_pkce: bool = True


@dataclass
class OAuthTokens:
    """OAuth tokens"""
    
    access_token: str
    refresh_token: str | None = None
    expires_at: datetime | None = None
    token_type: str = "Bearer"
    scope: str | None = None
    
    def is_expired(self) -> bool:
        """Check if access token is expired"""
        if self.expires_at is None:
            return False
        return datetime.now() >= self.expires_at
    
    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for storage"""
        return {
            "access_token": self.access_token,
            "refresh_token": self.refresh_token,
            "expires_at": self.expires_at.isoformat() if self.expires_at else None,
            "token_type": self.token_type,
            "scope": self.scope
        }
    
    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> OAuthTokens:
        """Create from dictionary"""
        expires_at = data.get("expires_at")
        if expires_at and isinstance(expires_at, str):
            expires_at = datetime.fromisoformat(expires_at)
        
        return cls(
            access_token=data["access_token"],
            refresh_token=data.get("refresh_token"),
            expires_at=expires_at,
            token_type=data.get("token_type", "Bearer"),
            scope=data.get("scope")
        )


class PKCEGenerator:
    """Generate PKCE code verifier and challenge"""
    
    @staticmethod
    def generate_code_verifier() -> str:
        """Generate a random code verifier"""
        return base64.urlsafe_b64encode(secrets.token_bytes(32)).decode('utf-8').rstrip('=')
    
    @staticmethod
    def generate_code_challenge(verifier: str) -> str:
        """Generate code challenge from verifier using S256"""
        digest = hashlib.sha256(verifier.encode('utf-8')).digest()
        return base64.urlsafe_b64encode(digest).decode('utf-8').rstrip('=')


class OAuthHandler:
    """Handles OAuth flows for MCP servers"""
    
    def __init__(self, supabase_client: Any = None):
        """
        Initialize OAuth handler.
        
        Args:
            supabase_client: Optional Supabase client for token storage
        """
        self.supabase = supabase_client
        self._token_store: dict[str, OAuthTokens] = {}
        self._state_store: dict[str, dict[str, Any]] = {}
    
    async def register_client(self, config: OAuthConfig) -> tuple[str, str]:
        """
        Perform Dynamic Client Registration (DCR).
        
        Args:
            config: OAuth configuration
        
        Returns:
            Tuple of (client_id, client_secret)
        """
        if not config.registration_url:
            raise ValueError("DCR not supported: no registration_url provided")
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                config.registration_url,
                json={
                    "client_name": f"atomsAgent MCP Server {config.server_id}",
                    "redirect_uris": [config.redirect_uri],
                    "grant_types": ["authorization_code", "refresh_token"],
                    "response_types": ["code"],
                    "token_endpoint_auth_method": "client_secret_basic"
                }
            )
            response.raise_for_status()
            data = response.json()
            
            return data["client_id"], data.get("client_secret", "")
    
    def generate_authorization_url(
        self,
        config: OAuthConfig,
        state: str | None = None
    ) -> tuple[str, str, str | None]:
        """
        Generate OAuth authorization URL.
        
        Args:
            config: OAuth configuration
            state: Optional state parameter (generated if not provided)
        
        Returns:
            Tuple of (authorization_url, state, code_verifier)
        """
        if state is None:
            state = secrets.token_urlsafe(32)
        
        params: dict[str, Any] = {
            "client_id": config.client_id,
            "redirect_uri": config.redirect_uri,
            "response_type": "code",
            "state": state
        }
        
        if config.scopes:
            params["scope"] = " ".join(config.scopes)
        
        code_verifier = None
        if config.use_pkce:
            code_verifier = PKCEGenerator.generate_code_verifier()
            code_challenge = PKCEGenerator.generate_code_challenge(code_verifier)
            params["code_challenge"] = code_challenge
            params["code_challenge_method"] = "S256"
            
            # Store code verifier for later
            self._state_store[state] = {
                "code_verifier": code_verifier,
                "server_id": config.server_id
            }
        
        url = f"{config.authorization_url}?{urlencode(params)}"
        return url, state, code_verifier
    
    async def exchange_code_for_tokens(
        self,
        config: OAuthConfig,
        code: str,
        state: str
    ) -> OAuthTokens:
        """
        Exchange authorization code for tokens.
        
        Args:
            config: OAuth configuration
            code: Authorization code
            state: State parameter
        
        Returns:
            OAuth tokens
        """
        # Get code verifier from state store
        state_data = self._state_store.get(state, {})
        code_verifier = state_data.get("code_verifier")
        
        params: dict[str, Any] = {
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": config.redirect_uri,
            "client_id": config.client_id
        }
        
        if code_verifier:
            params["code_verifier"] = code_verifier
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                config.token_url,
                data=params,
                auth=(config.client_id, config.client_secret) if config.client_secret else None
            )
            response.raise_for_status()
            data = response.json()
            
            # Calculate expiration
            expires_in = data.get("expires_in")
            expires_at = None
            if expires_in:
                expires_at = datetime.now() + timedelta(seconds=expires_in)
            
            tokens = OAuthTokens(
                access_token=data["access_token"],
                refresh_token=data.get("refresh_token"),
                expires_at=expires_at,
                token_type=data.get("token_type", "Bearer"),
                scope=data.get("scope")
            )
            
            # Store tokens
            await self.store_tokens(config.server_id, tokens)
            
            # Clean up state store
            self._state_store.pop(state, None)
            
            return tokens
    
    async def refresh_tokens(
        self,
        config: OAuthConfig,
        refresh_token: str
    ) -> OAuthTokens:
        """
        Refresh access token using refresh token.
        
        Args:
            config: OAuth configuration
            refresh_token: Refresh token
        
        Returns:
            New OAuth tokens
        """
        params = {
            "grant_type": "refresh_token",
            "refresh_token": refresh_token,
            "client_id": config.client_id
        }
        
        async with httpx.AsyncClient() as client:
            response = await client.post(
                config.token_url,
                data=params,
                auth=(config.client_id, config.client_secret) if config.client_secret else None
            )
            response.raise_for_status()
            data = response.json()
            
            expires_in = data.get("expires_in")
            expires_at = None
            if expires_in:
                expires_at = datetime.now() + timedelta(seconds=expires_in)
            
            tokens = OAuthTokens(
                access_token=data["access_token"],
                refresh_token=data.get("refresh_token", refresh_token),
                expires_at=expires_at,
                token_type=data.get("token_type", "Bearer"),
                scope=data.get("scope")
            )
            
            # Store new tokens
            await self.store_tokens(config.server_id, tokens)
            
            return tokens
    
    async def store_tokens(self, server_id: str, tokens: OAuthTokens) -> None:
        """
        Store OAuth tokens.
        
        Args:
            server_id: MCP server ID
            tokens: OAuth tokens
        """
        # Store in database if available
        if self.supabase:
            try:
                await self.supabase.update(
                    "mcp_servers",
                    filters={"id": f"eq.{server_id}"},
                    payload={
                        "auth_config": {"oauth_tokens": tokens.to_dict()},
                        "auth_status": "connected",
                    },
                )
                return
            except Exception as e:
                print(f"Error storing tokens in database: {e}")

        # Fall back to memory if Supabase storage failed or is unavailable
        self._token_store[server_id] = tokens
    
    async def get_tokens(self, server_id: str) -> OAuthTokens | None:
        """
        Retrieve OAuth tokens for a server.
        
        Args:
            server_id: MCP server ID
        
        Returns:
            OAuth tokens or None if not found
        """
        # Try database first
        if self.supabase:
            try:
                result = await self.supabase.select(
                    "mcp_servers",
                    columns="auth_config",
                    filters={"id": f"eq.{server_id}"},
                    limit=1,
                )
                if result.data:
                    auth_config = result.data[0].get("auth_config") or {}
                    oauth_tokens = auth_config.get("oauth_tokens")
                    if oauth_tokens:
                        return OAuthTokens.from_dict(oauth_tokens)
            except Exception as e:
                print(f"Error retrieving tokens from database: {e}")
        
        # Fall back to memory
        return self._token_store.get(server_id)
    
    async def get_valid_access_token(
        self,
        config: OAuthConfig
    ) -> str:
        """
        Get a valid access token, refreshing if necessary.
        
        Args:
            config: OAuth configuration
        
        Returns:
            Valid access token
        """
        tokens = await self.get_tokens(config.server_id)
        
        if tokens is None:
            raise ValueError(f"No tokens found for server {config.server_id}")
        
        # Refresh if expired
        if tokens.is_expired() and tokens.refresh_token:
            tokens = await self.refresh_tokens(config, tokens.refresh_token)
        
        return tokens.access_token

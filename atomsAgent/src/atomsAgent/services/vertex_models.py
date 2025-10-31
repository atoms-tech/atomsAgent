from __future__ import annotations

import json
import time
from dataclasses import dataclass

import httpx
from aiocache import Cache
from tenacity import AsyncRetrying, retry_if_exception_type, stop_after_attempt, wait_exponential

from atomsAgent.schemas.openai import ModelInfo, ModelListResponse

try:  # pragma: no cover - optional dependency
    from google.auth.transport.requests import Request  # type: ignore[import] # pyright: ignore[reportMissingImports] # mypy: ignore[import-not-found]
    from google.oauth2 import service_account  # type: ignore[import] # pyright: ignore[reportMissingImports] # mypy: ignore[import-not-found]
    from google.oauth2.credentials import Credentials as OAuth2Credentials  # type: ignore[import] # pyright: ignore[reportMissingImports] # mypy: ignore[import-not-found]
except ImportError:  # pragma: no cover
    service_account = None  # type: ignore
    Request = None  # type: ignore
    OAuth2Credentials = None  # type: ignore


@dataclass
class _CachedModels:
    timestamp: float
    data: list[ModelInfo]


class VertexModelService:
    """Discover and cache available Vertex AI models."""

    def __init__(
        self,
        *,
        project_id: str,
        location: str,
        cache_ttl: int = 600,
        credentials_path: str | None = None,
        credentials_json: str | None = None,
        cache: Cache | None = None,
    ) -> None:
        self.project_id = project_id
        self.location = location
        self.cache_ttl = cache_ttl
        self._credentials_path = credentials_path
        self._credentials_json = credentials_json
        self._cache = cache if cache is not None else Cache(Cache.MEMORY)
        self._cache_key = f"vertex-models:{project_id}:{location}"

    async def list_models(self) -> ModelListResponse:
        """Return cached list of models or fetch from Vertex AI."""
        cached = await self._cache.get(self._cache_key)  # type: ignore
        if (
            cached
            and isinstance(cached, _CachedModels)
            and (time.time() - cached.timestamp) < self.cache_ttl
        ):
            return ModelListResponse(data=cached.data)

        models = await self._fetch_models()
        await self._cache.set(  # type: ignore
            self._cache_key,
            _CachedModels(timestamp=time.time(), data=models),
            ttl=self.cache_ttl,
        )
        return ModelListResponse(data=models)

    async def _fetch_models_from_publisher(
        self, access_token: str, publisher: str
    ) -> list[ModelInfo]:
        """Fetch models from a specific publisher."""
        endpoint = (
            f"https://{self.location}-aiplatform.googleapis.com/v1beta1/"
            f"publishers/{publisher}/models"
        )

        async for attempt in AsyncRetrying(
            stop=stop_after_attempt(3),
            wait=wait_exponential(multiplier=1, min=1, max=4),
            retry=retry_if_exception_type(httpx.HTTPError),
            reraise=True,
        ):
            with attempt:
                async with httpx.AsyncClient(timeout=30.0) as client:
                    headers = {
                        "Authorization": f"Bearer {access_token}",
                        "x-goog-user-project": self.project_id,  # Set quota project explicitly
                    }
                    response = await client.get(
                        endpoint,
                        headers=headers,
                        params={
                            "pageSize": 100,  # Get up to 100 models
                        },
                    )

                    if response.status_code != 200:
                        print(f"Vertex API error: {response.status_code} - {response.text}")
                        return []

                    payload = response.json()
                    print(f"Vertex API response: {json.dumps(payload, indent=2)}")

                    # Handle empty response or missing publisherModels key
                    if not payload or "publisherModels" not in payload:
                        print("No publisherModels key in response")
                        return []

                    items = payload.get("publisherModels", [])
                    if not items:
                        print("No models found in response")
                        return []

                    models: list[ModelInfo] = []
                    for item in items:
                        # Extract model name from the full resource name
                        # Format: publishers/anthropic/models/claude-sonnet-4-5
                        full_model_name = item.get("name", "")
                        if not full_model_name:
                            continue

                        # Extract just the model name from the full path
                        # publishers/anthropic/models/claude-sonnet-4-5 -> claude-sonnet-4-5
                        model_name = full_model_name.split("/")[-1]

                        # Get version ID to construct the SDK-compatible model identifier
                        # Claude Agent SDK expects format: claude-sonnet-4-5@20250929
                        version_id = item.get("versionId", "")

                        # Construct the model ID with version (short format for SDK)
                        if version_id:
                            model_id = f"{model_name}@{version_id}"
                        else:
                            model_id = model_name

                        # Extract display name and description
                        display_name = item.get("displayName", model_name)
                        description = item.get("description", display_name)

                        # Extract publisher (should be "google")
                        publisher = item.get("publisher", "google")

                        # Determine capabilities based on model name
                        capabilities = ["chat.completion"]
                        model_lower = model_id.lower()
                        if "thinking" in model_lower or "2.0" in model_lower:
                            capabilities.append("reasoning")

                        # Determine context length based on model type
                        context_length = 1048576  # Default 1M tokens
                        if "pro" in model_lower:
                            context_length = 2097152  # 2M tokens for Pro models

                        models.append(
                            ModelInfo(
                                id=model_id,
                                owned_by=publisher,
                                created=int(time.time()),
                                description=description,
                                context_length=context_length,
                                provider="vertexai",
                                capabilities=capabilities,
                            )
                        )

                    return models

        # Fallback return (should not be reached, but satisfies type checker)
        return []

    def _should_include_model(self, model_id: str) -> bool:
        """
        Filter models to only include Gemini 2.5 and Claude 4.5 variants.

        Returns True if the model should be included in the list.
        """
        model_lower = model_id.lower()

        # Include Gemini 2.5 variants (pro, flash, flash-lite)
        if "gemini-2.5" in model_lower or "gemini-2-5" in model_lower:
            return True

        # Include Claude 4.5 variants (sonnet, haiku)
        if "claude" in model_lower:
            # Check for 4.5, 4-5, or version numbers like 20250929
            if "4.5" in model_lower or "4-5" in model_lower:
                return True
            # Also check for sonnet-4-5 or haiku-4-5 patterns
            if "sonnet-4-5" in model_lower or "haiku-4-5" in model_lower:
                return True

        return False

    async def _fetch_models(self) -> list[ModelInfo]:
        """Fetch models from Google AI Platform publisher models API."""
        access_token = await self._get_access_token()
        if not access_token:
            error_msg = "No access token available - cannot fetch models from Vertex AI"
            print(f"ERROR: {error_msg}")
            raise RuntimeError(error_msg)

        # Fetch models from multiple publishers
        all_models: list[ModelInfo] = []

        # Fetch Google models (filtered for Gemini 2.5)
        try:
            google_models = await self._fetch_models_from_publisher(access_token, "google")
            filtered_google = [m for m in google_models if self._should_include_model(m.id)]
            all_models.extend(filtered_google)
            print(f"Fetched {len(filtered_google)} Gemini 2.5 models from Google publisher (filtered from {len(google_models)} total)")
        except Exception as e:
            print(f"Failed to fetch Google models: {e}")

        # Fetch Anthropic (Claude) models (filtered for Claude 4.5)
        try:
            anthropic_models = await self._fetch_models_from_publisher(access_token, "anthropic")
            filtered_anthropic = [m for m in anthropic_models if self._should_include_model(m.id)]
            all_models.extend(filtered_anthropic)
            print(f"Fetched {len(filtered_anthropic)} Claude 4.5 models from Anthropic publisher (filtered from {len(anthropic_models)} total)")
        except Exception as e:
            print(f"Failed to fetch Anthropic models: {e}")

        if not all_models:
            error_msg = "No models available from Vertex AI in the configured region"
            print(f"ERROR: {error_msg}")
            raise RuntimeError(error_msg)

        print(f"Successfully fetched {len(all_models)} filtered models from Vertex AI")
        return all_models

    async def _get_access_token(self) -> str | None:
        if service_account is None or Request is None:
            return None

        credentials_data = None
        if self._credentials_json:
            credentials_data = json.loads(self._credentials_json)
        elif self._credentials_path:
            try:
                with open(self._credentials_path, encoding="utf-8") as fp:
                    credentials_data = json.load(fp)
            except FileNotFoundError:
                return None

        # If no explicit credentials provided, try Application Default Credentials (ADC)
        if not credentials_data:
            try:
                import google.auth  # type: ignore[import]

                credentials, project = google.auth.default(
                    scopes=["https://www.googleapis.com/auth/cloud-platform"],
                    quota_project_id=self.project_id,
                )
                credentials.refresh(Request())
                return credentials.token
            except Exception as e:
                print(f"Failed to get Application Default Credentials: {e}")
                return None

        # Handle both service account and OAuth2 credentials
        if credentials_data.get("type") == "service_account":
            # Service account credentials
            credentials = service_account.Credentials.from_service_account_info(
                credentials_data,
                scopes=["https://www.googleapis.com/auth/cloud-platform"],
            )
            credentials.refresh(Request())
            return credentials.token
        elif credentials_data.get("type") == "authorized_user":
            # OAuth2 user credentials (client credentials flow)
            try:
                from google.oauth2.credentials import Credentials as OAuth2Credentials  # type: ignore[import] # pyright: ignore[reportMissingImports] # mypy: ignore[import-not-found]

                credentials = OAuth2Credentials.from_authorized_user_info(
                    credentials_data,
                    scopes=["https://www.googleapis.com/auth/cloud-platform"],
                )
                credentials.refresh(Request())
                return credentials.token
            except Exception:
                # Fallback: try using the refresh token directly
                return await self._refresh_oauth2_token(credentials_data)

        return None

    async def _refresh_oauth2_token(self, credentials_data: dict) -> str | None:
        """Refresh OAuth2 token using refresh token."""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    "https://oauth2.googleapis.com/token",
                    data={
                        "client_id": credentials_data["client_id"],
                        "client_secret": credentials_data["client_secret"],
                        "refresh_token": credentials_data["refresh_token"],
                        "grant_type": "refresh_token",
                    },
                )
                if response.status_code == 200:
                    token_data = response.json()
                    return token_data.get("access_token")
        except Exception:
            pass
        return None

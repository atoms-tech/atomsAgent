from __future__ import annotations

from functools import lru_cache

from atomsAgent.db.supabase import SupabaseClient
from atomsAgent.settings.secrets import get_secrets


@lru_cache
def get_supabase_client() -> SupabaseClient:
    """Return a cached Supabase REST client configured from secrets."""
    secrets = get_secrets()
    url = getattr(secrets, "supabase_url", None)
    key = getattr(secrets, "supabase_service_key", None)

    if not url or not key:
        raise RuntimeError(
            "Supabase credentials not configured. "
            "Set supabase_url and supabase_service_key in config/secrets.yml.",
        )

    return SupabaseClient(url=url, service_role_key=key)

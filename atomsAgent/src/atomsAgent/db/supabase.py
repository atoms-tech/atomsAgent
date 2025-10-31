from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any

import httpx


@dataclass(slots=True)
class SupabaseResponse:
    data: Any
    count: int | None = None


class SupabaseError(RuntimeError):
    pass


class SupabaseClient:
    """HTTP-based Supabase client using the REST interface."""

    def __init__(self, *, url: str, service_role_key: str, schema: str = "public") -> None:
        if not url or not service_role_key:
            raise ValueError("Supabase URL and service role key are required")
        self.base_url = url.rstrip("/") + "/rest/v1"
        self.rpc_url = url.rstrip("/") + "/rest/v1/rpc"
        self.service_role_key = service_role_key
        self.schema = schema
        self._default_headers = {
            "apikey": service_role_key,
            "Authorization": f"Bearer {service_role_key}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "Prefer": "return=representation",
            "Accept-Profile": schema,
            "Content-Profile": schema,
            # Bypass RLS for service role
            "X-Forwarded-For": "127.0.0.1",
        }

    async def select(
        self,
        table: str,
        *,
        columns: str = "*",
        filters: dict[str, str] | None = None,
        order: list[str] | None = None,
        limit: int | None = None,
        offset: int | None = None,
        count: bool = False,
    ) -> SupabaseResponse:
        params: dict[str, str] = {"select": columns}
        if filters:
            params.update(filters)
        if order:
            params["order"] = ",".join(order)
        if limit is not None:
            params["limit"] = str(limit)
        if offset is not None:
            params["offset"] = str(offset)
        headers = dict(self._default_headers)
        if count:
            headers["Prefer"] = headers["Prefer"] + ",count=exact"

        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{self.base_url}/{table}",
                params=params,
                headers=headers,
            )
        self._raise_for_status(response)
        total = self._extract_count(response) if count else None
        return SupabaseResponse(data=response.json(), count=total)

    async def insert(self, table: str, payload: dict[str, Any]) -> SupabaseResponse:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/{table}",
                headers=self._default_headers,
                content=json.dumps(payload),
            )
        self._raise_for_status(response)
        return SupabaseResponse(data=response.json())

    async def update(
        self,
        table: str,
        *,
        filters: dict[str, str],
        payload: dict[str, Any],
    ) -> SupabaseResponse:
        async with httpx.AsyncClient() as client:
            response = await client.patch(
                f"{self.base_url}/{table}",
                params=filters,
                headers=self._default_headers,
                content=json.dumps(payload),
            )
        self._raise_for_status(response)
        return SupabaseResponse(data=response.json())

    async def delete(self, table: str, *, filters: dict[str, str]) -> SupabaseResponse:
        async with httpx.AsyncClient() as client:
            response = await client.delete(
                f"{self.base_url}/{table}",
                params=filters,
                headers=self._default_headers,
            )
        self._raise_for_status(response)
        return SupabaseResponse(data=response.json())

    async def rpc(
        self, function_name: str, *, params: dict[str, Any] | None = None
    ) -> SupabaseResponse:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.rpc_url}/{function_name}",
                headers=self._default_headers,
                content=json.dumps(params or {}),
            )
        self._raise_for_status(response)
        return SupabaseResponse(data=response.json())

    @staticmethod
    def _raise_for_status(response: httpx.Response) -> None:
        if response.status_code >= 400:
            try:
                payload = response.json()
            except ValueError:
                payload = response.text
            raise SupabaseError(f"Supabase error {response.status_code}: {payload}")

    @staticmethod
    def _extract_count(response: httpx.Response) -> int | None:
        content_range = response.headers.get("content-range")
        if not content_range:
            return None
        try:
            _, total = content_range.split("/")
            return int(total)
        except ValueError:
            return None

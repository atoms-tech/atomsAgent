from __future__ import annotations

import asyncio
import os
import shutil
from dataclasses import dataclass
from pathlib import Path


@dataclass(slots=True)
class SandboxContext:
    """Represents a per-session sandbox environment."""

    sandbox_id: str
    workspace_path: Path
    created_by: str | None = None


class SandboxManager:
    """Creates and tracks sandbox workspaces inside the root container."""

    def __init__(self, *, root_path: str) -> None:
        self.root_path = Path(root_path).expanduser().resolve()
        self.root_path.mkdir(parents=True, exist_ok=True)
        self._locks: dict[str, asyncio.Lock] = {}

    def _get_lock(self, sandbox_id: str) -> asyncio.Lock:
        if sandbox_id not in self._locks:
            self._locks[sandbox_id] = asyncio.Lock()
        return self._locks[sandbox_id]

    async def acquire(self, sandbox_id: str, *, created_by: str | None = None) -> SandboxContext:
        """Ensure a workspace directory exists for the sandbox."""
        lock = self._get_lock(sandbox_id)
        async with lock:
            workspace = self.root_path / sandbox_id
            workspace.mkdir(parents=True, exist_ok=True)
            # Ensure workspaces have restrictive permissions
            os.chmod(workspace, 0o700)
            return SandboxContext(
                sandbox_id=sandbox_id, workspace_path=workspace, created_by=created_by
            )

    async def reset(self, sandbox_id: str) -> None:
        """Delete and recreate the sandbox workspace."""
        lock = self._get_lock(sandbox_id)
        async with lock:
            workspace = self.root_path / sandbox_id
            if workspace.exists():
                shutil.rmtree(workspace, ignore_errors=True)
            workspace.mkdir(parents=True, exist_ok=True)
            os.chmod(workspace, 0o700)

    async def release(self, sandbox_id: str, *, delete: bool = False) -> None:
        """Optionally delete sandbox workspace when no longer needed."""
        lock = self._get_lock(sandbox_id)
        async with lock:
            if delete:
                workspace = self.root_path / sandbox_id
                if workspace.exists():
                    shutil.rmtree(workspace, ignore_errors=True)

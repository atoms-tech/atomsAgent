from __future__ import annotations

from collections.abc import Iterable
from uuid import UUID

from jinja2 import Environment, StrictUndefined


class PromptOrchestrator:
    """Compose system prompts from platform, organization, and user scopes."""

    def __init__(
        self,
        *,
        prompt_repository: PromptRepository | None = None,
        platform_prompt: str | None = None,
        workflow_prompts: dict[str, str] | None = None,
    ) -> None:
        self._prompt_repository = prompt_repository
        self._platform_prompt = platform_prompt
        self._workflow_prompts = workflow_prompts or {}
        self._jinja_env = Environment(autoescape=False, undefined=StrictUndefined)

    async def compose_prompt(
        self,
        *,
        organization_id: str | None,
        user_id: str | None,
        workflow: str | None = None,
        variables: dict | None = None,
    ) -> str:
        prompts: list[str] = []

        if self._platform_prompt:
            prompts.append(self._platform_prompt)

        if self._prompt_repository:
            org_uuid = UUID(organization_id) if organization_id else None
            user_uuid = UUID(user_id) if user_id else None
            scoped_prompts = await self._prompt_repository.list_prompts(
                organization_id=org_uuid,
                user_id=user_uuid,
            )
            prompts.extend(self._render_templates(scoped_prompts, variables))

        if workflow and workflow in self._workflow_prompts:
            prompts.append(self._workflow_prompts[workflow])

        merged = "\n\n".join(p.strip() for p in prompts if p and p.strip())
        return merged.strip()

    @staticmethod
    def _render_templates(prompts: Iterable[PromptRecord], variables: dict | None) -> list[str]:
        rendered: list[str] = []
        safe_variables = variables or {}
        jinja_env = Environment(autoescape=False, undefined=StrictUndefined)
        for prompt in prompts:
            content = getattr(prompt, "content", "") or ""
            template_str = getattr(prompt, "template", None)
            if template_str:
                template = jinja_env.from_string(template_str)
                content = template.render(**safe_variables)
            try:
                template = jinja_env.from_string(content)
                rendered.append(template.render(**safe_variables))
            except Exception:
                rendered.append(content)
        return rendered


# Avoid circular imports by using TYPE_CHECKING guard
try:  # pragma: no cover - optional import for type hints
    from atomsAgent.db.repositories import PromptRecord, PromptRepository  # type: ignore
except ImportError:  # pragma: no cover
    from uuid import UUID

    class PromptRepository:  # type: ignore
        async def list_prompts(
            self, *, organization_id: UUID | None, user_id: UUID | None
        ) -> list[PromptRecord]:
            return []

    class PromptRecord:  # type: ignore
        content: str = ""

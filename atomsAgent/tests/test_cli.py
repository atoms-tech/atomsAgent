from __future__ import annotations

from uuid import UUID

from typer.testing import CliRunner

from atomsAgent.cli.main import app

runner = CliRunner()


class _FakeModel:
    def __init__(self, id: str, provider: str) -> None:
        self.id = id
        self.provider = provider
        self.owned_by = provider
        self.capabilities = ["chat.completion"]
        self.created = 1

    def model_dump(self) -> dict[str, str | int | list[str]]:
        return {
            "id": self.id,
            "provider": self.provider,
            "owned_by": self.owned_by,
            "capabilities": self.capabilities,
            "created": self.created,
        }


def test_vertex_models_json(monkeypatch):
    class _FakeService:
        async def list_models(self):  # pragma: no cover - simple stub
            class _Response:
                def __init__(self) -> None:
                    self.data = [_FakeModel("model-x", "vertexai")]

            return _Response()

    monkeypatch.setattr("atomsAgent.cli.main._load_vertex_service", lambda: _FakeService())
    result = runner.invoke(app, ["vertex", "models", "--json"])
    assert result.exit_code == 0
    assert "model-x" in result.stdout


def test_mcp_list(monkeypatch):
    class _FakeResponse:
        def __init__(self) -> None:
            self.items = []

    class _FakeService:
        async def list(self, **kwargs):  # pragma: no cover
            return _FakeResponse()

        async def create(self, payload):
            return payload

        async def update(self, config_id, payload):
            return payload

        async def delete(self, config_id):
            return None

    monkeypatch.setattr("atomsAgent.cli.main._load_mcp_service", lambda: _FakeService())
    result = runner.invoke(app, ["mcp", "list", "--org", str(UUID(int=0))])
    assert result.exit_code == 0


def test_supabase_generate_models(monkeypatch, tmp_path):
    schema = tmp_path / "schema.sql"
    schema.write_text("-- test schema")
    out_dir = tmp_path / "models"

    class _FakeCompleted:
        returncode = 0
        stderr = ""

    monkeypatch.setattr(
        "atomsAgent.cli.main.subprocess.run", lambda *args, **kwargs: _FakeCompleted()
    )
    result = runner.invoke(
        app,
        [
            "supabase",
            "generate-models",
            "--schema",
            str(schema),
            "--output",
            str(out_dir),
        ],
    )
    assert result.exit_code == 0


def test_prompt_show_json(monkeypatch):
    class _FakePrompt:
        def __init__(self, content: str) -> None:
            self.scope = "global"
            self.priority = 10
            self.content = content

    class _FakeRepo:
        async def list_prompts(self, **kwargs):  # pragma: no cover
            return [_FakePrompt("Hello")]  # type: ignore[list-item]

    class _FakeOrchestrator:
        def __init__(self) -> None:
            self._prompt_repository = _FakeRepo()

        async def compose_prompt(self, **kwargs):  # pragma: no cover
            return "Resolved prompt"

    monkeypatch.setattr(
        "atomsAgent.cli.main._load_prompt_orchestrator", lambda: _FakeOrchestrator()
    )
    result = runner.invoke(app, ["prompt", "show", "--org", str(UUID(int=1)), "--json"])
    assert result.exit_code == 0
    assert "Resolved prompt" in result.stdout


def test_server_run(monkeypatch):
    called = {}

    def fake_run(*args, **kwargs):  # pragma: no cover
        called["value"] = (args, kwargs)

    monkeypatch.setattr("atomsAgent.cli.main.uvicorn.run", fake_run)
    result = runner.invoke(app, ["server", "run", "--host", "0.0.0.0", "--port", "9000"])
    assert result.exit_code == 0
    assert "value" in called

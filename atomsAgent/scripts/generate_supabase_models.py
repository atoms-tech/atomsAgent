#!/usr/bin/env python3
"""Regenerate Supabase Pydantic models for atomsAgent.

Configuration order of precedence:
1. Environment variables (``SUPABASE_DB_URL`` or credential pieces)
2. ``.env`` / ``.env.local`` from the atoms.tech workspace
3. ``config/secrets.yml`` in this repository
4. ``../atoms-mcp-prod/secrets.yml``

If a full database URL is not supplied, the script assembles one from the
available credential pieces. Only ``supabase_pydantic`` is required.
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterable

try:
    import yaml  # type: ignore
except ImportError:  # pragma: no cover
    yaml = None

DEFAULT_HOST = "aws-0-us-west-1.pooler.supabase.com"
DEFAULT_PORT = 6543
DEFAULT_DB = "postgres"
DEFAULT_USER = "postgres.ydogoylwenufckscqijp"

REPO_ROOT = Path(__file__).resolve().parents[1]
TARGET_DIR = REPO_ROOT / "src/atomsAgent/db/generated"
SCHEMA_FILE = TARGET_DIR / "fastapi" / "schema_public_latest.py"

ENV_FILES: tuple[Path, ...] = (
    REPO_ROOT.parent / "atoms.tech/.env.local",
    REPO_ROOT.parent / "atoms.tech/.env",
    REPO_ROOT.parents[1] / "atoms.tech/.env.local",
    REPO_ROOT.parents[1] / "atoms.tech/.env",
    REPO_ROOT.parents[2] / "clean/deploy/atoms.tech/.env.local",
    REPO_ROOT.parents[2] / "clean/deploy/atoms.tech/.env",
    REPO_ROOT / ".env.local",
    REPO_ROOT / ".env",
)

YAML_FILES: tuple[Path, ...] = (
    REPO_ROOT / "config/secrets.yml",
    REPO_ROOT.parent / "atoms-mcp-prod/secrets.yml",
    REPO_ROOT.parents[1] / "atoms-mcp-prod/secrets.yml",
)

_CONFIG_CACHE: dict[str, str] | None = None


def load_text_file(path: Path) -> list[str]:
    try:
        return path.read_text().splitlines()
    except FileNotFoundError:
        return []


def parse_env_file(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for line in load_text_file(path):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if "=" not in stripped:
            continue
        key, value = stripped.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        values[key] = value
    return values


def parse_yaml_file(path: Path) -> dict[str, str]:
    if not path.exists():
        return {}
    if yaml is not None:
        try:
            data = yaml.safe_load(path.read_text())
        except Exception:
            data = None
    else:  # pragma: no cover - tiny fallback parser
        data = None

    if isinstance(data, dict):
        flattened: dict[str, str] = {}

        def _flatten(prefix: str, obj: object) -> None:
            if isinstance(obj, dict):
                for k, v in obj.items():
                    new_prefix = f"{prefix}{k}" if not prefix else f"{prefix}.{k}"
                    _flatten(new_prefix, v)
            elif obj is not None:
                flattened[prefix] = str(obj)

        for key, value in data.items():
            _flatten(key, value)
        return flattened

    # Fallback: treat file as simple key/value pairs
    pairs: dict[str, str] = {}
    for line in load_text_file(path):
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or ":" not in stripped:
            continue
        key, value = stripped.split(":", 1)
        pairs[key.strip()] = value.strip().strip('"').strip("'")
    return pairs


def combined_config() -> dict[str, str]:
    global _CONFIG_CACHE
    if _CONFIG_CACHE is not None:
        return _CONFIG_CACHE
    config: dict[str, str] = {}
    for yaml_path in YAML_FILES:
        config.update(parse_yaml_file(yaml_path))
    for env_path in ENV_FILES:
        config.update(parse_env_file(env_path))
    config.update({k: v for k, v in os.environ.items()})
    _CONFIG_CACHE = config
    return config


def build_db_url() -> str:
    config = combined_config()
    if env_url := config.get("SUPABASE_DB_URL"):
        return env_url
    password = config.get("SUPABASE_DB_PASSWORD")
    if not password:
        raise RuntimeError(
            "Unable to locate SUPABASE_DB_URL or SUPABASE_DB_PASSWORD in environment/.env/secrets"
        )
    host = config.get("SUPABASE_DB_HOST", DEFAULT_HOST)
    port = int(config.get("SUPABASE_DB_PORT", DEFAULT_PORT))
    user = config.get("SUPABASE_DB_USER", DEFAULT_USER)
    db = config.get("SUPABASE_DB_NAME", DEFAULT_DB)
    query = "sslmode=require"
    return f"postgresql://{user}:{password}@{host}:{port}/{db}?{query}"


def run_supabase_pydantic(db_url: str) -> None:
    TARGET_DIR.mkdir(parents=True, exist_ok=True)
    cmd: list[str] = [
        sys.executable,
        "-m",
        "supabase_pydantic",
        "gen",
        "--db-url",
        db_url,
        "--db-type",
        "postgres",
        "--schema",
        "public",
        "--type",
        "pydantic",
        "--framework",
        "fastapi",
        "--dir",
        str(TARGET_DIR),
        "--singular-names",
    ]
    print("Running:", " ".join(cmd))
    subprocess.run(cmd, check=True)


def _clean_multiline_string_literals(text: str) -> str:
    pattern = re.compile(r'"(?:[^"\\]|\\.|\n)*?"', re.DOTALL)

    def replace(match: re.Match[str]) -> str:
        value = match.group(0)
        if value.startswith('"""'):
            return value
        if "\n" not in value:
            return value
        inner = " ".join(value.strip('"').split())
        return f'"{inner}"'

    return pattern.sub(replace, text)


def post_process_schema() -> None:
    if not SCHEMA_FILE.exists():
        raise FileNotFoundError(f"Generated schema missing at {SCHEMA_FILE}")
    text = SCHEMA_FILE.read_text()
    text = _clean_multiline_string_literals(text)
    lines: list[str] = []
    seen_any = False
    for line in text.splitlines():
        if line.strip() == "from typing import Any":
            if seen_any:
                continue
            seen_any = True
        lines.append(line)
    SCHEMA_FILE.write_text("\n".join(lines) + "\n")


def main() -> None:
    db_url = build_db_url()
    print(f"Using database URL: {db_url.split('@')[-1]}")
    run_supabase_pydantic(db_url)
    post_process_schema()
    print(f"✅ Supabase models updated in {TARGET_DIR}")


if __name__ == "__main__":
    main()

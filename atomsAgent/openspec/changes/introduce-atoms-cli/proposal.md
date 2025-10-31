# Proposal: Introduce atomsAgent CLI

## Why
- Engineers currently interact with atomsAgent via disparate scripts (`codegen`, local helpers) and direct FastAPI runs. There is no cohesive CLI entry point.
- Common workflows (refresh Supabase models, inspect available Vertex models, manage MCP configs, launch dev server) require memorising commands or python snippets.
- Providing an official CLI with Typer + Rich aligns with the project’s tooling stack and improves UX with consistent flags, structured output, and discoverability.

## What Changes
- Ship a `atoms-agent` Typer CLI packaging canonical commands: generate Supabase models, list Vertex models, summarize prompts, manage MCP registrations, and run health checks.
- Wrap commands with Rich output (tables/progress) and expose flag-driven configuration (project id, location, output formats) instead of environment shims.
- Document the CLI usage in README and ensure CLI entry point is wired into packaging (`[project.scripts]`).
- Provide unit coverage for CLI command surfaces (Typer runner) to ensure options/responses behave as specified.

## Impact
- Developers gain a single entry point (`atoms-agent ...`) for routine tasks.
- Improves onboarding and operational ergonomics; reduces risk of running stale scripts.
- Requires packaging adjustments (setup scripts) and new dependencies (Typer, Rich already used in codegen).

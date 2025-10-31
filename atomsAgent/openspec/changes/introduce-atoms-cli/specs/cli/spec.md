## ADDED Requirements

### Requirement: atoms-agent CLI exposes canonical workflows
The CLI MUST provide a Typer-based `atoms-agent` executable that centralises developer and operator workflows with Rich-enhanced UX and flag-driven configuration.

#### Scenario: Generate Supabase models
- **GIVEN** the project is installed with atomsAgent dependencies
- **WHEN** `atoms-agent supabase generate-models --schema database/schema.sql --output atomsAgent/db/models.py` is executed
- **THEN** the CLI regenerates Supabase Pydantic models using `sb_pydantic`, showing progress with Rich, and writes the output file indicated by `--output`
- **AND** the command exits with code `0` and logs errors with a non-zero exit code when generation fails.

#### Scenario: List Vertex models
- **WHEN** `atoms-agent vertex models` runs without flags
- **THEN** the CLI fetches available models via the Vertex model service and renders a Rich table with columns `id`, `provider`, `owned_by`, `capabilities`, `created`
- **AND** providing `--json` outputs compact JSON instead of a table.

#### Scenario: Manage MCP configurations
- **WHEN** `atoms-agent mcp list --org <uuid>` executes
- **THEN** the CLI queries MCP configurations scoped to the specified organization (and user when `--user` supplied) and displays the result in a Rich table or JSON when `--json`
- **AND** `mcp create`, `update`, and `delete` subcommands accept flags rather than environment variables, validate scope (platform/org/user), and report success/failure with appropriate exit codes.

#### Scenario: Show prompts for an org/user
- **WHEN** `atoms-agent prompt show --org <uuid>` runs (optionally with `--user` and `--workflow`)
- **THEN** the CLI resolves hierarchical prompts via the prompt orchestrator and prints merged content in Rich formatting or JSON when requested.

#### Scenario: Run development server via CLI
- **WHEN** `atoms-agent server run --host 0.0.0.0 --port 3284 --reload`
- **THEN** the CLI invokes Uvicorn to run `atomsAgent.main:app` with the provided arguments, surfacing startup status and propagating any startup errors.

#### Scenario: Provide consistent help and exit semantics
- **WHEN** a user runs `atoms-agent --help`
- **THEN** the CLI surfaces Typer-generated help that lists all top-level and nested subcommands with descriptions, ensuring each command has descriptive help text and flag usage instructions.

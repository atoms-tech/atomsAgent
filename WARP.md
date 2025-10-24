# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Build Commands

**Primary build commands:**
- `make build` - Full build: embeds Next.js chat UI and builds Go binary to `out/agentapi`
- `make embed` - Build and embed Next.js chat UI into Go static assets
- `go build -o out/agentapi main.go` - Direct Go build (without chat UI embedding)
- `go generate ./...` - Generate OpenAPI schema and version info

**Development and testing:**
- `go test ./...` - Run all Go tests
- `CGO_ENABLED=0 go test -count=1 -v ./...` - Run tests with verbose output (CI style)
- `make gen` - Generate OpenAPI schema and update version info
- `make lint` - Run all linters (Go, TypeScript, shell, GitHub Actions)

**Chat UI development:**
- `cd chat && bun install` - Install chat UI dependencies
- `cd chat && bun run dev` - Start Next.js dev server with Turbopack
- `cd chat && bun lint` - Lint TypeScript/React code

## Architecture Overview

AgentAPI is a Go HTTP server that provides programmatic control over terminal-based coding agents (Claude Code, Aider, Goose, etc.) through an in-memory terminal emulator.

**Core Architecture:**
- **Terminal Emulation**: Runs agents in pseudo-terminals, translates HTTP API calls to terminal keystrokes
- **Message Parsing**: Analyzes terminal output diffs to extract structured agent messages
- **Agent Type Support**: Handles message formatting differences across 11+ agent types
- **Embedded Web UI**: Next.js chat interface compiled into Go binary as static assets

**Key Components:**
- `main.go` + `cmd/` - Cobra CLI framework with server/attach commands
- `lib/httpapi/` - HTTP server, OpenAPI schema, SSE events
- `lib/termexec/` - Terminal process execution and management  
- `lib/screentracker/` - Terminal output parsing and message splitting logic
- `lib/msgfmt/` - Agent-specific message formatting and user input removal
- `chat/` - Next.js TypeScript web UI (gets embedded into Go binary)

**Message Flow:**
1. HTTP POST `/message` → terminal snapshot taken
2. Message sent as keystrokes to agent's terminal
3. Terminal output changes tracked via diffs
4. New content parsed into structured agent response
5. SSE events stream updates to connected clients

## Agent Types and Message Formatting

The system supports multiple agent types with different terminal UI patterns:

**Auto-detected agents:**
- `claude` (default), `goose`, `aider`, `warp`, `droid`

**Explicit type required (use `--type=<agent>`):**
- `codex`, `gemini`, `copilot`, `amp`, `cursor`, `auggie`, `amazonq`, `opencode`, `custom`

Each agent type has specialized message formatting in `lib/msgfmt/` to handle:
- User input echo removal (agents often repeat user input)
- TUI element stripping (input boxes, borders, etc.)
- Agent-specific terminal layouts

## API Endpoints

- `GET /messages` - Get conversation history
- `POST /message` - Send message to agent (`{"content": "...", "type": "user"}`)
- `GET /status` - Agent status: `"stable"` or `"running"`
- `GET /events` - SSE stream of message/status updates
- `GET /docs` - OpenAPI documentation UI
- `GET /chat` - Embedded Next.js web interface

## Development Patterns

**Testing:** 
- Tests are co-located with source files (e.g., `server_test.go` alongside `server.go`)
- Includes unit tests and E2E tests in `e2e/`
- Test with `go test ./...` or specific packages

**Code Generation:**
- OpenAPI schema auto-generated to `openapi.json`
- Version info managed via `version.sh` script
- Use `go generate ./...` to regenerate all

**Linting:**
- Go: golangci-lint with exhaustive switch checks
- TypeScript: ESLint via Next.js config  
- Shell: shellcheck for bash scripts
- GitHub Actions: actionlint

**Terminal Emulation Details:**
- Uses `github.com/ActiveState/termtest/xpty` for pseudo-terminals
- Terminal state changes tracked via screen diffs
- Message boundaries detected by comparing terminal snapshots before/after user input

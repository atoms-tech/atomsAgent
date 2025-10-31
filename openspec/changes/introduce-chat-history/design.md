# Design: Chat History Management

## Overview
The change introduces a history access surface inside the chat UI and supporting API scaffolding. The UI should expose a hoverable control near the conversation header that opens a combobox listing recent sessions; clicking the control brings users to a dedicated page with tabs for "Recent" and future categories. agentapi must expose session metadata endpoints with stubbed data until persistence is available.

## Architecture Considerations
- **Front-end (Next.js)**: Add a history trigger component, route for the history page, and client hooks to fetch session lists. Favor reusable types in `chat/src/lib` to share between components.
- **Agent API (Go + Python services)**: Provide REST endpoints under `/sessions` to list history and fetch transcripts. Implement placeholder handlers that return empty arrays with TODO markers. Ensure SSE stream remains unaffected.
- **Storage**: Document expectation to use Redis for fast lookups, with Supabase as a backup option if Redis unavailable. For now, stub repository interfaces without concrete storage.
- **Claude Agent Integration**: Ensure session IDs returned align with Claude session management so resuming a chat reuses the right session identifier.

## Trade-offs
- Providing stubs avoids blocking UI work but requires follow-up implementation. Mark TODOs clearly.
- Centralizing session metadata in Redis offers low latency but adds an operational dependency; Supabase is slower but already available. The spec should allow either with a preference for Redis.
- Tabbed UI introduces extra navigation complexity; to keep scope minimal we restrict to a single "Recent" tab plus placeholder for future expansions.

## Risks
- Users may expect persistence immediately; communicate that history may be empty until backend wired.
- Mismatch between session IDs and transcripts could break resume functionality; validation scenarios should cover this.

## Validation Strategy
- Unit tests for UI components to ensure hover + click behaviors trigger expected state changes.
- API contract tests validating stubbed endpoints return deterministic structures.
- Manual QA to verify navigation between live chat and history tab works without page reload errors.


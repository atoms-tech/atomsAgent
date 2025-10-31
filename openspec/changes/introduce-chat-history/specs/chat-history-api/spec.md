# Spec: Chat History API Stubs

## Summary
Create REST endpoints and server-side scaffolding that supply chat session history to the UI with deterministic placeholder data until persistence is implemented.

## ADDED Requirements

### Requirement: List chat sessions endpoint
- agentapi MUST expose `GET /sessions` returning an array of session metadata objects.
- Each session object MUST include `id`, `title`, `agent_type`, `started_at`, `updated_at`, and `message_preview` fields.
- Until persistence exists, the handler MUST return either stub data (feature-flagged) or an empty list with `"status":"not-implemented"` metadata.
- The endpoint MUST respond within 200ms under stub mode.

#### Scenario: Stubbed list response
1. GIVEN persistence is not yet wired
2. WHEN `GET /sessions` is called
3. THEN the response is HTTP 200 with an empty `sessions` array and `"not-implemented"` status flag
4. AND the response body matches the OpenAPI schema.

### Requirement: Fetch session transcript endpoint
- agentapi MUST expose `GET /sessions/{session_id}` returning the message transcript for that session.
- The response MUST include the same metadata as the list endpoint plus a `messages` array matching the existing conversation format.
- In stub mode, the handler MUST return HTTP 501 with explanatory error payload.

#### Scenario: Stubbed transcript response
1. GIVEN persistence is not yet wired
2. WHEN `GET /sessions/{session_id}` is invoked
3. THEN the response is HTTP 501 with a JSON body describing that transcripts are not yet implemented
4. AND the body includes a `retry_after` hint set to null.

### Requirement: Update OpenAPI and SDKs
- OpenAPI schema MUST document the new endpoints, response objects, and error payloads.
- Generated SDK client for atomsAgent (Python/TypeScript) MUST include functions for `list_sessions` and `get_session`.
- SDK functions MUST default to safe fallbacks if the server returns stubbed data.

#### Scenario: SDK handles stub gracefully
1. GIVEN the server returns HTTP 200 with empty sessions
2. WHEN the SDK consumer calls `list_sessions()`
3. THEN the SDK returns an empty list without throwing, and exposes the status flag so the UI can surface a message.

### Requirement: Document storage strategy
- Repository interfaces MUST be defined for a Redis-backed session store with optional Supabase fallback.
- Documentation MUST record the preference order and any operational prerequisites.
- The stub implementation MUST log TODO markers referencing the future storage layer.

#### Scenario: Repository interface defined
1. GIVEN the codebase builds after the change
2. WHEN developers inspect the session repository module
3. THEN they see interfaces for Redis and Supabase implementations with `NotImplementedError`/TODO placeholders
4. AND build/test commands succeed.


# Project Context
## Purpose and Vision
atomsAgent delivers an OpenAI-compatible service layer that routes chat completions to Claude Code and Gemini models on Vertex AI while preserving downstream client expectations around request shape, streaming semantics, and usage accounting.
The project exists to let product teams embed stateful coding agents without dealing with Claude Agent SDK details, sandbox orchestration, or vendor-specific prompt handling.
### Strategic Goals
- Present a stable OpenAI-compatible API so existing SDKs, CLIs, and tooling can swap endpoints without invasive rewrites.
- Allow teams to roll out multi-tenant prompt governance by layering platform, organization, user, and workflow instructions.
- Centralize MCP server registration and secure credential storage so agents discover enterprise tools at runtime.
- Provide observability and admin surfaces that satisfy SaaS operational requirements (audit logs, admin roster, usage metrics, health indicators).
- Maintain vendor portability by abstracting Claude- and Gemini-specific behavior into isolated services that can be swapped or expanded.
### Primary Use Cases
- Power a hosted chat UI that feels identical to OpenAI Chat Completions but actually invokes Claude Code on Vertex AI.
- Orchestrate workflow-specific prompts for automated coding agents used by internal developer productivity tooling.
- Manage MCP integrations for enterprise customers, allowing each organization to surface custom tools and secure them with appropriate auth types.
- Offer platform administrators endpoints to monitor adoption, manage admins, and review audit trails of agent activity.
- Serve as a building block for future automation products that require anthropic- or google-backed reasoning capabilities without exposing raw SDKs.
## Audience Personas
- **Platform Operators** ensure configuration, credentials, and observability metrics keep the service healthy across environments.
- **Product Engineers** integrate OpenAI-compatible endpoints into UIs or CLI clients without touching low-level Claude session logic.
- **Prompt Engineers** iterate on platform, org, and workflow prompts through Supabase-backed repositories without redeploying code.
- **Security and Compliance Teams** rely on audit endpoints, sandbox isolation, and scoped MCP credentials to meet organizational requirements.
- **SRE / DevOps** teams deploy the ASGI app, tune caching, and manage secrets for Vertex AI and Supabase connectivity.
## High-Level Architecture
- FastAPI application instantiated in `src/atomsAgent/main.py` with ORJSON responses for performance and deterministic JSON encoding.
- Routers under `src/atomsAgent/api/routes/` expose OpenAI chat completions, MCP registry CRUD, and platform admin endpoints.
- Dependency wiring in `src/atomsAgent/dependencies.py` memoizes sandbox managers, Supabase clients, Claude session managers, prompt orchestrators, Vertex model service, and domain services.
- Service layer modules under `src/atomsAgent/services/` encapsulate vendor integrations, prompt logic, sandbox orchestration, and domain mapping.
- Data access layer in `src/atomsAgent/db/` implements a thin Supabase REST client and repository abstractions for prompts, MCP configs, admins, and audit logs.
- Schemas under `src/atomsAgent/schemas/` define OpenAI request/response models, MCP payloads, platform DTOs, and workflow metadata for type safety.
- Settings modules under `src/atomsAgent/settings/` supply strongly-typed configuration classes split between public config and secrets with environment prefixes.
- Tests under `tests/` validate async router behavior using fake service implementations to avoid external dependencies during CI runs.
## Request Lifecycle: Chat Completion (Non-Streaming)
1. Client issues POST `/v1/chat/completions` with messages, desired model, optional metadata, and tuning parameters such as temperature and top_p.
2. FastAPI router `create_chat_completion` (see `src/atomsAgent/api/routes/openai.py:37`) resolves dependencies for `ClaudeAgentClient` and `PromptOrchestrator` via `Depends`.
3. Metadata is parsed for session identifier, workflow, organization/user scope, prompt variables, custom tool allowlist, setting sources, and MCP server definitions.
4. `PromptOrchestrator.compose_prompt` merges platform prompt (from config), scoped Supabase prompts, and workflow-specific templates; request-level system prompt overrides orchestrated output when provided.
5. Messages are converted to dictionaries using `Pydantic.model_dump()`; system messages are excluded from `_compile_prompt` because system instructions are already applied via `system_prompt`.
6. Validation ensures at least one non-system message exists; otherwise an HTTP 400 is raised to match OpenAI error semantics.
7. `ClaudeAgentClient.complete` acquires or creates a session through `ClaudeSessionManager`, securing a sandbox workspace and constructing `ClaudeAgentOptions` with prompt, model, allowed tools, and metadata.
8. Claude Agent SDK `query` call issues the prompt; Tenacity retries up to three attempts with fixed backoff on generic exceptions to mitigate transient failures.
9. Streaming responses from `receive_response` accumulate assistant text blocks and capture usage metrics from `ResultMessage` objects.
10. Completion text, usage counts, and raw message payloads are returned to the router, which formats them into an OpenAI-compatible `ChatCompletionResponse`, including `system_fingerprint` seeded with session id.
11. Final JSON response includes a unique completion id, finish reason, assistant message content, and usage totals (prompt, completion, total tokens).
## Request Lifecycle: Chat Completion (Streaming)
1. When `stream=True`, the router constructs a server-sent event (SSE) stream using `StreamingResponse` with media type `text/event-stream`.
2. A stream identifier `chatcmpl-<uuid>` and creation timestamp are generated and reused across emitted chunks to align with OpenAI conventions.
3. The same session resolution and prompt orchestration occur before streaming begins, ensuring consistent system prompt and MCP configuration usage.
4. `ClaudeAgentClient.stream_complete` yields `CompletionChunk` instances containing incremental deltas, done flags, and optional usage snapshots.
5. `_serialize_chunk` converts each `CompletionChunk` into SSE payloads; the first delta includes `role="assistant"` while subsequent deltas omit the role to mirror OpenAI streaming behavior.
6. When `chunk.done` is True, a final payload with `finish_reason="stop"` is sent along with usage metrics if available, followed by the `[DONE]` sentinel to instruct clients to close the stream.
7. If `ClaudeAgentClient` raises a `ValueError`, the router emits an error payload with `type="invalid_request_error"` and still terminates the stream with `[DONE]` to avoid client hangs.
8. Clients must aggregate `delta.content` fragments to reconstruct the full assistant reply; this matches SSE parsing in the official OpenAI SDKs.
## Vertex Model Listing Workflow
1. GET `/v1/models` resolves the `VertexModelService` dependency via `get_vertex_model_service` in `dependencies.py`.
2. `VertexModelService.list_models` first checks the in-memory aiocache entry keyed by project/location; cache eviction occurs when TTL expires (`model_cache_ttl_seconds`, default 600).
3. When cache is cold, `_fetch_models` obtains an access token using service account credentials provided via either JSON blob or filesystem path in secret settings.
4. HTTPX `AsyncClient` requests the Vertex AI Model Garden endpoint using the `publishers/-/models` listing filtered by published state; Tenacity retries with exponential backoff on HTTP errors.
5. Response items are normalized into `ModelInfo` objects capturing model id, display name (as description fallback), provider, and capabilities list; created timestamp uses current epoch seconds.
6. Failure to obtain a token or non-200 responses trigger `_default_models`, ensuring OpenAI-compatible clients still receive a deterministic list of supported models.
7. Successful fetch results are cached alongside a retrieval timestamp to honor TTL semantics and avoid redundant API calls.
8. JSON response uses OpenAI list schema semantics (`object: "list"`, `data: [...]`).
## MCP Registry Lifecycle
- `list_mcp_servers`: requires `organization_id` query parameter and optional `user_id` plus `include_platform`. Service filters Supabase records by scope, ensuring platform-level MCPs are included only when requested.
- `create_mcp_server`: accepts `MCPCreateRequest` with scope validation. User scope requires both organization and user ids; organization scope forbids missing org id. Payload optionally includes bearer token or OAuth provider metadata.
- `update_mcp_server`: builds partial update payload while preserving existing fields when request omits values; updating bearer tokens is allowed by specifying new value or null.
- `delete_mcp_server`: issues a Supabase delete operation keyed by UUID.
- Service layer (`MCPRegistryService`) converts `MCPConfigRecord` dataclasses into API schemas with computed scope objects and metadata wrappers (`MCPMetadata`).
- Repository `MCPRepository.list_configs` uses Supabase REST filters: `enabled=eq.true` and an `or` filter to include organization-specific plus platform entries. User-level records are only returned when `user_id` filter matches.
- Metadata fields `args` and `env` are mapped to `MCPMetadata` to instruct agents about CLI args and environment variables when launching MCP HTTP servers.
- OAuth/DCR flows rely on stored provider metadata; OAuth-specific payloads are only preserved when `auth_type` equals `oauth`.
## Platform Administration Endpoints
- `GET /api/v1/platform/stats`: surfaces totals for users, active sessions, organizations, requests, tokens, MCP servers, and system health metadata. Repository currently aggregates counts via Supabase `select` with `count=true` options.
- `GET /api/v1/platform/admins`: returns admin roster ordered by creation timestamp descending. Response includes admin id, email, name, created_at, and created_by fields.
- `POST /api/v1/platform/admins`: inserts a new admin after validating email and WorkOS id presence; created_by is derived from request workos_id to capture provenance.
- `DELETE /api/v1/platform/admins/{email}`: removes admin record by email and returns confirmation payload with `status="success"`.
- `GET /api/v1/platform/audit`: supports pagination via limit/offset. Repository normalizes JSON string payloads into dicts to satisfy `AuditEntry.metadata` contract.
- Platform stats currently stub out `total_requests`, `requests_today`, and token counters (set to zero) pending integration with usage telemetry systems.
- System health block includes `status`, `circuit_breaker_status`, and `active_agents` array so dashboards can highlight agent availability across Claude and Gemini backends.
## Prompt Management Flow
- Platform prompts are configured via settings (`platform_system_prompt`).
- Organization/user prompts are stored in Supabase `system_prompts` table with scope fields and optional Jinja templates for dynamic rendering.
- Workflow prompts map workflow identifiers to string templates using `workflow_prompt_map` in config settings; this allows feature teams to ship workflow-specific instructions without migrating database tables.
- `PromptRepository.list_prompts` fetches all enabled prompts ordered by priority descending, promoting deterministic merging when multiple prompts apply to an organization/user pair.
- `PromptOrchestrator.compose_prompt` merges prompts in the order: platform → Supabase scoped prompts → workflow prompts. Empty strings are filtered out, extra whitespace trimmed, and final output uses double newline separation.
- Templates use `jinja2.StrictUndefined` so missing variables fail loudly; final fallback adds prompt content as-is when template rendering raises an exception to maintain continuity.
- Variables provided through metadata (via `metadata.variables`) become the context for template rendering; they may include workflow-specific context, environment toggles, or tenant metadata.
- Composed system prompt is applied to Claude session creation unless caller supplies explicit `system_prompt` (allowing advanced clients to override orchestrated prompts).
## Claude Session Management
- `ClaudeSessionManager` keeps an async dictionary of session id to `ClaudeSession` dataclasses, each containing sandbox context, config, client instance, lock, last_used timestamp, and connection flag.
- Sessions share sandbox directories across turns to persist code artifacts, logs, or repository clones generated by agent tools.
- Sessions are lazily created on first request; subsequent completions reuse the session, which keeps `continue_conversation=True` to provide conversational context to Claude agent runtime.
- Sandbox manager ensures workspace directories exist with 0700 permissions, preventing other users on the system from reading session artifacts.
- `ClaudeAgentOptions` is built with allowed tools (explicit from request metadata or defaults), setting_sources (enabling dynamic retrieval of prompts/tool configs), and `mcp_servers` map controlling available connectors.
- Extra arguments such as temperature, topP, and maxTokens are passed as strings per Claude SDK expectations; `_filter_none` removes unset keys.
- `session.client.query` is wrapped in Tenacity retry loop (three attempts, one second intervals) to protect against transient network issues or Vertex API hiccups.
- Streaming path listens to `session.client.receive_messages` producing incremental assistant responses and capturing final usage via `ResultMessage`. Non-stream path listens to `receive_response` for aggregated content.
- `default_session_id` helper generates unique session identifiers when metadata does not provide one, ensuring sandbox directories do not collide.
- `ClaudeSessionManager.release_session` disconnects client instances and optionally deletes sandbox directories, supporting explicit cleanup flows.
## Sandbox Operations
- Root sandbox path defaults to `/tmp/atomsAgent/sandboxes`; override via `ATOMS_SANDBOX_ROOT_DIR` to store sandboxes on persistent volumes in production.
- `SandboxManager.acquire` creates directories idempotently, sets secure permissions, and records optional creator metadata for debugging.
- Reset operation deletes and recreates workspace directories, useful when sandbox state becomes corrupted or too large.
- Release operation optionally removes workspace to reclaim disk; deleting is usually coupled with session termination or TTL-based sweeps.
- Locks keyed by sandbox id prevent concurrent modifications when multiple coroutines interact with the same session.
- Sandboxes host agent-executed commands (Read, Write, Edit, Bash, Skill) provided by Claude; ensure quotas and cleanup policies maintain disk hygiene.
- Consider background job to prune old sandboxes based on `last_used` timestamps tracked in `ClaudeSession` instances.
- When running in container orchestration environments, mount sandbox root to writable ephemeral storage per pod to avoid cross-tenant leakage.
## Supabase Integration
- `SupabaseClient` in `src/atomsAgent/db/supabase.py` uses REST interface with service role key for elevated access; requests include `Prefer=return=representation` to receive mutated rows in responses.
- Select queries optionally request exact counts by adding `count=exact`; response count is parsed from `content-range` header via `_extract_count`.
- Insert/update/delete operations marshal payloads as JSON and raise `SupabaseError` when status >= 400, including textual responses for debugging.
- RPC method supports calling stored procedures by posting JSON parameters to `/rest/v1/rpc/<function_name>`.
- Repositories convert response JSON into dataclasses for downstream mapping, providing typed access to columns (e.g., `AdminRecord`, `AuditLogRecord`).
- Service role keys should never be exposed to clients; environment configuration must secure `ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY`.
- Consider caching prompt listings if Supabase becomes a latency bottleneck; currently prompt orchestration hits Supabase for each completion when repository is available.
- Database schema is expected to be generated into `src/atomsAgent/db/models.py` via `sb-pydantic`; placeholder file warns against manual edits.
## Schema Models Overview
- `ChatCompletionRequest` enforces temperature between 0-2, max_tokens between 1-4000, and top_p between 0-1. Default max_tokens of 4000 aligns with typical Claude API constraints but can be tuned.
- `ChatMessage.content` accepts either string or list of `MessageContentText`; validator normalizes plain strings and dictionary payloads into typed structure for downstream use.
- `UsageInfo` surfaces prompt/completion totals; routers set `total_tokens` by reading `UsageStats.total_tokens` property from service result.
- `ModelInfo` includes `object="model"`, provider label, optional context length, and capability list; when Vertex returns additional metadata, extend this schema accordingly.
- `MCPScope` distinguishes platform, organization, and user scopes, with UUID typing enforcing valid identifiers; service ensures combination integrity.
- `MCPMetadata` holds CLI args and environment variables to feed into MCP clients; defaults to empty list/dict for easier merges.
- `MCPConfiguration` and `MCPCreateRequest` share fields; create request allows inline bearer token for initial bootstrap while configuration returns metadata and computed scope.
- `PlatformStats` nests `SystemHealth` object to allow future expansion (e.g., per-agent metrics, error rates).
- `AuditLogResponse` enumerates entries with pagination metadata; ensures consistent shape for UI consumption.
- `WorkflowMetadata` (in `schemas/workflows.py`) documents optional metadata attached to OpenAI-style requests to drive downstream automation triggers.
## Configuration Surfaces
- `ConfigSettings` (prefix `ATOMS_`): app version, docs toggle, CORS origins, Vertex project/location, model cache TTL, platform prompt ids, prompt map, sandbox root directory, default model, allowed tools, setting sources.
- `SecretSettings` (prefix `ATOMS_SECRET_`): Vertex credentials path/json, Claude API key (reserved for future use), Supabase URL, Supabase service key, Redis URL.
- `SettingsProxy` merges both classes, allowing attribute-style access (`settings.default_model`, `settings.supabase_url`).
- Settings allow unknown extra keys (`extra="allow"` for config, `extra="ignore"` for secrets) to future-proof environment variable additions without immediate code changes.
- Missing Supabase configuration triggers RuntimeError in `get_supabase_client`, causing dependency injection to fail fast when secrets are absent.
- Vertex credentials can be provided either via JSON string (`ATOMS_SECRET_VERTEX_CREDENTIALS_JSON`) or file path (`ATOMS_SECRET_VERTEX_CREDENTIALS_PATH`); service chooses JSON first, then path, otherwise returns default models.
- `ATOMS_DEFAULT_ALLOWED_TOOLS` default list includes `Read`, `Write`, `Edit`, `Bash`, `Skill`, aligning with Claude Code agent toolset; override via env var to restrict capabilities per environment.
- CORS middleware is only added when `cors_allow_origins` is non-empty to avoid overhead on internal deployments.
- Docs UI toggled via `ATOMS_ENABLE_DOCS`; turning off hides Swagger/Redoc routes for hardened production environments.
- Workflow prompts expect JSON mapping string (e.g., via env var) parsed by Pydantic into `dict[str, str]`.
## Dependency Injection Practices
- `@lru_cache` wrappers ensure expensive objects (Supabase client, Vertex model service, session manager) are singletons per process, matching FastAPI dependency caching semantics.
- Services consuming Supabase repositories instantiate repository classes lazily to avoid hitting database when endpoint is unused.
- Vertex model service obtains its own aiocache instance, preventing cross-service cache pollution; consider injecting shared cache if future needs require global eviction control.
- Claude client and prompt orchestrator rely on underlying dependencies (`get_session_manager`, `get_supabase_client`); stacking lru caches ensures deterministic object graphs.
- Dependency functions raise runtime errors when configuration is missing; FastAPI translates these into 500 responses, surfacing misconfiguration quickly in staging environments.
- For testing, dependencies can be overridden using FastAPI's dependency override mechanism, though current tests import router functions directly for simplicity.
## Testing Strategy Details
- Tests use `pytest.mark.asyncio` to run coroutine-based endpoints directly, bypassing ASGI test clients to keep feedback loop fast.
- `FakeClaudeClient` and `FakePromptOrchestrator` provide deterministic responses; streaming test aggregates SSE chunks and asserts presence of expected text fragment.
- `FakeMCPService` records arguments to ensure routers pass metadata correctly and returns seeded data to mimic repository/service responses.
- `FakePlatformService` inherits from `PlatformService` to reuse type signatures but overrides async methods to return sample data without hitting Supabase.
- Tests cover success paths; expanding coverage to include error cases (invalid payloads, missing messages, invalid scopes) will strengthen contract enforcement.
- Consider adding FastAPI test client integration tests to validate dependency wiring and middleware (CORS) behavior under realistic request flows.
- Future test suite should mock Tenacity retry loops and Supabase errors to verify error handling semantics align with expectations.
- `tests/conftest.py` modifies `sys.path` to ensure `src` package is importable without requiring editable installs during testing.
## Local Development Workflow
- Install dependencies with `uv pip install -e ".[dev]"` to get runtime and dev tools in an editable environment.
- Run Supabase model generator (`python -m atomsAgent.codegen.supabase`) after schema migrations to refresh generated Pydantic models; script verifies CLI availability and prints troubleshooting tips when local database is unreachable.
- Start development server via `uvicorn atomsAgent.main:app --reload`; optionally set `ATOMS_ENABLE_DOCS=true` to expose Swagger UI at `/docs`.
- Use HTTP client (Insomnia, Postman, curl) or local chat UI to exercise endpoints; SSE streaming can be tested with `curl -N` to observe incremental chunks.
- Populate Supabase tables with sample prompts and MCP configs to test orchestration flows locally; ensure service role key is scoped to dev project.
- Configure environment variables using `.env` files only for local dev; production should inject secrets via secret manager or container environment settings.
- Run `ruff check .` and `pytest tests` before committing changes to catch lint and regression issues early.
- Use `pre-commit` (optional dependency) to automate formatting and linting hooks; configure `.pre-commit-config.yaml` if repo expands to include formatting tools.
- Document new capabilities via OpenSpec proposals prior to implementation to maintain change traceability and review workflows.
- When adding new endpoints, update `README.md` and project context to keep documentation aligned with behavior.
## Deployment Considerations
- Deploy as ASGI app behind an ingress capable of streaming SSE responses (e.g., Cloud Run, GKE with nginx, AWS ALB with HTTP/2).
- Ensure worker processes run with sufficient file descriptor limits to handle concurrent streaming connections; uvicorn default workers may need scaling adjustments.
- Vertex and Supabase connectivity requires outbound internet access; configure VPC connectors or NAT gateways in locked-down environments.
- Supply service account JSON via secret manager and mount as file or environment variable; avoid baking credentials into container images.
- Configure sandbox root on ephemeral storage for performance; mount persistent storage only when session artifacts must survive pod restarts.
- Enable health checks hitting `/v1/models` or `/api/v1/platform/stats` to verify both FastAPI process and Supabase connectivity at startup.
- Integrate logging pipeline (Stackdriver, CloudWatch, etc.) capturing request metadata, Claude errors, and Tenacity retry logs for production observability.
- Consider enabling metrics (Prometheus, OpenTelemetry) to report latency, token usage, and error rates for SLO monitoring.
- Align deployment region with Vertex AI location to minimize latency; default configuration uses `us-central1`.
- For high availability, run multiple replicas behind load balancer; manage session affinity only if sandbox reuse across requests is required, otherwise rely on metadata-specified session ids.
## Observability and Monitoring
- Capture structured logs for each completion, including session id, organization, workflow, model, tokens, and latency to inform analytics and cost tracking.
- Instrument `ClaudeAgentClient` to log retry attempts and final success or failure outcomes; differentiate between client errors (400s) and upstream failures.
- Monitor Supabase latency and error rates; add metrics around prompt retrieval times to identify database bottlenecks.
- Track sandbox disk usage to anticipate cleanup needs; emit gauge metrics for active sandboxes and storage consumption.
- Expose Vertex model cache hit ratios to evaluate TTL tuning and potential over-fetching.
- Implement tracing (OpenTelemetry) across router to service to external calls to visualize end-to-end latency and identify hot spots.
- Add audit logging for MCP CRUD operations to satisfy compliance requirements; log actor metadata and request context.
- Build dashboards combining platform stats endpoint data with actual usage metrics to provide comprehensive operational overview.
- Configure alerting for spikes in HTTP 5xx rates, Supabase errors, or Claude SDK failures exceeding thresholds.
- Periodically review SSE completion durations to detect regressions in streaming performance.
## Error Handling Philosophy
- Input validation errors raise FastAPI `HTTPException` with 400 status codes and descriptive messages (e.g., missing messages array, invalid MCP scope).
- Claude SDK errors that manifest as `ValueError` are surfaced to clients as invalid request errors in streaming SSE payloads.
- Repository methods let Supabase errors propagate as custom `SupabaseError`, which should be caught at service or router layer to translate into meaningful HTTP responses (future enhancement).
- Runtime misconfiguration (missing Supabase credentials) raises `RuntimeError` during dependency resolution, causing startup to fail rather than returning partial functionality.
- Tenacity retries currently catch generic `Exception`; refine to specific network or client errors to avoid retrying non-transient issues.
- SSE streams always terminate with `[DONE]`, even when errors occur, to prevent clients from waiting indefinitely.
- Logging (future addition) should capture stack traces for unhandled exceptions with correlation ids derived from session or request ids.
- Consider implementing custom exception handlers in FastAPI to normalize error payloads and hide sensitive upstream details.
- Expand validation for MCP endpoints to ensure endpoints use HTTPS, names are unique per scope, and metadata conforms to expected schema.
- Guard against extremely large prompts or metadata payloads to protect from memory exhaustion; enforce size limits at router level.
## Performance Considerations
- Use ORJSON response class by default to minimize serialization overhead for high-throughput endpoints.
- Cache Vertex model listings to reduce repeated network calls and lower latency for `/v1/models`.
- Optionally layer Redis caching by configuring `ATOMS_SECRET_REDIS_URL` and extending `VertexModelService` or prompt retrieval to use distributed cache in multi-instance deployments.
- Limit sandbox cleanup operations to off-peak periods to avoid I/O contention while streaming responses.
- Evaluate asynchronous Supabase queries for prompts; consider local caching or in-process memoization keyed by organization or user to avoid repeated database hits within same session.
- Prefetch workflow prompts into memory if the map remains small and static to eliminate per-request dictionary lookups (micro-optimization).
- Monitor Claude session reuse effectiveness; encourage clients to supply consistent `session_id` in metadata to avoid repeated sandbox creations.
- Investigate streaming chunk sizes to ensure clients receive timely updates; tune Claude agent options if necessary.
- Provide rate limiting or concurrency controls if backend quotas or sandbox resources become constrained.
- Explore asynchronous logging and metrics emission to avoid blocking request handling.
## Security Posture
- Sandbox directories enforce 0700 permissions; ensure deployment user is dedicated to atomsAgent to prevent cross-application access.
- Secrets retrieved via environment variables must never be printed or logged; sanitize logs to avoid leaking credentials.
- Validate MCP endpoints to require HTTPS URLs and limit exposure to internal-only services when configured appropriately.
- Consider encrypting stored bearer tokens in Supabase or using references to secret manager entries instead of raw secrets.
- OAuth provider metadata should be validated to ensure redirect URIs and client credentials conform to organization policies.
- Platform admin endpoints should eventually require authentication and authorization middleware; current implementation assumes upstream gateway handles auth.
- SSE responses should avoid echoing sensitive metadata; only include necessary information in streaming payloads.
- Implement audit logging for changes to prompts, MCP configs, and admin roster to meet compliance standards.
- Rotate Supabase service role keys regularly and scope them to minimal privileges necessary for operations.
- Ensure Vertex service accounts adhere to principle of least privilege, granting only necessary AI Platform permissions.
## Multi-Tenancy Model
- Each request may carry `organization_id` and `user_id` metadata to scope prompts, MCP servers, and session behavior.
- Platform-level prompts and MCPs act as global defaults; organization- and user-level entries override or augment these defaults.
- Repository queries enforce organization equality before including records; user-scoped MCPs require matching organization and user ids to prevent leakage.
- Session metadata can include `setting_sources` to fetch custom configurations from Claude agent runtime, enabling per-tenant customization beyond prompts.
- Future enhancements may include per-tenant rate limiting, usage quotas, and resource isolation to maintain fairness across organizations.
- When storing audit logs, ensure resource identifiers include organization context to facilitate per-tenant reporting.
- Expose tenant id in logs and metrics to support cost allocation and debugging for specific customers.
- Avoid caching prompt results globally without tenant-specific keys to prevent cross-tenant prompt leakage.
- MCP bearer tokens should be scoped to organization or user; never reuse tokens across tenants.
- Provide tooling for tenant admins to rotate prompts and MCP credentials through secure UI or API endpoints.
## MCP Integration Deep Dive
- Supported auth types: `none`, `bearer`, `oauth`; repository stores bearer tokens directly and OAuth provider metadata for DCR flows.
- `MCPCreateRequest` accepts initial bearer token; storing hashed tokens or referencing external secret stores is a future enhancement for security hardening.
- MCP metadata `args` may include command-line flags, while `env` stores key-value pairs injected into MCP process environment.
- Organization scope requires `organization_id`; user scope requires both `organization_id` and `user_id`. Platform scope sets both to null.
- Repository ensures `include_platform` flag controls whether platform-scoped MCPs appear alongside org or user scoped ones.
- Update payloads do not automatically null fields unless explicitly provided; this prevents accidental removal of configured endpoints or credentials.
- Delete operation silently succeeds; calling code should verify service state or rely on subsequent `list` to confirm removal.
- Consider implementing soft-delete or enabled flag toggles for easier rollback and auditing.
- MCP responses include `created_at` timestamp (if available) and computed scope object, ensuring clients can display configuration metadata consistently.
- Future roadmap includes attaching tags, categories, or agent compatibility metadata to MCP configs for richer discovery experiences.
## Prompt Orchestration Examples
- Platform prompt might instruct agents on general behavior (e.g., abide by company coding standards); stored in configuration for global control.
- Organization prompt could specify repository naming conventions, allowed dependencies, or security requirements unique to the tenant.
- User prompt might include personal preferences or active tasks retrieved from CMS or issue trackers.
- Workflow prompt may adapt instructions for special flows such as `analyze_requirements` or `migration_assistant`; triggered by metadata `workflow` field.
- Variables dictionary may supply repository name, branch, ticket id, or environment details; templates reference variables using `{{ variable_name }}` syntax.
- Prompt ordering ensures higher-level context appears first; duplicates or blank prompts are filtered to prevent noise.
- When prompt repository is unavailable (e.g., Supabase misconfigured), orchestrator gracefully returns platform plus workflow prompts only.
- Template rendering uses dedicated Jinja environment per invocation to avoid cross-request state leakage.
- Exceptions during template rendering are swallowed after capturing raw content, preventing runtime failures caused by misconfigured templates.
- Consider logging template errors with prompt id and scope to aid debugging without exposing sensitive content to end users.
## Vertex Model Service Configuration
- `model_cache_ttl_seconds` controls how long cached model lists remain valid; choose TTL balancing freshness with API quota usage.
- Credentials priority: JSON env var greater than file path; ensure only one is set to avoid ambiguity.
- Access token retrieval uses offline service account credentials; ensure service account has `aiplatform.models.read` permission.
- Default model list includes four entries covering Claude Sonnet, Claude Haiku, Gemini Pro, and Gemini Flash with capabilities metadata. Update as new models become available.
- Implementation stores `_CachedModels` dataclass in memory; customizing cache backend enables cross-process sharing when multiple workers run.
- HTTP requests include `timeout=30.0` seconds; adjust if Vertex API exhibits slower responses or needs shorter timeouts to free resources.
- Tenacity uses exponential backoff with minimum one second and maximum four seconds; total retries limited to three attempts to avoid prolonged wait times.
- For environments without internet access, rely on default model list to keep `/v1/models` operational; surface warnings via logs to highlight degraded state.
- Extend service to include pricing, context length, or tool compatibility metadata when Vertex API exposes richer descriptors.
- Optionally enrich response with `owned_by` mapping to product names or teams for better UI display.
## Claude Agent Client Configuration
- Default model set by `ATOMS_DEFAULT_MODEL`; ensures clients that omit `model` still target a supported backend.
- Allowed tools default to canonical Claude Code toolchain; customizing the list can enforce restricted execution (for example, disable `Bash` in sandboxed environments).
- Setting sources allow referencing prompt repositories or knowledge bases accessible by Claude agent runtime; defaults to config-level value.
- `mcp_servers` metadata enables per-request tool injection; ensures agent can leverage organization-specific connectors without global configuration.
- `user_identifier` parameter is passed to Claude query; map to user ids or email addresses for personalized experiences or auditing.
- Temperature and top_p controls propagate to Claude; ensure upstream service supports requested ranges to avoid `ValueError` exceptions.
- Max tokens defaults to request value; ensure values respect model-specific limitations to avoid truncation.
- Raw Claude messages stored in `CompletionResult.raw_messages` can be persisted or analyzed for debugging; consider redacting sensitive content before logging.
- When streaming is not requested, aggregated text is joined from individual assistant messages to ensure consistent formatting.
- Investigate session eviction policies (LRU, TTL) to prevent unbounded growth; current implementation retains sessions indefinitely until release is called.
## File and Module Inventory
- `src/atomsAgent/main.py`: constructs FastAPI app, attaches CORS middleware conditionally, registers routers, and instantiates settings-driven configuration.
- `src/atomsAgent/api/routes/openai.py`: defines chat completion and model listing endpoints, SSE serialization helper, and metadata handling logic.
- `src/atomsAgent/api/routes/mcp.py`: exposes MCP CRUD endpoints; simple validation and dependency injection for `MCPRegistryService`.
- `src/atomsAgent/api/routes/platform.py`: surfaces platform stats, admin management, and audit logs with HTTP validation and service delegation.
- `src/atomsAgent/services/claude_client.py`: houses session manager, sandbox coordination, Tenacity retries, streaming logic, and usage aggregation.
- `src/atomsAgent/services/prompts.py`: orchestrates platform plus scoped prompts, handles Jinja rendering, and gracefully degrades on errors.
- `src/atomsAgent/services/platform.py`: maps repository records to API responses, constructs health metadata, and provides admin CRUD helpers.
- `src/atomsAgent/services/mcp_registry.py`: validates scope, builds Supabase payloads, and converts dataclasses to Pydantic models.
- `src/atomsAgent/services/sandbox.py`: manages sandbox lifecycle with asynchronous locks and secure file system primitives.
- `src/atomsAgent/services/vertex_models.py`: integrates with Vertex AI, implements caching, and supplies fallback model roster.
- `src/atomsAgent/db/supabase.py`: REST client with JSON serialization, error handling, and count parsing logic.
- `src/atomsAgent/db/repositories.py`: data access for prompts, MCP configs, platform stats, admin roster, and audit logs with dataclass representations.
- `src/atomsAgent/settings/config.py`: Pydantic BaseSettings definitions for public configuration and environment variable parsing.
- `src/atomsAgent/settings/secrets.py`: secret-specific BaseSettings for sensitive data.
- `src/atomsAgent/dependencies.py`: lru_cached factory functions constructing service instances and enforcing configuration presence.
- `src/atomsAgent/config.py`: settings proxy aggregator to unify config plus secrets spaces.
- `src/atomsAgent/codegen/supabase.py`: script to generate Pydantic models from Supabase schema using `supabase_pydantic` CLI.
- `src/atomsAgent/utils/caching.py`: simple synchronous TTL cache decorator (currently unused but available for future optimization).
- `tests/test_api_endpoints.py`: async tests covering chat completions, streaming SSE, MCP flows, and platform endpoints using fakes.
- `README.md`: developer onboarding instructions, feature overview, and configuration guidance.
## Documentation and Spec Workflow
- OpenSpec instructions located in `openspec/AGENTS.md` outline three-stage workflow (proposal, implementation, archive).
- `openspec/project.md` (this file) documents high-level context for AI assistants and contributors.
- Currently no specs under `openspec/specs/`; future work should codify API capabilities to support spec-driven development.
- `openspec/changes/` directory is empty; proposals should be created there with unique change ids describing planned modifications.
- Managed instruction stubs (`AGENTS.md`, `CLAUDE.md`, `CLINE.md`) ensure agents know to read OpenSpec guidance prior to planning or coding.
- When drafting proposals, include `proposal.md`, `tasks.md`, and delta specs; validate with `openspec validate <change-id> --strict`.
- Avoid implementing new features without approved proposals to maintain alignment with spec-driven process.
- Document environment-specific behaviors (Vertex quotas, MCP auth flows) in future specs to provide canonical references for QA.
- Use `openspec archive` to move deployed changes into archive directory with timestamped folder names.
- Update this project context as architecture evolves to keep agents informed of new conventions or dependencies.
## Operational Runbooks (Draft)
- **Vertex token failure**: verify `ATOMS_SECRET_VERTEX_CREDENTIALS_JSON` or `_PATH` environment variables are set and service account has required roles; fallback models keep API responsive but log warnings.
- **Supabase 401 or 403**: ensure service role key is valid and Supabase project allows REST access from deployment IP. Rotate credentials if compromised.
- **Sandbox disk exhaustion**: run cleanup script to remove directories older than threshold; tune TTL and ensure deletion occurs after session release.
- **Claude session import error**: confirm `claude-agent-sdk` dependency is installed; install via `pip install claude-agent-sdk` when running tests that require actual SDK usage.
- **Streaming stalls**: inspect logs for Tenacity retries, confirm SSE proxy supports flush, and verify Claude agent returns `ResultMessage` to close stream.
- **Prompt rendering errors**: log template id and variables, update Supabase prompt entry to include required variables or fallback content.
- **MCP registration failure**: check request payload for missing scope fields, ensure endpoint is HTTPS, and verify Supabase connectivity.
- **Admin endpoint auth**: ensure upstream gateway or middleware enforces auth; current code assumes trusted environment.
- **Cache staleness**: adjust `model_cache_ttl_seconds` or implement manual invalidation when deploying new models.
- **Redis integration**: when enabling distributed cache, configure TLS and authentication; update code to instantiate Redis-backed aiocache.
## Future Enhancements Backlog (Non-exhaustive)
- Add OpenSpec specs for each API capability to formalize behavior and enable automated contract testing.
- Implement dependency overrides or fixtures to support integration testing with FastAPI test client and dependency injection overrides.
- Integrate structured logging with correlation ids, tenant metadata, and Claude session details for better observability.
- Add rate limiting middleware to protect backend resources from abuse or misconfigured clients.
- Implement session eviction policies (TTL or max sessions) to reclaim resources and avoid unbounded sandbox growth.
- Support asynchronous background cleanup of old audit logs or prompts to satisfy data retention policies.
- Expand Vertex model service to include metadata such as pricing tiers, capability flags (vision, reasoning), and context limits per model.
- Provide admin UI or CLI commands for managing prompts and MCP configurations without direct Supabase access.
- Introduce health endpoint that verifies Claude SDK connectivity by issuing lightweight requests against sandbox or vendor API.
- Support multi-region deployments with region-specific configuration maps for Vertex and Supabase endpoints.
## Known Limitations
- No built-in authentication or authorization layer; relies on upstream infrastructure to gate platform endpoints.
- Supabase interactions occur on every prompt orchestration; caching layer is not yet implemented, leading to potential latency under heavy load.
- Session eviction not implemented; long-lived sessions may accumulate sandboxes unless external cleanup runs.
- Tests do not cover error paths or Supabase failure scenarios, leaving certain edge cases unverified.
- Vertex model listing fallback does not indicate degraded mode to clients; consider adding metadata or headers to inform consumers.
- Prompt orchestrator does not currently respect prompt priority beyond ordering; future enhancements may include conditional logic or categories.
- MCP repository does not enforce unique names per scope; duplicates may confuse clients until uniqueness constraints are added in database.
- Platform stats return zero for request or token metrics; requires integration with telemetry pipeline to provide meaningful data.
- Supabase client currently lacks connection pooling; each request instantiates new HTTPX client (though reused within context).
- CLI agent command directories (`.augment`, `.cursor`, etc.) are placeholders; no automation is provided out-of-the-box.
## Glossary of Terms
- **Claude Agent SDK**: Anthropic-provided SDK enabling terminal-based coding agents with session persistence and tool integrations.
- **Claude Code**: Anthropic's coding-focused agent variant used to power developer workflows.
- **Vertex AI**: Google Cloud platform hosting third-party models like Claude and Gemini; provides API endpoints and model garden listings.
- **OpenAI Compatibility**: Adhering to request and response shapes defined by OpenAI APIs to make client integration seamless.
- **MCP (Model Context Protocol)**: Protocol for connecting agents to external tools or services with standardized handshake and metadata.
- **Supabase**: Backend-as-a-service offering Postgres, REST, and auth; used here as datastore for prompts, MCP configs, and audit logs.
- **Sandbox**: Filesystem workspace assigned to a session where agent commands execute (read and write code, run scripts).
- **Session**: Logical conversation context maintained by Claude Agent SDK, persisting state across turns and mapping to sandbox directories.
- **Prompt Orchestration**: Process of merging prompts from multiple scopes into a single system prompt presented to agent runtime.
- **SSE (Server-Sent Events)**: Streaming mechanism used to deliver incremental completion chunks to clients, aligning with OpenAI streaming behavior.
## Tables and Data Expectations (Supabase)
- `system_prompts`: columns include id, content, priority, scope (`global`, `organization`, `user`), organization_id, user_id, template, enabled. Stored prompts may include Jinja templates referencing metadata variables.
- `mcp_configurations`: columns include id, organization_id, user_id, name, type (http), endpoint, auth_type, bearer_token, oauth_provider, enabled, args array, env jsonb, created_at timestamps.
- `platform_admins`: columns include id, workos_id, email, name, created_at, created_by; used for admin roster management.
- `audit_logs`: columns include id, timestamp, action, resource_type, resource_id, details (json or text), success. Details may store user or org context and additional metadata.
- `organizations`, `users`, `user_sessions`: referenced for counts in platform stats; expected to exist in Supabase schema though direct models are not generated yet.
- Ensure Supabase row-level security (RLS) is configured appropriately; service role bypasses RLS, so underlying stored procedures should enforce tenant isolation.
- Use Supabase migrations or SQL scripts to maintain schema; run generator after schema changes to keep code in sync.
- Monitor Supabase rate limits; caching or batching may be required under heavy usage.
- For large prompts or metadata, evaluate Postgres storage limits and ensure indexes support query patterns used by repository filters.
- Plan for data retention and archival strategies, especially for audit logs that may grow quickly in enterprise deployments.
## SSE Payload Example (Annotated)
- Example chunk: `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"claude-4.5-sonnet","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}],"system_fingerprint":"session_abc"}`
- First chunk includes role to inform clients about assistant speaker; subsequent chunks omit role for brevity.
- Done chunk: `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"claude-4.5-sonnet","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"system_fingerprint":"session_abc","usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`
- Stream terminates with `data: [DONE]` on its own line; clients should treat this as signal to close connection.
- Error chunk example: `data: {"error":{"message":"conversation must contain at least one user message","type":"invalid_request_error"}}` emitted before `[DONE]`.
- SSE newline separation ensures compatibility with EventSource API; every payload ends with double newline to flush chunk.
- Clients should handle optional `usage` field, which may be absent if upstream service fails to provide metrics.
- `system_fingerprint` allows clients to map completions back to session ids for debugging or caching purposes.
- Timestamps use integer UNIX seconds to match OpenAI API behavior.
- Example shows quoting and escaping; maintain JSON serialization via ORJSON for performance and deterministic ordering.
## Session Metadata Fields
- `session_id`: string identifying conversation; used to fetch or create Claude session and map to sandbox directory.
- `workflow`: optional identifier mapping to workflow-specific prompt templates (for example, `migration_assistant`, `bug_fix`).
- `organization_id`: UUID string representing tenant; ensures prompts and MCP configs are loaded correctly.
- `user_id`: UUID string representing end user; enables user-scoped prompts and MCP access restrictions.
- `variables`: dictionary providing template variables (repository name, ticket id, branch, etc.).
- `allowed_tools`: list of tool names to override default allowed toolset; controls capabilities available to agent (Read, Write, etc.).
- `setting_sources`: list of identifiers enabling agent runtime to load additional settings or prompt segments from external sources.
- `mcp_servers`: dictionary mapping server aliases to configuration (endpoint, auth); passed directly to Claude agent runtime.
- `user`: optional user identifier string passed to Claude query for analytics or personalization; not part of metadata but request root field.
- Additional metadata keys may be introduced in future; router currently pulls known keys but leaves remainder unused to preserve forward compatibility.
## Environment Variable Reference (Non-exhaustive)
- `ATOMS_APP_VERSION`: string displayed in FastAPI docs; default `0.1.0`.
- `ATOMS_ENABLE_DOCS`: boolean toggle for Swagger and Redoc; disable in production when using external API documentation portals.
- `ATOMS_CORS_ALLOW_ORIGINS`: JSON array or comma-separated list recognized by Pydantic for CORS origins; empty list disables middleware.
- `ATOMS_VERTEX_AI_PROJECT_ID` (alias `ATOMS_VERTEX_PROJECT_ID`): GCP project containing Vertex AI resources.
- `ATOMS_VERTEX_AI_LOCATION` (alias `ATOMS_VERTEX_LOCATION`): Vertex AI region such as `us-central1`.
- `ATOMS_MODEL_CACHE_TTL_SECONDS`: integer TTL for Vertex model cache; default 600 seconds.
- `ATOMS_PLATFORM_SYSTEM_PROMPT`: string storing platform-level instructions applied to all sessions unless overridden.
- `ATOMS_WORKFLOW_PROMPT_MAP`: JSON dict mapping workflow ids to prompt strings.
- `ATOMS_SANDBOX_ROOT_DIR`: filesystem path for sandbox root; ensure process user has write permissions.
- `ATOMS_DEFAULT_MODEL`: fallback model when request omits model id (default `claude-4.5-sonnet`).
- `ATOMS_DEFAULT_ALLOWED_TOOLS`: JSON array of allowed tool names; default `["Read","Write","Edit","Bash","Skill"]`.
- `ATOMS_DEFAULT_SETTING_SOURCES`: JSON array of setting source identifiers; default empty list.
- `ATOMS_SECRET_VERTEX_CREDENTIALS_JSON`: service account JSON blob used to mint Vertex access tokens.
- `ATOMS_SECRET_VERTEX_CREDENTIALS_PATH`: filesystem path to service account JSON file.
- `ATOMS_SECRET_CLAUDE_API_KEY`: reserved for direct Claude API usage if needed in future integrations.
- `ATOMS_SECRET_SUPABASE_URL`: Supabase project base URL (e.g., `https://<ref>.supabase.co`).
- `ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY`: Supabase service role key granting elevated REST access.
- `ATOMS_SECRET_REDIS_URL`: optional Redis connection string for distributed caching.
- Additional project-specific variables may be introduced; BaseSettings allow unknown keys to pass through without raising errors.
## CLI and Automation Artifacts
- `.augment/commands`: reserved for Auggie automation scripts; currently empty but can store high-value command templates or macros.
- `.claude/commands`: placeholder for Claude CLI integrations; populate with scenario-specific command presets as workflow matures.
- `.cursor/commands`: used by Cursor editor agents to record frequently used commands or context retrieval steps.
- `.factory/commands`: supports Factory Droid automation; maintain consistent formatting if multiple agents share command files.
- These directories illustrate multi-agent tooling integration; keep them synchronized with documentation to avoid stale instructions.
- When adding commands, prefer idempotent scripts and note prerequisite environment variables.
- Consider versioning automation scripts to track compatibility with evolving project architecture.
- Provide README entries or inline comments in command files to explain intent and expected outcomes.
- Avoid committing secrets or environment-specific paths in automation scripts; rely on variables or placeholder tokens.
- Encourage contributors to document new automation flows in OpenSpec proposals or project context to inform other agents.
## Developer Productivity Tips
- Use FastAPI dependency overrides during local testing to substitute fake services without modifying production code.
- Leverage `uvicorn --reload` with `--reload-exclude` to ignore sandbox directories and reduce reload noise.
- Configure Ruff to fix whitespace and import order automatically via `ruff check . --fix` when safe.
- Use `pytest -k chat` to run subset of tests focusing on chat completion flows during iterative development.
- Inspect SSE streams with browser DevTools or CLI tools like `xh --stream` to validate proper chunk formatting.
- Employ logging breakpoints or `print` statements within fake services during tests to observe metadata propagation.
- When debugging Supabase interactions, enable HTTPX logging (`HTTPX_LOG_LEVEL=DEBUG`) to see raw requests and responses.
- Maintain sample environment variable files (e.g., `.env.sample`) documenting required configuration for new developers.
- Use `watchfiles` or editor-integrated watchers to trigger tests or linting on file changes for rapid feedback.
- Keep README onboarding section updated with latest commands and prerequisites to minimize ramp-up time for newcomers.
## Observability Instrumentation Ideas
- Integrate OpenTelemetry FastAPI instrumentation to emit traces for each request, capturing dependency calls to Claude and Supabase.
- Record metrics for request latency segmented by endpoint, streaming vs non-streaming, and model type.
- Capture token usage per organization to support cost allocation reporting.
- Emit counters for MCP CRUD operations grouped by scope (platform, organization, user).
- Track prompt orchestration latency and failure counts to identify bottlenecks in Supabase or template rendering.
- Monitor sandbox creation and deletion events to ensure cleanup keeps pace with session lifecycle.
- Add structured log fields for workflow, session id, organization id, and user id to simplify filtering in log aggregators.
- Collect Vertex API error codes and response times to inform caching strategy adjustments.
- Log Tenacity retry statistics to identify services requiring resilience improvements.
- Instrument `PromptOrchestrator` to count template rendering failures versus successes for continuous prompt quality monitoring.
## Troubleshooting Checklist
- Verify environment variables when encountering startup failures; missing Supabase or Vertex settings commonly trigger runtime errors.
- For 500 responses on chat completions, inspect logs for Claude SDK exceptions or Supabase connectivity issues.
- If streaming hangs, confirm ingress or reverse proxy supports chunked transfer encoding and SSE; disable buffering where necessary.
- When prompts appear missing, query Supabase `system_prompts` table to confirm entries exist and `enabled` flag is true.
- If MCP list excludes expected entries, ensure `include_platform` flag matches desired scope and organization id is correct.
- For audit log data discrepancies, validate Supabase query filters and confirm timestamps are stored in consistent timezone (UTC recommended).
- If Vertex models endpoint returns fallback list unexpectedly, check credential validity and Vertex API access; log warnings should explain fallback reason.
- In local dev, ensure `.venv` or global environment includes `claude-agent-sdk`; missing dependency triggers RuntimeError in session manager.
- To debug sandbox issues, inspect filesystem permissions and ensure application user owns sandbox root directory.
- For linting failures, run `ruff check . --fix` and address flagged style issues before committing.
## Compliance and Governance Notes
- Audit log endpoint provides basis for compliance reporting; ensure retention policies meet regulatory requirements (e.g., SOC 2, ISO 27001).
- MCP configuration changes should be logged with actor metadata to support change management audits.
- Prompt updates stored in Supabase should include metadata (creator, timestamp) when schema evolves to support change tracking.
- Sandbox cleanup policies must balance privacy (removing customer data) with need to retain troubleshooting artifacts; align with data retention standards.
- Secrets management strategy should comply with organizational policies (e.g., use GCP Secret Manager, AWS Secrets Manager, or Vault).
- Streaming responses must avoid leaking sensitive internal reasoning unless explicitly approved; consider redaction or gating prompts for confidential information.
- Ensure Vertex AI usage complies with regional data residency requirements; configure project and location accordingly.
- Platform admin endpoints should integrate with enterprise identity providers (WorkOS, Okta) for authentication and audit trail alignment.
- Document third-party subprocesses (Claude agent tool executions, MCP connectors) for risk assessments and vendor security reviews.
- Perform periodic penetration testing focused on sandbox escape attempts, MCP endpoint injection, and prompt tampering.
## Third-Party Dependencies Overview
- `fastapi>=0.110.0`: web framework providing routing, dependency injection, and request validation.
- `uvicorn[standard]>=0.28.0`: ASGI server used for local development and potential production deployments.
- `httpx>=0.27.0`: async HTTP client for Vertex and Supabase interactions.
- `aiocache>=0.12.0`: caching library with in-memory and redis backends supporting async interfaces.
- `tenacity>=8.2.0`: retry utility used for resilient external API calls (Claude, Vertex).
- `pydantic>=2.6.0` and `pydantic-settings>=2.2.0`: data validation and environment configuration management.
- `jinja2>=3.1.2`: templating engine for prompt rendering.
- `supabase>=2.4.0`: official Supabase Python client (used primarily for type hints; actual implementation uses custom REST client).
- `claude-agent-sdk>=0.1.5`: Anthropic agent SDK providing session and tool orchestration.
- `orjson>=3.9.10`: high-performance JSON serializer used as default FastAPI response class.
## Development Dependencies
- `ruff>=0.3.0`: linter enforcing style, import order, and bugbear-like rules.
- `pytest>=8.0.0`: testing framework with modern fixture and assert introspection support.
- `pytest-asyncio>=0.23.0`: enables async test coroutines.
- `pytest-cov>=4.1.0`: coverage reporting plugin.
- `pytest-mock>=3.12.0`: provides pytest fixtures for mocking.
- `typer>=0.9.0`: CLI framework; may be leveraged for future management commands.
- `pre-commit>=3.6.0`: hook runner to enforce linting and formatting before commits.
- `supabase-pydantic>=0.6.0`: utility to generate Pydantic models from Supabase schemas.
- Consider pinning development dependencies when reproducibility becomes critical.
- Use `uv pip install -e ".[dev]"` to install both runtime and development dependencies in editable mode.
## Code Organization Principles
- Keep routers thin and delegate business logic to services for easier testing and future reuse across multiple transport layers.
- Group related services by domain (Claude, Vertex, MCP, Platform) to simplify discovery and ownership.
- Place configuration classes in dedicated `settings` package; avoid circular imports by referencing `atomsAgent.config.settings` in application modules.
- Use dataclasses for repository return types to provide lightweight structure without heavy ORM abstraction.
- Maintain strong typing (Pydantic, type hints) to facilitate static analysis and reduce runtime errors.
- Encapsulate third-party client logic (Supabase, Vertex) in dedicated modules to centralize error handling and retry policies.
- Avoid mixing sync and async code; ensure external libraries used in async routes provide async variants.
- Document key functions and classes with concise docstrings focusing on behavior and invariants rather than restating implementation details.
- Keep tests close to modules they cover; add additional test modules when introducing new routers or services.
- Ensure new modules follow existing naming conventions (`snake_case` for files, `CamelCase` for classes).
## Contribution Workflow
- Create or update OpenSpec proposal when introducing new capability, architectural change, or breaking adjustment to existing APIs.
- Fork repository or create feature branch named after change id (e.g., `add-mcp-oauth-support`).
- Implement code changes alongside test updates, ensuring `pytest` and `ruff` pass locally.
- Update documentation (`README.md`, `openspec/project.md`, spec files) to reflect new behavior.
- Submit PR referencing OpenSpec change id; request review from maintainers familiar with relevant subsystem (Claude, MCP, Platform, etc.).
- Address review feedback promptly, maintaining clear commit history (squash when merging if preferred).
- After deployment, archive proposal (`openspec archive <change-id> --yes`) to keep spec backlog manageable.
- Maintain release notes summarizing significant changes, new endpoints, or configuration updates for downstream consumers.
- Coordinate with ops team when new environment variables or secrets must be provisioned.
- Ensure security review occurs for changes impacting authentication, sandbox execution, or external integrations.
## Testing Matrix Suggestions
- **Unit**: service methods (prompt orchestration, MCP mapping, platform stats) with mocked repositories; ensures logic correctness without HTTP layer.
- **API (fast)**: router functions invoked directly with fake dependencies as in current tests; validates serialization and error handling.
- **API (integration)**: FastAPI test client hitting endpoints with dependency overrides that use in-memory or mock services.
- **End-to-end**: optional tests using real Supabase sandbox or Vertex sandbox in staging environment to validate external integrations.
- **Performance**: load testing streaming endpoint to measure latency and throughput under realistic concurrency.
- **Security**: fuzzing prompts and MCP payloads to detect injection vulnerabilities or improper validation.
- **Regression**: contract tests comparing responses against saved fixtures to detect breaking changes in API shape.
- **Chaos**: simulate Supabase downtime or Vertex token failures to validate fallback paths and error messaging.
- **Compatibility**: verify clients built on OpenAI SDKs (Python, JS) operate without modification against atomsAgent endpoints.
- **Tooling**: ensure CLI automation scripts continue to operate after command or workflow changes.
## Release Management Notes
- Tag releases in version control with semantic versioning aligned to API changes (major for breaking, minor for new features, patch for bug fixes).
- Update `pyproject.toml` version and `ConfigSettings.app_version` when cutting releases to keep docs accurate.
- Generate changelog summarizing new features, fixes, and migration steps; publish alongside release tag.
- Coordinate with operations to roll out new environment variables or infrastructure changes prior to deploying code updates.
- Monitor telemetry post-deployment to ensure new releases stabilize within expected latency and error budgets.
- Roll back quickly if completion success rates drop or streaming errors spike; maintain rollback playbook.
- Communicate release details to stakeholders (product, support, customers) when user-facing behavior changes.
- Use feature flags or staged rollout when introducing high-risk features to limit impact of regressions.
- Validate backward compatibility with existing clients before releasing major changes.
- Document manual steps required for deployment (e.g., secret rotation, schema migrations) in release notes or runbooks.
## Business Continuity Considerations
- Maintain multi-region backups of Supabase data to restore prompts, MCP configs, and audit logs after outage.
- Store Claude sandbox artifacts only as long as necessary; ensure critical state is persisted in durable storage rather than relying on sandbox files.
- Keep Vertex service accounts separate per environment to avoid cascading failures from credential revocation.
- Design failover plan where completions fall back to alternative provider or degrade gracefully when Claude or Vertex is unavailable.
- Document manual procedures for rehydrating environment variables and secrets from secret manager in disaster recovery scenarios.
- Ensure monitoring alerts page on-call engineers with context (affected endpoints, suspected cause) to accelerate incident response.
- Periodically test disaster recovery playbooks (restore from backup, rotate keys, redeploy infrastructure) to validate readiness.
- Evaluate cost of running warm standby environment versus acceptable recovery time objective (RTO).
- Include vendor contact details (Anthropic, Google) in runbooks for escalation during prolonged outages.
- Capture lessons learned after incidents and update documentation and code to prevent recurrence.
## Accessibility and UX Considerations
- Ensure API error messages are human-readable and actionable for UI consumers; include guidance on remediation when possible.
- Provide consistent response schema across success and error states to simplify UI parsing.
- Support SSE streaming in browsers by adhering to EventSource requirements (UTF-8 encoding, double newlines, `[DONE]` sentinel).
- Document request and response examples in API docs (`README.md` or future OpenSpec specs) for developers integrating UI components.
- Consider rate limiting or pagination strategies to keep admin endpoints performant for large datasets.
- When embedding chat UI, expose configuration toggles (workflow selection, session reset) to empower users without requiring backend changes.
- Provide health status information (from `/api/v1/platform/stats`) to front-end applications for display in dashboards.
- Record usage analytics (without sensitive data) to understand user behavior and inform UX improvements.
- Offer sandbox cleanup controls or diagnostics in admin interface to help users resolve stuck sessions.
- Align API naming and resource structure with familiar patterns (OpenAI-like) to reduce learning curve.
## Configuration Examples
- Example minimal `.env` for development:
    ```env
    ATOMS_ENABLE_DOCS=true
    ATOMS_VERTEX_AI_PROJECT_ID=dev-project
    ATOMS_VERTEX_AI_LOCATION=us-central1
    ATOMS_DEFAULT_MODEL=claude-4.5-haiku
    ATOMS_PLATFORM_SYSTEM_PROMPT=You are a helpful coding assistant.
    ATOMS_SECRET_SUPABASE_URL=https://dev.supabase.co
    ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY=dev-service-role
    ```
- Example workflow prompt map environment variable (JSON string):
    ```json
    {"bug_fix": "Focus on reproducible steps and commit messages.", "code_review": "Provide constructive feedback and highlight potential regressions."}
    ```
- Example MCP creation payload:
    ```json
    {
      "name": "internal-docs",
      "endpoint": "https://mcp.internal.example.com",
      "auth_type": "bearer",
      "bearer_token": "<token>",
      "scope": {"type": "organization", "organization_id": "00000000-0000-0000-0000-000000000001"}
    }
    ```
- Example chat completion request with metadata:
    ```json
    {
      "model": "claude-4.5-sonnet",
      "messages": [{"role": "user", "content": "Generate unit tests for the new API."}],
      "temperature": 0.2,
      "stream": true,
      "metadata": {
        "session_id": "team-alpha-session",
        "workflow": "test_writer",
        "organization_id": "00000000-0000-0000-0000-000000000222",
        "variables": {"repository": "atoms-agent", "language": "python"}
      }
    }
    ```
- Example audit log response snippet:
    ```json
    {
      "entries": [
        {
          "id": "audit-123",
          "timestamp": "2024-01-01T00:00:00Z",
          "action": "mcp.create",
          "resource": "mcp",
          "resource_id": "00000000-0000-0000-0000-000000000002",
          "metadata": {"user_id": "user-1", "organization_id": "org-1"}
        }
      ],
      "count": 1,
      "limit": 50,
      "offset": 0
    }
    ```
## Example Workflow Metadata Usage
- `trigger`: indicates the action that initiated the workflow (chat, analyze_requirements, custom). Use to branch logic in downstream services.
- `context`: optional dictionary storing additional metadata such as issue numbers, repository paths, or environment labels.
- Agents can inspect metadata to modify behavior (e.g., `analyze_requirements` may gather more context before coding).
- Workflows can map to environment-specific prompts or tool configurations to align with team workflows.
- Ensure metadata remains lightweight to avoid passing excessive data to Claude or saturating logs.
- Document supported workflow names so client teams know which prompts exist.
- Provide default behavior when metadata is absent to avoid runtime errors.
- Avoid storing sensitive data in metadata since it may be logged or stored in session history.
- When designing new workflows, update configuration map and project documentation simultaneously.
- Consider adding validation to ensure unknown workflow ids produce graceful errors or fall back to default behavior.
## Data Privacy Considerations
- Treat sandbox contents as transient; avoid persisting user code beyond necessary timeframe to respect privacy agreements.
- Ensure prompts and metadata stored in Supabase do not contain personally identifiable information unless strictly required.
- Sanitize audit logs before exposing them to customers to prevent leakage of sensitive details.
- When logging request metadata, redact secrets (bearer tokens, API keys) and hashed identifiers where possible.
- Implement data retention policies to purge stale records (prompts, audit logs) based on customer contracts.
- Provide mechanisms for customers to request deletion of stored prompts or MCP entries associated with their organization.
- Ensure backups containing customer data are encrypted at rest and in transit.
- Document data flow diagrams for privacy assessments and regulatory compliance.
- Limit access to Supabase dashboards to authorized personnel; enforce MFA and principle of least privilege.
- Monitor data export activities and audit trail access to detect potential misuse.
## Infrastructure Recommendations
- Use container images built via CI pipeline with pinned dependency versions and vulnerability scanning.
- Deploy on managed platforms (Cloud Run, GKE, ECS) with autoscaling based on CPU or concurrency metrics to handle bursts.
- Configure load balancer with idle timeout greater than maximum expected streaming duration to avoid premature disconnects.
- Use managed secret stores for environment variables containing credentials; inject at runtime rather than baking into image.
- Employ centralized logging (e.g., Stackdriver) with structured JSON output for easier analysis.
- Implement blue-green or canary deployments to minimize downtime during upgrades.
- Monitor resource usage (CPU, memory) and adjust instance sizing to balance cost and performance.
- Keep infrastructure-as-code definitions (Terraform, Cloud Deployment Manager) in version control for reproducible environments.
- Ensure TLS termination occurs at load balancer or ingress; optionally enable mTLS between services if required.
- Align backup and restore processes with business continuity plans, scripting them to reduce manual effort.
## Open Questions for Future Iteration
- Should prompt orchestrator cache Supabase results per session to reduce repeated database queries?
- How should session eviction be handled (LRU, TTL, manual API) to manage sandbox lifecycle at scale?
- Do we need multi-provider support beyond Claude and Gemini, and how would service abstractions evolve to accommodate additional models?
- What authentication strategy best secures platform endpoints (API keys, OAuth, JWT) while remaining flexible for different customers?
- Should MCP configuration include rate limits or quotas per connector to prevent abuse?
- Will audit logs require tamper-evident storage or external archival to satisfy compliance requirements?
- How should we expose agent usage metrics to customers (dashboard, exports, API)?
- Could prompt templates benefit from versioning to support rollbacks and A/B experimentation?
- What governance is needed around sandbox tool execution to prevent long-running or resource-intensive tasks?
- Should we provide configuration toggles to disable certain tools (e.g., `Bash`) on a per-tenant basis without code changes?
## Community and Support Guidance
- Encourage contributors to open issues or discussions before major refactors to align on approach and scope.
- Maintain issue templates for bug reports and feature requests capturing environment, reproduction steps, and expected behavior.
- Provide support rotation or contact information for operational incidents to ensure timely responses.
- Document etiquette for code reviews (constructive feedback, turnaround expectations, approval criteria).
- Offer onboarding guides for new maintainers covering architecture overview, build steps, and deployment processes.
- Keep changelog or release notes accessible to external stakeholders to track progress and upcoming features.
- Promote knowledge sharing through internal demos or brown bag sessions showcasing new capabilities.
- Encourage adding unit tests when fixing bugs to prevent regressions and improve code confidence.
- Foster inclusive and respectful communication within issue trackers, reviews, and chat channels.
- Monitor community contributions for quality and security; conduct thorough reviews before merging external pull requests.
## Sample Logging Fields (Proposed)
- `request_id`: unique identifier per HTTP request (from tracing middleware or generated at entry).
- `session_id`: identifier derived from metadata or generated default to link logs to conversations.
- `organization_id`: tenant context for multi-tenant analytics and debugging.
- `user_id`: optional user identifier when provided in metadata.
- `workflow`: workflow name aiding behavioral analysis.
- `model`: target model (Claude Sonnet, Gemini Pro, etc.).
- `streaming`: boolean indicating streaming vs non-streaming completion.
- `duration_ms`: measured latency for completion call.
- `token_usage`: dictionary capturing prompt, completion, total tokens.
- `error_code`: string capturing upstream error or HTTP status for failures.
## Suggested Metrics (Prometheus-style)
- `atoms_completion_requests_total{model,streaming}`: counter incremented per completion request.
- `atoms_completion_duration_seconds_bucket{model}`: histogram capturing latency per model.
- `atoms_completion_tokens_total{model,type}`: counter for prompt vs completion tokens consumed.
- `atoms_supabase_request_duration_seconds_bucket{operation}`: histogram for Supabase latency.
- `atoms_vertex_cache_hit_ratio`: gauge representing cache effectiveness.
- `atoms_sandbox_active_sessions`: gauge for number of active Claude sessions.
- `atoms_mcp_configs_total{scope}`: gauge for MCP entries by scope.
- `atoms_prompt_render_failures_total`: counter for Jinja rendering exceptions.
- `atoms_retry_attempts_total{service}`: counter for Tenacity retries per external service.
- `atoms_streaming_termination_total{outcome}`: counter distinguishing normal vs error-terminated streams.
## Checklist Before Production Deployment
- [ ] All required secrets (`ATOMS_SECRET_*`) populated in secret manager or environment.
- [ ] Vertex service account permissions validated for target project and location.
- [ ] Supabase schema migrations applied and generator run if models required.
- [ ] Logging and monitoring configured with dashboards and alerts.
- [ ] Load and stress tests executed to verify performance under expected concurrency.
- [ ] Security review completed for new endpoints or integrations.
- [ ] Runbooks updated with latest troubleshooting steps and contact information.
- [ ] Sandbox cleanup automation scheduled or manual process documented.
- [ ] Release notes drafted and communicated to stakeholders.
- [ ] Rollback strategy rehearsed or documented (e.g., revert to previous container image).
## Appendix A: Directory Map
```
.
├── openspec/
│   ├── AGENTS.md
│   ├── changes/
│   ├── project.md
│   └── specs/
├── src/
│   └── atomsAgent/
│       ├── api/
│       │   └── routes/
│       ├── services/
│       ├── db/
│       ├── settings/
│       ├── schemas/
│       ├── codegen/
│       ├── utils/
│       ├── config.py
│       ├── dependencies.py
│       └── main.py
├── tests/
├── README.md
└── pyproject.toml
```
- Virtual environment artifacts located under `src/atomsAgent/.venv/`; treat as tooling, not source.
- Hidden `.augment`, `.claude`, `.cursor`, `.factory` directories store agent-specific command presets.
- `pyproject.toml` defines project metadata, dependencies, and tooling configuration (ruff, pytest, build targets).
- `README.md` provides onboarding steps, configuration overview, and primary feature summary.
- `tests/` currently contains async API tests; expand as new features are added.
## Appendix B: Tenacity Retry Configuration
- `AsyncRetrying` configured with `stop_after_attempt(3)` meaning up to three attempts (initial try plus two retries).
- `wait_fixed(1)` in `ClaudeAgentClient` ensures one-second delay between retries to avoid hammering the backend.
- `retry_if_exception_type(Exception)` currently broad; consider narrowing to network-related exceptions.
- Retries raise original exception (`reraise=True`) after exhausting attempts to allow caller to handle failure.
- Vertex service uses `wait_exponential` with multiplier one second, min one, max four, balancing quick retries with backoff.
- Logging retry attempts can aid debugging when upstream services exhibit intermittent issues.
- Evaluate idempotency of operations before adjusting retry counts to avoid duplicate side effects.
- Ensure metrics capture retry counts to highlight degraded upstream dependencies.
- Consider customizing retry behavior per service (different stop conditions or jitter).
- Document retry strategy so operators understand expected behavior during partial outages.
## Appendix C: Sandbox Cleanup Script Sketch
```python
import asyncio
from pathlib import Path
from atomsAgent.services.sandbox import SandboxManager

async def purge_old_sandboxes(root: str, ttl_seconds: int) -> None:
    manager = SandboxManager(root_path=root)
    now = asyncio.get_running_loop().time()
    for path in Path(root).iterdir():
        if not path.is_dir():
            continue
        age = now - path.stat().st_mtime
        if age > ttl_seconds:
            await manager.release(path.name, delete=True)

asyncio.run(purge_old_sandboxes('/tmp/atomsAgent/sandboxes', ttl_seconds=86400))
```
- Run periodically via cron or async background task to control disk usage.
- Adapt script to read TTL and root path from environment variables for flexibility.
- Ensure concurrency controls prevent interference with active sessions (use manager locks or offline execution).
- Log deletions for auditing and debugging.
- Consider notifying stakeholders or shipping metrics when cleanup removes large number of sandboxes.
## Appendix D: Sample Test Case Ideas
- Validate `create_chat_completion` returns 400 when messages array is empty.
- Ensure streaming response yields `[DONE]` even when `ValueError` occurs midstream.
- Confirm prompt orchestrator merges prompts in correct order and trims whitespace.
- Test MCP create endpoint rejects user scope without user id.
- Verify platform stats endpoint returns system health payload with defaults when repository supplies minimal data.
- Simulate Supabase error in repository and assert service surfaces exception appropriately (future handler work).
- Check Vertex model service uses cached result when within TTL and fetches new data after expiration.
- Validate sandbox reset recreates directory and maintains permissions.
- Confirm Claude session manager reuse occurs when same session id requested twice.
- Add regression test for workflow prompt map to ensure metadata workflow names match configured keys.
## Appendix E: Suggested Logging Messages
- `"starting_completion"`: include session_id, model, streaming flag, workflow.
- `"completion_success"`: include duration, token usage, retries, sandbox_id.
- `"completion_failed"`: include exception type, message, retries, metadata snapshot (sanitized).
- `"vertex_models_cache_hit"`: include project, location, age of cached data.
- `"vertex_models_fetch_error"`: include status code, response text (sanitized), fallback used.
- `"mcp_create"`: include scope type, endpoint host, success flag.
- `"sandbox_acquired"`: include sandbox_id, path, created_by if provided.
- `"sandbox_reset"`: include sandbox_id, requester, reason.
- `"prompt_render_error"`: include prompt id, scope, missing variables.
- `"supabase_request"`: include table, operation, status, duration.
## Appendix F: Environment-Specific Notes
- **Development**: enable docs, use local Supabase emulator or staging project, relaxed logging for debugging.
- **Staging**: mirror production configuration, run integration tests, enable verbose logging temporarily for validation.
- **Production**: disable docs, enforce strict CORS, run behind WAF or API gateway, tighten logging to avoid sensitive data exposure.
- **Sandbox**: create isolated environment for experimentation with relaxed quotas but separated credentials.
- Maintain separate Vertex service accounts per environment to prevent accidental cross-environment access.
- Configure Supabase projects with environment-specific databases to prevent data leakage.
- Align feature flags or configuration toggles across environments to ensure consistent behavior during promotion.
- Use infrastructure-as-code to replicate environments reliably.
- Document differences (e.g., default models, allowed tools) between environments for transparency.
- Automate synchronization of workflow prompt maps and platform prompts across environments when necessary.
## Appendix G: Integration Touchpoints
- **Chat UI**: consumes `/v1/chat/completions` for interactive experiences; must handle streaming SSE and usage metadata.
- **CLI Tools**: rely on OpenAI-compatible endpoints to provide completion and streaming features in terminal-based workflows.
- **Internal Dashboards**: query `/api/v1/platform/stats` and `/api/v1/platform/audit` to surface operational data.
- **Automation Pipelines**: may register MCP servers programmatically using `/atoms/mcp` endpoints to expose toolchains for agents.
- **Support Systems**: access audit logs to trace user actions and resolve incidents.
- **Billing Systems**: ingest usage metrics (tokens, requests) to calculate charges per organization.
- **Prompt Management UI** (future): interfaces with Supabase or dedicated APIs to edit prompts and templates.
- **Knowledge Bases**: integrate via MCP connectors, enabling agents to query documentation or code repositories.
- **Authentication Providers**: integrate with platform admin endpoints via upstream gateway to ensure only authorized admins manage configuration.
- **Monitoring Stack**: collects metrics and logs for observability; ensure endpoints emit necessary data.
## Appendix H: Change Management Guidelines
- Announce upcoming breaking changes to stakeholders with migration instructions and timelines.
- Provide deprecation windows for API modifications, supporting both old and new behavior during transition.
- Tag spec versions in OpenSpec to track capability evolution.
- Maintain compatibility suites verifying older clients still function after backend changes.
- Document database migrations with rollback steps and impact analysis.
- Update project context promptly when altering architecture or introducing new dependencies.
- Encourage incremental rollouts with feature flags to limit blast radius of large changes.
- Solicit feedback from downstream teams before finalizing API changes to ensure requirements are met.
- Keep track of configuration changes (new env vars) in centralized documentation for environment managers.
- Archive obsolete documentation to reduce confusion and keep knowledge base current.
## Appendix I: Security Testing Ideas
- Pen-test sandbox escape by attempting to access files outside sandbox root; ensure permissions prevent traversal.
- Fuzz MCP endpoints with invalid URLs, auth types, and payloads to verify validation and error handling.
- Attempt to register MCP with duplicate name and check repository behavior; consider adding uniqueness constraints.
- Simulate credential leakage by providing invalid Supabase key and observe failure handling.
- Test SSE endpoint for injection vulnerability by sending malicious content and verifying encoding.
- Validate rate limiting on platform endpoints to prevent brute-force or enumeration attacks once auth is implemented.
- Inspect logs to ensure sensitive data (bearer tokens, secrets) is never written.
- Conduct dependency vulnerability scans and update packages regularly.
- Ensure third-party connectors (via MCP) do not introduce insecure protocols or endpoints.
- Verify TLS enforcement by attempting plain HTTP connections to production endpoints.
## Appendix J: Intellectual Property and Licensing
- Project is distributed under license referenced in `../LICENSE`; ensure compliance when integrating third-party code.
- Respect licenses of dependencies (MIT, Apache, etc.); track obligations such as attribution when shipping binaries.
- Document contributions from external collaborators to maintain attribution and handle CLA requirements if applicable.
- Ensure prompts and templates do not include proprietary content without appropriate permissions.
- Clarify ownership of generated code or outputs if service is packaged as SaaS offering.
- Maintain audit trail of third-party libraries and versions for legal review.
- Provide notice of license changes in release notes when dependencies change terms.
- Comply with export controls when deploying models across regions; consult legal for restrictions.
- Consider patent implications when building unique automation workflows; document innovations accordingly.
- Encourage contributors to report potential IP concerns via designated channels.
## Appendix K: Support Playbook
- Provide tiered support structure (L1 triage, L2 engineering, L3 vendor escalation).
- Maintain knowledge base articles for common issues (prompt errors, MCP auth failures, streaming timeouts).
- Track incidents in ticketing system with severity classification and resolution notes.
- Establish SLA targets for response and resolution times per issue severity.
- Conduct root cause analysis for major incidents and share summaries with stakeholders.
- Schedule regular review meetings to discuss open issues, backlog, and customer feedback.
- Train support staff on new features prior to release to ensure readiness.
- Provide runbooks with step-by-step remediation for frequent problems.
- Coordinate with product teams to prioritize bug fixes affecting customer experience.
- Implement feedback loop from support to engineering to inform roadmap adjustments.
## Appendix L: Training and Onboarding Plan
- Week 1: Read project context, walkthrough architecture, set up local environment, run tests.
- Week 2: Pair with mentor to implement small bug fix or documentation update; learn deployment workflow.
- Week 3: Shadow on-call rotation to understand operational tooling and incident response.
- Week 4: Own a minor feature or enhancement with guidance, following OpenSpec proposal process.
- Provide curated list of resources (FastAPI docs, Claude Agent SDK docs, Vertex AI references) for self-study.
- Encourage participation in code reviews to learn style and patterns.
- Maintain checklist ensuring access to necessary tools (Supabase dashboard, monitoring, secret manager).
- Schedule architecture deep-dive sessions covering prompt orchestration, MCP registry, and Claude sessions.
- Document responsibilities and escalation paths for new team members.
- Solicit feedback from new hires to continuously improve onboarding materials.
## Appendix M: Risk Register (Snapshot)
- **R1**: Claude Agent SDK breaking change impacting session API. *Mitigation*: pin dependency versions, monitor release notes, run regression suite before upgrades.
- **R2**: Supabase outage causing prompt retrieval failures. *Mitigation*: implement caching and graceful degradation, maintain contact with Supabase support.
- **R3**: Sandbox storage exhaustion. *Mitigation*: automate cleanup, monitor disk usage, enforce quotas.
- **R4**: Unauthorized access to MCP credentials. *Mitigation*: secure storage, audit logging, encryption at rest, least privilege.
- **R5**: Vertex quota exhaustion leading to throttled model calls. *Mitigation*: monitor usage, request quota increases, implement rate limiting.
- **R6**: Misconfigured prompts causing agent misbehavior. *Mitigation*: validation tooling, staging tests, change approval process.
- **R7**: Streaming compatibility regression with OpenAI clients. *Mitigation*: maintain compatibility tests, follow OpenAI spec updates.
- **R8**: Lack of authentication on platform endpoints. *Mitigation*: integrate auth gateway, implement API keys or JWT soon.
- **R9**: Data retention non-compliance. *Mitigation*: define retention policies, implement purge jobs, document compliance status.
- **R10**: Dependency vulnerability exposure. *Mitigation*: schedule dependency audits, apply security patches promptly.
## Appendix N: Cost Optimization Ideas
- Cache prompt and model data aggressively to reduce repeated Supabase or Vertex calls.
- Implement token usage tracking to identify heavy tenants and adjust billing or throttling.
- Use autoscaling policies to reduce idle capacity in low-traffic periods.
- Optimize sandbox storage lifecycle to minimize persistent volume costs.
- Consider spot instances for non-critical workloads (e.g., staging) when supported by infrastructure provider.
- Evaluate cost of Vertex models and choose appropriate default models based on usage patterns.
- Compress logs or use sampling to reduce logging costs while retaining necessary observability.
- Batch audit log writes (future improvement) to minimize database operations.
- Use CDN or caching layer for static documentation or UI assets embedded in binary.
- Encourage efficient prompt design to reduce token consumption while maintaining quality outputs.
## Appendix O: Integration Checklist for New Clients
- [ ] Verify client SDK compatibility with OpenAI Chat Completions API.
- [ ] Provide base URL and authentication mechanism (once implemented) to client team.
- [ ] Share required metadata fields (session_id, organization_id) and recommended usage patterns.
- [ ] Offer sample requests/responses for streaming and non-streaming flows.
- [ ] Ensure client handles SSE `[DONE]` sentinel and error payloads.
- [ ] Coordinate testing window in staging environment before production rollout.
- [ ] Confirm client understands prompt orchestration behavior and how to supply metadata variables.
- [ ] Document process for registering MCP servers and managing credentials.
- [ ] Establish support contacts and escalation paths for integration issues.
- [ ] Review rate limits, quotas, and pricing with client to set expectations.
## Appendix P: Change Log Template
```
## [Version] - YYYY-MM-DD
### Added
- 

### Changed
- 

### Fixed
- 

### Deprecated
- 

### Removed
- 

### Security
- 
```
- Update with each release to capture notable changes.
- Link to relevant OpenSpec change ids or pull requests for traceability.
- Highlight breaking changes prominently to inform consumers.
- Include migration steps when necessary (schema changes, env vars, etc.).
## Appendix Q: Communication Channels
- **Slack**: #atoms-agent-dev for engineering discussion, #atoms-agent-ops for incident coordination.
- **Email**: atoms-agent@company.example for announcements and support escalations.
- **Docs**: Internal wiki page aggregating architecture diagrams, runbooks, and onboarding guides.
- **Calendar**: Regular sync meetings (weekly standup, bi-weekly architecture review).
- **Issue Tracker**: GitHub or Jira project dedicated to atomsAgent tasks and bug reports.
- **Status Page**: Public or internal status page to broadcast incidents and maintenance windows.
- **Incident Response**: PagerDuty or equivalent for on-call notifications.
- **Roadmap**: Shared document or board outlining upcoming features and priorities.
- **Knowledge Base**: curated articles for support teams and customers.
- **Community Forum**: optional discussion board for external users if productized.
## Appendix R: Diagram Descriptions (Textual)
- **Sequence: Chat Completion**
    1. Client sends request to FastAPI endpoint.
    2. Router resolves dependencies (Claude client, prompt orchestrator).
    3. Prompt orchestrator fetches prompts from Supabase and configuration.
    4. Claude client acquires sandbox and session from session manager.
    5. Claude SDK query executes; results streamed back to router.
    6. Router serializes SSE or JSON response to client.
- **Sequence: MCP Create**
    1. Client sends POST to `/atoms/mcp` with scope and endpoint.
    2. Router validates payload and calls `MCPRegistryService`.
    3. Service builds payload and invokes `MCPRepository.create_config`.
    4. Repository uses Supabase client to insert record.
    5. Response record mapped to `MCPConfiguration` and returned to client.
- **Component Diagram**
    - FastAPI app orchestrates routers.
    - Dependencies module wires services.
    - Services interact with Claude SDK, Vertex API, Supabase REST, sandbox manager.
    - Data flows from Supabase to prompt orchestrator and platform services.
    - Clients (chat UI, admin tools) consume HTTP endpoints.
## Appendix S: Localization Considerations
- API responses currently in English; ensure error messages remain clear and avoid idioms if future localization is planned.
- Prompt content stored in Supabase can include localized text per tenant; document strategy for multi-language support.
- Audit logs should capture locale when relevant to assist with regional compliance.
- Configure logging and monitoring tools to handle UTF-8 characters from localized prompts or messages.
- Consider timezone handling in audit logs and stats; currently timestamps should be stored in UTC.
- Provide guidance on encoding for clients sending localized prompts to avoid character encoding issues.
- SSE and JSON responses use UTF-8 by default; ensure clients parse accordingly.
- When building UI, internationalize labels for stats and admin interfaces.
- Evaluate need for locale-specific workflows or allowed tools configurations.
- Maintain documentation translations if serving global audience.
## Appendix T: Innovation Opportunities
- Explore fine-tuning prompts using feedback loops from audit logs or user ratings.
- Integrate codebase analysis MCP connectors (e.g., Sourcegraph) to enrich agent capabilities.
- Provide self-service configuration UI empowering tenants to manage prompts and MCPs safely.
- Investigate automatic sandbox cleanup based on resource usage thresholds or inactivity heuristics.
- Build analytics dashboards showing usage trends by workflow, model, and token consumption.
- Experiment with multi-agent coordination by orchestrating multiple Claude sessions per workflow.
- Integrate with CI/CD pipelines to trigger automated code reviews or test generation using chat completions.
- Offer SLA-backed plans by introducing priority queues or dedicated resources per premium tenant.
- Extend SSE streaming to include tool invocation updates for richer UI experiences.
- Implement offline task queues for long-running analyses that return results asynchronously.
## Appendix U: Environment Variable Checklist Script (Pseudo)
```python
import os
REQUIRED = [
    "ATOMS_VERTEX_AI_PROJECT_ID",
    "ATOMS_VERTEX_AI_LOCATION",
    "ATOMS_SECRET_SUPABASE_URL",
    "ATOMS_SECRET_SUPABASE_SERVICE_ROLE_KEY",
]
missing = [key for key in REQUIRED if not os.getenv(key)]
if missing:
    raise RuntimeError(f"Missing required env vars: {', '.join(missing)}")
```
- Extend script with optional variables and defaults for improved diagnostics.
- Include script in deployment pipeline to fail fast when configuration incomplete.
- Print masked values or length checks when debugging to avoid exposing secrets.
- Provide human-readable guidance for resolving missing variables.
## Appendix V: Additional Notes
- Supplemental detail line 1: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 2: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 3: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 4: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 5: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 6: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 7: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 8: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 9: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 10: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 11: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 12: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 13: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 14: reserved for future elaboration on architecture, governance, or operational practices.
- Supplemental detail line 15: reserved for future elaboration on architecture, governance, or operational practices.

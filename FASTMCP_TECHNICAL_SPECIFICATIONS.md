# FastMCP 2.0 Technical Specifications

This document provides detailed technical specifications for integrating FastMCP 2.0 into AgentAPI, covering architecture, OAuth integration, transport mechanisms, production requirements, and Go integration patterns.

## 1. FastMCP Architecture

### 1.1 Client Initialization Patterns

#### Basic Client Creation
FastMCP clients use a Generic transport-based architecture where the client type is parameterized by the transport:

```python
from fastmcp import Client
from fastmcp.client.transports import (
    HTTPTransport,
    SSETransport,
    StdioTransport,
    StreamableHttpTransport
)

# Transport types determine client behavior
Client[SSETransport]          # For SSE connections
Client[StreamableHttpTransport]  # For HTTP streaming (default for HTTP URLs)
Client[StdioTransport]        # For stdio process communication
Client[FastMCPTransport]      # For in-memory server connections
```

#### Transport Inference
FastMCP automatically infers the correct transport based on the connection source:

```python
# Automatic inference from URL
client = Client("http://localhost:8080")  # -> StreamableHttpTransport
client = Client("http://localhost:8080/sse")  # -> SSETransport

# Automatic inference from file path
client = Client("./server.py")  # -> PythonStdioTransport
client = Client("./server.js")  # -> NodeStdioTransport

# Automatic inference from server instance
client = Client(fastmcp_server)  # -> FastMCPTransport

# Automatic inference from MCPConfig
config = {"mcpServers": {...}}
client = Client(config)  # -> MCPConfigTransport
```

#### Configuration Options

```python
Client(
    transport,                    # Connection source
    name=None,                   # Client name for logging
    roots=None,                  # Filesystem roots handler
    sampling_handler=None,       # Sampling request handler
    elicitation_handler=None,    # User input elicitation handler
    log_handler=None,            # Log message handler
    message_handler=None,        # Protocol message handler
    progress_handler=None,       # Progress notification handler
    timeout=None,                # Request timeout (timedelta/float/int)
    init_timeout=None,           # Initial connection timeout
    client_info=None,            # Client implementation info
    auth=None                    # Authentication (httpx.Auth/"oauth"/str)
)
```

### 1.2 Connection Management

#### Reentrant Context Managers
FastMCP clients support reentrant context managers with reference counting:

```python
async with client:  # Establishes session (ref count = 1)
    async with client:  # Reuses session (ref count = 2)
        await client.call_tool("tool1", {})
    # ref count = 1, session remains active
# ref count = 0, session closes
```

**Internal Mechanism:**
- **ClientSessionState**: Holds session state separate from configuration
  - `session`: Active ClientSession instance
  - `nesting_counter`: Reference count for active contexts
  - `lock`: anyio.Lock for thread-safe state management
  - `session_task`: Background task managing session lifecycle
  - `ready_event`: Signals when session is ready
  - `stop_event`: Signals session should terminate
  - `initialize_result`: Result of MCP initialization handshake

- **Session Lifecycle:**
  1. First `__aenter__`: Creates background session task, waits for ready
  2. Subsequent `__aenter__`: Increments counter, reuses session
  3. `__aexit__`: Decrements counter, closes only when counter reaches 0
  4. Force disconnect: Resets counter to 0, stops session immediately

#### Session Management Patterns

**Fresh Client Creation:**
```python
# Create independent client with same configuration
fresh_client = client.new()
async with fresh_client:
    # Completely separate session
    await fresh_client.call_tool("tool", {})
```

**Manual Connection Control:**
```python
await client.close()  # Force disconnect and cleanup
client.is_connected()  # Check connection status
```

### 1.3 Connection Management for HTTP/SSE/stdio

#### HTTP/SSE Transport Configuration

**SSETransport:**
```python
SSETransport(
    url: str | AnyUrl,                    # HTTP(S) URL to SSE endpoint
    headers: dict[str, str] | None,       # Custom HTTP headers
    auth: httpx.Auth | "oauth" | str,     # Authentication
    sse_read_timeout: timedelta | float,  # SSE stream read timeout
    httpx_client_factory: McpHttpClientFactory  # Custom HTTP client factory
)
```

**StreamableHttpTransport:**
```python
StreamableHttpTransport(
    url: str | AnyUrl,                    # HTTP(S) URL
    headers: dict[str, str] | None,       # Custom HTTP headers
    auth: httpx.Auth | "oauth" | str,     # Authentication
    sse_read_timeout: timedelta | float,  # Read timeout
    httpx_client_factory: McpHttpClientFactory  # Custom HTTP client factory
)
```

**Key Features:**
- Automatic header forwarding from HTTP request context (for proxy scenarios)
- Configurable read timeouts for long-running streams
- Custom httpx client factory for connection pooling
- URL path preservation (no automatic trailing slash modification)

#### Stdio Transport Configuration

**Base StdioTransport:**
```python
StdioTransport(
    command: str,              # Command to execute
    args: list[str],          # Command arguments
    env: dict[str, str],      # Environment variables
    cwd: str,                 # Working directory
    keep_alive: bool = True   # Keep subprocess alive between connections
)
```

**Specialized Transports:**

**PythonStdioTransport:**
```python
PythonStdioTransport(
    script_path: str | Path,
    args: list[str] | None,
    env: dict[str, str] | None,
    cwd: str | None,
    python_cmd: str = sys.executable,
    keep_alive: bool = True
)
```

**NodeStdioTransport:**
```python
NodeStdioTransport(
    script_path: str | Path,
    args: list[str] | None,
    env: dict[str, str] | None,
    cwd: str | None,
    node_cmd: str = "node",
    keep_alive: bool = True
)
```

**UvStdioTransport:**
```python
UvStdioTransport(
    command: str,
    args: list[str] | None,
    module: bool = False,
    project_directory: str | None,
    python_version: str | None,
    with_packages: list[str] | None,
    with_requirements: str | None,
    env_vars: dict[str, str] | None,
    keep_alive: bool = True
)
```

**NpxStdioTransport:**
```python
NpxStdioTransport(
    package: str,
    args: list[str] | None,
    project_directory: str | None,
    env_vars: dict[str, str] | None,
    use_package_lock: bool = True,
    keep_alive: bool = True
)
```

#### Stdio Connection Lifecycle

**Background Task Pattern:**
```python
# Internal connection task (_stdio_transport_connect_task)
1. Create StdioServerParameters with command/args/env/cwd
2. Enter stdio_client context to spawn subprocess
3. Create ClientSession with read/write streams
4. Signal ready via ready_event
5. Wait for stop_event (disconnect signal)
6. Clean up subprocess and session
```

**Keep-Alive Behavior:**
- `keep_alive=True` (default): Subprocess persists after context exit, reusable
- `keep_alive=False`: Subprocess terminates when context exits
- Automatic cleanup on garbage collection via `__del__`

### 1.4 OAuth 2.1 Provider Integration

#### OAuth Class Architecture

```python
class OAuth(OAuthClientProvider):
    """
    OAuth client provider implementing browser-based authentication flow
    with local callback server and persistent token storage.
    """

    def __init__(
        self,
        mcp_url: str,                              # Full MCP endpoint URL
        scopes: str | list[str] | None,           # OAuth scopes
        client_name: str = "FastMCP Client",      # Client registration name
        token_storage_cache_dir: Path | None,     # Cache directory
        additional_client_metadata: dict | None,  # Extra metadata
        callback_port: int | None                 # Fixed callback port
    )
```

#### OAuth Flow Components

**1. Client Registration:**
```python
OAuthClientMetadata(
    client_name="FastMCP Client",
    redirect_uris=["http://localhost:{port}/callback"],
    grant_types=["authorization_code", "refresh_token"],
    response_types=["code"],
    scope="space separated scopes"
)
```

**2. Token Storage (FileTokenStorage):**
- **Location**: `~/.fastmcp/oauth-mcp-client-cache/`
- **Files per server**:
  - `{scheme}://{host}_tokens.json`: OAuth tokens with expiry
  - `{scheme}://{host}_client_info.json`: Client registration data
- **Format**:
  ```json
  {
    "data": {
      "token_payload": {
        "access_token": "...",
        "refresh_token": "...",
        "expires_in": 3600,
        "token_type": "Bearer"
      },
      "expires_at": "2025-10-23T12:00:00Z"
    },
    "timestamp": "2025-10-23T11:00:00Z"
  }
  ```

**3. Authorization Flow:**
```python
async def redirect_handler(authorization_url: str):
    # Pre-flight check for invalid client_id
    response = await client.get(authorization_url)
    if response.status_code == 400:
        raise ClientNotFoundError("Cached credentials stale")

    # Open browser for user authorization
    webbrowser.open(authorization_url)
```

**4. Callback Handling:**
```python
async def callback_handler() -> tuple[str, str | None]:
    # Create Uvicorn server on callback port
    server = create_oauth_callback_server(port, server_url, response_future)

    # Run server with 5-minute timeout
    async with anyio.create_task_group():
        await server.serve()
        auth_code, state = await response_future  # Wait for callback

    return auth_code, state
```

**5. Token Refresh:**
- Automatic token expiry calculation
- `expires_in` recalculated relative to current time on load
- Expired tokens return `None`, triggering re-authentication

**6. Error Recovery:**
```python
async def async_auth_flow(request):
    try:
        # Attempt with cached credentials
        yield from super().async_auth_flow(request)
    except ClientNotFoundError:
        # Clear cache and retry once with fresh registration
        self.context.storage.clear()
        self._initialized = False
        yield from super().async_auth_flow(request)
```

### 1.5 Progress Monitoring and User Elicitation

#### Progress Monitoring

**Handler Interface:**
```python
ProgressHandler = Callable[
    [str | int, float, float | None, str | None],
    Awaitable[None]
]

# Parameters: (progress_token, progress, total, message)
async def progress_handler(
    token: str | int,
    progress: float,
    total: float | None,
    message: str | None
):
    print(f"Progress: {progress}/{total} - {message}")
```

**Client Integration:**
```python
async def default_progress_handler(token, progress, total, message):
    """Default handler that logs to logger"""
    if total:
        percentage = (progress / total) * 100
        logger.info(f"[{token}] {percentage:.1f}% - {message or ''}")
    else:
        logger.info(f"[{token}] {progress} - {message or ''}")

client = Client(
    transport,
    progress_handler=default_progress_handler
)
```

**Server-Side Progress Notifications:**
```python
await client.progress(
    progress_token="task-123",
    progress=50.0,
    total=100.0,
    message="Processing items..."
)
```

#### User Elicitation

**Handler Interface:**
```python
ElicitationHandler = Callable[
    [dict[str, Any]],
    Awaitable[dict[str, Any]]
]

# Receives prompts, returns user responses
async def elicitation_handler(prompts: dict) -> dict:
    responses = {}
    for key, prompt in prompts.items():
        responses[key] = input(f"{prompt}: ")
    return responses
```

**Usage Pattern:**
```python
def create_elicitation_callback(handler: ElicitationHandler):
    async def callback(params: dict) -> dict:
        return await handler(params)
    return callback

client = Client(
    transport,
    elicitation_handler=my_elicitation_handler
)
```

### 1.6 Error Handling and Retry Strategies

#### Error Handling Middleware

**ErrorHandlingMiddleware:**
```python
ErrorHandlingMiddleware(
    logger=None,                    # Logger instance (default: 'fastmcp.errors')
    include_traceback=False,        # Include full traceback in logs
    error_callback=None,            # Callback for each error
    transform_errors=True           # Transform to McpError
)
```

**Error Transformation:**
- `ValueError`, `TypeError` -> `-32602` (Invalid params)
- `FileNotFoundError`, `KeyError` -> `-32001` (Resource not found)
- `PermissionError` -> `-32000` (Permission denied)
- `TimeoutError`, `asyncio.TimeoutError` -> `-32000` (Request timeout)
- Other exceptions -> `-32603` (Internal error)

**Error Tracking:**
```python
middleware.get_error_stats()
# Returns: {"ValueError:tools/call": 5, "TimeoutError:resources/read": 2}
```

#### Retry Middleware

**RetryMiddleware:**
```python
RetryMiddleware(
    max_retries=3,                          # Maximum retry attempts
    base_delay=1.0,                         # Initial delay (seconds)
    max_delay=60.0,                         # Maximum delay (seconds)
    backoff_multiplier=2.0,                 # Exponential multiplier
    retry_exceptions=(ConnectionError, TimeoutError),
    logger=None
)
```

**Exponential Backoff:**
```python
delay = base_delay * (backoff_multiplier ** attempt)
delay = min(delay, max_delay)

# Example with base_delay=1.0, multiplier=2.0:
# Attempt 0: 1s
# Attempt 1: 2s
# Attempt 2: 4s
# Attempt 3: 8s (up to max_delay)
```

**Retry Logic:**
```python
for attempt in range(max_retries + 1):
    try:
        return await call_next(context)
    except Exception as error:
        if attempt == max_retries or not should_retry(error):
            raise

        delay = calculate_delay(attempt)
        logger.warning(f"Retry {attempt+1}/{max_retries+1} in {delay}s")
        await asyncio.sleep(delay)
```

#### Client-Level Error Handling

**Timeout Configuration:**
```python
client = Client(
    transport,
    timeout=30.0,              # Request timeout (seconds/timedelta)
    init_timeout=10.0          # Initial handshake timeout
)

# Per-request timeout override
await client.call_tool("tool", {}, timeout=60.0)
```

**Connection Error Recovery:**
```python
try:
    async with client:
        await client.initialize()
except TimeoutError:
    raise RuntimeError("Failed to initialize server session")
except anyio.ClosedResourceError:
    raise RuntimeError("Server session was closed unexpectedly")
except httpx.HTTPStatusError as e:
    # HTTP error with status code
    logger.error(f"HTTP {e.response.status_code}: {e.response.text}")
    raise
```

**OAuth Error Recovery:**
```python
# Automatic retry with fresh credentials on ClientNotFoundError
# Clears cached client info and re-registers
try:
    await client.some_request()
except ClientNotFoundError:
    # Already handled internally - retries once with fresh registration
    pass
```

---

## 2. OAuth Integration

### 2.1 Detailed OAuth Flow for Providers

#### Auth0 Integration

**Configuration:**
```python
from fastmcp.client.auth import OAuth

auth = OAuth(
    mcp_url="https://your-mcp-server.com/mcp",
    scopes=["openid", "profile", "email"],
    additional_client_metadata={
        "audience": "https://your-api.auth0.com/api/v2/",  # Required for JWT tokens
    }
)

transport = SSETransport(
    url="https://your-mcp-server.com/mcp/sse",
    auth=auth
)
```

**Server-Side (MCP Server with Auth0):**
```python
# Using OIDC Proxy for auto-discovery
from fastmcp.server.auth import OIDCProxyProvider

auth_provider = OIDCProxyProvider(
    issuer="https://your-tenant.auth0.com",
    client_id="your_auth0_client_id",
    client_secret="your_auth0_client_secret",
    extra_authorize_params={"audience": "https://your-api.auth0.com/api/v2/"}
)

mcp = FastMCP("Auth0MCP", auth=auth_provider)
```

**Auth0 Specifics:**
- Requires `audience` parameter for JWT tokens (not opaque tokens)
- Supports OIDC discovery at `https://{tenant}.auth0.com/.well-known/openid-configuration`
- Scopes: `openid`, `profile`, `email`, custom API scopes

#### Google OAuth Integration

**Client Configuration:**
```python
auth = OAuth(
    mcp_url="https://your-mcp-server.com/mcp",
    scopes=["openid", "email", "profile"],  # Google OIDC scopes
)

# Google automatically discovered via OIDC if server configured
```

**Server Configuration (OAuth Proxy):**
```python
from fastmcp.server.auth import OAuthProxyProvider

auth_provider = OAuthProxyProvider(
    authorize_url="https://accounts.google.com/o/oauth2/v2/auth",
    token_url="https://oauth2.googleapis.com/token",
    client_id="your_google_client_id.apps.googleusercontent.com",
    client_secret="your_google_client_secret",
    scopes=["openid", "email", "profile"]
)
```

**Google Cloud Console Setup:**
1. Create OAuth 2.0 Client ID (Web application)
2. Add authorized redirect URIs: `http://localhost:{port}/callback`
3. Configure OAuth consent screen
4. Use Client ID and Secret in server configuration

#### GitHub OAuth Integration

**Client Configuration:**
```python
auth = OAuth(
    mcp_url="https://your-mcp-server.com/mcp",
    scopes=["user", "repo", "read:org"],
)
```

**Server Configuration:**
```python
from fastmcp.server.auth import OAuthProxyProvider

auth_provider = OAuthProxyProvider(
    authorize_url="https://github.com/login/oauth/authorize",
    token_url="https://github.com/login/oauth/access_token",
    client_id="your_github_client_id",
    client_secret="your_github_client_secret",
    scopes=["user", "repo"]
)
```

**GitHub Developer Settings:**
1. Register OAuth App at https://github.com/settings/developers
2. Set Homepage URL: `http://localhost:3000` (or your app URL)
3. Set Authorization callback URL: `http://localhost:{port}/callback`
4. Use Client ID and Secret

#### Azure AD Integration

**Client Configuration:**
```python
auth = OAuth(
    mcp_url="https://your-mcp-server.com/mcp",
    scopes=["openid", "profile", "email", "User.Read"],
)
```

**Server Configuration (OIDC Proxy):**
```python
from fastmcp.server.auth import OIDCProxyProvider

auth_provider = OIDCProxyProvider(
    issuer="https://login.microsoftonline.com/{tenant_id}/v2.0",
    client_id="your_azure_client_id",
    client_secret="your_azure_client_secret",
    scopes=["openid", "profile", "email", "User.Read"]
)
```

**Azure Portal Setup:**
1. Register app in Azure AD
2. Add redirect URI: `http://localhost:{port}/callback`
3. Generate client secret
4. Configure API permissions
5. Use Application (client) ID and secret

### 2.2 Token Storage and Refresh Mechanisms

#### Token Storage Architecture

**FileTokenStorage Implementation:**
```python
class FileTokenStorage(TokenStorage):
    def __init__(self, server_url: str, cache_dir: Path | None):
        self.server_url = server_url
        self._storage = JSONFileStorage(cache_dir or default_cache_dir())

    @staticmethod
    def get_base_url(url: str) -> str:
        """Extract base URL for storage key"""
        parsed = urlparse(url)
        return f"{parsed.scheme}://{parsed.netloc}"

    def _get_storage_key(self, file_type: Literal["client_info", "tokens"]) -> str:
        base_url = self.get_base_url(self.server_url)
        return f"{base_url}_{file_type}"
```

**Token Persistence:**
```python
async def set_tokens(self, tokens: OAuthToken) -> None:
    # Calculate absolute expiry
    expires_at = None
    if tokens.expires_in:
        expires_at = datetime.now(timezone.utc) + timedelta(seconds=tokens.expires_in)

    # Store with metadata
    stored = StoredToken(
        token_payload=tokens,
        expires_at=expires_at
    )
    await self._storage.set(key, stored.model_dump(mode="json"))
```

**Token Retrieval with Expiry Check:**
```python
async def get_tokens(self) -> OAuthToken | None:
    data = await self._storage.get(key)
    if not data:
        return None

    stored = stored_token_adapter.validate_python(data)

    # Check expiry
    if stored.expires_at:
        now = datetime.now(timezone.utc)
        if now >= stored.expires_at:
            return None  # Token expired

        # Recalculate expires_in relative to now
        remaining = stored.expires_at - now
        stored.token_payload.expires_in = max(0, int(remaining.total_seconds()))

    return stored.token_payload
```

#### Token Refresh Flow

**Automatic Refresh (via OAuthClientProvider):**
```python
# Inherited from mcp.client.auth.OAuthClientProvider
async def async_auth_flow(self, request):
    # Check if tokens are expired
    if self.context.current_tokens:
        if self._is_expired(self.context.current_tokens):
            # Refresh using refresh_token
            new_tokens = await self._refresh_tokens(
                self.context.current_tokens.refresh_token
            )
            await self.context.storage.set_tokens(new_tokens)
            self.context.current_tokens = new_tokens

    # Add access token to request
    request.headers["Authorization"] = f"Bearer {self.context.current_tokens.access_token}"
    yield request
```

**Token Expiry Management:**
```python
def update_token_expiry(self, tokens: OAuthToken):
    """Update context's token_expiry_time based on expires_in"""
    if tokens.expires_in:
        self.context.token_expiry_time = (
            datetime.now(timezone.utc) +
            timedelta(seconds=tokens.expires_in)
        )

def _is_expired(self, tokens: OAuthToken) -> bool:
    """Check if tokens are expired with 60-second buffer"""
    if not self.context.token_expiry_time:
        return False

    buffer = timedelta(seconds=60)
    return datetime.now(timezone.utc) + buffer >= self.context.token_expiry_time
```

#### Client Info Storage

**Client Registration Persistence:**
```python
async def set_client_info(self, client_info: OAuthClientInformationFull):
    """Save client registration data"""
    key = self._get_storage_key("client_info")
    await self._storage.set(key, client_info.model_dump(mode="json"))

async def get_client_info(self) -> OAuthClientInformationFull | None:
    """Load client info with validation"""
    data = await self._storage.get(key)
    if not data:
        return None

    client_info = OAuthClientInformationFull.model_validate(data)

    # Validate tokens exist (incomplete OAuth flow check)
    tokens = await self.get_tokens()
    if not tokens:
        # Clear incomplete client info
        await self._storage.delete(key)
        return None

    return client_info
```

**Cache Management:**
```python
def clear(self) -> None:
    """Clear all cached data for this server"""
    for file_type in ["client_info", "tokens"]:
        path = self._get_file_path(file_type)
        path.unlink(missing_ok=True)

@classmethod
def clear_all(cls, cache_dir: Path | None = None) -> None:
    """Clear all cached data for all servers"""
    cache_dir = cache_dir or default_cache_dir()
    for file_type in ["client_info", "tokens"]:
        for file in cache_dir.glob(f"*_{file_type}.json"):
            file.unlink(missing_ok=True)
```

### 2.3 PKCE Implementation Details

#### PKCE (Proof Key for Code Exchange)

**Implementation in OAuthClientProvider:**
```python
from secrets import token_urlsafe
import hashlib
import base64

def generate_pkce_pair() -> tuple[str, str]:
    """Generate code_verifier and code_challenge"""
    # code_verifier: 43-128 character random string
    code_verifier = token_urlsafe(64)  # ~86 characters

    # code_challenge: SHA256 hash of verifier
    challenge_bytes = hashlib.sha256(code_verifier.encode()).digest()
    code_challenge = base64.urlsafe_b64encode(challenge_bytes).decode().rstrip('=')

    return code_verifier, code_challenge
```

**Authorization Request with PKCE:**
```python
code_verifier, code_challenge = generate_pkce_pair()

# Store verifier for token exchange
self.context.code_verifier = code_verifier

# Add to authorization URL
auth_params = {
    "client_id": client_id,
    "redirect_uri": redirect_uri,
    "response_type": "code",
    "scope": scopes,
    "state": state,
    "code_challenge": code_challenge,
    "code_challenge_method": "S256"
}

authorization_url = f"{authorize_url}?{urlencode(auth_params)}"
```

**Token Exchange with PKCE:**
```python
token_params = {
    "grant_type": "authorization_code",
    "code": auth_code,
    "redirect_uri": redirect_uri,
    "client_id": client_id,
    "code_verifier": self.context.code_verifier  # Proves code ownership
}

# Exchange code for tokens
response = await httpx.post(token_url, data=token_params)
tokens = OAuthToken.model_validate(response.json())
```

**PKCE Security:**
- **Without PKCE**: Authorization code can be intercepted and used by attacker
- **With PKCE**: Attacker cannot exchange code without code_verifier
- Required for public clients (mobile apps, SPAs)
- Recommended for all OAuth flows

### 2.4 Security Best Practices

#### Token Security

**1. Secure Token Storage:**
```python
# Use SecretStr for in-memory tokens
from pydantic import SecretStr

class SecureTokenStorage:
    def __init__(self):
        self._access_token = None

    def set_token(self, token: str):
        self._access_token = SecretStr(token)

    def get_token(self) -> str:
        return self._access_token.get_secret_value()
```

**2. File Permissions:**
```python
# Restrict cache directory permissions
cache_dir = Path.home() / ".fastmcp" / "oauth-mcp-client-cache"
cache_dir.mkdir(mode=0o700, parents=True, exist_ok=True)

# Set restrictive file permissions
for file in cache_dir.glob("*.json"):
    file.chmod(0o600)  # Owner read/write only
```

**3. Token Rotation:**
```python
# Always use refresh tokens when available
if tokens.refresh_token:
    # Store refresh token securely
    await storage.set_tokens(tokens)

    # Rotate access token before expiry
    if self._is_expired(tokens):
        new_tokens = await self._refresh_tokens(tokens.refresh_token)
        await storage.set_tokens(new_tokens)
```

#### OAuth Flow Security

**1. State Parameter Validation:**
```python
import secrets

# Generate cryptographically secure state
state = secrets.token_urlsafe(32)
self.context.state = state

# Include in authorization request
auth_params["state"] = state

# Validate on callback
if callback_state != self.context.state:
    raise SecurityError("State mismatch - possible CSRF attack")
```

**2. Redirect URI Validation:**
```python
# Use exact redirect URI match
redirect_uri = f"http://localhost:{port}/callback"

# Server must validate exact match
if request_redirect_uri != stored_redirect_uri:
    raise SecurityError("Redirect URI mismatch")
```

**3. TLS/HTTPS:**
```python
# Enforce HTTPS for production
if mcp_url.startswith("http://") and not is_development:
    raise SecurityError("HTTPS required for production OAuth")

# Use HTTPS for OAuth endpoints
auth_provider = OAuthProxyProvider(
    authorize_url="https://provider.com/oauth/authorize",  # HTTPS only
    token_url="https://provider.com/oauth/token"
)
```

**4. Scope Minimization:**
```python
# Request only necessary scopes
scopes = ["openid", "profile"]  # Minimal for identity
# Avoid: ["openid", "profile", "email", "admin:all", "delete:everything"]

auth = OAuth(
    mcp_url=mcp_url,
    scopes=scopes  # Principle of least privilege
)
```

**5. Client Secret Protection:**
```python
# Never commit client secrets
# Use environment variables
client_secret = os.getenv("OAUTH_CLIENT_SECRET")
if not client_secret:
    raise ValueError("OAUTH_CLIENT_SECRET not set")

# Or use secret management service
from gcp_secret_manager import get_secret
client_secret = get_secret("oauth-client-secret")
```

**6. Token Expiry Enforcement:**
```python
# Validate token expiry on every use
def validate_token(self, tokens: OAuthToken) -> bool:
    if not tokens.expires_in:
        return True  # No expiry specified

    if self._is_expired(tokens):
        logger.warning("Token expired, refreshing...")
        return False

    # Warn if expiring soon (< 5 minutes)
    remaining = self.context.token_expiry_time - datetime.now(timezone.utc)
    if remaining < timedelta(minutes=5):
        logger.info(f"Token expiring in {remaining.total_seconds()}s")

    return True
```

**7. Error Information Leakage:**
```python
# Don't expose sensitive error details
try:
    tokens = await self._exchange_code(code)
except OAuthError as e:
    # Log full error for debugging
    logger.error(f"OAuth error: {e.error_description}")

    # Return generic error to client
    raise OAuthError("Authentication failed") from e
```

---

## 3. MCP Server Types

### 3.1 HTTP Transport Implementation

#### StreamableHttpTransport (Recommended)

**Protocol:** HTTP/1.1 with chunked transfer encoding for bidirectional streaming

**Transport Creation:**
```python
from fastmcp.client.transports import StreamableHttpTransport

transport = StreamableHttpTransport(
    url="https://api.example.com/mcp",
    headers={"X-Custom-Header": "value"},
    auth="oauth",  # or BearerAuth("token") or custom httpx.Auth
    sse_read_timeout=300.0,  # 5 minute read timeout
    httpx_client_factory=custom_factory  # For connection pooling
)
```

**Connection Process:**
```python
async with transport.connect_session() as session:
    # 1. Establish HTTP connection
    # 2. Set up bidirectional streams
    # 3. Initialize MCP handshake
    # 4. Return ClientSession
    pass
```

**Internal Implementation:**
```python
from mcp.client.streamable_http import streamablehttp_client

@contextlib.asynccontextmanager
async def connect_session(self, **session_kwargs):
    client_kwargs = {
        "headers": get_http_headers() | self.headers,
        "timeout": session_kwargs.get("read_timeout_seconds"),
    }

    if self.sse_read_timeout:
        client_kwargs["sse_read_timeout"] = self.sse_read_timeout

    if self.httpx_client_factory:
        client_kwargs["httpx_client_factory"] = self.httpx_client_factory

    async with streamablehttp_client(
        self.url,
        auth=self.auth,
        **client_kwargs
    ) as transport:
        read_stream, write_stream, _ = transport
        async with ClientSession(read_stream, write_stream, **session_kwargs) as session:
            yield session
```

**Authentication Methods:**

1. **OAuth 2.1:**
```python
from fastmcp.client.auth import OAuth

auth = OAuth(
    mcp_url="https://api.example.com/mcp",
    scopes=["mcp:read", "mcp:write"]
)

transport = StreamableHttpTransport(url="https://api.example.com/mcp", auth=auth)
```

2. **Bearer Token:**
```python
from fastmcp.client.auth import BearerAuth

auth = BearerAuth("your_bearer_token_here")
transport = StreamableHttpTransport(url="https://api.example.com/mcp", auth=auth)

# Or shorthand
transport = StreamableHttpTransport(url="https://api.example.com/mcp", auth="your_token")
```

3. **Custom HTTPX Auth:**
```python
import httpx

class APIKeyAuth(httpx.Auth):
    def __init__(self, api_key: str):
        self.api_key = api_key

    def auth_flow(self, request):
        request.headers["X-API-Key"] = self.api_key
        yield request

transport = StreamableHttpTransport(
    url="https://api.example.com/mcp",
    auth=APIKeyAuth("your_api_key")
)
```

**Connection Pooling:**
```python
from mcp.shared._httpx_utils import McpHttpClientFactory
import httpx

class CustomClientFactory(McpHttpClientFactory):
    def __call__(self) -> httpx.AsyncClient:
        return httpx.AsyncClient(
            limits=httpx.Limits(
                max_keepalive_connections=100,
                max_connections=200,
                keepalive_expiry=30.0
            ),
            timeout=httpx.Timeout(30.0, connect=10.0),
            http2=True  # Enable HTTP/2
        )

transport = StreamableHttpTransport(
    url="https://api.example.com/mcp",
    httpx_client_factory=CustomClientFactory()
)
```

### 3.2 SSE Transport Implementation

#### SSETransport (Legacy)

**Protocol:** Server-Sent Events (unidirectional server->client, HTTP POST for client->server)

**Transport Creation:**
```python
from fastmcp.client.transports import SSETransport

transport = SSETransport(
    url="https://api.example.com/mcp/sse",  # Must end in /sse or contain "sse"
    headers={"X-Custom-Header": "value"},
    auth="oauth",
    sse_read_timeout=300.0,
    httpx_client_factory=custom_factory
)
```

**Connection Process:**
```python
from mcp.client.sse import sse_client

@contextlib.asynccontextmanager
async def connect_session(self, **session_kwargs):
    client_kwargs = {
        "headers": get_http_headers() | self.headers,
    }

    if self.sse_read_timeout:
        client_kwargs["sse_read_timeout"] = self.sse_read_timeout.total_seconds()

    if session_kwargs.get("read_timeout_seconds"):
        client_kwargs["timeout"] = session_kwargs["read_timeout_seconds"].total_seconds()

    async with sse_client(self.url, auth=self.auth, **client_kwargs) as transport:
        read_stream, write_stream = transport
        async with ClientSession(read_stream, write_stream, **session_kwargs) as session:
            yield session
```

**Authentication Methods:** Same as StreamableHttpTransport (OAuth, Bearer, Custom)

**SSE vs StreamableHttp:**
- **SSE**: Legacy protocol, unidirectional streaming, requires POST for client messages
- **StreamableHttp**: Modern protocol, bidirectional streaming, single HTTP connection
- **Migration**: Use StreamableHttpTransport for new implementations

### 3.3 stdio Transport Implementation

#### Process-Based Communication

**Transport Creation:**
```python
from fastmcp.client.transports import PythonStdioTransport

transport = PythonStdioTransport(
    script_path="/path/to/server.py",
    args=["--verbose"],
    env={"MCP_ENV": "production"},
    cwd="/working/directory",
    python_cmd="/usr/bin/python3",
    keep_alive=True  # Persist subprocess between connections
)
```

**Connection Lifecycle:**
```python
async def _stdio_transport_connect_task(
    command, args, env, cwd,
    session_kwargs, ready_event, stop_event, session_future
):
    async with contextlib.AsyncExitStack() as stack:
        # 1. Create server parameters
        server_params = StdioServerParameters(
            command=command,
            args=args,
            env=env,
            cwd=cwd
        )

        # 2. Spawn subprocess with stdio pipes
        transport = await stack.enter_async_context(
            stdio_client(server_params)
        )
        read_stream, write_stream = transport

        # 3. Create MCP session
        session = await stack.enter_async_context(
            ClientSession(read_stream, write_stream, **session_kwargs)
        )
        session_future.set_result(session)

        # 4. Signal ready
        ready_event.set()

        # 5. Wait for disconnect
        await stop_event.wait()

        # 6. Clean up (subprocess terminated)
```

**Keep-Alive Behavior:**
```python
@contextlib.asynccontextmanager
async def connect_session(self, **session_kwargs):
    try:
        await self.connect(**session_kwargs)
        yield self._session
    finally:
        if not self.keep_alive:
            await self.disconnect()  # Terminate subprocess
        else:
            # Subprocess remains alive for next connection
            pass
```

**Authentication Methods:**

stdio transports don't support HTTP-based authentication. Authentication is handled through:

1. **Environment Variables:**
```python
transport = PythonStdioTransport(
    script_path="server.py",
    env={
        "MCP_API_KEY": "secret_key",
        "MCP_USER": "username"
    }
)
```

2. **Command-Line Arguments:**
```python
transport = PythonStdioTransport(
    script_path="server.py",
    args=["--api-key", "secret_key", "--user", "username"]
)
```

3. **Configuration Files:**
```python
transport = PythonStdioTransport(
    script_path="server.py",
    env={"MCP_CONFIG": "/path/to/config.json"}
)

# config.json contains authentication credentials
```

#### Specialized stdio Transports

**UvStdioTransport (Python with uv):**
```python
transport = UvStdioTransport(
    command="my_mcp_server",  # Module or script name
    args=["--port", "8080"],
    module=True,  # Run as module (-m flag)
    project_directory="/path/to/project",
    python_version="3.11",
    with_packages=["fastmcp>=2.0", "httpx"],
    with_requirements="requirements.txt",
    env_vars={"DEBUG": "true"},
    keep_alive=True
)

# Runs: uv run --python 3.11 --directory /path/to/project \
#       --with fastmcp>=2.0 --with httpx \
#       --with-requirements requirements.txt \
#       --module my_mcp_server --port 8080
```

**NpxStdioTransport (Node.js with npx):**
```python
transport = NpxStdioTransport(
    package="@example/mcp-server",
    args=["--config", "config.json"],
    project_directory="/path/to/project",
    env_vars={"NODE_ENV": "production"},
    use_package_lock=True,  # Use --prefer-offline
    keep_alive=True
)

# Runs: npx --prefer-offline @example/mcp-server --config config.json
```

**NodeStdioTransport (Direct Node.js):**
```python
transport = NodeStdioTransport(
    script_path="/path/to/server.js",
    args=["--verbose"],
    env={"NODE_ENV": "production"},
    cwd="/working/directory",
    node_cmd="/usr/bin/node",
    keep_alive=True
)

# Runs: /usr/bin/node /path/to/server.js --verbose
```

---

## 4. Production Requirements

### 4.1 Async/Await Patterns

#### Async Client Usage

**Basic Pattern:**
```python
async def use_mcp_client():
    client = Client("https://api.example.com/mcp")

    async with client:
        # All operations are async
        tools = await client.list_tools()
        result = await client.call_tool("tool_name", {"arg": "value"})
        resources = await client.list_resources()
```

**Concurrent Operations:**
```python
import asyncio

async def concurrent_operations(client: Client):
    async with client:
        # Run multiple operations concurrently
        tools_task = client.list_tools()
        resources_task = client.list_resources()
        prompts_task = client.list_prompts()

        tools, resources, prompts = await asyncio.gather(
            tools_task,
            resources_task,
            prompts_task
        )
```

**Task Groups for Structured Concurrency:**
```python
import anyio

async def structured_operations(client: Client):
    async with client:
        async with anyio.create_task_group() as tg:
            # Start multiple operations
            tg.start_soon(process_tools, client)
            tg.start_soon(process_resources, client)
            tg.start_soon(process_prompts, client)

            # All tasks complete before exiting context
```

**Timeout Management:**
```python
async def with_timeout(client: Client):
    async with client:
        # Per-operation timeout
        try:
            with anyio.fail_after(30.0):  # 30 second timeout
                result = await client.call_tool("long_operation", {})
        except TimeoutError:
            logger.error("Operation timed out")
```

#### Background Task Patterns

**Long-Running Client:**
```python
class MCPService:
    def __init__(self):
        self.client = None
        self._task = None

    async def start(self):
        self.client = Client("https://api.example.com/mcp")
        await self.client.__aenter__()

    async def stop(self):
        if self.client:
            await self.client.__aexit__(None, None, None)

    async def run_forever(self):
        await self.start()
        try:
            while True:
                await asyncio.sleep(60)
                # Periodic operations
                await self.health_check()
        finally:
            await self.stop()

# Usage
service = MCPService()
asyncio.create_task(service.run_forever())
```

**Graceful Shutdown:**
```python
class GracefulMCPService:
    def __init__(self):
        self.client = None
        self.shutdown_event = asyncio.Event()

    async def run(self):
        async with Client("https://api.example.com/mcp") as client:
            self.client = client

            # Run until shutdown signal
            await self.shutdown_event.wait()

    async def shutdown(self):
        self.shutdown_event.set()
        # Wait for clean shutdown
        await asyncio.sleep(0.5)

# Signal handling
import signal

service = GracefulMCPService()

def handle_signal(sig, frame):
    asyncio.create_task(service.shutdown())

signal.signal(signal.SIGTERM, handle_signal)
signal.signal(signal.SIGINT, handle_signal)

await service.run()
```

### 4.2 Connection Pooling

#### HTTPX Connection Pooling

**Custom Client Factory:**
```python
from mcp.shared._httpx_utils import McpHttpClientFactory
import httpx

class ProductionClientFactory(McpHttpClientFactory):
    def __call__(self) -> httpx.AsyncClient:
        return httpx.AsyncClient(
            # Connection limits
            limits=httpx.Limits(
                max_keepalive_connections=100,  # Keep 100 connections alive
                max_connections=200,             # Maximum 200 total connections
                keepalive_expiry=30.0            # Keep connections alive for 30s
            ),

            # Timeouts
            timeout=httpx.Timeout(
                connect=10.0,   # 10s to establish connection
                read=30.0,      # 30s to read response
                write=10.0,     # 10s to write request
                pool=5.0        # 5s to acquire connection from pool
            ),

            # HTTP/2 support
            http2=True,

            # Retries
            transport=httpx.AsyncHTTPTransport(retries=3)
        )

transport = StreamableHttpTransport(
    url="https://api.example.com/mcp",
    httpx_client_factory=ProductionClientFactory()
)
```

**Pool Configuration Guidelines:**
- **max_connections**: 2-4x expected concurrent requests
- **max_keepalive_connections**: ~50% of max_connections
- **keepalive_expiry**: 30-60 seconds for most APIs
- **connect timeout**: 5-10 seconds
- **read timeout**: Depends on operation (30-300 seconds)

#### Database Connection Pooling

**For MCP Servers with Database Access:**
```python
from fastmcp import FastMCP
import asyncpg

class DatabaseMCPServer:
    def __init__(self):
        self.mcp = FastMCP("DatabaseMCP")
        self.db_pool = None

    async def initialize(self):
        # Create connection pool
        self.db_pool = await asyncpg.create_pool(
            dsn="postgresql://user:pass@localhost/db",
            min_size=10,           # Minimum connections
            max_size=100,          # Maximum connections
            max_queries=50000,     # Rotate connections after 50k queries
            max_inactive_connection_lifetime=300.0,  # 5 minutes
            command_timeout=60.0   # 60 second query timeout
        )

    async def cleanup(self):
        if self.db_pool:
            await self.db_pool.close()

    @self.mcp.tool()
    async def query_database(self, sql: str):
        async with self.db_pool.acquire() as conn:
            return await conn.fetch(sql)
```

### 4.3 Rate Limiting

#### Middleware-Based Rate Limiting

**Global Rate Limiting:**
```python
from fastmcp.server.middleware.rate_limiting import RateLimitMiddleware

mcp = FastMCP("RateLimitedMCP")

rate_limiter = RateLimitMiddleware(
    max_requests=100,        # 100 requests
    time_window=60.0,        # per 60 seconds
    identifier_func=lambda ctx: ctx.headers.get("X-User-ID", "anonymous"),
    rate_limit_exceeded_callback=lambda ctx: logger.warning(f"Rate limit exceeded: {ctx}")
)

mcp.add_middleware(rate_limiter)
```

**Per-Tool Rate Limiting:**
```python
from functools import wraps
import time
from collections import defaultdict

class ToolRateLimiter:
    def __init__(self, max_calls: int, window: float):
        self.max_calls = max_calls
        self.window = window
        self.calls = defaultdict(list)

    def limit(self, func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            now = time.time()
            tool_name = func.__name__

            # Remove old calls outside window
            self.calls[tool_name] = [
                t for t in self.calls[tool_name]
                if now - t < self.window
            ]

            # Check limit
            if len(self.calls[tool_name]) >= self.max_calls:
                raise ToolError(f"Rate limit exceeded: {self.max_calls} calls per {self.window}s")

            # Record call
            self.calls[tool_name].append(now)

            return await func(*args, **kwargs)
        return wrapper

limiter = ToolRateLimiter(max_calls=10, window=60.0)

@mcp.tool()
@limiter.limit
async def expensive_operation():
    # Limited to 10 calls per minute
    pass
```

**Token Bucket Algorithm:**
```python
import asyncio
import time

class TokenBucket:
    def __init__(self, rate: float, capacity: int):
        self.rate = rate          # Tokens per second
        self.capacity = capacity  # Maximum tokens
        self.tokens = capacity
        self.last_update = time.time()
        self.lock = asyncio.Lock()

    async def acquire(self, tokens: int = 1) -> bool:
        async with self.lock:
            now = time.time()
            elapsed = now - self.last_update

            # Add tokens based on elapsed time
            self.tokens = min(
                self.capacity,
                self.tokens + elapsed * self.rate
            )
            self.last_update = now

            # Check if enough tokens
            if self.tokens >= tokens:
                self.tokens -= tokens
                return True
            return False

# Usage
bucket = TokenBucket(rate=10.0, capacity=100)  # 10 req/s, burst 100

@mcp.tool()
async def rate_limited_tool():
    if not await bucket.acquire():
        raise ToolError("Rate limit exceeded, please try again later")

    # Execute tool logic
```

### 4.4 Logging and Monitoring

#### Structured Logging

**Client-Side Logging:**
```python
import logging
import json
from datetime import datetime

class StructuredLogger:
    def __init__(self, name: str):
        self.logger = logging.getLogger(name)
        handler = logging.StreamHandler()
        handler.setFormatter(logging.Formatter('%(message)s'))
        self.logger.addHandler(handler)
        self.logger.setLevel(logging.INFO)

    def log(self, level: str, event: str, **kwargs):
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "level": level,
            "event": event,
            **kwargs
        }
        self.logger.info(json.dumps(log_entry))

logger = StructuredLogger("mcp.client")

async def log_handler(level, logger_name, data):
    """MCP log callback"""
    logger.log(
        level=level,
        event="mcp_log",
        logger=logger_name,
        data=data
    )

client = Client(
    transport,
    log_handler=log_handler
)
```

**Server-Side Logging Middleware:**
```python
from fastmcp.server.middleware import Middleware, MiddlewareContext, CallNext
import time

class LoggingMiddleware(Middleware):
    def __init__(self, logger):
        self.logger = logger

    async def on_request(self, context: MiddlewareContext, call_next: CallNext):
        start_time = time.time()

        self.logger.log(
            "info",
            "request_start",
            method=context.method,
            params=context.params
        )

        try:
            result = await call_next(context)

            duration = time.time() - start_time
            self.logger.log(
                "info",
                "request_success",
                method=context.method,
                duration_ms=int(duration * 1000)
            )

            return result

        except Exception as e:
            duration = time.time() - start_time
            self.logger.log(
                "error",
                "request_failure",
                method=context.method,
                error=str(e),
                error_type=type(e).__name__,
                duration_ms=int(duration * 1000)
            )
            raise

mcp = FastMCP("LoggedMCP")
mcp.add_middleware(LoggingMiddleware(logger))
```

#### Metrics Collection

**Prometheus Metrics:**
```python
from prometheus_client import Counter, Histogram, Gauge
import time

# Define metrics
mcp_requests_total = Counter(
    'mcp_requests_total',
    'Total MCP requests',
    ['method', 'status']
)

mcp_request_duration = Histogram(
    'mcp_request_duration_seconds',
    'MCP request duration',
    ['method']
)

mcp_active_connections = Gauge(
    'mcp_active_connections',
    'Active MCP connections'
)

class MetricsMiddleware(Middleware):
    async def on_request(self, context: MiddlewareContext, call_next: CallNext):
        start_time = time.time()
        mcp_active_connections.inc()

        try:
            result = await call_next(context)

            # Record success
            mcp_requests_total.labels(
                method=context.method,
                status='success'
            ).inc()

            return result

        except Exception:
            # Record failure
            mcp_requests_total.labels(
                method=context.method,
                status='error'
            ).inc()
            raise

        finally:
            # Record duration
            duration = time.time() - start_time
            mcp_request_duration.labels(
                method=context.method
            ).observe(duration)

            mcp_active_connections.dec()

# Expose metrics endpoint
from prometheus_client import make_asgi_app

metrics_app = make_asgi_app()

# Mount to FastMCP HTTP server
mcp.http_app.mount("/metrics", metrics_app)
```

**Health Checks:**
```python
@mcp.resource("health://status")
async def health_check():
    """Health check endpoint"""
    checks = {
        "database": await check_database(),
        "cache": await check_cache(),
        "external_api": await check_external_api()
    }

    all_healthy = all(checks.values())

    return {
        "status": "healthy" if all_healthy else "unhealthy",
        "checks": checks,
        "timestamp": datetime.utcnow().isoformat()
    }

async def check_database():
    try:
        async with db_pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        return True
    except Exception:
        return False
```

### 4.5 Error Recovery

#### Automatic Reconnection

**Client Reconnection Logic:**
```python
class ResilientMCPClient:
    def __init__(self, transport, max_retries=3, backoff_base=2.0):
        self.client = Client(transport)
        self.max_retries = max_retries
        self.backoff_base = backoff_base

    async def call_with_retry(self, tool_name: str, arguments: dict):
        last_error = None

        for attempt in range(self.max_retries):
            try:
                async with self.client:
                    return await self.client.call_tool(tool_name, arguments)

            except (ConnectionError, TimeoutError) as e:
                last_error = e

                if attempt < self.max_retries - 1:
                    delay = self.backoff_base ** attempt
                    logger.warning(
                        f"Connection failed (attempt {attempt + 1}), "
                        f"retrying in {delay}s..."
                    )
                    await asyncio.sleep(delay)

                    # Create fresh client for retry
                    self.client = self.client.new()

        raise last_error
```

**Circuit Breaker Pattern:**
```python
from enum import Enum
import time

class CircuitState(Enum):
    CLOSED = "closed"      # Normal operation
    OPEN = "open"          # Failing, reject requests
    HALF_OPEN = "half_open"  # Testing recovery

class CircuitBreaker:
    def __init__(
        self,
        failure_threshold: int = 5,
        timeout: float = 60.0,
        success_threshold: int = 2
    ):
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.success_threshold = success_threshold

        self.state = CircuitState.CLOSED
        self.failures = 0
        self.successes = 0
        self.last_failure_time = None

    async def call(self, func, *args, **kwargs):
        # Check if circuit should close
        if self.state == CircuitState.OPEN:
            if time.time() - self.last_failure_time >= self.timeout:
                self.state = CircuitState.HALF_OPEN
                self.successes = 0
            else:
                raise CircuitBreakerOpenError("Circuit breaker is open")

        try:
            result = await func(*args, **kwargs)

            # Success handling
            if self.state == CircuitState.HALF_OPEN:
                self.successes += 1
                if self.successes >= self.success_threshold:
                    self.state = CircuitState.CLOSED
                    self.failures = 0

            return result

        except Exception as e:
            # Failure handling
            self.failures += 1
            self.last_failure_time = time.time()

            if self.failures >= self.failure_threshold:
                self.state = CircuitState.OPEN
                logger.error("Circuit breaker opened")

            raise e

# Usage
breaker = CircuitBreaker(failure_threshold=5, timeout=60.0)

async def call_mcp_tool(client, tool, args):
    return await breaker.call(client.call_tool, tool, args)
```

---

## 5. Integration Considerations

### 5.1 Wrapping FastMCP for Go Consumption

#### Architecture Options

**1. Process-Based Communication (Current Implementation)**

**Pros:**
- No CGO dependency
- Simple implementation
- Language isolation

**Cons:**
- High overhead (~50ms latency)
- Process management complexity
- Limited error handling

**Go Client:**
```go
type FastMCPClient struct {
    process *exec.Cmd
    stdin   *bufio.Writer
    stdout  *bufio.Scanner
    mutex   sync.RWMutex
    clients map[string]bool
}
```

**Python Wrapper:**
```python
async def main():
    wrapper = FastMCPWrapper()

    # Read JSON commands from stdin
    command = json.loads(sys.stdin.readline())

    # Execute and return JSON response
    result = await wrapper.execute(command)
    print(json.dumps(result))
```

**2. gRPC Microservice (Recommended for Production)**

**Pros:**
- High performance (~5ms latency)
- Built-in load balancing
- Easy to scale independently
- Language agnostic

**Cons:**
- Additional infrastructure
- Service discovery needed

**Protocol Definition:**
```protobuf
syntax = "proto3";

package fastmcp;

service FastMCPService {
    rpc Connect(ConnectRequest) returns (ConnectResponse);
    rpc Disconnect(DisconnectRequest) returns (DisconnectResponse);
    rpc ListTools(ListToolsRequest) returns (ListToolsResponse);
    rpc CallTool(CallToolRequest) returns (CallToolResponse);
    rpc ListResources(ListResourcesRequest) returns (ListResourcesResponse);
    rpc ReadResource(ReadResourceRequest) returns (ReadResourceResponse);
    rpc ListPrompts(ListPromptsRequest) returns (ListPromptsResponse);
    rpc GetPrompt(GetPromptRequest) returns (GetPromptResponse);
}

message ConnectRequest {
    string id = 1;
    string name = 2;
    string type = 3;  // http, sse, stdio
    string endpoint = 4;
    string auth_type = 5;  // bearer, oauth, none
    map<string, string> config = 6;
    map<string, string> auth = 7;
}

message ConnectResponse {
    bool success = 1;
    string error = 2;
}

message Tool {
    string name = 1;
    string description = 2;
    string input_schema = 3;  // JSON string
}

message CallToolRequest {
    string mcp_id = 1;
    string tool_name = 2;
    string arguments = 3;  // JSON string
}

message CallToolResponse {
    string result = 1;  // JSON string
    string error = 2;
}
```

**Python gRPC Server:**
```python
import grpc
from concurrent import futures
from fastmcp import Client
from fastmcp.client.transports import (
    HTTPTransport, SSETransport, StdioTransport
)

class FastMCPServicer(fastmcp_pb2_grpc.FastMCPServiceServicer):
    def __init__(self):
        self.clients = {}

    async def Connect(self, request, context):
        try:
            # Create transport based on type
            if request.type == "http":
                transport = HTTPTransport(request.endpoint)
            elif request.type == "sse":
                transport = SSETransport(request.endpoint)
            elif request.type == "stdio":
                transport = StdioTransport(
                    command=request.config["command"],
                    args=request.config.get("args", [])
                )

            # Create and connect client
            client = Client(transport)
            await client.__aenter__()

            # Store client
            self.clients[request.id] = client

            return fastmcp_pb2.ConnectResponse(success=True)

        except Exception as e:
            return fastmcp_pb2.ConnectResponse(
                success=False,
                error=str(e)
            )

    async def CallTool(self, request, context):
        try:
            client = self.clients.get(request.mcp_id)
            if not client:
                raise ValueError(f"Client {request.mcp_id} not found")

            # Parse arguments
            arguments = json.loads(request.arguments)

            # Call tool
            result = await client.call_tool(request.tool_name, arguments)

            # Return result as JSON
            return fastmcp_pb2.CallToolResponse(
                result=json.dumps(result)
            )

        except Exception as e:
            return fastmcp_pb2.CallToolResponse(error=str(e))

# Start server
async def serve():
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    fastmcp_pb2_grpc.add_FastMCPServiceServicer_to_server(
        FastMCPServicer(), server
    )
    server.add_insecure_port('[::]:50051')
    await server.start()
    await server.wait_for_termination()
```

**Go gRPC Client:**
```go
package mcp

import (
    "context"
    "encoding/json"
    "google.golang.org/grpc"
    pb "agentapi/lib/mcp/proto"
)

type GRPCMCPClient struct {
    conn   *grpc.ClientConn
    client pb.FastMCPServiceClient
}

func NewGRPCMCPClient(addr string) (*GRPCMCPClient, error) {
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        return nil, err
    }

    return &GRPCMCPClient{
        conn:   conn,
        client: pb.NewFastMCPServiceClient(conn),
    }, nil
}

func (c *GRPCMCPClient) ConnectMCP(ctx context.Context, config MCPConfig) error {
    req := &pb.ConnectRequest{
        Id:       config.ID,
        Name:     config.Name,
        Type:     config.Type,
        Endpoint: config.Endpoint,
        AuthType: config.AuthType,
        Config:   config.Config,
        Auth:     config.Auth,
    }

    resp, err := c.client.Connect(ctx, req)
    if err != nil {
        return err
    }

    if !resp.Success {
        return fmt.Errorf("connect failed: %s", resp.Error)
    }

    return nil
}

func (c *GRPCMCPClient) CallTool(
    ctx context.Context,
    mcpID, toolName string,
    arguments map[string]any,
) (map[string]any, error) {
    argsJSON, err := json.Marshal(arguments)
    if err != nil {
        return nil, err
    }

    req := &pb.CallToolRequest{
        McpId:     mcpID,
        ToolName:  toolName,
        Arguments: string(argsJSON),
    }

    resp, err := c.client.CallTool(ctx, req)
    if err != nil {
        return nil, err
    }

    if resp.Error != "" {
        return nil, fmt.Errorf(resp.Error)
    }

    var result map[string]any
    if err := json.Unmarshal([]byte(resp.Result), &result); err != nil {
        return nil, err
    }

    return result, nil
}
```

**3. HTTP REST API**

**Pros:**
- Simple to implement
- Easy debugging
- RESTful interface

**Cons:**
- JSON serialization overhead
- HTTP connection overhead
- Less type safety

**FastAPI Server:**
```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from fastmcp import Client

app = FastAPI()
clients = {}

class ConnectRequest(BaseModel):
    id: str
    name: str
    type: str
    endpoint: str
    auth_type: str
    config: dict
    auth: dict

@app.post("/mcp/connect")
async def connect(req: ConnectRequest):
    try:
        client = Client(req.endpoint, auth=req.auth.get("token"))
        await client.__aenter__()
        clients[req.id] = client
        return {"success": True}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/mcp/{mcp_id}/tools/{tool_name}")
async def call_tool(mcp_id: str, tool_name: str, arguments: dict):
    client = clients.get(mcp_id)
    if not client:
        raise HTTPException(status_code=404, detail="Client not found")

    try:
        result = await client.call_tool(tool_name, arguments)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

**Go HTTP Client:**
```go
func (c *HTTPMCPClient) CallTool(
    ctx context.Context,
    mcpID, toolName string,
    arguments map[string]any,
) (map[string]any, error) {
    url := fmt.Sprintf("%s/mcp/%s/tools/%s", c.baseURL, mcpID, toolName)

    body, err := json.Marshal(arguments)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }

    var result map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}
```

### 5.2 State Management Across Multiple Clients

#### Client Instance Management

**Per-User Client Instances:**
```go
type ClientManager struct {
    clients sync.Map  // map[string]*Client
    mu      sync.RWMutex
}

func (m *ClientManager) GetOrCreateClient(
    userID string,
    config MCPConfig,
) (*Client, error) {
    // Check if client exists
    if client, ok := m.clients.Load(userID); ok {
        return client.(*Client), nil
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    // Double-check after acquiring lock
    if client, ok := m.clients.Load(userID); ok {
        return client.(*Client), nil
    }

    // Create new client
    client, err := NewClient(config)
    if err != nil {
        return nil, err
    }

    // Connect
    if err := client.Connect(context.Background()); err != nil {
        return nil, err
    }

    m.clients.Store(userID, client)
    return client, nil
}

func (m *ClientManager) RemoveClient(userID string) error {
    if client, ok := m.clients.LoadAndDelete(userID); ok {
        return client.(*Client).Close()
    }
    return nil
}
```

**Session-Based State:**
```go
type SessionState struct {
    UserID      string
    MCPClients  map[string]*Client  // MCP ID -> Client
    CreatedAt   time.Time
    LastAccessed time.Time
    mutex       sync.RWMutex
}

type SessionManager struct {
    sessions sync.Map  // map[string]*SessionState
}

func (m *SessionManager) GetSession(sessionID string) (*SessionState, error) {
    if state, ok := m.sessions.Load(sessionID); ok {
        session := state.(*SessionState)
        session.LastAccessed = time.Now()
        return session, nil
    }
    return nil, errors.New("session not found")
}

func (m *SessionManager) CreateSession(userID string) (string, error) {
    sessionID := generateSessionID()

    session := &SessionState{
        UserID:       userID,
        MCPClients:   make(map[string]*Client),
        CreatedAt:    time.Now(),
        LastAccessed: time.Now(),
    }

    m.sessions.Store(sessionID, session)
    return sessionID, nil
}

// Cleanup stale sessions
func (m *SessionManager) CleanupStale(maxAge time.Duration) {
    now := time.Now()

    m.sessions.Range(func(key, value any) bool {
        session := value.(*SessionState)

        if now.Sub(session.LastAccessed) > maxAge {
            // Close all MCP clients
            for _, client := range session.MCPClients {
                client.Close()
            }

            // Remove session
            m.sessions.Delete(key)
        }

        return true
    })
}
```

#### Shared State Management

**Redis-Backed State:**
```go
import "github.com/go-redis/redis/v8"

type RedisStateManager struct {
    redis *redis.Client
}

func (m *RedisStateManager) StoreClientState(
    userID, mcpID string,
    state ClientState,
) error {
    key := fmt.Sprintf("mcp:state:%s:%s", userID, mcpID)

    data, err := json.Marshal(state)
    if err != nil {
        return err
    }

    return m.redis.Set(
        context.Background(),
        key,
        data,
        24*time.Hour,  // TTL
    ).Err()
}

func (m *RedisStateManager) GetClientState(
    userID, mcpID string,
) (ClientState, error) {
    key := fmt.Sprintf("mcp:state:%s:%s", userID, mcpID)

    data, err := m.redis.Get(context.Background(), key).Bytes()
    if err != nil {
        return ClientState{}, err
    }

    var state ClientState
    if err := json.Unmarshal(data, &state); err != nil {
        return ClientState{}, err
    }

    return state, nil
}
```

### 5.3 Multi-Tenant Isolation Requirements

#### Tenant-Scoped Clients

**Tenant Context:**
```go
type TenantContext struct {
    TenantID     string
    UserID       string
    Permissions  []string
    MCPConfigs   map[string]MCPConfig
}

type MultiTenantClientManager struct {
    clients sync.Map  // map[string]map[string]*Client (tenant -> mcp -> client)
}

func (m *MultiTenantClientManager) GetClient(
    ctx context.Context,
    mcpID string,
) (*Client, error) {
    tenant := ctx.Value("tenant").(*TenantContext)

    // Get tenant's clients
    tenantClients, ok := m.clients.Load(tenant.TenantID)
    if !ok {
        tenantClients = &sync.Map{}
        m.clients.Store(tenant.TenantID, tenantClients)
    }

    // Get or create MCP client
    if client, ok := tenantClients.(*sync.Map).Load(mcpID); ok {
        return client.(*Client), nil
    }

    // Create new client with tenant config
    config := tenant.MCPConfigs[mcpID]
    client, err := m.createTenantClient(tenant, config)
    if err != nil {
        return nil, err
    }

    tenantClients.(*sync.Map).Store(mcpID, client)
    return client, nil
}

func (m *MultiTenantClientManager) createTenantClient(
    tenant *TenantContext,
    config MCPConfig,
) (*Client, error) {
    // Add tenant-specific auth
    config.Auth["tenant_id"] = tenant.TenantID
    config.Auth["user_id"] = tenant.UserID

    // Create isolated client
    return NewClient(config)
}
```

#### Resource Isolation

**Memory Limits:**
```go
import "github.com/shirou/gopsutil/v3/process"

type ResourceMonitor struct {
    limits map[string]ResourceLimits
}

type ResourceLimits struct {
    MaxMemoryMB      int
    MaxGoroutines    int
    MaxConnections   int
}

func (m *ResourceMonitor) CheckLimits(tenantID string) error {
    limits := m.limits[tenantID]

    // Check memory usage
    proc, _ := process.NewProcess(int32(os.Getpid()))
    memInfo, _ := proc.MemoryInfo()

    if memInfo.RSS > uint64(limits.MaxMemoryMB)*1024*1024 {
        return errors.New("memory limit exceeded")
    }

    // Check goroutines
    if runtime.NumGoroutine() > limits.MaxGoroutines {
        return errors.New("goroutine limit exceeded")
    }

    return nil
}
```

**Rate Limiting per Tenant:**
```go
type TenantRateLimiter struct {
    limiters sync.Map  // map[string]*rate.Limiter
}

func (l *TenantRateLimiter) Allow(tenantID string) bool {
    limiter, ok := l.limiters.Load(tenantID)
    if !ok {
        // 100 requests per second per tenant
        newLimiter := rate.NewLimiter(100, 200)
        l.limiters.Store(tenantID, newLimiter)
        limiter = newLimiter
    }

    return limiter.(*rate.Limiter).Allow()
}
```

### 5.4 Credential Management Best Practices

#### Secure Storage

**GCP Secret Manager Integration:**
```go
import secretmanager "cloud.google.com/go/secretmanager/apiv1"

type SecretManager struct {
    client *secretmanager.Client
    project string
}

func (m *SecretManager) GetOAuthCredentials(
    ctx context.Context,
    provider string,
) (OAuthCredentials, error) {
    name := fmt.Sprintf(
        "projects/%s/secrets/oauth-%s-credentials/versions/latest",
        m.project,
        provider,
    )

    result, err := m.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
        Name: name,
    })
    if err != nil {
        return OAuthCredentials{}, err
    }

    var creds OAuthCredentials
    if err := json.Unmarshal(result.Payload.Data, &creds); err != nil {
        return OAuthCredentials{}, err
    }

    return creds, nil
}

func (m *SecretManager) StoreTokens(
    ctx context.Context,
    userID, provider string,
    tokens OAuthTokens,
) error {
    name := fmt.Sprintf("oauth-tokens-%s-%s", userID, provider)

    data, err := json.Marshal(tokens)
    if err != nil {
        return err
    }

    // Create or update secret
    _, err = m.client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
        Parent:   fmt.Sprintf("projects/%s", m.project),
        SecretId: name,
        Secret: &secretmanagerpb.Secret{
            Replication: &secretmanagerpb.Replication{
                Replication: &secretmanagerpb.Replication_Automatic_{
                    Automatic: &secretmanagerpb.Replication_Automatic{},
                },
            },
        },
    })

    // Add version
    _, err = m.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
        Parent: fmt.Sprintf("projects/%s/secrets/%s", m.project, name),
        Payload: &secretmanagerpb.SecretPayload{
            Data: data,
        },
    })

    return err
}
```

**Encrypted Database Storage:**
```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
)

type EncryptedCredentialStore struct {
    db     *sql.DB
    cipher cipher.AEAD
}

func NewEncryptedStore(db *sql.DB, key []byte) (*EncryptedCredentialStore, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    return &EncryptedCredentialStore{
        db:     db,
        cipher: aesgcm,
    }, nil
}

func (s *EncryptedCredentialStore) StoreToken(
    ctx context.Context,
    userID, provider, token string,
) error {
    // Encrypt token
    nonce := make([]byte, s.cipher.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return err
    }

    encrypted := s.cipher.Seal(nonce, nonce, []byte(token), nil)

    // Store in database
    _, err := s.db.ExecContext(
        ctx,
        "INSERT INTO oauth_tokens (user_id, provider, token, created_at) VALUES (?, ?, ?, ?)",
        userID, provider, encrypted, time.Now(),
    )

    return err
}

func (s *EncryptedCredentialStore) GetToken(
    ctx context.Context,
    userID, provider string,
) (string, error) {
    var encrypted []byte

    err := s.db.QueryRowContext(
        ctx,
        "SELECT token FROM oauth_tokens WHERE user_id = ? AND provider = ?",
        userID, provider,
    ).Scan(&encrypted)

    if err != nil {
        return "", err
    }

    // Decrypt token
    nonceSize := s.cipher.NonceSize()
    nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

    plaintext, err := s.cipher.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

#### Token Refresh Management

**Automatic Token Refresh:**
```go
type TokenRefresher struct {
    store     CredentialStore
    refresher map[string]func(context.Context, string) (OAuthTokens, error)
    ticker    *time.Ticker
}

func (r *TokenRefresher) Start(ctx context.Context) {
    r.ticker = time.NewTicker(5 * time.Minute)

    go func() {
        for {
            select {
            case <-r.ticker.C:
                r.refreshExpiredTokens(ctx)
            case <-ctx.Done():
                r.ticker.Stop()
                return
            }
        }
    }()
}

func (r *TokenRefresher) refreshExpiredTokens(ctx context.Context) {
    // Get all tokens expiring in next 10 minutes
    tokens, err := r.store.GetExpiringTokens(ctx, 10*time.Minute)
    if err != nil {
        log.Error("Failed to get expiring tokens:", err)
        return
    }

    for _, token := range tokens {
        refreshFunc := r.refresher[token.Provider]
        if refreshFunc == nil {
            continue
        }

        // Refresh token
        newTokens, err := refreshFunc(ctx, token.RefreshToken)
        if err != nil {
            log.Error("Failed to refresh token:", err)
            continue
        }

        // Store new tokens
        if err := r.store.UpdateTokens(ctx, token.UserID, token.Provider, newTokens); err != nil {
            log.Error("Failed to store refreshed tokens:", err)
        }
    }
}
```

---

## Summary

This technical specification provides comprehensive details for integrating FastMCP 2.0 into AgentAPI:

1. **Architecture**: Transport-based client system with automatic inference, reentrant context managers, and connection lifecycle management
2. **OAuth**: Complete OAuth 2.1 implementation with multi-provider support, PKCE, token storage, and automatic refresh
3. **Transports**: HTTP (StreamableHttp/SSE) and stdio transports with authentication and connection pooling
4. **Production**: Async patterns, connection pooling, rate limiting, structured logging, and error recovery
5. **Integration**: Multiple Go integration options (gRPC recommended), state management, multi-tenancy, and secure credential storage

All specifications are based on actual FastMCP 2.0 source code analysis and production deployment patterns.

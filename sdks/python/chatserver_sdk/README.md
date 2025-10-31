# ChatServer Python SDK

A Python client library for interacting with the [ChatServer API](https://github.com/coder/agentapi). This SDK provides an OpenAI-compatible interface for chat completions with multi-agent backend support (Claude, Droid, CCRouter, etc.).

## Installation

```bash
pip install chatserver-sdk
```

For development:
```bash
pip install chatserver-sdk[dev]
```

## Quick Start

```python
from chatserver_sdk import ChatServerClient, Message, MessageRole

# Create client with API key
client = ChatServerClient(api_key="your-api-key", base_url="http://localhost:3284")

# List available models
models = client.list_models()
print(f"Available models: {[model.id for model in models.data]}")

# Create chat completion
messages = [
    Message(role=MessageRole.SYSTEM, content="You are a helpful assistant."),
    Message(role=MessageRole.USER, content="Write a hello world function in Python.")
]

response = client.create_completion(
    model="claude-3-haiku",
    messages=messages
)

print(f"Assistant: {response.choices[0].message.content}")
print(f"Tokens used: {response.usage.total_tokens}")

client.close()
```

Or use the context manager:

```python
from chatserver_sdk import ChatServerClient

with ChatServerClient(api_key="your-api-key") as client:
    response = client.create_completion(
        model="claude-3-haiku",
        messages=[{"role": "user", "content": "Hello!"}]
    )
    print(response.choices[0].message.content)
    # Client automatically closed here
```

## Features

- ✅ OpenAI-compatible API
- ✅ Chat completions with streaming support
- ✅ Multi-agent backend support
- ✅ Model listing with provider information
- ✅ Platform administration endpoints
- ✅ Token usage tracking
- ✅ Session history APIs (list, detail) and resume metadata
- ✅ Error handling with specific exceptions
- ✅ Type hints with dataclasses
- ✅ Context manager support
- ✅ Audit logging capabilities

## API Reference

### Client Methods

#### `ChatServerClient(api_key=None, base_url="http://localhost:3284", timeout=30)`
Create a new client instance.

#### `create_completion(model, messages, temperature=0.7, max_tokens=4000, top_p=1.0, stream=False, user=None, system_prompt=None, *, session_id=None, metadata=None, organization_id=None, workflow=None, variables=None, allowed_tools=None, setting_sources=None, mcp_servers=None)`
Create a chat completion. Returns `ChatCompletionResponse` with `system_fingerprint` populated for the active session.

#### `list_models()`
List available models. Returns `ModelsResponse`.

#### `list_sessions(user_id, page=1, page_size=20)`
List chat sessions for a user. Returns `ChatSessionListResponse`.

#### `get_session(session_id, *, user_id)`
Fetch a chat session transcript (messages + metadata). Returns `ChatSessionDetailResponse`.

#### `get_platform_stats()`
Get platform statistics (requires platform admin). Returns `PlatformStats`.

#### `list_admins()`
List platform administrators (requires platform admin). Returns dict with 'admins' and 'count'.

#### `add_admin(workos_id, email, name="")`
Add platform administrator (requires platform admin). Returns `AdminResponse`.

#### `remove_admin(email)`
Remove platform administrator (requires platform admin). Returns `AdminResponse`.

#### `get_audit_log(limit=50, offset=0)`
Get audit log entries (requires platform admin). Returns `AuditLogResponse`.

### Data Models

#### `Message`
- `role`: MessageRole (SYSTEM, USER, ASSISTANT)
- `content`: str

#### `ChatCompletionRequest`
- `model`: str
- `messages`: List[Message]
- `temperature`: float (0-2)
- `max_tokens`: int
- `top_p`: float (0-1)
- `stream`: bool
- `user`: str
- `system_prompt`: str

#### `ChatCompletionResponse`
- `id`: str
- `object`: str
- `created`: int
- `model`: str
- `choices`: List[ChatCompletionChoice]
- `usage`: UsageInfo
- `system_fingerprint`: Optional[str]

#### `ChatSessionSummary`
- `id`: str
- `user_id`: str
- `organization_id`: Optional[str]
- `title`: Optional[str]
- `model`: Optional[str]
- `agent_type`: Optional[str]
- `created_at`: datetime
- `updated_at`: datetime
- `last_message_at`: Optional[datetime]
- `message_count`: int
- `tokens_in`, `tokens_out`, `tokens_total`: int
- `metadata`: dict[str, Any]
- `archived`: bool

#### `ChatSessionListResponse`
- `sessions`: list[ChatSessionSummary]
- `total`: int
- `page`: int
- `page_size`: int
- `has_more`: bool

#### `ChatSessionDetailResponse`
- `session`: ChatSessionSummary
- `messages`: list[ChatMessageRecord]

## Examples

### Streaming Completions

```python
from chatserver_sdk import ChatServerClient

client = ChatServerClient(api_key="your-api-key")

messages = [{"role": "user", "content": "Tell me a story"}]

print("Streaming response:")
stream = client.create_completion(
    model="claude-3-haiku",
    messages=messages,
    stream=True
)

for chunk in stream:
    print(chunk, end='', flush=True)

print("\nSession fingerprint:", stream.metadata.get("system_fingerprint"))

client.close()
```

### Multi-turn Conversations

```python
from chatserver_sdk import ChatServerClient, Message, MessageRole

client = ChatServerClient(api_key="your-api-key")

conversation = [
    Message(role=MessageRole.USER, content="What is recursion?")
]

# First response
response1 = client.create_completion(
    model="claude-3-haiku",
    messages=conversation
)

if response1.choices:
    # Add assistant response to conversation
    conversation.append(response1.choices[0].message)
    
    # Follow-up question
    conversation.append(
        Message(role=MessageRole.USER, content="Can you give me an example?")
    )
    
    # Get follow-up response
    response2 = client.create_completion(
        model="claude-3-haiku",
        messages=conversation
    )
    
    print(response2.choices[0].message.content)

client.close()

### Session History and Resume

```python
from chatserver_sdk import ChatServerClient, Message, MessageRole

client = ChatServerClient(api_key="your-api-key")
user_id = "00000000-0000-0000-0000-000000000001"  # Supabase profile ID

try:
    # Start a new session; passing user ensures history is recorded
    first = client.create_completion(
        model="claude-3-haiku",
        messages=[Message(role=MessageRole.USER, content="Draft a project update email.")],
        user=user_id,
    )

    session_id = first.system_fingerprint
    if not session_id:
        raise RuntimeError("ChatServer did not return a session fingerprint")

    # Continue in streaming mode using the same session id
    stream = client.create_completion(
        model="claude-3-haiku",
        messages=[{"role": "user", "content": "Add a bullet list of risks."}],
        stream=True,
        user=user_id,
        session_id=session_id,
    )

    for delta in stream:
        print(delta, end="", flush=True)

    print("\nStream recorded under:", stream.metadata.get("system_fingerprint"))

    # Query stored history
    sessions = client.list_sessions(user_id)
    latest = sessions.sessions[0] if sessions.sessions else None
    if latest:
        detail = client.get_session(latest.id, user_id=user_id)
        print(f"Loaded {len(detail.messages)} stored messages")
finally:
    client.close()
```
```

### Platform Administration

```python
from chatserver_sdk import ChatServerClient

client = ChatServerClient(api_key="admin-api-key")

# Get platform statistics
stats = client.get_platform_stats()
print(f"Total users: {stats.total_users}")
print(f"Active users: {stats.active_users}")
print(f"Total requests: {stats.total_requests}")

# List admins
admins = client.list_admins()
print(f"Platform admins: {admins['count']}")

# Get audit log
audit_log = client.get_audit_log(limit=20)
for entry in audit_log.entries:
    print(f"[{entry.timestamp}] {entry.action}: {entry.resource}")

client.close()
```

### Error Handling

```python
from chatserver_sdk import ChatServerClient
from chatserver_sdk.exceptions import BadRequestError, UnauthorizedError

client = ChatServerClient(api_key="your-api-key")

try:
    response = client.create_completion(
        model="",  # Invalid model
        messages=[]
    )
except BadRequestError as e:
    print(f"Bad request: {e.message}")
    print(f"Status code: {e.status_code}")
except UnauthorizedError as e:
    print(f"Unauthorized: {e.message}")
except Exception as e:
    print(f"Other error: {e}")

client.close()
```

## Error Types

The SDK provides specific exceptions for different HTTP errors:

- `BadRequestError` (400) - Invalid request parameters
- `UnauthorizedError` (401) - Missing or invalid API key
- `ForbiddenError` (403) - Insufficient permissions
- `NotFoundError` (404) - Resource not found
- `InternalServerError` (500) - Server error
- `ChatServerError` - Base exception class

## Message Formats

You can pass messages in several formats:

```python
# Using Message objects (recommended)
messages = [
    Message(role=MessageRole.SYSTEM, content="You are helpful."),
    Message(role=MessageRole.USER, content="Hello!")
]

# Using dictionaries
messages = [
    {"role": "system", "content": "You are helpful."},
    {"role": "user", "content": "Hello!"}
]

# Simple strings (converted to user messages)
messages = [
    "Hello!",
    "How are you?"
]
```

## Development

### Setup development environment

```bash
# Clone repository
git clone https://github.com/coder/agentapi.git
cd agentapi/sdks/python/chatserver_sdk

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Run linting
black chatserver_sdk/
flake8 chatserver_sdk/
mypy chatserver_sdk/
```

## License

MIT License - see [LICENSE](../../../LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## Support

- Issues: [GitHub Issues](https://github.com/coder/agentapi/issues)
- Documentation: [ChatServer Documentation](https://github.com/coder/agentapi#readme)

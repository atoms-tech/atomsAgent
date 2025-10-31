# AgentAPI Python SDK

A Python client library for interacting with the [AgentAPI](https://github.com/coder/agentapi) server. This SDK provides a clean interface to communicate with coding agents (Claude Code, Droid, CCRouter) via HTTP API.

## Installation

```bash
pip install agentapi-sdk
```

For development:
```bash
pip install agentapi-sdk[dev]
```

## Quick Start

```python
from agentapi_sdk import AgentAPIClient, MessageType

# Create client
with AgentAPIClient("http://localhost:3284") as client:
    # Check agent status
    status = client.get_status()
    print(f"Agent Status: {status.status}")
    
    # Send a message
    response = client.send_message("Hello, agent!")
    print(f"Message sent: {response.ok}")
    
    # Get conversation history
    messages = client.get_messages()
    for msg in messages.messages:
        print(f"[{msg.role}] {msg.content}")
```

## Features

- ✅ Send messages to agents
- ✅ Get conversation history
- ✅ Check agent status
- ✅ Upload files
- ✅ Subscribe to real-time events (Server-Sent Events)
- ✅ Send raw keystrokes (for escape sequences)
- ✅ Type hints and dataclasses for better IDE support
- ✅ Comprehensive error handling

## API Reference

### Client Methods

#### `get_status() -> StatusResponse`
Get the current status of the agent.

#### `get_messages() -> MessagesResponse`
Get the conversation history with the agent.

#### `send_message(content: str, message_type: MessageType = MessageType.USER) -> MessageResponse`
Send a message to the agent.

#### `upload_file(file_obj: bytes | str, filename: str) -> UploadResponse`
Upload a file to the server.

#### `subscribe_events() -> Iterator[Dict[str, Any]]`
Subscribe to real-time events from the server using Server-Sent Events.

### Message Types

- `MessageType.USER`: Regular user message (saved in conversation history)
- `MessageType.RAW`: Raw keystrokes (not saved, useful for escape sequences)

### Data Models

#### `StatusResponse`
- `status`: AgentStatus (RUNNING or STABLE)
- `agent_type`: str (claude, droid, ccrouter)

#### `Message`
- `id`: int
- `content`: str
- `role`: ConversationRole (AGENT or USER)
- `time`: datetime

## Examples

### Basic Usage

```python
from agentapi_sdk import AgentAPIClient

client = AgentAPIClient("http://localhost:3284")

# Check if agent is ready
status = client.get_status()
if status.status == "running":
    print("Agent is busy, wait...")
    # Wait for stable state
    import time
    while status.status == "running":
        time.sleep(1)
        status = client.get_status()

# Send message when agent is ready
response = client.send_message("Write a hello world function")
print(f"Success: {response.ok}")

client.close()
```

### Event Streaming

```python
from agentapi_sdk import AgentAPIClient

client = AgentAPIClient("http://localhost:3284")

try:
    for event in client.subscribe_events():
        if event['event'] == 'message_update':
            print(f"New message: {event['data']['message'][:100]}...")
        elif event['event'] == 'status_change':
            print(f"Status changed to: {event['data']['status']}")
except KeyboardInterrupt:
    print("Stopping...")
finally:
    client.close()
```

### File Upload

```python
from agentapi_sdk import AgentAPIClient

client = AgentAPIClient("http://localhost:3284")

# Upload a file
with open("my_file.txt", "rb") as f:
    response = client.upload_file(f.read(), "my_file.txt")
    print(f"Uploaded to: {response.file_path}")

client.close()
```

### Raw Keystrokes

```python
from agentapi_sdk import AgentAPIClient, MessageType

client = AgentAPIClient("http://localhost:3284")

# Send Ctrl+C to interrupt
response = client.send_message("\x03", MessageType.RAW)
print(f"Interrupt sent: {response.ok}")

client.close()
```

## Error Handling

The SDK provides specific exception types for different HTTP errors:

```python
from agentapi_sdk import AgentAPIClient
from agentapi_sdk.exceptions import BadRequestError, NotFoundError

client = AgentAPIClient("http://localhost:3284")

try:
    response = client.send_message("")
except BadRequestError as e:
    print(f"Bad request: {e.message}")
except NotFoundError as e:
    print(f"Not found: {e.message}")
except Exception as e:
    print(f"Error: {e}")
```

Exception types:
- `BadRequestError` (400)
- `UnauthorizedError` (401)
- `ForbiddenError` (403)
- `NotFoundError` (404)
- `InternalServerError` (500)
- `AgentAPIError` (base class)

## Development

### Setup development environment

```bash
# Clone repository
git clone https://github.com/coder/agentapi.git
cd agentapi/sdks/python

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Run linting
black agentapi_sdk/
flake8 agentapi_sdk/
mypy agentapi_sdk/
```

## License

MIT License - see [LICENSE](../../LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## Support

- Issues: [GitHub Issues](https://github.com/coder/agentapi/issues)
- Documentation: [AgentAPI Documentation](https://github.com/coder/agentapi#readme)

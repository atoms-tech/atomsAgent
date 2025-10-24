# FastMCP Service

Production-ready FastAPI service for managing MCP (Model Context Protocol) clients using FastMCP 2.0.

## Features

- **Multiple Transport Types**: HTTP, SSE (Server-Sent Events), and STDIO
- **Flexible Authentication**: Bearer tokens, OAuth 2.0, or no authentication
- **Thread-Safe Client Management**: Async-safe client storage with locks
- **Comprehensive Error Handling**: Proper error responses and logging
- **Request/Response Validation**: Pydantic models for type safety
- **Health Monitoring**: Health check endpoint with metrics
- **CORS Support**: Configurable cross-origin resource sharing
- **Request Logging**: Detailed logging of all requests and responses
- **Timeout Handling**: Configurable timeouts for connections and operations
- **Resource Cleanup**: Proper disconnection on shutdown

## Installation

### Prerequisites

- Python 3.8 or higher
- pip (Python package manager)

### Install Dependencies

```bash
pip install -r requirements.txt
```

The service requires:
- `fastmcp>=2.0.0` - FastMCP client library
- `fastapi>=0.104.0` - FastAPI web framework
- `uvicorn[standard]>=0.24.0` - ASGI server
- `httpx>=0.24.0` - HTTP client
- `pydantic>=2.0.0` - Data validation

## Running the Service

### Basic Usage

```bash
python lib/mcp/fastmcp_service.py
```

### With Custom Port

```bash
python lib/mcp/fastmcp_service.py --port 8080
```

### With Auto-Reload (Development)

```bash
python lib/mcp/fastmcp_service.py --reload
```

### With Debug Mode

```bash
python lib/mcp/fastmcp_service.py --debug
```

### All Options

```bash
python lib/mcp/fastmcp_service.py \
  --host 0.0.0.0 \
  --port 8080 \
  --reload \
  --debug
```

## API Documentation

Once running, visit:
- **Interactive API Docs**: http://localhost:8080/docs
- **Alternative Docs**: http://localhost:8080/redoc
- **OpenAPI Schema**: http://localhost:8080/openapi.json

## API Endpoints

### Health Check

**GET** `/health`

Returns the service health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T12:00:00",
  "active_clients": 5,
  "version": "2.0.0"
}
```

### Connect to MCP Server

**POST** `/mcp/connect`

Creates a new MCP client connection.

**Request Body (HTTP Transport):**
```json
{
  "transport": "http",
  "mcp_url": "http://localhost:3000/mcp",
  "auth_type": "bearer",
  "bearer_token": "your-token-here",
  "name": "my-mcp-client",
  "timeout": 30
}
```

**Request Body (SSE Transport):**
```json
{
  "transport": "sse",
  "mcp_url": "http://localhost:3000/events",
  "auth_type": "none"
}
```

**Request Body (STDIO Transport):**
```json
{
  "transport": "stdio",
  "command": ["node", "path/to/mcp-server.js"],
  "auth_type": "none"
}
```

**Request Body (OAuth Authentication):**
```json
{
  "transport": "http",
  "mcp_url": "http://localhost:3000/mcp",
  "auth_type": "oauth",
  "oauth_client_id": "your-client-id",
  "oauth_client_secret": "your-client-secret",
  "oauth_auth_url": "https://provider.com/oauth/authorize",
  "oauth_token_url": "https://provider.com/oauth/token",
  "oauth_redirect_uri": "http://localhost:8080/callback",
  "oauth_scopes": ["read", "write"]
}
```

**Response:**
```json
{
  "status": "connected",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "tools": [...],
  "resources": [...],
  "prompts": [...],
  "message": "Successfully connected to MCP server"
}
```

### List Tools

**GET** `/mcp/list_tools?client_id={client_id}`

Lists all available tools from the MCP server.

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "tools": [
    {
      "name": "search",
      "description": "Search for information",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {"type": "string"}
        }
      }
    }
  ],
  "count": 1
}
```

### Call Tool

**POST** `/mcp/call_tool`

Executes a tool on the MCP server.

**Request Body:**
```json
{
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "tool_name": "search",
  "arguments": {
    "query": "python async programming"
  },
  "timeout": 60
}
```

**Response:**
```json
{
  "status": "success",
  "result": {
    "results": [...]
  },
  "error": null,
  "execution_time": 1.234
}
```

### List Resources

**GET** `/mcp/list_resources?client_id={client_id}`

Lists all available resources from the MCP server.

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "resources": [
    {
      "uri": "file:///path/to/resource",
      "name": "My Resource",
      "description": "A sample resource",
      "mimeType": "text/plain"
    }
  ],
  "count": 1
}
```

### Read Resource

**POST** `/mcp/read_resource`

Reads the contents of a resource.

**Request Body:**
```json
{
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "uri": "file:///path/to/resource"
}
```

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "uri": "file:///path/to/resource",
  "content": {
    "text": "Resource content here"
  }
}
```

### List Prompts

**GET** `/mcp/list_prompts?client_id={client_id}`

Lists all available prompts from the MCP server.

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "prompts": [
    {
      "name": "code_review",
      "description": "Review code for issues",
      "arguments": [...]
    }
  ],
  "count": 1
}
```

### Get Prompt

**POST** `/mcp/get_prompt`

Retrieves a prompt with arguments.

**Request Body:**
```json
{
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "prompt_name": "code_review",
  "arguments": {
    "language": "python",
    "file": "app.py"
  }
}
```

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "prompt_name": "code_review",
  "content": {
    "messages": [...]
  }
}
```

### Disconnect

**POST** `/mcp/disconnect`

Disconnects from an MCP server and cleans up resources.

**Request Body:**
```json
{
  "client_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**
```json
{
  "status": "disconnected",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Successfully disconnected from MCP server"
}
```

### List Clients

**GET** `/mcp/clients`

Lists all active MCP clients.

**Response:**
```json
{
  "status": "success",
  "clients": [
    {
      "client_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-15T12:00:00",
      "transport": "http",
      "auth_type": "bearer",
      "name": "my-mcp-client",
      "mcp_url": "http://localhost:3000/mcp",
      "last_activity": "2024-01-15T12:05:00"
    }
  ],
  "count": 1
}
```

### Get Client Info

**GET** `/mcp/client/{client_id}`

Gets detailed information about a specific client.

**Response:**
```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "metadata": {
    "created_at": "2024-01-15T12:00:00",
    "transport": "http",
    "auth_type": "bearer",
    "name": "my-mcp-client",
    "mcp_url": "http://localhost:3000/mcp",
    "last_activity": "2024-01-15T12:05:00"
  }
}
```

## Usage Examples

### Python Client Example

```python
import httpx
import asyncio

async def main():
    async with httpx.AsyncClient() as client:
        # Connect to MCP server
        connect_response = await client.post(
            "http://localhost:8080/mcp/connect",
            json={
                "transport": "http",
                "mcp_url": "http://localhost:3000/mcp",
                "auth_type": "bearer",
                "bearer_token": "my-token"
            }
        )
        client_id = connect_response.json()["client_id"]

        # List tools
        tools_response = await client.get(
            f"http://localhost:8080/mcp/list_tools?client_id={client_id}"
        )
        tools = tools_response.json()["tools"]

        # Call a tool
        result_response = await client.post(
            "http://localhost:8080/mcp/call_tool",
            json={
                "client_id": client_id,
                "tool_name": "search",
                "arguments": {"query": "test"}
            }
        )
        result = result_response.json()

        # Disconnect
        await client.post(
            "http://localhost:8080/mcp/disconnect",
            json={"client_id": client_id}
        )

if __name__ == "__main__":
    asyncio.run(main())
```

### cURL Example

```bash
# Connect
RESPONSE=$(curl -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "http",
    "mcp_url": "http://localhost:3000/mcp",
    "auth_type": "none"
  }')

CLIENT_ID=$(echo $RESPONSE | jq -r '.client_id')

# List tools
curl http://localhost:8080/mcp/list_tools?client_id=$CLIENT_ID

# Call tool
curl -X POST http://localhost:8080/mcp/call_tool \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"tool_name\": \"search\",
    \"arguments\": {\"query\": \"test\"}
  }"

# Disconnect
curl -X POST http://localhost:8080/mcp/disconnect \
  -H "Content-Type: application/json" \
  -d "{\"client_id\": \"$CLIENT_ID\"}"
```

### Go Integration Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type ConnectRequest struct {
    Transport string `json:"transport"`
    McpURL    string `json:"mcp_url"`
    AuthType  string `json:"auth_type"`
}

type ConnectResponse struct {
    Status   string `json:"status"`
    ClientID string `json:"client_id"`
}

func main() {
    // Connect to MCP
    connectReq := ConnectRequest{
        Transport: "http",
        McpURL:    "http://localhost:3000/mcp",
        AuthType:  "none",
    }

    body, _ := json.Marshal(connectReq)
    resp, _ := http.Post(
        "http://localhost:8080/mcp/connect",
        "application/json",
        bytes.NewBuffer(body),
    )

    var connectResp ConnectResponse
    json.NewDecoder(resp.Body).Decode(&connectResp)

    fmt.Printf("Connected with client ID: %s\n", connectResp.ClientID)
}
```

## Error Handling

The service uses standard HTTP status codes:

- `200 OK` - Successful operation
- `201 Created` - Resource created (new connection)
- `400 Bad Request` - Invalid request data
- `404 Not Found` - Client not found
- `408 Request Timeout` - Operation timeout
- `422 Unprocessable Entity` - Validation error
- `500 Internal Server Error` - Server error

Error responses follow this format:

```json
{
  "detail": "Error message describing what went wrong"
}
```

## Logging

The service logs to both stdout and a file (`fastmcp_service.log`).

Log format:
```
2024-01-15 12:00:00 - fastmcp_service - INFO - Message here
```

## Configuration

### Environment Variables

- `HOST` - Host to bind to (default: 0.0.0.0)
- `PORT` - Port to bind to (default: 8080)

### CORS Configuration

By default, CORS is enabled for all origins. In production, update the CORS middleware configuration:

```python
app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://yourapp.com"],  # Specific origins
    allow_credentials=True,
    allow_methods=["GET", "POST"],
    allow_headers=["*"],
)
```

## Testing

Run the test suite:

```bash
# Install test dependencies
pip install pytest pytest-asyncio

# Run tests
pytest lib/mcp/test_fastmcp_service.py -v
```

## Production Deployment

### Using Gunicorn

```bash
gunicorn lib.mcp.fastmcp_service:app \
  --workers 4 \
  --worker-class uvicorn.workers.UvicornWorker \
  --bind 0.0.0.0:8080
```

### Using Docker

Create a `Dockerfile`:

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY lib/mcp/fastmcp_service.py lib/mcp/

EXPOSE 8080

CMD ["python", "lib/mcp/fastmcp_service.py", "--host", "0.0.0.0", "--port", "8080"]
```

Build and run:

```bash
docker build -t fastmcp-service .
docker run -p 8080:8080 fastmcp-service
```

### Using Systemd

Create `/etc/systemd/system/fastmcp-service.service`:

```ini
[Unit]
Description=FastMCP Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/agentapi
ExecStart=/usr/bin/python3 lib/mcp/fastmcp_service.py --port 8080
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable fastmcp-service
sudo systemctl start fastmcp-service
```

## Performance Considerations

- **Connection Pooling**: Each client maintains its own connection
- **Async Operations**: All I/O operations are asynchronous
- **Thread Safety**: Client storage uses async locks
- **Resource Cleanup**: Clients are properly disconnected on shutdown
- **Timeouts**: Configurable timeouts prevent hanging requests

## Security Considerations

1. **CORS**: Configure allowed origins for production
2. **Authentication**: Use bearer tokens or OAuth for MCP servers
3. **HTTPS**: Use HTTPS in production (reverse proxy)
4. **Rate Limiting**: Consider adding rate limiting middleware
5. **Input Validation**: All inputs are validated with Pydantic

## Troubleshooting

### Service won't start

- Check if port is already in use: `lsof -i :8080`
- Verify Python version: `python3 --version` (must be 3.8+)
- Check dependencies: `pip list | grep fastmcp`

### Connection failures

- Verify MCP server is running
- Check network connectivity
- Verify authentication credentials
- Review service logs

### Timeout errors

- Increase timeout values in requests
- Check MCP server performance
- Verify network latency

## Contributing

Contributions are welcome! Please ensure:

1. Code follows PEP 8 style guide
2. Tests pass: `pytest -v`
3. Documentation is updated
4. Type hints are included

## License

This service is part of the AgentAPI project.

## Support

For issues and questions:
- Check the logs: `tail -f fastmcp_service.log`
- Review the API documentation: http://localhost:8080/docs
- Check FastMCP documentation: https://github.com/modelcontextprotocol/fastmcp

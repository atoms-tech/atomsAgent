# FastMCP Service - Quickstart Guide

Get up and running with FastMCP Service in 5 minutes.

## Prerequisites

- Python 3.8 or higher
- pip (Python package manager)
- An MCP server to connect to (optional for testing)

## Installation

### 1. Install Dependencies

```bash
cd lib/mcp
pip install -r requirements.txt
```

Or use the Makefile:

```bash
make install
```

### 2. Start the Service

#### Option A: Direct Python

```bash
python fastmcp_service.py
```

#### Option B: Using Makefile

Development mode (auto-reload):
```bash
make dev
```

Production mode:
```bash
make run
```

#### Option C: Using Docker

```bash
make docker-build
make docker-run
```

### 3. Verify Service is Running

Open your browser to:
- Health Check: http://localhost:8080/health
- API Documentation: http://localhost:8080/docs

Or use curl:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T12:00:00",
  "active_clients": 0,
  "version": "2.0.0"
}
```

## Basic Usage

### Example 1: Connect to an MCP Server

```bash
# Save the client_id from the response
CLIENT_ID=$(curl -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "http",
    "mcp_url": "http://localhost:3000/mcp",
    "auth_type": "none"
  }' | jq -r '.client_id')

echo "Connected with client ID: $CLIENT_ID"
```

### Example 2: List Available Tools

```bash
curl "http://localhost:8080/mcp/list_tools?client_id=$CLIENT_ID" | jq
```

### Example 3: Call a Tool

```bash
curl -X POST http://localhost:8080/mcp/call_tool \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"tool_name\": \"search\",
    \"arguments\": {
      \"query\": \"python async programming\"
    }
  }" | jq
```

### Example 4: Disconnect

```bash
curl -X POST http://localhost:8080/mcp/disconnect \
  -H "Content-Type: application/json" \
  -d "{\"client_id\": \"$CLIENT_ID\"}" | jq
```

## Python Client Example

```python
import asyncio
import httpx

async def main():
    async with httpx.AsyncClient() as client:
        # Connect
        response = await client.post(
            "http://localhost:8080/mcp/connect",
            json={
                "transport": "http",
                "mcp_url": "http://localhost:3000/mcp",
                "auth_type": "none"
            }
        )
        client_id = response.json()["client_id"]
        print(f"Connected: {client_id}")

        # List tools
        response = await client.get(
            f"http://localhost:8080/mcp/list_tools?client_id={client_id}"
        )
        print("Tools:", response.json())

        # Call a tool
        response = await client.post(
            "http://localhost:8080/mcp/call_tool",
            json={
                "client_id": client_id,
                "tool_name": "search",
                "arguments": {"query": "test"}
            }
        )
        print("Result:", response.json())

        # Disconnect
        await client.post(
            "http://localhost:8080/mcp/disconnect",
            json={"client_id": client_id}
        )

if __name__ == "__main__":
    asyncio.run(main())
```

## Using the Example Script

Run the provided example script:

```bash
python example_usage.py
```

Edit the script to uncomment specific examples:

```python
# In example_usage.py
async def main():
    # Uncomment the examples you want to run
    await example_http_connection()
    await example_bearer_auth()
    # await example_complete_workflow()
```

## Common Operations

### Check Service Health

```bash
curl http://localhost:8080/health
```

### List Active Clients

```bash
curl http://localhost:8080/mcp/clients | jq
```

### Get Client Info

```bash
curl http://localhost:8080/mcp/client/$CLIENT_ID | jq
```

### View Logs

If running directly:
```bash
tail -f fastmcp_service.log
```

If running with Docker:
```bash
make docker-logs
```

If running with systemd:
```bash
make logs-systemd
```

## Transport Types

### HTTP Transport

```json
{
  "transport": "http",
  "mcp_url": "http://localhost:3000/mcp",
  "auth_type": "none"
}
```

### SSE (Server-Sent Events) Transport

```json
{
  "transport": "sse",
  "mcp_url": "http://localhost:3000/events",
  "auth_type": "none"
}
```

### STDIO Transport

```json
{
  "transport": "stdio",
  "command": ["node", "server.js"],
  "auth_type": "none"
}
```

## Authentication Types

### No Authentication

```json
{
  "auth_type": "none"
}
```

### Bearer Token

```json
{
  "auth_type": "bearer",
  "bearer_token": "your-secret-token"
}
```

### OAuth 2.0

```json
{
  "auth_type": "oauth",
  "oauth_client_id": "your-client-id",
  "oauth_client_secret": "your-client-secret",
  "oauth_auth_url": "https://provider.com/oauth/authorize",
  "oauth_token_url": "https://provider.com/oauth/token",
  "oauth_redirect_uri": "http://localhost:8080/callback",
  "oauth_scopes": ["read", "write"]
}
```

## Testing

Run the test suite:

```bash
make test
```

Or directly with pytest:

```bash
pytest test_fastmcp_service.py -v
```

## Troubleshooting

### Service won't start

1. Check if port 8080 is available:
   ```bash
   lsof -i :8080
   ```

2. Try a different port:
   ```bash
   python fastmcp_service.py --port 8081
   ```

### Cannot connect to MCP server

1. Verify the MCP server is running
2. Check the MCP URL is correct
3. Test with curl:
   ```bash
   curl http://localhost:3000/mcp
   ```

### Import errors

Install dependencies:
```bash
pip install -r requirements.txt
```

### Connection timeout

Increase timeout in the connect request:
```json
{
  "timeout": 60
}
```

## Next Steps

- Read the full documentation: [FASTMCP_SERVICE_README.md](./FASTMCP_SERVICE_README.md)
- Check out the example script: [example_usage.py](./example_usage.py)
- View the API documentation: http://localhost:8080/docs
- Run the tests: `make test`

## Production Deployment

### Using Systemd

```bash
make install-systemd
make start-systemd
```

### Using Docker

```bash
make docker-build
make docker-run
```

### Using Gunicorn

```bash
make run-gunicorn
```

## Need Help?

- View logs: `tail -f fastmcp_service.log`
- Check health: `curl http://localhost:8080/health`
- View active clients: `curl http://localhost:8080/mcp/clients`
- API docs: http://localhost:8080/docs

## Configuration

Environment variables:
- `HOST` - Bind host (default: 0.0.0.0)
- `PORT` - Bind port (default: 8080)

Command line options:
```bash
python fastmcp_service.py --help
```

## Stop the Service

### Direct Python
Press `Ctrl+C`

### Docker
```bash
make docker-stop
```

### Systemd
```bash
make stop-systemd
```

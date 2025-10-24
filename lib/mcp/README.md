# FastMCP Service

Production-ready FastAPI service for managing MCP (Model Context Protocol) clients using FastMCP 2.0.

[![Python Version](https://img.shields.io/badge/python-3.8%2B-blue)](https://www.python.org/downloads/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.104%2B-009688)](https://fastapi.tiangolo.com/)
[![FastMCP](https://img.shields.io/badge/FastMCP-2.0%2B-green)](https://github.com/modelcontextprotocol/fastmcp)

## Quick Start

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Start the service
python fastmcp_service.py

# 3. Check health
curl http://localhost:8080/health

# 4. View API docs
open http://localhost:8080/docs
```

## Features

- **Multiple Transport Types**: HTTP, SSE, STDIO
- **Flexible Authentication**: Bearer tokens, OAuth 2.0
- **Thread-Safe**: Async client management with locks
- **Production-Ready**: Comprehensive error handling and logging
- **Well-Tested**: Full test suite with pytest
- **Well-Documented**: Complete API reference and guides

## Documentation

- **[QUICKSTART.md](./QUICKSTART.md)** - Get started in 5 minutes
- **[FASTMCP_SERVICE_README.md](./FASTMCP_SERVICE_README.md)** - Complete documentation
- **[API_REFERENCE.md](./API_REFERENCE.md)** - Full API reference
- **[PROJECT_SUMMARY.md](./PROJECT_SUMMARY.md)** - Project overview

## Project Structure

```
lib/mcp/
├── fastmcp_service.py           # Main FastAPI service (847 lines)
├── test_fastmcp_service.py      # Comprehensive test suite
├── example_usage.py             # Example Python client
├── FASTMCP_SERVICE_README.md    # Complete documentation
├── API_REFERENCE.md             # Full API reference
├── QUICKSTART.md                # Quick start guide
├── PROJECT_SUMMARY.md           # Project overview
├── Dockerfile                   # Docker configuration
├── docker-compose.yml           # Docker Compose setup
├── Makefile                     # Build automation
├── start.sh                     # Startup script
├── verify_setup.sh              # Setup verification
└── fastmcp-service.service      # Systemd service file
```

**Total:** 5,218 lines of code, 292KB

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/mcp/connect` | Connect to MCP server |
| POST | `/mcp/call_tool` | Execute a tool |
| GET | `/mcp/list_tools` | List available tools |
| GET | `/mcp/list_resources` | List available resources |
| POST | `/mcp/read_resource` | Read a resource |
| GET | `/mcp/list_prompts` | List available prompts |
| POST | `/mcp/get_prompt` | Get a prompt |
| POST | `/mcp/disconnect` | Disconnect from server |
| GET | `/mcp/clients` | List active clients |
| GET | `/mcp/client/{id}` | Get client info |

## Usage Example

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

        # Call a tool
        response = await client.post(
            "http://localhost:8080/mcp/call_tool",
            json={
                "client_id": client_id,
                "tool_name": "search",
                "arguments": {"query": "test"}
            }
        )
        print(response.json())

        # Disconnect
        await client.post(
            "http://localhost:8080/mcp/disconnect",
            json={"client_id": client_id}
        )

asyncio.run(main())
```

## Installation

### Prerequisites

- Python 3.8 or higher
- pip (Python package manager)

### Install Dependencies

```bash
pip install -r requirements.txt
```

Or with the Makefile:

```bash
make install
```

## Running the Service

### Development Mode

```bash
make dev
# or
python fastmcp_service.py --reload --debug
```

### Production Mode

```bash
make run
# or
python fastmcp_service.py
```

### With Docker

```bash
make docker-build
make docker-run
```

### With Systemd

```bash
make install-systemd
make start-systemd
```

## Testing

```bash
make test
# or
pytest test_fastmcp_service.py -v
```

## Makefile Commands

- `make install` - Install dependencies
- `make test` - Run tests
- `make run` - Production mode
- `make dev` - Development mode
- `make docker-build` - Build Docker image
- `make docker-run` - Run in Docker
- `make clean` - Cleanup
- `make lint` - Code linting
- `make format` - Code formatting

## Configuration

### Environment Variables

- `HOST` - Bind host (default: 0.0.0.0)
- `PORT` - Bind port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)
- `WORKERS` - Gunicorn workers (default: 4)

### Command Line Options

```bash
python fastmcp_service.py --help
```

Options:
- `--host HOST` - Host to bind to
- `--port PORT` - Port to bind to
- `--reload` - Enable auto-reload
- `--debug` - Enable debug mode

## Transport Types

### HTTP Transport
```json
{
  "transport": "http",
  "mcp_url": "http://localhost:3000/mcp"
}
```

### SSE Transport
```json
{
  "transport": "sse",
  "mcp_url": "http://localhost:3000/events"
}
```

### STDIO Transport
```json
{
  "transport": "stdio",
  "command": ["node", "server.js"]
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
  "bearer_token": "your-token"
}
```

### OAuth 2.0
```json
{
  "auth_type": "oauth",
  "oauth_client_id": "your-client-id",
  "oauth_client_secret": "your-secret",
  "oauth_auth_url": "https://provider.com/oauth/authorize",
  "oauth_token_url": "https://provider.com/oauth/token"
}
```

## Error Handling

The service uses standard HTTP status codes:

- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `408 Request Timeout` - Timeout
- `422 Unprocessable Entity` - Validation error
- `500 Internal Server Error` - Server error

## Logging

Logs are written to:
- Console (stdout)
- File (`fastmcp_service.log`)

View logs:
```bash
tail -f fastmcp_service.log
```

## Health Check

Check service health:
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T12:00:00",
  "active_clients": 0,
  "version": "2.0.0"
}
```

## API Documentation

Once running, visit:
- **Swagger UI**: http://localhost:8080/docs
- **ReDoc**: http://localhost:8080/redoc
- **OpenAPI JSON**: http://localhost:8080/openapi.json

## Deployment Options

### Using Docker

```bash
docker build -t fastmcp-service .
docker run -p 8080:8080 fastmcp-service
```

### Using Docker Compose

```bash
docker-compose up -d
```

### Using Gunicorn

```bash
gunicorn fastmcp_service:app \
  --workers 4 \
  --worker-class uvicorn.workers.UvicornWorker \
  --bind 0.0.0.0:8080
```

### Using Systemd

```bash
sudo cp fastmcp-service.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable fastmcp-service
sudo systemctl start fastmcp-service
```

## Verification

Verify setup:
```bash
./verify_setup.sh
```

This checks:
- Python version
- Required files
- File permissions
- Dependencies
- Syntax

## Troubleshooting

### Service won't start

1. Check if port is available:
   ```bash
   lsof -i :8080
   ```

2. Try a different port:
   ```bash
   python fastmcp_service.py --port 8081
   ```

### Cannot connect to MCP server

1. Verify MCP server is running
2. Check MCP URL is correct
3. Test with curl:
   ```bash
   curl http://localhost:3000/mcp
   ```

### Import errors

Install dependencies:
```bash
pip install -r requirements.txt
```

## Performance

- **Memory**: ~50MB base + per-client overhead
- **CPU**: Low (async I/O)
- **Connections**: Unlimited concurrent clients
- **Timeout**: 30s connect, 60s tool execution (configurable)

## Security

- CORS: Configure allowed origins
- Authentication: Bearer/OAuth support
- Validation: All inputs validated
- Logging: Comprehensive logging
- Timeouts: Prevent hanging requests

## Contributing

1. Code follows PEP 8
2. Tests pass: `pytest -v`
3. Documentation updated
4. Type hints included

## Dependencies

- `fastmcp>=2.0.0` - FastMCP client
- `fastapi>=0.104.0` - FastAPI framework
- `uvicorn[standard]>=0.24.0` - ASGI server
- `httpx>=0.24.0` - HTTP client
- `pydantic>=2.0.0` - Data validation

## Version

**v2.0.0** - Production-ready release

## License

Part of the AgentAPI project.

## Support

- View logs: `tail -f fastmcp_service.log`
- Check health: `curl http://localhost:8080/health`
- API docs: http://localhost:8080/docs

## Links

- [FastMCP Documentation](https://github.com/modelcontextprotocol/fastmcp)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)
- [MCP Protocol](https://modelcontextprotocol.io/)

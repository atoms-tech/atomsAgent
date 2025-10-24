# FastMCP Service - API Reference

Complete API reference for FastMCP Service v2.0.0

## Base URL

```
http://localhost:8080
```

## Authentication

Currently, the FastMCP Service does not require authentication for the service itself. However, you can configure authentication when connecting to MCP servers.

## Content Type

All requests and responses use JSON:

```
Content-Type: application/json
```

## Error Responses

All endpoints may return the following error responses:

| Status Code | Description |
|-------------|-------------|
| 400 | Bad Request - Invalid request data |
| 404 | Not Found - Resource not found |
| 408 | Request Timeout - Operation timed out |
| 422 | Unprocessable Entity - Validation error |
| 500 | Internal Server Error - Server error |

Error format:
```json
{
  "detail": "Error message"
}
```

---

## Endpoints

### Health Check

Check the service health and get status information.

**Endpoint:** `GET /health`

**Parameters:** None

**Response:** `200 OK`

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T12:00:00.000000",
  "active_clients": 5,
  "version": "2.0.0"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

---

### Connect to MCP Server

Create a new MCP client connection.

**Endpoint:** `POST /mcp/connect`

**Request Body:**

```json
{
  "transport": "http|sse|stdio",
  "mcp_url": "string (required for http/sse)",
  "command": ["string"] (required for stdio),
  "auth_type": "none|bearer|oauth",
  "bearer_token": "string (required if auth_type=bearer)",
  "oauth_client_id": "string (required if auth_type=oauth)",
  "oauth_client_secret": "string (required if auth_type=oauth)",
  "oauth_auth_url": "string (required if auth_type=oauth)",
  "oauth_token_url": "string (required if auth_type=oauth)",
  "oauth_redirect_uri": "string (optional)",
  "oauth_scopes": ["string"] (optional),
  "name": "string (optional)",
  "version": "string (default: 1.0.0)",
  "timeout": number (default: 30)
}
```

**Response:** `201 Created`

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

**Examples:**

HTTP transport with no auth:
```bash
curl -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "http",
    "mcp_url": "http://localhost:3000/mcp",
    "auth_type": "none"
  }'
```

HTTP transport with bearer token:
```bash
curl -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "http",
    "mcp_url": "http://localhost:3000/mcp",
    "auth_type": "bearer",
    "bearer_token": "your-secret-token"
  }'
```

STDIO transport:
```bash
curl -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "stdio",
    "command": ["node", "server.js"],
    "auth_type": "none"
  }'
```

---

### Call Tool

Execute a tool on the connected MCP server.

**Endpoint:** `POST /mcp/call_tool`

**Request Body:**

```json
{
  "client_id": "string (required)",
  "tool_name": "string (required)",
  "arguments": {
    "key": "value"
  },
  "timeout": number (default: 60)
}
```

**Response:** `200 OK`

```json
{
  "status": "success|error",
  "result": any,
  "error": "string|null",
  "execution_time": number
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/mcp/call_tool \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "550e8400-e29b-41d4-a716-446655440000",
    "tool_name": "search",
    "arguments": {
      "query": "python async programming"
    },
    "timeout": 60
  }'
```

---

### List Tools

List all available tools from an MCP server.

**Endpoint:** `GET /mcp/list_tools`

**Query Parameters:**
- `client_id` (required): The client ID

**Response:** `200 OK`

```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "tools": [
    {
      "name": "tool_name",
      "description": "Tool description",
      "inputSchema": {
        "type": "object",
        "properties": {...}
      }
    }
  ],
  "count": 1
}
```

**Example:**

```bash
curl "http://localhost:8080/mcp/list_tools?client_id=550e8400-e29b-41d4-a716-446655440000"
```

---

### List Resources

List all available resources from an MCP server.

**Endpoint:** `GET /mcp/list_resources`

**Query Parameters:**
- `client_id` (required): The client ID

**Response:** `200 OK`

```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "resources": [
    {
      "uri": "file:///path/to/resource",
      "name": "Resource Name",
      "description": "Resource description",
      "mimeType": "text/plain"
    }
  ],
  "count": 1
}
```

**Example:**

```bash
curl "http://localhost:8080/mcp/list_resources?client_id=550e8400-e29b-41d4-a716-446655440000"
```

---

### Read Resource

Read the contents of a resource from an MCP server.

**Endpoint:** `POST /mcp/read_resource`

**Request Body:**

```json
{
  "client_id": "string (required)",
  "uri": "string (required)"
}
```

**Response:** `200 OK`

```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "uri": "file:///path/to/resource",
  "content": {
    "text": "Resource content"
  }
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/mcp/read_resource \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "550e8400-e29b-41d4-a716-446655440000",
    "uri": "file:///path/to/resource"
  }'
```

---

### List Prompts

List all available prompts from an MCP server.

**Endpoint:** `GET /mcp/list_prompts`

**Query Parameters:**
- `client_id` (required): The client ID

**Response:** `200 OK`

```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "prompts": [
    {
      "name": "prompt_name",
      "description": "Prompt description",
      "arguments": [...]
    }
  ],
  "count": 1
}
```

**Example:**

```bash
curl "http://localhost:8080/mcp/list_prompts?client_id=550e8400-e29b-41d4-a716-446655440000"
```

---

### Get Prompt

Get a prompt with arguments from an MCP server.

**Endpoint:** `POST /mcp/get_prompt`

**Request Body:**

```json
{
  "client_id": "string (required)",
  "prompt_name": "string (required)",
  "arguments": {
    "key": "value"
  }
}
```

**Response:** `200 OK`

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

**Example:**

```bash
curl -X POST http://localhost:8080/mcp/get_prompt \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "550e8400-e29b-41d4-a716-446655440000",
    "prompt_name": "code_review",
    "arguments": {
      "language": "python",
      "file": "app.py"
    }
  }'
```

---

### Disconnect

Disconnect from an MCP server and clean up resources.

**Endpoint:** `POST /mcp/disconnect`

**Request Body:**

```json
{
  "client_id": "string (required)"
}
```

**Response:** `200 OK`

```json
{
  "status": "disconnected",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Successfully disconnected from MCP server"
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/mcp/disconnect \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

---

### List Clients

List all active MCP clients.

**Endpoint:** `GET /mcp/clients`

**Parameters:** None

**Response:** `200 OK`

```json
{
  "status": "success",
  "clients": [
    {
      "client_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-15T12:00:00.000000",
      "transport": "http",
      "auth_type": "bearer",
      "name": "my-client",
      "mcp_url": "http://localhost:3000/mcp",
      "last_activity": "2024-01-15T12:05:00.000000"
    }
  ],
  "count": 1
}
```

**Example:**

```bash
curl http://localhost:8080/mcp/clients
```

---

### Get Client Info

Get detailed information about a specific MCP client.

**Endpoint:** `GET /mcp/client/{client_id}`

**Path Parameters:**
- `client_id` (required): The client ID

**Response:** `200 OK`

```json
{
  "status": "success",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "metadata": {
    "created_at": "2024-01-15T12:00:00.000000",
    "transport": "http",
    "auth_type": "bearer",
    "name": "my-client",
    "mcp_url": "http://localhost:3000/mcp",
    "last_activity": "2024-01-15T12:05:00.000000"
  }
}
```

**Example:**

```bash
curl http://localhost:8080/mcp/client/550e8400-e29b-41d4-a716-446655440000
```

---

## Data Types

### Transport Types

- `http` - HTTP transport
- `sse` - Server-Sent Events transport
- `stdio` - Standard input/output transport

### Auth Types

- `none` - No authentication
- `bearer` - Bearer token authentication
- `oauth` - OAuth 2.0 authentication

### Client Status

- `connecting` - Client is connecting
- `connected` - Client is connected
- `disconnected` - Client is disconnected
- `error` - Client encountered an error

---

## Complete Workflow Example

```bash
# 1. Connect to MCP server
RESPONSE=$(curl -s -X POST http://localhost:8080/mcp/connect \
  -H "Content-Type: application/json" \
  -d '{
    "transport": "http",
    "mcp_url": "http://localhost:3000/mcp",
    "auth_type": "none",
    "name": "workflow-example"
  }')

CLIENT_ID=$(echo $RESPONSE | jq -r '.client_id')
echo "Connected: $CLIENT_ID"

# 2. List tools
curl -s "http://localhost:8080/mcp/list_tools?client_id=$CLIENT_ID" | jq '.tools[].name'

# 3. Call a tool
curl -s -X POST http://localhost:8080/mcp/call_tool \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"tool_name\": \"search\",
    \"arguments\": {\"query\": \"test\"}
  }" | jq

# 4. List resources
curl -s "http://localhost:8080/mcp/list_resources?client_id=$CLIENT_ID" | jq

# 5. List prompts
curl -s "http://localhost:8080/mcp/list_prompts?client_id=$CLIENT_ID" | jq

# 6. Get client info
curl -s "http://localhost:8080/mcp/client/$CLIENT_ID" | jq

# 7. Disconnect
curl -s -X POST http://localhost:8080/mcp/disconnect \
  -H "Content-Type: application/json" \
  -d "{\"client_id\": \"$CLIENT_ID\"}" | jq
```

---

## Rate Limiting

Currently, no rate limiting is implemented. Consider adding rate limiting middleware for production use.

---

## Pagination

Currently, no pagination is implemented. All results are returned in a single response.

---

## Versioning

API version: `2.0.0`

The version is included in health check responses.

---

## CORS

CORS is enabled for all origins by default. Configure the allowed origins in production:

```python
app.add_middleware(
    CORSMiddleware,
    allow_origins=["https://yourapp.com"],
    allow_credentials=True,
    allow_methods=["GET", "POST"],
    allow_headers=["*"],
)
```

---

## OpenAPI/Swagger

Interactive API documentation is available at:
- Swagger UI: http://localhost:8080/docs
- ReDoc: http://localhost:8080/redoc
- OpenAPI JSON: http://localhost:8080/openapi.json

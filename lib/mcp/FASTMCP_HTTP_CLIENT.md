# FastMCP HTTP Client

A comprehensive Go HTTP client for communicating with the FastMCP service, providing a robust interface for managing Model Context Protocol (MCP) server connections.

## Overview

The FastMCP HTTP Client (`fastmcp_http_client.go`) provides a clean, type-safe way to interact with the FastMCP service running on `http://localhost:8000` (or a custom URL). It handles:

- Connection management for multiple MCP servers
- Tool invocation with type-safe request/response handling
- Comprehensive error handling and retry logic
- Context-aware timeouts and cancellation
- Thread-safe concurrent operations

## Features

### Core Functionality

1. **Connection Management**
   - Connect to MCP servers via stdio, SSE, or HTTP transports
   - Support for OAuth authentication
   - Graceful disconnection
   - Multiple concurrent connections

2. **Tool Operations**
   - List available tools from connected servers
   - Call tools with arguments
   - Type-safe response handling

3. **Reliability**
   - Automatic retry with exponential backoff (max 3 retries)
   - Configurable timeouts
   - Context cancellation support
   - Thread-safe operations with mutex protection

4. **Error Handling**
   - Connection errors
   - HTTP status code errors
   - JSON serialization/deserialization errors
   - Timeout errors
   - Service unavailable errors

## Architecture

### Main Components

```
FastMCPHTTPClient
├── baseURL: string              # Service endpoint (default: http://localhost:8000)
├── httpClient: *http.Client     # HTTP client with timeout
├── timeout: time.Duration       # Request timeout (default: 30s)
└── mu: sync.RWMutex            # Thread-safe operations
```

### Request/Response Types

**Connection:**
- `ConnectRequest` / `ConnectResponse`
- `DisconnectRequest` / `DisconnectResponse`

**Tool Operations:**
- `ToolCallRequest` / `ToolCallResponse`
- `ListToolsRequest` / `ListToolsResponse`

**Health Check:**
- `HealthResponse`

**Configuration:**
- `HTTPMCPConfig` - Configuration for MCP server connections

## API Reference

### Creating a Client

```go
// Use default URL (http://localhost:8000)
client := NewFastMCPHTTPClient("")

// Use custom URL
client := NewFastMCPHTTPClient("http://localhost:9000")
```

### Connection Methods

#### Connect
```go
func (c *FastMCPHTTPClient) Connect(
    ctx context.Context,
    clientID string,
    config HTTPMCPConfig
) error
```

Establishes a connection to an MCP server.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `clientID`: Unique identifier for this connection
- `config`: Connection configuration (transport type, command/URL, etc.)

**Example:**
```go
config := HTTPMCPConfig{
    Transport: "stdio",
    Command:   "python",
    Args:      []string{"-m", "mcp.server.example"},
    Env: map[string]string{
        "DEBUG": "true",
    },
}
err := client.Connect(ctx, "my-client", config)
```

#### Disconnect
```go
func (c *FastMCPHTTPClient) Disconnect(
    ctx context.Context,
    clientID string
) error
```

Closes a connection to an MCP server.

### Tool Methods

#### ListTools
```go
func (c *FastMCPHTTPClient) ListTools(
    ctx context.Context,
    clientID string
) ([]Tool, error)
```

Retrieves available tools from a connected MCP server.

**Returns:**
- `[]Tool`: List of available tools with name, description, and input schema

**Example:**
```go
tools, err := client.ListTools(ctx, "my-client")
for _, tool := range tools {
    fmt.Printf("%s: %s\n", tool.Name, tool.Description)
}
```

#### CallTool
```go
func (c *FastMCPHTTPClient) CallTool(
    ctx context.Context,
    clientID string,
    toolName string,
    args map[string]any
) (map[string]any, error)
```

Invokes a tool on the MCP server.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `clientID`: Connection identifier
- `toolName`: Name of the tool to call
- `args`: Tool arguments as key-value map

**Returns:**
- `map[string]any`: Tool execution result

**Example:**
```go
result, err := client.CallTool(ctx, "my-client", "echo", map[string]any{
    "message": "Hello, World!",
})
```

### Health Check

#### Health
```go
func (c *FastMCPHTTPClient) Health(ctx context.Context) error
```

Checks if the FastMCP service is healthy and available.

**Example:**
```go
if err := client.Health(ctx); err != nil {
    log.Fatalf("Service unavailable: %v", err)
}
```

### Configuration

#### SetTimeout
```go
func (c *FastMCPHTTPClient) SetTimeout(timeout time.Duration)
```

Sets the HTTP client timeout.

**Example:**
```go
client.SetTimeout(60 * time.Second)
```

## Transport Types

### Stdio Transport
For MCP servers running as local processes:

```go
config := HTTPMCPConfig{
    Transport: "stdio",
    Command:   "python",
    Args:      []string{"-m", "mcp.server.myserver"},
    Env: map[string]string{
        "API_KEY": "secret",
    },
}
```

### SSE Transport
For Server-Sent Events connections:

```go
config := HTTPMCPConfig{
    Transport: "sse",
    MCPURL:    "https://example.com/mcp/sse",
}
```

### HTTP Transport
For HTTP-based MCP servers:

```go
config := HTTPMCPConfig{
    Transport: "http",
    MCPURL:    "https://api.example.com/mcp",
}
```

### OAuth Authentication
For OAuth-protected endpoints:

```go
config := HTTPMCPConfig{
    Transport:     "http",
    MCPURL:        "https://api.example.com/mcp",
    OAuthProvider: "google",
}
```

## Error Handling

The client provides comprehensive error handling:

### Error Types

1. **Connection Errors**
   - Network failures
   - Service unavailable
   - Connection refused

2. **HTTP Errors**
   - 4xx client errors (not retried)
   - 5xx server errors (retried with backoff)

3. **Timeout Errors**
   - Context deadline exceeded
   - Request timeout

4. **Serialization Errors**
   - JSON marshal/unmarshal failures

### Retry Logic

The client automatically retries transient errors with exponential backoff:

- **Max Retries:** 3
- **Backoff:** 100ms, 200ms, 400ms (capped at 5s)
- **Retryable Errors:**
  - Connection refused/reset
  - Timeout
  - HTTP 500, 502, 503, 504

**Non-retryable errors:**
- 4xx client errors
- Context cancellation
- Serialization errors

### Example Error Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := client.Connect(ctx, clientID, config); err != nil {
    switch {
    case ctx.Err() == context.DeadlineExceeded:
        log.Printf("Connection timed out")
    case strings.Contains(err.Error(), "connection refused"):
        log.Printf("Service not running")
    case strings.Contains(err.Error(), "HTTP 4"):
        log.Printf("Client error: %v", err)
    default:
        log.Printf("Connection failed: %v", err)
    }
    return
}
```

## Thread Safety

The client is thread-safe and can be used concurrently:

- Read operations (CallTool, ListTools, Health) use `RLock()`
- Write operations (Connect, Disconnect) use `Lock()`
- Safe for concurrent use across multiple goroutines

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        result, _ := client.CallTool(ctx, "client", "tool", args)
        fmt.Printf("Result %d: %v\n", n, result)
    }(i)
}
wg.Wait()
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/coder/agentapi/lib/mcp"
)

func main() {
    // Create client
    client := mcp.NewFastMCPHTTPClient("http://localhost:8000")
    client.SetTimeout(60 * time.Second)

    ctx := context.Background()

    // Check service health
    if err := client.Health(ctx); err != nil {
        log.Fatalf("Service unavailable: %v", err)
    }

    // Connect to MCP server
    clientID := "my-app"
    config := mcp.HTTPMCPConfig{
        Transport: "stdio",
        Command:   "python",
        Args:      []string{"-m", "mcp.server.example"},
    }

    if err := client.Connect(ctx, clientID, config); err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Disconnect(ctx, clientID)

    // List tools
    tools, err := client.ListTools(ctx, clientID)
    if err != nil {
        log.Fatalf("Failed to list tools: %v", err)
    }

    fmt.Printf("Available tools:\n")
    for _, tool := range tools {
        fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
    }

    // Call a tool
    result, err := client.CallTool(ctx, clientID, "echo", map[string]any{
        "message": "Hello, FastMCP!",
    })
    if err != nil {
        log.Fatalf("Tool call failed: %v", err)
    }

    fmt.Printf("Result: %v\n", result)
}
```

## Testing

The client includes comprehensive tests in `fastmcp_http_client_test.go`:

```bash
# Run all tests
go test ./lib/mcp/

# Run with verbose output
go test -v ./lib/mcp/

# Run specific test
go test -v ./lib/mcp/ -run TestConnect
```

**Test Coverage:**
- Client initialization
- Connection management
- Tool operations
- Health checks
- Error handling
- Context cancellation
- Retry logic
- HTTP error responses
- Timeouts

## Integration with Existing Code

The HTTP client works alongside the existing Python wrapper-based client:

- **FastMCPClient** (`fastmcp_client.go`): Python subprocess wrapper
- **FastMCPHTTPClient** (`fastmcp_http_client.go`): HTTP service client

Choose based on your needs:
- Use **FastMCPHTTPClient** for production deployments with the FastMCP service
- Use **FastMCPClient** for direct Python integration

## Dependencies

- Standard library only (no external dependencies)
- Compatible with Go 1.16+

## API Endpoints

The client communicates with these FastMCP service endpoints:

- `POST /api/connect` - Establish MCP connection
- `POST /api/disconnect` - Close MCP connection
- `POST /api/call_tool` - Invoke a tool
- `POST /api/list_tools` - List available tools
- `GET /health` - Health check

## Best Practices

1. **Always use contexts with timeouts:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

2. **Always disconnect when done:**
   ```go
   defer client.Disconnect(ctx, clientID)
   ```

3. **Check service health before connecting:**
   ```go
   if err := client.Health(ctx); err != nil {
       return err
   }
   ```

4. **Handle errors appropriately:**
   - Don't ignore errors
   - Log failures for debugging
   - Implement fallback strategies

5. **Use unique client IDs:**
   - Prevents connection conflicts
   - Enables concurrent connections
   - Simplifies debugging

## Troubleshooting

### Service Not Available
**Error:** `connection refused` or `health check failed`

**Solution:** Ensure FastMCP service is running:
```bash
# Start the FastMCP service
python -m fastmcp.server --port 8000
```

### Connection Timeout
**Error:** `request timeout` or `context deadline exceeded`

**Solution:** Increase timeout or check network:
```go
client.SetTimeout(120 * time.Second)
```

### Tool Call Failed
**Error:** `tool call failed: <error message>`

**Solution:**
1. Verify tool exists: `client.ListTools(ctx, clientID)`
2. Check arguments match input schema
3. Review MCP server logs

### Max Retries Exceeded
**Error:** `max retries exceeded`

**Solution:**
1. Check service logs
2. Verify service is responsive
3. Reduce load or increase timeout

## Contributing

When modifying the client:

1. Run tests: `go test ./lib/mcp/`
2. Run linter: `golangci-lint run`
3. Update documentation
4. Add tests for new features

## License

Same as the parent agentapi project.

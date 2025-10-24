# FastMCP Integration Guide

This document explains how to integrate FastMCP 2.0 with AgentAPI for enterprise MCP management.

## Architecture Overview

### FastMCP Integration Pattern

Instead of using the basic Go MCP SDK, we use FastMCP 2.0 as a Python service that acts as an MCP client wrapper:

```
AgentAPI (Go) → FastMCP Wrapper (Python) → MCP Servers
```

### Key Benefits of FastMCP 2.0

1. **Mature OAuth 2.1 Support**: Built-in OAuth providers for Auth0, Google, GitHub, Azure, etc.
2. **Advanced Authentication**: Bearer tokens, JWT, OIDC, and custom auth flows
3. **Rich Client Features**: Progress monitoring, user elicitation, logging, sampling
4. **Production Ready**: HTTP deployment, cloud hosting, comprehensive testing
5. **Extensible**: Middleware, composition, proxy servers

## Implementation

### 1. FastMCP Wrapper Service

The `lib/mcp/fastmcp_wrapper.py` provides a Python service that:

- Manages multiple MCP connections
- Handles OAuth flows automatically
- Provides a JSON-RPC interface for Go
- Supports HTTP, SSE, and stdio transports

### 2. Go Client Interface

The `lib/mcp/fastmcp_client.go` provides a Go interface that:

- Spawns and manages the Python process
- Sends commands via stdin/stdout
- Handles responses asynchronously
- Provides type-safe Go APIs

### 3. OAuth Integration

FastMCP supports multiple OAuth providers out of the box:

```python
# Example OAuth configuration
from fastmcp.client.auth import OAuthAuth

auth = OAuthAuth(
    client_id="your_client_id",
    client_secret="your_client_secret", 
    auth_url="https://auth.provider.com/oauth/authorize",
    token_url="https://auth.provider.com/oauth/token"
)
```

## Supported MCP Types

### 1. HTTP MCPs
- REST API-based MCP servers
- Bearer token authentication
- OAuth 2.1 flows
- Custom headers and configuration

### 2. SSE MCPs
- Server-Sent Events based MCP servers
- Real-time communication
- Automatic reconnection
- Event streaming

### 3. Stdio MCPs
- Command-line MCP servers
- Process management
- Environment variable configuration
- Automatic cleanup

## OAuth Flow Implementation

### Frontend OAuth Client

For MCPs requiring OAuth, implement a frontend client:

```typescript
// Example OAuth flow
class MCPOAuthClient {
  async initiateOAuth(mcpConfig: MCPConfig) {
    const authUrl = this.buildAuthUrl(mcpConfig);
    const popup = window.open(authUrl, 'oauth', 'width=600,height=600');
    
    return new Promise((resolve, reject) => {
      const handleMessage = (event: MessageEvent) => {
        if (event.data.type === 'oauth_callback') {
          popup.close();
          resolve(event.data.credentials);
        }
      };
      
      window.addEventListener('message', handleMessage);
    });
  }
}
```

### Backend Integration

Store OAuth credentials securely:

```go
// Store OAuth credentials in Supabase
type OAuthCredentials struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    Scope        string    `json:"scope"`
}
```

## MCP Management API

### Create MCP Connection

```bash
POST /api/v1/mcps
{
  "name": "GitHub MCP",
  "type": "http",
  "endpoint": "https://api.github.com/mcp",
  "auth_type": "oauth",
  "config": {
    "client_id": "github_client_id",
    "scopes": ["repo", "user"]
  }
}
```

### List Available Tools

```bash
GET /api/v1/mcps/{id}/tools
```

### Execute Tool

```bash
POST /api/v1/mcps/{id}/tools/{tool_name}/execute
{
  "arguments": {
    "repository": "owner/repo",
    "issue_number": 123
  }
}
```

## Security Considerations

### 1. Credential Management

- Store OAuth tokens in GCP Secret Manager
- Implement automatic token refresh
- Use short-lived access tokens
- Encrypt sensitive configuration

### 2. MCP Validation

- Validate MCP server certificates
- Implement allowlists for MCP endpoints
- Scan MCP tools for security issues
- Monitor MCP usage and access

### 3. User Isolation

- Each user session gets isolated MCP connections
- MCP credentials scoped to user/organization
- Audit logging for all MCP operations
- Rate limiting per user/MCP

## Deployment

### Docker Configuration

The updated Dockerfile includes:

- Python 3.11 runtime
- FastMCP dependencies
- MCP wrapper script
- All agent CLIs

### Environment Variables

```bash
# FastMCP Configuration
FASTMCP_PYTHON_PATH=/usr/local/bin/python3
FASTMCP_WRAPPER_PATH=/usr/local/bin/fastmcp_wrapper.py

# OAuth Providers
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi-multitenant
spec:
  template:
    spec:
      containers:
      - name: agentapi
        image: agentapi:multitenant
        env:
        - name: FASTMCP_PYTHON_PATH
          value: "/usr/local/bin/python3"
        - name: GITHUB_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: oauth-secrets
              key: github-client-id
```

## Monitoring and Observability

### Metrics

- MCP connection status
- Tool execution success/failure rates
- OAuth flow completion rates
- Response times per MCP server

### Logging

- Structured JSON logs for all MCP operations
- OAuth flow audit trail
- Error tracking and alerting
- Performance monitoring

### Health Checks

- FastMCP wrapper process health
- MCP server connectivity
- OAuth token validity
- Resource utilization

## Development Workflow

### Local Development

```bash
# Install Python dependencies
pip install -r requirements.txt

# Start FastMCP wrapper
python3 lib/mcp/fastmcp_wrapper.py

# Test MCP connection
curl -X POST http://localhost:3284/api/v1/mcps \
  -H "Content-Type: application/json" \
  -d '{"name": "Test MCP", "type": "http", "endpoint": "https://api.example.com/mcp"}'
```

### Testing

```bash
# Run Go tests
go test ./lib/mcp/...

# Run Python tests
python3 -m pytest lib/mcp/tests/

# Integration tests
go test ./e2e/mcp/...
```

## Migration from Basic MCP SDK

### 1. Update MCP Client Usage

```go
// Before: Basic MCP client
client := mcp.NewClient(...)

// After: FastMCP wrapper
client, err := NewFastMCPClient()
```

### 2. OAuth Integration

```go
// Before: Manual OAuth handling
// (No built-in OAuth support)

// After: FastMCP OAuth
config := MCPConfig{
    AuthType: "oauth",
    Auth: map[string]string{
        "client_id":     "github_client_id",
        "client_secret": "github_client_secret",
        "auth_url":      "https://github.com/login/oauth/authorize",
        "token_url":     "https://github.com/login/oauth/access_token",
    },
}
```

### 3. Enhanced Features

```go
// Progress monitoring
result, err := client.CallTool(ctx, mcpID, "long_running_task", args)

// User elicitation
prompt, err := client.GetPrompt(ctx, mcpID, "user_input", map[string]any{
    "field": "repository_name",
    "description": "Enter the repository name",
})

// Resource access
resources, err := client.ListResources(ctx, mcpID)
```

## Troubleshooting

### Common Issues

1. **Python Process Not Starting**
   - Check Python installation
   - Verify FastMCP dependencies
   - Check file permissions

2. **OAuth Flow Failures**
   - Verify client credentials
   - Check redirect URLs
   - Validate scopes

3. **MCP Connection Issues**
   - Check network connectivity
   - Verify MCP server endpoints
   - Validate authentication

### Debug Mode

```bash
# Enable debug logging
export FASTMCP_DEBUG=1
export FASTMCP_LOG_LEVEL=debug

# Start with verbose output
python3 lib/mcp/fastmcp_wrapper.py --verbose
```

## Next Steps

1. **Implement OAuth Flows**: Set up OAuth providers for common MCPs
2. **Add MCP Discovery**: Implement automatic MCP server discovery
3. **Enhanced Security**: Add MCP server validation and scanning
4. **Performance Optimization**: Implement connection pooling and caching
5. **Monitoring**: Add comprehensive metrics and alerting

## Resources

- [FastMCP Documentation](https://gofastmcp.com/)
- [MCP Specification](https://modelcontextprotocol.io/)
- [OAuth 2.1 RFC](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/)
- [AgentAPI Multi-tenant Guide](./MULTITENANT.md)
# Multi-Tenant AgentAPI Implementation

This document describes the multi-tenant implementation of AgentAPI with CCRouter support, MCP integration, and enterprise features.

## Architecture Overview

### Core Components

1. **Session Management**: Isolated user sessions with separate workspaces
2. **MCP Integration**: Model Context Protocol support with OAuth flows
3. **System Prompt Management**: Hierarchical prompt configuration
4. **Multi-tenant Database**: Supabase PostgreSQL with RLS
5. **Container Orchestration**: Docker-based deployment with security isolation

### Key Features

- **Jira-scale Multi-tenancy**: Support for thousands of concurrent users
- **SOC2 Compliance**: Audit logging, data encryption, access controls
- **VertexAI Integration**: CCRouter support for Google Cloud VertexAI models
- **MCP Support**: HTTP, SSE, and stdio MCP clients with OAuth
- **System Prompt Management**: Global/Org/User scoped prompts with templates
- **Container Isolation**: Separate containers per user session for security

## Database Schema

The implementation uses Supabase PostgreSQL with Row Level Security (RLS):

### Core Tables

- `organizations`: Organization management
- `users`: User profiles linked to Supabase auth
- `user_sessions`: Isolated user sessions
- `mcp_configs`: MCP server configurations
- `system_prompts`: System prompt configurations
- `audit_logs`: Compliance and security logging

### Security Features

- **RLS Policies**: Data isolation at the database level
- **Audit Logging**: Complete audit trail for compliance
- **Credential Management**: Secure storage of API keys and tokens
- **Session Isolation**: Separate workspaces and processes per user

## API Endpoints

### Session Management

```
POST   /api/v1/sessions                    # Create new session
GET    /api/v1/sessions/{id}               # Get session details
DELETE /api/v1/sessions/{id}               # Terminate session
GET    /api/v1/sessions                    # List user sessions
```

### MCP Management

```
GET    /api/v1/mcps                        # List available MCPs
POST   /api/v1/mcps                        # Add new MCP
PUT    /api/v1/mcps/{id}                   # Update MCP config
DELETE /api/v1/mcps/{id}                   # Remove MCP
POST   /api/v1/mcps/{id}/validate          # Validate MCP connection
```

### System Prompts

```
GET    /api/v1/prompts                     # Get system prompts
POST   /api/v1/prompts                     # Create system prompt
PUT    /api/v1/prompts/{id}                # Update system prompt
DELETE /api/v1/prompts/{id}                # Delete system prompt
```

## CCRouter Integration

### Supported Agent Types

- `ccrouter` / `ccr`: Claude Code Router with VertexAI support
- `droid`: Factory AI Droid CLI
- All existing agent types (claude, goose, aider, etc.)

### Usage

```bash
# Start server with CCRouter
./agentapi server --type=ccrouter -- ccr code

# Start server with Droid
./agentapi server --type=droid -- droid
```

### CCRouter Configuration

CCRouter supports VertexAI models through its built-in providers:

```json
{
  "Providers": [
    {
      "name": "vertex-gemini",
      "api_base_url": "https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/",
      "api_key": "${VERTEX_AI_API_KEY}",
      "models": ["gemini-1.5-pro", "gemini-1.5-flash"],
      "transformer": {
        "use": ["vertex-gemini"]
      }
    }
  ],
  "Router": {
    "default": "vertex-gemini,gemini-1.5-pro"
  }
}
```

## MCP Integration

### Supported MCP Types

1. **HTTP MCPs**: REST API-based MCP servers
2. **SSE MCPs**: Server-Sent Events MCP servers
3. **Stdio MCPs**: Command-line MCP servers

### OAuth Flow

For MCPs requiring OAuth authentication:

1. Frontend client initiates OAuth flow
2. User completes authentication
3. Credentials stored in Supabase
4. MCP client connects with stored credentials
5. Periodic token refresh handled automatically

### Example MCP Configuration

```json
{
  "id": "github-mcp",
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

## System Prompt Management

### Hierarchical Prompts

1. **Global Prompts**: Applied to all users
2. **Organization Prompts**: Applied to users in specific org
3. **User Prompts**: Applied to specific users

### Template Support

Prompts support Go templates for dynamic content:

```go
// Example template
"Hello {{.UserName}}, you are working on project {{.ProjectName}} in organization {{.OrgName}}."
```

### Security Features

- **Prompt Validation**: Detects dangerous patterns
- **Content Sanitization**: Prevents XSS and injection attacks
- **Audit Logging**: Tracks all prompt modifications

## Deployment

### Docker Compose

```bash
# Start multi-tenant AgentAPI
docker-compose -f docker-compose.multitenant.yml up -d
```

### Environment Variables

```bash
# Supabase Configuration
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_anon_key
SUPABASE_SERVICE_ROLE_KEY=your_service_role_key

# GCP Configuration
GCP_PROJECT_ID=your_project_id
GCP_SECRET_MANAGER_KEY=your_secret_manager_key

# AgentAPI Configuration
AGENTAPI_PORT=3284
AGENTAPI_ALLOWED_HOSTS=*
AGENTAPI_ALLOWED_ORIGINS=*
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentapi-multitenant
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentapi
  template:
    metadata:
      labels:
        app: agentapi
    spec:
      containers:
      - name: agentapi
        image: agentapi:multitenant
        ports:
        - containerPort: 3284
        env:
        - name: SUPABASE_URL
          valueFrom:
            secretKeyRef:
              name: supabase-secrets
              key: url
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

## Security & Compliance

### SOC2 Compliance

- **Audit Logging**: All actions logged with user context
- **Data Encryption**: Sensitive data encrypted at rest
- **Access Controls**: RLS policies enforce data isolation
- **Session Management**: Secure session handling with expiration

### Security Features

- **Container Isolation**: Each user session runs in separate container
- **Credential Management**: Secure storage using GCP Secret Manager
- **Input Validation**: All inputs validated and sanitized
- **Rate Limiting**: API rate limiting per user/organization

## Monitoring & Observability

### Metrics

- Session count and duration
- MCP connection status
- API response times
- Error rates by endpoint

### Logging

- Structured JSON logging
- Audit trail for compliance
- Error tracking and alerting
- Performance monitoring

### Health Checks

- Database connectivity
- MCP server availability
- Container health status
- Resource utilization

## Development

### Local Development

```bash
# Start development environment
docker-compose -f docker-compose.dev.yml up -d

# Run tests
go test ./...

# Build for production
make build
```

### Testing

```bash
# Run unit tests
go test ./lib/...

# Run integration tests
go test ./e2e/...

# Run security tests
go test ./security/...
```

## Roadmap

### Phase 1 (MVP)
- [x] Basic multi-tenant architecture
- [x] CCRouter integration
- [x] MCP client framework
- [x] System prompt management
- [x] Docker containerization

### Phase 2 (Enterprise)
- [ ] Advanced MCP OAuth flows
- [ ] Kubernetes orchestration
- [ ] Advanced monitoring
- [ ] Performance optimization

### Phase 3 (Compliance)
- [ ] FedRAMP compliance
- [ ] HIPAA compliance
- [ ] Advanced security features
- [ ] Enterprise SSO integration

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
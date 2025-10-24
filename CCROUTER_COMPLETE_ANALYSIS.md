# CCRouter Complete Integration Analysis

## Executive Summary

CCRouter (Claude Code Router) is a powerful LLM request routing and transformation layer that enables Claude Code (and other agents) to work with multiple LLM providers without native account requirements. Version 1.0.58 is currently installed and running on port 3456.

---

## 1. CCROUTER CLI COMMANDS & OPTIONS

### Available Commands

```
ccr start          Start server 
ccr stop           Stop server
ccr restart        Restart server
ccr status         Show server status
ccr statusline     Integrated statusline (beta)
ccr code           Execute claude command
ccr ui             Open the web UI in browser
ccr -v/--version   Show version information
ccr -h/--help      Show help information
```

### Usage Examples

```bash
# Start the router server
ccr start

# Check server status
ccr status

# Run Claude Code with routing
ccr code

# Open web UI for configuration
ccr ui

# Check version
ccr -v

# View help
ccr --help
```

---

## 2. CONFIGURATION FILES & STRUCTURE

### Configuration Location
```
~/.claude-code-router/
├── config.json              # Main configuration (empty by default)
├── .claude-code-router.pid  # Server process ID
├── plugins/                 # Custom transformer plugins directory
└── logs/
    └── ccr-*.log           # Server and application logs
```

### Default Configuration Structure
```json
{
  "PORT": 3456,
  "Providers": [],
  "Router": {}
}
```

### Complete Configuration Schema

#### Core Settings
- **PORT** (integer): Server port (default: 3456)
- **APIKEY** (string, optional): Secret key for API authentication
  - When set, clients must provide `Authorization: Bearer <key>` or `x-api-key` header
  - If not set, server forces HOST to 127.0.0.1 for security
- **HOST** (string, optional): Server host address (requires APIKEY if not 127.0.0.1)
- **PROXY_URL** (string, optional): HTTP proxy for API requests (e.g., "http://127.0.0.1:7890")
- **LOG** (boolean, default: true): Enable/disable logging
- **LOG_LEVEL** (string, default: "debug"): Log level (fatal, error, warn, info, debug, trace)
- **API_TIMEOUT_MS** (number): API call timeout in milliseconds
- **NON_INTERACTIVE_MODE** (boolean): Enable for CI/CD environments
- **CUSTOM_ROUTER_PATH** (string): Path to custom router script

#### Logging Systems
Two separate logging systems:
1. **Server-level logs** (`~/.claude-code-router/logs/ccr-*.log`)
   - HTTP requests, API calls, server events
   - Uses pino logger
2. **Application-level logs** (`~/.claude-code-router/claude-code-router.log`)
   - Routing decisions, business logic events

#### Environment Variable Interpolation
Supports `$VAR_NAME` or `${VAR_NAME}` syntax for secure API key management:
```json
{
  "Providers": [
    {
      "api_key": "$OPENAI_API_KEY"
    }
  ]
}
```

---

## 3. VERTEXAI MODEL ROUTING

### VertexAI Transformer

The `vertex-gemini` transformer handles Google Cloud VertexAI authentication and API formatting.

### Configuration Example

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

### VertexAI API Endpoints
- **Base URL**: `https://us-central1-aiplatform.googleapis.com/v1/`
- **Projects Path**: `projects/{PROJECT_ID}/locations/{LOCATION}/publishers/google/models/`
- **Available Locations**: us-central1 (primary), others available

### Available VertexAI Models
- `gemini-1.5-pro`: High-performance reasoning model
- `gemini-1.5-flash`: Fast, lightweight model

### Authentication
- Uses Google Cloud service account credentials
- API key format: Google Cloud project credentials
- Environment variable: `VERTEX_AI_API_KEY`

---

## 4. CCROUTER NPM PACKAGE STRUCTURE

### Package Information
```
Name: @musistudio/claude-code-router
Version: 1.0.58
Location: /opt/homebrew/lib/node_modules/@musistudio/claude-code-router/
Bin: dist/cli.js (compiled to /opt/homebrew/bin/ccr)
```

### Key Dependencies
```
"@fastify/static": "^8.2.0"      # Static file serving
"@musistudio/llms": "^1.0.36"    # LLM provider abstractions
"dotenv": "^16.4.7"              # Environment variable management
"find-process": "^2.0.0"         # Process discovery
"json5": "^2.2.3"                # JSON5 parsing
"minimist": "^1.2.8"             # CLI argument parsing
"openurl": "^1.1.1"              # URL opening utility
"rotating-file-stream": "^3.2.7" # Log rotation
"shell-quote": "^1.8.3"          # Shell command quoting
"tiktoken": "^1.0.21"            # Token counting
"uuid": "^11.1.0"                # UUID generation
```

### Package Structure
```
@musistudio/claude-code-router/
├── dist/
│   ├── cli.js              # Compiled CLI binary (3.4MB)
│   ├── index.html          # Web UI
│   └── tiktoken_bg.wasm    # WASM module for tokenization
├── README.md               # Comprehensive documentation
├── package.json            # Package metadata
├── custom-router.example.js # Example custom router
└── node_modules/           # Dependencies
```

---

## 5. CCROUTER - CLAUDE CODE COMMUNICATION

### Integration Points

#### 1. Server Mode
CCRouter runs as an HTTP server that acts as a proxy between Claude Code and various LLM providers.

```
Claude Code → CCRouter (localhost:3456) → Provider API
```

#### 2. API Endpoints
- **Standard OpenAI-compatible**: `/v1/chat/completions`
- **Messages endpoint**: `/v1/messages`
- **Response endpoint**: `/v1/responses`

#### 3. Communication Protocol
- **Protocol**: HTTP/HTTPS with streaming support
- **Format**: OpenAI API-compatible JSON
- **Streaming**: Server-Sent Events (SSE) for real-time responses

#### 4. Request Flow
1. Claude Code sends request to `http://127.0.0.1:3456`
2. CCRouter receives request and extracts routing information
3. CCRouter applies transformers to request
4. Request forwarded to appropriate provider API
5. Response received and transformed back to OpenAI format
6. Response streamed to Claude Code

#### 5. Environment Variables for Integration
```bash
# Override Anthropic endpoint to use CCRouter
ANTHROPIC_BASE_URL=http://localhost:3456
ANTHROPIC_API_KEY=any-valid-string  # Can be any string when using CCRouter
```

---

## 6. ENVIRONMENT VARIABLES & CONFIGURATION

### System Environment Variables

#### Required for VertexAI
```bash
VERTEX_AI_API_KEY         # Google Cloud credentials
VERTEX_AI_PROJECT_ID      # GCP project ID
```

#### Common Provider Variables
```bash
OPENAI_API_KEY           # OpenAI API key
DEEPSEEK_API_KEY         # DeepSeek API key
GEMINI_API_KEY          # Google Gemini API key
ANTHROPIC_API_KEY       # Anthropic API key (for direct use)
```

#### CCRouter Configuration
```bash
ANTHROPIC_BASE_URL      # Override to use CCRouter (http://127.0.0.1:3456)
CLAUDECODE              # Set to "1" when using Claude Code
```

#### CI/CD Environment
```bash
CI=true                 # Indicates CI/CD environment
FORCE_COLOR=0          # Disable colored output
NON_INTERACTIVE_MODE   # Set to true in config.json for automation
```

### Configuration File Variables
All variables can be set in `~/.claude-code-router/config.json`:

```json
{
  "PORT": 3456,
  "APIKEY": "optional-secret-key",
  "HOST": "127.0.0.1",
  "PROXY_URL": "http://127.0.0.1:7890",
  "LOG": true,
  "LOG_LEVEL": "debug",
  "API_TIMEOUT_MS": 600000,
  "NON_INTERACTIVE_MODE": false,
  "CUSTOM_ROUTER_PATH": "/path/to/custom-router.js"
}
```

---

## 7. TESTING CCROUTER COMMANDS

### Status Check
```bash
$ ccr status
📊 Claude Code Router Status
════════════════════════════════════════
✅ Status: Running
🆔 Process ID: 3985
🌐 Port: 3456
📡 API Endpoint: http://127.0.0.1:3456
📄 PID File: /Users/kooshapari/.claude-code-router/.claude-code-router.pid

Ready to use! Run the following commands:
   ccr code    # Start coding with Claude
   ccr stop   # Stop the service
```

### API Test
```bash
$ curl -X POST http://127.0.0.1:3456/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model": "test"}'

# Response: {"error":"Missing model in request body"}
```

### Web UI Access
```bash
$ ccr ui
# Opens browser to configuration UI at http://localhost:3456
```

### Service Management
```bash
# Restart service
ccr restart
# Response: Service stopped, starting in background

# Stop service  
ccr stop

# Verify running
ccr status
```

---

## 8. BUILT-IN TRANSFORMERS

### Available Transformers

#### Provider-Specific
- **Anthropic**: Preserves original request/response parameters
- **deepseek**: DeepSeek API request/response adaptation
- **gemini**: Google Gemini API formatting
- **openrouter**: OpenRouter API with provider routing
- **groq**: Groq API adaptation
- **vertex-gemini**: VertexAI Gemini authentication and formatting (KEY FOR VERTEXAI)
- **vertex-claude**: VertexAI Claude authentication (if available)

#### Processing Transformers
- **tooluse**: Optimizes tool usage via `tool_choice`
- **maxtoken**: Sets specific `max_tokens` value
- **sampling**: Processes sampling parameters (temperature, top_p, top_k, repetition_penalty)
- **enhancetool**: Adds error tolerance to tool calls
- **cleancache**: Clears `cache_control` field
- **reasoning**: Processes `reasoning_content` field
- **maxcompletiontokens**: Sets max completion tokens
- **streamoptions**: Stream configuration

#### Specialized/Experimental
- **gemini-cli**: Unofficial Gemini CLI support
- **qwen-cli**: Qwen3-coder-plus model support
- **rovo-cli**: Atlassian Rovo Dev CLI support
- **chutes-glm**: GLM 4.5 model support

### Transformer Configuration

#### Global Transformer (All Models)
```json
{
  "name": "provider",
  "transformer": { "use": ["transformername"] }
}
```

#### Model-Specific Transformer
```json
{
  "name": "provider",
  "transformer": {
    "use": ["global-transformer"],
    "specific-model": { "use": ["model-specific-transformer"] }
  }
}
```

#### With Options
```json
{
  "transformer": {
    "use": [
      ["maxtoken", { "max_tokens": 16384 }],
      "enhancetool"
    ]
  }
}
```

---

## 9. PROVIDERS & ROUTING

### Supported Providers (Full List)

#### Commercial LLM Services
1. **OpenRouter**: Multi-provider routing
   ```json
   {
     "name": "openrouter",
     "api_base_url": "https://openrouter.ai/api/v1/chat/completions",
     "api_key": "sk-xxx"
   }
   ```

2. **DeepSeek**: DeepSeek models
   ```json
   {
     "name": "deepseek",
     "api_base_url": "https://api.deepseek.com/chat/completions",
     "api_key": "sk-xxx"
   }
   ```

3. **Google Gemini**: Direct Google API
   ```json
   {
     "name": "gemini",
     "api_base_url": "https://generativelanguage.googleapis.com/v1beta/models/",
     "api_key": "sk-xxx"
   }
   ```

4. **Google VertexAI**: Enterprise Google Cloud
   ```json
   {
     "name": "vertex-gemini",
     "api_base_url": "https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/",
     "api_key": "${VERTEX_AI_API_KEY}"
   }
   ```

5. **OpenAI**: ChatGPT models
   - Can use via OpenRouter

6. **Ollama**: Local models
   ```json
   {
     "name": "ollama",
     "api_base_url": "http://localhost:11434/v1/chat/completions",
     "api_key": "ollama"
   }
   ```

#### Chinese LLM Services
- **ModelScope**: Qwen models
- **DashScope**: Aliyun Qwen
- **Volcengine**: Bytedance models
- **SiliconFlow**: Kimi and other models
- **AIHubMix**: Multi-provider hub

### Router Configuration

#### Default Routing
```json
{
  "Router": {
    "default": "provider-name,model-name"
  }
}
```

#### Scenario-Based Routing
```json
{
  "Router": {
    "default": "primary-provider,primary-model",
    "background": "cheap-provider,cheap-model",
    "think": "powerful-provider,reasoning-model",
    "longContext": "context-provider,long-model",
    "longContextThreshold": 60000,
    "webSearch": "search-provider,search-model",
    "image": "image-provider,image-model"
  }
}
```

#### Dynamic Model Switching
In Claude Code, use `/model` command:
```
/model provider_name,model_name
/model openrouter,anthropic/claude-3.5-sonnet
```

---

## 10. CUSTOM ROUTER SCRIPTS

### Custom Router Setup

Create file at `~/.claude-code-router/custom-router.js`:

```javascript
module.exports = async function router(req, config) {
  // req.body contains the full request object
  const userMessage = req.body.messages.find((m) => m.role === "user")?.content;
  
  // Route based on custom logic
  if (userMessage && userMessage.includes("explain code")) {
    return "openrouter,anthropic/claude-3.5-sonnet";
  }
  
  if (userMessage && userMessage.includes("quick answer")) {
    return "ollama,qwen2.5-coder:latest";
  }
  
  // Return null to use default router
  return null;
};
```

### Reference in Config
```json
{
  "CUSTOM_ROUTER_PATH": "/Users/username/.claude-code-router/custom-router.js"
}
```

### Router Request Object
- `req.body.messages`: Full message history
- `req.body.model`: Requested model
- `req.body.tools`: Tool definitions
- `config`: Complete application config

---

## 11. AGENTAPI INTEGRATION WITH CCROUTER

### AgentAPI Agent Types Supporting CCRouter

From `/lib/msgfmt/msgfmt.go` and `/cmd/server/server.go`:

```go
const (
    AgentTypeClaude   AgentType = "claude"
    AgentTypeGoose    AgentType = "goose"
    AgentTypeAider    AgentType = "aider"
    AgentTypeCodex    AgentType = "codex"
    AgentTypeGemini   AgentType = "gemini"
    AgentTypeCopilot  AgentType = "copilot"
    AgentTypeAmp      AgentType = "amp"
    AgentTypeCursor   AgentType = "cursor"
    AgentTypeAuggie   AgentType = "auggie"
    AgentTypeAmazonQ  AgentType = "amazonq"
    AgentTypeOpencode AgentType = "opencode"
    AgentTypeWarp     AgentType = "warp"
    AgentTypeDroid    AgentType = "droid"
    AgentTypeCCRouter AgentType = "ccrouter"  // NEW
    AgentTypeCustom   AgentType = "custom"
)
```

### Agent Type Aliases
```go
"ccrouter": AgentTypeCCRouter
"ccr":      AgentTypeCCRouter
```

### Usage with AgentAPI

```bash
# Start AgentAPI with CCRouter
./agentapi server --type=ccrouter -- ccr code

# Alternative alias
./agentapi server --type=ccr -- ccr code
```

### Message Formatting for CCRouter

CCRouter uses generic message formatting (same as Claude, Goose, Aider):
- Removes echoed user input
- Removes message box decorations
- Trims empty lines
- Preserves agent response content

---

## 12. MULTI-TENANT & FASTMCP ARCHITECTURE

### Multi-Tenant AgentAPI Features

From `/MULTITENANT.md`:

#### Core Components
1. **Session Management**: Isolated user sessions with separate workspaces
2. **MCP Integration**: Model Context Protocol support with OAuth flows
3. **System Prompt Management**: Hierarchical prompt configuration
4. **Multi-tenant Database**: Supabase PostgreSQL with RLS
5. **Container Orchestration**: Docker-based deployment

#### Database Tables
- `organizations`: Organization management
- `users`: User profiles
- `user_sessions`: Isolated sessions
- `mcp_configs`: MCP configurations
- `system_prompts`: System prompt configs
- `audit_logs`: Compliance logging

#### Security Features
- Row-Level Security (RLS) policies
- Audit logging for compliance
- Credential management
- Session isolation

#### API Endpoints
```
POST   /api/v1/sessions                  # Create session
GET    /api/v1/sessions/{id}             # Get session
DELETE /api/v1/sessions/{id}             # Terminate session
GET    /api/v1/sessions                  # List sessions
GET    /api/v1/mcps                      # List MCPs
POST   /api/v1/mcps                      # Add MCP
PUT    /api/v1/mcps/{id}                 # Update MCP
DELETE /api/v1/mcps/{id}                 # Remove MCP
```

### MCP Types Supported
1. **HTTP MCPs**: REST API-based
2. **SSE MCPs**: Server-Sent Events
3. **Stdio MCPs**: Command-line based

### OAuth Integration
1. Frontend initiates OAuth flow
2. User completes authentication
3. Credentials stored in Supabase
4. MCP connects with credentials
5. Automatic token refresh

---

## 13. STATUS LINE & MONITORING (BETA)

### Status Line Feature

Enable in UI for runtime monitoring:

```bash
ccr ui
# Enable "statusline" option in configuration
```

### Status Information Displayed
- Model routing decisions
- Request processing
- Performance metrics
- API call status

---

## 14. GITHUB ACTIONS INTEGRATION

### Example Workflow

```yaml
name: Claude Code with CCRouter

on:
  issue_comment:
    types: [created]

jobs:
  claude:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Router
        run: |
          curl -fsSL https://bun.sh/install | bash
          mkdir -p $HOME/.claude-code-router
          cat << 'EOF' > $HOME/.claude-code-router/config.json
          {
            "log": true,
            "NON_INTERACTIVE_MODE": true,
            "OPENAI_API_KEY": "${{ secrets.OPENAI_API_KEY }}",
            "OPENAI_BASE_URL": "https://api.deepseek.com",
            "OPENAI_MODEL": "deepseek-chat"
          }
          EOF
      
      - name: Start Router
        run: |
          nohup ~/.bun/bin/bunx @musistudio/claude-code-router@1.0.58 start &
      
      - name: Run Claude Code
        uses: anthropics/claude-code-action@beta
        env:
          ANTHROPIC_BASE_URL: http://localhost:3456
        with:
          anthropic_api_key: "any-string-is-ok"
```

### Key Settings
- `NON_INTERACTIVE_MODE: true` for CI/CD
- `ANTHROPIC_BASE_URL`: Route to CCRouter
- `CI=true`, `FORCE_COLOR=0` set automatically

---

## 15. COMMON CONFIGURATIONS

### Example 1: VertexAI Only

```json
{
  "PORT": 3456,
  "LOG": true,
  "Providers": [
    {
      "name": "vertex-gemini",
      "api_base_url": "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/",
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

### Example 2: Multi-Provider with VertexAI

```json
{
  "PORT": 3456,
  "Providers": [
    {
      "name": "vertex-gemini",
      "api_base_url": "https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/",
      "api_key": "${VERTEX_AI_API_KEY}",
      "models": ["gemini-1.5-pro", "gemini-1.5-flash"],
      "transformer": {"use": ["vertex-gemini"]}
    },
    {
      "name": "deepseek",
      "api_base_url": "https://api.deepseek.com/chat/completions",
      "api_key": "${DEEPSEEK_API_KEY}",
      "models": ["deepseek-chat", "deepseek-reasoner"],
      "transformer": {"use": ["deepseek"]}
    },
    {
      "name": "ollama",
      "api_base_url": "http://localhost:11434/v1/chat/completions",
      "api_key": "ollama",
      "models": ["qwen2.5-coder:latest"]
    }
  ],
  "Router": {
    "default": "vertex-gemini,gemini-1.5-pro",
    "background": "ollama,qwen2.5-coder:latest",
    "think": "deepseek,deepseek-reasoner",
    "longContext": "vertex-gemini,gemini-1.5-pro"
  }
}
```

### Example 3: With Custom Router

```json
{
  "PORT": 3456,
  "CUSTOM_ROUTER_PATH": "/Users/username/.claude-code-router/custom-router.js",
  "Providers": [
    {
      "name": "vertex-gemini",
      "api_base_url": "https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/",
      "api_key": "${VERTEX_AI_API_KEY}",
      "models": ["gemini-1.5-pro"]
    }
  ],
  "Router": {
    "default": "vertex-gemini,gemini-1.5-pro"
  }
}
```

---

## 16. TROUBLESHOOTING & ERROR HANDLING

### Common Issues

#### Issue: "Missing model in request body"
- **Cause**: Invalid request format to CCRouter
- **Solution**: Ensure request includes model field

#### Issue: Service won't start
```bash
ccr restart
# Check logs
tail -f ~/.claude-code-router/logs/ccr-*.log
```

#### Issue: Status line JSON parse error
```bash
# Clear configuration and reinitialize
rm ~/.claude-code-router/config.json
ccr restart
ccr ui
```

#### Issue: Environment variable interpolation not working
```bash
# Verify variable is exported
export VERTEX_AI_API_KEY="your-key"
# Restart router
ccr restart
```

### Log Locations
- Server logs: `~/.claude-code-router/logs/ccr-*.log`
- App logs: `~/.claude-code-router/claude-code-router.log`
- PID file: `~/.claude-code-router/.claude-code-router.pid`

---

## 17. SECURITY CONSIDERATIONS

### Best Practices

1. **API Key Management**
   - Use environment variables, not hardcoded in config.json
   - Use `$VAR_NAME` or `${VAR_NAME}` syntax for interpolation
   - Restrict file permissions: `chmod 600 ~/.claude-code-router/config.json`

2. **Network Security**
   - Default: Listen on 127.0.0.1 only (unless APIKEY set)
   - Use APIKEY for network exposure
   - Use PROXY_URL for VPN/proxy access

3. **API Authentication**
   - Set APIKEY in config for requests from untrusted sources
   - Require `Authorization: Bearer <key>` or `x-api-key` header
   - Rotate keys regularly

4. **Access Control**
   - Restrict config.json file permissions
   - Use SOC2 compliance features in enterprise
   - Implement RLS in database layer

---

## 18. PERFORMANCE & OPTIMIZATION

### Token Counting
- Built-in tiktoken support for accurate token counting
- Supports long context thresholds (default 60K tokens)
- Enables intelligent model routing based on token count

### Request/Response Transformation
- Transformers add overhead but enable compatibility
- Multiple transformer chain support
- Optional caching for repeated requests

### Streaming Support
- SSE streaming for real-time responses
- Maintains compatibility with Claude Code streaming
- Lower latency with streaming than buffered responses

---

## 19. COMPLETE FEATURE MATRIX

| Feature | Support | Notes |
|---------|---------|-------|
| Model Routing | Full | Dynamic via `/model` command |
| VertexAI Support | Full | vertex-gemini transformer |
| Multi-Provider | Full | 8+ built-in providers |
| Custom Transformers | Full | JavaScript plugin system |
| Custom Router | Full | Per-request routing logic |
| GitHub Actions | Full | CI/CD integration ready |
| MCP Integration | Full | HTTP/SSE/Stdio types |
| OAuth Support | Full | For MCP authentication |
| Streaming | Full | SSE streaming support |
| Token Counting | Full | Tiktoken-based |
| Web UI | Full | Configuration management |
| Status Line | Beta | Runtime monitoring |
| Logging | Full | Multiple log levels/systems |
| Non-Interactive Mode | Full | CI/CD environment support |

---

## 20. QUICK REFERENCE

### Installation & Setup
```bash
# Install
npm install -g @musistudio/claude-code-router

# Verify
ccr --version

# Start
ccr start

# Configure
ccr ui

# Check status
ccr status
```

### VertexAI Setup
```bash
# Set environment variable
export VERTEX_AI_API_KEY="your-gcp-credentials"

# Edit config.json with vertex-gemini provider
# Restart service
ccr restart

# Use with Claude Code
export ANTHROPIC_BASE_URL=http://localhost:3456
ccr code
```

### API Endpoint Format
```
HTTP POST http://127.0.0.1:3456/v1/messages
Content-Type: application/json

{
  "model": "provider-name,model-name",
  "messages": [...],
  "stream": true/false
}
```

---

## 21. REFERENCES & DOCUMENTATION

### Official Resources
- **GitHub**: https://github.com/musistudio/claude-code-router
- **npm**: https://www.npmjs.com/package/@musistudio/claude-code-router
- **README**: /opt/homebrew/lib/node_modules/@musistudio/claude-code-router/README.md

### Local Documentation
- **AgentAPI**: /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/README.md
- **Multi-Tenant**: /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/MULTITENANT.md
- **Agents Guide**: /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/AGENTS.md

### Configuration Files
- **Config Location**: ~/.claude-code-router/config.json
- **Example**: /opt/homebrew/lib/node_modules/@musistudio/claude-code-router/custom-router.example.js

---

## Summary

CCRouter is a comprehensive LLM routing platform that:

1. **Enables Model Flexibility**: Route between 8+ LLM providers including VertexAI
2. **Maintains Compatibility**: OpenAI-compatible API with request transformation
3. **Provides Control**: Dynamic routing via configuration and custom scripts
4. **Scales Efficiently**: Multi-tenant support with MCP integration
5. **Ensures Security**: API key management, authentication, audit logging
6. **Integrates Seamlessly**: Works with AgentAPI and Claude Code without modification

The feature set includes VertexAI support via the vertex-gemini transformer, comprehensive provider support, custom router scripting, GitHub Actions integration, and enterprise-grade security features.


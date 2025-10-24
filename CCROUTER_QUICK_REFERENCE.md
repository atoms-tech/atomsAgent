# CCRouter Quick Reference Guide

## Installation Status

```
Package: @musistudio/claude-code-router v1.0.58
Binary: /opt/homebrew/bin/ccr
Config: ~/.claude-code-router/config.json
Status: Running on port 3456
```

## Essential Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `ccr status` | Check service status | `ccr status` |
| `ccr start` | Start service | `ccr start` |
| `ccr restart` | Restart service | `ccr restart` |
| `ccr stop` | Stop service | `ccr stop` |
| `ccr code` | Run Claude Code with routing | `ccr code` |
| `ccr ui` | Open web configuration UI | `ccr ui` |
| `ccr --version` | Show version | `ccr -v` |

## Configuration Quick Setup

### 1. Access Configuration
```bash
# Open web UI
ccr ui

# Or edit directly
nano ~/.claude-code-router/config.json
```

### 2. Add VertexAI Provider
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
    }
  ],
  "Router": {
    "default": "vertex-gemini,gemini-1.5-pro"
  }
}
```

### 3. Set Environment Variables
```bash
export VERTEX_AI_API_KEY="your-gcp-service-account-key"
```

### 4. Restart Service
```bash
ccr restart
```

### 5. Use with Claude Code
```bash
export ANTHROPIC_BASE_URL=http://localhost:3456
export ANTHROPIC_API_KEY=any-string
ccr code
```

## VertexAI Configuration Details

### Required Fields
- **Provider Name**: `vertex-gemini`
- **API Base URL**: `https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/`
- **API Key**: Google Cloud service account credentials (use `${VERTEX_AI_API_KEY}`)
- **Transformer**: `["vertex-gemini"]`

### Available Models
- `gemini-1.5-pro` - Advanced reasoning
- `gemini-1.5-flash` - Fast responses

### Model Routing Options
```json
"Router": {
  "default": "vertex-gemini,gemini-1.5-pro",
  "background": "vertex-gemini,gemini-1.5-flash",
  "think": "vertex-gemini,gemini-1.5-pro",
  "longContext": "vertex-gemini,gemini-1.5-pro"
}
```

## Multi-Provider Example

```json
{
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
      "models": ["deepseek-chat"],
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
    "think": "deepseek,deepseek-chat",
    "longContext": "vertex-gemini,gemini-1.5-pro"
  }
}
```

## Key Concepts

### Providers
Define LLM services (VertexAI, DeepSeek, Ollama, etc.)

### Transformers
Transform requests/responses for compatibility:
- `vertex-gemini` - VertexAI formatting
- `deepseek` - DeepSeek API formatting
- `gemini` - Google Gemini formatting
- `tooluse` - Tool usage optimization
- `maxtoken` - Set token limits

### Router
Determine which model/provider to use:
- `default` - General tasks
- `background` - Background jobs
- `think` - Reasoning tasks
- `longContext` - Large contexts (>60K tokens)
- `webSearch` - Web search tasks
- `image` - Image tasks

## Dynamic Model Selection

In Claude Code, switch models with:
```
/model provider-name,model-name
/model vertex-gemini,gemini-1.5-pro
/model deepseek,deepseek-chat
```

## Troubleshooting

### Service Won't Start
```bash
# Check logs
tail -f ~/.claude-code-router/logs/ccr-*.log

# Clear config and reinitialize
rm ~/.claude-code-router/config.json
ccr restart
```

### Environment Variable Not Working
```bash
# Verify export
echo $VERTEX_AI_API_KEY

# Set and verify
export VERTEX_AI_API_KEY="your-key"
ccr restart
```

### API Connection Issues
```bash
# Test endpoint
curl -X POST http://127.0.0.1:3456/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model": "test"}'

# Should respond with error about missing model (API is working)
```

## Configuration Locations

| File/Directory | Purpose |
|---|---|
| `~/.claude-code-router/config.json` | Main configuration |
| `~/.claude-code-router/logs/` | Server logs |
| `~/.claude-code-router/plugins/` | Custom transformers |
| `~/.claude-code-router/.claude-code-router.pid` | Process ID file |

## Log Levels

Set in config.json:
```json
{
  "LOG_LEVEL": "debug"  // fatal, error, warn, info, debug, trace
}
```

## Security

### API Key Protection
```json
{
  "api_key": "${VERTEX_AI_API_KEY}"  // Use env var, never hardcode
}
```

### Network Security
- Default: Listens on 127.0.0.1 only
- Set APIKEY for network access
- Client must provide `Authorization: Bearer <key>`

### File Permissions
```bash
chmod 600 ~/.claude-code-router/config.json
```

## Integration with AgentAPI

### Start with CCRouter
```bash
./agentapi server --type=ccrouter -- ccr code
```

### Environment Setup
```bash
export ANTHROPIC_BASE_URL=http://localhost:3456
export ANTHROPIC_API_KEY=any-string
./agentapi server -- claude
```

## Performance Tuning

### Token Limit for Long Contexts
```json
{
  "Router": {
    "longContextThreshold": 60000
  },
  "transformer": {
    "use": [
      ["maxtoken", {"max_tokens": 65536}]
    ]
  }
}
```

### API Timeout
```json
{
  "API_TIMEOUT_MS": 600000
}
```

## CI/CD Integration

```bash
# Set non-interactive mode
export NON_INTERACTIVE_MODE=true

# Configure
mkdir -p ~/.claude-code-router
cat > ~/.claude-code-router/config.json << 'JSON'
{
  "NON_INTERACTIVE_MODE": true,
  "Providers": [...]
}
JSON

# Start service
ccr start

# Use with GitHub Actions
export ANTHROPIC_BASE_URL=http://localhost:3456
```

## Useful Resources

- Configuration: `~/.claude-code-router/config.json`
- Documentation: `/opt/homebrew/lib/node_modules/@musistudio/claude-code-router/README.md`
- Example Router: `/opt/homebrew/lib/node_modules/@musistudio/claude-code-router/custom-router.example.js`
- AgentAPI Integration: `/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/MULTITENANT.md`

## Common Providers

| Provider | Base URL | Transformer |
|----------|----------|-------------|
| VertexAI | `https://us-central1-aiplatform.googleapis.com/v1/...` | `vertex-gemini` |
| DeepSeek | `https://api.deepseek.com/chat/completions` | `deepseek` |
| OpenRouter | `https://openrouter.ai/api/v1/chat/completions` | `openrouter` |
| Gemini | `https://generativelanguage.googleapis.com/v1beta/models/` | `gemini` |
| Ollama | `http://localhost:11434/v1/chat/completions` | (none) |

---

**Quick Start Summary:**
1. `ccr restart` - Ensure service is running
2. `ccr ui` - Configure providers and routing
3. Set environment variables (e.g., `VERTEX_AI_API_KEY`)
4. `ccr restart` - Apply changes
5. `export ANTHROPIC_BASE_URL=http://localhost:3456`
6. Use with Claude Code or AgentAPI

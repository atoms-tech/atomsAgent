# Atoms Agent Dev UI Guide

Standalone development interface for testing chat completions, models, and MCP servers.

## 🚀 Quick Start

### Installation

```bash
# Install with dev dependencies (includes Gradio)
cd atomsAgent
uv pip install -e '.[dev]'

# Or with pip
pip install -e '.[dev]'
```

### Launch Dev UI

```bash
# Launch on default port (7860)
atoms-agent dev-ui launch

# Launch with custom port
atoms-agent dev-ui launch --port 8080

# Create shareable public link
atoms-agent dev-ui launch --share

# Custom host and port
atoms-agent dev-ui launch --host 0.0.0.0 --port 7860
```

The UI will be available at `http://localhost:7860` (or your custom port).

---

## 💬 Chat Commands

### Interactive Chat (CLI)

Start an interactive chat session in your terminal:

```bash
# Basic interactive chat
atoms-agent chat interactive

# With custom model
atoms-agent chat interactive --model claude-3-5-haiku-20241022

# With custom system prompt
atoms-agent chat interactive --system-prompt "You are a Python expert"

# With custom temperature
atoms-agent chat interactive --temperature 1.2

# Disable streaming
atoms-agent chat interactive --no-stream
```

**Controls:**
- Type your message and press Enter
- Type `exit`, `quit`, or `q` to end session
- Press `Ctrl+C` to exit

### Single Message

Send a single message and get a response:

```bash
# Basic usage
atoms-agent chat once "Explain quantum computing"

# With options
atoms-agent chat once "Write a Python function to sort a list" \
  --model claude-3-5-sonnet-20241022 \
  --temperature 0.5 \
  --system-prompt "You are a code expert"
```

---

## 🧪 Test Commands

### Test Completion

Test non-streaming chat completion:

```bash
atoms-agent test completion --prompt "Hello, world\!"

# With custom model
atoms-agent test completion \
  --prompt "Explain machine learning" \
  --model claude-3-5-sonnet-20241022 \
  --temperature 0.7
```

**Output includes:**
- Response content
- Token usage (prompt, completion, total)
- Latency
- Finish reason

### Test Streaming

Test streaming chat completion:

```bash
atoms-agent test streaming --prompt "Count from 1 to 10"

# With custom model
atoms-agent test streaming \
  --prompt "Write a short story" \
  --model claude-3-5-sonnet-20241022
```

**Output includes:**
- Streamed response (real-time)
- Total chunks received
- Total characters
- Latency
- Characters per second

### Test Models

List available models:

```bash
atoms-agent test models
```

**Output:**
- Model ID
- Provider
- Context length

---

## 🎨 Dev UI Features

### Chat Tab

**Features:**
- Real-time streaming responses
- Message history
- Retry last message
- Clear conversation
- Copy messages

**Settings:**
- Model selector
- Temperature (0-2)
- Max tokens (100-4096)
- Top-p (0-1)
- System prompt editor

**Stats Display:**
- Latency (ms)
- Chunks received
- Characters generated
- Characters per second

### Settings Tab

**Model Management:**
- View available models
- Refresh model list
- Test API connection
- Configure API URL and key

**System Prompts:**
- Save custom prompts
- Load saved prompts
- Preview prompts
- Pre-configured templates:
  - Default Assistant
  - Code Expert
  - Creative Writer
  - Technical Explainer

### MCP Servers Tab

**Features:**
- View configured MCP servers
- Add new servers
- Test server connections
- View server tools and resources

**Note:** Full MCP management requires CLI commands with proper authentication.

---

## 🔧 Configuration

### API Configuration

The client reads configuration from:

1. **Environment variables:**
   ```bash
   export AGENTAPI_URL="http://localhost:3284"
   export AGENTAPI_KEY="your-api-key"
   ```

2. **Config file** (if using atomsAgent config system)

3. **Command-line options** (for dev-ui)

### Default Values

- **Base URL:** `http://localhost:3284`
- **API Key:** None (optional)
- **Timeout:** 60 seconds
- **Default Model:** `claude-3-5-sonnet-20241022`

---

## 📋 Command Reference

### Dev UI Commands

```bash
atoms-agent dev-ui launch [OPTIONS]

Options:
  --port INTEGER      Port to run on [default: 7860]
  --host TEXT         Host to bind to [default: 127.0.0.1]
  --share            Create public shareable link
  --debug            Enable debug mode
  --help             Show help message
```

### Chat Commands

```bash
atoms-agent chat interactive [OPTIONS]
atoms-agent chat once PROMPT [OPTIONS]

Options:
  --model TEXT              Model to use [default: claude-3-5-sonnet-20241022]
  --system-prompt TEXT      System prompt [default: "You are a helpful assistant."]
  --temperature FLOAT       Sampling temperature (0-2) [default: 0.7]
  --stream/--no-stream     Enable streaming [default: stream]
  --max-tokens INTEGER     Maximum tokens to generate
  --help                   Show help message
```

### Test Commands

```bash
atoms-agent test completion [OPTIONS]
atoms-agent test streaming [OPTIONS]
atoms-agent test models

Options:
  --prompt TEXT         Test prompt (required for completion/streaming)
  --model TEXT          Model to test [default: claude-3-5-sonnet-20241022]
  --temperature FLOAT   Sampling temperature [default: 0.7]
  --help               Show help message
```

---

## 🤝 Sharing with Team

### Option 1: Public Link (Gradio Share)

```bash
atoms-agent dev-ui launch --share
```

This creates a temporary public URL (valid for 72 hours) that you can share with teammates.

**Example output:**
```
Running on local URL:  http://127.0.0.1:7860
Running on public URL: https://abc123.gradio.live
```

### Option 2: Network Access

```bash
atoms-agent dev-ui launch --host 0.0.0.0 --port 7860
```

Team members on the same network can access at `http://YOUR_IP:7860`

### Option 3: SSH Tunnel

```bash
# On server
atoms-agent dev-ui launch --port 7860

# On local machine
ssh -L 7860:localhost:7860 user@server
```

Then access at `http://localhost:7860`

---

## 🐛 Troubleshooting

### Gradio Not Installed

```
Error: Gradio not installed.
```

**Solution:**
```bash
uv pip install -e '.[dev]'
# or
pip install gradio sseclient-py markdown
```

### Connection Refused

```
Error: Connection refused
```

**Solution:**
- Ensure AgentAPI server is running
- Check `AGENTAPI_URL` environment variable
- Test connection: `atoms-agent test models`

### Import Errors

```
ModuleNotFoundError: No module named 'atomsAgent.services.chat_client'
```

**Solution:**
```bash
# Reinstall in editable mode
cd atomsAgent
uv pip install -e .
```

### Port Already in Use

```
Error: Address already in use
```

**Solution:**
```bash
# Use different port
atoms-agent dev-ui launch --port 7861
```

---

## 📚 Examples

### Example 1: Quick Test

```bash
# Test if everything works
atoms-agent test models
atoms-agent test completion --prompt "Hello\!"
```

### Example 2: Interactive Session

```bash
# Start chat with code expert prompt
atoms-agent chat interactive \
  --system-prompt "You are an expert Python programmer" \
  --temperature 0.5
```

### Example 3: Team Demo

```bash
# Launch shareable UI for team demo
atoms-agent dev-ui launch --share
```

### Example 4: Local Development

```bash
# Launch UI for local testing
atoms-agent dev-ui launch --port 7860 --debug
```

---

## 🎯 Best Practices

1. **Use streaming for long responses** - Better UX and faster perceived latency
2. **Adjust temperature based on task:**
   - 0.0-0.3: Factual, deterministic tasks
   - 0.5-0.7: Balanced creativity and consistency
   - 0.8-2.0: Creative, varied outputs
3. **Set appropriate max_tokens** - Prevents runaway costs
4. **Use system prompts** - Better control over assistant behavior
5. **Test with CLI first** - Faster iteration than UI

---

## 🔐 Security Notes

- **API Keys:** Never commit API keys to version control
- **Public Sharing:** Be cautious with `--share` flag - creates public URL
- **Network Access:** Use `--host 0.0.0.0` only on trusted networks
- **Authentication:** MCP operations require proper org/user context

---

## 📝 Development

### Adding New Features

1. **Chat Client:** Modify `src/atomsAgent/services/chat_client.py`
2. **CLI Commands:** Add to `src/atomsAgent/cli/`
3. **UI Components:** Modify `src/atomsAgent/ui/`

### Running Tests

```bash
pytest tests/test_cli.py
pytest tests/test_api_endpoints.py
```

---

## 🆘 Support

- **Issues:** Report bugs on GitHub
- **Documentation:** See main README.md
- **CLI Help:** `atoms-agent --help`
- **Command Help:** `atoms-agent COMMAND --help`

---

**Built with:**
- [Gradio](https://gradio.app) - UI framework
- [Typer](https://typer.tiangolo.com) - CLI framework
- [Rich](https://rich.readthedocs.io) - Terminal formatting
- [HTTPX](https://www.python-httpx.org) - HTTP client

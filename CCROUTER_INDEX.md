# CCRouter Documentation Index

This documentation provides a comprehensive guide to CCRouter (Claude Code Router) version 1.0.58 and its integration with the AgentAPI project.

## Documentation Files

### 1. CCROUTER_QUICK_REFERENCE.md (7.2 KB)
**Quick start and reference guide for immediate use**

Essential for:
- Getting started quickly
- Common commands and workflows
- VertexAI setup in 5 steps
- Troubleshooting quick fixes
- Configuration examples
- Common provider configurations

Start here if you just need to get CCRouter working with VertexAI.

### 2. CCROUTER_COMPLETE_ANALYSIS.md (25 KB)
**Comprehensive technical documentation and API reference**

Covers all aspects:
1. CLI Commands & Options (8 command types)
2. Configuration Files & Structure (detailed schema)
3. VertexAI Model Routing (transformer setup)
4. NPM Package Structure (dependencies & files)
5. Claude Code Communication Protocol (HTTP/streaming)
6. Environment Variables & Configuration (21 variables)
7. Testing CCRouter Commands (status checks, API tests)
8. Built-in Transformers (15+ transformers listed)
9. Providers & Routing (8+ provider configurations)
10. Custom Router Scripts (JavaScript customization)
11. AgentAPI Integration (agent types, message formatting)
12. Multi-Tenant & FastMCP Architecture (enterprise features)
13. Status Line & Monitoring (beta features)
14. GitHub Actions Integration (CI/CD setup)
15. Common Configurations (3 production examples)
16. Troubleshooting & Error Handling (common issues & logs)
17. Security Considerations (best practices)
18. Performance & Optimization (token counting, streaming)
19. Feature Matrix (complete feature support table)
20. Quick Reference (installation, VertexAI setup, API format)
21. References & Documentation (external resources)

Read this for in-depth understanding of all features and capabilities.

## Quick Navigation

### By Use Case

#### Getting Started (5 minutes)
1. Read: CCROUTER_QUICK_REFERENCE.md - "Quick Start Summary"
2. Run: `ccr restart && ccr status`
3. Configure: `ccr ui`

#### Setting Up VertexAI (10 minutes)
1. Read: CCROUTER_QUICK_REFERENCE.md - "VertexAI Configuration Details"
2. Configure: Add provider to config.json
3. Environment: `export VERTEX_AI_API_KEY=...`
4. Restart: `ccr restart`

#### Integration with AgentAPI
1. Read: CCROUTER_COMPLETE_ANALYSIS.md - Section 11
2. Read: MULTITENANT.md (in project)
3. Start: `./agentapi server --type=ccrouter -- ccr code`

#### Custom Router Development
1. Read: CCROUTER_COMPLETE_ANALYSIS.md - Section 10
2. Create: `~/.claude-code-router/custom-router.js`
3. Example: See custom-router.example.js in CCRouter package

#### Production Deployment
1. Read: CCROUTER_COMPLETE_ANALYSIS.md - Sections 12-17
2. Setup: GitHub Actions (Section 14)
3. Secure: API keys, permissions, logging
4. Monitor: Status line, logs, metrics

### By Topic

| Topic | Quick Ref | Complete Ref |
|-------|-----------|---|
| CLI Commands | Commands table | Section 1 |
| Configuration | Setup steps | Section 2 |
| VertexAI | Details section | Section 3 |
| Providers | Examples | Section 9 |
| Transformers | Key concepts | Section 8 |
| Routing | Examples | Section 9 |
| AgentAPI | Integration steps | Section 11 |
| Security | File permissions | Section 17 |
| Troubleshooting | Common issues | Section 16 |
| CI/CD | GitHub Actions | Section 14 |
| Performance | Tuning | Section 18 |

## Installation Verification

```bash
# Check version (currently 1.0.58)
ccr --version

# Check status (should show running on port 3456)
ccr status

# View current configuration
cat ~/.claude-code-router/config.json

# View recent logs
tail -f ~/.claude-code-router/logs/ccr-*.log
```

## Key Findings

### Supported Features
- 8+ LLM providers (OpenRouter, DeepSeek, Gemini, VertexAI, Ollama, etc.)
- 15+ built-in transformers for API compatibility
- Custom JavaScript router for intelligent routing
- Dynamic model switching with `/model` command
- Streaming support (SSE)
- Token counting with tiktoken
- Multi-tenant support with PostgreSQL RLS
- MCP integration (HTTP/SSE/Stdio)
- GitHub Actions integration
- Web UI configuration
- Status line monitoring (beta)

### VertexAI Integration
- Transformer: `vertex-gemini`
- Models: gemini-1.5-pro, gemini-1.5-flash
- Authentication: Google Cloud service account
- Base URL: `https://us-central1-aiplatform.googleapis.com/v1/projects/{PROJECT_ID}/locations/us-central1/publishers/google/models/`
- Environment variable: `VERTEX_AI_API_KEY`

### AgentAPI Integration
- Agent type: `ccrouter` or `ccr`
- Message formatting: Generic (same as Claude, Goose, Aider)
- Usage: `./agentapi server --type=ccrouter -- ccr code`
- Multi-tenant support with session isolation
- Support for MCP servers and system prompts

## File Locations

```
Installation:
  Binary: /opt/homebrew/bin/ccr
  Package: /opt/homebrew/lib/node_modules/@musistudio/claude-code-router/

Configuration:
  Config: ~/.claude-code-router/config.json
  Logs: ~/.claude-code-router/logs/
  Plugins: ~/.claude-code-router/plugins/

Documentation (this project):
  CCROUTER_INDEX.md (this file)
  CCROUTER_QUICK_REFERENCE.md
  CCROUTER_COMPLETE_ANALYSIS.md
  MULTITENANT.md (AgentAPI integration)
```

## Related Documentation

### AgentAPI Project Files
- **README.md** - AgentAPI overview and quickstart
- **AGENTS.md** - Agent types and development commands
- **MULTITENANT.md** - Multi-tenant architecture with CCRouter support
- **FASTMCP_INTEGRATION.md** - FastMCP 2.0 integration
- **openapi.json** - API schema

### External Resources
- **GitHub**: https://github.com/musistudio/claude-code-router
- **npm**: https://www.npmjs.com/package/@musistudio/claude-code-router
- **Official README**: /opt/homebrew/lib/node_modules/@musistudio/claude-code-router/README.md

## Command Reference

```bash
# Service Management
ccr start              # Start service
ccr stop               # Stop service
ccr restart            # Restart service
ccr status             # Check status

# Usage
ccr code               # Run Claude Code with routing
ccr ui                 # Open web configuration UI

# Information
ccr --version          # Show version
ccr --help             # Show help
```

## Configuration Checklist

Before using CCRouter with VertexAI:

- [ ] Verify installation: `ccr --version`
- [ ] Check service running: `ccr status`
- [ ] Set API key: `export VERTEX_AI_API_KEY=...`
- [ ] Edit config: `ccr ui` or `nano ~/.claude-code-router/config.json`
- [ ] Add vertex-gemini provider
- [ ] Set default router
- [ ] Restart service: `ccr restart`
- [ ] Test connection: `curl http://127.0.0.1:3456/status`
- [ ] Configure AgentAPI if using with it
- [ ] Test with Claude Code: `ccr code`

## Support & Troubleshooting

### Viewing Logs
```bash
# Server logs
tail -f ~/.claude-code-router/logs/ccr-*.log

# Application logs
tail -f ~/.claude-code-router/claude-code-router.log
```

### Common Commands for Debugging
```bash
# Test API endpoint
curl -X POST http://127.0.0.1:3456/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model": "test"}'

# Check environment variable
echo $VERTEX_AI_API_KEY

# View process
ps aux | grep ccr
```

### Reset Configuration
```bash
# Clear config and reinitialize
rm ~/.claude-code-router/config.json
ccr restart
ccr ui
```

## Summary

CCRouter is a comprehensive LLM routing platform that enables:

1. **Flexibility**: Route between multiple LLM providers including VertexAI
2. **Compatibility**: OpenAI-compatible API with automatic transformation
3. **Control**: Configuration-based and custom JavaScript routing
4. **Scale**: Multi-tenant support for enterprise deployments
5. **Integration**: Works seamlessly with AgentAPI and Claude Code
6. **Security**: API key management, authentication, audit logging

This documentation covers all aspects of installation, configuration, usage, troubleshooting, and integration with the AgentAPI project.

---

**Last Updated**: October 23, 2025
**CCRouter Version**: 1.0.58
**Status**: Running on port 3456

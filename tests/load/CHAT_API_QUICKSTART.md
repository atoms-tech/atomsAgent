# Chat API Load Testing - Quick Start

Get up and running with chat completions load testing in 5 minutes.

## Prerequisites

1. **K6 installed** (see installation below)
2. **AgentAPI server running** on `http://localhost:3284`
3. **Valid authentication token** (optional for testing)

## 1-Minute Install

### macOS
```bash
brew install k6
```

### Linux (Debian/Ubuntu)
```bash
curl -s https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x77C6C491D6AC1D69 | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

### Windows
```powershell
choco install k6
```

## Quick Test Run

### Option 1: Using the Helper Script (Recommended)

```bash
# Make script executable (first time only)
chmod +x tests/load/run-chat-test.sh

# Check prerequisites
./tests/load/run-chat-test.sh --check-only

# Run all tests
./tests/load/run-chat-test.sh

# Run specific scenario
./tests/load/run-chat-test.sh --scenario basic_load
```

### Option 2: Direct K6 Execution

```bash
# Basic run (default configuration)
k6 run tests/load/k6_chat_api_test.js

# With custom URL
k6 run --env BASE_URL=http://your-api.com tests/load/k6_chat_api_test.js

# With auth token
k6 run --env AUTH_TOKEN="Bearer your-token" tests/load/k6_chat_api_test.js

# Both URL and token
k6 run \
  --env BASE_URL=http://your-api.com \
  --env AUTH_TOKEN="Bearer your-token" \
  tests/load/k6_chat_api_test.js
```

## Quick Command Reference

```bash
# List available scenarios
./tests/load/run-chat-test.sh --list-scenarios

# Run individual scenarios (saves time)
./tests/load/run-chat-test.sh --scenario basic_load      # 2 min
./tests/load/run-chat-test.sh --scenario ramp_up         # 5 min
./tests/load/run-chat-test.sh --scenario stress_test     # 1 min
./tests/load/run-chat-test.sh --scenario sustained_load  # 10 min
./tests/load/run-chat-test.sh --scenario streaming_test  # 9 min

# Show help
./tests/load/run-chat-test.sh --help
```

## What Gets Tested

### Endpoints
- `POST /v1/chat/completions` (non-streaming)
- `POST /v1/chat/completions` (streaming SSE)

### Models
- gemini-1.5-pro
- gpt-4 / gpt-4-turbo
- claude-3-opus / claude-3-sonnet

### Request Types
- Short messages (< 20 words)
- Medium messages (20-50 words)
- Long messages (100+ words)

### Load Patterns
- **Basic:** 10 req/s steady
- **Ramp:** 0 → 100 req/s gradual
- **Stress:** 200 req/s spike
- **Sustained:** 50 req/s endurance
- **Streaming:** 1-50 concurrent streams

## Understanding Results

### Console Output (During Test)
```
✓ status is 200
✓ has response body
✓ content-type is json
✓ response is valid JSON
✓ has id field
✓ has choices array
✓ has message content
✓ has usage info
```

### Final Summary (After Test)
```
Overall Performance:
  Response Time (p95): 1456.89ms  ← Must be < 2000ms
  Error Rate: 0.23%               ← Must be < 5%
  Total Chat Requests: 125430

Success Rates:
  Overall Success Rate: 99.77%    ← Must be > 95%
  Streaming Success Rate: 99.65%
  Non-Streaming Success Rate: 99.82%
```

### Output Files
- `tests/load/chat_api_summary.html` - Visual report (open in browser)
- `tests/load/chat_api_summary.json` - Raw data (for automation)

## Performance Targets

| Metric | Target | Critical |
|--------|--------|----------|
| Overall p95 latency | < 2000ms | < 5000ms |
| Non-streaming p95 | < 1000ms | < 2000ms |
| Streaming p95 | < 2000ms | < 5000ms |
| Error rate | < 1% | < 5% |
| Success rate | > 99% | > 95% |
| Time to first chunk | < 500ms | < 1000ms |

## Common Use Cases

### 1. Pre-Deployment Validation
```bash
# Quick smoke test (2 minutes)
./tests/load/run-chat-test.sh --scenario basic_load
```

### 2. Capacity Planning
```bash
# Full ramp test (5 minutes)
./tests/load/run-chat-test.sh --scenario ramp_up
```

### 3. Breaking Point Analysis
```bash
# Stress test (1 minute)
./tests/load/run-chat-test.sh --scenario stress_test
```

### 4. Stability Testing
```bash
# Sustained load (10 minutes)
./tests/load/run-chat-test.sh --scenario sustained_load
```

### 5. Streaming Performance
```bash
# Streaming-specific test (9 minutes)
./tests/load/run-chat-test.sh --scenario streaming_test
```

## Environment Configuration

### Method 1: Environment Variables
```bash
export BASE_URL=http://localhost:3284
export AUTH_TOKEN="Bearer your-token-here"
./tests/load/run-chat-test.sh
```

### Method 2: Config File (Best for Regular Testing)
```bash
# Copy example
cp tests/load/.env.chat-test.example tests/load/.env.chat-test

# Edit with your values
nano tests/load/.env.chat-test

# Run (automatically loads config)
./tests/load/run-chat-test.sh
```

### Method 3: Command Line Arguments
```bash
./tests/load/run-chat-test.sh \
  --url http://api.example.com \
  --token "Bearer xyz123"
```

## Troubleshooting

### "Cannot reach API"
```bash
# Check if server is running
curl http://localhost:3284/status

# Or start the server
agentapi server -- claude
```

### "k6 command not found"
```bash
# Install k6 (see installation section above)
brew install k6  # macOS
```

### "High error rate"
```bash
# Check auth token is valid
curl -H "Authorization: Bearer your-token" \
  http://localhost:3284/v1/chat/completions

# Reduce load if server is overloaded
# Edit k6_chat_api_test.js and lower rate/maxVUs
```

### "Connection timeout"
```bash
# Increase timeout in test file
# Edit k6_chat_api_test.js, find timeout parameters
# Change from '30s' to '60s' or higher
```

## Advanced Usage

### Run Subset of Scenarios
```bash
# Only basic_load and streaming_test
k6 run \
  --include-scenario-in-stats basic_load,streaming_test \
  tests/load/k6_chat_api_test.js
```

### Save Results for Later Analysis
```bash
k6 run \
  --out json=my-test-results.json \
  tests/load/k6_chat_api_test.js
```

### Run from Different Machine
```bash
# From remote machine targeting your API
k6 run \
  --env BASE_URL=https://api.production.com \
  --env AUTH_TOKEN="Bearer prod-token" \
  tests/load/k6_chat_api_test.js
```

### Token Rotation Testing
```bash
# Test with multiple auth tokens
k6 run \
  --env AUTH_TOKENS="Bearer token1,Bearer token2,Bearer token3" \
  tests/load/k6_chat_api_test.js
```

## CI/CD Integration

### Quick GitHub Actions Example
```yaml
# .github/workflows/load-test.yml
name: Chat API Load Test
on:
  push:
    branches: [main]
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup K6
        run: |
          curl -s https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x77C6C491D6AC1D69 | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update && sudo apt-get install k6
      - name: Run Quick Test
        run: |
          k6 run \
            --env BASE_URL=${{ secrets.API_URL }} \
            --env AUTH_TOKEN=${{ secrets.API_TOKEN }} \
            tests/load/k6_chat_api_test.js
```

## What to Monitor

### During Test Execution
- Console output showing check results
- Error messages (if any)
- System resource usage (CPU, memory, network)

### After Test Completion
- Overall success rate (should be > 95%)
- p95 latency (should meet targets)
- Error breakdown (auth, validation, server)
- Model-specific performance differences

## Next Steps

1. **Baseline** - Run basic_load to establish baseline
2. **Document** - Save results for comparison
3. **Regular Testing** - Schedule weekly/monthly runs
4. **Alert Setup** - Configure alerts for regressions
5. **Deep Dive** - Review full documentation in CHAT_API_TESTING.md

## File Reference

- `k6_chat_api_test.js` - Main test suite (this file)
- `CHAT_API_TESTING.md` - Complete documentation
- `run-chat-test.sh` - Helper script for easy execution
- `.env.chat-test.example` - Configuration template
- `chat_api_summary.html` - Test results (generated)
- `chat_api_summary.json` - Test data (generated)

## Getting Help

1. **Check the logs** - Console output shows detailed errors
2. **Review HTML report** - Visual breakdown of all metrics
3. **Read full docs** - See CHAT_API_TESTING.md for details
4. **Test manually** - Use curl to debug specific requests

## Sample Manual Test

```bash
# Non-streaming request
curl -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'

# Streaming request
curl -N -X POST http://localhost:3284/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

---

**Ready to start?**

```bash
./tests/load/run-chat-test.sh --scenario basic_load
```

Good luck with your load testing! 🚀

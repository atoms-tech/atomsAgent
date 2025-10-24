# Chat API Load Testing Guide

This guide covers comprehensive load testing for the AgentAPI `/v1/chat/completions` endpoint using K6.

## Overview

The `k6_chat_api_test.js` suite provides extensive testing coverage for:

- **Non-streaming chat completions** - Standard request/response
- **Streaming chat completions (SSE)** - Server-Sent Events streaming
- **Multiple AI models** - gemini-1.5-pro, gpt-4, claude-3-opus, etc.
- **Variable message lengths** - Short, medium, and long prompts
- **Concurrent load patterns** - 1-50+ virtual users with various scenarios
- **Performance validation** - Response times, error rates, token tracking

## Quick Start

### 1. Install K6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows:**
```powershell
choco install k6
```

### 2. Run the Test

**Basic execution (uses defaults):**
```bash
k6 run tests/load/k6_chat_api_test.js
```

**With custom configuration:**
```bash
k6 run \
  --env BASE_URL=http://localhost:3284 \
  --env AUTH_TOKEN="Bearer your-token-here" \
  tests/load/k6_chat_api_test.js
```

**With multiple auth tokens (token rotation):**
```bash
k6 run \
  --env BASE_URL=http://localhost:3284 \
  --env AUTH_TOKENS="Bearer token1,Bearer token2,Bearer token3" \
  tests/load/k6_chat_api_test.js
```

## Test Scenarios

The suite includes 5 distinct scenarios that run sequentially:

### 1. Basic Load Test (2 minutes)
- **Load:** 10 requests/second
- **Purpose:** Baseline performance validation
- **Mix:** 70% non-streaming, 30% streaming
- **Message lengths:** Mixed (short, medium, long)

### 2. Ramp Up Test (5 minutes)
- **Load:** 0 → 20 → 50 → 100 req/s
- **Purpose:** Test system under increasing load
- **Mix:** 50% non-streaming, 50% streaming
- **Duration:** Gradual increase over 5 minutes

### 3. Stress Test (1 minute)
- **Load:** 200 requests/second (spike)
- **Purpose:** Identify breaking points
- **Mix:** 80% non-streaming, 20% streaming
- **Focus:** Short to medium messages only

### 4. Sustained Load Test (10 minutes)
- **Load:** 50 requests/second
- **Purpose:** Endurance and stability testing
- **Mix:** 60% non-streaming, 40% streaming
- **Features:** Error recovery with retries

### 5. Streaming-Specific Test (9 minutes)
- **VUs:** 1 → 10 → 30 → 50 concurrent streams
- **Purpose:** Deep streaming performance analysis
- **Focus:** All message lengths + model-specific tests
- **Metrics:** Time to first chunk, full stream duration

## Performance Thresholds

The test enforces the following SLAs:

### Overall Performance
- **p95 latency:** < 2000ms
- **p99 latency:** < 5000ms
- **Error rate:** < 5%

### Non-Streaming Specific
- **p95 latency:** < 1000ms
- **p99 latency:** < 2000ms

### Streaming Specific
- **p95 latency:** < 2000ms (full stream)
- **p99 latency:** < 5000ms (full stream)
- **Time to first chunk:** < 1000ms

### Message Length Thresholds
- **Short messages (p95):** < 1000ms
- **Medium messages (p95):** < 1500ms
- **Long messages (p95):** < 2500ms

## Custom Metrics

The test tracks comprehensive custom metrics:

### Success Rates
- `chat_completion_success_rate` - Overall success rate
- `streaming_success_rate` - Streaming-specific success
- `non_streaming_success_rate` - Non-streaming success

### Response Times
- `non_streaming_duration` - Non-streaming latency
- `streaming_duration` - Full streaming latency
- `streaming_latency` - Time to first SSE chunk
- `ttfb` - Time to first byte

### Model-Specific Metrics
- `gemini_duration` - Gemini 1.5 Pro performance
- `gpt4_duration` - GPT-4 performance
- `claude_duration` - Claude 3 performance

### Message Length Metrics
- `short_msg_duration` - Short message latency
- `medium_msg_duration` - Medium message latency
- `long_msg_duration` - Long message latency

### Counters
- `total_chat_requests` - Total requests sent
- `streaming_requests` - Streaming requests
- `non_streaming_requests` - Non-streaming requests
- `auth_failures` - Authentication failures (401/403)
- `validation_errors` - Request validation errors (400/422)
- `server_errors` - Server errors (5xx)
- `token_count` - Total tokens processed

### Gauges
- `concurrent_streams` - Active concurrent streams
- `active_requests` - Active requests at any time

## Request Validation

### Non-Streaming Response Checks
- ✓ HTTP 200 status
- ✓ JSON content-type
- ✓ Valid JSON structure
- ✓ OpenAI-compatible format (`object: "chat.completion"`)
- ✓ Choices array with message content
- ✓ Finish reason present
- ✓ Token usage information (prompt, completion, total)

### Streaming Response Checks
- ✓ HTTP 200 status
- ✓ SSE content-type (`text/event-stream`)
- ✓ Valid SSE format (`data:` lines)
- ✓ Multiple chunks received
- ✓ Content accumulation
- ✓ Finish reason in final chunk
- ✓ Stream completion marker (`[DONE]`)
- ✓ First chunk received within 1 second

## Test Data

### Supported Models
```javascript
- gemini-1.5-pro
- gpt-4
- gpt-4-turbo
- claude-3-opus
- claude-3-sonnet
```

### Message Templates

**Short Messages (< 20 words):**
- Simple greetings
- Single questions
- Quick commands

**Medium Messages (20-50 words):**
- Technical questions
- Best practice queries
- Concept explanations

**Long Messages (100+ words):**
- Complex architectural questions
- Multi-part requirements
- Detailed migration scenarios

### System Prompts
```javascript
- "You are a helpful coding assistant."
- "You are an expert software architect."
- "You are a senior developer specializing in distributed systems."
- null (no system prompt)
```

## Output Reports

The test generates three output files:

### 1. Console Output
Real-time test execution with color-coded results

### 2. HTML Report
`tests/load/chat_api_summary.html`
- Visual charts and graphs
- Detailed metrics breakdown
- Performance trends
- Error analysis

### 3. JSON Report
`tests/load/chat_api_summary.json`
- Complete raw test data
- All metric values
- Timestamp information
- For further analysis/automation

## Interpreting Results

### Example Summary Output
```
================================================================================
CHAT API LOAD TEST SUMMARY
================================================================================

Overall Performance:
--------------------------------------------------------------------------------
  Response Time (avg): 856.32ms
  Response Time (p50): 734.21ms
  Response Time (p95): 1456.89ms
  Response Time (p99): 2134.56ms
  Error Rate: 0.23%
  Total Chat Requests: 125430

Streaming vs Non-Streaming:
--------------------------------------------------------------------------------
  Non-Streaming (p95): 982.34ms
  Streaming (p95): 1789.45ms
  Time to First Chunk (p95): 234.56ms
  Streaming Requests: 37629
  Non-Streaming Requests: 87801

Success Rates:
--------------------------------------------------------------------------------
  Overall Success Rate: 99.77%
  Streaming Success Rate: 99.65%
  Non-Streaming Success Rate: 99.82%

Model Performance (p95):
--------------------------------------------------------------------------------
  Gemini 1.5 Pro: 1234.56ms
  GPT-4: 1456.78ms
  Claude 3: 1345.67ms

Message Length Performance (p95):
--------------------------------------------------------------------------------
  Short Messages: 567.89ms
  Medium Messages: 1123.45ms
  Long Messages: 2234.56ms

Error Breakdown:
--------------------------------------------------------------------------------
  Auth Failures (401/403): 12
  Validation Errors (400/422): 45
  Server Errors (5xx): 56

Token Usage:
--------------------------------------------------------------------------------
  Total Tokens Processed: 45678901
================================================================================
```

### Key Indicators

**✓ Healthy Performance:**
- Overall success rate > 99%
- p95 latency < thresholds
- Error rate < 1%
- Streaming latency < 2x non-streaming

**⚠️ Warning Signs:**
- Success rate 95-99%
- p95 latency approaching thresholds
- Error rate 1-5%
- Increasing trend in response times

**❌ Critical Issues:**
- Success rate < 95%
- p95 latency > thresholds
- Error rate > 5%
- High server error count (5xx)

## Advanced Usage

### Custom Scenarios

Edit the `options` object to customize test scenarios:

```javascript
export const options = {
  scenarios: {
    my_custom_scenario: {
      executor: 'constant-arrival-rate',
      exec: 'customFunction',
      rate: 50,           // requests per timeUnit
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 25,
      maxVUs: 100,
    },
  },
};
```

### Environment Variables

```bash
# Base API URL
BASE_URL=http://localhost:3284

# Single auth token
AUTH_TOKEN="Bearer your-token-here"

# Multiple tokens for rotation
AUTH_TOKENS="Bearer token1,Bearer token2,Bearer token3"
```

### Running Specific Scenarios

```bash
# Run only the streaming test
k6 run \
  --include-scenario-in-stats streaming_test \
  tests/load/k6_chat_api_test.js
```

### Cloud Execution

Run tests from K6 Cloud:

```bash
k6 cloud tests/load/k6_chat_api_test.js
```

### Distributed Testing

Run tests across multiple machines:

```bash
# Machine 1
k6 run --out json=results1.json tests/load/k6_chat_api_test.js

# Machine 2
k6 run --out json=results2.json tests/load/k6_chat_api_test.js

# Combine results
k6 inspect results1.json results2.json
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Load Test Chat API

on:
  schedule:
    - cron: '0 0 * * *'  # Daily at midnight
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install K6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Run Chat API Load Test
        env:
          BASE_URL: ${{ secrets.API_BASE_URL }}
          AUTH_TOKEN: ${{ secrets.API_AUTH_TOKEN }}
        run: |
          k6 run \
            --out json=results.json \
            tests/load/k6_chat_api_test.js

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: |
            tests/load/chat_api_summary.html
            tests/load/chat_api_summary.json
```

## Troubleshooting

### Common Issues

**1. Connection Refused**
```bash
# Check if API is running
curl http://localhost:3284/status

# Verify BASE_URL is correct
k6 run --env BASE_URL=http://correct-url tests/load/k6_chat_api_test.js
```

**2. High Error Rate**
```bash
# Check auth token
curl -H "Authorization: Bearer your-token" http://localhost:3284/v1/chat/completions

# Enable verbose logging
k6 run --verbose tests/load/k6_chat_api_test.js
```

**3. SSE Streaming Failures**
```bash
# Test SSE endpoint manually
curl -N -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"stream":true}' \
  http://localhost:3284/v1/chat/completions
```

**4. Out of Memory**
```bash
# Reduce concurrent VUs
# Edit options.scenarios.*.maxVUs to lower values
```

## Performance Optimization Tips

### For API Server
1. **Enable HTTP/2** - Better streaming performance
2. **Connection pooling** - Reduce overhead
3. **Response compression** - Faster transfers
4. **Caching layers** - For repeated queries
5. **Load balancing** - Distribute traffic

### For Testing
1. **Run from same network** - Minimize network latency
2. **Use multiple load generators** - Distributed testing
3. **Warm up the system** - Before capturing metrics
4. **Monitor system resources** - CPU, memory, network

## Best Practices

1. **Start small** - Begin with basic_load scenario
2. **Establish baseline** - Run when system is healthy
3. **Regular testing** - Schedule weekly/monthly tests
4. **Monitor trends** - Track performance over time
5. **Alert on regressions** - Set up automated alerts
6. **Document changes** - Note when performance shifts

## References

- [K6 Documentation](https://k6.io/docs/)
- [K6 Metrics Guide](https://k6.io/docs/using-k6/metrics/)
- [OpenAI API Specification](https://platform.openai.com/docs/api-reference)
- [Server-Sent Events Spec](https://html.spec.whatwg.org/multipage/server-sent-events.html)

## Support

For issues or questions:
- Check existing test logs in `tests/load/chat_api_summary.html`
- Review the main test file comments
- Consult the [AgentAPI documentation](../../README.md)

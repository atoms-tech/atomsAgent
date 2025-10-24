# K6 Load Testing for AgentAPI

This directory contains comprehensive load testing scripts for the AgentAPI using [K6](https://k6.io/), a modern load testing tool.

## Prerequisites

### Install K6

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

**Docker:**
```bash
docker pull grafana/k6:latest
```

## Test Scenarios

The load test suite includes 6 comprehensive scenarios:

### 1. Authentication (100 users)
- Tests OAuth initialization flow
- Validates token generation
- Checks agent status verification
- Duration: 9 minutes

### 2. Create MCP Connection (50 users)
- Tests MCP connection establishment
- Validates OAuth provider integration
- Monitors connection success rates
- Duration: 8 minutes

### 3. Execute Tool (200 users)
- Tests message sending to agent
- Validates tool execution
- Monitors execution success rates
- Duration: 9 minutes

### 4. List Tools (150 users)
- Tests status endpoint performance
- Validates message history retrieval
- Monitors read operation latency
- Duration: 8 minutes

### 5. Disconnect (50 users)
- Tests connection teardown
- Validates token revocation
- Monitors cleanup operations
- Duration: 7 minutes

### 6. Mixed Workload (300 users)
- Simulates realistic user behavior
- Weighted operation distribution:
  - 30% Status checks
  - 20% Get messages
  - 20% Send messages
  - 15% OAuth initialization
  - 10% Token refresh
  - 5% SSE subscription
- Duration: 13 minutes

## Performance Thresholds

The tests enforce strict performance thresholds:

### Response Time
- **p95 latency**: < 500ms (overall)
- **p99 latency**: < 2000ms (overall)

### Success Rates
- **MCP connection**: > 99%
- **Tool execution**: > 99%
- **Authentication**: > 99%
- **Overall error rate**: < 1%

### Endpoint-Specific
- Status endpoint: p95 < 200ms
- Messages endpoint: p95 < 300ms
- Message POST: p95 < 600ms
- OAuth init: p95 < 500ms

## Running the Tests

### Basic Usage

```bash
# Run with default configuration (localhost:3284)
k6 run tests/load/k6_tests.js
```

### Custom Configuration

```bash
# Run against production
k6 run --env BASE_URL=https://api.example.com tests/load/k6_tests.js

# Run with custom OAuth endpoint
k6 run \
  --env BASE_URL=https://api.example.com \
  --env OAUTH_BASE_URL=https://api.example.com/api/mcp/oauth \
  tests/load/k6_tests.js
```

### Run Specific Scenarios

```bash
# Run only authentication scenario
k6 run --scenarios=authentication tests/load/k6_tests.js

# Run multiple specific scenarios
k6 run --scenarios=authentication,tool_execution tests/load/k6_tests.js
```

### Output Options

```bash
# Generate HTML report
k6 run --out json=results.json tests/load/k6_tests.js

# Stream metrics to InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6_tests.js

# Stream to Prometheus
k6 run --out experimental-prometheus-rw tests/load/k6_tests.js

# Multiple outputs
k6 run \
  --out json=results.json \
  --out influxdb=http://localhost:8086/k6 \
  tests/load/k6_tests.js
```

### Using Docker

```bash
# Run in Docker container
docker run --rm -i \
  -v $PWD/tests/load:/scripts \
  -e BASE_URL=http://host.docker.internal:3284 \
  grafana/k6:latest run /scripts/k6_tests.js

# With custom environment
docker run --rm -i \
  -v $PWD/tests/load:/scripts \
  -e BASE_URL=https://api.example.com \
  -e OAUTH_BASE_URL=https://api.example.com/api/mcp/oauth \
  grafana/k6:latest run /scripts/k6_tests.js
```

## Test Results

After running the tests, you'll find:

- **Console Output**: Real-time metrics and progress
- **tests/load/summary.html**: HTML report with charts and graphs
- **tests/load/summary.json**: Raw JSON data for further analysis

### Understanding the Results

#### Key Metrics

1. **http_req_duration**: Response time for HTTP requests
   - p(50): Median response time
   - p(95): 95th percentile (only 5% of requests are slower)
   - p(99): 99th percentile (only 1% of requests are slower)

2. **http_req_failed**: Percentage of failed requests

3. **Custom Metrics**:
   - `mcp_connection_success_rate`: Success rate for MCP connections
   - `tool_execution_success_rate`: Success rate for tool executions
   - `auth_success_rate`: Success rate for authentication
   - `token_refresh_count`: Number of token refreshes
   - `connection_failures`: Number of connection failures
   - `timeout_count`: Number of timeouts

#### Threshold Violations

If any threshold is violated, K6 will:
- Display the violation in the summary
- Exit with a non-zero status code
- Mark the test run as failed

Example:
```
✓ http_req_duration..............: avg=245ms  p(95)=480ms ✓ p(99)=1890ms ✓
✗ http_req_failed................: 1.2% ✗ (threshold: <1%)
```

## Monitoring During Tests

### Real-Time Monitoring

While tests are running, you can monitor:

1. **K6 Cloud** (requires account):
```bash
k6 login cloud
k6 run --out cloud tests/load/k6_tests.js
```

2. **Local Grafana Dashboard**:
```bash
# Start InfluxDB and Grafana
docker-compose up -d

# Run tests with InfluxDB output
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6_tests.js

# Open Grafana at http://localhost:3000
```

3. **Prometheus + Grafana**:
```bash
# Start Prometheus and Grafana
docker-compose -f docker-compose.prometheus.yml up -d

# Run tests with Prometheus output
k6 run --out experimental-prometheus-rw tests/load/k6_tests.js
```

## Troubleshooting

### Common Issues

#### 1. Connection Refused
```
ERRO[0001] connection refused
```
**Solution**: Ensure the API is running and accessible at the specified BASE_URL.

#### 2. Timeout Errors
```
ERRO[0030] request timeout
```
**Solution**:
- Increase timeout values in the script
- Check API performance
- Reduce concurrent users

#### 3. Memory Issues
```
ERRO[0100] out of memory
```
**Solution**:
- Reduce number of VUs (virtual users)
- Increase available memory
- Use distributed load testing

#### 4. Threshold Violations
```
✗ http_req_duration.p(95) < 500
```
**Solution**:
- Optimize API endpoints
- Scale infrastructure
- Adjust thresholds if current values are unrealistic

### Debug Mode

Run with verbose logging:
```bash
k6 run --verbose tests/load/k6_tests.js
```

Enable HTTP debug logs:
```bash
k6 run --http-debug tests/load/k6_tests.js
```

### Reducing Test Intensity

For development/testing, you can reduce the load:

```bash
# Scale down to 10% of original load
k6 run --vus-factor=0.1 tests/load/k6_tests.js

# Run for shorter duration
# Edit the script to reduce stage durations
```

## Best Practices

### Before Running Tests

1. **Baseline Test**: Run a small-scale test first to verify everything works
2. **API Health**: Ensure the API is healthy and responsive
3. **Resources**: Ensure the API has sufficient resources (CPU, memory, database connections)
4. **Monitoring**: Set up monitoring on the API side to track resource usage

### During Tests

1. **Monitor API**: Watch API logs, metrics, and resource usage
2. **Database**: Monitor database connections and query performance
3. **Network**: Check for network bottlenecks
4. **Errors**: Watch for specific error patterns

### After Tests

1. **Analyze Results**: Review all metrics and threshold violations
2. **Identify Bottlenecks**: Use results to identify performance bottlenecks
3. **Correlate Data**: Cross-reference K6 results with API monitoring
4. **Document Findings**: Keep a record of test results and improvements

## CI/CD Integration

### GitHub Actions

```yaml
name: Load Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Run nightly
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

      - name: Run Load Tests
        run: |
          k6 run \
            --env BASE_URL=${{ secrets.API_URL }} \
            --out json=results.json \
            tests/load/k6_tests.js

      - name: Upload Results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: k6-results
          path: |
            results.json
            tests/load/summary.html
            tests/load/summary.json
```

### GitLab CI

```yaml
load-test:
  stage: test
  image: grafana/k6:latest
  script:
    - k6 run --env BASE_URL=$API_URL tests/load/k6_tests.js
  artifacts:
    paths:
      - tests/load/summary.html
      - tests/load/summary.json
    expire_in: 1 week
  only:
    - schedules
```

## Advanced Usage

### Custom Scenarios

You can create custom scenarios by modifying the `options.scenarios` object in the test file.

Example - Spike Test:
```javascript
spike_test: {
  executor: 'ramping-vus',
  startVUs: 0,
  stages: [
    { duration: '2m', target: 100 },
    { duration: '1m', target: 1000 },  // Spike
    { duration: '3m', target: 1000 },
    { duration: '2m', target: 100 },
    { duration: '2m', target: 0 },
  ],
}
```

### Environment Variables

You can use environment variables to customize the test:

```bash
export BASE_URL=https://api.example.com
export OAUTH_BASE_URL=https://api.example.com/api/mcp/oauth
export TEST_DURATION_MULTIPLIER=2  # Run for 2x the default duration

k6 run tests/load/k6_tests.js
```

## Support

For issues or questions:
- K6 Documentation: https://k6.io/docs/
- K6 Community: https://community.k6.io/
- AgentAPI Issues: https://github.com/coder/agentapi/issues

## License

This test suite is part of the AgentAPI project and follows the same license.

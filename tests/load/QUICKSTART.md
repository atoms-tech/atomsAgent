# Quick Start Guide - K6 Load Testing

This guide will get you running load tests in under 5 minutes.

## Prerequisites

- AgentAPI running locally on port 3284
- K6 installed (or Docker)

## Option 1: Run with K6 (Fastest)

### 1. Install K6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
curl -fsSL https://k6.io/install.sh | sh
```

**Windows:**
```powershell
choco install k6
```

### 2. Start AgentAPI

```bash
# In the agentapi project root
agentapi server -- claude
```

### 3. Run Tests

```bash
# Navigate to tests directory
cd tests/load

# Run all tests
k6 run k6_tests.js

# Or use the helper script
./run-tests.sh
```

That's it! K6 will run all test scenarios and display results in your terminal.

## Option 2: Run with Docker (No K6 Install Required)

### 1. Start AgentAPI

```bash
agentapi server -- claude
```

### 2. Run Tests in Docker

```bash
cd tests/load
./run-tests.sh --docker
```

## Option 3: Run with Monitoring (Recommended)

Get real-time metrics visualization with Grafana.

### 1. Start Monitoring Stack

```bash
cd tests/load
./run-tests.sh --docker --monitor
```

### 2. View Results

- **Terminal**: Live metrics during test run
- **Grafana**: Open http://localhost:3001 for real-time dashboards
- **HTML Report**: `tests/load/summary.html` after test completes

## Quick Test Options

### Run Specific Scenario

```bash
# Only test authentication
k6 run --scenarios=authentication k6_tests.js

# Or with helper script
./run-tests.sh --scenario authentication
```

### Reduced Load (For Quick Testing)

Edit `k6_tests.js` and reduce the `target` values in each scenario's stages:

```javascript
stages: [
  { duration: '30s', target: 10 },  // Instead of 100
  { duration: '1m', target: 10 },
  { duration: '30s', target: 0 },
]
```

### Run Against Different Environment

```bash
# Staging
./run-tests.sh --env staging

# Production (use with caution!)
./run-tests.sh --env production
```

## Understanding Results

### Success Criteria

After the test completes, look for:

```
✓ http_req_duration..............: p(95)=245ms ✓ p(99)=890ms ✓
✓ http_req_failed................: 0.12% ✓
✓ mcp_connection_success_rate....: 99.8% ✓
✓ tool_execution_success_rate....: 99.5% ✓
```

All metrics with ✓ mean the test passed!

### Key Metrics

- **p(95)**: 95% of requests completed in this time or less
- **p(99)**: 99% of requests completed in this time or less
- **http_req_failed**: Percentage of failed requests
- **Success rates**: Percentage of successful operations

### When Tests Fail

If you see ✗ next to a metric:

1. **Check API logs** - Is the API running correctly?
2. **Check resources** - Is the system under stress?
3. **Review thresholds** - Are they realistic for your setup?

## Common Issues

### "Connection refused"

**Problem**: AgentAPI is not running

**Solution**:
```bash
# Start AgentAPI first
agentapi server -- claude
```

### "Timeout errors"

**Problem**: API is slow or overloaded

**Solution**:
- Reduce load by scaling down VUs
- Increase timeout values in the script
- Check system resources

### Docker issues

**Problem**: Tests can't reach localhost API

**Solution**:
```bash
# Use host.docker.internal instead of localhost
export BASE_URL=http://host.docker.internal:3284
./run-tests.sh --docker
```

## Next Steps

- **Customize tests**: Edit `k6_tests.js` to add your own scenarios
- **CI/CD integration**: See `README.md` for GitHub Actions/GitLab CI examples
- **Advanced monitoring**: Set up Prometheus + Grafana for production monitoring
- **Distributed testing**: Use K6 Cloud or multiple K6 instances for larger tests

## Getting Help

- Check the full [README.md](README.md) for detailed documentation
- K6 documentation: https://k6.io/docs/
- AgentAPI issues: https://github.com/coder/agentapi/issues

## Tips

1. **Start small**: Run a single scenario first to verify everything works
2. **Monitor resources**: Watch CPU, memory, and network during tests
3. **Test during off-peak**: Avoid running heavy load tests during production hours
4. **Keep records**: Save test results for comparison over time
5. **Iterate**: Start with lower load and gradually increase

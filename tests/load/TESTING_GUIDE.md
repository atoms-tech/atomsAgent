# Complete Load Testing Guide for AgentAPI

This guide provides comprehensive information about the K6 load testing suite for AgentAPI.

## Table of Contents

1. [Overview](#overview)
2. [Test Scenarios](#test-scenarios)
3. [Performance Targets](#performance-targets)
4. [Running Tests](#running-tests)
5. [Interpreting Results](#interpreting-results)
6. [Troubleshooting](#troubleshooting)
7. [Best Practices](#best-practices)

## Overview

The K6 load testing suite simulates real-world usage of the AgentAPI with comprehensive scenarios covering:

- **Authentication flows**: OAuth initialization and token management
- **MCP connections**: Connection lifecycle management
- **Tool execution**: Message sending and agent interaction
- **Mixed workloads**: Realistic user behavior patterns

### Total Test Duration

- **Full suite**: ~15 minutes
- **Individual scenarios**: 7-13 minutes each
- **Quick test** (reduced VUs): 5-8 minutes

### Resource Requirements

**Client (K6 runner):**
- CPU: 2+ cores
- RAM: 4GB minimum, 8GB recommended
- Network: Stable connection to API

**Server (AgentAPI):**
- CPU: 4+ cores for 300 concurrent users
- RAM: 8GB minimum, 16GB recommended
- Database: 100+ concurrent connections
- Redis: 1000+ concurrent operations

## Test Scenarios

### 1. Authentication (100 concurrent users)

**Purpose**: Validate OAuth flow performance and token management

**Operations:**
- Initialize OAuth flow
- Generate state and PKCE parameters
- Verify agent status

**Duration**: 9 minutes
- Ramp up: 3 minutes (0 → 100 users)
- Steady state: 3 minutes
- Ramp down: 2 minutes

**Expected Results:**
- p95 latency: <300ms
- Success rate: >99%
- No token generation failures

### 2. Create MCP Connection (50 concurrent users)

**Purpose**: Test MCP connection establishment and OAuth integration

**Operations:**
- Initialize OAuth for MCP
- Create connection state in Redis
- Verify connection status

**Duration**: 8 minutes
- Ramp up: 3 minutes (0 → 50 users)
- Steady state: 3 minutes
- Ramp down: 2 minutes

**Expected Results:**
- p95 latency: <600ms
- Connection success rate: >99%
- Redis operations: <100ms

### 3. Execute Tool (200 concurrent users)

**Purpose**: Validate message sending and tool execution performance

**Operations:**
- Wait for agent stability
- Send user messages
- Execute agent tools (Bash, Edit, etc.)
- Retrieve conversation history

**Duration**: 9 minutes
- Ramp up: 4 minutes (0 → 200 users)
- Steady state: 3 minutes
- Ramp down: 2 minutes

**Expected Results:**
- p95 latency: <800ms
- Execution success rate: >99%
- Message delivery: 100%

### 4. List Tools (150 concurrent users)

**Purpose**: Test read-heavy operations and caching

**Operations:**
- Get agent status
- List available tools
- Retrieve message history

**Duration**: 8 minutes
- Ramp up: 3 minutes (0 → 150 users)
- Steady state: 3 minutes
- Ramp down: 2 minutes

**Expected Results:**
- p95 latency: <400ms
- Cache hit rate: >80%
- No read failures

### 5. Disconnect (50 concurrent users)

**Purpose**: Validate cleanup and disconnection operations

**Operations:**
- Create temporary connection
- Revoke OAuth tokens
- Clean up Redis state

**Duration**: 7 minutes
- Ramp up: 3 minutes (0 → 50 users)
- Steady state: 2 minutes
- Ramp down: 2 minutes

**Expected Results:**
- p95 latency: <300ms
- Cleanup success: >99%
- No resource leaks

### 6. Mixed Workload (300 concurrent users)

**Purpose**: Simulate realistic user behavior with varied operations

**Operation Distribution:**
- 30%: Status checks (lightweight)
- 20%: Get messages (read-heavy)
- 20%: Send messages (write-heavy)
- 15%: OAuth initialization
- 10%: Token refresh
- 5%: SSE subscriptions

**Duration**: 13 minutes
- Ramp up: 6 minutes (0 → 300 users)
- Steady state: 4 minutes
- Ramp down: 3 minutes

**Expected Results:**
- p95 latency: <500ms
- Overall success rate: >99%
- Balanced resource usage

## Performance Targets

### Response Time Targets

| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| Status check | <50ms | <200ms | <500ms |
| Get messages | <100ms | <300ms | <800ms |
| Send message | <200ms | <600ms | <1500ms |
| OAuth init | <150ms | <500ms | <1200ms |
| Token refresh | <100ms | <400ms | <1000ms |
| Disconnect | <100ms | <300ms | <800ms |

### Reliability Targets

- **Overall error rate**: <1%
- **Authentication success**: >99%
- **MCP connection success**: >99%
- **Tool execution success**: >99%
- **Timeout rate**: <0.1%

### Throughput Targets

- **Requests per second**: 500-1000 RPS (peak)
- **Concurrent connections**: 300+ simultaneous users
- **Messages per minute**: 100+ agent interactions

## Running Tests

### Prerequisites Checklist

- [ ] K6 installed (or Docker available)
- [ ] AgentAPI running and accessible
- [ ] Redis available and configured
- [ ] OAuth providers configured
- [ ] Test environment variables set

### Quick Start

```bash
# 1. Navigate to test directory
cd tests/load

# 2. Start AgentAPI (if not running)
agentapi server -- claude

# 3. Run tests
./run-tests.sh
```

### Advanced Options

**Run specific scenario:**
```bash
./run-tests.sh --scenario authentication
```

**Run with monitoring:**
```bash
./run-tests.sh --docker --monitor
# View at http://localhost:3001
```

**Run against staging:**
```bash
./run-tests.sh --env staging
```

**Reduce load (for testing):**
```bash
# Edit k6_tests.js and reduce target values
# Or use environment variable
export VUS_FACTOR=0.1
./run-tests.sh
```

## Interpreting Results

### Console Output

During the test, K6 displays real-time metrics:

```
running (05m00s), 150/300 VUs, 12450 complete and 0 interrupted iterations
default ✓ [======================================>-------] 150/300 VUs  5m0s/15m0s

     ✓ http_req_duration..............: avg=245ms  min=12ms  med=198ms  max=2.1s   p(95)=480ms p(99)=890ms
     ✓ http_req_failed................: 0.12%  ✓ 15      ✗ 12435
     ✓ mcp_connection_success_rate....: 99.8%  ✓ 1245    ✗ 3
```

**What to watch:**
- VUs: Should ramp up smoothly
- Duration: Progress through test phases
- Metrics: Should stay within thresholds (✓)

### HTML Report

After the test, open `summary.html` for detailed visualizations:

- **Overview**: Key metrics at a glance
- **Charts**: Response time distribution
- **Trends**: Performance over time
- **Breakdowns**: By scenario, endpoint, status code

### JSON Results

`summary.json` contains raw data for further analysis:

```bash
# Extract specific metrics
jq '.metrics.http_req_duration.values["p(95)"]' summary.json

# Get error rate
jq '.metrics.http_req_failed.values.rate' summary.json

# List all custom metrics
jq '.metrics | keys' summary.json
```

### Success Indicators

**Test PASSED:**
```
✓ All thresholds passed
✓ Error rate < 1%
✓ p95 latency within targets
✓ No timeout errors
✓ All scenarios completed
```

**Test FAILED:**
```
✗ One or more thresholds failed
✗ High error rate (>1%)
✗ Latency spikes
✗ Connection failures
✗ Timeout errors
```

## Troubleshooting

### High Error Rates

**Symptoms:**
- Error rate >1%
- Many 500 status codes
- Connection failures

**Diagnosis:**
```bash
# Check API logs
tail -f /var/log/agentapi.log

# Check system resources
top
htop
df -h
```

**Solutions:**
1. Scale up API resources (CPU, RAM)
2. Optimize database queries
3. Increase connection pools
4. Add caching layer

### High Latency

**Symptoms:**
- p95 >500ms
- p99 >2000ms
- Slow response times

**Diagnosis:**
```bash
# Profile API
# Use built-in profiling or APM tools

# Check database performance
# Review slow query logs

# Check network latency
ping api.example.com
```

**Solutions:**
1. Optimize slow endpoints
2. Add database indexes
3. Enable response caching
4. Use CDN for static assets

### Connection Timeouts

**Symptoms:**
- Timeout errors
- Aborted connections
- Incomplete requests

**Diagnosis:**
```bash
# Check connection limits
ulimit -n

# Check Redis connections
redis-cli INFO clients

# Check database connections
# Review database connection pool
```

**Solutions:**
1. Increase timeout values
2. Expand connection pools
3. Add connection retry logic
4. Scale infrastructure

### Memory Issues

**Symptoms:**
- Out of memory errors
- Increasing memory usage
- Swap usage

**Diagnosis:**
```bash
# Monitor memory
free -h
vmstat 1

# Check for leaks
# Use memory profiler
```

**Solutions:**
1. Fix memory leaks
2. Increase available memory
3. Optimize data structures
4. Implement pagination

## Best Practices

### Before Running Tests

1. **Establish baseline**: Run with minimal load first
2. **Monitor infrastructure**: Ensure healthy state
3. **Review code**: Check recent changes
4. **Notify team**: Inform about test schedule
5. **Prepare rollback**: Have plan ready if issues arise

### During Tests

1. **Monitor actively**: Watch metrics in real-time
2. **Check logs**: Look for errors and warnings
3. **Track resources**: Monitor CPU, memory, disk, network
4. **Take notes**: Document observations
5. **Be ready to stop**: Kill test if critical issues appear

### After Tests

1. **Analyze results**: Review all metrics thoroughly
2. **Compare trends**: Check against previous runs
3. **Document findings**: Record issues and insights
4. **Share results**: Distribute to team
5. **Plan improvements**: Create action items

### Continuous Improvement

1. **Regular testing**: Run load tests frequently
2. **Track metrics**: Keep historical data
3. **Set baselines**: Define acceptable performance
4. **Iterate**: Continuously optimize based on results
5. **Automate**: Integrate into CI/CD pipeline

### Load Testing Ethics

1. **Test environments**: Never production (unless planned)
2. **Off-peak hours**: Minimize impact on users
3. **Gradual ramp**: Don't spike load instantly
4. **Monitor impact**: Watch for service degradation
5. **Communication**: Keep stakeholders informed

## Advanced Topics

### Distributed Load Testing

For tests exceeding single machine capacity:

```bash
# Use K6 Cloud
k6 cloud k6_tests.js

# Or run multiple K6 instances
# Instance 1
k6 run --scenarios=authentication k6_tests.js

# Instance 2
k6 run --scenarios=tool_execution k6_tests.js
```

### Custom Metrics

Add your own metrics in `k6_tests.js`:

```javascript
import { Counter, Trend } from 'k6/metrics';

const myMetric = new Counter('my_custom_counter');
const myTrend = new Trend('my_custom_trend');

// Use in test
myMetric.add(1);
myTrend.add(duration);
```

### Integration with APM

Send metrics to monitoring systems:

```bash
# Datadog
k6 run --out datadog k6_tests.js

# New Relic
k6 run --out newrelic k6_tests.js

# Prometheus
k6 run --out experimental-prometheus-rw k6_tests.js
```

## Resources

- [K6 Documentation](https://k6.io/docs/)
- [K6 Community Forum](https://community.k6.io/)
- [AgentAPI Documentation](../../README.md)
- [Load Testing Best Practices](https://k6.io/docs/testing-guides/test-types/)

## Support

For issues or questions:

1. Check this guide and [README.md](README.md)
2. Review [QUICKSTART.md](QUICKSTART.md)
3. Search K6 documentation
4. Ask in team chat
5. File issue on GitHub

---

**Happy Load Testing!** 🚀

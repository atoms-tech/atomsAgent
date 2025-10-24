# K6 Load Testing Suite - Implementation Summary

## Overview

A comprehensive K6 load testing suite has been created for the AgentAPI project at:
`/Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi/tests/load/`

## What Was Created

### Core Test Files

1. **k6_tests.js** (807 lines)
   - Main K6 test script
   - 6 comprehensive test scenarios
   - Custom metrics tracking
   - Performance thresholds
   - Setup/teardown functions

### Documentation

2. **README.md**
   - Complete documentation
   - Installation instructions
   - Usage examples
   - Troubleshooting guide
   - CI/CD integration examples

3. **QUICKSTART.md**
   - 5-minute quick start guide
   - Common use cases
   - Quick troubleshooting

4. **TESTING_GUIDE.md**
   - Comprehensive testing guide
   - Detailed scenario descriptions
   - Performance targets
   - Best practices

### Configuration Files

5. **run-tests.sh** (executable)
   - Convenient test runner script
   - Environment selection
   - Scenario selection
   - Docker support
   - Monitoring integration

6. **package.json**
   - NPM scripts for common tasks
   - Easy command aliases

7. **.env.example**
   - Environment variable template
   - Configuration options

8. **.env.local**
   - Local environment config

9. **.env.staging**
   - Staging environment config

10. **.env.production**
    - Production environment config

11. **.gitignore**
    - Ignore test results
    - Ignore sensitive files

### Docker & Monitoring

12. **docker-compose.yml**
    - InfluxDB configuration
    - Grafana setup
    - K6 runner container

13. **grafana/datasources/influxdb.yml**
    - InfluxDB data source config

14. **grafana/dashboards/dashboard.yml**
    - Dashboard provisioning config

### CI/CD Integration

15. **.github/workflows/load-tests.yml**
    - Automated load testing workflow
    - Scheduled tests (nightly)
    - PR integration
    - Result notifications

## Test Scenarios

### 1. Authentication (100 users)
- OAuth initialization
- Token generation
- Agent status verification
- Duration: 9 minutes

### 2. Create MCP Connection (50 users)
- MCP connection establishment
- OAuth provider integration
- Redis state management
- Duration: 8 minutes

### 3. Execute Tool (200 users)
- Message sending
- Tool execution
- Conversation history
- Duration: 9 minutes

### 4. List Tools (150 users)
- Status endpoint
- Tool listing
- Message retrieval
- Duration: 8 minutes

### 5. Disconnect (50 users)
- Connection teardown
- Token revocation
- State cleanup
- Duration: 7 minutes

### 6. Mixed Workload (300 users)
- Realistic usage patterns
- Multiple operation types
- Weighted distribution
- Duration: 13 minutes

## Performance Thresholds

### Response Time
- **p95 latency**: < 500ms (overall)
- **p99 latency**: < 2000ms (overall)

### Success Rates
- **MCP connection**: > 99%
- **Tool execution**: > 99%
- **Authentication**: > 99%
- **Error rate**: < 1%

### Endpoint-Specific
- Status: p95 < 200ms
- Messages: p95 < 300ms
- Message POST: p95 < 600ms
- OAuth init: p95 < 500ms

## Custom Metrics

The test suite tracks custom metrics:

1. **mcp_connection_success_rate** - MCP connection success percentage
2. **tool_execution_success_rate** - Tool execution success percentage
3. **auth_success_rate** - Authentication success percentage
4. **disconnect_success_rate** - Disconnection success percentage
5. **token_refresh_count** - Number of token refreshes
6. **connection_failures** - Number of connection failures
7. **timeout_count** - Number of timeout errors
8. **total_requests** - Total HTTP requests made
9. **active_connections** - Active MCP connections gauge
10. **concurrent_users** - Concurrent user gauge

## Quick Start

### Local Testing

```bash
# Navigate to test directory
cd tests/load

# Run all tests
./run-tests.sh

# Or directly with k6
k6 run k6_tests.js
```

### With Monitoring

```bash
# Start monitoring stack and run tests
./run-tests.sh --docker --monitor

# View Grafana at http://localhost:3001
```

### Specific Scenario

```bash
# Run only authentication scenario
./run-tests.sh --scenario authentication

# Or with k6
k6 run --scenario authentication k6_tests.js
```

### Different Environments

```bash
# Staging
./run-tests.sh --env staging

# Production (reduced load)
./run-tests.sh --env production
```

## Usage Examples

### Using NPM Scripts

```bash
cd tests/load

# Run all tests
npm test

# Run specific scenario
npm run test:auth
npm run test:mcp
npm run test:tools

# Run with monitoring
npm run test:monitor

# Clean up results
npm run clean
```

### Docker Usage

```bash
# Run in Docker
docker run --rm -i \
  -v $PWD:/scripts \
  -e BASE_URL=http://host.docker.internal:3284 \
  grafana/k6:latest run /scripts/k6_tests.js
```

### CI/CD

The GitHub Actions workflow runs automatically:
- **Nightly**: 2 AM UTC daily
- **On PR**: When load test files change
- **Manual**: Via workflow_dispatch

## Output & Results

After running tests, you'll find:

1. **Console output**: Real-time metrics and progress
2. **summary.html**: Visual report with charts
3. **summary.json**: Raw JSON data
4. **results.json**: Detailed request logs (if configured)

## Monitoring Stack

When using `--monitor` flag:

### InfluxDB
- URL: http://localhost:8086
- Database: k6
- Stores time-series metrics

### Grafana
- URL: http://localhost:3001
- Auto-configured with InfluxDB
- Real-time dashboards

## Key Features

1. **Comprehensive Coverage**
   - All major API endpoints
   - OAuth flows
   - MCP operations
   - Mixed workloads

2. **Realistic Scenarios**
   - Gradual ramp-up/down
   - Steady-state testing
   - Weighted operations
   - Think times

3. **Detailed Metrics**
   - Response times (p50, p95, p99)
   - Success rates
   - Custom metrics
   - Per-scenario breakdown

4. **Strict Thresholds**
   - Performance targets
   - Reliability targets
   - Automatic failure detection

5. **Easy to Use**
   - Helper scripts
   - Multiple run options
   - Environment configs
   - Docker support

6. **Production Ready**
   - CI/CD integration
   - Monitoring integration
   - Result archival
   - Notification support

## File Structure

```
tests/load/
├── k6_tests.js              # Main test script (807 lines)
├── run-tests.sh             # Test runner script (executable)
├── package.json             # NPM scripts
├── README.md                # Complete documentation
├── QUICKSTART.md            # 5-minute guide
├── TESTING_GUIDE.md         # Comprehensive guide
├── SUMMARY.md               # This file
├── .env.example             # Environment template
├── .env.local               # Local config
├── .env.staging             # Staging config
├── .env.production          # Production config
├── .gitignore               # Git ignore rules
├── docker-compose.yml       # Docker services
└── grafana/
    ├── datasources/
    │   └── influxdb.yml     # InfluxDB config
    └── dashboards/
        └── dashboard.yml    # Dashboard config

../.github/workflows/
└── load-tests.yml           # CI/CD workflow
```

## Next Steps

1. **Install K6**
   ```bash
   brew install k6  # macOS
   # Or use Docker
   ```

2. **Start AgentAPI**
   ```bash
   agentapi server -- claude
   ```

3. **Run Tests**
   ```bash
   cd tests/load
   ./run-tests.sh
   ```

4. **Review Results**
   - Check console output
   - Open summary.html
   - Review metrics

5. **Customize**
   - Adjust thresholds
   - Add scenarios
   - Configure environments

## Support Resources

- **Quick Start**: See QUICKSTART.md
- **Full Docs**: See README.md
- **Testing Guide**: See TESTING_GUIDE.md
- **K6 Docs**: https://k6.io/docs/
- **Issues**: https://github.com/coder/agentapi/issues

## Summary Statistics

- **Total Files Created**: 15
- **Lines of Code**: 807 (k6_tests.js)
- **Test Scenarios**: 6
- **Total Test Duration**: ~15 minutes
- **Max Concurrent Users**: 300
- **Custom Metrics**: 10
- **Performance Thresholds**: 20+
- **Supported Environments**: 3 (local, staging, production)

## Success Criteria

Tests are considered successful when:
- ✓ All scenarios complete without errors
- ✓ p95 latency < 500ms
- ✓ p99 latency < 2000ms
- ✓ Error rate < 1%
- ✓ Success rates > 99%
- ✓ No timeout errors
- ✓ All custom thresholds pass

---

**Implementation Complete!** 🎉

The K6 load testing suite is ready to use. Start with QUICKSTART.md for immediate usage
or README.md for comprehensive documentation.

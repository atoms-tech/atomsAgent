/**
 * K6 Load Testing Suite for AgentAPI
 *
 * This comprehensive test suite covers:
 * - Login/Authentication flows
 * - MCP connection lifecycle
 * - Tool execution
 * - Mixed workload scenarios
 *
 * Run with: k6 run tests/load/k6_tests.js
 * Or with custom env: k6 run --env BASE_URL=https://your-api.com tests/load/k6_tests.js
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// ==================== CONFIGURATION ====================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3284';
const OAUTH_BASE_URL = __ENV.OAUTH_BASE_URL || 'http://localhost:3000/api/mcp/oauth';

// Test scenarios configuration
export const options = {
  scenarios: {
    // Scenario 1: Login/Authentication (100 users)
    authentication: {
      executor: 'ramping-vus',
      exec: 'authenticationScenario',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 20 },   // Ramp up to 20 users
        { duration: '1m', target: 50 },   // Ramp up to 50 users
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '3m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '30s',
      tags: { scenario: 'authentication' },
    },

    // Scenario 2: Create MCP Connection (50 users)
    mcp_connection: {
      executor: 'ramping-vus',
      exec: 'mcpConnectionScenario',
      startVUs: 0,
      startTime: '2m',
      stages: [
        { duration: '1m', target: 10 },   // Ramp up to 10 users
        { duration: '1m', target: 25 },   // Ramp up to 25 users
        { duration: '1m', target: 50 },   // Ramp up to 50 users
        { duration: '3m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '30s',
      tags: { scenario: 'mcp_connection' },
    },

    // Scenario 3: Execute Tool (200 users)
    tool_execution: {
      executor: 'ramping-vus',
      exec: 'toolExecutionScenario',
      startVUs: 0,
      startTime: '3m',
      stages: [
        { duration: '1m', target: 50 },   // Ramp up to 50 users
        { duration: '1m', target: 100 },  // Ramp up to 100 users
        { duration: '2m', target: 200 },  // Ramp up to 200 users
        { duration: '3m', target: 200 },  // Stay at 200 users
        { duration: '2m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '30s',
      tags: { scenario: 'tool_execution' },
    },

    // Scenario 4: List Tools (150 users)
    list_tools: {
      executor: 'ramping-vus',
      exec: 'listToolsScenario',
      startVUs: 0,
      startTime: '4m',
      stages: [
        { duration: '1m', target: 30 },   // Ramp up to 30 users
        { duration: '1m', target: 75 },   // Ramp up to 75 users
        { duration: '1m', target: 150 },  // Ramp up to 150 users
        { duration: '3m', target: 150 },  // Stay at 150 users
        { duration: '2m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '30s',
      tags: { scenario: 'list_tools' },
    },

    // Scenario 5: Disconnect (50 users)
    disconnect: {
      executor: 'ramping-vus',
      exec: 'disconnectScenario',
      startVUs: 0,
      startTime: '5m',
      stages: [
        { duration: '1m', target: 10 },   // Ramp up to 10 users
        { duration: '1m', target: 25 },   // Ramp up to 25 users
        { duration: '1m', target: 50 },   // Ramp up to 50 users
        { duration: '2m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '30s',
      tags: { scenario: 'disconnect' },
    },

    // Scenario 6: Mixed Workload (300 users)
    mixed_workload: {
      executor: 'ramping-vus',
      exec: 'mixedWorkloadScenario',
      startVUs: 0,
      startTime: '6m',
      stages: [
        { duration: '2m', target: 75 },   // Ramp up to 75 users
        { duration: '2m', target: 150 },  // Ramp up to 150 users
        { duration: '2m', target: 300 },  // Ramp up to 300 users
        { duration: '4m', target: 300 },  // Stay at 300 users (steady state)
        { duration: '3m', target: 0 },    // Ramp down to 0
      ],
      gracefulRampDown: '1m',
      tags: { scenario: 'mixed_workload' },
    },
  },

  // Performance thresholds
  thresholds: {
    // Overall HTTP metrics
    'http_req_duration': ['p(95)<500', 'p(99)<2000'],
    'http_req_failed': ['rate<0.01'],  // Error rate < 1%

    // Custom metrics thresholds
    'mcp_connection_success_rate': ['rate>0.99'],
    'tool_execution_success_rate': ['rate>0.99'],
    'auth_success_rate': ['rate>0.99'],

    // Scenario-specific thresholds
    'http_req_duration{scenario:authentication}': ['p(95)<300'],
    'http_req_duration{scenario:mcp_connection}': ['p(95)<600'],
    'http_req_duration{scenario:tool_execution}': ['p(95)<800'],
    'http_req_duration{scenario:list_tools}': ['p(95)<400'],
    'http_req_duration{scenario:disconnect}': ['p(95)<300'],
    'http_req_duration{scenario:mixed_workload}': ['p(95)<500'],

    // API endpoint specific thresholds
    'http_req_duration{endpoint:status}': ['p(95)<200'],
    'http_req_duration{endpoint:messages}': ['p(95)<300'],
    'http_req_duration{endpoint:message}': ['p(95)<600'],
    'http_req_duration{endpoint:oauth_init}': ['p(95)<500'],
  },
};

// ==================== CUSTOM METRICS ====================

// Success rates
const mcpConnectionSuccessRate = new Rate('mcp_connection_success_rate');
const toolExecutionSuccessRate = new Rate('tool_execution_success_rate');
const authSuccessRate = new Rate('auth_success_rate');
const disconnectSuccessRate = new Rate('disconnect_success_rate');

// Response times
const authDuration = new Trend('auth_duration');
const mcpConnectionDuration = new Trend('mcp_connection_duration');
const toolExecutionDuration = new Trend('tool_execution_duration');
const disconnectDuration = new Trend('disconnect_duration');

// Counters
const tokenRefreshCount = new Counter('token_refresh_count');
const connectionFailures = new Counter('connection_failures');
const timeoutCount = new Counter('timeout_count');
const totalRequests = new Counter('total_requests');

// Gauges
const activeConnections = new Gauge('active_connections');
const concurrentUsers = new Gauge('concurrent_users');

// ==================== HELPER FUNCTIONS ====================

/**
 * Generate a random user for testing
 */
function generateTestUser() {
  return {
    id: `user_${randomString(16)}`,
    email: `test_${randomString(8)}@example.com`,
    session_id: `session_${randomString(24)}`,
  };
}

/**
 * Generate OAuth token (mock for testing)
 */
function generateOAuthToken(userId) {
  return `Bearer ${Buffer.from(JSON.stringify({
    user_id: userId,
    exp: Date.now() + 3600000
  })).toString('base64')}`;
}

/**
 * Common headers for API requests
 */
function getHeaders(token = null) {
  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };

  if (token) {
    headers['Authorization'] = token;
  }

  return headers;
}

/**
 * Check if response is successful
 */
function isSuccess(response, expectedStatus = 200) {
  return check(response, {
    [`status is ${expectedStatus}`]: (r) => r.status === expectedStatus,
    'response has body': (r) => r.body && r.body.length > 0,
    'no server errors': (r) => r.status < 500,
  });
}

/**
 * Log error details
 */
function logError(endpoint, response, context = {}) {
  console.error(`[ERROR] ${endpoint}:`, {
    status: response.status,
    body: response.body ? response.body.substring(0, 200) : 'empty',
    duration: response.timings.duration,
    ...context,
  });
}

/**
 * Wait for agent to become stable
 */
function waitForStableAgent(timeout = 30000) {
  const startTime = Date.now();

  while (Date.now() - startTime < timeout) {
    const statusRes = http.get(`${BASE_URL}/status`, {
      tags: { endpoint: 'status' },
    });

    if (statusRes.status === 200) {
      const status = JSON.parse(statusRes.body);
      if (status.status === 'stable') {
        return true;
      }
    }

    sleep(1);
  }

  timeoutCount.add(1);
  return false;
}

// ==================== SCENARIO FUNCTIONS ====================

/**
 * Scenario 1: Authentication Flow
 */
export function authenticationScenario() {
  const user = generateTestUser();

  group('Authentication', () => {
    const startTime = Date.now();

    // Step 1: Initialize OAuth flow
    const initRes = http.post(
      `${OAUTH_BASE_URL}/init`,
      JSON.stringify({
        mcp_name: 'github',
        provider: 'github',
        redirect_uri: 'http://localhost:3000/oauth/callback',
      }),
      {
        headers: getHeaders(generateOAuthToken(user.id)),
        tags: { endpoint: 'oauth_init' },
      }
    );

    totalRequests.add(1);

    const initSuccess = isSuccess(initRes);
    authSuccessRate.add(initSuccess);

    if (!initSuccess) {
      logError('OAuth Init', initRes, { user_id: user.id });
      return;
    }

    // Step 2: Simulate OAuth callback (would normally involve user interaction)
    sleep(randomIntBetween(1, 3));

    // Step 3: Check agent status
    const statusRes = http.get(`${BASE_URL}/status`, {
      tags: { endpoint: 'status' },
    });

    totalRequests.add(1);

    const statusSuccess = check(statusRes, {
      'status check successful': (r) => r.status === 200,
      'has agent_type': (r) => JSON.parse(r.body).agent_type !== undefined,
      'has status': (r) => JSON.parse(r.body).status !== undefined,
    });

    authSuccessRate.add(statusSuccess);

    const duration = Date.now() - startTime;
    authDuration.add(duration);

    if (!statusSuccess) {
      logError('Status Check', statusRes, { user_id: user.id });
    }
  });

  sleep(randomIntBetween(1, 5));
}

/**
 * Scenario 2: Create MCP Connection
 */
export function mcpConnectionScenario() {
  const user = generateTestUser();
  const mcpName = `mcp_${randomString(8)}`;

  group('MCP Connection', () => {
    const startTime = Date.now();

    // Step 1: Initialize MCP connection via OAuth
    const initRes = http.post(
      `${OAUTH_BASE_URL}/init`,
      JSON.stringify({
        mcp_name: mcpName,
        provider: ['github', 'google', 'azure'][randomIntBetween(0, 2)],
        redirect_uri: 'http://localhost:3000/oauth/callback',
      }),
      {
        headers: getHeaders(generateOAuthToken(user.id)),
        tags: { endpoint: 'oauth_init' },
      }
    );

    totalRequests.add(1);

    const success = isSuccess(initRes);
    mcpConnectionSuccessRate.add(success);

    if (success) {
      const data = JSON.parse(initRes.body);
      activeConnections.add(1);

      // Step 2: Verify connection by checking status
      sleep(1);

      const statusRes = http.get(`${BASE_URL}/status`, {
        tags: { endpoint: 'status' },
      });

      totalRequests.add(1);

      if (statusRes.status !== 200) {
        connectionFailures.add(1);
        activeConnections.add(-1);
      }
    } else {
      connectionFailures.add(1);
      logError('MCP Connection Init', initRes, {
        user_id: user.id,
        mcp_name: mcpName
      });
    }

    const duration = Date.now() - startTime;
    mcpConnectionDuration.add(duration);
  });

  sleep(randomIntBetween(2, 6));
}

/**
 * Scenario 3: Tool Execution
 */
export function toolExecutionScenario() {
  const user = generateTestUser();

  group('Tool Execution', () => {
    const startTime = Date.now();

    // Step 1: Wait for agent to be stable
    if (!waitForStableAgent(10000)) {
      toolExecutionSuccessRate.add(false);
      return;
    }

    // Step 2: Send message to agent (tool execution)
    const messages = [
      'List the files in the current directory',
      'Show git status',
      'Check the version of Node.js',
      'Display system information',
      'Run tests',
    ];

    const message = messages[randomIntBetween(0, messages.length - 1)];

    const messageRes = http.post(
      `${BASE_URL}/message`,
      JSON.stringify({
        content: message,
        type: 'user',
      }),
      {
        headers: getHeaders(),
        tags: { endpoint: 'message' },
        timeout: '60s',
      }
    );

    totalRequests.add(1);

    const success = check(messageRes, {
      'message sent successfully': (r) => r.status === 200,
      'response has ok field': (r) => {
        try {
          return JSON.parse(r.body).ok !== undefined;
        } catch {
          return false;
        }
      },
    });

    toolExecutionSuccessRate.add(success);

    if (success) {
      // Step 3: Get conversation history
      sleep(randomIntBetween(1, 3));

      const messagesRes = http.get(`${BASE_URL}/messages`, {
        headers: getHeaders(),
        tags: { endpoint: 'messages' },
      });

      totalRequests.add(1);

      check(messagesRes, {
        'messages retrieved': (r) => r.status === 200,
        'has messages array': (r) => {
          try {
            return Array.isArray(JSON.parse(r.body).messages);
          } catch {
            return false;
          }
        },
      });
    } else {
      logError('Tool Execution', messageRes, {
        user_id: user.id,
        message: message.substring(0, 50)
      });
    }

    const duration = Date.now() - startTime;
    toolExecutionDuration.add(duration);
  });

  sleep(randomIntBetween(3, 8));
}

/**
 * Scenario 4: List Tools
 */
export function listToolsScenario() {
  group('List Tools', () => {
    // Get agent status (which includes available tools info)
    const statusRes = http.get(`${BASE_URL}/status`, {
      tags: { endpoint: 'status' },
    });

    totalRequests.add(1);

    check(statusRes, {
      'status retrieved': (r) => r.status === 200,
      'has agent type': (r) => {
        try {
          return JSON.parse(r.body).agent_type !== undefined;
        } catch {
          return false;
        }
      },
    });

    // Get conversation history (shows tool usage)
    const messagesRes = http.get(`${BASE_URL}/messages`, {
      tags: { endpoint: 'messages' },
    });

    totalRequests.add(1);

    check(messagesRes, {
      'messages retrieved': (r) => r.status === 200,
    });
  });

  sleep(randomIntBetween(1, 4));
}

/**
 * Scenario 5: Disconnect
 */
export function disconnectScenario() {
  const user = generateTestUser();
  const mcpName = `mcp_${randomString(8)}`;

  group('Disconnect', () => {
    const startTime = Date.now();

    // Step 1: Create a connection first
    const initRes = http.post(
      `${OAUTH_BASE_URL}/init`,
      JSON.stringify({
        mcp_name: mcpName,
        provider: 'github',
        redirect_uri: 'http://localhost:3000/oauth/callback',
      }),
      {
        headers: getHeaders(generateOAuthToken(user.id)),
        tags: { endpoint: 'oauth_init' },
      }
    );

    totalRequests.add(1);

    if (isSuccess(initRes)) {
      sleep(1);

      // Step 2: Revoke/disconnect
      const revokeRes = http.post(
        `${OAUTH_BASE_URL}/revoke`,
        JSON.stringify({
          mcp_config_id: `config_${randomString(16)}`,
        }),
        {
          headers: getHeaders(generateOAuthToken(user.id)),
          tags: { endpoint: 'oauth_revoke' },
        }
      );

      totalRequests.add(1);

      const success = check(revokeRes, {
        'disconnect successful': (r) => r.status === 200 || r.status === 404,
      });

      disconnectSuccessRate.add(success);

      if (success) {
        activeConnections.add(-1);
      } else {
        logError('Disconnect', revokeRes, {
          user_id: user.id,
          mcp_name: mcpName
        });
      }

      const duration = Date.now() - startTime;
      disconnectDuration.add(duration);
    }
  });

  sleep(randomIntBetween(1, 3));
}

/**
 * Scenario 6: Mixed Workload
 * Simulates realistic user behavior with various operations
 */
export function mixedWorkloadScenario() {
  const user = generateTestUser();
  const token = generateOAuthToken(user.id);

  concurrentUsers.add(1);

  group('Mixed Workload', () => {
    // Random operation selection with weighted probabilities
    const operation = randomIntBetween(1, 100);

    if (operation <= 30) {
      // 30% - Check status
      const statusRes = http.get(`${BASE_URL}/status`, {
        tags: { endpoint: 'status', operation: 'status_check' },
      });
      totalRequests.add(1);
      isSuccess(statusRes);

    } else if (operation <= 50) {
      // 20% - Get messages
      const messagesRes = http.get(`${BASE_URL}/messages`, {
        tags: { endpoint: 'messages', operation: 'get_messages' },
      });
      totalRequests.add(1);
      isSuccess(messagesRes);

    } else if (operation <= 70) {
      // 20% - Send message
      if (waitForStableAgent(5000)) {
        const messageRes = http.post(
          `${BASE_URL}/message`,
          JSON.stringify({
            content: `Test message ${randomString(10)}`,
            type: 'user',
          }),
          {
            headers: getHeaders(),
            tags: { endpoint: 'message', operation: 'send_message' },
            timeout: '30s',
          }
        );
        totalRequests.add(1);
        toolExecutionSuccessRate.add(isSuccess(messageRes));
      }

    } else if (operation <= 85) {
      // 15% - OAuth init
      const initRes = http.post(
        `${OAUTH_BASE_URL}/init`,
        JSON.stringify({
          mcp_name: `mcp_${randomString(8)}`,
          provider: ['github', 'google', 'azure'][randomIntBetween(0, 2)],
          redirect_uri: 'http://localhost:3000/oauth/callback',
        }),
        {
          headers: getHeaders(token),
          tags: { endpoint: 'oauth_init', operation: 'oauth_init' },
        }
      );
      totalRequests.add(1);
      mcpConnectionSuccessRate.add(isSuccess(initRes));

    } else if (operation <= 95) {
      // 10% - Token refresh
      const refreshRes = http.post(
        `${OAUTH_BASE_URL}/refresh`,
        JSON.stringify({
          mcp_config_id: `config_${randomString(16)}`,
        }),
        {
          headers: getHeaders(token),
          tags: { endpoint: 'oauth_refresh', operation: 'token_refresh' },
        }
      );
      totalRequests.add(1);

      if (isSuccess(refreshRes) || refreshRes.status === 404) {
        tokenRefreshCount.add(1);
      }

    } else {
      // 5% - Subscribe to events (SSE)
      const eventsRes = http.get(`${BASE_URL}/events`, {
        tags: { endpoint: 'events', operation: 'sse_subscribe' },
        timeout: '5s',
      });
      totalRequests.add(1);
      isSuccess(eventsRes);
    }
  });

  sleep(randomIntBetween(1, 5));
}

// ==================== SETUP & TEARDOWN ====================

/**
 * Setup function - runs once before all scenarios
 */
export function setup() {
  console.log('='.repeat(60));
  console.log('Starting K6 Load Test for AgentAPI');
  console.log('='.repeat(60));
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`OAuth URL: ${OAUTH_BASE_URL}`);
  console.log(`Total Duration: ~15 minutes`);
  console.log('='.repeat(60));

  // Verify API is accessible
  const statusRes = http.get(`${BASE_URL}/status`);

  if (statusRes.status !== 200) {
    console.error('ERROR: API is not accessible!');
    console.error(`Status: ${statusRes.status}`);
    console.error(`Body: ${statusRes.body}`);
    throw new Error('API health check failed');
  }

  console.log('API health check passed');

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function - runs once after all scenarios
 */
export function teardown(data) {
  const endTime = Date.now();
  const duration = (endTime - data.startTime) / 1000;

  console.log('='.repeat(60));
  console.log('K6 Load Test Completed');
  console.log('='.repeat(60));
  console.log(`Total Duration: ${duration.toFixed(2)} seconds`);
  console.log('='.repeat(60));
}

/**
 * Handle summary - generates HTML report
 */
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/load/summary.html': htmlReport(data),
    'tests/load/summary.json': JSON.stringify(data),
  };
}

/**
 * Text summary helper
 */
function textSummary(data, options = {}) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n' + indent + '='.repeat(60) + '\n';
  summary += indent + 'LOAD TEST SUMMARY\n';
  summary += indent + '='.repeat(60) + '\n\n';

  // Metrics
  summary += indent + 'Key Metrics:\n';
  summary += indent + '-'.repeat(60) + '\n';

  if (data.metrics.http_req_duration) {
    const p50 = data.metrics.http_req_duration.values['p(50)'];
    const p95 = data.metrics.http_req_duration.values['p(95)'];
    const p99 = data.metrics.http_req_duration.values['p(99)'];

    summary += indent + `  Response Time (p50): ${p50.toFixed(2)}ms\n`;
    summary += indent + `  Response Time (p95): ${p95.toFixed(2)}ms\n`;
    summary += indent + `  Response Time (p99): ${p99.toFixed(2)}ms\n`;
  }

  if (data.metrics.http_req_failed) {
    const errorRate = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
    summary += indent + `  Error Rate: ${errorRate}%\n`;
  }

  if (data.metrics.total_requests) {
    summary += indent + `  Total Requests: ${data.metrics.total_requests.values.count}\n`;
  }

  summary += '\n' + indent + 'Custom Metrics:\n';
  summary += indent + '-'.repeat(60) + '\n';

  if (data.metrics.mcp_connection_success_rate) {
    const rate = (data.metrics.mcp_connection_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  MCP Connection Success Rate: ${rate}%\n`;
  }

  if (data.metrics.tool_execution_success_rate) {
    const rate = (data.metrics.tool_execution_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  Tool Execution Success Rate: ${rate}%\n`;
  }

  if (data.metrics.auth_success_rate) {
    const rate = (data.metrics.auth_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  Auth Success Rate: ${rate}%\n`;
  }

  if (data.metrics.token_refresh_count) {
    summary += indent + `  Token Refreshes: ${data.metrics.token_refresh_count.values.count}\n`;
  }

  if (data.metrics.connection_failures) {
    summary += indent + `  Connection Failures: ${data.metrics.connection_failures.values.count}\n`;
  }

  if (data.metrics.timeout_count) {
    summary += indent + `  Timeouts: ${data.metrics.timeout_count.values.count}\n`;
  }

  summary += '\n' + indent + '='.repeat(60) + '\n';

  return summary;
}

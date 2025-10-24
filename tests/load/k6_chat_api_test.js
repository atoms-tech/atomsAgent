/**
 * K6 Load Testing Suite for AgentAPI Chat Completions Endpoint
 *
 * This comprehensive test suite covers:
 * - Non-streaming chat completions
 * - Streaming chat completions (SSE)
 * - Multiple model types (gemini-1.5-pro, gpt-4, claude-3-opus)
 * - Variable message lengths (short, medium, long)
 * - Concurrent user scenarios (1-50 VUs)
 * - Performance validation and error handling
 *
 * Run with: k6 run tests/load/k6_chat_api_test.js
 * Or with custom config: k6 run --env BASE_URL=http://your-api.com --env AUTH_TOKEN=your-token tests/load/k6_chat_api_test.js
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { randomString, randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// ==================== CONFIGURATION ====================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3284';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'Bearer test-token-12345';

// Support multiple auth tokens for testing token rotation
const AUTH_TOKENS = __ENV.AUTH_TOKENS
  ? __ENV.AUTH_TOKENS.split(',')
  : [AUTH_TOKEN];

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Basic load test - steady state
    basic_load: {
      executor: 'constant-arrival-rate',
      exec: 'basicLoadScenario',
      rate: 10,                    // 10 requests per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 10,
      maxVUs: 20,
      tags: { scenario: 'basic_load' },
    },

    // Scenario 2: Ramp up test - gradual increase
    ramp_up: {
      executor: 'ramping-arrival-rate',
      exec: 'rampUpScenario',
      startRate: 0,
      timeUnit: '1s',
      stages: [
        { duration: '1m', target: 20 },   // Ramp to 20 req/s
        { duration: '1m', target: 50 },   // Ramp to 50 req/s
        { duration: '1m', target: 100 },  // Ramp to 100 req/s
        { duration: '1m', target: 100 },  // Hold at 100 req/s
        { duration: '1m', target: 0 },    // Ramp down
      ],
      preAllocatedVUs: 50,
      maxVUs: 150,
      startTime: '2m',
      tags: { scenario: 'ramp_up' },
    },

    // Scenario 3: Stress test - spike
    stress_test: {
      executor: 'constant-arrival-rate',
      exec: 'stressTestScenario',
      rate: 200,                   // 200 requests per second
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 100,
      maxVUs: 300,
      startTime: '7m',
      tags: { scenario: 'stress_test' },
    },

    // Scenario 4: Sustained load - endurance
    sustained_load: {
      executor: 'constant-arrival-rate',
      exec: 'sustainedLoadScenario',
      rate: 50,                    // 50 requests per second
      timeUnit: '1s',
      duration: '10m',
      preAllocatedVUs: 50,
      maxVUs: 100,
      startTime: '8m',
      tags: { scenario: 'sustained_load' },
    },

    // Scenario 5: Streaming-specific test
    streaming_test: {
      executor: 'ramping-vus',
      exec: 'streamingTestScenario',
      startVUs: 1,
      stages: [
        { duration: '1m', target: 10 },   // Ramp to 10 concurrent streams
        { duration: '2m', target: 30 },   // Ramp to 30 concurrent streams
        { duration: '2m', target: 50 },   // Ramp to 50 concurrent streams
        { duration: '3m', target: 50 },   // Hold at 50 concurrent streams
        { duration: '1m', target: 0 },    // Ramp down
      ],
      startTime: '18m',
      tags: { scenario: 'streaming_test' },
    },
  },

  // Performance thresholds
  thresholds: {
    // Overall HTTP metrics
    'http_req_duration': ['p(95)<2000', 'p(99)<5000'],
    'http_req_failed': ['rate<0.05'],  // Error rate < 5%

    // Custom metrics thresholds
    'chat_completion_success_rate': ['rate>0.95'],
    'streaming_success_rate': ['rate>0.95'],

    // Non-streaming response time (should be faster)
    'non_streaming_duration': ['p(95)<1000', 'p(99)<2000'],

    // Streaming response time (includes full stream)
    'streaming_duration': ['p(95)<2000', 'p(99)<5000'],

    // Scenario-specific thresholds
    'http_req_duration{scenario:basic_load}': ['p(95)<1500'],
    'http_req_duration{scenario:ramp_up}': ['p(95)<2000'],
    'http_req_duration{scenario:stress_test}': ['p(95)<3000'],
    'http_req_duration{scenario:sustained_load}': ['p(95)<1800'],
    'http_req_duration{scenario:streaming_test}': ['p(95)<2500'],

    // Model-specific thresholds
    'http_req_duration{model:gemini-1.5-pro}': ['p(95)<2000'],
    'http_req_duration{model:gpt-4}': ['p(95)<2000'],
    'http_req_duration{model:claude-3-opus}': ['p(95)<2000'],

    // Message length thresholds
    'http_req_duration{msg_length:short}': ['p(95)<1000'],
    'http_req_duration{msg_length:medium}': ['p(95)<1500'],
    'http_req_duration{msg_length:long}': ['p(95)<2500'],
  },
};

// ==================== CUSTOM METRICS ====================

// Success rates
const chatCompletionSuccessRate = new Rate('chat_completion_success_rate');
const streamingSuccessRate = new Rate('streaming_success_rate');
const nonStreamingSuccessRate = new Rate('non_streaming_success_rate');

// Response times by category
const nonStreamingDuration = new Trend('non_streaming_duration');
const streamingDuration = new Trend('streaming_duration');
const ttfb = new Trend('time_to_first_byte');           // Time to first byte
const streamingLatency = new Trend('streaming_latency'); // Time to first SSE chunk

// Model-specific metrics
const geminiDuration = new Trend('gemini_duration');
const gpt4Duration = new Trend('gpt4_duration');
const claudeDuration = new Trend('claude_duration');

// Message length metrics
const shortMsgDuration = new Trend('short_msg_duration');
const mediumMsgDuration = new Trend('medium_msg_duration');
const longMsgDuration = new Trend('long_msg_duration');

// Counters
const totalChatRequests = new Counter('total_chat_requests');
const streamingRequests = new Counter('streaming_requests');
const nonStreamingRequests = new Counter('non_streaming_requests');
const authFailures = new Counter('auth_failures');
const validationErrors = new Counter('validation_errors');
const serverErrors = new Counter('server_errors');
const tokenCount = new Counter('token_count');

// Gauges
const concurrentStreams = new Gauge('concurrent_streams');
const activeRequests = new Gauge('active_requests');

// ==================== TEST DATA CONFIGURATION ====================

// Available models to test
const MODELS = [
  'gemini-1.5-pro',
  'gpt-4',
  'claude-3-opus',
  'gpt-4-turbo',
  'claude-3-sonnet',
];

// Message templates by length
const SHORT_MESSAGES = [
  'Hello!',
  'What is 2+2?',
  'List files',
  'Show status',
  'Help me',
];

const MEDIUM_MESSAGES = [
  'Can you help me understand how to implement a REST API in Go?',
  'What are the best practices for error handling in distributed systems?',
  'Explain the differences between microservices and monolithic architecture.',
  'How do I optimize database queries for better performance?',
  'What are the key principles of clean code architecture?',
];

const LONG_MESSAGES = [
  `I'm working on a large-scale distributed system that needs to handle millions of requests per day.
   I need help designing the architecture with proper load balancing, fault tolerance, and scalability.
   The system should support multiple tenants, have robust authentication and authorization,
   implement rate limiting, and provide comprehensive monitoring and alerting.
   Can you provide a detailed architecture proposal with specific technology recommendations?`,

  `I need to refactor a legacy monolithic application into microservices. The application has about
   500,000 lines of code, uses multiple databases, and has complex business logic. I need advice on
   how to approach this migration incrementally, what patterns to use for service communication,
   how to handle distributed transactions, and how to maintain backward compatibility during the transition.
   Please provide a step-by-step migration strategy with best practices.`,

  `I'm implementing a real-time analytics system that needs to process streaming data from multiple sources,
   perform complex aggregations, and provide sub-second query responses. The system should scale horizontally,
   handle late-arriving data, support exactly-once processing semantics, and integrate with our existing
   data warehouse. What architecture and technologies would you recommend for this use case?`,
];

const SYSTEM_PROMPTS = [
  'You are a helpful coding assistant.',
  'You are an expert software architect.',
  'You are a senior developer specializing in distributed systems.',
  null, // test without system prompt
];

// ==================== HELPER FUNCTIONS ====================

/**
 * Get random auth token from pool (for token rotation testing)
 */
function getAuthToken() {
  return randomItem(AUTH_TOKENS);
}

/**
 * Generate common headers for API requests
 */
function getHeaders(streaming = false) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': getAuthToken(),
  };

  if (streaming) {
    headers['Accept'] = 'text/event-stream';
  } else {
    headers['Accept'] = 'application/json';
  }

  return headers;
}

/**
 * Generate a random chat completion request
 */
function generateChatRequest(messageLength = 'short', streaming = false, modelOverride = null) {
  let messages;
  let msgLengthTag;

  switch (messageLength) {
    case 'short':
      messages = [{ role: 'user', content: randomItem(SHORT_MESSAGES) }];
      msgLengthTag = 'short';
      break;
    case 'medium':
      messages = [{ role: 'user', content: randomItem(MEDIUM_MESSAGES) }];
      msgLengthTag = 'medium';
      break;
    case 'long':
      messages = [{ role: 'user', content: randomItem(LONG_MESSAGES) }];
      msgLengthTag = 'long';
      break;
    default:
      messages = [{ role: 'user', content: randomItem(MEDIUM_MESSAGES) }];
      msgLengthTag = 'medium';
  }

  const systemPrompt = randomItem(SYSTEM_PROMPTS);
  if (systemPrompt) {
    messages.unshift({ role: 'system', content: systemPrompt });
  }

  const model = modelOverride || randomItem(MODELS);

  return {
    model: model,
    messages: messages,
    temperature: 0.7 + (Math.random() * 0.3), // 0.7 to 1.0
    max_tokens: randomIntBetween(100, 4000),
    top_p: 0.9 + (Math.random() * 0.1), // 0.9 to 1.0
    stream: streaming,
    user: `load-test-user-${randomString(8)}`,
  };
}

/**
 * Check non-streaming response validity
 */
function checkNonStreamingResponse(response, model, msgLength) {
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'has response body': (r) => r.body && r.body.length > 0,
    'content-type is json': (r) => r.headers['Content-Type']?.includes('application/json'),
    'response is valid JSON': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch {
        return false;
      }
    },
    'has id field': (r) => {
      try {
        return JSON.parse(r.body).id !== undefined;
      } catch {
        return false;
      }
    },
    'has object field': (r) => {
      try {
        return JSON.parse(r.body).object === 'chat.completion';
      } catch {
        return false;
      }
    },
    'has choices array': (r) => {
      try {
        const data = JSON.parse(r.body);
        return Array.isArray(data.choices) && data.choices.length > 0;
      } catch {
        return false;
      }
    },
    'has message content': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.choices[0]?.message?.content !== undefined;
      } catch {
        return false;
      }
    },
    'has finish_reason': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.choices[0]?.finish_reason !== undefined;
      } catch {
        return false;
      }
    },
    'has usage info': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.usage?.total_tokens !== undefined;
      } catch {
        return false;
      }
    },
    'usage has prompt_tokens': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.usage?.prompt_tokens > 0;
      } catch {
        return false;
      }
    },
    'usage has completion_tokens': (r) => {
      try {
        const data = JSON.parse(r.body);
        return data.usage?.completion_tokens > 0;
      } catch {
        return false;
      }
    },
  });

  // Track token usage
  if (success && response.status === 200) {
    try {
      const data = JSON.parse(response.body);
      if (data.usage?.total_tokens) {
        tokenCount.add(data.usage.total_tokens);
      }
    } catch {
      // Ignore parsing errors for metrics
    }
  }

  return success;
}

/**
 * Check streaming response validity
 */
function checkStreamingResponse(response, model, msgLength) {
  let firstChunkTime = 0;
  let chunkCount = 0;
  let hasFinishReason = false;
  let contentReceived = '';

  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'content-type is event-stream': (r) => r.headers['Content-Type']?.includes('text/event-stream'),
    'has response body': (r) => r.body && r.body.length > 0,
  });

  if (!success) {
    return false;
  }

  // Parse SSE stream
  const lines = response.body.split('\n');
  const startTime = Date.now();

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();

    if (line.startsWith('data: ')) {
      const data = line.substring(6);

      // Check for stream end
      if (data === '[DONE]') {
        hasFinishReason = true;
        break;
      }

      try {
        const chunk = JSON.parse(data);
        chunkCount++;

        // Record time to first chunk
        if (chunkCount === 1) {
          firstChunkTime = Date.now() - startTime;
          streamingLatency.add(firstChunkTime);
        }

        // Accumulate content
        if (chunk.choices && chunk.choices[0]?.delta?.content) {
          contentReceived += chunk.choices[0].delta.content;
        }

        // Check for finish_reason
        if (chunk.choices && chunk.choices[0]?.finish_reason) {
          hasFinishReason = true;
        }
      } catch (e) {
        console.error(`Failed to parse SSE chunk: ${data}`);
      }
    }
  }

  // Validate streaming requirements
  const streamingValid = check(response, {
    'received multiple chunks': () => chunkCount > 0,
    'has finish reason': () => hasFinishReason,
    'received content': () => contentReceived.length > 0,
    'first chunk within 1s': () => firstChunkTime < 1000,
  });

  return streamingValid;
}

/**
 * Log detailed error information
 */
function logError(endpoint, response, context = {}) {
  console.error(`[ERROR] ${endpoint}:`, {
    status: response.status,
    body: response.body ? response.body.substring(0, 500) : 'empty',
    headers: JSON.stringify(response.headers),
    duration: response.timings.duration,
    ...context,
  });

  // Track error types
  if (response.status === 401 || response.status === 403) {
    authFailures.add(1);
  } else if (response.status === 400 || response.status === 422) {
    validationErrors.add(1);
  } else if (response.status >= 500) {
    serverErrors.add(1);
  }
}

/**
 * Execute non-streaming chat completion
 */
function executeChatCompletion(messageLength = 'medium', modelOverride = null) {
  const req = generateChatRequest(messageLength, false, modelOverride);
  const startTime = Date.now();

  activeRequests.add(1);

  const response = http.post(
    `${BASE_URL}/v1/chat/completions`,
    JSON.stringify(req),
    {
      headers: getHeaders(false),
      tags: {
        endpoint: 'chat_completions',
        model: req.model,
        msg_length: messageLength,
        streaming: 'false',
      },
      timeout: '30s',
    }
  );

  activeRequests.add(-1);
  totalChatRequests.add(1);
  nonStreamingRequests.add(1);

  const duration = Date.now() - startTime;
  nonStreamingDuration.add(duration);
  ttfb.add(response.timings.waiting);

  // Track model-specific metrics
  switch (req.model) {
    case 'gemini-1.5-pro':
      geminiDuration.add(duration);
      break;
    case 'gpt-4':
    case 'gpt-4-turbo':
      gpt4Duration.add(duration);
      break;
    case 'claude-3-opus':
    case 'claude-3-sonnet':
      claudeDuration.add(duration);
      break;
  }

  // Track message length metrics
  switch (messageLength) {
    case 'short':
      shortMsgDuration.add(duration);
      break;
    case 'medium':
      mediumMsgDuration.add(duration);
      break;
    case 'long':
      longMsgDuration.add(duration);
      break;
  }

  const success = checkNonStreamingResponse(response, req.model, messageLength);
  chatCompletionSuccessRate.add(success);
  nonStreamingSuccessRate.add(success);

  if (!success) {
    logError('Non-streaming chat completion', response, {
      model: req.model,
      msg_length: messageLength,
    });
  }

  return success;
}

/**
 * Execute streaming chat completion
 */
function executeStreamingCompletion(messageLength = 'medium', modelOverride = null) {
  const req = generateChatRequest(messageLength, true, modelOverride);
  const startTime = Date.now();

  activeRequests.add(1);
  concurrentStreams.add(1);

  const response = http.post(
    `${BASE_URL}/v1/chat/completions`,
    JSON.stringify(req),
    {
      headers: getHeaders(true),
      tags: {
        endpoint: 'chat_completions',
        model: req.model,
        msg_length: messageLength,
        streaming: 'true',
      },
      timeout: '60s', // Longer timeout for streaming
    }
  );

  activeRequests.add(-1);
  concurrentStreams.add(-1);
  totalChatRequests.add(1);
  streamingRequests.add(1);

  const duration = Date.now() - startTime;
  streamingDuration.add(duration);
  ttfb.add(response.timings.waiting);

  // Track model-specific metrics
  switch (req.model) {
    case 'gemini-1.5-pro':
      geminiDuration.add(duration);
      break;
    case 'gpt-4':
    case 'gpt-4-turbo':
      gpt4Duration.add(duration);
      break;
    case 'claude-3-opus':
    case 'claude-3-sonnet':
      claudeDuration.add(duration);
      break;
  }

  // Track message length metrics
  switch (messageLength) {
    case 'short':
      shortMsgDuration.add(duration);
      break;
    case 'medium':
      mediumMsgDuration.add(duration);
      break;
    case 'long':
      longMsgDuration.add(duration);
      break;
  }

  const success = checkStreamingResponse(response, req.model, messageLength);
  chatCompletionSuccessRate.add(success);
  streamingSuccessRate.add(success);

  if (!success) {
    logError('Streaming chat completion', response, {
      model: req.model,
      msg_length: messageLength,
    });
  }

  return success;
}

// ==================== SCENARIO FUNCTIONS ====================

/**
 * Scenario 1: Basic Load Test
 * Steady 10 req/s for 2 minutes with mixed message lengths
 */
export function basicLoadScenario() {
  group('Basic Load - Mixed Requests', () => {
    const messageLength = randomItem(['short', 'medium', 'long']);
    const isStreaming = Math.random() < 0.3; // 30% streaming

    if (isStreaming) {
      executeStreamingCompletion(messageLength);
    } else {
      executeChatCompletion(messageLength);
    }
  });

  sleep(randomIntBetween(1, 3));
}

/**
 * Scenario 2: Ramp Up Test
 * Gradually increase load from 0 to 100 req/s
 */
export function rampUpScenario() {
  group('Ramp Up - Incremental Load', () => {
    const messageLength = randomItem(['short', 'medium', 'long']);
    const isStreaming = Math.random() < 0.5; // 50% streaming

    if (isStreaming) {
      executeStreamingCompletion(messageLength);
    } else {
      executeChatCompletion(messageLength);
    }
  });

  sleep(randomIntBetween(1, 2));
}

/**
 * Scenario 3: Stress Test
 * Spike to 200 req/s for 1 minute
 */
export function stressTestScenario() {
  group('Stress Test - High Load Spike', () => {
    // Favor shorter messages during stress test
    const messageLength = Math.random() < 0.7 ? 'short' : 'medium';
    const isStreaming = Math.random() < 0.2; // 20% streaming (less stress)

    if (isStreaming) {
      executeStreamingCompletion(messageLength);
    } else {
      executeChatCompletion(messageLength);
    }
  });

  // Minimal sleep during stress test
  sleep(randomIntBetween(0, 1));
}

/**
 * Scenario 4: Sustained Load
 * 50 req/s for 10 minutes with error recovery
 */
export function sustainedLoadScenario() {
  group('Sustained Load - Endurance Test', () => {
    const messageLength = randomItem(['short', 'medium', 'long']);
    const isStreaming = Math.random() < 0.4; // 40% streaming

    let success = false;
    let retries = 0;
    const maxRetries = 3;

    // Implement retry logic for sustained test
    while (!success && retries < maxRetries) {
      if (isStreaming) {
        success = executeStreamingCompletion(messageLength);
      } else {
        success = executeChatCompletion(messageLength);
      }

      if (!success) {
        retries++;
        sleep(1); // Wait before retry
      }
    }
  });

  sleep(randomIntBetween(1, 3));
}

/**
 * Scenario 5: Streaming-Specific Test
 * Focus on streaming performance with concurrent streams
 */
export function streamingTestScenario() {
  group('Streaming Test - Concurrent Streams', () => {
    // Test all message lengths for streaming
    const messageLengths = ['short', 'medium', 'long'];

    for (const msgLength of messageLengths) {
      executeStreamingCompletion(msgLength);
      sleep(0.5); // Small delay between streams
    }

    // Test model-specific streaming
    const models = ['gemini-1.5-pro', 'gpt-4', 'claude-3-opus'];
    const model = randomItem(models);
    executeStreamingCompletion('medium', model);
  });

  sleep(randomIntBetween(2, 5));
}

// ==================== SETUP & TEARDOWN ====================

/**
 * Setup - runs once before all scenarios
 */
export function setup() {
  console.log('='.repeat(80));
  console.log('K6 Load Test - AgentAPI Chat Completions Endpoint');
  console.log('='.repeat(80));
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Auth Tokens: ${AUTH_TOKENS.length} configured`);
  console.log(`Test Duration: ~28 minutes (multiple scenarios)`);
  console.log(`Models: ${MODELS.join(', ')}`);
  console.log('='.repeat(80));

  // Verify API accessibility
  const healthCheck = http.get(`${BASE_URL}/status`, {
    headers: { 'Accept': 'application/json' },
  });

  if (healthCheck.status !== 200) {
    console.error('ERROR: API health check failed!');
    console.error(`Status: ${healthCheck.status}`);
    console.error(`Body: ${healthCheck.body}`);
    throw new Error('API is not accessible');
  }

  console.log('✓ API health check passed');
  console.log('='.repeat(80));

  return {
    startTime: Date.now(),
    baseUrl: BASE_URL,
  };
}

/**
 * Teardown - runs once after all scenarios
 */
export function teardown(data) {
  const endTime = Date.now();
  const durationSeconds = (endTime - data.startTime) / 1000;
  const durationMinutes = (durationSeconds / 60).toFixed(2);

  console.log('='.repeat(80));
  console.log('K6 Load Test Completed');
  console.log('='.repeat(80));
  console.log(`Total Duration: ${durationMinutes} minutes (${durationSeconds.toFixed(2)} seconds)`);
  console.log('='.repeat(80));
}

/**
 * Handle summary - generates reports
 */
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/load/chat_api_summary.html': htmlReport(data),
    'tests/load/chat_api_summary.json': JSON.stringify(data, null, 2),
  };
}

/**
 * Generate text summary of test results
 */
function textSummary(data, options = {}) {
  const indent = options.indent || '';

  let summary = '\n' + indent + '='.repeat(80) + '\n';
  summary += indent + 'CHAT API LOAD TEST SUMMARY\n';
  summary += indent + '='.repeat(80) + '\n\n';

  // Overall metrics
  summary += indent + 'Overall Performance:\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.http_req_duration) {
    const p50 = data.metrics.http_req_duration.values['p(50)'];
    const p95 = data.metrics.http_req_duration.values['p(95)'];
    const p99 = data.metrics.http_req_duration.values['p(99)'];
    const avg = data.metrics.http_req_duration.values.avg;

    summary += indent + `  Response Time (avg): ${avg.toFixed(2)}ms\n`;
    summary += indent + `  Response Time (p50): ${p50.toFixed(2)}ms\n`;
    summary += indent + `  Response Time (p95): ${p95.toFixed(2)}ms\n`;
    summary += indent + `  Response Time (p99): ${p99.toFixed(2)}ms\n`;
  }

  if (data.metrics.http_req_failed) {
    const errorRate = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
    summary += indent + `  Error Rate: ${errorRate}%\n`;
  }

  if (data.metrics.total_chat_requests) {
    summary += indent + `  Total Chat Requests: ${data.metrics.total_chat_requests.values.count}\n`;
  }

  // Streaming vs Non-streaming
  summary += '\n' + indent + 'Streaming vs Non-Streaming:\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.non_streaming_duration) {
    const p95 = data.metrics.non_streaming_duration.values['p(95)'];
    summary += indent + `  Non-Streaming (p95): ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.streaming_duration) {
    const p95 = data.metrics.streaming_duration.values['p(95)'];
    summary += indent + `  Streaming (p95): ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.streaming_latency) {
    const p95 = data.metrics.streaming_latency.values['p(95)'];
    summary += indent + `  Time to First Chunk (p95): ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.streaming_requests) {
    summary += indent + `  Streaming Requests: ${data.metrics.streaming_requests.values.count}\n`;
  }

  if (data.metrics.non_streaming_requests) {
    summary += indent + `  Non-Streaming Requests: ${data.metrics.non_streaming_requests.values.count}\n`;
  }

  // Success rates
  summary += '\n' + indent + 'Success Rates:\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.chat_completion_success_rate) {
    const rate = (data.metrics.chat_completion_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  Overall Success Rate: ${rate}%\n`;
  }

  if (data.metrics.streaming_success_rate) {
    const rate = (data.metrics.streaming_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  Streaming Success Rate: ${rate}%\n`;
  }

  if (data.metrics.non_streaming_success_rate) {
    const rate = (data.metrics.non_streaming_success_rate.values.rate * 100).toFixed(2);
    summary += indent + `  Non-Streaming Success Rate: ${rate}%\n`;
  }

  // Model performance
  summary += '\n' + indent + 'Model Performance (p95):\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.gemini_duration) {
    const p95 = data.metrics.gemini_duration.values['p(95)'];
    summary += indent + `  Gemini 1.5 Pro: ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.gpt4_duration) {
    const p95 = data.metrics.gpt4_duration.values['p(95)'];
    summary += indent + `  GPT-4: ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.claude_duration) {
    const p95 = data.metrics.claude_duration.values['p(95)'];
    summary += indent + `  Claude 3: ${p95.toFixed(2)}ms\n`;
  }

  // Message length performance
  summary += '\n' + indent + 'Message Length Performance (p95):\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.short_msg_duration) {
    const p95 = data.metrics.short_msg_duration.values['p(95)'];
    summary += indent + `  Short Messages: ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.medium_msg_duration) {
    const p95 = data.metrics.medium_msg_duration.values['p(95)'];
    summary += indent + `  Medium Messages: ${p95.toFixed(2)}ms\n`;
  }

  if (data.metrics.long_msg_duration) {
    const p95 = data.metrics.long_msg_duration.values['p(95)'];
    summary += indent + `  Long Messages: ${p95.toFixed(2)}ms\n`;
  }

  // Error breakdown
  summary += '\n' + indent + 'Error Breakdown:\n';
  summary += indent + '-'.repeat(80) + '\n';

  if (data.metrics.auth_failures) {
    summary += indent + `  Auth Failures (401/403): ${data.metrics.auth_failures.values.count}\n`;
  }

  if (data.metrics.validation_errors) {
    summary += indent + `  Validation Errors (400/422): ${data.metrics.validation_errors.values.count}\n`;
  }

  if (data.metrics.server_errors) {
    summary += indent + `  Server Errors (5xx): ${data.metrics.server_errors.values.count}\n`;
  }

  // Token usage
  if (data.metrics.token_count) {
    summary += '\n' + indent + 'Token Usage:\n';
    summary += indent + '-'.repeat(80) + '\n';
    summary += indent + `  Total Tokens Processed: ${data.metrics.token_count.values.count}\n`;
  }

  summary += '\n' + indent + '='.repeat(80) + '\n';

  return summary;
}

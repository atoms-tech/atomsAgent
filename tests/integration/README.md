# OAuth Integration Tests

This directory contains comprehensive integration tests for the OAuth flow in agentapi.

## Overview

The test suite validates the complete OAuth 2.0 authentication flow including:
- OAuth initiation and state management
- Token exchange and callback handling
- Token refresh mechanisms
- MCP (Model Context Protocol) integration with OAuth
- Redis-based token caching and session management
- Circuit breaker patterns
- Rate limiting
- Error scenarios and edge cases

## Test Structure

### Test Files

- `oauth_integration_test.go` - Main integration test suite

### Test Categories

#### 1. OAuth Initiation Tests (`TestOAuthInitiation`)
- Valid provider configuration
- Invalid provider handling
- State generation and storage in Redis
- State expiration

#### 2. OAuth Callback Tests (`TestOAuthCallback`)
- Successful token exchange
- CSRF protection (invalid state detection)
- Expired authorization code handling
- Token encryption verification

#### 3. Token Refresh Tests (`TestOAuthTokenRefresh`)
- Successful token refresh
- Expired refresh token handling
- Auto-refresh threshold detection

#### 4. MCP Integration Tests (`TestMCPConnectionWithOAuth`)
- Full OAuth → Token → MCP flow
- Token refresh before MCP connection
- OAuth token usage in MCP client

#### 5. Redis Integration Tests (`TestRedisIntegration`)
- Session data persistence
- Token cache operations
- Cleanup on logout
- Health checks

#### 6. Error Scenario Tests (`TestErrorScenarios`)
- Invalid state (CSRF attacks)
- Expired codes
- Token refresh failures
- Redis connection failures
- Encryption key validation
- Token validation errors

#### 7. Circuit Breaker Tests (`TestCircuitBreaker`)
- Circuit opening on failures
- Circuit recovery
- Fallback behavior

#### 8. Rate Limiting Tests (`TestRateLimiting`)
- Rate limit enforcement
- 429 response handling
- Rate limit reset

#### 9. Concurrent Operations Tests (`TestConcurrentOAuthFlows`)
- Concurrent token storage
- Concurrent token refresh
- Race condition handling

#### 10. Metrics Tests (`TestOAuthMetrics`)
- Circuit breaker metrics
- Token cache statistics

## Prerequisites

### Required Services

1. **Redis** (optional for most tests, uses in-memory fallback)
   ```bash
   # Start Redis with Docker
   docker run -d -p 6379:6379 redis:7-alpine
   ```

2. **MCP Server** (optional, for full integration tests)
   ```bash
   # Start MCP server on port 8000
   # See MCP documentation for setup
   ```

### Environment Variables

```bash
# Optional: Custom Redis URL
export REDIS_URL="redis://localhost:6379/15"

# Optional: OAuth provider credentials (for real provider tests)
export OAUTH_CLIENT_ID="your-client-id"
export OAUTH_CLIENT_SECRET="your-client-secret"
```

## Running Tests

### Run All Integration Tests

```bash
cd /Users/kooshapari/temp-PRODVERCEL/485/kush/agentapi
go test ./tests/integration -v
```

### Run Specific Test

```bash
# Test OAuth initiation
go test ./tests/integration -v -run TestOAuthInitiation

# Test token refresh
go test ./tests/integration -v -run TestOAuthTokenRefresh

# Test circuit breaker
go test ./tests/integration -v -run TestCircuitBreaker
```

### Run with Coverage

```bash
go test ./tests/integration -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run with Race Detection

```bash
go test ./tests/integration -v -race
```

### Run in Short Mode (Skip Long Tests)

```bash
go test ./tests/integration -v -short
```

## Test Dependencies

The tests use:
- `stretchr/testify` - Assertions and test utilities
- `httptest` - Mock HTTP servers for OAuth providers
- In-memory Redis fallback when Redis is unavailable
- Mock MCP servers for integration testing

## Key Features

### Mock OAuth Server

The tests include a fully functional mock OAuth server that simulates:
- Authorization endpoint (`/oauth/authorize`)
- Token exchange endpoint (`/oauth/token`)
- Token revocation endpoint (`/oauth/revoke`)
- Both authorization code and refresh token grant types

### Automatic Fallbacks

Tests automatically fall back to in-memory implementations when external services are unavailable:
- Redis → In-memory cache
- MCP Server → Mock/skip tests

### Concurrent Testing

Tests validate thread-safety with concurrent operations:
- Multiple users authenticating simultaneously
- Concurrent token refreshes
- Race condition detection

### Security Testing

Tests validate security measures:
- CSRF protection via state parameter
- Token encryption at rest (AES-256-GCM)
- Secure token storage in Redis
- Proper cleanup on logout

## Test Data

Tests use consistent test data:
- User ID: `test-user-123`
- Org ID: `test-org-456`
- Provider: GitHub (configurable)
- Redis DB: 15 (test database)

All test data is cleaned up after each test run.

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest

    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run integration tests
        run: |
          go test ./tests/integration -v -race -cover
        env:
          REDIS_URL: redis://localhost:6379/15
```

## Troubleshooting

### Redis Connection Issues

If tests fail due to Redis connection:
```
Error: failed to create redis client: connection failed
```

**Solution**: Tests will automatically fall back to in-memory mode. To use real Redis:
```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Or use environment variable
export REDIS_URL="redis://localhost:6379/15"
```

### MCP Server Not Available

If MCP integration tests skip:
```
SKIP: MCP server not available
```

**Solution**: This is expected. Tests gracefully skip MCP-specific tests when the server isn't running.

### Rate Limit Tests Failing

If rate limit tests are flaky:
```
Error: expected some requests to be rate limited
```

**Solution**: Tests may need adjustment based on system load. Increase the request count or decrease the rate limit for more consistent results.

## Best Practices

1. **Isolation**: Each test is isolated with its own test context
2. **Cleanup**: All test data is cleaned up in `teardownTestContext`
3. **Idempotency**: Tests can be run multiple times without side effects
4. **Parallelization**: Tests can run in parallel (use `t.Parallel()` where safe)
5. **Determinism**: Uses fixed test data and deterministic mocks

## Future Enhancements

- [ ] Add tests for additional OAuth providers (Google, Microsoft, Slack)
- [ ] Test webhook validation
- [ ] Test OAuth token rotation policies
- [ ] Add performance benchmarks
- [ ] Test distributed rate limiting across multiple instances
- [ ] Add chaos engineering tests (network failures, timeouts)

## Contributing

When adding new tests:

1. Follow the existing test structure
2. Use the `setupTestContext` and `teardownTestContext` helpers
3. Add proper cleanup in defer statements
4. Document new test cases in this README
5. Ensure tests pass in both Redis and in-memory modes
6. Add appropriate skip conditions for optional dependencies

## License

Same as parent project.

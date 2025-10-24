package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedClientRetry tests the retry logic with exponential backoff
func TestEnhancedClientRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail the first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "service unavailable",
			})
			return
		}
		// Succeed on the 3rd attempt
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ConnectResponse{
			Success: true,
			Message: "connected",
		})
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)

	ctx := context.Background()
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "test-command",
	}

	start := time.Now()
	err := client.ConnectWithRetry(ctx, "test-client", config)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "Should have made 3 attempts")
	assert.Greater(t, duration, 100*time.Millisecond, "Should have backoff delay")
	assert.Less(t, duration, 2*time.Second, "Should not exceed max backoff")
}

// TestEnhancedClientNonRetryableError tests that non-retryable errors don't retry
func TestEnhancedClientNonRetryableError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Return 400 Bad Request (non-retryable)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "bad request",
		})
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)

	ctx := context.Background()
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "test-command",
	}

	err := client.ConnectWithRetry(ctx, "test-client", config)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "Should only attempt once for non-retryable error")
}

// TestEnhancedClientContextTimeout tests context timeout behavior
func TestEnhancedClientContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "test-command",
	}

	start := time.Now()
	err := client.ConnectWithRetry(ctx, "test-client", config)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, duration, 1*time.Second, "Should respect context timeout")
}

// TestEnhancedClientCallToolWithRetry tests call tool with retry
func TestEnhancedClientCallToolWithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ToolCallResponse{
			Success: true,
			Result: map[string]any{
				"output": "success",
			},
		})
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)

	ctx := context.Background()
	result, err := client.CallToolWithRetry(ctx, "test-client", "test-tool", map[string]any{
		"param": "value",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["output"])
	assert.Equal(t, 2, attempts)
}

// TestEnhancedClientListToolsWithRetry tests list tools with retry
func TestEnhancedClientListToolsWithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ListToolsResponse{
			Success: true,
			Tools: []Tool{
				{Name: "tool1", Description: "Test tool 1"},
				{Name: "tool2", Description: "Test tool 2"},
			},
		})
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)

	ctx := context.Background()
	tools, err := client.ListToolsWithRetry(ctx, "test-client")

	assert.NoError(t, err)
	assert.Len(t, tools, 2)
	assert.Equal(t, "tool1", tools[0].Name)
	assert.Equal(t, 2, attempts)
}

// TestBackoffWithJitter tests the backoff calculation with jitter
func TestBackoffWithJitter(t *testing.T) {
	client := NewEnhancedFastMCPHTTPClient("http://localhost:8000")

	// Test multiple backoff calculations
	for attempt := 0; attempt < 5; attempt++ {
		backoff := client.calculateBackoffWithJitter(attempt)

		// Calculate expected range
		expectedBase := time.Duration(float64(enhancedInitialRetryDelay) * float64(1<<attempt))
		if expectedBase > enhancedMaxRetryDelay {
			expectedBase = enhancedMaxRetryDelay
		}

		minBackoff := time.Duration(float64(expectedBase) * (1 - enhancedJitterPercent))
		maxBackoff := time.Duration(float64(expectedBase) * (1 + enhancedJitterPercent))

		assert.GreaterOrEqual(t, backoff, minBackoff, "Backoff should be >= min")
		assert.LessOrEqual(t, backoff, maxBackoff, "Backoff should be <= max")

		t.Logf("Attempt %d: backoff=%v (range: %v - %v)", attempt, backoff, minBackoff, maxBackoff)
	}
}

// TestRedisDLQ tests the Redis Dead Letter Queue implementation
func TestRedisDLQ(t *testing.T) {
	// Skip if Redis is not available
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping DLQ tests")
	}
	defer redisClient.Close()

	dlq := NewRedisDLQ(redisClient)

	// Clean up before test
	redisClient.Del(ctx, dlqListKey)

	t.Run("Store and Get", func(t *testing.T) {
		op := FailedOperation{
			ID:        "test-op-1",
			ClientID:  "test-client",
			Operation: "connect",
			Endpoint:  "/api/connect",
			RequestBody: map[string]interface{}{
				"test": "data",
			},
			LastError:   "connection failed",
			RetryCount:  3,
			CreatedAt:   time.Now(),
			LastAttempt: time.Now(),
		}

		err := dlq.Store(ctx, op)
		require.NoError(t, err)

		retrieved, err := dlq.Get(ctx, op.ID)
		require.NoError(t, err)
		assert.Equal(t, op.ID, retrieved.ID)
		assert.Equal(t, op.ClientID, retrieved.ClientID)
		assert.Equal(t, op.Operation, retrieved.Operation)
	})

	t.Run("List", func(t *testing.T) {
		ops, err := dlq.List(ctx, 100)
		require.NoError(t, err)
		assert.Greater(t, len(ops), 0)
	})

	t.Run("Count", func(t *testing.T) {
		count, err := dlq.Count(ctx)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("GetByOperation", func(t *testing.T) {
		ops, err := dlq.GetByOperation(ctx, "connect", 10)
		require.NoError(t, err)
		assert.Greater(t, len(ops), 0)
		for _, op := range ops {
			assert.Equal(t, "connect", op.Operation)
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats, err := dlq.GetStats(ctx)
		require.NoError(t, err)
		assert.Greater(t, stats.TotalOperations, int64(0))
		assert.NotEmpty(t, stats.OperationCounts)
	})

	t.Run("Delete", func(t *testing.T) {
		err := dlq.Delete(ctx, "test-op-1")
		require.NoError(t, err)

		_, err = dlq.Get(ctx, "test-op-1")
		assert.Error(t, err)
	})
}

// TestDLQCleanup tests the cleanup functionality
func TestDLQCleanup(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping DLQ tests")
	}
	defer redisClient.Close()

	dlq := NewRedisDLQ(redisClient)

	// Create old operation
	oldOp := FailedOperation{
		ID:          "old-op",
		ClientID:    "test-client",
		Operation:   "connect",
		Endpoint:    "/api/connect",
		LastError:   "test error",
		RetryCount:  3,
		CreatedAt:   time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
		LastAttempt: time.Now().Add(-10 * 24 * time.Hour),
	}

	err := dlq.Store(ctx, oldOp)
	require.NoError(t, err)

	// Cleanup operations older than 7 days
	err = dlq.Cleanup(ctx, 7*24*time.Hour)
	require.NoError(t, err)

	// Verify old operation is deleted
	_, err = dlq.Get(ctx, oldOp.ID)
	assert.Error(t, err)
}

// TestMetricsIntegration tests metrics recording
func TestMetricsIntegration(t *testing.T) {
	metrics := InitMCPMetrics("test")
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.RetryCount)
	assert.NotNil(t, metrics.RetryBackoff)
	assert.NotNil(t, metrics.OperationTotal)
	assert.NotNil(t, metrics.OperationTime)
	assert.NotNil(t, metrics.DLQOperations)

	// Test metric recording
	metrics.RetryCount.WithLabelValues("connect", "retryable_error").Inc()
	metrics.RetryBackoff.WithLabelValues("connect").Observe(0.1)
	metrics.OperationTotal.WithLabelValues("connect", "success").Inc()
	metrics.OperationTime.WithLabelValues("connect").Observe(1.5)
	metrics.DLQOperations.Inc()
}

// BenchmarkBackoffCalculation benchmarks the backoff calculation
func BenchmarkBackoffCalculation(b *testing.B) {
	client := NewEnhancedFastMCPHTTPClient("http://localhost:8000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.calculateBackoffWithJitter(i % 10)
	}
}

// BenchmarkRetryLogic benchmarks the full retry logic
func BenchmarkRetryLogic(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ConnectResponse{
			Success: true,
		})
	}))
	defer server.Close()

	client := NewEnhancedFastMCPHTTPClient(server.URL)
	ctx := context.Background()
	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "test-command",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.ConnectWithRetry(ctx, "test-client", config)
	}
}

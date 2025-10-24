package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.MinRetryBackoff)
	assert.Equal(t, 3*time.Second, config.MaxRetryBackoff)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 10, config.PoolSize)
	assert.Equal(t, ProtocolNative, config.PreferredProtocol)
}

func TestNewRedisClient_InvalidConfig(t *testing.T) {
	config := DefaultConfig()
	// Both URL and RESTBaseURL are empty
	config.URL = ""
	config.RESTBaseURL = ""

	_, err := NewRedisClient(config)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidURL)
}

func TestRedisClient_Integration(t *testing.T) {
	// Skip if Redis is not available
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL environment variable not set")
	}

	config := DefaultConfig()
	config.URL = redisURL

	client, err := NewRedisClient(config)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		err := client.Health()
		assert.NoError(t, err)
	})

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test:key:1"
		value := "test-value"

		// Set
		err := client.Set(ctx, key, value, 0)
		require.NoError(t, err)

		// Get
		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Cleanup
		err = client.Delete(ctx, key)
		assert.NoError(t, err)
	})

	t.Run("SetWithTTL", func(t *testing.T) {
		key := "test:key:ttl"
		value := "temporary-value"

		// Set with TTL
		err := client.Set(ctx, key, value, 1*time.Second)
		require.NoError(t, err)

		// Get immediately
		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Wait for expiration
		time.Sleep(2 * time.Second)

		// Get after expiration
		result, err = client.Get(ctx, key)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "test:key:delete"
		value := "to-be-deleted"

		// Set
		err := client.Set(ctx, key, value, 0)
		require.NoError(t, err)

		// Verify exists
		exists, err := client.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete
		err = client.Delete(ctx, key)
		require.NoError(t, err)

		// Verify deleted
		exists, err = client.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Exists", func(t *testing.T) {
		key := "test:key:exists"

		// Check non-existent key
		exists, err := client.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		// Set key
		err = client.Set(ctx, key, "value", 0)
		require.NoError(t, err)

		// Check existing key
		exists, err = client.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		err = client.Delete(ctx, key)
		assert.NoError(t, err)
	})

	t.Run("Increment", func(t *testing.T) {
		key := "test:counter"

		// Delete if exists from previous test
		_ = client.Delete(ctx, key)

		// Increment (creates key with value 1)
		err := client.Increment(ctx, key)
		require.NoError(t, err)

		// Get value
		val, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "1", val)

		// Increment again
		err = client.Increment(ctx, key)
		require.NoError(t, err)

		// Get value
		val, err = client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "2", val)

		// Cleanup
		err = client.Delete(ctx, key)
		assert.NoError(t, err)
	})
}

func TestRedisClient_REST(t *testing.T) {
	// Skip if REST credentials are not available
	restBaseURL := os.Getenv("REDIS_REST_URL")
	token := os.Getenv("REDIS_REST_TOKEN")

	if restBaseURL == "" || token == "" {
		t.Skip("REDIS_REST_URL or REDIS_REST_TOKEN environment variables not set")
	}

	config := DefaultConfig()
	config.RESTBaseURL = restBaseURL
	config.Token = token
	config.PreferredProtocol = ProtocolREST

	client, err := NewRedisClient(config)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		err := client.Health()
		assert.NoError(t, err)
		assert.Equal(t, ProtocolREST, client.GetActiveProtocol())
	})

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test:rest:key:1"
		value := "rest-test-value"

		// Set
		err := client.Set(ctx, key, value, 0)
		require.NoError(t, err)

		// Get
		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Cleanup
		err = client.Delete(ctx, key)
		assert.NoError(t, err)
	})
}

func TestRedisClient_Closed(t *testing.T) {
	// Create a client with REST only to avoid connection issues
	config := DefaultConfig()
	config.RESTBaseURL = "https://example.com"
	config.Token = "dummy-token"
	config.PreferredProtocol = ProtocolREST

	client, err := NewRedisClient(config)
	require.NoError(t, err)

	// Close the client
	err = client.Close()
	require.NoError(t, err)

	ctx := context.Background()

	// All operations should return ErrClientClosed
	t.Run("GetAfterClose", func(t *testing.T) {
		_, err := client.Get(ctx, "key")
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("SetAfterClose", func(t *testing.T) {
		err := client.Set(ctx, "key", "value", 0)
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("DeleteAfterClose", func(t *testing.T) {
		err := client.Delete(ctx, "key")
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("ExistsAfterClose", func(t *testing.T) {
		_, err := client.Exists(ctx, "key")
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("IncrementAfterClose", func(t *testing.T) {
		err := client.Increment(ctx, "key")
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	t.Run("HealthAfterClose", func(t *testing.T) {
		err := client.Health()
		assert.ErrorIs(t, err, ErrClientClosed)
	})

	// Closing again should not error
	t.Run("DoubleClose", func(t *testing.T) {
		err := client.Close()
		assert.NoError(t, err)
	})
}

func TestRedisClient_ContextCancellation(t *testing.T) {
	config := DefaultConfig()
	config.RESTBaseURL = "https://example.com"
	config.Token = "dummy-token"
	config.PreferredProtocol = ProtocolREST

	client, err := NewRedisClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should respect context cancellation
	_, err = client.Get(ctx, "key")
	assert.Error(t, err)
}

func BenchmarkRedisClient_Set(b *testing.B) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		b.Skip("REDIS_URL environment variable not set")
	}

	config := DefaultConfig()
	config.URL = redisURL

	client, err := NewRedisClient(config)
	require.NoError(b, err)
	defer client.Close()

	ctx := context.Background()
	key := "bench:set"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Set(ctx, key, "value", 0)
	}
}

func BenchmarkRedisClient_Get(b *testing.B) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		b.Skip("REDIS_URL environment variable not set")
	}

	config := DefaultConfig()
	config.URL = redisURL

	client, err := NewRedisClient(config)
	require.NoError(b, err)
	defer client.Close()

	ctx := context.Background()
	key := "bench:get"

	// Setup
	_ = client.Set(ctx, key, "value", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Get(ctx, key)
	}
}

func BenchmarkRedisClient_Increment(b *testing.B) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		b.Skip("REDIS_URL environment variable not set")
	}

	config := DefaultConfig()
	config.URL = redisURL

	client, err := NewRedisClient(config)
	require.NoError(b, err)
	defer client.Close()

	ctx := context.Background()
	key := "bench:incr"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Increment(ctx, key)
	}
}

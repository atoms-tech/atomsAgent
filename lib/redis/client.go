package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client errors
var (
	ErrClientClosed       = errors.New("redis client is closed")
	ErrInvalidURL         = errors.New("invalid Redis URL")
	ErrConnectionFailed   = errors.New("connection failed")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// Protocol represents the connection protocol type
type Protocol string

const (
	ProtocolNative Protocol = "native"
	ProtocolREST   Protocol = "rest"
)

// Config holds Redis client configuration
type Config struct {
	// Native Redis connection URL (rediss://...)
	URL string

	// REST API configuration
	RESTBaseURL string
	Token       string

	// Connection pool settings
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolSize        int
	MinIdleConns    int
	MaxIdleTime     time.Duration

	// Preferred protocol (native or rest)
	PreferredProtocol Protocol
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxRetries:        3,
		MinRetryBackoff:   100 * time.Millisecond,
		MaxRetryBackoff:   3 * time.Second,
		DialTimeout:       5 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		PoolSize:          10,
		MinIdleConns:      2,
		MaxIdleTime:       5 * time.Minute,
		PreferredProtocol: ProtocolNative,
	}
}

// RedisClient provides a dual-protocol Redis client with automatic fallback
type RedisClient struct {
	config Config

	// Native Redis client
	nativeClient *redis.Client

	// REST HTTP client
	restClient  *http.Client
	restBaseURL string
	restToken   string

	// State management
	mu          sync.RWMutex
	closed      bool
	activeProto Protocol
}

// NewRedisClient creates a new Redis client with both native and REST support
func NewRedisClient(config Config) (*RedisClient, error) {
	if config.URL == "" && config.RESTBaseURL == "" {
		return nil, fmt.Errorf("%w: either URL or RESTBaseURL must be provided", ErrInvalidURL)
	}

	client := &RedisClient{
		config:      config,
		restBaseURL: config.RESTBaseURL,
		restToken:   config.Token,
		activeProto: config.PreferredProtocol,
	}

	// Initialize REST client (always available as fallback)
	if config.RESTBaseURL != "" {
		client.restClient = &http.Client{
			Timeout: config.ReadTimeout + config.WriteTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        config.PoolSize,
				MaxIdleConnsPerHost: config.PoolSize,
				IdleConnTimeout:     config.MaxIdleTime,
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		}
	}

	// Initialize native Redis client if URL is provided
	if config.URL != "" {
		if err := client.initNativeClient(); err != nil {
			// If native client fails, fallback to REST if available
			if config.RESTBaseURL != "" {
				client.activeProto = ProtocolREST
			} else {
				return nil, fmt.Errorf("failed to initialize native client: %w", err)
			}
		}
	} else {
		// Only REST available
		client.activeProto = ProtocolREST
	}

	return client, nil
}

// initNativeClient initializes the native Redis connection
func (c *RedisClient) initNativeClient() error {
	opts, err := redis.ParseURL(c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Apply custom configuration
	opts.MaxRetries = c.config.MaxRetries
	opts.MinRetryBackoff = c.config.MinRetryBackoff
	opts.MaxRetryBackoff = c.config.MaxRetryBackoff
	opts.DialTimeout = c.config.DialTimeout
	opts.ReadTimeout = c.config.ReadTimeout
	opts.WriteTimeout = c.config.WriteTimeout
	opts.PoolSize = c.config.PoolSize
	opts.MinIdleConns = c.config.MinIdleConns
	opts.ConnMaxIdleTime = c.config.MaxIdleTime

	// Create client
	c.nativeClient = redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	if err := c.nativeClient.Ping(ctx).Err(); err != nil {
		c.nativeClient.Close()
		c.nativeClient = nil
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	return nil
}

// Get retrieves a value from Redis
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return "", ErrClientClosed
	}
	proto := c.activeProto
	c.mu.RUnlock()

	return c.executeWithFallback(ctx, func(p Protocol) (string, error) {
		if p == ProtocolNative && c.nativeClient != nil {
			val, err := c.nativeClient.Get(ctx, key).Result()
			if err == redis.Nil {
				return "", nil // Key does not exist
			}
			return val, err
		}

		// REST fallback
		return c.restGet(ctx, key)
	}, proto)
}

// Set stores a value in Redis with optional TTL
func (c *RedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	proto := c.activeProto
	c.mu.RUnlock()

	_, err := c.executeWithFallback(ctx, func(p Protocol) (string, error) {
		if p == ProtocolNative && c.nativeClient != nil {
			return "", c.nativeClient.Set(ctx, key, value, ttl).Err()
		}

		// REST fallback
		return "", c.restSet(ctx, key, value, ttl)
	}, proto)

	return err
}

// Delete removes a key from Redis
func (c *RedisClient) Delete(ctx context.Context, key string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	proto := c.activeProto
	c.mu.RUnlock()

	_, err := c.executeWithFallback(ctx, func(p Protocol) (string, error) {
		if p == ProtocolNative && c.nativeClient != nil {
			return "", c.nativeClient.Del(ctx, key).Err()
		}

		// REST fallback
		return "", c.restDelete(ctx, key)
	}, proto)

	return err
}

// Exists checks if a key exists in Redis
func (c *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false, ErrClientClosed
	}
	proto := c.activeProto
	c.mu.RUnlock()

	result, err := c.executeWithFallback(ctx, func(p Protocol) (string, error) {
		if p == ProtocolNative && c.nativeClient != nil {
			count, err := c.nativeClient.Exists(ctx, key).Result()
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%d", count), nil
		}

		// REST fallback
		val, err := c.restExists(ctx, key)
		return fmt.Sprintf("%d", val), err
	}, proto)

	if err != nil {
		return false, err
	}

	return result != "0", nil
}

// Increment increments a key's value by 1
func (c *RedisClient) Increment(ctx context.Context, key string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	proto := c.activeProto
	c.mu.RUnlock()

	_, err := c.executeWithFallback(ctx, func(p Protocol) (string, error) {
		if p == ProtocolNative && c.nativeClient != nil {
			return "", c.nativeClient.Incr(ctx, key).Err()
		}

		// REST fallback
		return "", c.restIncrement(ctx, key)
	}, proto)

	return err
}

// Health performs a health check on the Redis connection
func (c *RedisClient) Health() error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try native first
	if c.nativeClient != nil {
		if err := c.nativeClient.Ping(ctx).Err(); err == nil {
			c.mu.Lock()
			c.activeProto = ProtocolNative
			c.mu.Unlock()
			return nil
		}
	}

	// Fallback to REST
	if c.restClient != nil && c.restBaseURL != "" {
		if err := c.restPing(ctx); err == nil {
			c.mu.Lock()
			c.activeProto = ProtocolREST
			c.mu.Unlock()
			return nil
		}
	}

	return ErrConnectionFailed
}

// Close gracefully shuts down the Redis client
func (c *RedisClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.nativeClient != nil {
		if err := c.nativeClient.Close(); err != nil {
			return fmt.Errorf("failed to close native client: %w", err)
		}
	}

	if c.restClient != nil {
		c.restClient.CloseIdleConnections()
	}

	return nil
}

// GetActiveProtocol returns the currently active protocol
func (c *RedisClient) GetActiveProtocol() Protocol {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeProto
}

// executeWithFallback executes an operation with automatic protocol fallback
func (c *RedisClient) executeWithFallback(ctx context.Context, fn func(Protocol) (string, error), preferredProto Protocol) (string, error) {
	var lastErr error

	// Try preferred protocol first
	result, err := fn(preferredProto)
	if err == nil {
		return result, nil
	}
	lastErr = err

	// Fallback to alternative protocol
	fallbackProto := ProtocolREST
	if preferredProto == ProtocolREST {
		fallbackProto = ProtocolNative
	}

	// Only fallback if the alternative is available
	if (fallbackProto == ProtocolNative && c.nativeClient != nil) ||
		(fallbackProto == ProtocolREST && c.restClient != nil && c.restBaseURL != "") {
		result, err = fn(fallbackProto)
		if err == nil {
			// Update active protocol on successful fallback
			c.mu.Lock()
			c.activeProto = fallbackProto
			c.mu.Unlock()
			return result, nil
		}
		lastErr = err
	}

	return "", lastErr
}

// REST API implementations

type restResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

func (c *RedisClient) restPing(ctx context.Context) error {
	_, err := c.restCommand(ctx, "PING", nil)
	return err
}

func (c *RedisClient) restGet(ctx context.Context, key string) (string, error) {
	resp, err := c.restCommand(ctx, "GET", []string{key})
	if err != nil {
		return "", err
	}

	if resp.Result == nil {
		return "", nil
	}

	if str, ok := resp.Result.(string); ok {
		return str, nil
	}

	return fmt.Sprintf("%v", resp.Result), nil
}

func (c *RedisClient) restSet(ctx context.Context, key, value string, ttl time.Duration) error {
	args := []string{key, value}

	if ttl > 0 {
		args = append(args, "EX", fmt.Sprintf("%d", int64(ttl.Seconds())))
	}

	_, err := c.restCommand(ctx, "SET", args)
	return err
}

func (c *RedisClient) restDelete(ctx context.Context, key string) error {
	_, err := c.restCommand(ctx, "DEL", []string{key})
	return err
}

func (c *RedisClient) restExists(ctx context.Context, key string) (int64, error) {
	resp, err := c.restCommand(ctx, "EXISTS", []string{key})
	if err != nil {
		return 0, err
	}

	if num, ok := resp.Result.(float64); ok {
		return int64(num), nil
	}

	return 0, fmt.Errorf("unexpected result type: %T", resp.Result)
}

func (c *RedisClient) restIncrement(ctx context.Context, key string) error {
	_, err := c.restCommand(ctx, "INCR", []string{key})
	return err
}

func (c *RedisClient) restCommand(ctx context.Context, command string, args []string) (*restResponse, error) {
	// Build request URL
	endpoint := fmt.Sprintf("%s/%s", strings.TrimRight(c.restBaseURL, "/"), strings.ToLower(command))

	// Add args to URL path
	if len(args) > 0 {
		for _, arg := range args {
			endpoint = fmt.Sprintf("%s/%s", endpoint, url.PathEscape(arg))
		}
	}

	// Create request with retry logic
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff
			backoff := c.config.MinRetryBackoff * time.Duration(1<<uint(attempt-1))
			if backoff > c.config.MaxRetryBackoff {
				backoff = c.config.MaxRetryBackoff
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Add authorization header
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.restToken))

		resp, err = c.restClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Success
		break
	}

	if resp == nil {
		if lastErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
		}
		return nil, ErrMaxRetriesExceeded
	}

	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REST API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var result restResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("Redis error: %s", result.Error)
	}

	return &result, nil
}

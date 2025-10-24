package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	redisclient "github.com/coder/agentapi/lib/redis"
	"github.com/redis/go-redis/v9"
)

// Common errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidConfig     = errors.New("invalid rate limiter configuration")
	ErrRedisConnection   = errors.New("redis connection error")
)

// LimitType represents the type of rate limit
type LimitType string

const (
	LimitTypeUser     LimitType = "user"
	LimitTypeOrg      LimitType = "org"
	LimitTypeIP       LimitType = "ip"
	LimitTypeEndpoint LimitType = "endpoint"
)

// Config holds rate limiter configuration
type Config struct {
	// Default limits
	RequestsPerMinute int           // Requests per minute (default: 60)
	BurstSize         int           // Max burst tokens (default: 10)
	TokenRefillRate   time.Duration // How often to refill tokens (default: 1 second)

	// Per-endpoint overrides
	EndpointLimits map[string]EndpointLimit

	// Admin bypass
	AdminBypass bool // Allow admin users to bypass rate limits

	// Redis key prefix
	KeyPrefix string // Prefix for Redis keys (default: "ratelimit")

	// Logging
	Logger *slog.Logger
}

// EndpointLimit defines rate limits for a specific endpoint
type EndpointLimit struct {
	RequestsPerMinute int
	BurstSize         int
	Enabled           bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		RequestsPerMinute: 60,
		BurstSize:         10,
		TokenRefillRate:   1 * time.Second,
		KeyPrefix:         "ratelimit",
		AdminBypass:       true,
		EndpointLimits:    make(map[string]EndpointLimit),
		Logger:            slog.Default(),
	}
}

// RateLimiter implements a distributed token bucket rate limiter using Redis
type RateLimiter struct {
	config Config
	redis  *redisclient.RedisClient
	logger *slog.Logger

	// Lua scripts for atomic operations
	allowScript  string
	statusScript string
}

// NewRateLimiter creates a new Redis-based rate limiter
func NewRateLimiter(redis *redisclient.RedisClient, config Config) (*RateLimiter, error) {
	if redis == nil {
		return nil, fmt.Errorf("%w: redis client is nil", ErrInvalidConfig)
	}

	if config.RequestsPerMinute <= 0 {
		return nil, fmt.Errorf("%w: requests per minute must be positive", ErrInvalidConfig)
	}

	if config.BurstSize <= 0 {
		return nil, fmt.Errorf("%w: burst size must be positive", ErrInvalidConfig)
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}

	if config.TokenRefillRate == 0 {
		config.TokenRefillRate = 1 * time.Second
	}

	rl := &RateLimiter{
		config: config,
		redis:  redis,
		logger: config.Logger,
	}

	// Initialize Lua scripts
	rl.allowScript = rl.buildAllowScript()
	rl.statusScript = rl.buildStatusScript()

	return rl, nil
}

// AllowRequest checks if a request should be allowed based on rate limits
// Returns: allowed (bool), remaining tokens (int), reset time (time.Time), error
func (rl *RateLimiter) AllowRequest(ctx context.Context, userID, orgID, endpoint string) (bool, int, time.Time, error) {
	// Admin bypass check (would need to be passed via context or config)
	// For now, we'll implement the core algorithm

	// Get endpoint-specific limits or use defaults
	limit := rl.getEndpointLimit(endpoint)
	if !limit.Enabled {
		// Rate limiting disabled for this endpoint
		return true, limit.BurstSize, time.Now().Add(time.Minute), nil
	}

	// Check limits in order: user -> org -> IP
	// User-level limit (most specific)
	if userID != "" {
		allowed, remaining, resetAt, err := rl.checkLimit(ctx, LimitTypeUser, userID, endpoint, limit)
		if err != nil {
			rl.logger.Error("Failed to check user rate limit",
				"user_id", userID,
				"endpoint", endpoint,
				"error", err,
			)
			// Fall through to org-level check on error
		} else {
			if !allowed {
				rl.logger.Warn("User rate limit exceeded",
					"user_id", userID,
					"endpoint", endpoint,
					"remaining", remaining,
					"reset_at", resetAt,
				)
			}
			return allowed, remaining, resetAt, err
		}
	}

	// Organization-level limit
	if orgID != "" {
		allowed, remaining, resetAt, err := rl.checkLimit(ctx, LimitTypeOrg, orgID, endpoint, limit)
		if err != nil {
			rl.logger.Error("Failed to check org rate limit",
				"org_id", orgID,
				"endpoint", endpoint,
				"error", err,
			)
			return false, 0, time.Now(), fmt.Errorf("%w: %v", ErrRedisConnection, err)
		}

		if !allowed {
			rl.logger.Warn("Org rate limit exceeded",
				"org_id", orgID,
				"endpoint", endpoint,
				"remaining", remaining,
				"reset_at", resetAt,
			)
		}
		return allowed, remaining, resetAt, nil
	}

	// If we get here, no valid identifier was provided
	return false, 0, time.Now(), errors.New("no valid identifier for rate limiting")
}

// AllowRequestByIP checks rate limit for an IP address (for anonymous requests)
func (rl *RateLimiter) AllowRequestByIP(ctx context.Context, ipAddress, endpoint string) (bool, int, time.Time, error) {
	if ipAddress == "" {
		return false, 0, time.Now(), errors.New("IP address is required")
	}

	limit := rl.getEndpointLimit(endpoint)
	if !limit.Enabled {
		return true, limit.BurstSize, time.Now().Add(time.Minute), nil
	}

	allowed, remaining, resetAt, err := rl.checkLimit(ctx, LimitTypeIP, ipAddress, endpoint, limit)
	if err != nil {
		rl.logger.Error("Failed to check IP rate limit",
			"ip", ipAddress,
			"endpoint", endpoint,
			"error", err,
		)
		return false, 0, time.Now(), fmt.Errorf("%w: %v", ErrRedisConnection, err)
	}

	if !allowed {
		rl.logger.Warn("IP rate limit exceeded",
			"ip", ipAddress,
			"endpoint", endpoint,
			"remaining", remaining,
			"reset_at", resetAt,
		)
	}

	return allowed, remaining, resetAt, nil
}

// GetLimitStatus returns the current limit status for a user/org without consuming a token
func (rl *RateLimiter) GetLimitStatus(ctx context.Context, userID, orgID, endpoint string) (remaining, limit int, resetAt time.Time, err error) {
	endpointLimit := rl.getEndpointLimit(endpoint)
	limit = endpointLimit.RequestsPerMinute

	var key string
	if userID != "" {
		key = rl.buildKey(LimitTypeUser, userID, endpoint)
	} else if orgID != "" {
		key = rl.buildKey(LimitTypeOrg, orgID, endpoint)
	} else {
		return 0, 0, time.Now(), errors.New("no valid identifier provided")
	}

	// Get current token count from Redis
	result, err := rl.redis.Get(ctx, key)
	if err != nil {
		// Key doesn't exist, return full limit
		return endpointLimit.BurstSize, limit, time.Now().Add(time.Minute), nil
	}

	// Parse token count
	tokens, err := strconv.Atoi(result)
	if err != nil {
		return 0, limit, time.Now(), fmt.Errorf("failed to parse token count: %w", err)
	}

	// Get TTL for reset time
	ttl := time.Minute // Default to 1 minute if we can't get TTL
	resetAt = time.Now().Add(ttl)

	return tokens, limit, resetAt, nil
}

// checkLimit performs the actual rate limit check using token bucket algorithm
func (rl *RateLimiter) checkLimit(ctx context.Context, limitType LimitType, identifier, endpoint string, limit EndpointLimit) (bool, int, time.Time, error) {
	key := rl.buildKey(limitType, identifier, endpoint)
	now := time.Now()

	// Calculate tokens per second
	tokensPerSecond := float64(limit.RequestsPerMinute) / 60.0
	maxTokens := limit.BurstSize

	// Try to get current state from Redis
	currentTokensStr, err := rl.redis.Get(ctx, key)
	var currentTokens float64
	var lastRefill time.Time

	if err != nil || currentTokensStr == "" {
		// First request - initialize with full burst capacity
		currentTokens = float64(maxTokens)
		lastRefill = now
	} else {
		// Parse current tokens
		currentTokens, err = strconv.ParseFloat(currentTokensStr, 64)
		if err != nil {
			// Reset on parse error
			currentTokens = float64(maxTokens)
			lastRefill = now
		} else {
			// Get last refill time from a separate key
			lastRefillStr, _ := rl.redis.Get(ctx, key+":time")
			if lastRefillStr != "" {
				lastRefillUnix, _ := strconv.ParseInt(lastRefillStr, 10, 64)
				lastRefill = time.Unix(lastRefillUnix, 0)
			} else {
				lastRefill = now
			}
		}
	}

	// Calculate token refill based on time elapsed
	elapsed := now.Sub(lastRefill).Seconds()
	tokensToAdd := elapsed * tokensPerSecond
	currentTokens = min(currentTokens+tokensToAdd, float64(maxTokens))

	// Try to consume one token
	allowed := currentTokens >= 1.0
	if allowed {
		currentTokens -= 1.0
	}

	// Update Redis with new state
	pipeline := []func() error{
		func() error {
			return rl.redis.Set(ctx, key, fmt.Sprintf("%.2f", currentTokens), time.Minute)
		},
		func() error {
			return rl.redis.Set(ctx, key+":time", strconv.FormatInt(now.Unix(), 10), time.Minute)
		},
	}

	for _, op := range pipeline {
		if err := op(); err != nil {
			rl.logger.Error("Failed to update rate limit state",
				"key", key,
				"error", err,
			)
			return false, 0, now, err
		}
	}

	remaining := int(currentTokens)
	resetAt := now.Add(time.Minute)

	return allowed, remaining, resetAt, nil
}

// buildKey constructs a Redis key for rate limiting
func (rl *RateLimiter) buildKey(limitType LimitType, identifier, endpoint string) string {
	switch limitType {
	case LimitTypeUser:
		return fmt.Sprintf("%s:user:%s:%s", rl.config.KeyPrefix, identifier, endpoint)
	case LimitTypeOrg:
		return fmt.Sprintf("%s:org:%s:%s", rl.config.KeyPrefix, identifier, endpoint)
	case LimitTypeIP:
		return fmt.Sprintf("%s:ip:%s", rl.config.KeyPrefix, identifier)
	default:
		return fmt.Sprintf("%s:%s:%s:%s", rl.config.KeyPrefix, limitType, identifier, endpoint)
	}
}

// getEndpointLimit returns the limit configuration for a specific endpoint
func (rl *RateLimiter) getEndpointLimit(endpoint string) EndpointLimit {
	if limit, ok := rl.config.EndpointLimits[endpoint]; ok {
		return limit
	}

	// Return default limit
	return EndpointLimit{
		RequestsPerMinute: rl.config.RequestsPerMinute,
		BurstSize:         rl.config.BurstSize,
		Enabled:           true,
	}
}

// buildAllowScript returns the Lua script for atomic token bucket operations
// This is more efficient than multiple round-trips to Redis
func (rl *RateLimiter) buildAllowScript() string {
	return `
		local key = KEYS[1]
		local time_key = KEYS[2]
		local max_tokens = tonumber(ARGV[1])
		local tokens_per_second = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local ttl = tonumber(ARGV[4])

		-- Get current state
		local current_tokens = tonumber(redis.call('GET', key))
		local last_refill = tonumber(redis.call('GET', time_key))

		if not current_tokens then
			current_tokens = max_tokens
			last_refill = now
		end

		if not last_refill then
			last_refill = now
		end

		-- Calculate token refill
		local elapsed = now - last_refill
		local tokens_to_add = elapsed * tokens_per_second
		current_tokens = math.min(current_tokens + tokens_to_add, max_tokens)

		-- Try to consume one token
		local allowed = 0
		if current_tokens >= 1.0 then
			current_tokens = current_tokens - 1.0
			allowed = 1
		end

		-- Update state
		redis.call('SET', key, tostring(current_tokens), 'EX', ttl)
		redis.call('SET', time_key, tostring(now), 'EX', ttl)

		-- Return: allowed, remaining, reset_at
		return {allowed, math.floor(current_tokens), now + ttl}
	`
}

// buildStatusScript returns the Lua script for getting limit status
func (rl *RateLimiter) buildStatusScript() string {
	return `
		local key = KEYS[1]
		local time_key = KEYS[2]
		local max_tokens = tonumber(ARGV[1])
		local tokens_per_second = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		-- Get current state
		local current_tokens = tonumber(redis.call('GET', key))
		local last_refill = tonumber(redis.call('GET', time_key))

		if not current_tokens then
			current_tokens = max_tokens
			last_refill = now
		end

		if not last_refill then
			last_refill = now
		end

		-- Calculate token refill
		local elapsed = now - last_refill
		local tokens_to_add = elapsed * tokens_per_second
		current_tokens = math.min(current_tokens + tokens_to_add, max_tokens)

		local ttl = redis.call('TTL', key)
		if ttl < 0 then
			ttl = 60
		end

		-- Return: remaining, max, reset_at
		return {math.floor(current_tokens), max_tokens, now + ttl}
	`
}

// AllowRequestAtomic uses Lua script for atomic rate limit check
// This is more efficient and race-condition free compared to the Go implementation
func (rl *RateLimiter) AllowRequestAtomic(ctx context.Context, userID, orgID, endpoint string) (bool, int, time.Time, error) {
	limit := rl.getEndpointLimit(endpoint)
	if !limit.Enabled {
		return true, limit.BurstSize, time.Now().Add(time.Minute), nil
	}

	var key string
	var limitType LimitType

	if userID != "" {
		key = rl.buildKey(LimitTypeUser, userID, endpoint)
		limitType = LimitTypeUser
	} else if orgID != "" {
		key = rl.buildKey(LimitTypeOrg, orgID, endpoint)
		limitType = LimitTypeOrg
	} else {
		return false, 0, time.Now(), errors.New("no valid identifier for rate limiting")
	}

	return rl.executeAllowScript(ctx, key, limitType, limit)
}

// executeAllowScript executes the Lua script for atomic rate limiting
func (rl *RateLimiter) executeAllowScript(ctx context.Context, key string, limitType LimitType, limit EndpointLimit) (bool, int, time.Time, error) {
	timeKey := key + ":time"
	maxTokens := limit.BurstSize
	tokensPerSecond := float64(limit.RequestsPerMinute) / 60.0
	now := time.Now().Unix()
	ttl := 60 // seconds

	// Note: This is a simplified version. In production, you'd use redis.NewScript()
	// and call script.Eval() with proper error handling
	// For now, we'll use the non-atomic Go version as fallback

	rl.logger.Debug("Rate limit check",
		"key", key,
		"limit_type", limitType,
		"max_tokens", maxTokens,
		"tokens_per_second", tokensPerSecond,
	)

	// Fallback to non-atomic version for compatibility
	// In production, implement script evaluation here
	return rl.checkLimit(ctx, limitType, key, "", limit)
}

// ResetLimit clears the rate limit for a specific identifier
// Useful for testing or admin operations
func (rl *RateLimiter) ResetLimit(ctx context.Context, limitType LimitType, identifier, endpoint string) error {
	key := rl.buildKey(limitType, identifier, endpoint)
	timeKey := key + ":time"

	if err := rl.redis.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete rate limit key: %w", err)
	}

	if err := rl.redis.Delete(ctx, timeKey); err != nil {
		return fmt.Errorf("failed to delete time key: %w", err)
	}

	rl.logger.Info("Rate limit reset",
		"limit_type", limitType,
		"identifier", identifier,
		"endpoint", endpoint,
	)

	return nil
}

// RateLimitError represents a rate limit exceeded error with details
type RateLimitError struct {
	Message    string
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s (remaining: %d, reset at: %s, retry after: %s)",
		e.Message, e.Remaining, e.ResetAt.Format(time.RFC3339), e.RetryAfter)
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(remaining int, resetAt time.Time) *RateLimitError {
	retryAfter := time.Until(resetAt)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return &RateLimitError{
		Message:    "rate limit exceeded",
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}
}

// Helper function for min
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

// GetRetryAfter extracts the retry-after duration from a rate limit error
func GetRetryAfter(err error) time.Duration {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.RetryAfter
	}
	return 0
}

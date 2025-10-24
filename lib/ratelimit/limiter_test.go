package ratelimit

import (
	"context"
	"testing"
	"time"

	"log/slog"

	redisclient "github.com/coder/agentapi/lib/redis"
)

// TestNewRateLimiter tests rate limiter creation
func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name      string
		redis     *redisclient.RedisClient
		config    Config
		wantError bool
	}{
		{
			name:      "nil redis client",
			redis:     nil,
			config:    DefaultConfig(),
			wantError: true,
		},
		{
			name:  "invalid requests per minute",
			redis: &redisclient.RedisClient{},
			config: Config{
				RequestsPerMinute: -1,
				BurstSize:         10,
			},
			wantError: true,
		},
		{
			name:  "invalid burst size",
			redis: &redisclient.RedisClient{},
			config: Config{
				RequestsPerMinute: 60,
				BurstSize:         -1,
			},
			wantError: true,
		},
		{
			name:      "valid config",
			redis:     &redisclient.RedisClient{},
			config:    DefaultConfig(),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := NewRateLimiter(tt.redis, tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if rl == nil {
					t.Error("expected rate limiter but got nil")
				}
			}
		})
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.RequestsPerMinute != 60 {
		t.Errorf("expected RequestsPerMinute=60, got %d", config.RequestsPerMinute)
	}

	if config.BurstSize != 10 {
		t.Errorf("expected BurstSize=10, got %d", config.BurstSize)
	}

	if config.KeyPrefix != "ratelimit" {
		t.Errorf("expected KeyPrefix='ratelimit', got %s", config.KeyPrefix)
	}

	if !config.AdminBypass {
		t.Error("expected AdminBypass=true")
	}

	if config.EndpointLimits == nil {
		t.Error("expected EndpointLimits to be initialized")
	}
}

// TestBuildKey tests Redis key construction
func TestBuildKey(t *testing.T) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()
	rl, err := NewRateLimiter(redis, config)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}

	tests := []struct {
		name       string
		limitType  LimitType
		identifier string
		endpoint   string
		expected   string
	}{
		{
			name:       "user limit",
			limitType:  LimitTypeUser,
			identifier: "user123",
			endpoint:   "/api/v1/sessions",
			expected:   "ratelimit:user:user123:/api/v1/sessions",
		},
		{
			name:       "org limit",
			limitType:  LimitTypeOrg,
			identifier: "org456",
			endpoint:   "/api/v1/sessions",
			expected:   "ratelimit:org:org456:/api/v1/sessions",
		},
		{
			name:       "ip limit",
			limitType:  LimitTypeIP,
			identifier: "192.168.1.1",
			endpoint:   "",
			expected:   "ratelimit:ip:192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := rl.buildKey(tt.limitType, tt.identifier, tt.endpoint)
			if key != tt.expected {
				t.Errorf("expected key=%s, got %s", tt.expected, key)
			}
		})
	}
}

// TestGetEndpointLimit tests endpoint limit retrieval
func TestGetEndpointLimit(t *testing.T) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()

	// Add custom endpoint limit
	config.EndpointLimits["/api/v1/upload"] = EndpointLimit{
		RequestsPerMinute: 10,
		BurstSize:         2,
		Enabled:           true,
	}

	rl, err := NewRateLimiter(redis, config)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}

	tests := []struct {
		name            string
		endpoint        string
		expectedRPM     int
		expectedBurst   int
		expectedEnabled bool
	}{
		{
			name:            "custom endpoint",
			endpoint:        "/api/v1/upload",
			expectedRPM:     10,
			expectedBurst:   2,
			expectedEnabled: true,
		},
		{
			name:            "default endpoint",
			endpoint:        "/api/v1/sessions",
			expectedRPM:     60,
			expectedBurst:   10,
			expectedEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := rl.getEndpointLimit(tt.endpoint)
			if limit.RequestsPerMinute != tt.expectedRPM {
				t.Errorf("expected RPM=%d, got %d", tt.expectedRPM, limit.RequestsPerMinute)
			}
			if limit.BurstSize != tt.expectedBurst {
				t.Errorf("expected BurstSize=%d, got %d", tt.expectedBurst, limit.BurstSize)
			}
			if limit.Enabled != tt.expectedEnabled {
				t.Errorf("expected Enabled=%v, got %v", tt.expectedEnabled, limit.Enabled)
			}
		})
	}
}

// TestRateLimitError tests rate limit error
func TestRateLimitError(t *testing.T) {
	resetAt := time.Now().Add(30 * time.Second)
	err := NewRateLimitError(0, resetAt)

	if err.Remaining != 0 {
		t.Errorf("expected Remaining=0, got %d", err.Remaining)
	}

	if err.ResetAt.Unix() != resetAt.Unix() {
		t.Errorf("expected ResetAt=%v, got %v", resetAt, err.ResetAt)
	}

	if err.RetryAfter <= 0 {
		t.Error("expected RetryAfter to be positive")
	}

	// Test error message
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
}

// TestIsRateLimitError tests error type checking
func TestIsRateLimitError(t *testing.T) {
	rateLimitErr := NewRateLimitError(0, time.Now().Add(time.Minute))
	genericErr := ErrInvalidConfig

	if !IsRateLimitError(rateLimitErr) {
		t.Error("expected IsRateLimitError to return true for RateLimitError")
	}

	if IsRateLimitError(genericErr) {
		t.Error("expected IsRateLimitError to return false for generic error")
	}
}

// TestGetRetryAfter tests retry-after extraction
func TestGetRetryAfter(t *testing.T) {
	resetAt := time.Now().Add(45 * time.Second)
	rateLimitErr := NewRateLimitError(0, resetAt)
	genericErr := ErrInvalidConfig

	retryAfter := GetRetryAfter(rateLimitErr)
	if retryAfter <= 0 {
		t.Error("expected positive RetryAfter duration")
	}

	retryAfter = GetRetryAfter(genericErr)
	if retryAfter != 0 {
		t.Error("expected zero RetryAfter for generic error")
	}
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a        float64
		b        float64
		expected float64
	}{
		{1.0, 2.0, 1.0},
		{5.0, 3.0, 3.0},
		{0.0, 0.0, 0.0},
		{-1.0, 1.0, -1.0},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%f, %f) = %f, expected %f", tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestLuaScripts tests that Lua scripts are properly initialized
func TestLuaScripts(t *testing.T) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()
	rl, err := NewRateLimiter(redis, config)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}

	if rl.allowScript == "" {
		t.Error("allowScript should not be empty")
	}

	if rl.statusScript == "" {
		t.Error("statusScript should not be empty")
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	redis := &redisclient.RedisClient{}

	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "zero requests per minute",
			config: Config{
				RequestsPerMinute: 0,
				BurstSize:         10,
			},
			wantError: true,
		},
		{
			name: "negative requests per minute",
			config: Config{
				RequestsPerMinute: -10,
				BurstSize:         10,
			},
			wantError: true,
		},
		{
			name: "zero burst size",
			config: Config{
				RequestsPerMinute: 60,
				BurstSize:         0,
			},
			wantError: true,
		},
		{
			name: "valid minimal config",
			config: Config{
				RequestsPerMinute: 1,
				BurstSize:         1,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRateLimiter(redis, tt.config)
			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestEndpointLimitOverride tests endpoint-specific limit overrides
func TestEndpointLimitOverride(t *testing.T) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()

	// Configure different limits for different endpoints
	config.EndpointLimits["/api/v1/heavy"] = EndpointLimit{
		RequestsPerMinute: 10,
		BurstSize:         2,
		Enabled:           true,
	}

	config.EndpointLimits["/api/v1/light"] = EndpointLimit{
		RequestsPerMinute: 1000,
		BurstSize:         100,
		Enabled:           true,
	}

	config.EndpointLimits["/api/v1/disabled"] = EndpointLimit{
		RequestsPerMinute: 0,
		BurstSize:         0,
		Enabled:           false,
	}

	rl, err := NewRateLimiter(redis, config)
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}

	// Test heavy endpoint
	heavyLimit := rl.getEndpointLimit("/api/v1/heavy")
	if heavyLimit.RequestsPerMinute != 10 || heavyLimit.BurstSize != 2 {
		t.Error("heavy endpoint limit not applied correctly")
	}

	// Test light endpoint
	lightLimit := rl.getEndpointLimit("/api/v1/light")
	if lightLimit.RequestsPerMinute != 1000 || lightLimit.BurstSize != 100 {
		t.Error("light endpoint limit not applied correctly")
	}

	// Test disabled endpoint
	disabledLimit := rl.getEndpointLimit("/api/v1/disabled")
	if disabledLimit.Enabled {
		t.Error("disabled endpoint should not be enabled")
	}
}

// BenchmarkBuildKey benchmarks key construction
func BenchmarkBuildKey(b *testing.B) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()
	rl, _ := NewRateLimiter(redis, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.buildKey(LimitTypeUser, "user123", "/api/v1/sessions")
	}
}

// BenchmarkGetEndpointLimit benchmarks endpoint limit lookup
func BenchmarkGetEndpointLimit(b *testing.B) {
	redis := &redisclient.RedisClient{}
	config := DefaultConfig()
	config.EndpointLimits["/api/v1/test"] = EndpointLimit{
		RequestsPerMinute: 100,
		BurstSize:         20,
		Enabled:           true,
	}
	rl, _ := NewRateLimiter(redis, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.getEndpointLimit("/api/v1/test")
	}
}

// Example usage
func ExampleNewRateLimiter() {
	// Create Redis client
	redisConfig := redisclient.DefaultConfig()
	redisConfig.URL = "redis://localhost:6379"
	redisClient, _ := redisclient.NewRedisClient(redisConfig)

	// Create rate limiter
	config := DefaultConfig()
	config.RequestsPerMinute = 60
	config.BurstSize = 10

	// Add custom endpoint limits
	config.EndpointLimits["/api/v1/upload"] = EndpointLimit{
		RequestsPerMinute: 10,
		BurstSize:         2,
		Enabled:           true,
	}

	limiter, _ := NewRateLimiter(redisClient, config)

	// Check rate limit
	ctx := context.Background()
	allowed, remaining, resetAt, _ := limiter.AllowRequest(
		ctx,
		"user123",
		"org456",
		"/api/v1/sessions",
	)

	if allowed {
		slog.Info("Request allowed",
			"remaining", remaining,
			"reset_at", resetAt,
		)
	} else {
		slog.Warn("Rate limit exceeded",
			"reset_at", resetAt,
		)
	}
}

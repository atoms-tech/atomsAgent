package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	redisclient "github.com/coder/agentapi/lib/redis"
)

// Example demonstrates how to integrate the rate limiter into an HTTP API

// ExampleBasicUsage shows basic rate limiter usage
func ExampleBasicUsage() {
	// Create Redis client
	redisConfig := redisclient.DefaultConfig()
	redisConfig.URL = getEnvOrDefault("REDIS_URL", "redis://localhost:6379")

	redisClient, err := redisclient.NewRedisClient(redisConfig)
	if err != nil {
		slog.Error("Failed to create Redis client", "error", err)
		return
	}
	defer redisClient.Close()

	// Create rate limiter with custom configuration
	limiterConfig := DefaultConfig()
	limiterConfig.RequestsPerMinute = 100 // 100 requests per minute
	limiterConfig.BurstSize = 20          // Allow burst of 20 requests
	limiterConfig.AdminBypass = true      // Allow admins to bypass

	limiter, err := NewRateLimiter(redisClient, limiterConfig)
	if err != nil {
		slog.Error("Failed to create rate limiter", "error", err)
		return
	}

	// Check if request is allowed
	ctx := context.Background()
	allowed, remaining, resetAt, err := limiter.AllowRequest(
		ctx,
		"user123",
		"org456",
		"/api/v1/sessions",
	)

	if err != nil {
		slog.Error("Rate limit check failed", "error", err)
		return
	}

	if allowed {
		slog.Info("Request allowed",
			"remaining", remaining,
			"reset_at", resetAt,
		)
		// Process request
	} else {
		slog.Warn("Rate limit exceeded",
			"reset_at", resetAt,
			"retry_after", time.Until(resetAt),
		)
		// Return 429 Too Many Requests
	}
}

// ExampleHTTPIntegration shows HTTP middleware integration
func ExampleHTTPIntegration() {
	// Setup Redis
	redisConfig := redisclient.DefaultConfig()
	redisConfig.URL = getEnvOrDefault("REDIS_URL", "redis://localhost:6379")
	redisClient, _ := redisclient.NewRedisClient(redisConfig)
	defer redisClient.Close()

	// Setup rate limiter
	limiterConfig := DefaultConfig()
	limiterConfig.RequestsPerMinute = 60
	limiterConfig.BurstSize = 10

	// Configure per-endpoint limits
	limiterConfig.EndpointLimits = map[string]EndpointLimit{
		"/api/v1/upload": {
			RequestsPerMinute: 10,
			BurstSize:         2,
			Enabled:           true,
		},
		"/api/v1/search": {
			RequestsPerMinute: 200,
			BurstSize:         50,
			Enabled:           true,
		},
		"/api/v1/public": {
			Enabled: false, // No rate limiting for public endpoints
		},
	}

	limiter, _ := NewRateLimiter(redisClient, limiterConfig)

	// Setup middleware
	middlewareConfig := DefaultMiddlewareConfig(limiter)
	middlewareConfig.SkipPaths = []string{"/health", "/metrics", "/ping"}
	middlewareConfig.DetailedLogging = true

	// Create HTTP server
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/api/v1/sessions", handleSessions)
	mux.HandleFunc("/api/v1/upload", handleUpload)
	mux.HandleFunc("/api/v1/search", handleSearch)
	mux.HandleFunc("/health", handleHealth)

	// Wrap with rate limiting middleware
	handler := Middleware(middlewareConfig)(mux)

	// Start server
	slog.Info("Starting server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		slog.Error("Server failed", "error", err)
	}
}

// ExampleWithAuthentication shows integration with authentication
func ExampleWithAuthentication() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	limiter, _ := NewRateLimiter(redisClient, DefaultConfig())

	// Custom identifier extractor that works with your auth system
	customExtractor := func(r *http.Request) (userID, orgID string, isAdmin bool) {
		// Example: Extract from JWT claims in context
		// This assumes your auth middleware has already validated the token
		// and stored claims in the context

		if claims := r.Context().Value("claims"); claims != nil {
			if c, ok := claims.(map[string]interface{}); ok {
				userID, _ = c["sub"].(string)
				orgID, _ = c["org_id"].(string)
				role, _ := c["role"].(string)
				isAdmin = role == "admin"
			}
		}

		return userID, orgID, isAdmin
	}

	middlewareConfig := DefaultMiddlewareConfig(limiter)
	middlewareConfig.IdentifierExtractor = customExtractor

	// Create handler chain: auth -> rate limit -> handler
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/data", handleData)

	// Apply middlewares in order
	handler := authMiddleware(Middleware(middlewareConfig)(mux))

	http.ListenAndServe(":8080", handler)
}

// ExampleIPBasedRateLimiting shows IP-based rate limiting for anonymous requests
func ExampleIPBasedRateLimiting() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	limiterConfig := DefaultConfig()
	limiterConfig.RequestsPerMinute = 30 // Stricter for anonymous
	limiterConfig.BurstSize = 5

	limiter, _ := NewRateLimiter(redisClient, limiterConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ipAddress := getClientIP(r)

		// Check rate limit
		allowed, remaining, resetAt, err := limiter.AllowRequestByIP(
			r.Context(),
			ipAddress,
			r.URL.Path,
		)

		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", "30")
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt).Seconds())))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Process request
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Request successful",
			"remaining": remaining,
		})
	})

	http.ListenAndServe(":8080", handler)
}

// ExampleCustomErrorHandler shows custom error handling
func ExampleCustomErrorHandler() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	limiter, _ := NewRateLimiter(redisClient, DefaultConfig())

	// Custom error handler with detailed response
	customErrorHandler := func(w http.ResponseWriter, r *http.Request, err *RateLimitError) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(err.RetryAfter.Seconds())))
		w.WriteHeader(http.StatusTooManyRequests)

		response := map[string]interface{}{
			"error": map[string]interface{}{
				"code":        "RATE_LIMIT_EXCEEDED",
				"message":     "Too many requests. Please slow down.",
				"remaining":   err.Remaining,
				"reset_at":    err.ResetAt.Format(time.RFC3339),
				"retry_after": int(err.RetryAfter.Seconds()),
			},
			"documentation_url": "https://docs.example.com/rate-limits",
		}

		json.NewEncoder(w).Encode(response)
	}

	middlewareConfig := DefaultMiddlewareConfig(limiter)
	middlewareConfig.ErrorHandler = customErrorHandler

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/data", handleData)

	handler := Middleware(middlewareConfig)(mux)
	http.ListenAndServe(":8080", handler)
}

// ExampleDynamicLimits shows how to adjust limits dynamically
func ExampleDynamicLimits() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	// Start with default limits
	config := DefaultConfig()

	// Add endpoint-specific limits
	config.EndpointLimits = map[string]EndpointLimit{
		// Heavy operations - very strict
		"/api/v1/reports/generate": {
			RequestsPerMinute: 5,
			BurstSize:         1,
			Enabled:           true,
		},
		// Data uploads - strict
		"/api/v1/upload": {
			RequestsPerMinute: 10,
			BurstSize:         3,
			Enabled:           true,
		},
		// Read operations - moderate
		"/api/v1/data": {
			RequestsPerMinute: 60,
			BurstSize:         10,
			Enabled:           true,
		},
		// Search operations - relaxed
		"/api/v1/search": {
			RequestsPerMinute: 200,
			BurstSize:         50,
			Enabled:           true,
		},
		// Public endpoints - disabled
		"/api/v1/public": {
			Enabled: false,
		},
	}

	limiter, _ := NewRateLimiter(redisClient, config)

	middlewareConfig := DefaultMiddlewareConfig(limiter)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/reports/generate", handleReportGeneration)
	mux.HandleFunc("/api/v1/upload", handleUpload)
	mux.HandleFunc("/api/v1/data", handleData)
	mux.HandleFunc("/api/v1/search", handleSearch)
	mux.HandleFunc("/api/v1/public", handlePublic)

	handler := Middleware(middlewareConfig)(mux)
	http.ListenAndServe(":8080", handler)
}

// ExampleResetLimit shows how to reset rate limits (admin operation)
func ExampleResetLimit() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	limiter, _ := NewRateLimiter(redisClient, DefaultConfig())

	// Admin endpoint to reset rate limits
	http.HandleFunc("/admin/reset-limit", func(w http.ResponseWriter, r *http.Request) {
		// Verify admin permissions (not shown)

		userID := r.URL.Query().Get("user_id")
		endpoint := r.URL.Query().Get("endpoint")

		if userID == "" || endpoint == "" {
			http.Error(w, "Missing user_id or endpoint", http.StatusBadRequest)
			return
		}

		// Reset the limit
		err := limiter.ResetLimit(r.Context(), LimitTypeUser, userID, endpoint)
		if err != nil {
			slog.Error("Failed to reset limit", "error", err)
			http.Error(w, "Failed to reset limit", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Rate limit reset successfully",
		})
	})
}

// ExampleGetLimitStatus shows how to check limit status without consuming tokens
func ExampleGetLimitStatus() {
	redisClient, _ := redisclient.NewRedisClient(redisclient.DefaultConfig())
	defer redisClient.Close()

	limiter, _ := NewRateLimiter(redisClient, DefaultConfig())

	// Endpoint to check rate limit status
	http.HandleFunc("/api/v1/rate-limit-status", func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r) // Extract from auth context
		orgID := getOrgID(r)
		endpoint := "/api/v1/data"

		remaining, limit, resetAt, err := limiter.GetLimitStatus(
			r.Context(),
			userID,
			orgID,
			endpoint,
		)

		if err != nil {
			http.Error(w, "Failed to get limit status", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"limit":     limit,
			"remaining": remaining,
			"reset_at":  resetAt.Format(time.RFC3339),
			"reset_in":  int(time.Until(resetAt).Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// Handler examples
func handleSessions(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Sessions endpoint"})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Upload endpoint"})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Search endpoint"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleData(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Data endpoint"})
}

func handlePublic(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Public endpoint"})
}

func handleReportGeneration(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "Report generation started"})
}

// Mock auth middleware
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock authentication - in production, validate JWT, etc.
		ctx := r.Context()
		ctx = WithUserID(ctx, "user123")
		ctx = WithOrgID(ctx, "org456")
		ctx = WithAdminStatus(ctx, false)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function to get user ID from request (mock)
func getUserID(r *http.Request) string {
	return GetUserID(r.Context())
}

// Helper function to get org ID from request (mock)
func getOrgID(r *http.Request) string {
	return GetOrgID(r.Context())
}

// Helper to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

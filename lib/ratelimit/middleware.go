package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"log/slog"
)

// ContextKey type for context keys
type contextKey string

const (
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyOrgID is the context key for organization ID
	ContextKeyOrgID contextKey = "org_id"
	// ContextKeyIsAdmin is the context key for admin status
	ContextKeyIsAdmin contextKey = "is_admin"
)

// MiddlewareConfig holds configuration for the rate limit middleware
type MiddlewareConfig struct {
	// RateLimiter instance
	Limiter *RateLimiter

	// Skip rate limiting for these paths
	SkipPaths []string

	// Custom error handler
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err *RateLimitError)

	// Custom identifier extractor (if not using standard auth)
	IdentifierExtractor func(r *http.Request) (userID, orgID string, isAdmin bool)

	// Enable detailed logging
	DetailedLogging bool

	// Logger
	Logger *slog.Logger
}

// DefaultMiddlewareConfig returns middleware config with sensible defaults
func DefaultMiddlewareConfig(limiter *RateLimiter) MiddlewareConfig {
	return MiddlewareConfig{
		Limiter:             limiter,
		SkipPaths:           []string{"/health", "/metrics", "/ping"},
		ErrorHandler:        defaultErrorHandler,
		IdentifierExtractor: defaultIdentifierExtractor,
		DetailedLogging:     false,
		Logger:              slog.Default(),
	}
}

// Middleware returns an HTTP middleware that enforces rate limits
func Middleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	if config.IdentifierExtractor == nil {
		config.IdentifierExtractor = defaultIdentifierExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip rate limiting
			if shouldSkipPath(r.URL.Path, config.SkipPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract identifiers
			userID, orgID, isAdmin := config.IdentifierExtractor(r)

			// Admin bypass
			if isAdmin && config.Limiter.config.AdminBypass {
				if config.DetailedLogging {
					config.Logger.Debug("Admin bypass enabled",
						"user_id", userID,
						"path", r.URL.Path,
					)
				}
				next.ServeHTTP(w, r)
				return
			}

			endpoint := normalizeEndpoint(r.URL.Path)
			var allowed bool
			var remaining int
			var resetAt time.Time
			var err error

			// Check rate limit
			if userID != "" || orgID != "" {
				allowed, remaining, resetAt, err = config.Limiter.AllowRequest(
					r.Context(),
					userID,
					orgID,
					endpoint,
				)
			} else {
				// Fall back to IP-based rate limiting for anonymous requests
				ipAddress := getClientIP(r)
				allowed, remaining, resetAt, err = config.Limiter.AllowRequestByIP(
					r.Context(),
					ipAddress,
					endpoint,
				)
			}

			if err != nil {
				config.Logger.Error("Rate limit check failed",
					"error", err,
					"user_id", userID,
					"org_id", orgID,
					"endpoint", endpoint,
				)
				// On error, allow request but log the issue
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limiter.config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

			if !allowed {
				rateLimitErr := NewRateLimitError(remaining, resetAt)

				// Set Retry-After header
				retryAfterSeconds := int(rateLimitErr.RetryAfter.Seconds())
				if retryAfterSeconds < 1 {
					retryAfterSeconds = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))

				if config.DetailedLogging {
					config.Logger.Warn("Rate limit exceeded",
						"user_id", userID,
						"org_id", orgID,
						"endpoint", endpoint,
						"remaining", remaining,
						"reset_at", resetAt,
						"retry_after", rateLimitErr.RetryAfter,
					)
				}

				config.ErrorHandler(w, r, rateLimitErr)
				return
			}

			if config.DetailedLogging {
				config.Logger.Debug("Rate limit check passed",
					"user_id", userID,
					"org_id", orgID,
					"endpoint", endpoint,
					"remaining", remaining,
				)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// defaultErrorHandler is the default error handler for rate limit errors
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err *RateLimitError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":       "Too Many Requests",
		"message":     err.Message,
		"remaining":   err.Remaining,
		"reset_at":    err.ResetAt.Format(time.RFC3339),
		"retry_after": int(err.RetryAfter.Seconds()),
	}

	json.NewEncoder(w).Encode(response)
}

// defaultIdentifierExtractor attempts to extract user/org ID from standard auth context
func defaultIdentifierExtractor(r *http.Request) (userID, orgID string, isAdmin bool) {
	// Try to get from context (set by auth middleware)
	if uid := r.Context().Value(ContextKeyUserID); uid != nil {
		if u, ok := uid.(string); ok {
			userID = u
		}
	}

	if oid := r.Context().Value(ContextKeyOrgID); oid != nil {
		if o, ok := oid.(string); ok {
			orgID = o
		}
	}

	if admin := r.Context().Value(ContextKeyIsAdmin); admin != nil {
		if a, ok := admin.(bool); ok {
			isAdmin = a
		}
	}

	return userID, orgID, isAdmin
}

// shouldSkipPath checks if a path should skip rate limiting
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// normalizeEndpoint normalizes endpoint paths for rate limiting
// This groups similar endpoints together (e.g., /api/v1/user/123 -> /api/v1/user/:id)
func normalizeEndpoint(path string) string {
	// Basic normalization - remove trailing slashes
	path = strings.TrimSuffix(path, "/")

	// You can add more sophisticated normalization here
	// For example, replace UUIDs or numeric IDs with placeholders
	// This is application-specific and can be customized

	return path
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		return cfip
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimitInfo holds information about current rate limits
type RateLimitInfo struct {
	Allowed    bool          `json:"allowed"`
	Remaining  int           `json:"remaining"`
	Limit      int           `json:"limit"`
	ResetAt    time.Time     `json:"reset_at"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// GetRateLimitInfo extracts rate limit information from response headers
func GetRateLimitInfo(headers http.Header) (*RateLimitInfo, error) {
	info := &RateLimitInfo{
		Allowed: true, // Assume allowed if headers are present
	}

	// Parse limit
	if limitStr := headers.Get("X-RateLimit-Limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
			info.Limit = limit
		}
	}

	// Parse remaining
	if remainingStr := headers.Get("X-RateLimit-Remaining"); remainingStr != "" {
		var remaining int
		if _, err := fmt.Sscanf(remainingStr, "%d", &remaining); err == nil {
			info.Remaining = remaining
		}
	}

	// Parse reset time
	if resetStr := headers.Get("X-RateLimit-Reset"); resetStr != "" {
		var resetUnix int64
		if _, err := fmt.Sscanf(resetStr, "%d", &resetUnix); err == nil {
			info.ResetAt = time.Unix(resetUnix, 0)
		}
	}

	// Parse retry-after
	if retryStr := headers.Get("Retry-After"); retryStr != "" {
		var retrySeconds int
		if _, err := fmt.Sscanf(retryStr, "%d", &retrySeconds); err == nil {
			info.RetryAfter = time.Duration(retrySeconds) * time.Second
			info.Allowed = false
		}
	}

	return info, nil
}

// WithUserID adds user ID to request context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// WithOrgID adds organization ID to request context
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, ContextKeyOrgID, orgID)
}

// WithAdminStatus adds admin status to request context
func WithAdminStatus(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, ContextKeyIsAdmin, isAdmin)
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if uid := ctx.Value(ContextKeyUserID); uid != nil {
		if u, ok := uid.(string); ok {
			return u
		}
	}
	return ""
}

// GetOrgID extracts organization ID from context
func GetOrgID(ctx context.Context) string {
	if oid := ctx.Value(ContextKeyOrgID); oid != nil {
		if o, ok := oid.(string); ok {
			return o
		}
	}
	return ""
}

// GetAdminStatus extracts admin status from context
func GetAdminStatus(ctx context.Context) bool {
	if admin := ctx.Value(ContextKeyIsAdmin); admin != nil {
		if a, ok := admin.(bool); ok {
			return a
		}
	}
	return false
}

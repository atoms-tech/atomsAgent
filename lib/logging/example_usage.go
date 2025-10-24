//go:build example
// +build example

// This file contains usage examples (not compiled into the package)
// Build tag ensures it's only used for documentation

package logging

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Example 1: Basic usage
func ExampleBasicUsage() {
	logger := NewLogger("myapp", INFO)

	logger.Info("Application started")
	logger.Warn("Low disk space")
	logger.Error("Failed to connect")
}

// Example 2: Structured logging with fields
func ExampleStructuredLogging() {
	logger := NewLogger("api", INFO)

	logger.WithFields(map[string]interface{}{
		"user_id":  12345,
		"username": "alice",
		"action":   "login",
		"ip":       "192.168.1.1",
	}).Info("User authentication successful")
}

// Example 3: Error handling with stack traces
func ExampleErrorHandling() {
	logger := NewLogger("database", ERROR)

	err := errors.New("connection timeout")
	logger.WithError(err).Error("Database connection failed")

	// Or with custom message
	logger.ErrorWithError("Failed to execute query", err)
}

// Example 4: HTTP middleware for request logging
func LoggingMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID
			requestID := uuid.New().String()
			ctx := WithRequestID(r.Context(), requestID)

			// Log request start
			logger.WithContext(ctx).WithFields(map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}).Info("Request started")

			// Create response writer wrapper to capture status
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Log request completion
			duration := time.Since(start)
			logger.WithContext(ctx).WithFields(map[string]interface{}{
				"status":      wrapped.status,
				"duration_ms": duration.Milliseconds(),
			}).Info("Request completed")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Example 5: Service layer logging
type UserService struct {
	logger *Logger
}

func NewUserService() *UserService {
	return &UserService{
		logger: GetLogger("user-service"),
	}
}

func (s *UserService) CreateUser(ctx context.Context, username, email string) error {
	// Add context to logger
	logger := s.logger.WithContext(ctx)

	logger.WithFields(map[string]interface{}{
		"username": username,
		"email":    email,
	}).Info("Creating user")

	// Simulate validation
	if username == "" {
		err := errors.New("username is required")
		logger.WithError(err).Error("User validation failed")
		return err
	}

	// Simulate database operation
	if err := s.saveToDatabase(username, email); err != nil {
		logger.WithError(err).Error("Failed to save user to database")
		return err
	}

	logger.WithField("username", username).Info("User created successfully")
	return nil
}

func (s *UserService) saveToDatabase(username, email string) error {
	// Simulate database operation
	return nil
}

// Example 6: Application initialization
func InitializeApplication() {
	// Configure from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	// Set global level
	SetGlobalLevel(ParseLogLevel(logLevel))

	// Create component loggers
	apiLogger := GetLogger("api")
	dbLogger := GetLogger("database")
	authLogger := GetLogger("auth")

	// Override specific component levels if needed
	if os.Getenv("DEBUG_DB") == "true" {
		dbLogger.SetLevel(DEBUG)
	}

	// Log startup
	apiLogger.Info("API server initializing")
	dbLogger.Info("Database connection pool created")
	authLogger.Info("Authentication providers loaded")
}

// Example 7: Background job logging
type JobProcessor struct {
	logger *Logger
}

func NewJobProcessor() *JobProcessor {
	return &JobProcessor{
		logger: GetLogger("job-processor"),
	}
}

func (p *JobProcessor) ProcessJob(ctx context.Context, jobID string) {
	// Create job-specific logger
	jobLogger := p.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
		"worker": "worker-1",
	})

	jobLogger.Info("Job started")

	// Check if debug is enabled before expensive operations
	if jobLogger.IsDebugEnabled() {
		jobDetails := p.getJobDetails(jobID)
		jobLogger.Debugf("Job details: %+v", jobDetails)
	}

	// Simulate processing
	err := p.processInternal(ctx, jobID)
	if err != nil {
		jobLogger.WithError(err).Error("Job failed")
		return
	}

	jobLogger.Info("Job completed")
}

func (p *JobProcessor) getJobDetails(jobID string) map[string]interface{} {
	return map[string]interface{}{
		"id":      jobID,
		"created": time.Now(),
		"status":  "pending",
	}
}

func (p *JobProcessor) processInternal(ctx context.Context, jobID string) error {
	return nil
}

// Example 8: Testing with custom output
func ExampleTesting() {
	// In tests, capture log output
	logger := NewLogger("test", DEBUG)

	// Use a buffer for testing
	// buf := &bytes.Buffer{}
	// logger.SetOutput(buf)

	logger.Info("Test message")

	// Assert on buffer contents
}

// Example 9: Graceful degradation
func ExampleGracefulDegradation() {
	logger := NewLogger("app", INFO)

	// If JSON marshaling fails, logger will output fallback format
	// This handles edge cases with unmarshalable values

	// Example with channel (not JSON-serializable)
	ch := make(chan int)
	logger.WithField("channel", fmt.Sprintf("%v", ch)).Info("Using fallback")
}

// Example 10: Multi-tenant logging
type TenantService struct {
	logger *Logger
}

func NewTenantService() *TenantService {
	return &TenantService{
		logger: GetLogger("tenant-service"),
	}
}

func (s *TenantService) HandleRequest(ctx context.Context, tenantID string) {
	// Create tenant-specific logger
	tenantLogger := s.logger.WithFields(map[string]interface{}{
		"tenant_id": tenantID,
		"region":    "us-east-1",
	})

	// All logs will include tenant context
	tenantLogger.WithContext(ctx).Info("Processing tenant request")

	// Business logic
	if err := s.processForTenant(tenantID); err != nil {
		tenantLogger.WithError(err).Error("Tenant request failed")
		return
	}

	tenantLogger.Info("Tenant request completed")
}

func (s *TenantService) processForTenant(tenantID string) error {
	return nil
}

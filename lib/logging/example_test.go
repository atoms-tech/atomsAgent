package logging_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/coder/agentapi/lib/logging"
)

func ExampleLogger_basic() {
	// Create a logger
	logger := logging.NewLogger("myapp", logging.INFO)

	// Log messages at different levels
	logger.Debug("This won't be logged (level is INFO)")
	logger.Info("Application started")
	logger.Warn("Low disk space")
	logger.Error("Failed to connect to database")

	// Output will be JSON formatted
}

func ExampleLogger_WithField() {
	logger := logging.NewLogger("api", logging.INFO)

	// Add a single field
	logger.WithField("user_id", 12345).Info("User logged in")

	// Chain multiple fields
	logger.
		WithField("user_id", 12345).
		WithField("ip", "192.168.1.1").
		WithField("method", "POST").
		Info("Request processed")
}

func ExampleLogger_WithFields() {
	logger := logging.NewLogger("api", logging.INFO)

	// Add multiple fields at once
	logger.WithFields(map[string]interface{}{
		"user_id":  12345,
		"username": "alice",
		"role":     "admin",
		"active":   true,
	}).Info("User authenticated")
}

func ExampleLogger_WithError() {
	logger := logging.NewLogger("database", logging.INFO)

	err := errors.New("connection timeout")
	logger.WithError(err).Error("Database connection failed")

	// Or use ErrorWithError for both message and error
	logger.ErrorWithError("Failed to execute query", err)
}

func ExampleLogger_context() {
	logger := logging.NewLogger("api", logging.INFO)

	// Add request ID to context
	ctx := logging.WithRequestID(context.Background(), "req-abc-123")

	// Logger will automatically include request ID from context
	logger.WithContext(ctx).Info("Processing request")

	// You can also add the logger to context
	ctx = logging.WithLogger(ctx, logger)

	// Later, retrieve the logger from context
	ctxLogger := logging.FromContext(ctx)
	ctxLogger.Info("Using logger from context")
}

func ExampleGetLogger() {
	// Get or create a logger by name
	logger := logging.GetLogger("myservice")
	logger.Info("Service initialized")

	// Getting the same logger by name returns the same instance
	sameLogger := logging.GetLogger("myservice")
	sameLogger.Info("Same logger instance")
}

func ExampleSetGlobalLevel() {
	// Set global log level for all future loggers
	logging.SetGlobalLevel(logging.DEBUG)

	logger := logging.GetLogger("app")
	logger.Debug("This will now be logged")

	// Change level for specific logger
	logger.SetLevel(logging.ERROR)
	logger.Debug("This won't be logged")
	logger.Error("This will be logged")
}

func ExampleLogger_formatted() {
	logger := logging.NewLogger("api", logging.INFO)

	userID := 12345
	username := "alice"

	// Use formatted logging methods
	logger.Infof("User %s (ID: %d) logged in", username, userID)
	logger.Debugf("Processing %d items", 42)
	logger.Warnf("Memory usage: %.2f%%", 85.5)
	logger.Errorf("Failed to process user %s", username)
}

func ExampleLogger_levelChecks() {
	logger := logging.NewLogger("api", logging.INFO)

	// Check if a level is enabled before expensive operations
	if logger.IsDebugEnabled() {
		// This won't execute because logger level is INFO
		expensiveDebugData := computeExpensiveData()
		logger.Debugf("Debug data: %v", expensiveDebugData)
	}

	if logger.IsInfoEnabled() {
		// This will execute
		logger.Info("Level check passed")
	}
}

func computeExpensiveData() string {
	return "expensive data"
}

func ExampleLogger_httpMiddleware() {
	logger := logging.GetLogger("http")

	// Example HTTP middleware pattern
	handleRequest := func(requestID string) {
		ctx := logging.WithRequestID(context.Background(), requestID)

		// Log request start
		logger.WithContext(ctx).
			WithFields(map[string]interface{}{
				"method": "GET",
				"path":   "/api/users",
			}).
			Info("Request started")

		// Simulate processing
		err := processRequest()

		if err != nil {
			logger.WithContext(ctx).
				WithError(err).
				Error("Request failed")
		} else {
			logger.WithContext(ctx).
				WithField("status", 200).
				Info("Request completed")
		}
	}

	handleRequest("req-123")
}

func processRequest() error {
	return nil
}

func ExampleLogger_stackTrace() {
	logger := logging.NewLogger("app", logging.ERROR)

	// Error logs automatically capture stack traces
	logger.Error("Critical error occurred")

	// Stack trace will include file, line, and function information
	err := errors.New("database connection lost")
	logger.ErrorWithError("Failed to process transaction", err)
}

func ExampleLogger_multipleLoggers() {
	// Different loggers for different components
	apiLogger := logging.GetLogger("api")
	dbLogger := logging.GetLogger("database")
	cacheLogger := logging.GetLogger("cache")

	// Set different levels for different components
	apiLogger.SetLevel(logging.INFO)
	dbLogger.SetLevel(logging.DEBUG)
	cacheLogger.SetLevel(logging.WARN)

	apiLogger.Info("API server started")
	dbLogger.Debug("Connection pool initialized")
	cacheLogger.Warn("Cache size approaching limit")
}

func ExampleLogger_customOutput() {
	// Create logger with custom output
	logFile, err := os.CreateTemp("", "app-*.log")
	if err != nil {
		panic(err)
	}
	defer os.Remove(logFile.Name())

	logger := logging.NewLogger("app", logging.INFO)
	logger.SetOutput(logFile)

	logger.Info("This will be written to the log file")

	// You can also use SetOutput globally
	logging.SetOutput(os.Stdout) // All new loggers will write to stdout
}

func ExampleParseLogLevel() {
	// Parse log level from environment variable or config
	levelStr := "DEBUG" // From env var or config file

	level := logging.ParseLogLevel(levelStr)
	logger := logging.NewLogger("app", level)

	logger.Debug("Debug logging enabled")
}

func ExampleLogger_businessLogic() {
	logger := logging.GetLogger("orders")

	// Example: Processing an order
	processOrder := func(orderID int, userID int) error {
		logger.WithFields(map[string]interface{}{
			"order_id": orderID,
			"user_id":  userID,
		}).Info("Processing order")

		// Simulate validation
		if userID == 0 {
			err := errors.New("invalid user ID")
			logger.WithFields(map[string]interface{}{
				"order_id": orderID,
				"user_id":  userID,
			}).WithError(err).Error("Order validation failed")
			return err
		}

		// Simulate processing
		logger.WithFields(map[string]interface{}{
			"order_id": orderID,
			"amount":   99.99,
			"status":   "completed",
		}).Info("Order processed successfully")

		return nil
	}

	processOrder(12345, 67890)
}

func ExampleLogger_performance() {
	logger := logging.NewLogger("perf", logging.INFO)

	// Reuse logger with fields for better performance
	userLogger := logger.WithFields(map[string]interface{}{
		"user_id": 12345,
		"session": "abc-123",
	})

	// These will all include user_id and session without re-adding them
	userLogger.Info("User action: login")
	userLogger.Info("User action: view profile")
	userLogger.Info("User action: update settings")
}

// Example of complete application setup
func Example_applicationSetup() {
	// 1. Configure global settings at startup
	logging.SetGlobalLevel(logging.INFO)

	// 2. Create component-specific loggers
	apiLogger := logging.GetLogger("api")
	dbLogger := logging.GetLogger("database")
	authLogger := logging.GetLogger("auth")

	// 3. Optionally override levels for specific components
	if os.Getenv("DEBUG") == "true" {
		dbLogger.SetLevel(logging.DEBUG)
	}

	// 4. Use throughout application
	apiLogger.Info("Starting API server")
	dbLogger.Info("Connecting to database")
	authLogger.Info("Loading authentication providers")

	// 5. Use context for request tracing
	ctx := logging.WithRequestID(context.Background(), "req-001")
	apiLogger.WithContext(ctx).Info("Handling request")

	fmt.Println("Application configured")
	// Output: Application configured
}

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/agentapi/lib/logging"
)

func main() {
	fmt.Println("=== Structured JSON Logging Demo ===\n")

	// Example 1: Basic logging at different levels
	fmt.Println("1. Basic logging:")
	logger := logging.NewLogger("demo", logging.DEBUG)
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	fmt.Println("\n2. Logging with structured fields:")
	logger.WithFields(map[string]interface{}{
		"user_id":  12345,
		"username": "alice",
		"action":   "login",
	}).Info("User authenticated")

	fmt.Println("\n3. Logging with error:")
	err := errors.New("database connection timeout")
	logger.WithError(err).Error("Failed to connect to database")

	fmt.Println("\n4. Context with request ID:")
	ctx := logging.WithRequestID(context.Background(), "req-abc-123")
	logger.WithContext(ctx).WithField("method", "GET").Info("Processing request")

	fmt.Println("\n5. Chained fields:")
	logger.
		WithField("component", "api").
		WithField("endpoint", "/users").
		WithField("status", 200).
		Info("API request successful")

	fmt.Println("\n6. Formatted logging:")
	userCount := 42
	logger.Infof("Successfully processed %d users", userCount)

	fmt.Println("\n7. Level filtering:")
	infoLogger := logging.NewLogger("filtered", logging.INFO)
	infoLogger.Debug("This debug message won't appear")
	infoLogger.Info("This info message will appear")

	fmt.Println("\n8. Multiple loggers:")
	apiLogger := logging.GetLogger("api")
	dbLogger := logging.GetLogger("database")
	cacheLogger := logging.GetLogger("cache")

	apiLogger.Info("API server started on :8080")
	dbLogger.Info("Connected to PostgreSQL")
	cacheLogger.Info("Redis cache initialized")

	fmt.Println("\n9. Business logic example:")
	processOrder(12345, 67890)

	fmt.Println("\n10. Error with stack trace:")
	simulateError()

	fmt.Println("\n=== Demo Complete ===")
}

func processOrder(orderID, userID int) {
	logger := logging.GetLogger("orders")

	logger.WithFields(map[string]interface{}{
		"order_id": orderID,
		"user_id":  userID,
	}).Info("Processing order")

	// Simulate some processing
	logger.WithFields(map[string]interface{}{
		"order_id": orderID,
		"amount":   99.99,
		"status":   "completed",
	}).Info("Order processed successfully")
}

func simulateError() {
	logger := logging.GetLogger("error-demo")

	// This will include a full stack trace
	err := errors.New("critical system error")
	logger.ErrorWithError("System failure detected", err)
}

# Structured JSON Logging

A high-performance, structured JSON logging library for Go with zero external dependencies.

## Features

- **Structured JSON Output**: All logs are output as valid JSON with consistent schema
- **Multiple Log Levels**: DEBUG, INFO, WARN, ERROR with filtering
- **Context Fields**: Add structured key-value pairs to logs
- **Request ID Correlation**: Track requests across service boundaries
- **Stack Trace Capture**: Automatic stack traces for error logs
- **Context Propagation**: Integrate with Go's context.Context
- **Thread-Safe**: Safe for concurrent use
- **No External Dependencies**: Uses only stdlib (encoding/json)
- **Performance Optimized**: Minimal allocations, level filtering before processing

## Quick Start

```go
import "github.com/coder/agentapi/lib/logging"

// Create a logger
logger := logging.NewLogger("myapp", logging.INFO)

// Log messages
logger.Info("Application started")
logger.Warn("Low disk space")
logger.Error("Failed to connect")

// Add structured fields
logger.WithField("user_id", 12345).Info("User logged in")

// Add multiple fields
logger.WithFields(map[string]interface{}{
    "user_id": 12345,
    "action": "login",
    "ip": "192.168.1.1",
}).Info("Authentication successful")
```

## Log Levels

```go
const (
    DEBUG LogLevel = iota  // Detailed debugging information
    INFO                   // General informational messages
    WARN                   // Warning conditions
    ERROR                  // Error conditions
)
```

Logs are only output if their level is >= the logger's configured level.

## Logger Creation

### New Logger
```go
logger := logging.NewLogger("component-name", logging.INFO)
```

### Get or Create Logger
```go
// Returns existing logger or creates new one
logger := logging.GetLogger("component-name")
```

### Global Settings
```go
// Set default level for all new loggers
logging.SetGlobalLevel(logging.DEBUG)

// Set default output for all new loggers
logging.SetOutput(os.Stdout)
```

## Logging Methods

### Basic Logging
```go
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")  // Includes stack trace
```

### Formatted Logging
```go
logger.Debugf("Processing %d items", count)
logger.Infof("User %s logged in", username)
logger.Warnf("Memory usage: %.2f%%", usage)
logger.Errorf("Failed to process %s", item)
```

### Error Logging with Error Objects
```go
err := errors.New("connection failed")
logger.WithError(err).Error("Database error")

// Or
logger.ErrorWithError("Failed to connect", err)
```

## Structured Fields

### Single Field
```go
logger.WithField("user_id", 12345).Info("User action")
```

### Multiple Fields
```go
logger.WithFields(map[string]interface{}{
    "user_id": 12345,
    "action": "login",
    "success": true,
}).Info("Authentication attempt")
```

### Chaining
```go
logger.
    WithField("user_id", 12345).
    WithField("session", "abc-123").
    Info("Session created")
```

## Context Integration

### Request ID Tracking
```go
// Add request ID to context
ctx := logging.WithRequestID(context.Background(), "req-abc-123")

// Logger automatically includes request ID
logger.WithContext(ctx).Info("Processing request")

// Retrieve request ID
requestID := logging.GetRequestID(ctx)
```

### Logger in Context
```go
// Store logger in context
ctx := logging.WithLogger(context.Background(), logger)

// Retrieve logger from context
logger := logging.FromContext(ctx)
logger.Info("Using context logger")
```

## Output Format

All logs are output as JSON with the following schema:

```json
{
  "timestamp": "2024-01-15T10:30:45.123456789Z",
  "level": "INFO",
  "logger": "api",
  "message": "Request processed",
  "request_id": "req-abc-123",
  "fields": {
    "user_id": 12345,
    "method": "GET",
    "path": "/api/users"
  }
}
```

Error logs include stack traces:

```json
{
  "timestamp": "2024-01-15T10:30:45.123456789Z",
  "level": "ERROR",
  "logger": "database",
  "message": "Query failed",
  "error": "connection timeout",
  "stack_trace": [
    "/app/database/query.go:45 main.executeQuery",
    "/app/api/handler.go:23 main.handleRequest",
    "/usr/local/go/src/runtime/asm_amd64.s:1650 runtime.goexit"
  ],
  "fields": {
    "query": "SELECT * FROM users"
  }
}
```

## Performance Optimization

### Level Checks
```go
// Avoid expensive operations if level is disabled
if logger.IsDebugEnabled() {
    expensiveData := computeExpensiveDebugData()
    logger.Debugf("Data: %v", expensiveData)
}
```

### Field Reuse
```go
// Create logger with common fields once
userLogger := logger.WithFields(map[string]interface{}{
    "user_id": 12345,
    "session": "abc-123",
})

// Reuse for multiple logs
userLogger.Info("Login successful")
userLogger.Info("Profile viewed")
userLogger.Info("Settings updated")
```

## HTTP Middleware Example

```go
func LoggingMiddleware(logger *logging.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requestID := uuid.New().String()
            ctx := logging.WithRequestID(r.Context(), requestID)

            logger.WithContext(ctx).WithFields(map[string]interface{}{
                "method": r.Method,
                "path": r.URL.Path,
                "remote_addr": r.RemoteAddr,
            }).Info("Request started")

            next.ServeHTTP(w, r.WithContext(ctx))

            logger.WithContext(ctx).Info("Request completed")
        })
    }
}
```

## Application Setup Example

```go
func main() {
    // Configure global settings
    logging.SetGlobalLevel(logging.INFO)

    // Create component loggers
    apiLogger := logging.GetLogger("api")
    dbLogger := logging.GetLogger("database")
    authLogger := logging.GetLogger("auth")

    // Override specific component levels
    if os.Getenv("DEBUG") == "true" {
        dbLogger.SetLevel(logging.DEBUG)
    }

    // Use throughout application
    apiLogger.Info("Starting server")
    dbLogger.Info("Connecting to database")
    authLogger.Info("Loading auth providers")
}
```

## Best Practices

1. **Use Named Loggers**: Create separate loggers for different components
   ```go
   apiLogger := logging.GetLogger("api")
   dbLogger := logging.GetLogger("database")
   ```

2. **Add Context**: Include relevant context in fields
   ```go
   logger.WithFields(map[string]interface{}{
       "user_id": userID,
       "action": action,
   }).Info("User action")
   ```

3. **Request Tracking**: Use request IDs for distributed tracing
   ```go
   ctx := logging.WithRequestID(ctx, requestID)
   logger.WithContext(ctx).Info("Processing")
   ```

4. **Consistent Fields**: Use consistent field names across services
   - `user_id` not `userId` or `user_identifier`
   - `request_id` not `requestId` or `req_id`

5. **Error Details**: Always include error context
   ```go
   logger.WithError(err).WithField("query", query).Error("Query failed")
   ```

6. **Avoid PII**: Don't log sensitive information
   ```go
   // Bad
   logger.WithField("password", password).Info("Login")

   // Good
   logger.WithField("user_id", userID).Info("Login")
   ```

7. **Performance**: Check log level before expensive operations
   ```go
   if logger.IsDebugEnabled() {
       logger.Debugf("Complex data: %v", expensiveOperation())
   }
   ```

## Testing

The library includes comprehensive tests:

```bash
go test ./lib/logging/...
go test -bench=. ./lib/logging/...
```

## Thread Safety

All logger operations are thread-safe. Multiple goroutines can safely:
- Log to the same logger
- Create new loggers with fields
- Modify logger settings

```go
// Safe for concurrent use
for i := 0; i < 10; i++ {
    go func(id int) {
        logger.WithField("goroutine", id).Info("Concurrent log")
    }(i)
}
```

## Configuration from Environment

```go
func setupLogging() {
    // Parse level from environment
    levelStr := os.Getenv("LOG_LEVEL")
    if levelStr == "" {
        levelStr = "INFO"
    }
    level := logging.ParseLogLevel(levelStr)

    logging.SetGlobalLevel(level)

    // Optional: Configure output
    if logFile := os.Getenv("LOG_FILE"); logFile != "" {
        f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err == nil {
            logging.SetOutput(f)
        }
    }
}
```

## Integration with Existing Code

The logger can be gradually adopted:

```go
// Start with one logger
logger := logging.GetLogger("app")

// Replace fmt.Println
- fmt.Println("Server starting")
+ logger.Info("Server starting")

// Replace log.Printf
- log.Printf("User %s logged in", username)
+ logger.Infof("User %s logged in", username)

// Add structure to logs
+ logger.WithField("user_id", userID).Info("User logged in")
```

# Structured Logging - Complete API Reference

## Types

### LogLevel
```go
type LogLevel int

const (
    DEBUG LogLevel = iota  // Detailed debugging information
    INFO                   // General informational messages
    WARN                   // Warning conditions
    ERROR                  // Error conditions with stack traces
)
```

**Methods:**
- `String() string` - Returns "DEBUG", "INFO", "WARN", or "ERROR"

### Logger
```go
type Logger struct {
    // Private fields - access via methods
}
```

## Functions

### Logger Creation

#### NewLogger
```go
func NewLogger(name string, level LogLevel) *Logger
```
Creates a new logger with the specified name and level.

**Example:**
```go
logger := logging.NewLogger("myapp", logging.INFO)
```

#### GetLogger
```go
func GetLogger(name string) *Logger
```
Gets or creates a logger by name (singleton pattern).

**Example:**
```go
logger := logging.GetLogger("api")
```

#### ParseLogLevel
```go
func ParseLogLevel(s string) LogLevel
```
Converts a string to a LogLevel (case-insensitive).

**Example:**
```go
level := logging.ParseLogLevel("DEBUG")  // Returns DEBUG
level := logging.ParseLogLevel("info")   // Returns INFO
```

### Global Configuration

#### SetGlobalLevel
```go
func SetGlobalLevel(level LogLevel)
```
Sets the default log level for all future loggers.

**Example:**
```go
logging.SetGlobalLevel(logging.DEBUG)
```

#### SetOutput
```go
func SetOutput(w io.Writer)
```
Sets the default output writer for all future loggers.

**Example:**
```go
logging.SetOutput(os.Stdout)
```

## Logger Methods

### Configuration

#### SetLevel
```go
func (l *Logger) SetLevel(level LogLevel)
```
Sets the minimum log level for this logger.

**Example:**
```go
logger.SetLevel(logging.WARN)
```

#### GetLevel
```go
func (l *Logger) GetLevel() LogLevel
```
Returns the current log level.

**Example:**
```go
level := logger.GetLevel()
```

#### SetOutput
```go
func (l *Logger) SetOutput(w io.Writer)
```
Sets the output writer for this logger.

**Example:**
```go
logger.SetOutput(logFile)
```

### Field Management

#### WithField
```go
func (l *Logger) WithField(key string, value interface{}) *Logger
```
Returns a new logger with an additional field. Original logger is unchanged.

**Example:**
```go
userLogger := logger.WithField("user_id", 12345)
```

#### WithFields
```go
func (l *Logger) WithFields(fields map[string]interface{}) *Logger
```
Returns a new logger with multiple additional fields.

**Example:**
```go
requestLogger := logger.WithFields(map[string]interface{}{
    "method": "GET",
    "path": "/api/users",
    "ip": "192.168.1.1",
})
```

#### WithError
```go
func (l *Logger) WithError(err error) *Logger
```
Returns a new logger with an error field. If err is nil, returns the same logger.

**Example:**
```go
logger.WithError(err).Error("Operation failed")
```

#### WithContext
```go
func (l *Logger) WithContext(ctx context.Context) *Logger
```
Returns a new logger with fields extracted from context (e.g., request ID).

**Example:**
```go
logger.WithContext(ctx).Info("Processing request")
```

### Logging Methods

#### Debug / Debugf
```go
func (l *Logger) Debug(msg string)
func (l *Logger) Debugf(format string, args ...interface{})
```
Logs a debug message if level >= DEBUG.

**Example:**
```go
logger.Debug("Connection pool initialized")
logger.Debugf("Loaded %d configuration items", count)
```

#### Info / Infof
```go
func (l *Logger) Info(msg string)
func (l *Logger) Infof(format string, args ...interface{})
```
Logs an informational message if level >= INFO.

**Example:**
```go
logger.Info("Server started")
logger.Infof("Listening on port %d", port)
```

#### Warn / Warnf
```go
func (l *Logger) Warn(msg string)
func (l *Logger) Warnf(format string, args ...interface{})
```
Logs a warning message if level >= WARN.

**Example:**
```go
logger.Warn("Disk space running low")
logger.Warnf("Cache size %d exceeds threshold", size)
```

#### Error / Errorf
```go
func (l *Logger) Error(msg string)
func (l *Logger) Errorf(format string, args ...interface{})
```
Logs an error message with stack trace if level >= ERROR.

**Example:**
```go
logger.Error("Database connection failed")
logger.Errorf("Failed to process order %s", orderID)
```

#### ErrorWithError
```go
func (l *Logger) ErrorWithError(msg string, err error)
```
Logs an error message with an error object and stack trace.

**Example:**
```go
logger.ErrorWithError("Query failed", err)
```

### Level Checks

#### IsLevelEnabled
```go
func (l *Logger) IsLevelEnabled(level LogLevel) bool
```
Returns true if the given level would be logged.

**Example:**
```go
if logger.IsLevelEnabled(logging.DEBUG) {
    // Expensive operation
}
```

#### IsDebugEnabled
```go
func (l *Logger) IsDebugEnabled() bool
```
Returns true if debug logging is enabled.

**Example:**
```go
if logger.IsDebugEnabled() {
    data := computeExpensiveDebugData()
    logger.Debugf("Debug: %v", data)
}
```

#### IsInfoEnabled
```go
func (l *Logger) IsInfoEnabled() bool
```
Returns true if info logging is enabled.

#### IsWarnEnabled
```go
func (l *Logger) IsWarnEnabled() bool
```
Returns true if warn logging is enabled.

#### IsErrorEnabled
```go
func (l *Logger) IsErrorEnabled() bool
```
Returns true if error logging is enabled.

## Context Functions

### WithRequestID
```go
func WithRequestID(ctx context.Context, requestID string) context.Context
```
Adds a request ID to the context.

**Example:**
```go
ctx := logging.WithRequestID(context.Background(), "req-123")
```

### GetRequestID
```go
func GetRequestID(ctx context.Context) string
```
Retrieves the request ID from the context.

**Example:**
```go
requestID := logging.GetRequestID(ctx)
```

### WithLogger
```go
func WithLogger(ctx context.Context, logger *Logger) context.Context
```
Adds a logger to the context.

**Example:**
```go
ctx := logging.WithLogger(ctx, logger)
```

### FromContext
```go
func FromContext(ctx context.Context) *Logger
```
Retrieves a logger from the context, or returns a default logger.

**Example:**
```go
logger := logging.FromContext(ctx)
```

## JSON Output Schema

### Standard Log Entry
```go
type logEntry struct {
    Timestamp  string                 `json:"timestamp"`   // RFC3339Nano format
    Level      string                 `json:"level"`       // DEBUG, INFO, WARN, ERROR
    Logger     string                 `json:"logger,omitempty"`     // Logger name
    Message    string                 `json:"message"`     // Log message
    Fields     map[string]interface{} `json:"fields,omitempty"`     // Additional fields
    RequestID  string                 `json:"request_id,omitempty"` // Request ID if present
    Error      string                 `json:"error,omitempty"`      // Error message if present
    StackTrace []string               `json:"stack_trace,omitempty"` // Stack trace for errors
}
```

### Example Output

**Simple Info Log:**
```json
{
  "timestamp": "2025-10-24T06:10:08.616426Z",
  "level": "INFO",
  "logger": "api",
  "message": "Server started"
}
```

**Log with Fields:**
```json
{
  "timestamp": "2025-10-24T06:10:08.616709Z",
  "level": "INFO",
  "logger": "api",
  "message": "User authenticated",
  "fields": {
    "user_id": 12345,
    "username": "alice",
    "action": "login"
  }
}
```

**Log with Request ID:**
```json
{
  "timestamp": "2025-10-24T06:10:08.616723Z",
  "level": "INFO",
  "logger": "api",
  "message": "Processing request",
  "request_id": "req-abc-123",
  "fields": {
    "method": "GET",
    "path": "/api/users"
  }
}
```

**Error Log with Stack Trace:**
```json
{
  "timestamp": "2025-10-24T06:10:08.616754Z",
  "level": "ERROR",
  "logger": "database",
  "message": "Query failed",
  "error": "connection timeout",
  "stack_trace": [
    "/app/database/query.go:45 github.com/myapp/database.executeQuery",
    "/app/api/handler.go:23 github.com/myapp/api.handleRequest",
    "/usr/local/go/src/runtime/proc.go:285 runtime.main"
  ],
  "fields": {
    "query": "SELECT * FROM users",
    "duration_ms": 5000
  }
}
```

## Usage Patterns

### 1. Basic Logging
```go
logger := logging.NewLogger("myapp", logging.INFO)
logger.Info("Application started")
```

### 2. Structured Logging
```go
logger.WithFields(map[string]interface{}{
    "user_id": 123,
    "action": "login",
}).Info("User action")
```

### 3. Error Logging
```go
if err != nil {
    logger.WithError(err).Error("Operation failed")
}
```

### 4. Context Logging
```go
ctx := logging.WithRequestID(ctx, requestID)
logger.WithContext(ctx).Info("Processing request")
```

### 5. Chained Fields
```go
logger.
    WithField("component", "api").
    WithField("endpoint", "/users").
    WithField("status", 200).
    Info("Request completed")
```

### 6. Formatted Logging
```go
logger.Infof("Processed %d items in %v", count, duration)
```

### 7. Performance-Aware Logging
```go
if logger.IsDebugEnabled() {
    expensiveData := computeData()
    logger.Debugf("Data: %v", expensiveData)
}
```

### 8. Multiple Loggers
```go
apiLogger := logging.GetLogger("api")
dbLogger := logging.GetLogger("database")

apiLogger.Info("API started")
dbLogger.Info("DB connected")
```

### 9. Configuration from Environment
```go
level := os.Getenv("LOG_LEVEL")
logging.SetGlobalLevel(logging.ParseLogLevel(level))
```

### 10. Custom Output
```go
logFile, _ := os.Create("app.log")
logger.SetOutput(logFile)
```

## Thread Safety

All logger operations are thread-safe and can be called concurrently from multiple goroutines:

```go
// Safe to use from multiple goroutines
for i := 0; i < 10; i++ {
    go func(id int) {
        logger.WithField("goroutine", id).Info("Processing")
    }(i)
}
```

## Performance Characteristics

- **Filtered logs** (below level): ~15 ns/op, 0 allocations
- **Simple log**: ~463 ns/op, 3 allocations
- **Log with fields**: ~1.3 μs/op, 15 allocations
- **Thread-safe**: Minimal lock contention using RWMutex

## Best Practices

1. **Use named loggers for components:**
   ```go
   apiLogger := logging.GetLogger("api")
   dbLogger := logging.GetLogger("database")
   ```

2. **Add context to logs:**
   ```go
   logger.WithField("user_id", userID).Info("Action performed")
   ```

3. **Use request IDs for tracing:**
   ```go
   ctx := logging.WithRequestID(ctx, requestID)
   logger.WithContext(ctx).Info("Processing")
   ```

4. **Check level for expensive operations:**
   ```go
   if logger.IsDebugEnabled() {
       // Expensive debug data computation
   }
   ```

5. **Chain loggers for reuse:**
   ```go
   userLogger := logger.WithField("user_id", userID)
   userLogger.Info("Login")
   userLogger.Info("View profile")
   ```

6. **Use formatted logging sparingly:**
   ```go
   // Prefer
   logger.WithField("count", count).Info("Items processed")

   // Over
   logger.Infof("Processed %d items", count)
   ```

7. **Always log errors with context:**
   ```go
   logger.WithError(err).WithFields(fields).Error("Operation failed")
   ```

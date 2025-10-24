package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// DEBUG level for detailed debugging information
	DEBUG LogLevel = iota
	// INFO level for general informational messages
	INFO
	// WARN level for warning conditions
	WARN
	// ERROR level for error conditions
	ERROR
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(s string) LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

// Logger is a structured JSON logger
type Logger struct {
	name   string
	output io.Writer
	level  LogLevel
	fields map[string]interface{}
	mu     sync.RWMutex
}

// logEntry represents a single log entry
type logEntry struct {
	Timestamp  string                 `json:"timestamp"`
	Level      string                 `json:"level"`
	Logger     string                 `json:"logger,omitempty"`
	Message    string                 `json:"message"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StackTrace []string               `json:"stack_trace,omitempty"`
}

// Global logger registry
var (
	globalMu     sync.RWMutex
	globalLevel  = INFO
	globalOutput = io.Writer(os.Stderr)
	loggers      = make(map[string]*Logger)
)

// NewLogger creates a new logger with the given name and level
func NewLogger(name string, level LogLevel) *Logger {
	globalMu.RLock()
	output := globalOutput
	globalMu.RUnlock()

	return &Logger{
		name:   name,
		output: output,
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// GetLogger returns a logger by name, creating it if it doesn't exist
func GetLogger(name string) *Logger {
	globalMu.Lock()
	defer globalMu.Unlock()

	if logger, exists := loggers[name]; exists {
		return logger
	}

	logger := &Logger{
		name:   name,
		output: globalOutput,
		level:  globalLevel,
		fields: make(map[string]interface{}),
	}
	loggers[name] = logger
	return logger
}

// SetGlobalLevel sets the log level for all future loggers
func SetGlobalLevel(level LogLevel) {
	globalMu.Lock()
	globalLevel = level
	globalMu.Unlock()
}

// SetOutput sets the output writer for all future loggers
func SetOutput(w io.Writer) {
	globalMu.Lock()
	globalOutput = w
	globalMu.Unlock()
}

// WithField returns a new logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	fields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value

	return &Logger{
		name:   l.name,
		output: l.output,
		level:  l.level,
		fields: fields,
	}
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		name:   l.name,
		output: l.output,
		level:  l.level,
		fields: newFields,
	}
}

// WithError returns a new logger with an error field
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// WithContext returns a new logger with fields from context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	logger := l
	if requestID := GetRequestID(ctx); requestID != "" {
		logger = logger.WithField("request_id", requestID)
	}

	return logger
}

// SetLevel sets the minimum log level for this logger
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetOutput sets the output writer for this logger
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.log(DEBUG, msg, nil, false)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil, false)
}

// Info logs an informational message
func (l *Logger) Info(msg string) {
	l.log(INFO, msg, nil, false)
}

// Infof logs a formatted informational message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil, false)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.log(WARN, msg, nil, false)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil, false)
}

// Error logs an error message with stack trace
func (l *Logger) Error(msg string) {
	l.log(ERROR, msg, nil, true)
}

// Errorf logs a formatted error message with stack trace
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil, true)
}

// ErrorWithError logs an error message with an error object and stack trace
func (l *Logger) ErrorWithError(msg string, err error) {
	l.log(ERROR, msg, err, true)
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, msg string, err error, captureStack bool) {
	l.mu.RLock()
	currentLevel := l.level
	output := l.output
	fields := l.fields
	name := l.name
	l.mu.RUnlock()

	// Check if we should log at this level
	if level < currentLevel {
		return
	}

	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Logger:    name,
		Message:   msg,
	}

	// Copy fields
	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{}, len(fields))
		for k, v := range fields {
			entry.Fields[k] = v
		}
	}

	// Extract request ID from fields if present
	if reqID, ok := fields["request_id"]; ok {
		if reqIDStr, ok := reqID.(string); ok {
			entry.RequestID = reqIDStr
		}
	}

	// Add error if provided
	if err != nil {
		entry.Error = err.Error()
	}

	// Capture stack trace for errors
	if captureStack {
		entry.StackTrace = captureStackTrace(3) // skip 3 frames (captureStackTrace, log, Error/Errorf)
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple format if JSON marshaling fails
		fmt.Fprintf(output, `{"timestamp":"%s","level":"%s","message":"failed to marshal log entry: %v","original_message":"%s"}`+"\n",
			time.Now().UTC().Format(time.RFC3339Nano), level.String(), err, msg)
		return
	}

	// Write to output
	data = append(data, '\n')
	output.Write(data)
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) []string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])

	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	var traces []string

	for {
		frame, more := frames.Next()
		traces = append(traces, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))

		if !more {
			break
		}
	}

	return traces
}

// Context helpers

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves a logger from the context, or returns a default logger
func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return GetLogger("default")
	}
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return GetLogger("default")
}

// IsLevelEnabled checks if a given log level is enabled
func (l *Logger) IsLevelEnabled(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.IsLevelEnabled(DEBUG)
}

// IsInfoEnabled returns true if info logging is enabled
func (l *Logger) IsInfoEnabled() bool {
	return l.IsLevelEnabled(INFO)
}

// IsWarnEnabled returns true if warn logging is enabled
func (l *Logger) IsWarnEnabled() bool {
	return l.IsLevelEnabled(WARN)
}

// IsErrorEnabled returns true if error logging is enabled
func (l *Logger) IsErrorEnabled() bool {
	return l.IsLevelEnabled(ERROR)
}

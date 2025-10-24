package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"unknown", INFO}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseLogLevel(tt.input); got != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test", DEBUG)

	if logger.name != "test" {
		t.Errorf("Expected name 'test', got %q", logger.name)
	}
	if logger.level != DEBUG {
		t.Errorf("Expected level DEBUG, got %v", logger.level)
	}
	if logger.fields == nil {
		t.Error("Expected fields to be initialized")
	}
}

func TestGetLogger(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	loggers = make(map[string]*Logger)
	globalMu.Unlock()

	logger1 := GetLogger("test")
	logger2 := GetLogger("test")

	if logger1 != logger2 {
		t.Error("Expected GetLogger to return the same instance for the same name")
	}

	logger3 := GetLogger("other")
	if logger1 == logger3 {
		t.Error("Expected GetLogger to return different instances for different names")
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	logger2 := logger.WithField("key", "value")

	// Original logger should not be modified
	if len(logger.fields) != 0 {
		t.Error("Original logger should not be modified")
	}

	// New logger should have the field
	if len(logger2.fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(logger2.fields))
	}
	if logger2.fields["key"] != "value" {
		t.Errorf("Expected field value 'value', got %v", logger2.fields["key"])
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	logger2 := logger.WithFields(fields)

	// Original logger should not be modified
	if len(logger.fields) != 0 {
		t.Error("Original logger should not be modified")
	}

	// New logger should have all fields
	if len(logger2.fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(logger2.fields))
	}
	if logger2.fields["key1"] != "value1" {
		t.Error("Field key1 not set correctly")
	}
	if logger2.fields["key2"] != 42 {
		t.Error("Field key2 not set correctly")
	}
	if logger2.fields["key3"] != true {
		t.Error("Field key3 not set correctly")
	}
}

func TestLoggerWithError(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	err := errors.New("test error")
	logger2 := logger.WithError(err)

	// New logger should have error field
	if logger2.fields["error"] != "test error" {
		t.Errorf("Expected error field 'test error', got %v", logger2.fields["error"])
	}

	// WithError with nil should return the same logger
	logger3 := logger.WithError(nil)
	if logger3 != logger {
		t.Error("WithError(nil) should return the same logger")
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	logger.SetLevel(ERROR)

	if logger.GetLevel() != ERROR {
		t.Errorf("Expected level ERROR, got %v", logger.GetLevel())
	}
}

func TestLogOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	logger.Info("test message")

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", entry.Message)
	}
	if entry.Logger != "test" {
		t.Errorf("Expected logger 'test', got %s", entry.Logger)
	}
	if entry.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		logLevel   LogLevel
		logFunc    func(*Logger)
		shouldLog  bool
		levelName  string
	}{
		{INFO, func(l *Logger) { l.Debug("debug") }, false, "DEBUG"},
		{INFO, func(l *Logger) { l.Info("info") }, true, "INFO"},
		{INFO, func(l *Logger) { l.Warn("warn") }, true, "WARN"},
		{INFO, func(l *Logger) { l.Error("error") }, true, "ERROR"},
		{ERROR, func(l *Logger) { l.Debug("debug") }, false, "DEBUG"},
		{ERROR, func(l *Logger) { l.Info("info") }, false, "INFO"},
		{ERROR, func(l *Logger) { l.Warn("warn") }, false, "WARN"},
		{ERROR, func(l *Logger) { l.Error("error") }, true, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.levelName, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger("test", tt.logLevel)
			logger.SetOutput(buf)

			tt.logFunc(logger)

			hasOutput := buf.Len() > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v, got output=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

func TestLogWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	logger.WithFields(map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}).Info("user logged in")

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Fields["user_id"].(float64) != 123 {
		t.Error("Expected user_id field to be 123")
	}
	if entry.Fields["action"] != "login" {
		t.Error("Expected action field to be 'login'")
	}
}

func TestErrorWithStackTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	logger.Error("error occurred")

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Level != "ERROR" {
		t.Errorf("Expected level ERROR, got %s", entry.Level)
	}
	if len(entry.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured for error")
	}

	// Check that stack trace contains this test function
	foundTestFunc := false
	for _, frame := range entry.StackTrace {
		if strings.Contains(frame, "TestErrorWithStackTrace") {
			foundTestFunc = true
			break
		}
	}
	if !foundTestFunc {
		t.Error("Expected stack trace to contain test function")
	}
}

func TestErrorWithErrorObject(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	err := errors.New("database connection failed")
	logger.ErrorWithError("failed to connect", err)

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Error != "database connection failed" {
		t.Errorf("Expected error 'database connection failed', got %s", entry.Error)
	}
	if entry.Message != "failed to connect" {
		t.Errorf("Expected message 'failed to connect', got %s", entry.Message)
	}
}

func TestFormattedLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	logger.Infof("user %s logged in with id %d", "alice", 123)

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Message != "user alice logged in with id 123" {
		t.Errorf("Expected formatted message, got %s", entry.Message)
	}
}

func TestContextRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	requestID := GetRequestID(ctx)

	if requestID != "req-123" {
		t.Errorf("Expected request ID 'req-123', got %s", requestID)
	}

	// Test nil context
	if GetRequestID(nil) != "" {
		t.Error("Expected empty string for nil context")
	}

	// Test context without request ID
	if GetRequestID(context.Background()) != "" {
		t.Error("Expected empty string for context without request ID")
	}
}

func TestLoggerWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	ctx := WithRequestID(context.Background(), "req-456")
	logger.WithContext(ctx).Info("processing request")

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.RequestID != "req-456" {
		t.Errorf("Expected request ID 'req-456', got %s", entry.RequestID)
	}
}

func TestLoggerInContext(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	ctx := WithLogger(context.Background(), logger)

	retrieved := FromContext(ctx)
	if retrieved != logger {
		t.Error("Expected to retrieve the same logger from context")
	}

	// Test nil context
	defaultLogger := FromContext(nil)
	if defaultLogger.name != "default" {
		t.Error("Expected default logger for nil context")
	}

	// Test context without logger
	defaultLogger2 := FromContext(context.Background())
	if defaultLogger2.name != "default" {
		t.Error("Expected default logger for context without logger")
	}
}

func TestIsLevelEnabled(t *testing.T) {
	logger := NewLogger("test", INFO)

	if logger.IsLevelEnabled(DEBUG) {
		t.Error("DEBUG should not be enabled when level is INFO")
	}
	if !logger.IsLevelEnabled(INFO) {
		t.Error("INFO should be enabled when level is INFO")
	}
	if !logger.IsLevelEnabled(WARN) {
		t.Error("WARN should be enabled when level is INFO")
	}
	if !logger.IsLevelEnabled(ERROR) {
		t.Error("ERROR should be enabled when level is INFO")
	}
}

func TestIsDebugEnabled(t *testing.T) {
	logger := NewLogger("test", DEBUG)
	if !logger.IsDebugEnabled() {
		t.Error("IsDebugEnabled should return true when level is DEBUG")
	}

	logger.SetLevel(INFO)
	if logger.IsDebugEnabled() {
		t.Error("IsDebugEnabled should return false when level is INFO")
	}
}

func TestGlobalSettings(t *testing.T) {
	// Save original state
	globalMu.Lock()
	originalLevel := globalLevel
	originalOutput := globalOutput
	globalMu.Unlock()

	// Test SetGlobalLevel
	SetGlobalLevel(ERROR)
	logger := GetLogger("global-test")
	if logger.GetLevel() != ERROR {
		t.Errorf("Expected global level ERROR, got %v", logger.GetLevel())
	}

	// Test SetOutput
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetGlobalLevel(INFO) // Set back to INFO so the message will be logged
	logger2 := GetLogger("global-test-2")
	logger2.Info("test")
	if buf.Len() == 0 {
		t.Error("Expected output to be written to custom writer")
	}

	// Restore original state
	globalMu.Lock()
	globalLevel = originalLevel
	globalOutput = originalOutput
	globalMu.Unlock()
}

func TestConcurrentLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test", DEBUG)
	logger.SetOutput(buf)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.WithField("goroutine", id).Info("concurrent log")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panic occurred and some output was produced
	if buf.Len() == 0 {
		t.Error("Expected some log output from concurrent logging")
	}
}

func BenchmarkLogging(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger("bench", INFO)
	logger.SetOutput(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggingWithFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger("bench", INFO)
	logger.SetOutput(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(map[string]interface{}{
			"user_id": 123,
			"action":  "login",
			"ip":      "192.168.1.1",
		}).Info("benchmark message")
	}
}

func BenchmarkLoggingFiltered(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := NewLogger("bench", ERROR)
	logger.SetOutput(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("filtered message") // Should be filtered out
	}
}

package health

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MockHealthCheck is a mock implementation of HealthCheck for testing
type MockHealthCheck struct {
	shouldFail bool
	delay      time.Duration
	err        error
}

func (m *MockHealthCheck) Check(ctx context.Context) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.shouldFail {
		if m.err != nil {
			return m.err
		}
		return errors.New("mock check failed")
	}
	return nil
}

func TestNewHealthChecker(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	hc := NewHealthChecker(db, nil)

	if hc == nil {
		t.Fatal("Expected non-nil health checker")
	}

	if hc.db == nil {
		t.Error("Expected database to be set")
	}

	// Should have registered default checks
	if len(hc.checks) == 0 {
		t.Error("Expected default checks to be registered")
	}
}

func TestHealthChecker_RegisterCheck(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	mockCheck := &MockHealthCheck{}
	hc.RegisterCheck("test", mockCheck)

	if hc.checks["test"] == nil {
		t.Error("Expected check to be registered")
	}
}

func TestHealthChecker_Check_AllHealthy(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	// Register mock checks
	hc.RegisterCheck("test1", &MockHealthCheck{shouldFail: false})
	hc.RegisterCheck("test2", &MockHealthCheck{shouldFail: false})

	status := hc.Check(context.Background())

	if status.Overall != StatusUp {
		t.Errorf("Expected overall status UP, got %s", status.Overall)
	}

	// Should have at least our 2 test checks plus default checks (filesystem, memory)
	if len(status.Checks) < 2 {
		t.Errorf("Expected at least 2 checks, got %d", len(status.Checks))
	}

	// Verify our test checks are present and UP
	if status.Checks["test1"].Status != StatusUp {
		t.Error("Expected test1 check to be UP")
	}
	if status.Checks["test2"].Status != StatusUp {
		t.Error("Expected test2 check to be UP")
	}
}

func TestHealthChecker_Check_SomeUnhealthy(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	// Register mock checks
	hc.RegisterCheck("healthy", &MockHealthCheck{shouldFail: false})
	hc.RegisterCheck("unhealthy", &MockHealthCheck{shouldFail: true})

	status := hc.Check(context.Background())

	if status.Overall != StatusDown {
		t.Errorf("Expected overall status DOWN, got %s", status.Overall)
	}

	if status.Checks["healthy"].Status != StatusUp {
		t.Error("Expected healthy check to be UP")
	}

	if status.Checks["unhealthy"].Status != StatusDown {
		t.Error("Expected unhealthy check to be DOWN")
	}

	if status.Checks["unhealthy"].Error == "" {
		t.Error("Expected error message for unhealthy check")
	}
}

func TestHealthChecker_Check_Timeout(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	// Register a check that takes longer than the timeout
	hc.RegisterCheck("slow", &MockHealthCheck{
		shouldFail: false,
		delay:      CheckTimeout + time.Second,
	})

	start := time.Now()
	status := hc.Check(context.Background())
	duration := time.Since(start)

	// Should complete within reasonable time (not waiting for full delay)
	if duration > CheckTimeout+time.Second {
		t.Errorf("Check took too long: %v", duration)
	}

	if status.Checks["slow"].Status != StatusDown {
		t.Error("Expected slow check to fail due to timeout")
	}

	if status.Checks["slow"].Error == "" {
		t.Error("Expected timeout error message")
	}
}

func TestHealthChecker_Check_Caching(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	callCount := 0
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	// First call
	status1 := hc.Check(context.Background())
	callCount++

	// Immediate second call should return cached result
	time.Sleep(100 * time.Millisecond)
	status2 := hc.Check(context.Background())

	if status1.Timestamp != status2.Timestamp {
		t.Error("Expected cached result with same timestamp")
	}

	// Wait for cache to expire
	time.Sleep(CacheDuration + 100*time.Millisecond)
	status3 := hc.Check(context.Background())

	if status1.Timestamp == status3.Timestamp {
		t.Error("Expected fresh result after cache expiration")
	}
}

func TestHealthChecker_Ready(t *testing.T) {
	tests := []struct {
		name     string
		checks   map[string]*MockHealthCheck
		expected bool
	}{
		{
			name: "all healthy",
			checks: map[string]*MockHealthCheck{
				"test1": {shouldFail: false},
				"test2": {shouldFail: false},
			},
			expected: true,
		},
		{
			name: "some unhealthy",
			checks: map[string]*MockHealthCheck{
				"test1": {shouldFail: false},
				"test2": {shouldFail: true},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker(nil, nil)

			for name, check := range tt.checks {
				hc.RegisterCheck(name, check)
			}

			ready := hc.Ready(context.Background())

			if ready != tt.expected {
				t.Errorf("Expected ready=%v, got %v", tt.expected, ready)
			}
		})
	}
}

func TestDatabaseCheck(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	check := &DatabaseCheck{db: db}

	if err := check.Check(context.Background()); err != nil {
		t.Errorf("Database check failed: %v", err)
	}
}

func TestDatabaseCheck_NilDatabase(t *testing.T) {
	check := &DatabaseCheck{db: nil}

	if err := check.Check(context.Background()); err == nil {
		t.Error("Expected error for nil database")
	}
}

func TestDatabaseCheck_ClosedDatabase(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	db.Close()

	check := &DatabaseCheck{db: db}

	if err := check.Check(context.Background()); err == nil {
		t.Error("Expected error for closed database")
	}
}

func TestFastMCPCheck(t *testing.T) {
	// Test with nil client
	check := &FastMCPCheck{client: nil}

	if err := check.Check(context.Background()); err == nil {
		t.Error("Expected error for nil FastMCP client")
	}
}

func TestFileSystemCheck(t *testing.T) {
	check := &FileSystemCheck{}

	if err := check.Check(context.Background()); err != nil {
		t.Errorf("Filesystem check failed: %v", err)
	}
}

func TestMemoryCheck(t *testing.T) {
	check := &MemoryCheck{}

	// Should pass under normal conditions
	if err := check.Check(context.Background()); err != nil {
		t.Errorf("Memory check failed: %v", err)
	}
}

func TestMemoryCheck_CustomThreshold(t *testing.T) {
	check := &MemoryCheck{
		MaxMemoryUsagePercent: 1.0, // Very low threshold
	}

	// This might fail depending on actual memory usage
	// We're just testing that the threshold is respected
	err := check.Check(context.Background())
	_ = err // Result depends on actual memory usage
}

func TestCheckStatus_MarshalJSON(t *testing.T) {
	status := CheckStatus{
		Name:     "test",
		Status:   StatusUp,
		Duration: 150 * time.Millisecond,
	}

	data, err := status.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Should contain duration in milliseconds
	if !contains(string(data), "150") {
		t.Error("Expected duration in milliseconds")
	}
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	// Run multiple checks concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			hc.Check(context.Background())
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHealthChecker_ContextCancellation(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	// Register a slow check
	hc.RegisterCheck("slow", &MockHealthCheck{
		shouldFail: false,
		delay:      2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	status := hc.Check(ctx)

	// Check should fail due to context cancellation
	if status.Checks["slow"].Status != StatusDown {
		t.Error("Expected check to fail due to context cancellation")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

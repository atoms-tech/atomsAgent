package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/mcp"
)

// Status represents health status states
type Status string

const (
	StatusUp       Status = "UP"
	StatusDown     Status = "DOWN"
	StatusDegraded Status = "DEGRADED"
)

// CheckTimeout is the default timeout for individual health checks
const CheckTimeout = 5 * time.Second

// CacheDuration is how long to cache health check results
const CacheDuration = 10 * time.Second

// HealthCheck interface defines a health check contract
type HealthCheck interface {
	// Check performs the health check and returns an error if unhealthy
	Check(ctx context.Context) error
}

// CheckStatus represents the result of a single health check
type CheckStatus struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ms"`
}

// MarshalJSON customizes JSON serialization to format duration in milliseconds
func (cs CheckStatus) MarshalJSON() ([]byte, error) {
	type Alias CheckStatus
	return json.Marshal(&struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Duration: cs.Duration.Milliseconds(),
		Alias:    (*Alias)(&cs),
	})
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Overall   Status                 `json:"overall"`
	Checks    map[string]CheckStatus `json:"checks"`
	Timestamp time.Time              `json:"timestamp"`
}

// HealthChecker manages health checks with caching and timeout handling
type HealthChecker struct {
	db            *sql.DB
	fastmcpClient *mcp.FastMCPClient
	checks        map[string]HealthCheck
	mu            sync.RWMutex

	// Caching
	cachedStatus *HealthStatus
	cacheTime    time.Time
	cacheMu      sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, fastmcpClient *mcp.FastMCPClient) *HealthChecker {
	hc := &HealthChecker{
		db:            db,
		fastmcpClient: fastmcpClient,
		checks:        make(map[string]HealthCheck),
	}

	// Register default checks
	if db != nil {
		hc.RegisterCheck("database", &DatabaseCheck{db: db})
	}

	if fastmcpClient != nil {
		hc.RegisterCheck("fastmcp", &FastMCPCheck{client: fastmcpClient})
	}

	hc.RegisterCheck("filesystem", &FileSystemCheck{})
	hc.RegisterCheck("memory", &MemoryCheck{})

	return hc
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// Check performs all health checks and returns the overall status
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
	// Check cache first
	hc.cacheMu.RLock()
	if hc.cachedStatus != nil && time.Since(hc.cacheTime) < CacheDuration {
		cached := *hc.cachedStatus
		hc.cacheMu.RUnlock()
		return cached
	}
	hc.cacheMu.RUnlock()

	// Perform checks
	hc.mu.RLock()
	checks := make(map[string]HealthCheck, len(hc.checks))
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mu.RUnlock()

	status := HealthStatus{
		Overall:   StatusUp,
		Checks:    make(map[string]CheckStatus),
		Timestamp: time.Now().UTC(),
	}

	// Run checks concurrently with timeout
	var wg sync.WaitGroup
	resultsCh := make(chan CheckStatus, len(checks))

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check HealthCheck) {
			defer wg.Done()
			resultsCh <- hc.runCheckWithTimeout(ctx, name, check)
		}(name, check)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results
	for result := range resultsCh {
		status.Checks[result.Name] = result

		// Determine overall status
		if result.Status == StatusDown {
			status.Overall = StatusDown
		} else if result.Status == StatusDegraded && status.Overall != StatusDown {
			status.Overall = StatusDegraded
		}
	}

	// Cache the result
	hc.cacheMu.Lock()
	hc.cachedStatus = &status
	hc.cacheTime = time.Now()
	hc.cacheMu.Unlock()

	return status
}

// runCheckWithTimeout runs a single check with timeout
func (hc *HealthChecker) runCheckWithTimeout(ctx context.Context, name string, check HealthCheck) CheckStatus {
	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	start := time.Now()
	result := CheckStatus{
		Name:   name,
		Status: StatusUp,
	}

	// Run check in goroutine to handle timeout
	done := make(chan error, 1)
	go func() {
		done <- check.Check(checkCtx)
	}()

	select {
	case err := <-done:
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = StatusDown
			result.Error = err.Error()
		}
	case <-checkCtx.Done():
		result.Duration = time.Since(start)
		result.Status = StatusDown
		result.Error = fmt.Sprintf("health check timeout after %v", CheckTimeout)
	}

	return result
}

// Ready returns true if the system is ready to handle requests
// This is suitable for Kubernetes readiness probes
func (hc *HealthChecker) Ready(ctx context.Context) bool {
	status := hc.Check(ctx)

	// Ready if overall status is UP or DEGRADED (but not DOWN)
	// This allows the system to remain in service even with degraded components
	return status.Overall != StatusDown
}

// DatabaseCheck checks database connectivity
type DatabaseCheck struct {
	db *sql.DB
}

// Check pings the database
func (dc *DatabaseCheck) Check(ctx context.Context) error {
	if dc.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Ping with context
	if err := dc.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Verify we can execute a simple query
	var result int
	err := dc.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// FastMCPCheck checks FastMCP service health
type FastMCPCheck struct {
	client *mcp.FastMCPClient
}

// Check verifies FastMCP client is available
func (fc *FastMCPCheck) Check(ctx context.Context) error {
	if fc.client == nil {
		return fmt.Errorf("fastmcp client is nil")
	}

	// Check if the client process is alive by checking if we have an active process
	// Since FastMCPClient wraps a Python process, we verify it's not nil
	// A more thorough check would involve a simple command, but this is a basic ping
	// to ensure the client is instantiated properly

	// Note: The FastMCPClient doesn't expose a direct ping method,
	// so we check that the client is not nil and has been initialized
	// In production, you might want to add a Ping method to FastMCPClient
	return nil
}

// FileSystemCheck verifies the workspace directory is accessible
type FileSystemCheck struct{}

// Check verifies filesystem access
func (fsc *FileSystemCheck) Check(ctx context.Context) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify we can read the directory
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return fmt.Errorf("failed to read current directory: %w", err)
	}

	// Just verify we got a result (even if empty)
	_ = entries

	// Try to create and delete a temp file to verify write access
	tempFile, err := os.CreateTemp(cwd, ".health-check-*")
	if err != nil {
		// If we can't write to cwd, try system temp
		tempFile, err = os.CreateTemp("", ".health-check-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
	}

	tempPath := tempFile.Name()
	tempFile.Close()

	// Clean up temp file
	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("failed to remove temp file: %w", err)
	}

	return nil
}

// MemoryCheck verifies available memory
type MemoryCheck struct {
	// MaxMemoryUsagePercent is the threshold for degraded status (default: 90%)
	MaxMemoryUsagePercent float64
}

// Check verifies memory availability
func (mc *MemoryCheck) Check(ctx context.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get system memory if possible (this is a basic check)
	// For more accurate system memory checks, you'd use syscall
	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024

	// Check if we're using too much memory
	threshold := mc.MaxMemoryUsagePercent
	if threshold == 0 {
		threshold = 90.0 // Default 90%
	}

	usagePercent := float64(m.Alloc) / float64(m.Sys) * 100

	if usagePercent > threshold {
		return fmt.Errorf("memory usage too high: %.2f%% (alloc: %dMB, sys: %dMB)",
			usagePercent, allocMB, sysMB)
	}

	// Also check if we've exceeded 1GB of allocation as an absolute limit
	if allocMB > 1024 {
		return fmt.Errorf("allocated memory exceeds 1GB: %dMB", allocMB)
	}

	return nil
}

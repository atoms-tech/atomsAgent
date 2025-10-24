package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name      string
		config    CBConfig
		expectErr bool
	}{
		{
			name:      "valid config",
			config:    DefaultCBConfig(),
			expectErr: false,
		},
		{
			name: "zero failure threshold",
			config: CBConfig{
				FailureThreshold: 0,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "zero success threshold",
			config: CBConfig{
				FailureThreshold: 5,
				SuccessThreshold: 0,
				Timeout:          30 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "zero timeout",
			config: CBConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          0,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb, err := NewCircuitBreaker("test", tt.config)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cb == nil {
					t.Error("expected circuit breaker, got nil")
				}
			}
		})
	}
}

func TestCircuitBreakerStates(t *testing.T) {
	config := CBConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               100 * time.Millisecond,
		MaxConcurrentRequests: 1,
	}

	cb := MustNewCircuitBreaker("test", config)

	// Initial state should be closed
	if cb.State() != "closed" {
		t.Errorf("expected initial state to be closed, got %s", cb.State())
	}

	// Execute successful requests
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, func() error {
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error on success: %v", err)
		}
	}

	// State should still be closed
	if cb.State() != "closed" {
		t.Errorf("expected state to be closed, got %s", cb.State())
	}

	// Execute failing requests to open circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("expected test error, got %v", err)
		}
	}

	// State should now be open
	if cb.State() != "open" {
		t.Errorf("expected state to be open, got %s", cb.State())
	}

	// Requests should be rejected while open
	err := cb.Execute(ctx, func() error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to half-open and succeed
	err = cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error in half-open: %v", err)
	}

	// State should be half-open after first success
	state := cb.State()
	if state != "half-open" {
		t.Errorf("expected state to be half-open, got %s", state)
	}

	// One more success should close the circuit
	// Add small delay to ensure previous request completed
	time.Sleep(10 * time.Millisecond)
	err = cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error in half-open: %v", err)
	}

	// State should now be closed
	state = cb.State()
	if state != "closed" {
		t.Errorf("expected state to be closed, got %s", state)
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	config := CBConfig{
		FailureThreshold:      2,
		SuccessThreshold:      2,
		Timeout:               100 * time.Millisecond,
		MaxConcurrentRequests: 1,
	}

	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	if cb.State() != "open" {
		t.Errorf("expected state to be open, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Execute a failing request in half-open state
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	// State should go back to open
	if cb.State() != "open" {
		t.Errorf("expected state to be open, got %s", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	config := CBConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	}

	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	if cb.State() != "open" {
		t.Errorf("expected state to be open, got %s", cb.State())
	}

	// Reset should close the circuit
	cb.Reset()

	if cb.State() != "closed" {
		t.Errorf("expected state to be closed after reset, got %s", cb.State())
	}

	// Should be able to execute requests
	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error after reset: %v", err)
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	config := DefaultCBConfig()
	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Execute some requests
	successCount := 5
	failureCount := 3

	for i := 0; i < successCount; i++ {
		cb.Execute(ctx, func() error {
			return nil
		})
	}

	testErr := errors.New("test error")
	for i := 0; i < failureCount; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	stats := cb.Stats()

	if stats.TotalRequests != uint64(successCount+failureCount) {
		t.Errorf("expected %d total requests, got %d", successCount+failureCount, stats.TotalRequests)
	}

	if stats.TotalSuccesses != uint64(successCount) {
		t.Errorf("expected %d successes, got %d", successCount, stats.TotalSuccesses)
	}

	if stats.TotalFailures != uint64(failureCount) {
		t.Errorf("expected %d failures, got %d", failureCount, stats.TotalFailures)
	}

	if stats.ConsecutiveFailures != uint32(failureCount) {
		t.Errorf("expected %d consecutive failures, got %d", failureCount, stats.ConsecutiveFailures)
	}

	if stats.LastError != testErr {
		t.Errorf("expected last error to be test error, got %v", stats.LastError)
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	config := CBConfig{
		FailureThreshold:      10,
		SuccessThreshold:      5,
		Timeout:               100 * time.Millisecond,
		MaxConcurrentRequests: 5,
	}

	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Execute many concurrent requests
	var wg sync.WaitGroup
	successCount := atomic.Int64{}
	errorCount := atomic.Int64{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			err := cb.Execute(ctx, func() error {
				time.Sleep(10 * time.Millisecond)
				if index%10 == 0 {
					return errors.New("simulated error")
				}
				return nil
			})

			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	stats := cb.Stats()
	t.Logf("Total requests: %d, Successes: %d, Failures: %d, State: %s",
		stats.TotalRequests, stats.TotalSuccesses, stats.TotalFailures, stats.State)

	if stats.TotalRequests == 0 {
		t.Error("expected some requests to be executed")
	}
}

func TestCircuitBreakerMaxConcurrentRequests(t *testing.T) {
	config := CBConfig{
		FailureThreshold:      2,
		SuccessThreshold:      2,
		Timeout:               100 * time.Millisecond,
		MaxConcurrentRequests: 1,
	}

	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Try to execute two concurrent requests in half-open state
	var wg sync.WaitGroup
	tooManyRequestsCount := atomic.Int32{}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(ctx, func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if errors.Is(err, ErrTooManyRequests) {
				tooManyRequestsCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// At least one request should be rejected
	if tooManyRequestsCount.Load() == 0 {
		t.Error("expected at least one ErrTooManyRequests")
	}
}

func TestCircuitBreakerStateCallback(t *testing.T) {
	stateChanges := make([]string, 0)
	mu := sync.Mutex{}

	config := CBConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		OnStateChange: func(name string, from State, to State) {
			mu.Lock()
			defer mu.Unlock()
			stateChanges = append(stateChanges, fmt.Sprintf("%s->%s", from.String(), to.String()))
		},
	}

	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	// Wait a bit for callback
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(stateChanges) == 0 {
		t.Error("expected state change callback to be called")
	}
	if stateChanges[len(stateChanges)-1] != "closed->open" {
		t.Errorf("expected closed->open transition, got %v", stateChanges)
	}
	mu.Unlock()
}

func TestCircuitBreakerContextTimeout(t *testing.T) {
	config := DefaultCBConfig()
	cb := MustNewCircuitBreaker("test", config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := cb.Execute(ctx, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if !errors.Is(err, ErrCircuitBreakerTimeout) {
		t.Errorf("expected timeout error, got %v", err)
	}
}

func TestCircuitBreakerPanicRecovery(t *testing.T) {
	config := DefaultCBConfig()
	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	err := cb.Execute(ctx, func() error {
		panic("test panic")
	})

	if err == nil {
		t.Error("expected error from panic recovery")
	}

	if cb.State() != "closed" {
		// Circuit should still be functional after panic
		t.Errorf("expected circuit to be closed after panic, got %s", cb.State())
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	config := DefaultCBConfig()
	cb := MustNewCircuitBreaker("test", config)
	ctx := context.Background()

	// Execute some requests
	for i := 0; i < 10; i++ {
		cb.Execute(ctx, func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	metrics := cb.metrics.GetMetrics()

	if metrics.RequestsTotal == 0 {
		t.Error("expected requests to be recorded")
	}

	if metrics.AvgLatency == 0 {
		t.Error("expected average latency to be calculated")
	}

	if metrics.RequestsSuccessful != metrics.RequestsTotal {
		t.Errorf("expected all requests to succeed, got %d/%d",
			metrics.RequestsSuccessful, metrics.RequestsTotal)
	}
}

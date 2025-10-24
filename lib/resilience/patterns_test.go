package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMultiCircuitBreaker(t *testing.T) {
	mcb := NewMultiCircuitBreaker(DefaultCBConfig())

	// Get or create circuit breaker
	cb1 := mcb.GetOrCreate("service-1")
	cb2 := mcb.GetOrCreate("service-1") // Should return same instance

	if cb1 != cb2 {
		t.Error("expected same circuit breaker instance")
	}

	// Execute through multi circuit breaker
	ctx := context.Background()
	err := mcb.Execute(ctx, "service-2", func() error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that we now have 2 circuit breakers
	all := mcb.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 circuit breakers, got %d", len(all))
	}
}

func TestMultiCircuitBreakerHealth(t *testing.T) {
	config := CBConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	mcb := NewMultiCircuitBreaker(config)
	ctx := context.Background()

	// Create some circuit breakers in different states
	// Healthy service
	mcb.Execute(ctx, "healthy", func() error {
		return nil
	})

	// Unhealthy service
	testErr := errors.New("error")
	for i := 0; i < 2; i++ {
		mcb.Execute(ctx, "unhealthy", func() error {
			return testErr
		})
	}

	health := mcb.GetHealthStatus()

	if len(health.Healthy) != 1 {
		t.Errorf("expected 1 healthy service, got %d", len(health.Healthy))
	}

	if len(health.Unhealthy) != 1 {
		t.Errorf("expected 1 unhealthy service, got %d", len(health.Unhealthy))
	}
}

func TestCircuitBreakerGroup(t *testing.T) {
	cb1 := MustNewCircuitBreaker("cb1", DefaultCBConfig())
	cb2 := MustNewCircuitBreaker("cb2", DefaultCBConfig())

	group := NewCircuitBreakerGroup(cb1, cb2)

	ctx := context.Background()
	testErr := errors.New("error")

	results := group.ExecuteAll(ctx,
		func() error {
			return nil
		},
		func() error {
			return testErr
		},
	)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if results[0] != nil {
		t.Errorf("expected first result to be nil, got %v", results[0])
	}

	if results[1] != testErr {
		t.Errorf("expected second result to be test error, got %v", results[1])
	}
}

func TestCircuitBreakerWithRetry(t *testing.T) {
	cbConfig := CBConfig{
		FailureThreshold: 10, // High threshold so circuit doesn't open
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}

	retryConfig := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	cbr := NewCircuitBreakerWithRetry("retry-test", cbConfig, retryConfig)

	// Test successful retry
	attempts := 0
	ctx := context.Background()

	err := cbr.Execute(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestCircuitBreakerWithRetryTimeout(t *testing.T) {
	cbConfig := CBConfig{
		FailureThreshold: 10,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}

	retryConfig := RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	cbr := NewCircuitBreakerWithRetry("retry-test", cbConfig, retryConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := cbr.Execute(ctx, func() error {
		return errors.New("error")
	})

	if err == nil {
		t.Error("expected error due to context timeout")
	}
}

func TestCircuitBreakerWithFallback(t *testing.T) {
	config := DefaultCBConfig()

	fallbackCalled := false
	cbf := NewCircuitBreakerWithFallback("fallback-test", config, func() (string, error) {
		fallbackCalled = true
		return "fallback-value", nil
	})

	ctx := context.Background()

	// Test successful execution (no fallback)
	result, err := cbf.Execute(ctx, func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "success" {
		t.Errorf("expected 'success', got '%s'", result)
	}

	if fallbackCalled {
		t.Error("fallback should not have been called")
	}

	// Test failed execution (with fallback)
	result, err = cbf.Execute(ctx, func() (string, error) {
		return "", errors.New("error")
	})

	if err != nil {
		t.Errorf("expected fallback to handle error, got %v", err)
	}

	if result != "fallback-value" {
		t.Errorf("expected 'fallback-value', got '%s'", result)
	}

	if !fallbackCalled {
		t.Error("fallback should have been called")
	}
}

func TestAdaptiveCircuitBreaker(t *testing.T) {
	config := DefaultCBConfig()
	acb := NewAdaptiveCircuitBreaker("adaptive-test", config, 100*time.Millisecond)
	defer acb.Stop()

	ctx := context.Background()

	// Execute some requests
	for i := 0; i < 10; i++ {
		acb.Execute(ctx, func() error {
			if i%2 == 0 {
				return errors.New("error")
			}
			return nil
		})
	}

	// Wait for adjustment
	time.Sleep(150 * time.Millisecond)

	errorRate := acb.GetErrorRate()
	if errorRate == 0 {
		t.Error("expected error rate to be calculated")
	}

	// Error rate should be around 0.5 (5 errors out of 10)
	if errorRate < 0.4 || errorRate > 0.6 {
		t.Errorf("expected error rate around 0.5, got %f", errorRate)
	}
}

func TestMultiCircuitBreakerResetAll(t *testing.T) {
	config := CBConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
	}
	mcb := NewMultiCircuitBreaker(config)
	ctx := context.Background()

	// Open multiple circuits
	testErr := errors.New("error")
	for i := 0; i < 2; i++ {
		mcb.Execute(ctx, "service-1", func() error {
			return testErr
		})
		mcb.Execute(ctx, "service-2", func() error {
			return testErr
		})
	}

	// Both should be open
	if mcb.GetOrCreate("service-1").State() != "open" {
		t.Error("expected service-1 to be open")
	}
	if mcb.GetOrCreate("service-2").State() != "open" {
		t.Error("expected service-2 to be open")
	}

	// Reset all
	mcb.ResetAll()

	// Both should be closed
	if mcb.GetOrCreate("service-1").State() != "closed" {
		t.Error("expected service-1 to be closed after reset")
	}
	if mcb.GetOrCreate("service-2").State() != "closed" {
		t.Error("expected service-2 to be closed after reset")
	}
}

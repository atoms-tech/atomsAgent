package resilience

import (
	"context"
	"sync"
	"time"
)

// MultiCircuitBreaker manages multiple circuit breakers for different services
type MultiCircuitBreaker struct {
	mu              sync.RWMutex
	circuitBreakers map[string]*CircuitBreaker
	defaultConfig   CBConfig
}

// NewMultiCircuitBreaker creates a new multi-circuit breaker manager
func NewMultiCircuitBreaker(defaultConfig CBConfig) *MultiCircuitBreaker {
	return &MultiCircuitBreaker{
		circuitBreakers: make(map[string]*CircuitBreaker),
		defaultConfig:   defaultConfig,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *MultiCircuitBreaker) GetOrCreate(name string) *CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.circuitBreakers[name]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = m.circuitBreakers[name]; exists {
		return cb
	}

	cb = MustNewCircuitBreaker(name, m.defaultConfig)
	m.circuitBreakers[name] = cb
	return cb
}

// Execute executes a function with the named circuit breaker
func (m *MultiCircuitBreaker) Execute(ctx context.Context, name string, fn func() error) error {
	cb := m.GetOrCreate(name)
	return cb.Execute(ctx, fn)
}

// GetAll returns all circuit breakers
func (m *MultiCircuitBreaker) GetAll() map[string]*CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(m.circuitBreakers))
	for name, cb := range m.circuitBreakers {
		result[name] = cb
	}
	return result
}

// ResetAll resets all circuit breakers
func (m *MultiCircuitBreaker) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cb := range m.circuitBreakers {
		cb.Reset()
	}
}

// HealthStatus represents the health status of all circuit breakers
type HealthStatus struct {
	Healthy   []string
	Degraded  []string // Half-open
	Unhealthy []string // Open
}

// GetHealthStatus returns the health status of all circuit breakers
func (m *MultiCircuitBreaker) GetHealthStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := HealthStatus{
		Healthy:   make([]string, 0),
		Degraded:  make([]string, 0),
		Unhealthy: make([]string, 0),
	}

	for name, cb := range m.circuitBreakers {
		switch cb.StateEnum() {
		case StateClosed:
			status.Healthy = append(status.Healthy, name)
		case StateHalfOpen:
			status.Degraded = append(status.Degraded, name)
		case StateOpen:
			status.Unhealthy = append(status.Unhealthy, name)
		}
	}

	return status
}

// CircuitBreakerGroup executes functions with multiple circuit breakers in parallel
type CircuitBreakerGroup struct {
	cbs []*CircuitBreaker
}

// NewCircuitBreakerGroup creates a new circuit breaker group
func NewCircuitBreakerGroup(cbs ...*CircuitBreaker) *CircuitBreakerGroup {
	return &CircuitBreakerGroup{cbs: cbs}
}

// ExecuteAll executes all functions in parallel with their respective circuit breakers
func (g *CircuitBreakerGroup) ExecuteAll(ctx context.Context, fns ...func() error) []error {
	if len(fns) != len(g.cbs) {
		panic("number of functions must match number of circuit breakers")
	}

	results := make([]error, len(fns))
	var wg sync.WaitGroup

	for i := range fns {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = g.cbs[index].Execute(ctx, fns[index])
		}(i)
	}

	wg.Wait()
	return results
}

// Retry configuration for combining circuit breaker with retry logic
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []error
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// CircuitBreakerWithRetry wraps a circuit breaker with retry logic
type CircuitBreakerWithRetry struct {
	cb          *CircuitBreaker
	retryConfig RetryConfig
}

// NewCircuitBreakerWithRetry creates a circuit breaker with retry logic
func NewCircuitBreakerWithRetry(name string, cbConfig CBConfig, retryConfig RetryConfig) *CircuitBreakerWithRetry {
	return &CircuitBreakerWithRetry{
		cb:          MustNewCircuitBreaker(name, cbConfig),
		retryConfig: retryConfig,
	}
}

// Execute executes the function with retry logic
func (r *CircuitBreakerWithRetry) Execute(ctx context.Context, fn func() error) error {
	delay := r.retryConfig.InitialDelay

	for attempt := 0; attempt < r.retryConfig.MaxAttempts; attempt++ {
		err := r.cb.Execute(ctx, fn)
		if err == nil {
			return nil
		}

		// Don't retry if circuit is open
		if err == ErrCircuitOpen {
			return err
		}

		// Check if we should retry
		if !r.shouldRetry(err) {
			return err
		}

		// Last attempt, don't wait
		if attempt == r.retryConfig.MaxAttempts-1 {
			return err
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * r.retryConfig.BackoffFactor)
		if delay > r.retryConfig.MaxDelay {
			delay = r.retryConfig.MaxDelay
		}
	}

	return ErrCircuitBreakerTimeout
}

func (r *CircuitBreakerWithRetry) shouldRetry(err error) bool {
	if len(r.retryConfig.RetryableErrors) == 0 {
		// Retry all errors if no specific errors configured
		return true
	}

	for _, retryableErr := range r.retryConfig.RetryableErrors {
		if err == retryableErr {
			return true
		}
	}
	return false
}

// CircuitBreaker returns the underlying circuit breaker
func (r *CircuitBreakerWithRetry) CircuitBreaker() *CircuitBreaker {
	return r.cb
}

// FallbackFunc is a function that provides a fallback value
type FallbackFunc[T any] func() (T, error)

// CircuitBreakerWithFallback wraps a circuit breaker with fallback logic
type CircuitBreakerWithFallback[T any] struct {
	cb       *CircuitBreaker
	fallback FallbackFunc[T]
}

// NewCircuitBreakerWithFallback creates a circuit breaker with fallback
func NewCircuitBreakerWithFallback[T any](name string, config CBConfig, fallback FallbackFunc[T]) *CircuitBreakerWithFallback[T] {
	return &CircuitBreakerWithFallback[T]{
		cb:       MustNewCircuitBreaker(name, config),
		fallback: fallback,
	}
}

// Execute executes the function and falls back on failure
func (f *CircuitBreakerWithFallback[T]) Execute(ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	var fnErr error

	err := f.cb.Execute(ctx, func() error {
		result, fnErr = fn()
		return fnErr
	})

	if err != nil {
		// Try fallback
		if f.fallback != nil {
			return f.fallback()
		}
		return result, err
	}

	return result, nil
}

// CircuitBreaker returns the underlying circuit breaker
func (f *CircuitBreakerWithFallback[T]) CircuitBreaker() *CircuitBreaker {
	return f.cb
}

// AdaptiveCircuitBreaker adjusts thresholds based on error rates
type AdaptiveCircuitBreaker struct {
	cb               *CircuitBreaker
	mu               sync.RWMutex
	errorRate        float64
	adjustmentTicker *time.Ticker
	baseConfig       CBConfig
}

// NewAdaptiveCircuitBreaker creates a circuit breaker with adaptive thresholds
func NewAdaptiveCircuitBreaker(name string, config CBConfig, adjustmentInterval time.Duration) *AdaptiveCircuitBreaker {
	acb := &AdaptiveCircuitBreaker{
		cb:               MustNewCircuitBreaker(name, config),
		baseConfig:       config,
		adjustmentTicker: time.NewTicker(adjustmentInterval),
	}

	go acb.adjustThresholds()
	return acb
}

func (a *AdaptiveCircuitBreaker) adjustThresholds() {
	for range a.adjustmentTicker.C {
		stats := a.cb.Stats()

		if stats.TotalRequests == 0 {
			continue
		}

		// Calculate error rate
		errorRate := float64(stats.TotalFailures) / float64(stats.TotalRequests)
		a.mu.Lock()
		a.errorRate = errorRate
		a.mu.Unlock()

		// Adjust thresholds based on error rate
		// This is a simple example - you might want more sophisticated logic
		if errorRate > 0.5 {
			// High error rate, be more sensitive
			// In practice, you'd need to recreate the circuit breaker with new config
			// or implement dynamic threshold adjustment in the main circuit breaker
		} else if errorRate < 0.1 {
			// Low error rate, be more tolerant
		}
	}
}

// Execute executes the function through the adaptive circuit breaker
func (a *AdaptiveCircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	return a.cb.Execute(ctx, fn)
}

// GetErrorRate returns the current error rate
func (a *AdaptiveCircuitBreaker) GetErrorRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.errorRate
}

// Stop stops the adaptive circuit breaker
func (a *AdaptiveCircuitBreaker) Stop() {
	a.adjustmentTicker.Stop()
}

// CircuitBreaker returns the underlying circuit breaker
func (a *AdaptiveCircuitBreaker) CircuitBreaker() *CircuitBreaker {
	return a.cb
}

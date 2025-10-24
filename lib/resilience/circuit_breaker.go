package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed allows all requests through
	StateClosed State = iota
	// StateOpen rejects all requests
	StateOpen
	// StateHalfOpen allows limited requests to test recovery
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when too many requests are made in half-open state
	ErrTooManyRequests = errors.New("too many requests in half-open state")
	// ErrCircuitBreakerTimeout is returned when operation times out
	ErrCircuitBreakerTimeout = errors.New("circuit breaker operation timeout")
)

// CBConfig contains circuit breaker configuration
type CBConfig struct {
	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold uint32
	// SuccessThreshold is the number of consecutive successes to close from half-open
	SuccessThreshold uint32
	// Timeout is the duration to stay open before transitioning to half-open
	Timeout time.Duration
	// MaxConcurrentRequests is the maximum number of concurrent requests in half-open state
	MaxConcurrentRequests uint32
	// OnStateChange is called when the state changes
	OnStateChange func(name string, from State, to State)
}

// DefaultCBConfig returns a default circuit breaker configuration
func DefaultCBConfig() CBConfig {
	return CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 1,
		OnStateChange:         nil,
	}
}

// Validate validates the configuration
func (c *CBConfig) Validate() error {
	if c.FailureThreshold == 0 {
		return errors.New("failure threshold must be greater than 0")
	}
	if c.SuccessThreshold == 0 {
		return errors.New("success threshold must be greater than 0")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	if c.MaxConcurrentRequests == 0 {
		c.MaxConcurrentRequests = 1
	}
	return nil
}

// CBStats contains circuit breaker statistics
type CBStats struct {
	// TotalRequests is the total number of requests
	TotalRequests uint64
	// TotalSuccesses is the total number of successful requests
	TotalSuccesses uint64
	// TotalFailures is the total number of failed requests
	TotalFailures uint64
	// ConsecutiveSuccesses is the current number of consecutive successes
	ConsecutiveSuccesses uint32
	// ConsecutiveFailures is the current number of consecutive failures
	ConsecutiveFailures uint32
	// LastError is the last error encountered
	LastError error
	// LastErrorTime is the time of the last error
	LastErrorTime time.Time
	// State is the current state
	State State
	// StateChangedAt is when the state last changed
	StateChangedAt time.Time
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name   string
	config CBConfig

	mu                   sync.RWMutex
	state                State
	stateChangedAt       time.Time
	totalRequests        uint64
	totalSuccesses       uint64
	totalFailures        uint64
	consecutiveSuccesses uint32
	consecutiveFailures  uint32
	lastError            error
	lastErrorTime        time.Time
	halfOpenRequests     uint32
	metrics              *CBMetrics
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CBConfig) (*CircuitBreaker, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cb := &CircuitBreaker{
		name:           name,
		config:         config,
		state:          StateClosed,
		stateChangedAt: time.Now(),
		metrics:        NewCBMetrics(name),
	}

	return cb, nil
}

// MustNewCircuitBreaker creates a new circuit breaker and panics on error
func MustNewCircuitBreaker(name string, config CBConfig) *CircuitBreaker {
	cb, err := NewCircuitBreaker(name, config)
	if err != nil {
		panic(err)
	}
	return cb
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we can execute
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute with timeout if context has deadline
	errChan := make(chan error, 1)
	start := time.Now()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic recovered: %v", r)
			}
		}()
		errChan <- fn()
	}()

	var err error
	select {
	case <-ctx.Done():
		err = fmt.Errorf("%w: %v", ErrCircuitBreakerTimeout, ctx.Err())
	case err = <-errChan:
	}

	// Record result
	duration := time.Since(start)
	cb.afterRequest(err, duration)

	return err
}

// beforeRequest checks if the request can proceed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.stateChangedAt) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 1
			return nil
		}
		cb.metrics.RecordRejection()
		return ErrCircuitOpen

	case StateHalfOpen:
		// Check if we can allow another request
		if cb.halfOpenRequests >= cb.config.MaxConcurrentRequests {
			cb.metrics.RecordRejection()
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil

	default:
		return fmt.Errorf("unknown circuit breaker state: %v", cb.state)
	}
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(err error, duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Decrement half-open requests counter
	if cb.state == StateHalfOpen && cb.halfOpenRequests > 0 {
		cb.halfOpenRequests--
	}

	// Record metrics
	cb.metrics.RecordRequest(cb.state, err == nil, duration)

	if err != nil {
		cb.onFailure(err)
	} else {
		cb.onSuccess()
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.totalSuccesses++
	cb.consecutiveSuccesses++
	cb.consecutiveFailures = 0

	switch cb.state {
	case StateClosed:
		// Already closed, nothing to do

	case StateHalfOpen:
		// Check if we have enough successes to close
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.halfOpenRequests = 0
		}

	case StateOpen:
		// Should not happen, but handle gracefully
		cb.setState(StateHalfOpen)
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure(err error) {
	cb.totalFailures++
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	cb.lastError = err
	cb.lastErrorTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open
		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.setState(StateOpen)
		cb.halfOpenRequests = 0

	case StateOpen:
		// Already open, just update the state change time
		cb.stateChangedAt = time.Now()
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.stateChangedAt = time.Now()

	// Record state change
	cb.metrics.RecordStateChange(newState)

	// Call callback if set
	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid blocking
		go cb.config.OnStateChange(cb.name, oldState, newState)
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state.String()
}

// StateEnum returns the current state as an enum
func (cb *CircuitBreaker) StateEnum() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns the current statistics
func (cb *CircuitBreaker) Stats() CBStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CBStats{
		TotalRequests:        cb.totalRequests,
		TotalSuccesses:       cb.totalSuccesses,
		TotalFailures:        cb.totalFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		ConsecutiveFailures:  cb.consecutiveFailures,
		LastError:            cb.lastError,
		LastErrorTime:        cb.lastErrorTime,
		State:                cb.state,
		StateChangedAt:       cb.stateChangedAt,
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.stateChangedAt = time.Now()
	cb.consecutiveSuccesses = 0
	cb.consecutiveFailures = 0
	cb.halfOpenRequests = 0

	if oldState != StateClosed {
		cb.metrics.RecordStateChange(StateClosed)
		if cb.config.OnStateChange != nil {
			go cb.config.OnStateChange(cb.name, oldState, StateClosed)
		}
	}
}

// Name returns the circuit breaker name
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Config returns the circuit breaker configuration
func (cb *CircuitBreaker) Config() CBConfig {
	return cb.config
}

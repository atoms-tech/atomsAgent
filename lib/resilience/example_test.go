package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/resilience"
)

// Example demonstrates basic circuit breaker usage
func Example() {
	// Create a circuit breaker with custom configuration
	config := resilience.CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 1,
	}

	cb := resilience.MustNewCircuitBreaker("example-service", config)

	// Execute a protected operation
	ctx := context.Background()
	err := cb.Execute(ctx, func() error {
		// Your operation here
		return nil
	})

	if err != nil {
		log.Printf("Operation failed: %v", err)
	}

	// Check circuit breaker state
	fmt.Printf("Current state: %s\n", cb.State())
	// Output: Current state: closed
}

// ExampleCircuitBreaker_httpClient demonstrates using circuit breaker with HTTP client
func ExampleCircuitBreaker_httpClient() {
	config := resilience.DefaultCBConfig()
	cb := resilience.MustNewCircuitBreaker("api-client", config)

	makeRequest := func(url string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return cb.Execute(ctx, func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				return fmt.Errorf("server error: %d", resp.StatusCode)
			}

			return nil
		})
	}

	// Make requests
	for i := 0; i < 10; i++ {
		err := makeRequest("https://example.com/api")
		if err != nil {
			if errors.Is(err, resilience.ErrCircuitOpen) {
				log.Printf("Circuit is open, skipping request")
				continue
			}
			log.Printf("Request failed: %v", err)
		}
	}

	// Get statistics
	stats := cb.Stats()
	fmt.Printf("Total requests: %d, Failures: %d\n", stats.TotalRequests, stats.TotalFailures)
}

// ExampleCircuitBreaker_withStateCallback demonstrates state change callbacks
func ExampleCircuitBreaker_withStateCallback() {
	config := resilience.CBConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          10 * time.Second,
		OnStateChange: func(name string, from resilience.State, to resilience.State) {
			log.Printf("[%s] Circuit breaker state changed: %s -> %s", name, from, to)
		},
	}

	cb := resilience.MustNewCircuitBreaker("monitored-service", config)
	ctx := context.Background()

	// Simulate failures
	testErr := errors.New("simulated error")
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error {
			return testErr
		})
		if err != nil {
			log.Printf("Request %d failed: %v", i+1, err)
		}
	}

	// Circuit should now be open
	fmt.Printf("State: %s\n", cb.State())
	// Output: State: open
}

// ExampleCircuitBreaker_stats demonstrates statistics tracking
func ExampleCircuitBreaker_stats() {
	cb := resilience.MustNewCircuitBreaker("stats-example", resilience.DefaultCBConfig())
	ctx := context.Background()

	// Execute some operations
	for i := 0; i < 10; i++ {
		cb.Execute(ctx, func() error {
			if i%3 == 0 {
				return errors.New("occasional error")
			}
			return nil
		})
	}

	// Get detailed statistics
	stats := cb.Stats()
	fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("Successes: %d\n", stats.TotalSuccesses)
	fmt.Printf("Failures: %d\n", stats.TotalFailures)
	fmt.Printf("Current State: %s\n", stats.State)
	fmt.Printf("Consecutive Failures: %d\n", stats.ConsecutiveFailures)
	if stats.LastError != nil {
		fmt.Printf("Last Error: %v\n", stats.LastError)
	}
}

// ExampleCircuitBreaker_reset demonstrates manual reset
func ExampleCircuitBreaker_reset() {
	config := resilience.CBConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}

	cb := resilience.MustNewCircuitBreaker("reset-example", config)
	ctx := context.Background()

	// Force circuit open
	testErr := errors.New("error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testErr
		})
	}

	fmt.Printf("State before reset: %s\n", cb.State())

	// Manually reset the circuit breaker
	cb.Reset()

	fmt.Printf("State after reset: %s\n", cb.State())

	// Output:
	// State before reset: open
	// State after reset: closed
}

// ExampleCircuitBreaker_multipleServices demonstrates managing multiple circuit breakers
func ExampleCircuitBreaker_multipleServices() {
	type ServiceClient struct {
		name string
		cb   *resilience.CircuitBreaker
	}

	// Create circuit breakers for different services
	services := []ServiceClient{
		{
			name: "auth-service",
			cb:   resilience.MustNewCircuitBreaker("auth-service", resilience.DefaultCBConfig()),
		},
		{
			name: "payment-service",
			cb: resilience.MustNewCircuitBreaker("payment-service", resilience.CBConfig{
				FailureThreshold: 10, // Payment service gets higher threshold
				SuccessThreshold: 3,
				Timeout:          60 * time.Second,
			}),
		},
		{
			name: "notification-service",
			cb: resilience.MustNewCircuitBreaker("notification-service", resilience.CBConfig{
				FailureThreshold: 3,
				SuccessThreshold: 2,
				Timeout:          15 * time.Second,
			}),
		},
	}

	// Make requests to services
	ctx := context.Background()
	for _, svc := range services {
		err := svc.cb.Execute(ctx, func() error {
			// Simulate service call
			return nil
		})

		if err != nil {
			log.Printf("%s failed: %v", svc.name, err)
		} else {
			fmt.Printf("%s: healthy (%s)\n", svc.name, svc.cb.State())
		}
	}
}

// ExampleCircuitBreaker_metrics demonstrates metrics collection
func ExampleCircuitBreaker_metrics() {
	cb := resilience.MustNewCircuitBreaker("metrics-example", resilience.DefaultCBConfig())
	ctx := context.Background()

	// Execute operations with varying latencies
	for i := 0; i < 20; i++ {
		cb.Execute(ctx, func() error {
			// Simulate varying latencies
			time.Sleep(time.Duration(i) * 5 * time.Millisecond)
			return nil
		})
	}

	// Get metrics snapshot
	snapshot := cb.Stats()
	fmt.Printf("Total Requests: %d\n", snapshot.TotalRequests)
	fmt.Printf("Success Rate: %.2f%%\n",
		float64(snapshot.TotalSuccesses)/float64(snapshot.TotalRequests)*100)
}

// ExampleCircuitBreaker_gracefulDegradation demonstrates fallback patterns
func ExampleCircuitBreaker_gracefulDegradation() {
	cb := resilience.MustNewCircuitBreaker("cache-service", resilience.DefaultCBConfig())

	// Fallback function
	getFallbackData := func(key string) string {
		return "default-value"
	}

	// Primary function that might fail
	getCachedData := func(key string) (string, error) {
		ctx := context.Background()
		var result string

		err := cb.Execute(ctx, func() error {
			// Try to get from cache
			// This might fail if cache is down
			return errors.New("cache unavailable")
		})

		if err != nil {
			if errors.Is(err, resilience.ErrCircuitOpen) {
				// Circuit is open, use fallback immediately
				log.Printf("Circuit open, using fallback")
				return getFallbackData(key), nil
			}
			// Cache error, try fallback
			return getFallbackData(key), nil
		}

		return result, nil
	}

	data, err := getCachedData("user:123")
	if err != nil {
		log.Printf("Failed to get data: %v", err)
	} else {
		fmt.Printf("Got data: %s\n", data)
	}
}

// ExampleCircuitBreaker_bulkhead demonstrates combining circuit breaker with bulkhead pattern
func ExampleCircuitBreaker_bulkhead() {
	config := resilience.CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 3, // Bulkhead: limit concurrent requests in half-open
	}

	cb := resilience.MustNewCircuitBreaker("bulkhead-example", config)
	ctx := context.Background()

	// Simulate many concurrent requests
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := cb.Execute(ctx, func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})

			if err != nil {
				if errors.Is(err, resilience.ErrTooManyRequests) {
					log.Printf("Request %d rejected: too many concurrent requests", id)
				} else {
					log.Printf("Request %d failed: %v", id, err)
				}
			}
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
}

package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func BenchmarkCircuitBreakerClosed(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func() error {
				return nil
			})
		}
	})
}

func BenchmarkCircuitBreakerOpen(b *testing.B) {
	config := CBConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          1 * time.Hour, // Keep it open
	}
	cb := MustNewCircuitBreaker("bench", config)
	ctx := context.Background()

	// Open the circuit
	cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func() error {
				return nil
			})
		}
	})
}

func BenchmarkCircuitBreakerStateCheck(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.State()
		}
	})
}

func BenchmarkCircuitBreakerStats(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())
	ctx := context.Background()

	// Execute some requests to populate stats
	for i := 0; i < 100; i++ {
		cb.Execute(ctx, func() error {
			return nil
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Stats()
		}
	})
}

func BenchmarkCircuitBreakerWithLatency(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}
}

func BenchmarkCircuitBreakerMixedSuccess(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			cb.Execute(ctx, func() error {
				if i%10 == 0 {
					return errors.New("occasional error")
				}
				return nil
			})
		}
	})
}

func BenchmarkMetricsSnapshot(b *testing.B) {
	metrics := NewCBMetrics("bench")

	// Populate with some data
	for i := 0; i < 1000; i++ {
		metrics.RecordRequest(StateClosed, true, time.Duration(i)*time.Microsecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetMetrics()
	}
}

func BenchmarkCircuitBreakerCreation(b *testing.B) {
	config := DefaultCBConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewCircuitBreaker("bench", config)
	}
}

func BenchmarkCircuitBreakerReset(b *testing.B) {
	cb := MustNewCircuitBreaker("bench", DefaultCBConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Reset()
	}
}

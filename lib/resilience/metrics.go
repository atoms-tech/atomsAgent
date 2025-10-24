package resilience

import (
	"sync"
	"time"
)

// CBMetrics tracks circuit breaker metrics
type CBMetrics struct {
	name string

	mu                 sync.RWMutex
	stateTransitions   map[string]uint64 // state -> count
	requestsTotal      uint64
	requestsSuccessful uint64
	requestsFailed     uint64
	requestsRejected   uint64
	latencies          []time.Duration
	maxLatencies       int
}

// NewCBMetrics creates a new metrics tracker
func NewCBMetrics(name string) *CBMetrics {
	return &CBMetrics{
		name:             name,
		stateTransitions: make(map[string]uint64),
		maxLatencies:     100, // Keep last 100 latencies
		latencies:        make([]time.Duration, 0, 100),
	}
}

// RecordRequest records a request execution
func (m *CBMetrics) RecordRequest(state State, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal++
	if success {
		m.requestsSuccessful++
	} else {
		m.requestsFailed++
	}

	// Record latency
	if len(m.latencies) >= m.maxLatencies {
		// Remove oldest latency
		m.latencies = m.latencies[1:]
	}
	m.latencies = append(m.latencies, duration)
}

// RecordRejection records a rejected request
func (m *CBMetrics) RecordRejection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestsRejected++
}

// RecordStateChange records a state transition
func (m *CBMetrics) RecordStateChange(newState State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stateTransitions[newState.String()]++
}

// GetMetrics returns the current metrics
func (m *CBMetrics) GetMetrics() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		Name:               m.name,
		RequestsTotal:      m.requestsTotal,
		RequestsSuccessful: m.requestsSuccessful,
		RequestsFailed:     m.requestsFailed,
		RequestsRejected:   m.requestsRejected,
		StateTransitions:   make(map[string]uint64),
	}

	// Copy state transitions
	for state, count := range m.stateTransitions {
		snapshot.StateTransitions[state] = count
	}

	// Calculate latency statistics
	if len(m.latencies) > 0 {
		snapshot.AvgLatency = calculateAverage(m.latencies)
		snapshot.MinLatency = calculateMin(m.latencies)
		snapshot.MaxLatency = calculateMax(m.latencies)
		snapshot.P50Latency = calculatePercentile(m.latencies, 0.50)
		snapshot.P95Latency = calculatePercentile(m.latencies, 0.95)
		snapshot.P99Latency = calculatePercentile(m.latencies, 0.99)
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	Name               string
	RequestsTotal      uint64
	RequestsSuccessful uint64
	RequestsFailed     uint64
	RequestsRejected   uint64
	StateTransitions   map[string]uint64
	AvgLatency         time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	P50Latency         time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
}

// Helper functions for latency calculations

func calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}

func calculateMin(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	min := latencies[0]
	for _, l := range latencies[1:] {
		if l < min {
			min = l
		}
	}
	return min
}

func calculateMax(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	max := latencies[0]
	for _, l := range latencies[1:] {
		if l > max {
			max = l
		}
	}
	return max
}

func calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Create a sorted copy
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort (good enough for small arrays)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate index
	index := int(float64(len(sorted)-1) * percentile)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// Reset resets all metrics
func (m *CBMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal = 0
	m.requestsSuccessful = 0
	m.requestsFailed = 0
	m.requestsRejected = 0
	m.stateTransitions = make(map[string]uint64)
	m.latencies = make([]time.Duration, 0, m.maxLatencies)
}

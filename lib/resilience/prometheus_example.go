package resilience

// This file provides examples of how to integrate the circuit breaker with Prometheus metrics.
// Note: This is example code. To use it, you'll need to import prometheus packages:
// import (
//     "github.com/prometheus/client_golang/prometheus"
//     "github.com/prometheus/client_golang/prometheus/promauto"
// )

/*
// Example Prometheus metrics that you could define in your application

var (
	// Circuit breaker state gauge
	circuitBreakerStateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of circuit breakers (0=closed, 1=half-open, 2=open)",
		},
		[]string{"name"},
	)

	// Request counters
	circuitBreakerRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests through circuit breaker",
		},
		[]string{"name", "result"}, // result: success, failure, rejected
	)

	// State transition counter
	circuitBreakerStateChanges = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_state_changes_total",
			Help: "Total number of circuit breaker state changes",
		},
		[]string{"name", "from", "to"},
	)

	// Request duration histogram
	circuitBreakerDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "circuit_breaker_request_duration_seconds",
			Help:    "Duration of requests through circuit breaker",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"name", "state"},
	)

	// Current failure count gauge
	circuitBreakerConsecutiveFailures = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_consecutive_failures",
			Help: "Current number of consecutive failures",
		},
		[]string{"name"},
	)
)

// Example: Creating a circuit breaker with Prometheus metrics

func NewMonitoredCircuitBreaker(name string, config CBConfig) *CircuitBreaker {
	// Add state change callback to update Prometheus metrics
	config.OnStateChange = func(cbName string, from State, to State) {
		// Update state gauge
		var stateValue float64
		switch to {
		case StateClosed:
			stateValue = 0
		case StateHalfOpen:
			stateValue = 1
		case StateOpen:
			stateValue = 2
		}
		circuitBreakerStateGauge.WithLabelValues(cbName).Set(stateValue)

		// Record state transition
		circuitBreakerStateChanges.WithLabelValues(
			cbName,
			from.String(),
			to.String(),
		).Inc()
	}

	cb := MustNewCircuitBreaker(name, config)

	// Initialize metrics
	circuitBreakerStateGauge.WithLabelValues(name).Set(0) // Start in closed state

	// Start background metrics collector
	go collectMetrics(cb)

	return cb
}

// collectMetrics periodically collects and exports circuit breaker metrics
func collectMetrics(cb *CircuitBreaker) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := cb.Stats()
		name := cb.Name()

		// Update consecutive failures gauge
		circuitBreakerConsecutiveFailures.WithLabelValues(name).Set(
			float64(stats.ConsecutiveFailures),
		)

		// Get detailed metrics
		snapshot := cb.metrics.GetMetrics()

		// Record request counts (these are cumulative, so we just set the total)
		circuitBreakerRequestsTotal.WithLabelValues(name, "success").Add(
			float64(snapshot.RequestsSuccessful),
		)
		circuitBreakerRequestsTotal.WithLabelValues(name, "failure").Add(
			float64(snapshot.RequestsFailed),
		)
		circuitBreakerRequestsTotal.WithLabelValues(name, "rejected").Add(
			float64(snapshot.RequestsRejected),
		)
	}
}

// Example: Middleware for HTTP handlers

type CircuitBreakerMiddleware struct {
	cb *CircuitBreaker
}

func NewCircuitBreakerMiddleware(name string) *CircuitBreakerMiddleware {
	config := CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 10,
	}

	return &CircuitBreakerMiddleware{
		cb: NewMonitoredCircuitBreaker(name, config),
	}
}

func (m *CircuitBreakerMiddleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		state := m.cb.StateEnum()

		err := m.cb.Execute(r.Context(), func() error {
			// Create a response recorder to capture status code
			rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next(rec, r)

			// Treat 5xx as failures
			if rec.statusCode >= 500 {
				return fmt.Errorf("server error: %d", rec.statusCode)
			}
			return nil
		})

		duration := time.Since(start).Seconds()
		circuitBreakerDuration.WithLabelValues(m.cb.Name(), state.String()).Observe(duration)

		if err != nil {
			if errors.Is(err, ErrCircuitOpen) {
				circuitBreakerRequestsTotal.WithLabelValues(m.cb.Name(), "rejected").Inc()
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			circuitBreakerRequestsTotal.WithLabelValues(m.cb.Name(), "failure").Inc()
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		circuitBreakerRequestsTotal.WithLabelValues(m.cb.Name(), "success").Inc()
	}
}

// statusRecorder is a wrapper around http.ResponseWriter to capture status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Example: Database connection pool with circuit breaker

type DBClient struct {
	cb *CircuitBreaker
	db *sql.DB
}

func NewDBClient(dsn string) (*DBClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	config := CBConfig{
		FailureThreshold:      3,
		SuccessThreshold:      2,
		Timeout:               60 * time.Second,
		MaxConcurrentRequests: 5,
	}

	return &DBClient{
		cb: NewMonitoredCircuitBreaker("database", config),
		db: db,
	}, nil
}

func (c *DBClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows

	err := c.cb.Execute(ctx, func() error {
		var err error
		rows, err = c.db.QueryContext(ctx, query, args...)
		return err
	})

	return rows, err
}

func (c *DBClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result

	err := c.cb.Execute(ctx, func() error {
		var err error
		result, err = c.db.ExecContext(ctx, query, args...)
		return err
	})

	return result, err
}

// Example: Grafana Dashboard Queries

// Useful Prometheus queries for Grafana dashboards:
//
// 1. Circuit breaker state over time:
//    circuit_breaker_state{name="service-name"}
//
// 2. Request success rate:
//    rate(circuit_breaker_requests_total{result="success"}[5m]) /
//    rate(circuit_breaker_requests_total[5m])
//
// 3. State transitions:
//    rate(circuit_breaker_state_changes_total[5m])
//
// 4. Request duration p95:
//    histogram_quantile(0.95,
//      rate(circuit_breaker_request_duration_seconds_bucket[5m]))
//
// 5. Rejection rate:
//    rate(circuit_breaker_requests_total{result="rejected"}[5m])
//
// 6. Current consecutive failures:
//    circuit_breaker_consecutive_failures
//
// 7. Requests by state:
//    sum(rate(circuit_breaker_requests_total[5m])) by (state)

*/

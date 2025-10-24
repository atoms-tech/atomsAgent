package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRegistry wraps a Prometheus registry with application metrics
type MetricsRegistry struct {
	registry *prometheus.Registry

	// HTTP metrics
	httpRequestsTotal          *prometheus.CounterVec
	httpRequestDuration        *prometheus.HistogramVec
	httpRequestsInFlight       prometheus.Gauge
	httpResponseSizeBytes      *prometheus.HistogramVec

	// MCP metrics
	mcpConnectionsActive       prometheus.Gauge
	mcpConnectionErrorsTotal   *prometheus.CounterVec
	mcpOperationsTotal         *prometheus.CounterVec
	mcpOperationDuration       *prometheus.HistogramVec

	// Session metrics
	sessionCount               prometheus.Gauge
	sessionCreatedTotal        prometheus.Counter
	sessionDeletedTotal        prometheus.Counter
	sessionDuration            prometheus.Histogram

	// Database metrics
	databaseQueryDuration      *prometheus.HistogramVec
	databaseQueriesTotal       *prometheus.CounterVec
	databaseConnectionsActive  prometheus.Gauge
	databaseConnectionErrors   prometheus.Counter

	// Cache metrics
	cacheHitsTotal             *prometheus.CounterVec
	cacheMissesTotal           *prometheus.CounterVec
	cacheOperationDuration     *prometheus.HistogramVec
	cacheSize                  *prometheus.GaugeVec

	// System metrics
	goroutinesCount            prometheus.Gauge
	memoryAllocatedBytes       prometheus.Gauge
	memoryHeapBytes            prometheus.Gauge

	mu                         sync.RWMutex
	activeSessions             map[string]time.Time
}

// NewMetricsRegistry creates and initializes a new Prometheus metrics registry
func NewMetricsRegistry() *MetricsRegistry {
	registry := prometheus.NewRegistry()

	// Create metrics registry
	mr := &MetricsRegistry{
		registry:       registry,
		activeSessions: make(map[string]time.Time),
	}

	// Initialize HTTP metrics
	mr.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by method, path, and status code",
		},
		[]string{"method", "path", "status"},
	)

	mr.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds by method and path",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	mr.httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	mr.httpResponseSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Initialize MCP metrics
	mr.mcpConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mcp_connections_active",
			Help: "Current number of active MCP connections",
		},
	)

	mr.mcpConnectionErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_connection_errors_total",
			Help: "Total number of MCP connection failures by server name and error type",
		},
		[]string{"mcp_name", "error_type"},
	)

	mr.mcpOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_operations_total",
			Help: "Total number of MCP operations by server name, operation type, and status",
		},
		[]string{"mcp_name", "operation", "status"},
	)

	mr.mcpOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_operation_duration_seconds",
			Help:    "MCP operation duration in seconds",
			Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		},
		[]string{"mcp_name", "operation"},
	)

	// Initialize Session metrics
	mr.sessionCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "session_count",
			Help: "Current number of active sessions",
		},
	)

	mr.sessionCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "session_created_total",
			Help: "Total number of sessions created",
		},
	)

	mr.sessionDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "session_deleted_total",
			Help: "Total number of sessions deleted",
		},
	)

	mr.sessionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "session_duration_seconds",
			Help:    "Session duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
	)

	// Initialize Database metrics
	mr.databaseQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query latency in seconds by query type",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"query_type"},
	)

	mr.databaseQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries by type and status",
		},
		[]string{"query_type", "status"},
	)

	mr.databaseConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Current number of active database connections",
		},
	)

	mr.databaseConnectionErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "database_connection_errors_total",
			Help: "Total number of database connection errors",
		},
	)

	// Initialize Cache metrics
	mr.cacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits by cache name",
		},
		[]string{"cache_name"},
	)

	mr.cacheMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses by cache name",
		},
		[]string{"cache_name"},
	)

	mr.cacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .05},
		},
		[]string{"cache_name", "operation"},
	)

	mr.cacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_size_items",
			Help: "Current number of items in cache",
		},
		[]string{"cache_name"},
	)

	// Initialize System metrics
	mr.goroutinesCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_count",
			Help: "Current number of goroutines",
		},
	)

	mr.memoryAllocatedBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_allocated_bytes",
			Help: "Current allocated memory in bytes",
		},
	)

	mr.memoryHeapBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_heap_bytes",
			Help: "Current heap memory in bytes",
		},
	)

	// Register all metrics
	mr.registerMetrics()

	return mr
}

// registerMetrics registers all metrics with the Prometheus registry
func (mr *MetricsRegistry) registerMetrics() {
	// HTTP metrics
	mr.registry.MustRegister(mr.httpRequestsTotal)
	mr.registry.MustRegister(mr.httpRequestDuration)
	mr.registry.MustRegister(mr.httpRequestsInFlight)
	mr.registry.MustRegister(mr.httpResponseSizeBytes)

	// MCP metrics
	mr.registry.MustRegister(mr.mcpConnectionsActive)
	mr.registry.MustRegister(mr.mcpConnectionErrorsTotal)
	mr.registry.MustRegister(mr.mcpOperationsTotal)
	mr.registry.MustRegister(mr.mcpOperationDuration)

	// Session metrics
	mr.registry.MustRegister(mr.sessionCount)
	mr.registry.MustRegister(mr.sessionCreatedTotal)
	mr.registry.MustRegister(mr.sessionDeletedTotal)
	mr.registry.MustRegister(mr.sessionDuration)

	// Database metrics
	mr.registry.MustRegister(mr.databaseQueryDuration)
	mr.registry.MustRegister(mr.databaseQueriesTotal)
	mr.registry.MustRegister(mr.databaseConnectionsActive)
	mr.registry.MustRegister(mr.databaseConnectionErrors)

	// Cache metrics
	mr.registry.MustRegister(mr.cacheHitsTotal)
	mr.registry.MustRegister(mr.cacheMissesTotal)
	mr.registry.MustRegister(mr.cacheOperationDuration)
	mr.registry.MustRegister(mr.cacheSize)

	// System metrics
	mr.registry.MustRegister(mr.goroutinesCount)
	mr.registry.MustRegister(mr.memoryAllocatedBytes)
	mr.registry.MustRegister(mr.memoryHeapBytes)
}

// GetRegistry returns the underlying Prometheus registry
func (mr *MetricsRegistry) GetRegistry() *prometheus.Registry {
	return mr.registry
}

// HTTPHandler returns the Prometheus HTTP handler for the /metrics endpoint
func (mr *MetricsRegistry) HTTPHandler() http.Handler {
	return promhttp.HandlerFor(mr.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          mr.registry,
	})
}

// JSONHandler returns a handler that exports metrics in JSON format
func (mr *MetricsRegistry) JSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics, err := mr.registry.Gather()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert metrics to JSON-friendly format
		jsonMetrics := make(map[string]interface{})
		jsonMetrics["timestamp"] = time.Now().Unix()
		jsonMetrics["metrics"] = make([]map[string]interface{}, 0)

		for _, mf := range metrics {
			for _, m := range mf.GetMetric() {
				metric := map[string]interface{}{
					"name":   mf.GetName(),
					"help":   mf.GetHelp(),
					"type":   mf.GetType().String(),
					"labels": make(map[string]string),
				}

				// Add labels
				for _, label := range m.GetLabel() {
					metric["labels"].(map[string]string)[label.GetName()] = label.GetValue()
				}

				// Add value based on metric type
				switch mf.GetType() {
				case 0: // COUNTER
					if m.Counter != nil {
						metric["value"] = m.Counter.GetValue()
					}
				case 1: // GAUGE
					if m.Gauge != nil {
						metric["value"] = m.Gauge.GetValue()
					}
				case 4: // HISTOGRAM
					if m.Histogram != nil {
						metric["count"] = m.Histogram.GetSampleCount()
						metric["sum"] = m.Histogram.GetSampleSum()
					}
				}

				jsonMetrics["metrics"] = append(jsonMetrics["metrics"].([]map[string]interface{}), metric)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonMetrics)
	}
}

// ===== HTTP Metrics Helper Functions =====

// HTTPMiddleware wraps an HTTP handler to record request metrics
func (mr *MetricsRegistry) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		mr.httpRequestsInFlight.Inc()
		defer mr.httpRequestsInFlight.Dec()

		// Create response writer wrapper to capture status code and size
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapper.statusCode)
		path := mr.normalizePath(r)

		mr.httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		mr.httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		mr.httpResponseSizeBytes.WithLabelValues(r.Method, path).Observe(float64(wrapper.bytesWritten))
	})
}

// normalizePath extracts the route pattern from the request
func (mr *MetricsRegistry) normalizePath(r *http.Request) string {
	// Try to get the route pattern from chi router context
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		pattern := rctx.RoutePattern()
		if pattern != "" {
			return pattern
		}
	}

	// Fallback to raw path, but limit length and sanitize
	path := r.URL.Path
	if len(path) > 100 {
		path = path[:100]
	}

	// Replace IDs and UUIDs with placeholders to reduce cardinality
	path = sanitizePath(path)

	return path
}

// sanitizePath replaces UUIDs and numeric IDs with placeholders
func sanitizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Replace UUIDs (simple pattern matching)
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			parts[i] = "{id}"
		}
		// Replace numeric IDs
		if _, err := strconv.Atoi(part); err == nil && len(part) > 0 {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// ===== MCP Metrics Helper Functions =====

// RecordMCPConnection records an MCP connection attempt
func (mr *MetricsRegistry) RecordMCPConnection(mcpName string, success bool) {
	if success {
		mr.mcpConnectionsActive.Inc()
		mr.mcpOperationsTotal.WithLabelValues(mcpName, "connect", "success").Inc()
	} else {
		mr.mcpConnectionErrorsTotal.WithLabelValues(mcpName, "connection_failed").Inc()
		mr.mcpOperationsTotal.WithLabelValues(mcpName, "connect", "failure").Inc()
	}
}

// RecordMCPDisconnection records an MCP disconnection
func (mr *MetricsRegistry) RecordMCPDisconnection(mcpName string) {
	mr.mcpConnectionsActive.Dec()
	mr.mcpOperationsTotal.WithLabelValues(mcpName, "disconnect", "success").Inc()
}

// RecordMCPOperation records an MCP operation with duration
func (mr *MetricsRegistry) RecordMCPOperation(mcpName, operation string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	mr.mcpOperationsTotal.WithLabelValues(mcpName, operation, status).Inc()
	mr.mcpOperationDuration.WithLabelValues(mcpName, operation).Observe(duration.Seconds())
}

// RecordMCPError records an MCP error
func (mr *MetricsRegistry) RecordMCPError(mcpName, errorType string) {
	mr.mcpConnectionErrorsTotal.WithLabelValues(mcpName, errorType).Inc()
}

// MCPOperationTimer returns a function to record MCP operation duration
func (mr *MetricsRegistry) MCPOperationTimer(mcpName, operation string) func(success bool) {
	start := time.Now()
	return func(success bool) {
		duration := time.Since(start)
		mr.RecordMCPOperation(mcpName, operation, duration, success)
	}
}

// ===== Session Metrics Helper Functions =====

// RecordSessionCreated records a new session creation
func (mr *MetricsRegistry) RecordSessionCreated(sessionID string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.sessionCreatedTotal.Inc()
	mr.sessionCount.Inc()
	mr.activeSessions[sessionID] = time.Now()
}

// RecordSessionDeleted records a session deletion
func (mr *MetricsRegistry) RecordSessionDeleted(sessionID string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.sessionDeletedTotal.Inc()
	mr.sessionCount.Dec()

	if startTime, exists := mr.activeSessions[sessionID]; exists {
		duration := time.Since(startTime)
		mr.sessionDuration.Observe(duration.Seconds())
		delete(mr.activeSessions, sessionID)
	}
}

// GetActiveSessionCount returns the current number of active sessions
func (mr *MetricsRegistry) GetActiveSessionCount() int {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return len(mr.activeSessions)
}

// ===== Database Metrics Helper Functions =====

// RecordDBQuery records a database query execution
func (mr *MetricsRegistry) RecordDBQuery(queryType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	mr.databaseQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
	mr.databaseQueriesTotal.WithLabelValues(queryType, status).Inc()
}

// DBQueryTimer returns a function to record database query duration
func (mr *MetricsRegistry) DBQueryTimer(queryType string) func(error) {
	start := time.Now()
	return func(err error) {
		duration := time.Since(start)
		mr.RecordDBQuery(queryType, duration, err)
	}
}

// RecordDBConnection records database connection metrics
func (mr *MetricsRegistry) RecordDBConnection(active int, err error) {
	mr.databaseConnectionsActive.Set(float64(active))
	if err != nil {
		mr.databaseConnectionErrors.Inc()
	}
}

// ===== Cache Metrics Helper Functions =====

// RecordCacheHit records a cache hit
func (mr *MetricsRegistry) RecordCacheHit(cacheName string) {
	mr.cacheHitsTotal.WithLabelValues(cacheName).Inc()
}

// RecordCacheMiss records a cache miss
func (mr *MetricsRegistry) RecordCacheMiss(cacheName string) {
	mr.cacheMissesTotal.WithLabelValues(cacheName).Inc()
}

// RecordCacheOperation records a cache operation with duration
func (mr *MetricsRegistry) RecordCacheOperation(cacheName, operation string, duration time.Duration) {
	mr.cacheOperationDuration.WithLabelValues(cacheName, operation).Observe(duration.Seconds())
}

// UpdateCacheSize updates the cache size metric
func (mr *MetricsRegistry) UpdateCacheSize(cacheName string, size int) {
	mr.cacheSize.WithLabelValues(cacheName).Set(float64(size))
}

// CacheOperationTimer returns a function to record cache operation duration
func (mr *MetricsRegistry) CacheOperationTimer(cacheName, operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		mr.RecordCacheOperation(cacheName, operation, duration)
	}
}

// ===== System Metrics Helper Functions =====

// UpdateSystemMetrics updates system-level metrics
func (mr *MetricsRegistry) UpdateSystemMetrics(goroutines int, allocatedBytes, heapBytes uint64) {
	mr.goroutinesCount.Set(float64(goroutines))
	mr.memoryAllocatedBytes.Set(float64(allocatedBytes))
	mr.memoryHeapBytes.Set(float64(heapBytes))
}

// ===== Context-based helpers =====

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	metricsContextKey contextKey = "metrics_registry"
)

// WithMetrics adds the metrics registry to the context
func WithMetrics(ctx context.Context, mr *MetricsRegistry) context.Context {
	return context.WithValue(ctx, metricsContextKey, mr)
}

// FromContext retrieves the metrics registry from the context
func FromContext(ctx context.Context) *MetricsRegistry {
	if mr, ok := ctx.Value(metricsContextKey).(*MetricsRegistry); ok {
		return mr
	}
	return nil
}

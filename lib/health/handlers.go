package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Handler provides HTTP handlers for health check endpoints
type Handler struct {
	checker *HealthChecker
}

// NewHandler creates a new health check HTTP handler
func NewHandler(checker *HealthChecker) *Handler {
	return &Handler{
		checker: checker,
	}
}

// HealthResponse is the detailed health check response
type HealthResponse struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckStatus `json:"checks"`
}

// Health handles GET /health - returns detailed health status as JSON
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Perform health check
	status := h.checker.Check(ctx)

	// Prepare response
	response := HealthResponse{
		Status:    status.Overall,
		Timestamp: status.Timestamp,
		Checks:    status.Checks,
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status.Overall == StatusDown {
		statusCode = http.StatusServiceUnavailable
	} else if status.Overall == StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(statusCode)

	// Write response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Ready handles GET /ready - Kubernetes readiness probe
// Returns 200 if ready, 503 if not ready
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if ready
	ready := h.checker.Ready(ctx)

	// Set headers
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}
}

// Live handles GET /live - Kubernetes liveness probe
// Returns 200 if the application is alive
// This is a simpler check that just verifies the process is running
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	// For liveness, we just need to respond
	// The fact that we can handle this request means we're alive
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// RegisterRoutes registers health check routes on a ServeMux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/ready", h.Ready)
	mux.HandleFunc("/live", h.Live)
}

// WithTimeout wraps a handler with a timeout
func WithTimeout(timeout time.Duration, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// Create a channel to signal completion
		done := make(chan struct{})

		// Run handler in goroutine
		go func() {
			handler(w, r.WithContext(ctx))
			close(done)
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Handler completed successfully
			return
		case <-ctx.Done():
			// Timeout occurred
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
			return
		}
	}
}

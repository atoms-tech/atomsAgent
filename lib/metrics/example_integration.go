package metrics

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// ExampleServer demonstrates how to integrate metrics into your application
type ExampleServer struct {
	router  *chi.Mux
	metrics *MetricsRegistry
	logger  *slog.Logger
	db      *sql.DB
	cache   map[string]interface{}
}

// NewExampleServer creates a new example server with metrics
func NewExampleServer(logger *slog.Logger) *ExampleServer {
	// Initialize metrics registry
	metrics := NewMetricsRegistry()

	// Create the server
	s := &ExampleServer{
		router:  chi.NewRouter(),
		metrics: metrics,
		logger:  logger,
		cache:   make(map[string]interface{}),
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	// Start background metrics collector
	go s.collectSystemMetrics()

	return s
}

// setupMiddleware configures all middleware including metrics
func (s *ExampleServer) setupMiddleware() {
	// CORS middleware
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Metrics middleware - IMPORTANT: Add this early in the chain
	s.router.Use(s.metrics.HTTPMiddleware)

	// Add metrics to context for easy access in handlers
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithMetrics(r.Context(), s.metrics)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Other middleware (logging, recovery, etc.)
	s.router.Use(s.loggingMiddleware)
}

// setupRoutes configures all HTTP routes including metrics endpoints
func (s *ExampleServer) setupRoutes() {
	// API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/users", s.handleGetUsers)
		r.Get("/users/{id}", s.handleGetUser)
		r.Post("/users", s.handleCreateUser)
		r.Delete("/users/{id}", s.handleDeleteUser)

		// Session endpoints
		r.Post("/sessions", s.handleCreateSession)
		r.Delete("/sessions/{id}", s.handleDeleteSession)

		// MCP operations
		r.Post("/mcp/connect", s.handleMCPConnect)
		r.Post("/mcp/query", s.handleMCPQuery)
		r.Post("/mcp/disconnect", s.handleMCPDisconnect)
	})

	// Health check (excluded from metrics)
	s.router.Get("/health", s.handleHealth)

	// Metrics endpoints
	s.router.Handle("/metrics", s.metrics.HTTPHandler())
	s.router.Get("/metrics/json", s.metrics.JSONHandler())
}

// loggingMiddleware logs all requests
func (s *ExampleServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

// collectSystemMetrics periodically collects and updates system metrics
func (s *ExampleServer) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		s.metrics.UpdateSystemMetrics(
			runtime.NumGoroutine(),
			m.Alloc,
			m.HeapAlloc,
		)
	}
}

// Handler examples demonstrating metrics integration

func (s *ExampleServer) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	// Example: Database query with metrics
	done := s.metrics.DBQueryTimer("SELECT")
	defer done(nil)

	// Simulate database query
	time.Sleep(10 * time.Millisecond)

	// Example: Cache lookup with metrics
	cacheName := "users-list"
	if _, exists := s.cache[cacheName]; exists {
		s.metrics.RecordCacheHit(cacheName)
	} else {
		s.metrics.RecordCacheMiss(cacheName)
		// Would fetch from DB and cache here
		s.cache[cacheName] = []string{"user1", "user2"}
		s.metrics.UpdateCacheSize(cacheName, len(s.cache))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"users": []}`))
}

func (s *ExampleServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Example: Timed database query
	done := s.metrics.DBQueryTimer("SELECT")
	// Simulate query
	time.Sleep(5 * time.Millisecond)
	done(nil)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"user": {"id": "` + userID + `"}}`))
}

func (s *ExampleServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Example: Database operation with error handling
	done := s.metrics.DBQueryTimer("INSERT")
	err := s.createUserInDB()
	done(err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"user": {"id": "new-user"}}`))
}

func (s *ExampleServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Example: Database delete operation
	done := s.metrics.DBQueryTimer("DELETE")
	// Simulate delete
	time.Sleep(3 * time.Millisecond)
	done(nil)

	// Clear from cache if exists
	delete(s.cache, "user:"+userID)
	s.metrics.UpdateCacheSize("users", len(s.cache))

	w.WriteHeader(http.StatusNoContent)
}

func (s *ExampleServer) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	// Example: Session creation
	sessionID := "session-" + time.Now().Format("20060102150405")
	s.metrics.RecordSessionCreated(sessionID)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"session_id": "` + sessionID + `"}`))
}

func (s *ExampleServer) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	// Example: Session deletion
	sessionID := chi.URLParam(r, "id")
	s.metrics.RecordSessionDeleted(sessionID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *ExampleServer) handleMCPConnect(w http.ResponseWriter, r *http.Request) {
	// Example: MCP connection
	mcpName := "example-mcp-server"

	// Simulate connection attempt
	err := s.connectToMCP(mcpName)
	success := err == nil

	s.metrics.RecordMCPConnection(mcpName, success)

	if !success {
		http.Error(w, "connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "connected"}`))
}

func (s *ExampleServer) handleMCPQuery(w http.ResponseWriter, r *http.Request) {
	// Example: MCP operation with timer
	mcpName := "example-mcp-server"
	done := s.metrics.MCPOperationTimer(mcpName, "query")

	// Simulate query
	time.Sleep(20 * time.Millisecond)
	err := s.queryMCP(mcpName)

	done(err == nil)

	if err != nil {
		s.metrics.RecordMCPError(mcpName, "query_error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"result": "success"}`))
}

func (s *ExampleServer) handleMCPDisconnect(w http.ResponseWriter, r *http.Request) {
	// Example: MCP disconnection
	mcpName := "example-mcp-server"
	s.metrics.RecordMCPDisconnection(mcpName)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "disconnected"}`))
}

func (s *ExampleServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}

// Simulated helper functions

func (s *ExampleServer) createUserInDB() error {
	// Simulate database operation
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (s *ExampleServer) connectToMCP(name string) error {
	// Simulate connection
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (s *ExampleServer) queryMCP(name string) error {
	// Simulate query
	time.Sleep(20 * time.Millisecond)
	return nil
}

// ServeHTTP implements http.Handler
func (s *ExampleServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ExampleUsage demonstrates how to use the metrics package
func ExampleUsage() {
	logger := slog.Default()

	// Create server with metrics
	server := NewExampleServer(logger)

	// Start HTTP server
	http.ListenAndServe(":8080", server)
}

// ExampleManualMetrics shows how to manually record metrics without middleware
func ExampleManualMetrics() {
	metrics := NewMetricsRegistry()

	// Record MCP connection
	metrics.RecordMCPConnection("my-mcp-server", true)

	// Use operation timer
	done := metrics.MCPOperationTimer("my-mcp-server", "tool_call")
	// ... do work ...
	time.Sleep(100 * time.Millisecond)
	done(true)

	// Record database query
	queryDone := metrics.DBQueryTimer("SELECT")
	// ... execute query ...
	queryDone(nil)

	// Record cache operations
	metrics.RecordCacheHit("user-cache")
	metrics.RecordCacheMiss("user-cache")

	// Record session lifecycle
	sessionID := "session-123"
	metrics.RecordSessionCreated(sessionID)
	// ... session is active ...
	time.Sleep(5 * time.Second)
	metrics.RecordSessionDeleted(sessionID)

	// Update system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.UpdateSystemMetrics(runtime.NumGoroutine(), m.Alloc, m.HeapAlloc)
}

// ExampleContextUsage shows how to use metrics from context
func ExampleContextUsage() {
	metrics := NewMetricsRegistry()
	ctx := context.Background()

	// Add metrics to context
	ctx = WithMetrics(ctx, metrics)

	// Later in your code, retrieve from context
	if m := FromContext(ctx); m != nil {
		m.RecordCacheHit("my-cache")
	}
}

// ExampleWithCircuitBreaker shows integration with circuit breaker
func ExampleWithCircuitBreaker() {
	metrics := NewMetricsRegistry()
	mcpName := "external-api"

	// Simulated circuit breaker logic
	maxFailures := 5
	failures := 0

	executeWithCircuitBreaker := func() error {
		done := metrics.MCPOperationTimer(mcpName, "api_call")
		defer func() {
			if failures < maxFailures {
				done(true)
			} else {
				done(false)
				metrics.RecordMCPError(mcpName, "circuit_open")
			}
		}()

		// Simulate API call
		if failures >= maxFailures {
			return errors.New("circuit breaker open")
		}

		// Simulate occasional failure
		if time.Now().Unix()%10 == 0 {
			failures++
			return errors.New("api error")
		}

		failures = 0
		return nil
	}

	_ = executeWithCircuitBreaker
}

package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsRegistry(t *testing.T) {
	mr := NewMetricsRegistry()
	require.NotNil(t, mr)
	require.NotNil(t, mr.registry)
	require.NotNil(t, mr.activeSessions)
}

func TestHTTPMiddleware(t *testing.T) {
	mr := NewMetricsRegistry()

	// Create a test handler
	handler := mr.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))

	// Create a test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	// Execute the request
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())

	// Verify metrics were recorded (we can't easily check values without exposing internals)
	// But we can verify the handler doesn't panic
}

func TestHTTPMiddlewareWithChiRouter(t *testing.T) {
	mr := NewMetricsRegistry()

	// Create a chi router
	r := chi.NewRouter()
	r.Use(mr.HTTPMiddleware)
	r.Get("/api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user data"))
	})

	// Create a test request
	req := httptest.NewRequest("GET", "/api/users/123", nil)
	w := httptest.NewRecorder()

	// Execute the request
	r.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)

	// Test Write
	data := []byte("test data")
	n, err := rw.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.bytesWritten)
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UUID replacement",
			input:    "/api/users/550e8400-e29b-41d4-a716-446655440000/profile",
			expected: "/api/users/{id}/profile",
		},
		{
			name:     "Numeric ID replacement",
			input:    "/api/users/12345/posts",
			expected: "/api/users/{id}/posts",
		},
		{
			name:     "No replacement needed",
			input:    "/api/users/profile",
			expected: "/api/users/profile",
		},
		{
			name:     "Multiple IDs",
			input:    "/api/users/123/posts/456",
			expected: "/api/users/{id}/posts/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test successful connection
	mr.RecordMCPConnection("test-server", true)

	// Test failed connection
	mr.RecordMCPConnection("test-server", false)

	// Test disconnection
	mr.RecordMCPDisconnection("test-server")

	// Test operation
	mr.RecordMCPOperation("test-server", "query", 100*time.Millisecond, true)
	mr.RecordMCPOperation("test-server", "query", 200*time.Millisecond, false)

	// Test error recording
	mr.RecordMCPError("test-server", "timeout")
}

func TestMCPOperationTimer(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test successful operation
	done := mr.MCPOperationTimer("test-server", "query")
	time.Sleep(10 * time.Millisecond)
	done(true)

	// Test failed operation
	done = mr.MCPOperationTimer("test-server", "query")
	time.Sleep(5 * time.Millisecond)
	done(false)
}

func TestSessionMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test session creation
	mr.RecordSessionCreated("session-1")
	assert.Equal(t, 1, mr.GetActiveSessionCount())

	mr.RecordSessionCreated("session-2")
	assert.Equal(t, 2, mr.GetActiveSessionCount())

	// Test session deletion
	time.Sleep(10 * time.Millisecond)
	mr.RecordSessionDeleted("session-1")
	assert.Equal(t, 1, mr.GetActiveSessionCount())

	mr.RecordSessionDeleted("session-2")
	assert.Equal(t, 0, mr.GetActiveSessionCount())

	// Test deleting non-existent session (should not panic)
	mr.RecordSessionDeleted("non-existent")
	assert.Equal(t, 0, mr.GetActiveSessionCount())
}

func TestDatabaseMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test successful query
	mr.RecordDBQuery("SELECT", 50*time.Millisecond, nil)

	// Test failed query
	mr.RecordDBQuery("INSERT", 100*time.Millisecond, errors.New("constraint violation"))

	// Test connection metrics
	mr.RecordDBConnection(5, nil)
	mr.RecordDBConnection(5, errors.New("connection failed"))
}

func TestDBQueryTimer(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test successful query
	done := mr.DBQueryTimer("SELECT")
	time.Sleep(10 * time.Millisecond)
	done(nil)

	// Test failed query
	done = mr.DBQueryTimer("UPDATE")
	time.Sleep(5 * time.Millisecond)
	done(errors.New("update failed"))
}

func TestCacheMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test cache hit/miss
	mr.RecordCacheHit("user-cache")
	mr.RecordCacheMiss("user-cache")

	// Test cache operations
	mr.RecordCacheOperation("user-cache", "get", 1*time.Millisecond)
	mr.RecordCacheOperation("user-cache", "set", 2*time.Millisecond)

	// Test cache size
	mr.UpdateCacheSize("user-cache", 100)
	mr.UpdateCacheSize("user-cache", 150)
}

func TestCacheOperationTimer(t *testing.T) {
	mr := NewMetricsRegistry()

	done := mr.CacheOperationTimer("user-cache", "get")
	time.Sleep(5 * time.Millisecond)
	done()
}

func TestSystemMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	mr.UpdateSystemMetrics(100, 1024*1024, 2048*1024)
	mr.UpdateSystemMetrics(120, 1024*1024*2, 2048*1024*2)
}

func TestHTTPHandler(t *testing.T) {
	mr := NewMetricsRegistry()

	// Record some test metrics
	mr.RecordSessionCreated("test-session")
	mr.RecordCacheHit("test-cache")

	// Get the HTTP handler
	handler := mr.HTTPHandler()
	require.NotNil(t, handler)

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Execute the request
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify some metrics are present in the output
	body := w.Body.String()
	assert.Contains(t, body, "session_count")
	assert.Contains(t, body, "cache_hits_total")
}

func TestJSONHandler(t *testing.T) {
	mr := NewMetricsRegistry()

	// Record some test metrics
	mr.RecordSessionCreated("test-session")
	mr.RecordCacheHit("test-cache")

	// Get the JSON handler
	handler := mr.JSONHandler()
	require.NotNil(t, handler)

	// Create a test request
	req := httptest.NewRequest("GET", "/metrics/json", nil)
	w := httptest.NewRecorder()

	// Execute the request
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Verify JSON structure
	body := w.Body.String()
	assert.Contains(t, body, "timestamp")
	assert.Contains(t, body, "metrics")
}

func TestContextHelpers(t *testing.T) {
	mr := NewMetricsRegistry()
	ctx := context.Background()

	// Test adding metrics to context
	ctx = WithMetrics(ctx, mr)

	// Test retrieving metrics from context
	retrieved := FromContext(ctx)
	assert.NotNil(t, retrieved)
	assert.Equal(t, mr, retrieved)

	// Test retrieving from empty context
	emptyCtx := context.Background()
	retrieved = FromContext(emptyCtx)
	assert.Nil(t, retrieved)
}

func TestHTTPHandlerIntegration(t *testing.T) {
	mr := NewMetricsRegistry()

	// Create a simple router with metrics
	r := chi.NewRouter()
	r.Use(mr.HTTPMiddleware)

	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": []}`))
	})

	r.Get("/api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user": {"id": "123"}}`))
	})

	r.Post("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"user": {"id": "456"}}`))
	})

	r.Get("/metrics", mr.HTTPHandler().ServeHTTP)
	r.Get("/metrics/json", mr.JSONHandler())

	// Make several test requests
	testCases := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/api/users", http.StatusOK},
		{"GET", "/api/users/123", http.StatusOK},
		{"POST", "/api/users", http.StatusCreated},
		{"GET", "/api/users/456", http.StatusOK},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, tc.status, w.Code)
	}

	// Check metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Verify metrics are present
	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds")
}

func BenchmarkHTTPMiddleware(b *testing.B) {
	mr := NewMetricsRegistry()

	handler := mr.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRecordMCPOperation(b *testing.B) {
	mr := NewMetricsRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mr.RecordMCPOperation("test-server", "query", 10*time.Millisecond, true)
	}
}

func BenchmarkRecordCacheHit(b *testing.B) {
	mr := NewMetricsRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mr.RecordCacheHit("test-cache")
	}
}

func TestMetricsEndpointFormat(t *testing.T) {
	mr := NewMetricsRegistry()

	// Record various metrics
	mr.RecordSessionCreated("session-1")
	mr.RecordCacheHit("user-cache")
	mr.RecordCacheMiss("user-cache")
	mr.RecordMCPConnection("test-mcp", true)
	mr.RecordDBQuery("SELECT", 50*time.Millisecond, nil)

	// Get metrics in Prometheus format
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	mr.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()

	// Verify Prometheus format
	assert.Contains(t, body, "# HELP session_count")
	assert.Contains(t, body, "# TYPE session_count gauge")
	assert.Contains(t, body, "session_count 1")

	assert.Contains(t, body, "# HELP cache_hits_total")
	assert.Contains(t, body, "# TYPE cache_hits_total counter")
	assert.Contains(t, body, `cache_hits_total{cache_name="user-cache"} 1`)

	assert.Contains(t, body, "# HELP mcp_connections_active")
	assert.Contains(t, body, "# TYPE mcp_connections_active gauge")
}

func TestConcurrentMetrics(t *testing.T) {
	mr := NewMetricsRegistry()

	// Test concurrent session operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			sessionID := "session-" + string(rune(id))
			mr.RecordSessionCreated(sessionID)
			time.Sleep(10 * time.Millisecond)
			mr.RecordSessionDeleted(sessionID)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, 0, mr.GetActiveSessionCount())
}

func TestMetricsExport(t *testing.T) {
	mr := NewMetricsRegistry()

	// Add some test data
	mr.RecordSessionCreated("test-1")
	mr.RecordCacheHit("cache-1")
	mr.RecordMCPConnection("mcp-1", true)

	// Test Prometheus export
	promReq := httptest.NewRequest("GET", "/metrics", nil)
	promW := httptest.NewRecorder()
	mr.HTTPHandler().ServeHTTP(promW, promReq)

	promBody, err := io.ReadAll(promW.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, promBody)
	assert.Contains(t, string(promBody), "session_count")

	// Test JSON export
	jsonReq := httptest.NewRequest("GET", "/metrics/json", nil)
	jsonW := httptest.NewRecorder()
	mr.JSONHandler().ServeHTTP(jsonW, jsonReq)

	jsonBody, err := io.ReadAll(jsonW.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBody)
	assert.True(t, strings.HasPrefix(jsonW.Header().Get("Content-Type"), "application/json"))
}

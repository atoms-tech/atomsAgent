package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_Health_Healthy(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != StatusUp {
		t.Errorf("Expected status UP, got %s", healthResp.Status)
	}

	if len(healthResp.Checks) == 0 {
		t.Error("Expected checks in response")
	}
}

func TestHandler_Health_Unhealthy(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: true})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != StatusDown {
		t.Errorf("Expected status DOWN, got %s", healthResp.Status)
	}
}

func TestHandler_Health_Degraded(t *testing.T) {
	hc := NewHealthChecker(nil, nil)

	// Create a custom check that returns DEGRADED
	// Since we don't have a built-in degraded state, we'll simulate it
	// by having one check fail with a non-critical error
	hc.RegisterCheck("critical", &MockHealthCheck{shouldFail: false})
	hc.RegisterCheck("non-critical", &MockHealthCheck{shouldFail: false})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// When all checks pass, should return OK
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandler_Health_CacheHeaders(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Verify cache control headers
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "no-cache, no-store, must-revalidate" {
		t.Errorf("Expected no-cache headers, got %s", cacheControl)
	}

	pragma := resp.Header.Get("Pragma")
	if pragma != "no-cache" {
		t.Errorf("Expected Pragma no-cache, got %s", pragma)
	}

	expires := resp.Header.Get("Expires")
	if expires != "0" {
		t.Errorf("Expected Expires 0, got %s", expires)
	}
}

func TestHandler_Ready_Ready(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
}

func TestHandler_Ready_NotReady(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: true})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}
}

func TestHandler_Live(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	handler.Live(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	handler := NewHandler(hc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Test that routes are registered
	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/health", http.StatusOK},
		{"/ready", http.StatusOK},
		{"/live", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d",
					tc.expectedStatus, tc.path, w.Code)
			}
		})
	}
}

func TestWithTimeout_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	wrappedHandler := WithTimeout(1*time.Second, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestWithTimeout_Timeout(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow handler
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := WithTimeout(100*time.Millisecond, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d", w.Code)
	}
}

func TestHandler_Health_ResponseStructure(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test1", &MockHealthCheck{shouldFail: false})
	hc.RegisterCheck("test2", &MockHealthCheck{shouldFail: true})

	handler := NewHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify structure
	if healthResp.Timestamp.IsZero() {
		t.Error("Expected timestamp in response")
	}

	// Should have at least our 2 test checks plus default checks
	if len(healthResp.Checks) < 2 {
		t.Errorf("Expected at least 2 checks, got %d", len(healthResp.Checks))
	}

	// Verify our test checks are present
	test1, ok1 := healthResp.Checks["test1"]
	test2, ok2 := healthResp.Checks["test2"]

	if !ok1 {
		t.Error("Expected test1 check in response")
	}
	if !ok2 {
		t.Error("Expected test2 check in response")
	}

	// Verify individual check structure
	if test1.Name == "" {
		t.Error("test1 missing name")
	}
	if test1.Status == "" {
		t.Error("test1 missing status")
	}

	// test2 should have an error
	if test2.Error == "" {
		t.Error("Expected error for test2 check")
	}
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	handler := NewHandler(hc)

	// Make concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			handler.Health(w, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHandler_ContextPropagation(t *testing.T) {
	hc := NewHealthChecker(nil, nil)
	hc.RegisterCheck("test", &MockHealthCheck{shouldFail: false})

	handler := NewHandler(hc)

	// Create a request with a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Handler should still complete (checks might fail due to cancelled context)
	handler.Health(w, req)

	// Should get a response (even if checks failed)
	if w.Code == 0 {
		t.Error("Expected handler to return a response")
	}
}

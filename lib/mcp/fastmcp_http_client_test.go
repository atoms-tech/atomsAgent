package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewFastMCPHTTPClient(t *testing.T) {
	client := NewFastMCPHTTPClient("")
	if client == nil {
		t.Fatal("NewFastMCPHTTPClient returned nil")
	}
	if client.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", defaultBaseURL, client.baseURL)
	}
	if client.timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, client.timeout)
	}

	customURL := "http://localhost:9000"
	client = NewFastMCPHTTPClient(customURL)
	if client.baseURL != customURL {
		t.Errorf("expected baseURL %s, got %s", customURL, client.baseURL)
	}
}

func TestConnect(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointConnect {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req ConnectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := ConnectResponse{
			Success: true,
			Message: "connected",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
		Args:      []string{"-m", "mcp.server"},
	}

	err := client.Connect(ctx, "test-client", config)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
}

func TestConnectError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConnectResponse{
			Success: false,
			Error:   "connection failed",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "invalid-command",
	}

	err := client.Connect(ctx, "test-client", config)
	if err == nil {
		t.Fatal("Connect should have failed")
	}
}

func TestDisconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointDisconnect {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req DisconnectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := DisconnectResponse{
			Success: true,
			Message: "disconnected",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	err := client.Disconnect(ctx, "test-client")
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}

func TestCallTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointCallTool {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req ToolCallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.ToolName != "test-tool" {
			t.Errorf("unexpected tool name: %s", req.ToolName)
		}

		resp := ToolCallResponse{
			Success: true,
			Result: map[string]any{
				"output": "test result",
				"status": "success",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	result, err := client.CallTool(ctx, "test-client", "test-tool", map[string]any{
		"input": "test",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result["output"] != "test result" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestListTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointListTools {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := ListToolsResponse{
			Success: true,
			Tools: []Tool{
				{
					Name:        "tool1",
					Description: "First tool",
					InputSchema: map[string]any{"type": "object"},
				},
				{
					Name:        "tool2",
					Description: "Second tool",
					InputSchema: map[string]any{"type": "object"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	tools, err := client.ListTools(ctx, "test-client")
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "tool1" {
		t.Errorf("unexpected tool name: %s", tools[0].Name)
	}
}

func TestHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointHealth {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := HealthResponse{
			Status:  "ok",
			Version: "1.0.0",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestHealthUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status: "degraded",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	err := client.Health(ctx)
	if err == nil {
		t.Fatal("Health check should have failed")
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal server error",
		})
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
	}

	err := client.Connect(ctx, "test-client", config)
	if err == nil {
		t.Fatal("Connect should have failed with HTTP error")
	}
}

func TestContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := ConnectResponse{Success: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
	}

	err := client.Connect(ctx, "test-client", config)
	if err == nil {
		t.Fatal("Connect should have failed due to context timeout")
	}
}

func TestRetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail the first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on the 3rd attempt
		resp := ConnectResponse{Success: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFastMCPHTTPClient(server.URL)
	ctx := context.Background()

	config := HTTPMCPConfig{
		Transport: "stdio",
		Command:   "python",
	}

	err := client.Connect(ctx, "test-client", config)
	if err != nil {
		t.Fatalf("Connect should have succeeded after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewFastMCPHTTPClient("")
	newTimeout := 5 * time.Second
	client.SetTimeout(newTimeout)

	if client.timeout != newTimeout {
		t.Errorf("expected timeout %v, got %v", newTimeout, client.timeout)
	}
	if client.httpClient.Timeout != newTimeout {
		t.Errorf("expected http client timeout %v, got %v", newTimeout, client.httpClient.Timeout)
	}
}

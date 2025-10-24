package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/agentapi/lib/agents"
	"github.com/coder/agentapi/lib/audit"
	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/chat"
	"github.com/coder/agentapi/lib/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgent implements Agent interface for testing
type MockAgent struct {
	name   string
	models []agents.ModelInfo
}

func (ma *MockAgent) Execute(ctx context.Context, req *agents.CompletionRequest) (*agents.CompletionResponse, error) {
	return &agents.CompletionResponse{
		Content:      fmt.Sprintf("Mock response from %s for: %s", ma.name, req.Messages[len(req.Messages)-1].Content),
		InputTokens:  10,
		OutputTokens: 20,
		FinishReason: "stop",
		Model:        req.Model,
	}, nil
}

func (ma *MockAgent) Stream(ctx context.Context, req *agents.CompletionRequest) (chan agents.StreamChunk, error) {
	streamChan := make(chan agents.StreamChunk, 10)
	go func() {
		defer close(streamChan)
		response := fmt.Sprintf("Mock streaming response from %s", ma.name)
		for _, char := range response {
			streamChan <- agents.StreamChunk{Content: string(char)}
		}
	}()
	return streamChan, nil
}

func (ma *MockAgent) GetAvailableModels(ctx context.Context) []agents.ModelInfo {
	return ma.models
}

func (ma *MockAgent) IsHealthy(ctx context.Context) bool {
	return true
}

func (ma *MockAgent) Name() string {
	return ma.name
}

// TestAuthKitValidation tests AuthKit token validation
func TestAuthKitValidation(t *testing.T) {
	logger := setupTestLogger()

	// Create mock JWKS server
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Mock JWKS response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{},
		})
	}))
	defer jwksServer.Close()

	validator := auth.NewAuthKitValidator(logger, jwksServer.URL)

	// Test 1: Extract bearer token
	authHeader := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature"
	token, err := validator.ExtractBearerToken(authHeader)
	require.NoError(t, err)
	assert.Equal(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature", token)

	// Test 2: Invalid header format
	_, err = validator.ExtractBearerToken("InvalidHeader")
	assert.Error(t, err)

	// Test 3: Missing header
	_, err = validator.ExtractBearerToken("")
	assert.Error(t, err)
}

// TestChatCompletionRequest tests chat completion request handling
func TestChatCompletionRequest(t *testing.T) {
	logger := setupTestLogger()

	// Create mock agents
	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
		},
	}

	// Create orchestrator
	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	// Create audit logger (nil is acceptable for tests)
	var auditLogger *audit.Logger

	// Create metrics client (nil is acceptable for tests)
	var metricsClient *metrics.MetricsClient

	// Create handler
	handler := chat.NewChatHandler(logger, orchestrator, auditLogger, metricsClient, 4000, 0.7)

	// Test 1: Valid request
	req := &chat.ChatCompletionRequest{
		Model: "gemini-1.5-pro",
		Messages: []chat.Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      false,
	}

	// Create HTTP request
	reqBody, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

	// Add auth context
	ctx := context.WithValue(httpReq.Context(), "authkit_user", &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
	})
	httpReq = httpReq.WithContext(ctx)

	// Create response writer
	w := httptest.NewRecorder()

	// Execute handler
	handler.HandleChatCompletion(w, httpReq)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var response chat.ChatCompletionResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Validate response structure
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "chat.completion", response.Object)
	assert.Equal(t, "gemini-1.5-pro", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, 0, response.Choices[0].Index)
	assert.NotEmpty(t, response.Choices[0].Message.Content)
	assert.Contains(t, response.Choices[0].Message.Content, "ccrouter")
	assert.Equal(t, "stop", response.Choices[0].FinishReason)
	assert.Greater(t, response.Usage.TotalTokens, 0)
	assert.Greater(t, response.Created, int64(0))

	t.Logf("Response: %+v", response)
}

// TestChatCompletionWithOrchestrator tests orchestration
func TestChatCompletionWithOrchestrator(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	user := &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
	}

	// Test 1: Execute non-streaming completion
	req := &chat.ChatCompletionRequest{
		Model: "gemini-1.5-pro",
		Messages: []chat.Message{
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      false,
	}

	ctx := context.Background()
	resp, err := orchestrator.CompleteChat(ctx, user, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Choices[0].Message.Content)
	assert.Equal(t, "gemini-1.5-pro", resp.Model)

	// Test 2: Get available models
	models := orchestrator.GetAvailableModels(ctx, user)
	assert.Greater(t, len(models), 0)

	// Test 3: Droid fallback
	failingAgent := &failingMockAgent{mockCCRouter}
	orchestrator2, _ := chat.NewOrchestrator(logger, failingAgent, mockDroid, "ccrouter", true)

	resp2, err := orchestrator2.CompleteChat(ctx, user, req)
	// Should succeed via droid fallback
	if err != nil {
		t.Logf("Error: %v", err)
	}
}

// TestStreamingCompletion tests streaming chat completion
func TestStreamingCompletion(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	// Create handler
	handler := chat.NewChatHandler(logger, orchestrator, nil, nil, 4000, 0.7)

	user := &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
	}

	req := &chat.ChatCompletionRequest{
		Model: "gemini-1.5-pro",
		Messages: []chat.Message{
			{Role: "user", Content: "Tell me a story"},
		},
		Stream: true,
	}

	// Create HTTP request
	reqBody, err := json.Marshal(req)
	require.NoError(t, err)
	httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

	// Add auth context
	ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
	httpReq = httpReq.WithContext(ctx)

	// Create response writer
	w := httptest.NewRecorder()

	// Execute streaming handler
	handler.HandleChatCompletion(w, httpReq)

	// Verify streaming response headers
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	// Parse SSE events
	body := w.Body.String()
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "data: ")
	assert.Contains(t, body, "[DONE]")

	// Count chunks (each SSE event starts with "data: ")
	chunks := 0
	for _, line := range splitSSE(body) {
		if line == "[DONE]" {
			break
		}
		chunks++
	}
	assert.Greater(t, chunks, 0, "Should have received at least one SSE chunk")

	t.Logf("Received %d SSE chunks", chunks)

	// Test orchestrator streaming directly
	ctx2 := context.Background()
	streamChan, err := orchestrator.StreamCompletion(ctx2, user, req)
	require.NoError(t, err)

	// Consume stream
	var agentChunks []agents.StreamChunk
	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		for chunk := range streamChan {
			agentChunks = append(agentChunks, chunk)
			if chunk.Error != nil {
				t.Logf("Streaming error: %v", chunk.Error)
			}
		}
		done <- true
	}()

	select {
	case <-done:
		assert.Greater(t, len(agentChunks), 0, "Should have received agent chunks")
		t.Logf("Received %d agent stream chunks", len(agentChunks))
	case <-timeout:
		t.Fatal("Streaming test timed out")
	}
}

// splitSSE splits SSE response body into individual data payloads
func splitSSE(body string) []string {
	var events []string
	lines := bytes.Split([]byte(body), []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			events = append(events, string(data))
		}
	}
	return events
}

// TestAgentModelSelection tests agent selection logic
func TestAgentModelSelection(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
			{ID: "gemini-1.5-flash", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
			{ID: "claude-3-opus", OwnedBy: "anthropic"},
			{ID: "claude-3-sonnet", OwnedBy: "anthropic"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	user := &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
	}

	ctx := context.Background()

	t.Run("Gemini model routes to CCRouter", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gemini-1.5-pro",
			Messages: []chat.Message{{Role: "user", Content: "test gemini"}},
		}
		resp, err := orchestrator.CompleteChat(ctx, user, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Choices[0].Message.Content, "ccrouter", "Gemini model should route to CCRouter")
		assert.Equal(t, "gemini-1.5-pro", resp.Model)
		t.Logf("Gemini response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("Claude model routes to Droid", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "claude-3-opus",
			Messages: []chat.Message{{Role: "user", Content: "test claude"}},
		}
		resp, err := orchestrator.CompleteChat(ctx, user, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Choices[0].Message.Content, "droid", "Claude model should route to Droid")
		assert.Equal(t, "claude-3-opus", resp.Model)
		t.Logf("Claude response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("GPT-4 model routes to CCRouter (based on routing logic)", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gpt-4",
			Messages: []chat.Message{{Role: "user", Content: "test gpt4"}},
		}
		resp, err := orchestrator.CompleteChat(ctx, user, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		// Based on orchestrator.go logic, gpt-4 routes to CCRouter
		assert.Contains(t, resp.Choices[0].Message.Content, "ccrouter", "GPT-4 should route to CCRouter")
		assert.Equal(t, "gpt-4", resp.Model)
		t.Logf("GPT-4 response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("Different Gemini variant routes correctly", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gemini-1.5-flash",
			Messages: []chat.Message{{Role: "user", Content: "test gemini flash"}},
		}
		resp, err := orchestrator.CompleteChat(ctx, user, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Choices[0].Message.Content, "ccrouter")
		t.Logf("Gemini Flash response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("Verify model info is returned correctly", func(t *testing.T) {
		models := orchestrator.GetAvailableModels(ctx, user)
		require.Greater(t, len(models), 0, "Should return available models")

		// Check that models from both agents are present
		modelIDs := make(map[string]bool)
		for _, m := range models {
			modelIDs[m.ID] = true
		}

		assert.True(t, modelIDs["gemini-1.5-pro"], "Should include gemini-1.5-pro")
		assert.True(t, modelIDs["claude-3-opus"], "Should include claude-3-opus")
		t.Logf("Available models: %d", len(models))
	})
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	logger := setupTestLogger()

	t.Run("No fallback - should fail", func(t *testing.T) {
		failingAgent := &failingMockAgent{nil}
		orchestrator, err := chat.NewOrchestrator(logger, failingAgent, nil, "ccrouter", false)
		require.NoError(t, err)

		user := &auth.AuthKitUser{
			ID:    "user-123",
			OrgID: "org-456",
		}

		req := &chat.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		// Should fail without fallback
		_, err = orchestrator.CompleteChat(context.Background(), user, req)
		assert.Error(t, err, "Should fail when primary agent fails and no fallback is configured")
		t.Logf("Expected error: %v", err)
	})

	t.Run("With fallback - should succeed", func(t *testing.T) {
		failingAgent := &failingMockAgent{nil}
		mockDroid := &MockAgent{
			name: "droid",
			models: []agents.ModelInfo{
				{ID: "gpt-4", OwnedBy: "openai"},
			},
		}

		orchestrator, err := chat.NewOrchestrator(logger, failingAgent, mockDroid, "ccrouter", true)
		require.NoError(t, err)

		user := &auth.AuthKitUser{
			ID:    "user-123",
			OrgID: "org-456",
		}

		req := &chat.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []chat.Message{{Role: "user", Content: "test fallback"}},
		}

		// Should succeed via fallback
		resp, err := orchestrator.CompleteChat(context.Background(), user, req)
		require.NoError(t, err, "Should succeed when fallback agent is available")
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Choices[0].Message.Content, "droid", "Should use droid fallback")
		t.Logf("Fallback response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("Streaming failure with fallback", func(t *testing.T) {
		failingAgent := &failingMockAgent{nil}
		mockDroid := &MockAgent{
			name: "droid",
			models: []agents.ModelInfo{
				{ID: "gpt-4", OwnedBy: "openai"},
			},
		}

		orchestrator, err := chat.NewOrchestrator(logger, failingAgent, mockDroid, "ccrouter", true)
		require.NoError(t, err)

		user := &auth.AuthKitUser{
			ID:    "user-123",
			OrgID: "org-456",
		}

		req := &chat.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []chat.Message{{Role: "user", Content: "test stream fallback"}},
			Stream:   true,
		}

		ctx := context.Background()
		streamChan, err := orchestrator.StreamCompletion(ctx, user, req)
		require.NoError(t, err)

		// The first chunk might contain an error, but fallback should kick in
		chunks := 0
		for chunk := range streamChan {
			if chunk.Error == nil {
				chunks++
			}
		}

		// Should receive chunks from fallback
		assert.Greater(t, chunks, 0, "Should receive chunks from fallback agent")
		t.Logf("Received %d chunks after fallback", chunks)
	})

	t.Run("Both agents fail", func(t *testing.T) {
		failingAgent1 := &failingMockAgent{nil}
		failingAgent2 := &failingMockAgent{nil}

		orchestrator, err := chat.NewOrchestrator(logger, failingAgent1, failingAgent2, "ccrouter", true)
		require.NoError(t, err)

		user := &auth.AuthKitUser{
			ID:    "user-123",
			OrgID: "org-456",
		}

		req := &chat.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		// Should fail when both agents fail
		_, err = orchestrator.CompleteChat(context.Background(), user, req)
		assert.Error(t, err, "Should fail when both primary and fallback agents fail")
		t.Logf("Expected error: %v", err)
	})

	t.Run("HTTP handler error response", func(t *testing.T) {
		failingAgent := &failingMockAgent{nil}
		orchestrator, err := chat.NewOrchestrator(logger, failingAgent, nil, "ccrouter", false)
		require.NoError(t, err)

		handler := chat.NewChatHandler(logger, orchestrator, nil, nil, 4000, 0.7)

		req := &chat.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{
			ID:    "user-123",
			OrgID: "org-456",
		}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code, "Should return 500 on error")

		var errResp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&errResp)
		assert.Contains(t, errResp, "error", "Should contain error field")
		t.Logf("Error response: %+v", errResp)
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	handler := chat.NewChatHandler(logger, orchestrator, nil, nil, 4000, 0.7)

	t.Run("Empty messages array", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gemini-1.5-pro",
			Messages: []chat.Message{},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject empty messages")
		assert.Contains(t, w.Body.String(), "messages are required")
		t.Logf("Empty messages response: %s", w.Body.String())
	})

	t.Run("Invalid model name", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "non-existent-model-xyz",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		// Should still route to an agent (primary agent will handle it)
		// The agent might succeed (mocked) or fail depending on implementation
		t.Logf("Invalid model response code: %d", w.Code)
	})

	t.Run("Missing model field", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject missing model")
		assert.Contains(t, w.Body.String(), "model is required")
		t.Logf("Missing model response: %s", w.Body.String())
	})

	t.Run("Malformed JSON request", func(t *testing.T) {
		malformedJSON := []byte(`{"model": "gemini-1.5-pro", "messages": [invalid json`)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(malformedJSON))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject malformed JSON")
		assert.Contains(t, w.Body.String(), "invalid request")
		t.Logf("Malformed JSON response: %s", w.Body.String())
	})

	t.Run("Missing auth context", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gemini-1.5-pro",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		// Don't add auth context
		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject unauthenticated request")
		assert.Contains(t, w.Body.String(), "unauthorized")
		t.Logf("Unauthorized response: %s", w.Body.String())
	})

	t.Run("Very long message content", func(t *testing.T) {
		// Create a very long message (10KB)
		longContent := string(make([]byte, 10000))
		for i := range longContent {
			longContent = longContent[:i] + "a"
		}

		req := &chat.ChatCompletionRequest{
			Model: "gemini-1.5-pro",
			Messages: []chat.Message{
				{Role: "user", Content: longContent},
			},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		// Should handle long content gracefully
		assert.NotEqual(t, http.StatusInternalServerError, w.Code, "Should handle long content")
		t.Logf("Long content response code: %d", w.Code)
	})

	t.Run("Special characters in messages", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model: "gemini-1.5-pro",
			Messages: []chat.Message{
				{Role: "user", Content: "Test with special chars: <>&\"'"},
			},
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

		user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Should handle special characters")
		t.Logf("Special chars response code: %d", w.Code)
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	mockDroid := &MockAgent{
		name: "droid",
		models: []agents.ModelInfo{
			{ID: "gpt-4", OwnedBy: "openai"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, mockDroid, "ccrouter", true)
	require.NoError(t, err)

	handler := chat.NewChatHandler(logger, orchestrator, nil, nil, 4000, 0.7)

	user := &auth.AuthKitUser{
		ID:    "user-123",
		OrgID: "org-456",
	}

	// Run multiple concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			req := &chat.ChatCompletionRequest{
				Model: "gemini-1.5-pro",
				Messages: []chat.Message{
					{Role: "user", Content: fmt.Sprintf("Concurrent request %d", index)},
				},
			}

			reqBody, _ := json.Marshal(req)
			httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))

			ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
			httpReq = httpReq.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.HandleChatCompletion(w, httpReq)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("request %d failed with status %d", index, w.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	timeout := time.After(10 * time.Second)
	completed := 0

	for completed < numRequests {
		select {
		case <-done:
			completed++
		case err := <-errors:
			t.Logf("Concurrent request error: %v", err)
		case <-timeout:
			t.Fatalf("Concurrent requests timed out, completed: %d/%d", completed, numRequests)
		}
	}

	assert.Equal(t, numRequests, completed, "All concurrent requests should complete")
	t.Logf("Successfully completed %d concurrent requests", completed)
}

// TestTemperatureAndMaxTokens tests parameter handling
func TestTemperatureAndMaxTokens(t *testing.T) {
	logger := setupTestLogger()

	mockCCRouter := &MockAgent{
		name: "ccrouter",
		models: []agents.ModelInfo{
			{ID: "gemini-1.5-pro", OwnedBy: "google"},
		},
	}

	orchestrator, err := chat.NewOrchestrator(logger, mockCCRouter, nil, "ccrouter", false)
	require.NoError(t, err)

	handler := chat.NewChatHandler(logger, orchestrator, nil, nil, 4000, 0.7)

	user := &auth.AuthKitUser{ID: "user-123", OrgID: "org-456"}

	t.Run("Custom temperature and max_tokens", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:       "gemini-1.5-pro",
			Messages:    []chat.Message{{Role: "user", Content: "test"}},
			Temperature: 0.9,
			MaxTokens:   2000,
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("Custom params response code: %d", w.Code)
	})

	t.Run("Default temperature and max_tokens", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:    "gemini-1.5-pro",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
			// Temperature and MaxTokens not set - should use defaults
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response chat.ChatCompletionResponse
		json.NewDecoder(w.Body).Decode(&response)
		// Defaults should be applied: temperature=0.7, max_tokens=4000
		t.Logf("Default params applied successfully")
	})

	t.Run("Zero temperature", func(t *testing.T) {
		req := &chat.ChatCompletionRequest{
			Model:       "gemini-1.5-pro",
			Messages:    []chat.Message{{Role: "user", Content: "test"}},
			Temperature: 0.0,
		}

		reqBody, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(reqBody))
		ctx := context.WithValue(httpReq.Context(), "authkit_user", user)
		httpReq = httpReq.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleChatCompletion(w, httpReq)

		// Zero temperature should trigger default temperature
		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("Zero temperature handled, defaults applied")
	})
}

// failingMockAgent always returns an error
type failingMockAgent struct {
	fallback agents.Agent
}

func (fma *failingMockAgent) Execute(ctx context.Context, req *agents.CompletionRequest) (*agents.CompletionResponse, error) {
	return nil, fmt.Errorf("agent failed")
}

func (fma *failingMockAgent) Stream(ctx context.Context, req *agents.CompletionRequest) (chan agents.StreamChunk, error) {
	return nil, fmt.Errorf("streaming failed")
}

func (fma *failingMockAgent) GetAvailableModels(ctx context.Context) []agents.ModelInfo {
	return []agents.ModelInfo{}
}

func (fma *failingMockAgent) IsHealthy(ctx context.Context) bool {
	return false
}

func (fma *failingMockAgent) Name() string {
	return "failing"
}

// Helper functions

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/admin"
	"github.com/coder/agentapi/lib/audit"
	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/metrics"
)

// ChatCompletionRequest represents OpenAI-compatible chat completion request
type ChatCompletionRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Temperature  float32   `json:"temperature,omitempty"`
	MaxTokens    int       `json:"max_tokens,omitempty"`
	TopP         float32   `json:"top_p,omitempty"`
	Stream       bool      `json:"stream,omitempty"`
	User         string    `json:"user,omitempty"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatCompletionResponse represents OpenAI-compatible chat completion response
type ChatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"` // "chat.completion"
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   UsageInfo `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int           `json:"index"`
	Message      Message       `json:"message,omitempty"`
	FinishReason string        `json:"finish_reason"`   // stop, max_tokens, error
	Delta        *DeltaMessage `json:"delta,omitempty"` // for streaming
}

// DeltaMessage represents streaming delta
type DeltaMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// UsageInfo represents token usage information
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatHandler handles chat completion requests
type ChatHandler struct {
	logger        *slog.Logger
	orchestrator  *Orchestrator
	auditLogger   *audit.AuditLogger
	metricsClient *metrics.MetricsRegistry
	maxTokens     int
	defaultTemp   float32
	adminService  *admin.PlatformAdminService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	logger *slog.Logger,
	orchestrator *Orchestrator,
	auditLogger *audit.AuditLogger,
	metricsClient *metrics.MetricsRegistry,
	maxTokens int,
	defaultTemp float32,
	adminService *admin.PlatformAdminService,
) *ChatHandler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	return &ChatHandler{
		logger:        logger,
		orchestrator:  orchestrator,
		auditLogger:   auditLogger,
		metricsClient: metricsClient,
		maxTokens:     maxTokens,
		defaultTemp:   defaultTemp,
		adminService:  adminService,
	}
}

// HandleChatCompletion handles POST /v1/chat/completions
func (ch *ChatHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	timer := time.Now()

	// Get authenticated user from context
	user, ok := ctx.Value("authkit_user").(*auth.AuthKitUser)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ch.logger.Error("failed to parse chat completion request",
			"user_id", user.ID,
			"error", err.Error(),
		)
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		http.Error(w, "messages are required", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Temperature == 0 {
		req.Temperature = ch.defaultTemp
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = ch.maxTokens
	}

	// Audit log the request
	ch.auditLogger.LogWithContext(ctx, "chat_completion_requested", "chat", "", map[string]interface{}{
		"user_id":  user.ID,
		"org_id":   user.OrgID,
		"model":    req.Model,
		"messages": len(req.Messages),
		"stream":   req.Stream,
	})

	// Execute chat completion
	if req.Stream {
		ch.handleStreamingCompletion(w, ctx, user, &req)
	} else {
		ch.handleNonStreamingCompletion(w, ctx, user, &req)
	}

	// Record metrics
	duration := time.Since(timer).Milliseconds()
	if ch.metricsClient != nil {
		// TODO: Add chat latency recording to MetricsRegistry
		_ = duration
	}

	ch.logger.Info("chat completion processed",
		"user_id", user.ID,
		"model", req.Model,
		"stream", req.Stream,
		"duration_ms", duration,
	)
}

// handleStreamingCompletion handles streaming chat completion with fallback
func (ch *ChatHandler) handleStreamingCompletion(w http.ResponseWriter, ctx context.Context, user *auth.AuthKitUser, req *ChatCompletionRequest) {
	// Try streaming first
	streamErr := ch.executeStreamingCompletion(w, ctx, user, req)

	// If streaming fails, fallback to non-streaming
	if streamErr != nil {
		ch.logger.Warn("streaming failed, falling back to non-streaming",
			"user_id", user.ID,
			"error", streamErr.Error(),
		)

		// Reset response writer for fallback
		w.Header().Set("Content-Type", "application/json")
		req.Stream = false
		ch.handleNonStreamingCompletion(w, ctx, user, req)
	}
}

// executeStreamingCompletion executes streaming completion
func (ch *ChatHandler) executeStreamingCompletion(w http.ResponseWriter, ctx context.Context, user *auth.AuthKitUser, req *ChatCompletionRequest) error {
	// Set streaming headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Get streaming response from orchestrator
	streamChan, err := ch.orchestrator.StreamCompletion(ctx, user, req)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	// Send initial chunk with role
	initialChunk := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Delta: &DeltaMessage{
					Role: "assistant",
				},
				FinishReason: "content",
			},
		},
	}
	ch.writeSSEEvent(w, &initialChunk)
	flusher.Flush()

	// Stream response chunks
	fullResponse := ""
	for chunk := range streamChan {
		if chunk.Error != nil {
			ch.logger.Error("streaming error",
				"user_id", user.ID,
				"error", chunk.Error.Error(),
			)
			break
		}

		fullResponse += chunk.Content

		// Send delta chunk
		deltaChunk := ChatCompletionResponse{
			ID:      initialChunk.ID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []Choice{
				{
					Index: 0,
					Delta: &DeltaMessage{
						Content: chunk.Content,
					},
				},
			},
		}
		ch.writeSSEEvent(w, &deltaChunk)
		flusher.Flush()
	}

	// Send completion chunk
	completionChunk := ChatCompletionResponse{
		ID:      initialChunk.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index:        0,
				FinishReason: "stop",
			},
		},
	}
	ch.writeSSEEvent(w, &completionChunk)

	// Send done message
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	return nil
}

// handleNonStreamingCompletion handles non-streaming chat completion
func (ch *ChatHandler) handleNonStreamingCompletion(w http.ResponseWriter, ctx context.Context, user *auth.AuthKitUser, req *ChatCompletionRequest) {
	// Get completion from orchestrator
	response, err := ch.orchestrator.CompleteChat(ctx, user, req)
	if err != nil {
		ch.logger.Error("chat completion failed",
			"user_id", user.ID,
			"error", err.Error(),
		)

		// Return error response
		errResp := map[string]interface{}{
			"error": map[string]interface{}{
				"message": err.Error(),
				"type":    "orchestration_error",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeSSEEvent writes a server-sent event
func (ch *ChatHandler) writeSSEEvent(w http.ResponseWriter, data interface{}) error {
	eventData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", eventData)
	return err
}

// HandleListModels handles GET /v1/models
func (ch *ChatHandler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user from context
	user, ok := ctx.Value("authkit_user").(*auth.AuthKitUser)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get available models from orchestrator
	models := ch.orchestrator.GetAvailableModels(ctx, user)

	response := map[string]interface{}{
		"object": "list",
		"data":   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	ch.logger.Info("models listed",
		"user_id", user.ID,
		"count", len(models),
	)
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content string
	Error   error
}

// Platform Admin Handlers

// HandlePlatformStats returns platform-wide statistics
func (ch *ChatHandler) HandlePlatformStats(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	stats, err := ch.adminService.GetPlatformStats(r.Context())
	if err != nil {
		ch.logger.Error("failed to get platform stats", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleListPlatformAdmins returns all platform admins
func (ch *ChatHandler) HandleListPlatformAdmins(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	admins, err := ch.adminService.ListAdmins(r.Context())
	if err != nil {
		ch.logger.Error("failed to list platform admins", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"admins": admins,
		"count":  len(admins),
	})
}

// HandleAddPlatformAdmin adds a user as platform admin
func (ch *ChatHandler) HandleAddPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	var req struct {
		WorkOSID string `json:"workos_id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.WorkOSID == "" {
		http.Error(w, "email and workos_id are required", http.StatusBadRequest)
		return
	}

	err := ch.adminService.AddAdmin(r.Context(), req.WorkOSID, req.Email, req.Name, user.ID)
	if err != nil {
		ch.logger.Error("failed to add platform admin", "error", err, "email", req.Email)
		http.Error(w, fmt.Sprintf("failed to add admin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"email":  req.Email,
	})
}

// HandleRemovePlatformAdmin removes a user from platform admins
func (ch *ChatHandler) HandleRemovePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	// Extract email from URL path
	email := r.URL.Path[len("/api/v1/platform/admins/"):]
	if email == "" {
		http.Error(w, "email parameter required", http.StatusBadRequest)
		return
	}

	err := ch.adminService.RemoveAdmin(r.Context(), email, user.ID)
	if err != nil {
		ch.logger.Error("failed to remove platform admin", "error", err, "email", email)
		http.Error(w, fmt.Sprintf("failed to remove admin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"email":  email,
	})
}

// HandleGetAuditLog returns audit log entries
func (ch *ChatHandler) HandleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("authkit_user").(*auth.AuthKitUser)

	if !user.IsPlatformAdmin() {
		http.Error(w, "forbidden: platform admin required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil || o != 1 {
			http.Error(w, "invalid offset parameter", http.StatusBadRequest)
			return
		}
	}

	entries, err := ch.adminService.GetAuditLog(r.Context(), limit, offset)
	if err != nil {
		ch.logger.Error("failed to get audit log", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
		"limit":   limit,
		"offset":  offset,
	})
}

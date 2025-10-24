package agents

import (
	"context"
)

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// ModelInfo represents available model information
type ModelInfo struct {
	ID              string  `json:"id"`
	Object          string  `json:"object"` // "model"
	Created         int64   `json:"created"`
	OwnedBy         string  `json:"owned_by"`
	Description     string  `json:"description"`
	MaxTokens       int     `json:"max_tokens"`
	InputCostPer1K  float32 `json:"input_cost_per_1k"`
	OutputCostPer1K float32 `json:"output_cost_per_1k"`
}

// CompletionRequest represents a completion request to an agent
type CompletionRequest struct {
	Model        string
	Messages     []Message
	Temperature  float32
	MaxTokens    int
	TopP         float32
	SystemPrompt string
	UserID       string
	OrgID        string
	Stream       bool
}

// CompletionResponse represents a completion response from an agent
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	FinishReason string
	Model        string
}

// StreamChunk represents a chunk of streamed content
type StreamChunk struct {
	Content string
	Error   error
}

// Agent interface represents an AI agent backend
type Agent interface {
	// Execute runs a synchronous completion
	Execute(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream runs a streaming completion
	Stream(ctx context.Context, req *CompletionRequest) (chan StreamChunk, error)

	// GetAvailableModels returns list of available models
	GetAvailableModels(ctx context.Context) []ModelInfo

	// IsHealthy checks if agent is available and healthy
	IsHealthy(ctx context.Context) bool

	// Name returns the agent name
	Name() string
}

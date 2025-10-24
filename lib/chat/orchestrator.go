package chat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/agents"
	"github.com/coder/agentapi/lib/auth"
	"github.com/coder/agentapi/lib/resilience"
)

// Orchestrator manages agent backend selection and execution
type Orchestrator struct {
	logger          *slog.Logger
	ccrouter        agents.Agent
	droid           agents.Agent
	primaryAgent    string
	fallbackEnabled bool
	circuitBreaker  *resilience.CircuitBreaker
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator(
	logger *slog.Logger,
	ccrouter agents.Agent,
	droid agents.Agent,
	primaryAgent string,
	fallbackEnabled bool,
) (*Orchestrator, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	// Create circuit breaker for orchestration
	cbConfig := resilience.CBConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 100,
	}
	cb, err := resilience.NewCircuitBreaker("chat_orchestration", cbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
	}

	return &Orchestrator{
		logger:          logger,
		ccrouter:        ccrouter,
		droid:           droid,
		primaryAgent:    primaryAgent,
		fallbackEnabled: fallbackEnabled,
		circuitBreaker:  cb,
	}, nil
}

// CompleteChat executes a chat completion request
func (o *Orchestrator) CompleteChat(
	ctx context.Context,
	user *auth.AuthKitUser,
	req *ChatCompletionRequest,
) (*ChatCompletionResponse, error) {
	startTime := time.Now()

	// Execute via circuit breaker
	var response *ChatCompletionResponse
	err := o.circuitBreaker.Execute(ctx, func() error {
		var err error
		response, err = o.executeCompletion(ctx, user, req)
		return err
	})

	if err != nil {
		o.logger.Error("chat completion failed",
			"user_id", user.ID,
			"model", req.Model,
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, err
	}
	response.Created = time.Now().Unix()

	o.logger.Info("chat completion succeeded",
		"user_id", user.ID,
		"model", req.Model,
		"tokens", response.Usage.TotalTokens,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return response, nil
}

// executeCompletion executes the actual completion
func (o *Orchestrator) executeCompletion(
	ctx context.Context,
	user *auth.AuthKitUser,
	req *ChatCompletionRequest,
) (*ChatCompletionResponse, error) {
	// Determine which agent to use
	agent, agentName := o.selectAgent(req.Model)
	if agent == nil {
		return nil, fmt.Errorf("no agent available for model: %s", req.Model)
	}

	o.logger.Debug("executing completion",
		"user_id", user.ID,
		"agent", agentName,
		"model", req.Model,
	)

	// Build agent request
	agentReq := &agents.CompletionRequest{
		Model:        req.Model,
		Messages:     convertMessages(req.Messages),
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		TopP:         req.TopP,
		SystemPrompt: req.SystemPrompt,
		UserID:       user.ID,
		OrgID:        user.OrgID,
	}

	// Execute on selected agent
	agentResp, err := agent.Execute(ctx, agentReq)
	if err != nil {
		// Try fallback if enabled
		if o.fallbackEnabled && agentName == o.primaryAgent {
			o.logger.Warn("primary agent failed, trying fallback",
				"user_id", user.ID,
				"primary_agent", agentName,
				"error", err.Error(),
			)

			fallbackAgent := o.getFallbackAgent(agentName)
			if fallbackAgent != nil {
				agentResp, err = fallbackAgent.Execute(ctx, agentReq)
				if err != nil {
					return nil, fmt.Errorf("fallback agent also failed: %w", err)
				}
				o.logger.Info("fallback agent succeeded",
					"user_id", user.ID,
					"fallback_agent", o.getAgentName(fallbackAgent),
				)
			}
		}

		if err != nil {
			return nil, fmt.Errorf("agent execution failed: %w", err)
		}
	}

	// Convert agent response to chat completion response
	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      Message{Role: "assistant", Content: agentResp.Content},
				FinishReason: "stop",
			},
		},
		Usage: UsageInfo{
			PromptTokens:     agentResp.InputTokens,
			CompletionTokens: agentResp.OutputTokens,
			TotalTokens:      agentResp.InputTokens + agentResp.OutputTokens,
		},
	}, nil
}

// StreamCompletion returns a channel of streaming response chunks
func (o *Orchestrator) StreamCompletion(
	ctx context.Context,
	user *auth.AuthKitUser,
	req *ChatCompletionRequest,
) (chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 10)

	// Determine which agent to use
	agent, agentName := o.selectAgent(req.Model)
	if agent == nil {
		close(streamChan)
		return nil, fmt.Errorf("no agent available for model: %s", req.Model)
	}

	// Build agent request
	agentReq := &agents.CompletionRequest{
		Model:        req.Model,
		Messages:     convertMessages(req.Messages),
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		TopP:         req.TopP,
		SystemPrompt: req.SystemPrompt,
		UserID:       user.ID,
		OrgID:        user.OrgID,
		Stream:       true,
	}

	// Start streaming in goroutine
	go func() {
		defer close(streamChan)

		// Get streaming response
		streamRespChan, err := agent.Stream(ctx, agentReq)
		if err != nil {
			streamChan <- StreamChunk{Error: fmt.Errorf("stream failed: %w", err)}
			return
		}

		// Forward chunks from agent stream to response stream
		for chunk := range streamRespChan {
			if chunk.Error != nil {
				// Try fallback if enabled
				if o.fallbackEnabled && agentName == o.primaryAgent {
					o.logger.Warn("primary agent streaming failed, trying fallback",
						"user_id", user.ID,
						"error", chunk.Error.Error(),
					)

					fallbackAgent := o.getFallbackAgent(agentName)
					if fallbackAgent != nil {
						// Restart streaming from fallback
						newStreamChan, err := fallbackAgent.Stream(ctx, agentReq)
						if err == nil {
							for newChunk := range newStreamChan {
								streamChan <- StreamChunk{
									Content: newChunk.Content,
									Error:   newChunk.Error,
								}
							}
							return
						}
					}
				}

				streamChan <- StreamChunk{
					Content: chunk.Content,
					Error:   chunk.Error,
				}
				return
			}

			streamChan <- StreamChunk{
				Content: chunk.Content,
				Error:   chunk.Error,
			}
		}

		o.logger.Info("streaming completed",
			"user_id", user.ID,
			"model", req.Model,
			"agent", agentName,
		)
	}()

	return streamChan, nil
}

// GetAvailableModels returns list of available models
func (o *Orchestrator) GetAvailableModels(ctx context.Context, user *auth.AuthKitUser) []agents.ModelInfo {
	var models []agents.ModelInfo

	// Get CCRouter models
	if o.ccrouter != nil {
		ccModels := o.ccrouter.GetAvailableModels(ctx)
		models = append(models, ccModels...)
	}

	// Get Droid models
	if o.droid != nil {
		droidModels := o.droid.GetAvailableModels(ctx)
		models = append(models, droidModels...)
	}

	return models
}

// selectAgent returns the agent to use for the given model
func (o *Orchestrator) selectAgent(model string) (agents.Agent, string) {
	// Check if model belongs to CCRouter
	if strings.Contains(model, "gemini") || strings.HasPrefix(model, "gpt-4") {
		if o.ccrouter != nil {
			return o.ccrouter, "ccrouter"
		}
	}

	// Default to primary agent
	if o.primaryAgent == "ccrouter" && o.ccrouter != nil {
		return o.ccrouter, "ccrouter"
	}

	if o.primaryAgent == "droid" && o.droid != nil {
		return o.droid, "droid"
	}

	// Fall back to whichever is available
	if o.ccrouter != nil {
		return o.ccrouter, "ccrouter"
	}

	return o.droid, "droid"
}

// getFallbackAgent returns the fallback agent
func (o *Orchestrator) getFallbackAgent(currentAgent string) agents.Agent {
	if currentAgent == "ccrouter" {
		return o.droid
	}
	return o.ccrouter
}

// getAgentName returns the name of the given agent
func (o *Orchestrator) getAgentName(agent agents.Agent) string {
	if agent == o.ccrouter {
		return "ccrouter"
	}
	return "droid"
}

// convertMessages converts chat.Message to agents.Message
func convertMessages(chatMessages []Message) []agents.Message {
	agentMessages := make([]agents.Message, len(chatMessages))
	for i, msg := range chatMessages {
		agentMessages[i] = agents.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return agentMessages
}

package agents

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CCRouterAgent wraps CCRouter CLI for chat completions
type CCRouterAgent struct {
	logger  *slog.Logger
	ccrPath string
	timeout time.Duration
	models  []ModelInfo
}

// CCRouterResponse represents the JSON response from CCRouter
type CCRouterResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
	Tokens  struct {
		Input  int `json:"input"`
		Output int `json:"output"`
	} `json:"tokens"`
}

// NewCCRouterAgent creates a new CCRouter agent
func NewCCRouterAgent(logger *slog.Logger, ccrPath string, timeout time.Duration) *CCRouterAgent {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	agent := &CCRouterAgent{
		logger:  logger,
		ccrPath: ccrPath,
		timeout: timeout,
		models: []ModelInfo{
			{
				ID:              "gemini-1.5-pro",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "google",
				Description:     "Gemini 1.5 Pro via VertexAI",
				MaxTokens:       8000,
				InputCostPer1K:  0.00125,
				OutputCostPer1K: 0.005,
			},
			{
				ID:              "gemini-1.5-flash",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "google",
				Description:     "Gemini 1.5 Flash via VertexAI",
				MaxTokens:       8000,
				InputCostPer1K:  0.000075,
				OutputCostPer1K: 0.0003,
			},
			{
				ID:              "gpt-4-turbo",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "openai",
				Description:     "GPT-4 Turbo via OpenAI",
				MaxTokens:       4096,
				InputCostPer1K:  0.01,
				OutputCostPer1K: 0.03,
			},
			{
				ID:              "claude-3-opus",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "anthropic",
				Description:     "Claude 3 Opus via CCRouter",
				MaxTokens:       4096,
				InputCostPer1K:  0.015,
				OutputCostPer1K: 0.075,
			},
		},
	}

	// Verify CCRouter is available
	if !agent.IsHealthy(context.Background()) {
		logger.Warn("CCRouter not available at path", "path", ccrPath)
	}

	return agent
}

// Execute runs a synchronous completion via CCRouter
func (ca *CCRouterAgent) Execute(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, ca.timeout)
	defer cancel()

	// Build system prompt
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant."
	}

	// Build user message
	userContent := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userContent = msg.Content
			break
		}
	}

	if userContent == "" {
		return nil, fmt.Errorf("no user message found")
	}

	// Build CCRouter command
	args := []string{
		"code",
		"--model", ca.mapModelName(req.Model),
		"--system", systemPrompt,
		"--temperature", fmt.Sprintf("%.2f", req.Temperature),
	}

	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
	}

	// Execute CCRouter
	cmd := exec.CommandContext(ctx, ca.ccrPath, args...)

	// Set stdin to user message
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ccrouter: %w", err)
	}

	// Write user message to stdin
	go func() {
		defer stdin.Close()
		fmt.Fprint(stdin, userContent)
	}()

	// Read response
	responseData := ""
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		responseData += scanner.Text() + "\n"
	}

	// Check for errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	// Read stderr for debugging
	stderrData := ""
	errScanner := bufio.NewScanner(stderr)
	for errScanner.Scan() {
		stderrData += errScanner.Text() + "\n"
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		ca.logger.Error("ccrouter command failed",
			"model", req.Model,
			"error", err.Error(),
			"stderr", stderrData,
		)
		return nil, fmt.Errorf("ccrouter execution failed: %w", err)
	}

	// Parse response
	response := &CompletionResponse{
		Content:      strings.TrimSpace(responseData),
		InputTokens:  0, // CCRouter doesn't provide token counts
		OutputTokens: 0,
		FinishReason: "stop",
		Model:        req.Model,
	}

	ca.logger.Info("ccrouter execution succeeded",
		"model", req.Model,
		"response_length", len(response.Content),
	)

	return response, nil
}

// Stream runs a streaming completion via CCRouter
func (ca *CCRouterAgent) Stream(ctx context.Context, req *CompletionRequest) (chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 10)

	go func() {
		defer close(streamChan)

		// For now, execute synchronously and stream the result
		// In future, integrate with CCRouter's streaming capability
		resp, err := ca.Execute(ctx, req)
		if err != nil {
			streamChan <- StreamChunk{Error: err}
			return
		}

		// Stream character by character
		for i := 0; i < len(resp.Content); i++ {
			streamChan <- StreamChunk{
				Content: string(resp.Content[i]),
				Error:   nil,
			}
		}
	}()

	return streamChan, nil
}

// GetAvailableModels returns available models
func (ca *CCRouterAgent) GetAvailableModels(ctx context.Context) []ModelInfo {
	return ca.models
}

// IsHealthy checks if CCRouter is available
func (ca *CCRouterAgent) IsHealthy(ctx context.Context) bool {
	// Check if CCRouter binary exists
	info, err := os.Stat(ca.ccrPath)
	if err != nil {
		return false
	}

	// Check if it's executable
	return !info.IsDir()
}

// Name returns the agent name
func (ca *CCRouterAgent) Name() string {
	return "ccrouter"
}

// mapModelName maps model names to CCRouter equivalents
func (ca *CCRouterAgent) mapModelName(model string) string {
	modelMap := map[string]string{
		"gemini-1.5-pro":   "vertex-gemini",
		"gemini-1.5-flash": "vertex-gemini",
		"gpt-4-turbo":      "gpt-4",
		"gpt-4":            "gpt-4",
		"claude-3-opus":    "claude-3-opus",
		"claude-3-sonnet":  "claude-3-sonnet",
	}

	if mapped, ok := modelMap[model]; ok {
		return mapped
	}

	// Default to the model name as-is
	return model
}

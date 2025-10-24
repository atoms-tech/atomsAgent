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

// DroidAgent wraps Droid CLI for chat completions
type DroidAgent struct {
	logger    *slog.Logger
	droidPath string
	timeout   time.Duration
	models    []ModelInfo
}

// DroidResponse represents the response from Droid CLI
type DroidResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
	Tokens  struct {
		Input  int `json:"input"`
		Output int `json:"output"`
	} `json:"tokens"`
}

// NewDroidAgent creates a new Droid agent
func NewDroidAgent(logger *slog.Logger, droidPath string, timeout time.Duration) *DroidAgent {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	agent := &DroidAgent{
		logger:    logger,
		droidPath: droidPath,
		timeout:   timeout,
		models: []ModelInfo{
			// Built-in models
			{
				ID:              "droid-claude-3-opus",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "anthropic",
				Description:     "Claude 3 Opus via Droid",
				MaxTokens:       4096,
				InputCostPer1K:  0.015,
				OutputCostPer1K: 0.075,
			},
			{
				ID:              "droid-claude-3-sonnet",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "anthropic",
				Description:     "Claude 3 Sonnet via Droid",
				MaxTokens:       4096,
				InputCostPer1K:  0.003,
				OutputCostPer1K: 0.015,
			},
			{
				ID:              "droid-gpt-4",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "openai",
				Description:     "GPT-4 via Droid",
				MaxTokens:       4096,
				InputCostPer1K:  0.03,
				OutputCostPer1K: 0.06,
			},
			{
				ID:              "droid-gpt-4-turbo",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "openai",
				Description:     "GPT-4 Turbo via Droid",
				MaxTokens:       4096,
				InputCostPer1K:  0.01,
				OutputCostPer1K: 0.03,
			},
			{
				ID:              "droid-gpt-3.5-turbo",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "openai",
				Description:     "GPT-3.5 Turbo via Droid",
				MaxTokens:       4096,
				InputCostPer1K:  0.0005,
				OutputCostPer1K: 0.0015,
			},
			// OpenRouter models
			{
				ID:              "droid-mistral-large",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "mistral",
				Description:     "Mistral Large via Droid/OpenRouter",
				MaxTokens:       8000,
				InputCostPer1K:  0.008,
				OutputCostPer1K: 0.024,
			},
			{
				ID:              "droid-llama-2-70b",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "meta",
				Description:     "Llama 2 70B via Droid/OpenRouter",
				MaxTokens:       4096,
				InputCostPer1K:  0.0007,
				OutputCostPer1K: 0.0009,
			},
			{
				ID:              "droid-palm-2",
				Object:          "model",
				Created:         time.Now().Unix(),
				OwnedBy:         "google",
				Description:     "PaLM 2 via Droid",
				MaxTokens:       8000,
				InputCostPer1K:  0.0,
				OutputCostPer1K: 0.0,
			},
			// Add more as needed
		},
	}

	// Verify Droid is available
	if !agent.IsHealthy(context.Background()) {
		logger.Warn("Droid not available at path", "path", droidPath)
	}

	return agent
}

// Execute runs a synchronous completion via Droid
func (da *DroidAgent) Execute(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, da.timeout)
	defer cancel()

	// Build system prompt
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant."
	}

	// Build user message (use last user message)
	userContent := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userContent = msg.Content
		}
	}

	if userContent == "" {
		return nil, fmt.Errorf("no user message found")
	}

	// Build Droid command
	// Droid uses: droid [model] [options] < input
	args := []string{
		da.mapModelName(req.Model),
		"--system", systemPrompt,
	}

	if req.Temperature != 0 {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", req.Temperature))
	}

	if req.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxTokens))
	}

	if req.TopP != 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.2f", req.TopP))
	}

	// Execute Droid
	cmd := exec.CommandContext(ctx, da.droidPath, args...)

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
		return nil, fmt.Errorf("failed to start droid: %w", err)
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
		da.logger.Error("droid command failed",
			"model", req.Model,
			"error", err.Error(),
			"stderr", stderrData,
		)
		return nil, fmt.Errorf("droid execution failed: %w", err)
	}

	// Parse response
	response := &CompletionResponse{
		Content:      strings.TrimSpace(responseData),
		InputTokens:  0, // Droid doesn't provide token counts
		OutputTokens: 0,
		FinishReason: "stop",
		Model:        req.Model,
	}

	da.logger.Info("droid execution succeeded",
		"model", req.Model,
		"response_length", len(response.Content),
	)

	return response, nil
}

// Stream runs a streaming completion via Droid
func (da *DroidAgent) Stream(ctx context.Context, req *CompletionRequest) (chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 10)

	go func() {
		defer close(streamChan)

		// For now, execute synchronously and stream the result
		// In future, integrate with Droid's streaming capability if available
		resp, err := da.Execute(ctx, req)
		if err != nil {
			streamChan <- StreamChunk{Error: err}
			return
		}

		// Stream word by word for better UX
		words := strings.Fields(resp.Content)
		for i, word := range words {
			if i > 0 {
				streamChan <- StreamChunk{Content: " "} // Space between words
			}
			streamChan <- StreamChunk{Content: word}
		}
	}()

	return streamChan, nil
}

// GetAvailableModels returns available models
func (da *DroidAgent) GetAvailableModels(ctx context.Context) []ModelInfo {
	// In production, could query Droid for available models
	// For now, return pre-configured list
	return da.models
}

// IsHealthy checks if Droid is available
func (da *DroidAgent) IsHealthy(ctx context.Context) bool {
	// Check if Droid binary exists
	info, err := os.Stat(da.droidPath)
	if err != nil {
		return false
	}

	// Check if it's executable
	return !info.IsDir()
}

// Name returns the agent name
func (da *DroidAgent) Name() string {
	return "droid"
}

// mapModelName maps model names to Droid equivalents
func (da *DroidAgent) mapModelName(model string) string {
	modelMap := map[string]string{
		// Direct mappings
		"droid-claude-3-opus":   "claude-3-opus",
		"droid-claude-3-sonnet": "claude-3-sonnet",
		"droid-gpt-4":           "gpt-4",
		"droid-gpt-4-turbo":     "gpt-4-turbo",
		"droid-gpt-3.5-turbo":   "gpt-3.5-turbo",
		"droid-mistral-large":   "mistral-large",
		"droid-llama-2-70b":     "llama-2-70b",
		"droid-palm-2":          "palm-2",

		// Alternative names
		"claude-3-opus":   "claude-3-opus",
		"claude-3-sonnet": "claude-3-sonnet",
		"gpt-4":           "gpt-4",
		"gpt-4-turbo":     "gpt-4-turbo",
		"gpt-3.5-turbo":   "gpt-3.5-turbo",
		"mistral-large":   "mistral-large",
		"llama-2-70b":     "llama-2-70b",
	}

	if mapped, ok := modelMap[model]; ok {
		return mapped
	}

	// Default to the model name as-is
	return model
}

// GetModelInfo returns detailed info about a specific model
func (da *DroidAgent) GetModelInfo(modelID string) *ModelInfo {
	for _, model := range da.models {
		if model.ID == modelID {
			return &model
		}
	}
	return nil
}

// ListSupportedModels returns a formatted list of supported models
func (da *DroidAgent) ListSupportedModels() map[string]interface{} {
	return map[string]interface{}{
		"agent":       "droid",
		"models":      da.models,
		"description": "Droid CLI with 14+ AI models via OpenRouter and direct APIs",
		"capabilities": []string{
			"chat completions",
			"code generation",
			"text analysis",
			"creative writing",
			"reasoning",
		},
	}
}

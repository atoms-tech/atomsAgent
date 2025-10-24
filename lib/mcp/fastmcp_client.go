package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// FastMCPClient wraps the Python FastMCP client for Go
type FastMCPClient struct {
	process *exec.Cmd
	stdin   *bufio.Writer
	stdout  *bufio.Scanner
	mutex   sync.RWMutex
	clients map[string]bool // Track connected MCPs
}

// NewFastMCPClient creates a new FastMCP client
func NewFastMCPClient() (*FastMCPClient, error) {
	// Start the Python FastMCP wrapper process
	cmd := exec.Command("python3", "lib/mcp/fastmcp_wrapper.py")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start FastMCP wrapper: %w", err)
	}

	client := &FastMCPClient{
		process: cmd,
		stdin:   bufio.NewWriter(stdin),
		stdout:  bufio.NewScanner(stdout),
		clients: make(map[string]bool),
	}

	// Start goroutine to handle responses
	go client.handleResponses()

	return client, nil
}

// ConnectMCP connects to an MCP server using FastMCP
func (c *FastMCPClient) ConnectMCP(ctx context.Context, config MCPConfig) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	command := map[string]any{
		"action": "connect",
		"config": map[string]any{
			"id":        config.ID,
			"name":      config.Name,
			"type":      config.Type,
			"endpoint":  config.Endpoint,
			"auth_type": config.AuthType,
			"config":    config.Config,
			"auth":      config.Auth,
		},
	}

	if err := c.sendCommand(command); err != nil {
		return fmt.Errorf("failed to send connect command: %w", err)
	}

	// Wait for response
	response, err := c.waitForResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connect response: %w", err)
	}

	var result struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse connect response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to connect to MCP server")
	}

	c.clients[config.ID] = true
	return nil
}

// DisconnectMCP disconnects from an MCP server
func (c *FastMCPClient) DisconnectMCP(ctx context.Context, mcpID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	command := map[string]any{
		"action": "disconnect",
		"mcp_id": mcpID,
	}

	if err := c.sendCommand(command); err != nil {
		return fmt.Errorf("failed to send disconnect command: %w", err)
	}

	// Wait for response
	response, err := c.waitForResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to get disconnect response: %w", err)
	}

	var result struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("failed to parse disconnect response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to disconnect from MCP server")
	}

	delete(c.clients, mcpID)
	return nil
}

// ListTools lists available tools from an MCP server
func (c *FastMCPClient) ListTools(ctx context.Context, mcpID string) ([]Tool, error) {
	command := map[string]any{
		"action": "list_tools",
		"mcp_id": mcpID,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send list_tools command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get list_tools response: %w", err)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list_tools response: %w", err)
	}

	return result.Tools, nil
}

// CallTool calls a tool on an MCP server
func (c *FastMCPClient) CallTool(ctx context.Context, mcpID, toolName string, arguments map[string]any) (map[string]any, error) {
	command := map[string]any{
		"action":    "call_tool",
		"mcp_id":    mcpID,
		"tool_name": toolName,
		"arguments": arguments,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send call_tool command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get call_tool response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse call_tool response: %w", err)
	}

	return result, nil
}

// ListResources lists available resources from an MCP server
func (c *FastMCPClient) ListResources(ctx context.Context, mcpID string) ([]Resource, error) {
	command := map[string]any{
		"action": "list_resources",
		"mcp_id": mcpID,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send list_resources command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get list_resources response: %w", err)
	}

	var result struct {
		Resources []Resource `json:"resources"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list_resources response: %w", err)
	}

	return result.Resources, nil
}

// ReadResource reads a resource from an MCP server
func (c *FastMCPClient) ReadResource(ctx context.Context, mcpID, uri string) (map[string]any, error) {
	command := map[string]any{
		"action": "read_resource",
		"mcp_id": mcpID,
		"uri":    uri,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send read_resource command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get read_resource response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse read_resource response: %w", err)
	}

	return result, nil
}

// ListPrompts lists available prompts from an MCP server
func (c *FastMCPClient) ListPrompts(ctx context.Context, mcpID string) ([]Prompt, error) {
	command := map[string]any{
		"action": "list_prompts",
		"mcp_id": mcpID,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send list_prompts command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get list_prompts response: %w", err)
	}

	var result struct {
		Prompts []Prompt `json:"prompts"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list_prompts response: %w", err)
	}

	return result.Prompts, nil
}

// GetPrompt gets a prompt from an MCP server
func (c *FastMCPClient) GetPrompt(ctx context.Context, mcpID, promptName string, arguments map[string]any) (map[string]any, error) {
	command := map[string]any{
		"action":      "get_prompt",
		"mcp_id":      mcpID,
		"prompt_name": promptName,
		"arguments":   arguments,
	}

	if err := c.sendCommand(command); err != nil {
		return nil, fmt.Errorf("failed to send get_prompt command: %w", err)
	}

	response, err := c.waitForResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get get_prompt response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse get_prompt response: %w", err)
	}

	return result, nil
}

// Close closes the FastMCP client
func (c *FastMCPClient) Close() error {
	if c.process != nil {
		return c.process.Process.Kill()
	}
	return nil
}

// IsConnected checks if an MCP is connected
func (c *FastMCPClient) IsConnected(mcpID string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.clients[mcpID]
}

// sendCommand sends a command to the Python process
func (c *FastMCPClient) sendCommand(command map[string]any) error {
	data, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	return c.stdin.Flush()
}

// waitForResponse waits for a response from the Python process
func (c *FastMCPClient) waitForResponse(ctx context.Context) ([]byte, error) {
	// Simple timeout mechanism
	done := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		if c.stdout.Scan() {
			done <- []byte(c.stdout.Text())
		} else {
			errChan <- fmt.Errorf("failed to read response")
		}
	}()

	select {
	case response := <-done:
		return response, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// handleResponses handles responses from the Python process
func (c *FastMCPClient) handleResponses() {
	// This is a placeholder - in a real implementation,
	// you might want to handle responses asynchronously
}

package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Tool represents an MCP tool
type Tool struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Prompt represents an MCP prompt
type Prompt struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Arguments   map[string]any `json:"arguments,omitempty"`
}

// MCPConfig represents the configuration for connecting to an MCP server
type MCPConfig struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"` // http, sse, stdio
	Endpoint string            `json:"endpoint"`
	AuthType string            `json:"auth_type,omitempty"`
	Config   map[string]any    `json:"config,omitempty"`
	Auth     map[string]string `json:"auth,omitempty"`
}

// Client represents an MCP client wrapper using FastMCP
type Client struct {
	ID       string
	Name     string
	Type     string // http, sse, stdio
	Endpoint string
	Config   map[string]any
	Auth     map[string]string

	// FastMCP client
	fastMCPClient *FastMCPClient

	// Connection state
	connected bool
	lastError error
	mutex     sync.RWMutex
}

// NewClient creates a new MCP client using FastMCP
func NewClient(id, name, mcpType, endpoint string, config map[string]any, auth map[string]string) (*Client, error) {
	fastMCPClient, err := NewFastMCPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create FastMCP client: %w", err)
	}

	return &Client{
		ID:            id,
		Name:          name,
		Type:          mcpType,
		Endpoint:      endpoint,
		Config:        config,
		Auth:          auth,
		fastMCPClient: fastMCPClient,
	}, nil
}

// Connect establishes connection to the MCP server using FastMCP
func (c *Client) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	config := MCPConfig{
		ID:       c.ID,
		Name:     c.Name,
		Type:     c.Type,
		Endpoint: c.Endpoint,
		AuthType: c.getAuthType(),
		Config:   c.Config,
		Auth:     c.Auth,
	}

	if err := c.fastMCPClient.ConnectMCP(ctx, config); err != nil {
		c.lastError = err
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	c.connected = true
	return nil
}

// getAuthType determines the authentication type from config
func (c *Client) getAuthType() string {
	if c.Auth["token"] != "" {
		return "bearer"
	}
	if c.Auth["client_id"] != "" {
		return "oauth"
	}
	return "none"
}

// CallTool calls a tool on the MCP server using FastMCP
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]any) (map[string]any, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	return c.fastMCPClient.CallTool(ctx, c.ID, toolName, arguments)
}

// ListTools lists available tools from the MCP server using FastMCP
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	return c.fastMCPClient.ListTools(ctx, c.ID)
}

// GetResources lists available resources from the MCP server using FastMCP
func (c *Client) GetResources(ctx context.Context) ([]Resource, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	return c.fastMCPClient.ListResources(ctx, c.ID)
}

// ReadResource reads a resource from the MCP server using FastMCP
func (c *Client) ReadResource(ctx context.Context, uri string) (map[string]any, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	return c.fastMCPClient.ReadResource(ctx, c.ID, uri)
}

// Disconnect closes the MCP connection using FastMCP
func (c *Client) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		if err := c.fastMCPClient.DisconnectMCP(context.Background(), c.ID); err != nil {
			return fmt.Errorf("failed to disconnect MCP client: %w", err)
		}
		c.connected = false
	}

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetLastError returns the last connection error
func (c *Client) GetLastError() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastError
}

// Close closes the underlying FastMCP client
func (c *Client) Close() error {
	return c.fastMCPClient.Close()
}

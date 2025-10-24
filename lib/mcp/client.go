package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client represents an MCP client wrapper
type Client struct {
	ID       string
	Name     string
	Type     string // http, sse, stdio
	Endpoint string
	Config   map[string]any
	Auth     map[string]string
	
	// MCP client components
	client    *mcp.Client
	session   *mcp.Session
	transport mcp.Transport
	
	// Connection state
	connected bool
	lastError error
}

// NewClient creates a new MCP client
func NewClient(id, name, mcpType, endpoint string, config map[string]any, auth map[string]string) *Client {
	return &Client{
		ID:       id,
		Name:     name,
		Type:     mcpType,
		Endpoint: endpoint,
		Config:   config,
		Auth:     auth,
	}
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	switch c.Type {
	case "http":
		return c.connectHTTP(ctx)
	case "sse":
		return c.connectSSE(ctx)
	case "stdio":
		return c.connectStdio(ctx)
	default:
		return fmt.Errorf("unsupported MCP type: %s", c.Type)
	}
}

// connectHTTP connects to an HTTP-based MCP server
func (c *Client) connectHTTP(ctx context.Context) error {
	// Create HTTP transport
	transport := &HTTPTransport{
		Endpoint: c.Endpoint,
		Auth:     c.Auth,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
	
	// Create MCP client
	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    c.Name,
		Version: "1.0.0",
	}, nil)
	
	// Connect
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		c.lastError = err
		return fmt.Errorf("failed to connect to HTTP MCP server: %w", err)
	}
	
	c.session = session
	c.transport = transport
	c.connected = true
	return nil
}

// connectSSE connects to an SSE-based MCP server
func (c *Client) connectSSE(ctx context.Context) error {
	// Create SSE transport
	transport := &SSETransport{
		Endpoint: c.Endpoint,
		Auth:     c.Auth,
	}
	
	// Create MCP client
	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    c.Name,
		Version: "1.0.0",
	}, nil)
	
	// Connect
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		c.lastError = err
		return fmt.Errorf("failed to connect to SSE MCP server: %w", err)
	}
	
	c.session = session
	c.transport = transport
	c.connected = true
	return nil
}

// connectStdio connects to a stdio-based MCP server
func (c *Client) connectStdio(ctx context.Context) error {
	// Parse command from config
	command, ok := c.Config["command"].(string)
	if !ok {
		return fmt.Errorf("stdio MCP requires 'command' in config")
	}
	
	// Create command transport
	cmd := exec.CommandContext(ctx, command)
	transport := &mcp.CommandTransport{Command: cmd}
	
	// Create MCP client
	c.client = mcp.NewClient(&mcp.Implementation{
		Name:    c.Name,
		Version: "1.0.0",
	}, nil)
	
	// Connect
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		c.lastError = err
		return fmt.Errorf("failed to connect to stdio MCP server: %w", err)
	}
	
	c.session = session
	c.transport = transport
	c.connected = true
	return nil
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("MCP client not connected")
	}
	
	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}
	
	return c.session.CallTool(ctx, params)
}

// ListTools lists available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("MCP client not connected")
	}
	
	return c.session.ListTools(ctx)
}

// GetResources lists available resources from the MCP server
func (c *Client) GetResources(ctx context.Context) ([]*mcp.Resource, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("MCP client not connected")
	}
	
	return c.session.ListResources(ctx)
}

// ReadResource reads a resource from the MCP server
func (c *Client) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("MCP client not connected")
	}
	
	params := &mcp.ReadResourceParams{URI: uri}
	return c.session.ReadResource(ctx, params)
}

// Disconnect closes the MCP connection
func (c *Client) Disconnect() error {
	if c.session != nil {
		c.session.Close()
	}
	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected
}

// GetLastError returns the last connection error
func (c *Client) GetLastError() error {
	return c.lastError
}

// HTTPTransport implements MCP transport over HTTP
type HTTPTransport struct {
	Endpoint string
	Auth     map[string]string
	Client   *http.Client
}

func (t *HTTPTransport) Send(ctx context.Context, message json.RawMessage) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", t.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	// Add authentication headers
	for key, value := range t.Auth {
		req.Header.Set(key, value)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}
	
	return json.RawMessage(body), nil
}

// SSETransport implements MCP transport over Server-Sent Events
type SSETransport struct {
	Endpoint string
	Auth     map[string]string
}

func (t *SSETransport) Send(ctx context.Context, message json.RawMessage) (json.RawMessage, error) {
	// TODO: Implement SSE transport
	return nil, fmt.Errorf("SSE transport not yet implemented")
}
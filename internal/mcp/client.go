// ABOUTME: MCP client implementing initialize handshake, tool listing, and tool calling
// ABOUTME: Handles resource listing/reading and notifications/tools/list_changed

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Client communicates with a single MCP server.
type Client struct {
	transport  Transport
	serverCaps ServerCapabilities
	serverInfo ServerInfo
	tools      []MCPTool
	resources  []Resource

	mu        sync.RWMutex
	connected bool

	ctx    context.Context
	cancel context.CancelFunc
	// toolSem limits concurrent ListTools refreshes to 1.
	toolSem chan struct{}
}

// NewClient creates a new MCP client with the given transport.
func NewClient(transport Transport) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		transport: transport,
		ctx:       ctx,
		cancel:    cancel,
		toolSem:   make(chan struct{}, 1),
	}
}

// Connect performs the MCP initialize handshake.
func (c *Client) Connect(ctx context.Context) error {
	params, _ := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "pi-go",
			"version": "1.0.0",
		},
	})

	resp, err := c.transport.Send(ctx, &Request{
		Method: "initialize",
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parsing initialize result: %w", err)
	}

	c.mu.Lock()
	c.serverCaps = result.Capabilities
	c.serverInfo = result.ServerInfo
	c.connected = true
	c.mu.Unlock()

	// Send initialized notification
	if err := c.transport.Notify(ctx, &Notification{Method: "notifications/initialized"}); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}

	// Start listening for notifications
	go c.handleNotifications()

	return nil
}

// ListTools requests the tool list from the server.
func (c *Client) ListTools(ctx context.Context) ([]MCPTool, error) {
	resp, err := c.transport.Send(ctx, &Request{Method: "tools/list"})
	if err != nil {
		return nil, fmt.Errorf("tools/list request: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing tools list: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return result.Tools, nil
}

// CallTool invokes a tool on the server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (ToolCallResult, error) {
	params, _ := json.Marshal(map[string]any{
		"name":      name,
		"arguments": args,
	})

	resp, err := c.transport.Send(ctx, &Request{
		Method: "tools/call",
		Params: params,
	})
	if err != nil {
		return ToolCallResult{}, fmt.Errorf("tools/call request: %w", err)
	}
	if resp.Error != nil {
		return ToolCallResult{IsError: true, Content: []ContentItem{
			{Type: "text", Text: resp.Error.Message},
		}}, nil
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return ToolCallResult{}, fmt.Errorf("parsing tool result: %w", err)
	}
	return result, nil
}

// ListResources requests the resource list from the server.
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	resp, err := c.transport.Send(ctx, &Request{Method: "resources/list"})
	if err != nil {
		return nil, fmt.Errorf("resources/list request: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("resources/list error: %s", resp.Error.Message)
	}

	var result struct {
		Resources []Resource `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing resources list: %w", err)
	}

	c.mu.Lock()
	c.resources = result.Resources
	c.mu.Unlock()

	return result.Resources, nil
}

// ReadResource reads a resource from the server.
func (c *Client) ReadResource(ctx context.Context, uri string) (ResourceContent, error) {
	params, _ := json.Marshal(map[string]any{"uri": uri})

	resp, err := c.transport.Send(ctx, &Request{
		Method: "resources/read",
		Params: params,
	})
	if err != nil {
		return ResourceContent{}, fmt.Errorf("resources/read request: %w", err)
	}
	if resp.Error != nil {
		return ResourceContent{}, fmt.Errorf("resources/read error: %s", resp.Error.Message)
	}

	var result struct {
		Contents []ResourceContent `json:"contents"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return ResourceContent{}, fmt.Errorf("parsing resource content: %w", err)
	}

	if len(result.Contents) == 0 {
		return ResourceContent{}, fmt.Errorf("empty resource content for %q", uri)
	}
	return result.Contents[0], nil
}

// Tools returns the cached tool list.
func (c *Client) Tools() []MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// ServerInfo returns the server information from the handshake.
func (c *Client) ServerInfo() ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// Close shuts down the client and transport.
func (c *Client) Close() error {
	c.cancel()
	return c.transport.Close()
}

// handleNotifications processes incoming notifications.
// Uses the client context for cancellation and a semaphore to prevent concurrent ListTools.
func (c *Client) handleNotifications() {
	for msg := range c.transport.Receive() {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		var notif Notification
		if err := json.Unmarshal(msg, &notif); err != nil {
			continue
		}

		switch notif.Method {
		case "notifications/tools/list_changed":
			// Acquire semaphore; skip if another refresh is already running.
			select {
			case c.toolSem <- struct{}{}:
			default:
				continue
			}
			go func() {
				defer func() { <-c.toolSem }()
				ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
				defer cancel()
				_, _ = c.ListTools(ctx)
			}()
		}
	}
}

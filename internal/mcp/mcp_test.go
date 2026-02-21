// ABOUTME: Tests for MCP client, config loading, bridge, and server
// ABOUTME: Uses mock transport for client tests and temp dirs for config tests

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// mockTransport implements Transport for testing without spawning processes.
type mockTransport struct {
	mu       sync.Mutex
	handler  func(*Request) *Response
	incoming chan json.RawMessage
	closed   bool
}

func newMockTransport(handler func(*Request) *Response) *mockTransport {
	return &mockTransport{
		handler:  handler,
		incoming: make(chan json.RawMessage, 16),
	}
}

func (m *mockTransport) Send(_ context.Context, req *Request) (*Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.handler(req), nil
}

func (m *mockTransport) Notify(_ context.Context, _ *Notification) error {
	return nil
}

func (m *mockTransport) Receive() <-chan json.RawMessage {
	return m.incoming
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	close(m.incoming)
	return nil
}

func TestClient_Connect(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities:    ServerCapabilities{Tools: &ToolsCapability{ListChanged: true}},
				ServerInfo:      ServerInfo{Name: "test-server", Version: "1.0"},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown method"}}
		}
	})

	c := NewClient(mt)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	info := c.ServerInfo()
	if info.Name != "test-server" {
		t.Errorf("expected test-server, got %q", info.Name)
	}
}

func TestClient_ListTools(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      ServerInfo{Name: "test"},
			})
			return &Response{ID: req.ID, Result: result}
		case "tools/list":
			result, _ := json.Marshal(map[string]any{
				"tools": []MCPTool{
					{Name: "read_file", Description: "Reads a file"},
					{Name: "write_file", Description: "Writes a file"},
				},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())

	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "read_file" {
		t.Errorf("expected read_file, got %q", tools[0].Name)
	}
}

func TestClient_CallTool(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{ProtocolVersion: "2024-11-05"})
			return &Response{ID: req.ID, Result: result}
		case "tools/call":
			result, _ := json.Marshal(ToolCallResult{
				Content: []ContentItem{{Type: "text", Text: "file contents here"}},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())

	result, err := c.CallTool(context.Background(), "read_file", map[string]any{"path": "test.go"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "file contents here" {
		t.Errorf("unexpected content: %q", result.Content[0].Text)
	}
}

func TestClient_CallTool_Error(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{ProtocolVersion: "2024-11-05"})
			return &Response{ID: req.ID, Result: result}
		case "tools/call":
			return &Response{ID: req.ID, Error: &RPCError{Code: -32000, Message: "tool error"}}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())

	result, err := c.CallTool(context.Background(), "bad_tool", nil)
	if err != nil {
		t.Fatalf("CallTool should not return error for RPC error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError to be true")
	}
}

func TestClient_ListResources(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{ProtocolVersion: "2024-11-05"})
			return &Response{ID: req.ID, Result: result}
		case "resources/list":
			result, _ := json.Marshal(map[string]any{
				"resources": []Resource{
					{URI: "file:///test.go", Name: "test.go"},
				},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())

	resources, err := c.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
}

// --- Config tests ---

func TestLoadConfig_Empty(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	cfg := LoadConfig(project, home)
	if len(cfg) != 0 {
		t.Errorf("expected empty config, got %d entries", len(cfg))
	}
}

func TestLoadConfig_MCPJson(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	mcpJSON := `{"mcpServers": {"test": {"command": "echo", "args": ["hello"]}}}`
	writeTestFile(t, filepath.Join(project, ".mcp.json"), mcpJSON)

	cfg := LoadConfig(project, home)
	if len(cfg) != 1 {
		t.Fatalf("expected 1 server, got %d", len(cfg))
	}
	if cfg["test"].Command != "echo" {
		t.Errorf("expected echo command, got %q", cfg["test"].Command)
	}
}

func TestLoadConfig_SettingsJSON(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	mkTestDir(t, filepath.Join(home, ".pi-go"))
	settingsJSON := `{"mcpServers": {"global": {"command": "global-server"}}}`
	writeTestFile(t, filepath.Join(home, ".pi-go", "settings.json"), settingsJSON)

	cfg := LoadConfig(project, home)
	if len(cfg) != 1 {
		t.Fatalf("expected 1 server, got %d", len(cfg))
	}
	if cfg["global"].Command != "global-server" {
		t.Errorf("expected global-server, got %q", cfg["global"].Command)
	}
}

func TestLoadConfig_Override(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	mkTestDir(t, filepath.Join(home, ".pi-go"))
	writeTestFile(t, filepath.Join(home, ".pi-go", "settings.json"),
		`{"mcpServers": {"srv": {"command": "old"}}}`)

	writeTestFile(t, filepath.Join(project, ".mcp.json"),
		`{"mcpServers": {"srv": {"command": "new"}}}`)

	cfg := LoadConfig(project, home)
	if cfg["srv"].Command != "new" {
		t.Errorf("project .mcp.json should override global; got %q", cfg["srv"].Command)
	}
}

func TestLoadConfig_ClaudeCompat(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()

	mkTestDir(t, filepath.Join(home, ".claude"))
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"),
		`{"mcpServers": {"claude-srv": {"command": "claude-cmd"}}}`)

	cfg := LoadConfig(project, home)
	if cfg["claude-srv"].Command != "claude-cmd" {
		t.Errorf("Claude compat should be loaded; got %q", cfg["claude-srv"].Command)
	}
}

// --- Bridge tests ---

func TestBridgeTool_Name(t *testing.T) {
	tool := MCPTool{Name: "read-file", Description: "Reads a file"}
	c := NewClient(newMockTransport(nil))

	bridged := BridgeTool("my-server", tool, c)
	if bridged.Name != "mcp__my_server__read_file" {
		t.Errorf("unexpected bridged name: %q", bridged.Name)
	}
}

func TestBridgeAllTools(t *testing.T) {
	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{ProtocolVersion: "2024-11-05"})
			return &Response{ID: req.ID, Result: result}
		case "tools/list":
			result, _ := json.Marshal(map[string]any{
				"tools": []MCPTool{
					{Name: "tool_a"},
					{Name: "tool_b"},
				},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())
	_, _ = c.ListTools(context.Background())

	bridged := BridgeAllTools("test", c)
	if len(bridged) != 2 {
		t.Fatalf("expected 2 bridged tools, got %d", len(bridged))
	}
}

func TestBridgeTool_TextCap(t *testing.T) {
	oversized := make([]byte, maxBridgeTextBytes+100)
	for i := range oversized {
		oversized[i] = 'A'
	}

	mt := newMockTransport(func(req *Request) *Response {
		switch req.Method {
		case "initialize":
			result, _ := json.Marshal(InitializeResult{ProtocolVersion: "2024-11-05"})
			return &Response{ID: req.ID, Result: result}
		case "tools/call":
			result, _ := json.Marshal(ToolCallResult{
				Content: []ContentItem{{Type: "text", Text: string(oversized)}},
			})
			return &Response{ID: req.ID, Result: result}
		default:
			return &Response{ID: req.ID, Error: &RPCError{Code: -32601, Message: "unknown"}}
		}
	})

	c := NewClient(mt)
	_ = c.Connect(context.Background())

	tool := MCPTool{Name: "big-tool", Description: "returns oversized text"}
	bridged := BridgeTool("test", tool, c)

	result, err := bridged.Execute(context.Background(), "1", nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for oversized response")
	}
	if !strings.Contains(result.Content, "exceeded") {
		t.Errorf("expected truncation message, got %q", result.Content[:100])
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with-dashes", "with_dashes"},
		{"with.dots", "with_dots"},
		{"with spaces", "with_spaces"},
		{"CamelCase", "CamelCase"},
		{"under_score", "under_score"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Server tests ---

func TestServer_HandleInitialize(t *testing.T) {
	tools := map[string]*agent.AgentTool{}
	s := NewServer(tools)

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	// Override writer to capture output
	var buf testBuffer
	s.writer = &buf

	s.handleRequest(context.Background(), req)

	var resp Response
	if err := json.Unmarshal(buf.data, &resp); err != nil {
		t.Fatalf("parsing response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %s", resp.Error.Message)
	}
	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
}

func TestServer_HandleToolsList(t *testing.T) {
	tools := map[string]*agent.AgentTool{
		"read": {Name: "read", Description: "Read a file"},
	}
	s := NewServer(tools)

	var buf testBuffer
	s.writer = &buf

	s.handleRequest(context.Background(), &Request{ID: 2, Method: "tools/list"})

	var resp Response
	if err := json.Unmarshal(buf.data, &resp); err != nil {
		t.Fatalf("parsing response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("parsing tools: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
}

func TestServer_HandleToolsCall(t *testing.T) {
	tools := map[string]*agent.AgentTool{
		"echo": {
			Name: "echo",
			Execute: func(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
				msg, _ := params["message"].(string)
				return agent.ToolResult{Content: "echo: " + msg}, nil
			},
		},
	}
	s := NewServer(tools)

	var buf testBuffer
	s.writer = &buf

	callParams, _ := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"message": "hello"},
	})
	s.handleRequest(context.Background(), &Request{ID: 3, Method: "tools/call", Params: callParams})

	var resp Response
	if err := json.Unmarshal(buf.data, &resp); err != nil {
		t.Fatalf("parsing response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("parsing result: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text != "echo: hello" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	s := NewServer(map[string]*agent.AgentTool{})

	var buf testBuffer
	s.writer = &buf

	s.handleRequest(context.Background(), &Request{ID: 4, Method: "unknown/method"})

	var resp Response
	if err := json.Unmarshal(buf.data, &resp); err != nil {
		t.Fatalf("parsing response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
}

func TestServerConfigEnv(t *testing.T) {
	cfg := ServerConfig{
		Command: "test",
		Env:     map[string]string{"KEY": "VALUE"},
	}
	env := ServerConfigEnv(cfg)
	if len(env) != 1 || env[0] != "KEY=VALUE" {
		t.Errorf("unexpected env: %v", env)
	}
}

func TestRPCError_Error(t *testing.T) {
	err := &RPCError{Code: -32600, Message: "invalid request"}
	if err.Error() != "invalid request" {
		t.Errorf("expected 'invalid request', got %q", err.Error())
	}
}

// --- Helpers ---

type testBuffer struct {
	data []byte
}

func (b *testBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data[:0], p...)
	return len(p), nil
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

func mkTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

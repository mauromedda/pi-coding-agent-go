// ABOUTME: MCP server that exposes pi-go tools via JSON-RPC over stdin/stdout
// ABOUTME: Handles initialize, tools/list, tools/call, and resources/list methods

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// Server exposes pi-go tools as an MCP server.
type Server struct {
	tools  map[string]*agent.AgentTool
	reader *bufio.Scanner
	writer io.Writer
}

// NewServer creates an MCP server backed by the given tools.
func NewServer(tools map[string]*agent.AgentTool) *Server {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

	return &Server{
		tools:  tools,
		reader: scanner,
		writer: os.Stdout,
	}
}

// Serve reads JSON-RPC messages from stdin and dispatches them.
func (s *Server) Serve(ctx context.Context) error {
	for s.reader.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := s.reader.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(0, -32700, "Parse error")
			continue
		}

		s.handleRequest(ctx, &req)
	}

	return s.reader.Err()
}

func (s *Server) handleRequest(ctx context.Context, req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "resources/list":
		s.handleResourcesList(req)
	case "notifications/initialized":
		// ACK; no response needed
	default:
		s.writeError(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "pi-go",
			Version: "1.0.0",
		},
	}
	s.writeResult(req.ID, result)
}

func (s *Server) handleToolsList(req *Request) {
	tools := make([]MCPTool, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, MCPTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	s.writeResult(req.ID, map[string]any{"tools": tools})
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, "invalid params")
		return
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		s.writeError(req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
		return
	}

	result, err := tool.Execute(ctx, fmt.Sprintf("%d", req.ID), params.Arguments, nil)
	if err != nil {
		s.writeError(req.ID, -32000, err.Error())
		return
	}

	s.writeResult(req.ID, ToolCallResult{
		Content: []ContentItem{{Type: "text", Text: result.Content}},
		IsError: result.IsError,
	})
}

func (s *Server) handleResourcesList(req *Request) {
	s.writeResult(req.ID, map[string]any{"resources": []Resource{}})
}

func (s *Server) writeResult(id int64, result any) {
	data, _ := json.Marshal(result)
	resp := Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  data,
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", out)
}

func (s *Server) writeError(id int64, code int, message string) {
	resp := Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
	out, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", out)
}

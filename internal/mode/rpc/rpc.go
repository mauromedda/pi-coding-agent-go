// ABOUTME: RPC mode for external integrations (IDE extensions)
// ABOUTME: JSONL-based protocol over stdin/stdout

package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Server handles RPC requests from an external client.
type Server struct {
	reader  *bufio.Scanner
	writer  io.Writer
	handler func(Request) Response
}

// NewServer creates an RPC server reading from stdin, writing to stdout.
func NewServer(handler func(Request) Response) *Server {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return &Server{
		reader:  scanner,
		writer:  os.Stdout,
		handler: handler,
	}
}

// Run starts the RPC server loop.
func (s *Server) Run() error {
	for s.reader.Scan() {
		var req Request
		if err := json.Unmarshal(s.reader.Bytes(), &req); err != nil {
			s.sendError("", -32700, fmt.Sprintf("parse error: %v", err))
			continue
		}

		resp := s.handler(req)
		resp.ID = req.ID

		data, err := json.Marshal(resp)
		if err != nil {
			s.sendError(req.ID, -32603, fmt.Sprintf("internal error: %v", err))
			continue
		}

		data = append(data, '\n')
		if _, err := s.writer.Write(data); err != nil {
			return fmt.Errorf("writing response: %w", err)
		}
	}

	return s.reader.Err()
}

func (s *Server) sendError(id string, code int, message string) {
	resp := Response{
		ID:    id,
		Error: &Error{Code: code, Message: message},
	}
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	_, _ = s.writer.Write(data)
}

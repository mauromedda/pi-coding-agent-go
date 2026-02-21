// ABOUTME: Stdio transport for MCP: spawns process and communicates via JSON-RPC over stdin/stdout
// ABOUTME: Uses newline-delimited JSON messages with 10MB scanner buffer

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

const maxScannerBuffer = 10 * 1024 * 1024 // 10MB

// StdioTransport communicates with an MCP server via stdin/stdout of a spawned process.
type StdioTransport struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	scanner *bufio.Scanner

	incoming chan json.RawMessage
	pending  map[int64]chan *Response
	mu       sync.Mutex
	nextID   atomic.Int64
	done     chan struct{}
	closeOnce sync.Once
}

// ApproveFunc validates whether spawning an MCP server command is allowed.
// Return nil to approve; return an error to block.
type ApproveFunc func(command string, args []string) error

// NewStdioTransport creates a transport that spawns the given command.
func NewStdioTransport(ctx context.Context, command string, args []string, env []string) (*StdioTransport, error) {
	return NewStdioTransportWithApproval(ctx, command, args, env, nil)
}

// NewStdioTransportWithApproval creates a transport with an optional approval gate.
// If approveFn is non-nil, it is called before the command is spawned.
func NewStdioTransportWithApproval(ctx context.Context, command string, args []string, env []string, approveFn ApproveFunc) (*StdioTransport, error) {
	if approveFn != nil {
		if err := approveFn(command, args); err != nil {
			return nil, fmt.Errorf("MCP server %q denied: %w", command, err)
		}
	}
	return newStdioTransport(ctx, command, args, env)
}

func newStdioTransport(ctx context.Context, command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting MCP server %q: %w", command, err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

	t := &StdioTransport{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		scanner:  scanner,
		incoming: make(chan json.RawMessage, 64),
		pending:  make(map[int64]chan *Response),
		done:     make(chan struct{}),
	}

	go t.recvLoop()
	return t, nil
}

// Send sends a request and waits for the response.
func (t *StdioTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	req.JSONRPC = jsonRPCVersion
	if req.ID == 0 {
		req.ID = t.nextID.Add(1)
	}

	ch := make(chan *Response, 1)
	t.mu.Lock()
	t.pending[req.ID] = ch
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
	}()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("writing request: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	case <-t.done:
		return nil, fmt.Errorf("transport closed")
	}
}

// Notify sends a notification (no response expected).
func (t *StdioTransport) Notify(_ context.Context, notif *Notification) error {
	notif.JSONRPC = jsonRPCVersion

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshaling notification: %w", err)
	}
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("writing notification: %w", err)
	}
	return nil
}

// Receive returns a channel of incoming notifications.
func (t *StdioTransport) Receive() <-chan json.RawMessage {
	return t.incoming
}

// Close shuts down the transport.
func (t *StdioTransport) Close() error {
	var closeErr error
	t.closeOnce.Do(func() {
		close(t.done)
		t.stdin.Close()
		closeErr = t.cmd.Wait()
	})
	return closeErr
}

// recvLoop reads JSON-RPC messages from stdout and dispatches them.
func (t *StdioTransport) recvLoop() {
	defer close(t.incoming)

	for t.scanner.Scan() {
		line := t.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try to parse as response (has "id" field)
		var resp Response
		if err := json.Unmarshal(line, &resp); err == nil && resp.ID != 0 {
			t.mu.Lock()
			ch, ok := t.pending[resp.ID]
			t.mu.Unlock()
			if ok {
				ch <- &resp
			}
			continue
		}

		// Otherwise treat as notification
		select {
		case t.incoming <- json.RawMessage(append([]byte(nil), line...)):
		default:
			// Drop if buffer full
		}
	}
}

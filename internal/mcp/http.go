// ABOUTME: HTTP/SSE transport for MCP Streamable HTTP protocol
// ABOUTME: Posts JSON-RPC over HTTP; handles application/json and text/event-stream responses

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	headerSessionID = "Mcp-Session-Id"
	headerAccept    = "Accept"
	acceptValue     = "application/json, text/event-stream"
	contentTypeJSON = "application/json"
	contentTypeSSE  = "text/event-stream"

	// maxSSELineSize caps individual SSE line size at 1MB, matching
	// maxBridgeTextBytes from bridge.go.
	maxSSELineSize = 1 << 20
)

// HTTPTransport communicates with an MCP server over HTTP using Streamable HTTP.
type HTTPTransport struct {
	baseURL    string
	httpClient *http.Client
	authToken  string

	sessionID string
	mu        sync.RWMutex

	incoming  chan json.RawMessage
	done      chan struct{}
	closeOnce sync.Once

	// sseCancel stops the background SSE listener goroutine.
	sseCancel context.CancelFunc
	// sseWg tracks the SSE listener goroutine for clean shutdown.
	sseWg sync.WaitGroup
}

// NewHTTPTransport creates a transport that communicates over HTTP/SSE.
func NewHTTPTransport(baseURL, authToken string) *HTTPTransport {
	ctx, cancel := context.WithCancel(context.Background())

	t := &HTTPTransport{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
		authToken:  authToken,
		incoming:   make(chan json.RawMessage, 64),
		done:       make(chan struct{}),
		sseCancel:  cancel,
	}

	t.sseWg.Add(1)
	go t.listenSSE(ctx)
	return t
}

// Send posts a JSON-RPC request and returns the response.
// It handles both application/json and text/event-stream responses.
func (t *HTTPTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	req.JSONRPC = jsonRPCVersion

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	t.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", contentTypeJSON)
	httpReq.Header.Set(headerAccept, acceptValue)

	httpResp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP POST: %w", err)
	}
	defer httpResp.Body.Close()

	t.captureSessionID(httpResp)

	ct := httpResp.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(ct, contentTypeSSE):
		return t.readSSEResponse(httpResp.Body, req.ID)
	default:
		return t.readJSONResponse(httpResp.Body)
	}
}

// Notify sends a JSON-RPC notification (no response expected).
func (t *HTTPTransport) Notify(ctx context.Context, notif *Notification) error {
	notif.JSONRPC = jsonRPCVersion

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshaling notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating HTTP request: %w", err)
	}

	t.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", contentTypeJSON)

	httpResp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP POST notification: %w", err)
	}
	defer httpResp.Body.Close()

	t.captureSessionID(httpResp)

	// Drain body to allow connection reuse.
	_, _ = io.Copy(io.Discard, httpResp.Body)
	return nil
}

// Receive returns a channel of server-initiated notifications from the SSE stream.
func (t *HTTPTransport) Receive() <-chan json.RawMessage {
	return t.incoming
}

// Close shuts down the transport and sends a DELETE to terminate the session.
func (t *HTTPTransport) Close() error {
	var closeErr error
	t.closeOnce.Do(func() {
		// Stop SSE listener goroutine and wait for it to exit.
		t.sseCancel()
		close(t.done)
		t.sseWg.Wait()

		// Safe to close now: the only other writer (readSSEResponse)
		// runs synchronously inside Send, which cannot be called
		// after Close returns.
		close(t.incoming)

		// Send DELETE with session ID if we have one.
		t.mu.RLock()
		sid := t.sessionID
		t.mu.RUnlock()

		if sid != "" {
			req, err := http.NewRequest(http.MethodDelete, t.baseURL, nil)
			if err == nil {
				req.Header.Set(headerSessionID, sid)
				t.setAuthHeader(req)
				resp, err := t.httpClient.Do(req)
				if err != nil {
					closeErr = fmt.Errorf("DELETE session: %w", err)
				} else {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}
	})
	return closeErr
}

// setHeaders sets common headers (auth, session ID) on an outgoing request.
func (t *HTTPTransport) setHeaders(req *http.Request) {
	t.setAuthHeader(req)

	t.mu.RLock()
	sid := t.sessionID
	t.mu.RUnlock()

	if sid != "" {
		req.Header.Set(headerSessionID, sid)
	}
}

// setAuthHeader sets the Authorization header if an auth token is configured.
func (t *HTTPTransport) setAuthHeader(req *http.Request) {
	if t.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.authToken)
	}
}

// captureSessionID stores the session ID from a response header.
func (t *HTTPTransport) captureSessionID(resp *http.Response) {
	if sid := resp.Header.Get(headerSessionID); sid != "" {
		t.mu.Lock()
		t.sessionID = sid
		t.mu.Unlock()
	}
}

// readJSONResponse decodes a standard JSON-RPC response from the body.
func (t *HTTPTransport) readJSONResponse(body io.Reader) (*Response, error) {
	var resp Response
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding JSON response: %w", err)
	}
	return &resp, nil
}

// readSSEResponse reads SSE events from the body, looking for the response
// matching the given request ID. Each event's "data:" lines are parsed as JSON.
func (t *HTTPTransport) readSSEResponse(body io.Reader, reqID int64) (*Response, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSELineSize)

	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Empty line = event boundary.
		if line == "" {
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				dataLines = dataLines[:0]

				if resp, done := t.dispatchSSEData(data, reqID); done {
					return resp, nil
				}
			}
			continue
		}

		// Skip SSE comments.
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse "data: <value>" lines.
		if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			dataLines = append(dataLines, value)
		}
		// Ignore event:, id:, retry: fields for now.
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SSE stream: %w", err)
	}

	// Handle trailing event without final blank line.
	if len(dataLines) > 0 {
		data := strings.Join(dataLines, "\n")
		if resp, done := t.dispatchSSEData(data, reqID); done {
			return resp, nil
		}
	}

	return nil, fmt.Errorf("SSE stream ended without response for ID %d", reqID)
}

// dispatchSSEData attempts to parse data as a JSON-RPC response matching reqID.
// Returns (response, true) if matched, or (nil, false) if it was forwarded as a notification.
func (t *HTTPTransport) dispatchSSEData(data string, reqID int64) (*Response, bool) {
	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err == nil && resp.ID == reqID {
		return &resp, true
	}

	// Forward to incoming channel as raw notification.
	t.trySendIncoming(json.RawMessage(data))
	return nil, false
}

// trySendIncoming sends a message to the incoming channel if the transport
// is not closed, using a non-blocking send to avoid deadlocks.
func (t *HTTPTransport) trySendIncoming(msg json.RawMessage) {
	select {
	case <-t.done:
		return
	default:
	}
	select {
	case t.incoming <- msg:
	default:
	}
}

// listenSSE opens a GET SSE connection for server-initiated notifications.
func (t *HTTPTransport) listenSSE(ctx context.Context) {
	defer t.sseWg.Done()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL, nil)
	if err != nil {
		return
	}

	t.setHeaders(req)
	req.Header.Set(headerAccept, contentTypeSSE)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSELineSize)

	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				dataLines = dataLines[:0]

				t.trySendIncoming(json.RawMessage(data))
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			dataLines = append(dataLines, value)
		}
	}
}

// ABOUTME: Tests for RPC server, router, methods, and error handling
// ABOUTME: Uses pipe-based stdin/stdout mocks for JSONL protocol testing

package rpc

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// --- Error constructor tests ---

func TestNewParseError(t *testing.T) {
	e := NewParseError("bad json")
	if e.Code != ErrCodeParse {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeParse)
	}
	if e.Message != "bad json" {
		t.Errorf("Message = %q; want %q", e.Message, "bad json")
	}
}

func TestNewMethodNotFoundError(t *testing.T) {
	e := NewMethodNotFoundError("bogus")
	if e.Code != ErrCodeMethodNotFound {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeMethodNotFound)
	}
	if !strings.Contains(e.Message, "bogus") {
		t.Errorf("Message = %q; want it to contain %q", e.Message, "bogus")
	}
}

func TestNewInvalidParamsError(t *testing.T) {
	e := NewInvalidParamsError("missing field")
	if e.Code != ErrCodeInvalidParams {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeInvalidParams)
	}
}

func TestNewInternalError(t *testing.T) {
	e := NewInternalError("oops")
	if e.Code != ErrCodeInternal {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeInternal)
	}
}

func TestNewAgentRunningError(t *testing.T) {
	e := NewAgentRunningError()
	if e.Code != ErrCodeAgentRunning {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeAgentRunning)
	}
}

func TestNewNoSessionError(t *testing.T) {
	e := NewNoSessionError()
	if e.Code != ErrCodeNoSession {
		t.Errorf("Code = %d; want %d", e.Code, ErrCodeNoSession)
	}
}

// --- Router tests ---

func TestRouterDispatch(t *testing.T) {
	r := NewRouter()
	r.Register("echo", func(params json.RawMessage) Response {
		return Response{Result: "echoed"}
	})

	resp := r.Handle(Request{ID: "1", Method: "echo"})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.ID != "1" {
		t.Errorf("ID = %q; want %q", resp.ID, "1")
	}
	if resp.Result != "echoed" {
		t.Errorf("Result = %v; want %q", resp.Result, "echoed")
	}
}

func TestRouterMethodNotFound(t *testing.T) {
	r := NewRouter()

	resp := r.Handle(Request{ID: "2", Method: "nonexistent"})
	if resp.Error == nil {
		t.Fatal("expected error; got nil")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Code = %d; want %d", resp.Error.Code, ErrCodeMethodNotFound)
	}
	if resp.ID != "2" {
		t.Errorf("ID = %q; want %q", resp.ID, "2")
	}
}

func TestRouterParamsPassedThrough(t *testing.T) {
	r := NewRouter()
	r.Register("greet", func(params json.RawMessage) Response {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return Response{Error: NewInvalidParamsError(err.Error())}
		}
		return Response{Result: "hello " + p.Name}
	})

	req := Request{
		ID:     "3",
		Method: "greet",
		Params: json.RawMessage(`{"name":"wolf"}`),
	}
	resp := r.Handle(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result != "hello wolf" {
		t.Errorf("Result = %v; want %q", resp.Result, "hello wolf")
	}
}

// --- Method handler tests ---

func newTestDeps() *Deps {
	return &Deps{
		GetState:    func() string { return "idle" },
		GetModel:    func() string { return "test-model" },
		GetMessages: func() int { return 42 },
		GetTokens:   func() int { return 1337 },
		ListTools: func() []ToolInfo {
			return []ToolInfo{
				{Name: "read", Description: "Read a file"},
				{Name: "write", Description: "Write a file"},
			}
		},
		ListSessions: func() []SessionInfo {
			return []SessionInfo{
				{ID: "s1", Model: "gpt-4", Created: "2026-01-01T00:00:00Z"},
			}
		},
		Pause:  func() bool { return true },
		Cancel: func() bool { return true },
	}
}

func TestHandleGetStatus(t *testing.T) {
	r := NewRouter()
	RegisterHandlers(r, newTestDeps())

	resp := r.Handle(Request{ID: "10", Method: MethodGetStatus})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var result StatusResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.State != "idle" {
		t.Errorf("State = %q; want %q", result.State, "idle")
	}
	if result.Model != "test-model" {
		t.Errorf("Model = %q; want %q", result.Model, "test-model")
	}
	if result.Messages != 42 {
		t.Errorf("Messages = %d; want %d", result.Messages, 42)
	}
	if result.Tokens != 1337 {
		t.Errorf("Tokens = %d; want %d", result.Tokens, 1337)
	}
}

func TestHandleListTools(t *testing.T) {
	r := NewRouter()
	RegisterHandlers(r, newTestDeps())

	resp := r.Handle(Request{ID: "11", Method: MethodListTools})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result ToolListResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Tools) != 2 {
		t.Fatalf("len(Tools) = %d; want 2", len(result.Tools))
	}
	if result.Tools[0].Name != "read" {
		t.Errorf("Tools[0].Name = %q; want %q", result.Tools[0].Name, "read")
	}
}

func TestHandleListToolsNilSlice(t *testing.T) {
	d := newTestDeps()
	d.ListTools = func() []ToolInfo { return nil }
	r := NewRouter()
	RegisterHandlers(r, d)

	resp := r.Handle(Request{ID: "12", Method: MethodListTools})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result ToolListResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should be empty array, not null.
	if result.Tools == nil {
		t.Error("Tools should be empty slice, not nil")
	}
	if len(result.Tools) != 0 {
		t.Errorf("len(Tools) = %d; want 0", len(result.Tools))
	}
}

func TestHandleListSessions(t *testing.T) {
	r := NewRouter()
	RegisterHandlers(r, newTestDeps())

	resp := r.Handle(Request{ID: "13", Method: MethodListSessions})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result SessionListResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("len(Sessions) = %d; want 1", len(result.Sessions))
	}
	if result.Sessions[0].ID != "s1" {
		t.Errorf("Sessions[0].ID = %q; want %q", result.Sessions[0].ID, "s1")
	}
}

func TestHandleListSessionsNilSlice(t *testing.T) {
	d := newTestDeps()
	d.ListSessions = func() []SessionInfo { return nil }
	r := NewRouter()
	RegisterHandlers(r, d)

	resp := r.Handle(Request{ID: "14", Method: MethodListSessions})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result SessionListResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Sessions == nil {
		t.Error("Sessions should be empty slice, not nil")
	}
}

func TestHandlePause(t *testing.T) {
	r := NewRouter()
	RegisterHandlers(r, newTestDeps())

	resp := r.Handle(Request{ID: "15", Method: MethodPause})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result PauseResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !result.Paused {
		t.Error("Paused = false; want true")
	}
}

func TestHandlePauseFalse(t *testing.T) {
	d := newTestDeps()
	d.Pause = func() bool { return false }
	r := NewRouter()
	RegisterHandlers(r, d)

	resp := r.Handle(Request{ID: "16", Method: MethodPause})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result PauseResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Paused {
		t.Error("Paused = true; want false")
	}
}

func TestHandleCancel(t *testing.T) {
	r := NewRouter()
	RegisterHandlers(r, newTestDeps())

	resp := r.Handle(Request{ID: "17", Method: MethodCancel})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result CancelResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !result.Cancelled {
		t.Error("Cancelled = false; want true")
	}
}

func TestHandleCancelFalse(t *testing.T) {
	d := newTestDeps()
	d.Cancel = func() bool { return false }
	r := NewRouter()
	RegisterHandlers(r, d)

	resp := r.Handle(Request{ID: "18", Method: MethodCancel})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	data, _ := json.Marshal(resp.Result)
	var result CancelResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Cancelled {
		t.Error("Cancelled = true; want false")
	}
}

// --- JSONL round-trip via pipe ---

func TestServerJSONLRoundTrip(t *testing.T) {
	pr, pw := io.Pipe()

	router := NewRouter()
	RegisterHandlers(router, newTestDeps())

	srv := &Server{
		reader:  bufio.NewScanner(strings.NewReader(`{"id":"rt1","method":"get_status"}` + "\n")),
		writer:  pw,
		handler: router.Handle,
	}

	done := make(chan error, 1)
	go func() {
		err := srv.Run()
		pw.Close()
		done <- err
	}()

	scanner := bufio.NewScanner(pr)
	if !scanner.Scan() {
		t.Fatal("expected a response line")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != "rt1" {
		t.Errorf("ID = %q; want %q", resp.ID, "rt1")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if err := <-done; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestServerJSONLParseError(t *testing.T) {
	pr, pw := io.Pipe()

	srv := &Server{
		reader:  bufio.NewScanner(strings.NewReader("not json\n")),
		writer:  pw,
		handler: func(req Request) Response { return Response{} },
	}

	done := make(chan error, 1)
	go func() {
		err := srv.Run()
		pw.Close()
		done <- err
	}()

	scanner := bufio.NewScanner(pr)
	if !scanner.Scan() {
		t.Fatal("expected a response line")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected parse error; got nil")
	}
	if resp.Error.Code != ErrCodeParse {
		t.Errorf("Code = %d; want %d", resp.Error.Code, ErrCodeParse)
	}

	if err := <-done; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestServerJSONLMultipleRequests(t *testing.T) {
	input := `{"id":"m1","method":"get_status"}` + "\n" +
		`{"id":"m2","method":"pause"}` + "\n" +
		`{"id":"m3","method":"cancel"}` + "\n"

	pr, pw := io.Pipe()

	router := NewRouter()
	RegisterHandlers(router, newTestDeps())

	srv := &Server{
		reader:  bufio.NewScanner(strings.NewReader(input)),
		writer:  pw,
		handler: router.Handle,
	}

	done := make(chan error, 1)
	go func() {
		err := srv.Run()
		pw.Close()
		done <- err
	}()

	scanner := bufio.NewScanner(pr)
	ids := []string{"m1", "m2", "m3"}
	for i, wantID := range ids {
		if !scanner.Scan() {
			t.Fatalf("response %d: expected line", i)
		}
		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("response %d: unmarshal: %v", i, err)
		}
		if resp.ID != wantID {
			t.Errorf("response %d: ID = %q; want %q", i, resp.ID, wantID)
		}
		if resp.Error != nil {
			t.Errorf("response %d: unexpected error: %v", i, resp.Error)
		}
	}

	if err := <-done; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

// --- Schema JSON serialization ---

func TestStatusResultJSON(t *testing.T) {
	s := StatusResult{State: "running", Model: "m1", Messages: 5, Tokens: 100}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StatusResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != s {
		t.Errorf("round-trip mismatch: got %+v; want %+v", got, s)
	}
}

func TestToolListResultJSON(t *testing.T) {
	r := ToolListResult{Tools: []ToolInfo{{Name: "t1", Description: "d1"}}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"name":"t1"`) {
		t.Errorf("JSON = %s; want it to contain name field", data)
	}
}

func TestSessionListResultJSON(t *testing.T) {
	r := SessionListResult{Sessions: []SessionInfo{{ID: "s1", Model: "m", Created: "2026-01-01"}}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"id":"s1"`) {
		t.Errorf("JSON = %s; want it to contain id field", data)
	}
}

// --- Error code constants ---

func TestErrorCodeValues(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"parse", ErrCodeParse, -32700},
		{"invalid_req", ErrCodeInvalidReq, -32600},
		{"method_not_found", ErrCodeMethodNotFound, -32601},
		{"invalid_params", ErrCodeInvalidParams, -32602},
		{"internal", ErrCodeInternal, -32603},
		{"agent_running", ErrCodeAgentRunning, -32001},
		{"no_session", ErrCodeNoSession, -32002},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d; want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

// --- Method constants ---

func TestMethodConstants(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"get_status", MethodGetStatus, "get_status"},
		{"list_tools", MethodListTools, "list_tools"},
		{"list_sessions", MethodListSessions, "list_sessions"},
		{"pause", MethodPause, "pause"},
		{"cancel", MethodCancel, "cancel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

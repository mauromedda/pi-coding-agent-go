// ABOUTME: Handler implementations for RPC methods (stream, pause, cancel, status, tools, sessions)
// ABOUTME: Dispatches requests to appropriate handlers with input validation

package rpc

import "encoding/json"

// Method constants for new RPC methods.
const (
	MethodGetStatus    = "get_status"
	MethodListTools    = "list_tools"
	MethodListSessions = "list_sessions"
	MethodPause        = "pause"
	MethodCancel       = "cancel"
)

// HandlerFunc processes an RPC request's params and returns a Response.
type HandlerFunc func(params json.RawMessage) Response

// Router dispatches RPC requests to registered handlers by method name.
type Router struct {
	handlers map[string]HandlerFunc
}

// NewRouter creates a Router with an empty handler registry.
func NewRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc)}
}

// Register associates a method name with a handler function.
func (r *Router) Register(method string, handler HandlerFunc) {
	r.handlers[method] = handler
}

// Handle dispatches a request to the registered handler, or returns
// a method-not-found error if no handler is registered.
func (r *Router) Handle(req Request) Response {
	h, ok := r.handlers[req.Method]
	if !ok {
		return Response{
			ID:    req.ID,
			Error: NewMethodNotFoundError(req.Method),
		}
	}

	raw, err := marshalParams(req.Params)
	if err != nil {
		return Response{
			ID:    req.ID,
			Error: NewInvalidParamsError(err.Error()),
		}
	}

	resp := h(raw)
	resp.ID = req.ID
	return resp
}

// marshalParams converts the generic Params field into json.RawMessage
// so handlers can decode it themselves.
func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	if raw, ok := params.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(params)
}

// ToolInfo describes a single available tool.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SessionInfo describes a stored session.
type SessionInfo struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Created string `json:"created"`
}

// Deps holds the function dependencies that handlers call into.
type Deps struct {
	GetState     func() string
	GetModel     func() string
	GetMessages  func() int
	GetTokens    func() int
	ListTools    func() []ToolInfo
	ListSessions func() []SessionInfo
	Pause        func() bool
	Cancel       func() bool
}

// RegisterHandlers wires all new method handlers into the given router.
func RegisterHandlers(r *Router, d *Deps) {
	r.Register(MethodGetStatus, handleGetStatus(d))
	r.Register(MethodListTools, handleListTools(d))
	r.Register(MethodListSessions, handleListSessions(d))
	r.Register(MethodPause, handlePause(d))
	r.Register(MethodCancel, handleCancel(d))
}

func handleGetStatus(d *Deps) HandlerFunc {
	return func(_ json.RawMessage) Response {
		return Response{
			Result: StatusResult{
				State:    d.GetState(),
				Model:    d.GetModel(),
				Messages: d.GetMessages(),
				Tokens:   d.GetTokens(),
			},
		}
	}
}

func handleListTools(d *Deps) HandlerFunc {
	return func(_ json.RawMessage) Response {
		tools := d.ListTools()
		if tools == nil {
			tools = []ToolInfo{}
		}
		return Response{Result: ToolListResult{Tools: tools}}
	}
}

func handleListSessions(d *Deps) HandlerFunc {
	return func(_ json.RawMessage) Response {
		sessions := d.ListSessions()
		if sessions == nil {
			sessions = []SessionInfo{}
		}
		return Response{Result: SessionListResult{Sessions: sessions}}
	}
}

func handlePause(d *Deps) HandlerFunc {
	return func(_ json.RawMessage) Response {
		return Response{Result: PauseResult{Paused: d.Pause()}}
	}
}

func handleCancel(d *Deps) HandlerFunc {
	return func(_ json.RawMessage) Response {
		return Response{Result: CancelResult{Cancelled: d.Cancel()}}
	}
}

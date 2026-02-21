// ABOUTME: RPC request/response types for external integrations
// ABOUTME: JSON-serializable types for IDE extension communication

package rpc

// Request represents an RPC request from an external client.
type Request struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// Response represents an RPC response to an external client.
type Response struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Error represents an RPC error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Methods
const (
	MethodPrompt     = "prompt"
	MethodAbort      = "abort"
	MethodGetSession = "get_session"
	MethodSetModel   = "set_model"
)

// ABOUTME: Standard JSON-RPC error codes and custom application errors
// ABOUTME: Provides error constructors for common RPC failure scenarios

package rpc

// Standard JSON-RPC 2.0 error codes.
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidReq     = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// Custom application error codes.
const (
	ErrCodeAgentRunning = -32001
	ErrCodeNoSession    = -32002
)

// NewParseError returns an Error for malformed JSON input.
func NewParseError(msg string) *Error {
	return &Error{Code: ErrCodeParse, Message: msg}
}

// NewMethodNotFoundError returns an Error for an unknown RPC method.
func NewMethodNotFoundError(method string) *Error {
	return &Error{Code: ErrCodeMethodNotFound, Message: "method not found: " + method}
}

// NewInvalidParamsError returns an Error for invalid method parameters.
func NewInvalidParamsError(msg string) *Error {
	return &Error{Code: ErrCodeInvalidParams, Message: msg}
}

// NewInternalError returns an Error for unexpected server-side failures.
func NewInternalError(msg string) *Error {
	return &Error{Code: ErrCodeInternal, Message: msg}
}

// NewAgentRunningError returns an Error when an operation cannot proceed
// because the agent is already processing a request.
func NewAgentRunningError() *Error {
	return &Error{Code: ErrCodeAgentRunning, Message: "agent is already running"}
}

// NewNoSessionError returns an Error when no active session exists.
func NewNoSessionError() *Error {
	return &Error{Code: ErrCodeNoSession, Message: "no active session"}
}

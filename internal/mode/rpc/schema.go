// ABOUTME: Request/response schema types for new RPC methods
// ABOUTME: JSON-serializable types for get_status, list_tools, list_sessions

package rpc

// StatusResult is the response payload for the get_status method.
type StatusResult struct {
	State    string `json:"state"`
	Model    string `json:"model"`
	Messages int    `json:"messages"`
	Tokens   int    `json:"tokens"`
}

// ToolListResult is the response payload for the list_tools method.
type ToolListResult struct {
	Tools []ToolInfo `json:"tools"`
}

// SessionListResult is the response payload for the list_sessions method.
type SessionListResult struct {
	Sessions []SessionInfo `json:"sessions"`
}

// PauseResult is the response payload for the pause method.
type PauseResult struct {
	Paused bool `json:"paused"`
}

// CancelResult is the response payload for the cancel method.
type CancelResult struct {
	Cancelled bool `json:"cancelled"`
}

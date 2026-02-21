// ABOUTME: Hook lifecycle types: events, input/output structs for hook execution
// ABOUTME: Defines the contract between agent loop and hook engine

package hooks

// HookEvent identifies a lifecycle event in the agent loop.
type HookEvent string

const (
	PreToolUse       HookEvent = "PreToolUse"
	PostToolUse      HookEvent = "PostToolUse"
	UserPromptSubmit HookEvent = "UserPromptSubmit"
	Stop             HookEvent = "Stop"
	SessionStart     HookEvent = "SessionStart"
	SessionEnd       HookEvent = "SessionEnd"
)

// HookInput is the data passed to a hook command via stdin as JSON.
type HookInput struct {
	Event     HookEvent      `json:"event"`
	Tool      string         `json:"tool,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	WorkDir   string         `json:"work_dir,omitempty"`
}

// HookOutput is the JSON response expected from a hook command on stdout.
type HookOutput struct {
	Blocked bool              `json:"blocked,omitempty"`
	Message string            `json:"message,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

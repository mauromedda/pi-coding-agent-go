// ABOUTME: Core AI SDK types: Message, Content, Tool, Usage, Model, StopReason
// ABOUTME: Shared across all providers; wire-format agnostic

package ai

import "encoding/json"

// Role represents a message role in the conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopEndTurn   StopReason = "end_turn"
	StopMaxTokens StopReason = "max_tokens"
	StopToolUse   StopReason = "tool_use"
	StopStop      StopReason = "stop"
)

// ContentType identifies the kind of content block.
type ContentType string

const (
	ContentText     ContentType = "text"
	ContentImage    ContentType = "image"
	ContentToolUse  ContentType = "tool_use"
	ContentToolResult ContentType = "tool_result"
	ContentThinking ContentType = "thinking"
)

// CacheControl instructs the provider to cache a content block.
type CacheControl struct {
	Type string `json:"type"`
}

// Content represents a content block within a message.
type Content struct {
	Type         ContentType     `json:"type"`
	Text         string          `json:"text,omitempty"`
	ID           string          `json:"id,omitempty"`            // Tool use/result ID
	Name         string          `json:"name,omitempty"`          // Tool name
	Input        json.RawMessage `json:"input,omitempty"`         // Tool use input
	ResultText   string          `json:"result_text,omitempty"`   // Tool result content
	IsError      bool            `json:"is_error,omitempty"`      // Tool result error flag
	MediaType    string          `json:"media_type,omitempty"`    // Image media type
	Data         string          `json:"data,omitempty"`          // Base64 image data
	Thinking     string          `json:"thinking,omitempty"`      // Extended thinking text
	CacheControl *CacheControl   `json:"cache_control,omitempty"` // Provider caching hint
}

// Message represents a conversation message.
type Message struct {
	Role    Role      `json:"role"`
	Content []Content `json:"content"`
}

// NewTextMessage creates a message with a single text content block.
func NewTextMessage(role Role, text string) Message {
	return Message{
		Role:    role,
		Content: []Content{{Type: ContentText, Text: text}},
	}
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read_input_tokens,omitempty"`
	CacheCreate  int `json:"cache_creation_input_tokens,omitempty"`
}

// Tool defines a tool the model can invoke.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Parameters   json.RawMessage `json:"input_schema"`          // JSON Schema
	CacheControl *CacheControl   `json:"cache_control,omitempty"` // Provider caching hint
}

// Api identifies an API provider.
type Api string

const (
	ApiAnthropic Api = "anthropic"
	ApiOpenAI    Api = "openai"
	ApiGoogle    Api = "google"
	ApiVertex    Api = "vertex"
)

// Model defines a model's metadata.
type Model struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Api              Api               `json:"api"`
	MaxTokens        int               `json:"max_tokens"`
	MaxOutputTokens  int               `json:"max_output_tokens"`
	ContextWindow    int               `json:"context_window,omitempty"`
	SupportsImages   bool              `json:"supports_images"`
	SupportsTools    bool              `json:"supports_tools"`
	SupportsThinking bool              `json:"supports_thinking"`
	BaseURL          string            `json:"base_url,omitempty"`
	CustomHeaders    map[string]string `json:"custom_headers,omitempty"`
}

// EffectiveContextWindow returns ContextWindow if set, otherwise MaxTokens.
func (m *Model) EffectiveContextWindow() int {
	if m.ContextWindow > 0 {
		return m.ContextWindow
	}
	return m.MaxTokens
}

// Context holds the messages and tools for an LLM call.
type Context struct {
	System             string        `json:"system,omitempty"`
	Messages           []Message     `json:"messages"`
	Tools              []Tool        `json:"tools,omitempty"`
	SystemCacheControl *CacheControl `json:"system_cache_control,omitempty"` // Cache hint for system prompt
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	MaxTokens    int     `json:"max_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	TopP         float64 `json:"top_p,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
	Thinking     bool    `json:"thinking,omitempty"`
}

// AssistantMessage is the final result of a streaming response.
type AssistantMessage struct {
	Content    []Content  `json:"content"`
	StopReason StopReason `json:"stop_reason"`
	Usage      Usage      `json:"usage"`
	Model      string     `json:"model"`
}

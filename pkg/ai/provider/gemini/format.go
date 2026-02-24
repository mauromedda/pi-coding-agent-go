// ABOUTME: Shared Gemini API types used by both Google AI and Vertex AI providers
// ABOUTME: Eliminates duplication of identical wire-format types between providers

package gemini

// Content represents a content block with role and parts.
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

// InlineData carries base64-encoded binary data within a Gemini Part.
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// Part represents a single part within a content block.
type Part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *InlineData       `json:"inlineData,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// FunctionCall represents a tool invocation by the model.
type FunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

// FunctionResponse represents the result of a tool invocation.
type FunctionResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

// ToolDef wraps function declarations for the Gemini API.
type ToolDef struct {
	FunctionDeclarations []FunctionDecl `json:"functionDeclarations"`
}

// FunctionDecl describes a function that the model can call.
type FunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// GenerationConfig holds generation parameters for the Gemini API.
type GenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

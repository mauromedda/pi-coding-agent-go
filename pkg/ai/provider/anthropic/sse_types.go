// ABOUTME: Named SSE payload types for Anthropic streaming events
// ABOUTME: Promoted from anonymous structs for easyjson codegen (zero-reflection decoding)

//go:generate easyjson -all sse_types.go

package anthropic

import "github.com/mauromedda/pi-coding-agent-go/pkg/ai"

// messageStartPayload is the SSE payload for "message_start" events.
type messageStartPayload struct {
	Message struct {
		Model string   `json:"model"`
		Usage ai.Usage `json:"usage"`
	} `json:"message"`
}

// contentBlockStartPayload is the SSE payload for "content_block_start" events.
type contentBlockStartPayload struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
		Text string `json:"text"`
	} `json:"content_block"`
}

// contentBlockDeltaPayload is the SSE payload for "content_block_delta" events.
type contentBlockDeltaPayload struct {
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
}

// messageDeltaPayload is the SSE payload for "message_delta" events.
type messageDeltaPayload struct {
	Delta struct {
		StopReason ai.StopReason `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// sseErrorPayload is the SSE payload for "error" events.
type sseErrorPayload struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

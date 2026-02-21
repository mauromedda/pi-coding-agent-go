// ABOUTME: Message format conversion between internal types and OpenAI API format
// ABOUTME: Handles messages, tools, and tool calls for Chat Completions

package openai

import (
	"encoding/json"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

type chatMessage struct {
	Role       string          `json:"role"`
	Content    any             `json:"content,omitempty"`
	ToolCalls  []toolCallReq   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type toolCallReq struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function toolCallFuncReq `json:"function"`
}

type toolCallFuncReq struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function toolFuncDef `json:"function"`
}

type toolFuncDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// SSE response types
type chatCompletionChunk struct {
	ID      string              `json:"id"`
	Choices []chunkChoice       `json:"choices"`
	Usage   *chunkUsage         `json:"usage,omitempty"`
}

type chunkChoice struct {
	Index        int         `json:"index"`
	Delta        chunkDelta  `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type chunkDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []toolCallDelta `json:"tool_calls,omitempty"`
}

type toolCallDelta struct {
	Index    int             `json:"index"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function toolCallFuncDelta `json:"function,omitempty"`
}

type toolCallFuncDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type chunkUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type toolCallAccumulator struct {
	id   string
	name string
	args string
}

func buildRequestBody(model *ai.Model, ctx *ai.Context, opts *ai.StreamOptions) map[string]any {
	body := map[string]any{
		"model":  model.ID,
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}

	// Convert messages
	msgs := convertMessages(ctx)
	body["messages"] = msgs

	// Convert tools
	if len(ctx.Tools) > 0 {
		body["tools"] = convertTools(ctx.Tools)
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			body["max_tokens"] = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			body["temperature"] = opts.Temperature
		}
	}

	return body
}

func convertMessages(ctx *ai.Context) []chatMessage {
	var msgs []chatMessage

	if ctx.System != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: ctx.System})
	}

	for _, m := range ctx.Messages {
		msg := chatMessage{Role: string(m.Role)}

		// Simple text content
		if len(m.Content) == 1 && m.Content[0].Type == ai.ContentText {
			msg.Content = m.Content[0].Text
			msgs = append(msgs, msg)
			continue
		}

		// Tool use content (assistant with tool calls)
		var toolCalls []toolCallReq
		var textContent string
		for _, c := range m.Content {
			switch c.Type {
			case ai.ContentText:
				textContent += c.Text
			case ai.ContentToolUse:
				toolCalls = append(toolCalls, toolCallReq{
					ID:   c.ID,
					Type: "function",
					Function: toolCallFuncReq{
						Name:      c.Name,
						Arguments: string(c.Input),
					},
				})
			case ai.ContentToolResult:
				msgs = append(msgs, chatMessage{
					Role:       "tool",
					Content:    c.Content,
					ToolCallID: c.ID,
				})
				continue
			}
		}

		if len(toolCalls) > 0 {
			msg.Content = textContent
			msg.ToolCalls = toolCalls
		} else {
			msg.Content = textContent
		}

		msgs = append(msgs, msg)
	}

	return msgs
}

func convertTools(tools []ai.Tool) []toolDef {
	defs := make([]toolDef, len(tools))
	for i, t := range tools {
		defs[i] = toolDef{
			Type: "function",
			Function: toolFuncDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return defs
}

func processToolCallDelta(accum []toolCallAccumulator, delta toolCallDelta, stream *ai.EventStream) []toolCallAccumulator {
	// Extend accumulator if needed
	for len(accum) <= delta.Index {
		accum = append(accum, toolCallAccumulator{})
	}

	tc := &accum[delta.Index]

	if delta.ID != "" {
		tc.id = delta.ID
	}
	if delta.Function.Name != "" {
		tc.name = delta.Function.Name
		stream.Send(ai.StreamEvent{
			Type:     ai.EventToolUseStart,
			ToolID:   tc.id,
			ToolName: tc.name,
		})
	}
	if delta.Function.Arguments != "" {
		tc.args += delta.Function.Arguments
		stream.Send(ai.StreamEvent{
			Type:      ai.EventToolUseDelta,
			ToolID:    tc.id,
			ToolInput: delta.Function.Arguments,
		})
	}

	return accum
}

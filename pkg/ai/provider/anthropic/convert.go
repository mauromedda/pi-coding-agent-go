// ABOUTME: Message format conversion between internal ai types and Anthropic API format
// ABOUTME: Builds request bodies and converts messages/tools for the Anthropic Messages API

package anthropic

import (
	"encoding/json"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// convertMessages transforms internal messages into Anthropic API format.
func convertMessages(msgs []ai.Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, map[string]any{
			"role":    string(msg.Role),
			"content": convertContent(msg.Content),
		})
	}
	return out
}

// convertContent transforms internal content blocks into Anthropic API format.
func convertContent(blocks []ai.Content) []map[string]any {
	out := make([]map[string]any, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, convertContentBlock(b))
	}
	return out
}

// convertContentBlock converts a single content block to Anthropic API format.
func convertContentBlock(b ai.Content) map[string]any {
	switch b.Type {
	case ai.ContentText:
		return map[string]any{"type": "text", "text": b.Text}
	case ai.ContentImage:
		return map[string]any{
			"type": "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": b.MediaType,
				"data":       b.Data,
			},
		}
	case ai.ContentToolUse:
		return map[string]any{
			"type":  "tool_use",
			"id":    b.ID,
			"name":  b.Name,
			"input": json.RawMessage(b.Input),
		}
	case ai.ContentToolResult:
		result := map[string]any{
			"type":        "tool_result",
			"tool_use_id": b.ID,
		}
		if len(b.Images) > 0 {
			parts := make([]map[string]any, 0, 1+len(b.Images))
			parts = append(parts, map[string]any{"type": "text", "text": b.ResultText})
			for _, img := range b.Images {
				parts = append(parts, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": img.MediaType,
						"data":       img.Data,
					},
				})
			}
			result["content"] = parts
		} else {
			result["content"] = b.ResultText
		}
		if b.IsError {
			result["is_error"] = true
		}
		return result
	default:
		return map[string]any{"type": string(b.Type), "text": b.Text}
	}
}

// convertTools transforms internal tool definitions into Anthropic API format.
func convertTools(tools []ai.Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		entry := map[string]any{
			"name":        t.Name,
			"description": t.Description,
		}
		if t.Parameters != nil {
			entry["input_schema"] = json.RawMessage(t.Parameters)
		}
		out = append(out, entry)
	}
	return out
}

// buildRequestBody constructs the full Anthropic Messages API request body.
func buildRequestBody(model *ai.Model, ctx *ai.Context, opts *ai.StreamOptions) map[string]any {
	body := map[string]any{
		"model":      model.ID,
		"stream":     true,
		"max_tokens": resolveMaxTokens(model, opts),
	}

	if ctx.System != "" {
		body["system"] = ctx.System
	}

	if len(ctx.Messages) > 0 {
		body["messages"] = convertMessages(ctx.Messages)
	}

	if len(ctx.Tools) > 0 {
		body["tools"] = convertTools(ctx.Tools)
	}

	applyStreamOptions(body, opts)

	return body
}

// resolveMaxTokens returns the max tokens value, preferring opts over model defaults.
func resolveMaxTokens(model *ai.Model, opts *ai.StreamOptions) int {
	if opts != nil && opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return model.MaxOutputTokens
}

// applyStreamOptions applies optional streaming parameters to the request body.
func applyStreamOptions(body map[string]any, opts *ai.StreamOptions) {
	if opts == nil {
		return
	}
	if opts.Temperature > 0 {
		body["temperature"] = opts.Temperature
	}
	if opts.TopP > 0 {
		body["top_p"] = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		body["stop_sequences"] = opts.StopSequences
	}
}

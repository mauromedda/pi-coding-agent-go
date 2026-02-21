// ABOUTME: OpenAI Chat Completions streaming provider (also supports Ollama, vLLM)
// ABOUTME: Implements ApiProvider with SSE-based streaming for OpenAI-compatible APIs

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/internal/sse"
)

const defaultBaseURL = "https://api.openai.com"

// Provider implements the OpenAI Chat Completions API.
type Provider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New creates an OpenAI provider.
func New(apiKey, baseURL string) *Provider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

// Api returns the provider identifier.
func (p *Provider) Api() ai.Api {
	return ai.ApiOpenAI
}

// Stream initiates a streaming chat completion.
func (p *Provider) Stream(ctx context.Context, model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions) *ai.EventStream {
	stream := ai.NewEventStream(64)

	go func() {
		if err := p.doStream(ctx, model, llmCtx, opts, stream); err != nil {
			stream.FinishWithError(err)
		}
	}()

	return stream
}

func (p *Provider) doStream(ctx context.Context, model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions, stream *ai.EventStream) error {
	body := buildRequestBody(model, llmCtx, opts)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return p.processSSE(sse.NewReader(resp.Body), stream)
}

func (p *Provider) processSSE(reader *sse.Reader, stream *ai.EventStream) error {
	var result ai.AssistantMessage
	var toolCalls []toolCallAccumulator

	for {
		event, err := reader.Next()
		if err != nil {
			stream.Finish(&result)
			return nil
		}
		if event.Data == "[DONE]" {
			break
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			// Text content
			if delta.Content != "" {
				stream.Send(ai.StreamEvent{
					Type: ai.EventContentDelta,
					Text: delta.Content,
				})
				appendTextContent(&result, delta.Content)
			}

			// Tool calls
			for _, tc := range delta.ToolCalls {
				toolCalls = processToolCallDelta(toolCalls, tc, stream)
			}

			// Finish reason
			if choice.FinishReason != "" {
				result.StopReason = mapFinishReason(choice.FinishReason)
			}
		}

		if chunk.Usage != nil {
			result.Usage = ai.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
			}
		}
	}

	// Finalize tool calls
	for _, tc := range toolCalls {
		result.Content = append(result.Content, ai.Content{
			Type:  ai.ContentToolUse,
			ID:    tc.id,
			Name:  tc.name,
			Input: json.RawMessage(tc.args),
		})
	}

	result.Model = "openai"
	stream.Finish(&result)
	return nil
}

func appendTextContent(msg *ai.AssistantMessage, text string) {
	for i := range msg.Content {
		if msg.Content[i].Type == ai.ContentText {
			msg.Content[i].Text += text
			return
		}
	}
	msg.Content = append(msg.Content, ai.Content{Type: ai.ContentText, Text: text})
}

func mapFinishReason(reason string) ai.StopReason {
	switch reason {
	case "stop":
		return ai.StopEndTurn
	case "length":
		return ai.StopMaxTokens
	case "tool_calls":
		return ai.StopToolUse
	default:
		return ai.StopStop
	}
}

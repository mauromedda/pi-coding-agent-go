// ABOUTME: Anthropic Messages API streaming provider implementation
// ABOUTME: Handles SSE event parsing, content block accumulation, and tool use streaming

package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/internal/httputil"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/internal/sse"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	messagesPath     = "/v1/messages"
	streamBufferSize = 64
)

// Provider implements ai.ApiProvider for the Anthropic Messages API.
type Provider struct {
	client *httputil.Client
	apiKey string
}

// New creates an Anthropic provider. If apiKey is empty, it reads ANTHROPIC_API_KEY.
func New(apiKey, baseURL string) *Provider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = httputil.NormalizeBaseURL(baseURL)

	headers := map[string]string{
		"x-api-key":        apiKey,
		"anthropic-version": anthropicVersion,
		"content-type":      "application/json",
	}

	return &Provider{
		client: httputil.NewClient(baseURL, headers),
		apiKey: apiKey,
	}
}

// Api returns the Anthropic API identifier.
func (p *Provider) Api() ai.Api {
	return ai.ApiAnthropic
}

// Stream initiates a streaming call to the Anthropic Messages API.
func (p *Provider) Stream(ctx context.Context, model *ai.Model, aiCtx *ai.Context, opts *ai.StreamOptions) *ai.EventStream {
	stream := ai.NewEventStream(streamBufferSize)

	go p.runStream(ctx, stream, model, aiCtx, opts)

	return stream
}

// runStream performs the HTTP request and processes SSE events in a goroutine.
func (p *Provider) runStream(
	ctx context.Context,
	stream *ai.EventStream,
	model *ai.Model,
	aiCtx *ai.Context,
	opts *ai.StreamOptions,
) {
	body := buildRequestBody(model, aiCtx, opts)

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		stream.FinishWithError(fmt.Errorf("failed to marshal request body: %w", err))
		return
	}

	reader, resp, err := p.client.StreamSSE(ctx, http.MethodPost, messagesPath, bytes.NewReader(bodyJSON))
	if err != nil {
		stream.FinishWithError(fmt.Errorf("failed to start SSE stream: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		handleErrorResponse(stream, resp)
		return
	}

	processEvents(stream, reader)
}

// handleErrorResponse reads the error body and finishes the stream with an error.
func handleErrorResponse(stream *ai.EventStream, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	stream.FinishWithError(fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, body))
}

// processEvents reads SSE events and dispatches them to the EventStream.
func processEvents(stream *ai.EventStream, reader *sse.Reader) {
	acc := newAccumulator()

	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			stream.FinishWithError(fmt.Errorf("SSE read error: %w", err))
			return
		}

		if !dispatchEvent(stream, acc, ev) {
			return
		}
	}

	stream.Finish(acc.buildResult())
}

// dispatchEvent routes a single SSE event to the appropriate handler.
// Returns false if the stream loop should stop.
func dispatchEvent(stream *ai.EventStream, acc *accumulator, ev *sse.Event) bool {
	switch ev.Type {
	case "message_start":
		return handleMessageStart(acc, ev)
	case "content_block_start":
		return handleContentBlockStart(stream, acc, ev)
	case "content_block_delta":
		return handleContentBlockDelta(stream, acc, ev)
	case "content_block_stop":
		handleContentBlockStop(stream, acc)
		return true
	case "message_delta":
		return handleMessageDelta(stream, acc, ev)
	case "message_stop":
		stream.Send(ai.StreamEvent{Type: ai.EventMessageDone})
		return true
	case "ping":
		stream.Send(ai.StreamEvent{Type: ai.EventPing})
		return true
	case "error":
		handleSSEError(stream, ev)
		return false
	default:
		return true
	}
}

// handleMessageStart parses the message_start event for model and usage info.
func handleMessageStart(acc *accumulator, ev *sse.Event) bool {
	var payload struct {
		Message struct {
			Model string   `json:"model"`
			Usage ai.Usage `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal([]byte(ev.Data), &payload) == nil {
		acc.model = payload.Message.Model
		acc.usage = payload.Message.Usage
	}
	return true
}

// handleContentBlockStart begins a new content block in the accumulator.
func handleContentBlockStart(stream *ai.EventStream, acc *accumulator, ev *sse.Event) bool {
	var payload struct {
		Index        int `json:"index"`
		ContentBlock struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
			Text string `json:"text"`
		} `json:"content_block"`
	}
	if json.Unmarshal([]byte(ev.Data), &payload) != nil {
		return true
	}

	acc.startBlock(payload.ContentBlock.Type, payload.ContentBlock.ID, payload.ContentBlock.Name)

	if payload.ContentBlock.Type == "tool_use" {
		stream.Send(ai.StreamEvent{
			Type:     ai.EventToolUseStart,
			ToolID:   payload.ContentBlock.ID,
			ToolName: payload.ContentBlock.Name,
		})
	}

	return true
}

// handleContentBlockDelta processes content deltas (text or tool input JSON).
func handleContentBlockDelta(stream *ai.EventStream, acc *accumulator, ev *sse.Event) bool {
	var payload struct {
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	if json.Unmarshal([]byte(ev.Data), &payload) != nil {
		return true
	}

	switch payload.Delta.Type {
	case "text_delta":
		acc.appendText(payload.Delta.Text)
		stream.Send(ai.StreamEvent{Type: ai.EventContentDelta, Text: payload.Delta.Text})
	case "input_json_delta":
		acc.appendToolInput(payload.Delta.PartialJSON)
		stream.Send(ai.StreamEvent{Type: ai.EventToolUseDelta, ToolInput: payload.Delta.PartialJSON})
	}

	return true
}

// handleContentBlockStop finalizes the current content block.
func handleContentBlockStop(stream *ai.EventStream, acc *accumulator) {
	block := acc.finishBlock()
	if block == nil {
		return
	}

	switch block.Type {
	case ai.ContentText:
		stream.Send(ai.StreamEvent{Type: ai.EventContentDone, Text: block.Text})
	case ai.ContentToolUse:
		stream.Send(ai.StreamEvent{
			Type:      ai.EventToolUseDone,
			ToolID:    block.ID,
			ToolName:  block.Name,
			ToolInput: string(block.Input),
		})
	}
}

// handleMessageDelta processes message-level updates (stop_reason, usage).
func handleMessageDelta(stream *ai.EventStream, acc *accumulator, ev *sse.Event) bool {
	var payload struct {
		Delta struct {
			StopReason ai.StopReason `json:"stop_reason"`
		} `json:"delta"`
		Usage struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal([]byte(ev.Data), &payload) != nil {
		return true
	}

	acc.stopReason = payload.Delta.StopReason
	if payload.Usage.OutputTokens > 0 {
		acc.usage.OutputTokens = payload.Usage.OutputTokens
	}

	stream.Send(ai.StreamEvent{
		Type:       ai.EventMessageDelta,
		StopReason: payload.Delta.StopReason,
		Usage:      &acc.usage,
	})

	return true
}

// handleSSEError processes an error event from the SSE stream.
func handleSSEError(stream *ai.EventStream, ev *sse.Event) {
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	msg := ev.Data
	if json.Unmarshal([]byte(ev.Data), &payload) == nil && payload.Error.Message != "" {
		msg = payload.Error.Message
	}

	stream.FinishWithError(fmt.Errorf("anthropic stream error: %s", msg))
}

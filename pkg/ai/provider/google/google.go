// ABOUTME: Google Generative AI (Gemini) streaming provider
// ABOUTME: Implements ApiProvider for the Google AI Studio API

package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// Provider implements the Google Generative AI API.
type Provider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New creates a Google AI provider.
func New(apiKey, baseURL string) *Provider {
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
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
	return ai.ApiGoogle
}

// Stream initiates a streaming generation request.
func (p *Provider) Stream(model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions) *ai.EventStream {
	stream := ai.NewEventStream(64)

	go func() {
		if err := p.doStream(model, llmCtx, opts, stream); err != nil {
			stream.FinishWithError(err)
		}
	}()

	return stream
}

func (p *Provider) doStream(model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions, stream *ai.EventStream) error {
	body := buildGeminiRequestBody(llmCtx, opts)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s",
		p.baseURL, model.ID, p.apiKey)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	return processGeminiSSE(resp.Body, stream)
}

func processGeminiSSE(body io.Reader, stream *ai.EventStream) error {
	var result ai.AssistantMessage
	decoder := json.NewDecoder(body)

	for {
		var chunk geminiResponse
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			// Try to continue on parse errors
			continue
		}

		for _, candidate := range chunk.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					stream.Send(ai.StreamEvent{
						Type: ai.EventContentDelta,
						Text: part.Text,
					})
					result.Content = append(result.Content, ai.Content{
						Type: ai.ContentText,
						Text: part.Text,
					})
				}
				if part.FunctionCall != nil {
					argsBytes, _ := json.Marshal(part.FunctionCall.Args)
					result.Content = append(result.Content, ai.Content{
						Type:  ai.ContentToolUse,
						Name:  part.FunctionCall.Name,
						Input: argsBytes,
					})
					stream.Send(ai.StreamEvent{
						Type:     ai.EventToolUseStart,
						ToolName: part.FunctionCall.Name,
					})
				}
			}

			if candidate.FinishReason != "" {
				result.StopReason = mapGeminiFinishReason(candidate.FinishReason)
			}
		}

		if chunk.UsageMetadata != nil {
			result.Usage = ai.Usage{
				InputTokens:  chunk.UsageMetadata.PromptTokenCount,
				OutputTokens: chunk.UsageMetadata.CandidatesTokenCount,
			}
		}
	}

	result.Model = "google"
	stream.Finish(&result)
	return nil
}

func mapGeminiFinishReason(reason string) ai.StopReason {
	switch reason {
	case "STOP":
		return ai.StopEndTurn
	case "MAX_TOKENS":
		return ai.StopMaxTokens
	default:
		return ai.StopStop
	}
}

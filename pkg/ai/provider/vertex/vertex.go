// ABOUTME: Google Vertex AI streaming provider with service account auth
// ABOUTME: Uses same Gemini format as Google AI but with Vertex endpoint and OAuth

package vertex

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

// Provider implements the Vertex AI API using Gemini format.
type Provider struct {
	projectID string
	location  string
	baseURL   string
	client    *http.Client
	// tokenSource would be added when golang.org/x/oauth2 is integrated
}

// New creates a Vertex AI provider.
func New(projectID, location, baseURL string) *Provider {
	if projectID == "" {
		projectID = os.Getenv("VERTEX_PROJECT_ID")
	}
	if location == "" {
		location = os.Getenv("VERTEX_LOCATION")
		if location == "" {
			location = "us-central1"
		}
	}
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", location)
	}
	return &Provider{
		projectID: projectID,
		location:  location,
		baseURL:   baseURL,
		client:    &http.Client{},
	}
}

// Api returns the provider identifier.
func (p *Provider) Api() ai.Api {
	return ai.ApiVertex
}

// Stream initiates a streaming generation request via Vertex AI.
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
	body := buildVertexRequestBody(llmCtx, opts)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent",
		p.baseURL, p.projectID, p.location, model.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// TODO: Add OAuth2 token when golang.org/x/oauth2 is integrated
	// For now, use API key from env if available
	if key := os.Getenv("VERTEX_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Vertex API error: status %d: %s", resp.StatusCode, string(respBody))
	}

	// Process response using same format as Google AI
	return processVertexResponse(resp.Body, stream)
}

func processVertexResponse(body io.Reader, stream *ai.EventStream) error {
	var result ai.AssistantMessage
	decoder := json.NewDecoder(body)

	for {
		var chunk vertexResponse
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
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
			}
		}
	}

	result.Model = "vertex"
	stream.Finish(&result)
	return nil
}

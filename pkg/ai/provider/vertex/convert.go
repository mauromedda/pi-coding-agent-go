// ABOUTME: Message format conversion for Vertex AI
// ABOUTME: Uses same Gemini format with Vertex-specific request structure

package vertex

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Vertex uses the same format as Google AI (Gemini)
type vertexRequest struct {
	Contents         []vertexContent         `json:"contents"`
	SystemInstruction *vertexContent         `json:"systemInstruction,omitempty"`
	Tools            []vertexToolDef         `json:"tools,omitempty"`
	GenerationConfig *vertexGenerationConfig `json:"generationConfig,omitempty"`
}

type vertexContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []vertexPart `json:"parts"`
}

type vertexPart struct {
	Text string `json:"text,omitempty"`
}

type vertexToolDef struct {
	FunctionDeclarations []vertexFunctionDecl `json:"functionDeclarations"`
}

type vertexFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type vertexGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type vertexResponse struct {
	Candidates []vertexCandidate `json:"candidates"`
}

type vertexCandidate struct {
	Content vertexContent `json:"content"`
}

func buildVertexRequestBody(ctx *ai.Context, opts *ai.StreamOptions) vertexRequest {
	req := vertexRequest{}

	if ctx.System != "" {
		req.SystemInstruction = &vertexContent{
			Parts: []vertexPart{{Text: ctx.System}},
		}
	}

	for _, msg := range ctx.Messages {
		content := vertexContent{
			Role: vertexRole(msg.Role),
		}
		for _, c := range msg.Content {
			if c.Type == ai.ContentText {
				content.Parts = append(content.Parts, vertexPart{Text: c.Text})
			}
		}
		req.Contents = append(req.Contents, content)
	}

	if len(ctx.Tools) > 0 {
		var decls []vertexFunctionDecl
		for _, t := range ctx.Tools {
			decls = append(decls, vertexFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		req.Tools = []vertexToolDef{{FunctionDeclarations: decls}}
	}

	if opts != nil {
		req.GenerationConfig = &vertexGenerationConfig{
			MaxOutputTokens: opts.MaxTokens,
			Temperature:     opts.Temperature,
		}
	}

	return req
}

func vertexRole(role ai.Role) string {
	switch role {
	case ai.RoleUser:
		return "user"
	case ai.RoleAssistant:
		return "model"
	default:
		return "user"
	}
}

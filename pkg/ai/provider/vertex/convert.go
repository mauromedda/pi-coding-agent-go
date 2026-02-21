// ABOUTME: Message format conversion for Vertex AI
// ABOUTME: Uses shared Gemini types with Vertex-specific request/response wrappers

package vertex

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/gemini"
)

// Vertex-specific request/response types that wrap shared Gemini types.
type vertexRequest struct {
	Contents          []gemini.Content          `json:"contents"`
	SystemInstruction *gemini.Content           `json:"systemInstruction,omitempty"`
	Tools             []gemini.ToolDef          `json:"tools,omitempty"`
	GenerationConfig  *gemini.GenerationConfig  `json:"generationConfig,omitempty"`
}

type vertexResponse struct {
	Candidates []vertexCandidate `json:"candidates"`
}

type vertexCandidate struct {
	Content gemini.Content `json:"content"`
}

func buildVertexRequestBody(ctx *ai.Context, opts *ai.StreamOptions) vertexRequest {
	req := vertexRequest{}

	if ctx.System != "" {
		req.SystemInstruction = &gemini.Content{
			Parts: []gemini.Part{{Text: ctx.System}},
		}
	}

	for _, msg := range ctx.Messages {
		content := gemini.Content{
			Role: vertexRole(msg.Role),
		}
		for _, c := range msg.Content {
			if c.Type == ai.ContentText {
				content.Parts = append(content.Parts, gemini.Part{Text: c.Text})
			}
		}
		req.Contents = append(req.Contents, content)
	}

	if len(ctx.Tools) > 0 {
		var decls []gemini.FunctionDecl
		for _, t := range ctx.Tools {
			decls = append(decls, gemini.FunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		req.Tools = []gemini.ToolDef{{FunctionDeclarations: decls}}
	}

	if opts != nil {
		req.GenerationConfig = &gemini.GenerationConfig{
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

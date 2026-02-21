// ABOUTME: Message format conversion for Google Generative AI (Gemini) API
// ABOUTME: Converts between internal types and Gemini request/response format

package google

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/gemini"
)

// Google-specific request/response types that wrap shared Gemini types.
type geminiRequest struct {
	Contents          []gemini.Content          `json:"contents"`
	SystemInstruction *gemini.Content           `json:"systemInstruction,omitempty"`
	Tools             []gemini.ToolDef          `json:"tools,omitempty"`
	GenerationConfig  *gemini.GenerationConfig  `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      gemini.Content `json:"content"`
	FinishReason string         `json:"finishReason"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func buildGeminiRequestBody(ctx *ai.Context, opts *ai.StreamOptions) geminiRequest {
	req := geminiRequest{}

	// System instruction
	if ctx.System != "" {
		req.SystemInstruction = &gemini.Content{
			Parts: []gemini.Part{{Text: ctx.System}},
		}
	}

	// Convert messages
	for _, msg := range ctx.Messages {
		content := gemini.Content{
			Role: mapRole(msg.Role),
		}
		for _, c := range msg.Content {
			switch c.Type {
			case ai.ContentText:
				content.Parts = append(content.Parts, gemini.Part{Text: c.Text})
			case ai.ContentToolUse:
				content.Parts = append(content.Parts, gemini.Part{
					FunctionCall: &gemini.FunctionCall{
						Name: c.Name,
						Args: c.Input,
					},
				})
			case ai.ContentToolResult:
				content.Parts = append(content.Parts, gemini.Part{
					FunctionResponse: &gemini.FunctionResponse{
						Name:     c.Name,
						Response: map[string]string{"result": c.ResultText},
					},
				})
			}
		}
		req.Contents = append(req.Contents, content)
	}

	// Convert tools
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

	// Generation config
	if opts != nil {
		req.GenerationConfig = &gemini.GenerationConfig{
			MaxOutputTokens: opts.MaxTokens,
			Temperature:     opts.Temperature,
			TopP:            opts.TopP,
		}
	}

	return req
}

func mapRole(role ai.Role) string {
	switch role {
	case ai.RoleUser:
		return "user"
	case ai.RoleAssistant:
		return "model"
	default:
		return "user"
	}
}

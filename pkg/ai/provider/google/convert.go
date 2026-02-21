// ABOUTME: Message format conversion for Google Generative AI (Gemini) API
// ABOUTME: Converts between internal types and Gemini request/response format

package google

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Gemini API types
type geminiRequest struct {
	Contents         []geminiContent       `json:"contents"`
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	Tools            []geminiToolDef       `json:"tools,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

type geminiToolDef struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate  `json:"candidates"`
	UsageMetadata *geminiUsage       `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
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
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: ctx.System}},
		}
	}

	// Convert messages
	for _, msg := range ctx.Messages {
		content := geminiContent{
			Role: mapRole(msg.Role),
		}
		for _, c := range msg.Content {
			switch c.Type {
			case ai.ContentText:
				content.Parts = append(content.Parts, geminiPart{Text: c.Text})
			case ai.ContentToolUse:
				content.Parts = append(content.Parts, geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: c.Name,
						Args: c.Input,
					},
				})
			case ai.ContentToolResult:
				content.Parts = append(content.Parts, geminiPart{
					FunctionResponse: &geminiFunctionResponse{
						Name:     c.Name,
						Response: map[string]string{"result": c.Content},
					},
				})
			}
		}
		req.Contents = append(req.Contents, content)
	}

	// Convert tools
	if len(ctx.Tools) > 0 {
		var decls []geminiFunctionDecl
		for _, t := range ctx.Tools {
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		req.Tools = []geminiToolDef{{FunctionDeclarations: decls}}
	}

	// Generation config
	if opts != nil {
		req.GenerationConfig = &geminiGenerationConfig{
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

// ABOUTME: Per-model pricing table and cost estimation for LLM API calls
// ABOUTME: Supports Anthropic, OpenAI, Google models with input/output token rates

package telemetry

import "strings"

// ModelPricing holds per-million-token rates for a model.
type ModelPricing struct {
	InputPerMillion  float64 // USD per million input tokens
	OutputPerMillion float64 // USD per million output tokens
}

// defaultPricing is keyed by model ID prefix.
// LookupPricing uses the longest matching prefix.
var defaultPricing = map[string]ModelPricing{
	// Anthropic
	"claude-opus-4":    {InputPerMillion: 15.0, OutputPerMillion: 75.0},
	"claude-sonnet-4":  {InputPerMillion: 3.0, OutputPerMillion: 15.0},
	"claude-haiku-4":   {InputPerMillion: 1.0, OutputPerMillion: 5.0},
	"claude-haiku-3.5": {InputPerMillion: 0.80, OutputPerMillion: 4.0},
	"claude-3-5-sonnet": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
	"claude-3-opus":     {InputPerMillion: 15.0, OutputPerMillion: 75.0},
	"claude-3-haiku":    {InputPerMillion: 0.25, OutputPerMillion: 1.25},
	// OpenAI
	"gpt-4o":      {InputPerMillion: 2.50, OutputPerMillion: 10.0},
	"gpt-4o-mini": {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"gpt-4-turbo": {InputPerMillion: 10.0, OutputPerMillion: 30.0},
	"o1":          {InputPerMillion: 15.0, OutputPerMillion: 60.0},
	"o1-mini":     {InputPerMillion: 3.0, OutputPerMillion: 12.0},
	"o3-mini":     {InputPerMillion: 1.10, OutputPerMillion: 4.40},
	// Google
	"gemini-2.0-flash": {InputPerMillion: 0.10, OutputPerMillion: 0.40},
	"gemini-1.5-pro":   {InputPerMillion: 1.25, OutputPerMillion: 5.0},
	"gemini-1.5-flash": {InputPerMillion: 0.075, OutputPerMillion: 0.30},
}

// fallbackPricing is used when the model is not in the table.
var fallbackPricing = ModelPricing{InputPerMillion: 3.0, OutputPerMillion: 15.0}

// LookupPricing returns the pricing for a model ID.
// Tries exact match first, then longest prefix match, then fallback.
func LookupPricing(modelID string) ModelPricing {
	// Exact match.
	if p, ok := defaultPricing[modelID]; ok {
		return p
	}

	// Longest prefix match.
	bestKey := ""
	for key := range defaultPricing {
		if strings.HasPrefix(modelID, key) && len(key) > len(bestKey) {
			bestKey = key
		}
	}
	if bestKey != "" {
		return defaultPricing[bestKey]
	}

	return fallbackPricing
}

// EstimateCost returns the estimated cost in USD for a given model and token counts.
func EstimateCost(modelID string, inputTokens, outputTokens int) float64 {
	p := LookupPricing(modelID)
	inputCost := float64(inputTokens) / 1_000_000 * p.InputPerMillion
	outputCost := float64(outputTokens) / 1_000_000 * p.OutputPerMillion
	return inputCost + outputCost
}

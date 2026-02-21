// ABOUTME: Built-in model definitions for all supported providers
// ABOUTME: Provides defaults for Anthropic, OpenAI, Google, and local models

package ai

// Built-in model definitions.
var (
	ModelClaude4Opus = Model{
		ID:               "claude-opus-4-20250514",
		Name:             "Claude Opus 4",
		Api:              ApiAnthropic,
		MaxTokens:        200000,
		MaxOutputTokens:  16384,
		SupportsImages:   true,
		SupportsTools:    true,
		SupportsThinking: true,
	}

	ModelClaude4Sonnet = Model{
		ID:               "claude-sonnet-4-20250514",
		Name:             "Claude Sonnet 4",
		Api:              ApiAnthropic,
		MaxTokens:        200000,
		MaxOutputTokens:  16384,
		SupportsImages:   true,
		SupportsTools:    true,
		SupportsThinking: true,
	}

	ModelClaude35Haiku = Model{
		ID:               "claude-3-5-haiku-20241022",
		Name:             "Claude 3.5 Haiku",
		Api:              ApiAnthropic,
		MaxTokens:        200000,
		MaxOutputTokens:  8192,
		SupportsImages:   true,
		SupportsTools:    true,
		SupportsThinking: false,
	}

	ModelGPT4o = Model{
		ID:              "gpt-4o",
		Name:            "GPT-4o",
		Api:             ApiOpenAI,
		MaxTokens:       128000,
		MaxOutputTokens: 16384,
		SupportsImages:  true,
		SupportsTools:   true,
	}

	ModelGPT4oMini = Model{
		ID:              "gpt-4o-mini",
		Name:            "GPT-4o Mini",
		Api:             ApiOpenAI,
		MaxTokens:       128000,
		MaxOutputTokens: 16384,
		SupportsImages:  true,
		SupportsTools:   true,
	}

	ModelGemini25Pro = Model{
		ID:              "gemini-2.5-pro",
		Name:            "Gemini 2.5 Pro",
		Api:             ApiGoogle,
		MaxTokens:       1000000,
		MaxOutputTokens: 65536,
		SupportsImages:  true,
		SupportsTools:   true,
	}
)

// BuiltinModels returns all built-in model definitions.
func BuiltinModels() []Model {
	return []Model{
		ModelClaude4Opus,
		ModelClaude4Sonnet,
		ModelClaude35Haiku,
		ModelGPT4o,
		ModelGPT4oMini,
		ModelGemini25Pro,
	}
}

// modelIndex is a pre-built map for O(1) model lookups by ID.
var modelIndex = func() map[string]*Model {
	models := BuiltinModels()
	idx := make(map[string]*Model, len(models))
	for i := range models {
		idx[models[i].ID] = &models[i]
	}
	return idx
}()

// FindModel looks up a model by ID from the built-in list.
// Returns nil if not found. O(1) via pre-built index.
func FindModel(id string) *Model {
	return modelIndex[id]
}

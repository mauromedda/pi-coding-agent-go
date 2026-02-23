// ABOUTME: Prompt caching for Anthropic: marks system prompt and tools for provider-side caching
// ABOUTME: Gated by provider type; no-op for non-Anthropic models

package ai

// ApplyPromptCaching sets cache_control on the system prompt and tools
// when the model supports prompt caching (Anthropic API only).
// It modifies the Context in-place and returns true if caching was applied.
func ApplyPromptCaching(ctx *Context, modelApi Api) bool {
	if modelApi != ApiAnthropic {
		return false
	}

	ctx.SystemCacheControl = &CacheControl{Type: "ephemeral"}

	// Mark the last tool for caching; Anthropic caches the prefix up to
	// the last block annotated with cache_control.
	if len(ctx.Tools) > 0 {
		ctx.Tools[len(ctx.Tools)-1].CacheControl = &CacheControl{Type: "ephemeral"}
	}

	return true
}

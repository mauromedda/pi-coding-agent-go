// ABOUTME: Adaptive decisions engine: pure function from profile+tokens to runtime parameters
// ABOUTME: Selects MaxOutputTokens, caching, compaction, buffer size, and skill loading strategy

package perf

// AdaptiveParams holds runtime parameters derived from model profile and token estimates.
type AdaptiveParams struct {
	MaxOutputTokens  int  // capped to available budget
	UsePromptCaching bool // true for Anthropic models
	CompactBeforeCall bool // true when input exceeds compaction threshold
	StreamBufferSize int  // larger for local (4KB), smaller for slow (512B)
	PreloadSkills    bool // true for local (fast disk), false for slow (lazy load)
}

// reserveTokens is the safety margin subtracted from available output budget.
const reserveTokens = 512

// minOutputTokens is the floor for MaxOutputTokens to avoid degenerate requests.
const minOutputTokens = 1024

// compactThreshold is the fraction of context window utilization that triggers compaction.
const compactThreshold = 0.80

// Decide computes adaptive parameters from a model profile, estimated input tokens,
// and context window size. It is a pure function: no I/O, no side effects.
func Decide(profile ModelProfile, inputTokens int, contextWindow int) AdaptiveParams {
	params := AdaptiveParams{
		UsePromptCaching: profile.SupportsPromptCaching,
		PreloadSkills:    profile.Latency == LatencyLocal,
	}

	// Adaptive MaxOutputTokens: cap to available budget
	available := contextWindow - inputTokens - reserveTokens
	if available < minOutputTokens {
		available = minOutputTokens
	}
	if available < profile.MaxOutputTokens {
		params.MaxOutputTokens = available
	} else {
		params.MaxOutputTokens = profile.MaxOutputTokens
	}

	// Pre-flight compaction trigger: compact when input exceeds threshold
	utilization := float64(inputTokens) / float64(contextWindow)
	params.CompactBeforeCall = utilization > compactThreshold

	// Stream buffer size: local models benefit from larger buffers
	switch profile.Latency {
	case LatencyLocal:
		params.StreamBufferSize = 4096
	case LatencyFast:
		params.StreamBufferSize = 2048
	default:
		params.StreamBufferSize = 512
	}

	return params
}

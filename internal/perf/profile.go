// ABOUTME: Model profile registry: combines static model metadata with runtime probe results
// ABOUTME: Process-level cache keyed by modelID+baseURL; estimates tokens/sec from latency

package perf

import (
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// ModelProfile combines static model metadata with runtime probe results.
type ModelProfile struct {
	ContextWindow         int
	MaxOutputTokens       int
	SupportsPromptCaching bool
	SupportsBatching      bool
	TokensPerSecond       float64
	Latency               LatencyClass
}

// BuildProfile merges static model metadata with runtime probe results.
func BuildProfile(model *ai.Model, probe ProbeResult) ModelProfile {
	return ModelProfile{
		ContextWindow:         model.EffectiveContextWindow(),
		MaxOutputTokens:       model.MaxOutputTokens,
		SupportsPromptCaching: model.Api == ai.ApiAnthropic,
		SupportsBatching:      false,
		TokensPerSecond:       estimateTokensPerSecond(probe.Latency),
		Latency:               probe.Latency,
	}
}

// estimateTokensPerSecond returns a rough tokens/sec estimate based on latency class.
func estimateTokensPerSecond(l LatencyClass) float64 {
	switch l {
	case LatencyLocal:
		return 100.0
	case LatencyFast:
		return 80.0
	default:
		return 40.0
	}
}

// ProfileCache is a process-level cache keyed by modelID + baseURL.
type ProfileCache struct {
	entries sync.Map
}

// cacheKey builds the lookup key from model ID and base URL.
func cacheKey(modelID, baseURL string) string {
	return modelID + "|" + baseURL
}

// Get returns a cached profile or nil on miss.
func (c *ProfileCache) Get(modelID, baseURL string) *ModelProfile {
	v, ok := c.entries.Load(cacheKey(modelID, baseURL))
	if !ok {
		return nil
	}
	p := v.(ModelProfile)
	return &p
}

// Put stores a profile in the cache.
func (c *ProfileCache) Put(modelID, baseURL string, profile ModelProfile) {
	c.entries.Store(cacheKey(modelID, baseURL), profile)
}

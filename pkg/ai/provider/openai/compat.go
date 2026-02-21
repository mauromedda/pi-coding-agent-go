// ABOUTME: Compatibility flags for local inference servers (Ollama, vLLM)
// ABOUTME: Adjusts request format for API differences in local deployments

package openai

// CompatMode defines compatibility adjustments for different API servers.
type CompatMode int

const (
	CompatStandard CompatMode = iota // Standard OpenAI API
	CompatOllama                     // Ollama-specific adjustments
	CompatVLLM                       // vLLM-specific adjustments
)

// DetectCompat determines the compatibility mode from the base URL.
func DetectCompat(baseURL string) CompatMode {
	// Ollama typically runs on port 11434
	if baseURL != "" && baseURL != defaultBaseURL {
		// Local servers get standard compat for now;
		// can be refined based on actual server behavior
		return CompatStandard
	}
	return CompatStandard
}

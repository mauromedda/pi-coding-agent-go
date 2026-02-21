// ABOUTME: Provider registry for mapping API types to provider factories
// ABOUTME: Thread-safe registration and lookup of ApiProvider implementations

package ai

import (
	"context"
	"sync"
)

// ProviderFactory creates an ApiProvider given a base URL override (optional).
type ProviderFactory func(baseURL string) ApiProvider

// ApiProvider is the interface all LLM providers implement.
type ApiProvider interface {
	// Api returns the provider's API identifier.
	Api() Api

	// Stream initiates a streaming LLM call and returns an EventStream.
	// The context.Context controls cancellation of the underlying HTTP request.
	Stream(ctx context.Context, model *Model, llmCtx *Context, opts *StreamOptions) *EventStream
}

var (
	registryMu sync.RWMutex
	registry   = make(map[Api]ProviderFactory)
)

// RegisterProvider registers a factory for the given API.
func RegisterProvider(api Api, factory ProviderFactory) {
	registryMu.Lock()
	registry[api] = factory
	registryMu.Unlock()
}

// GetProvider returns a provider for the given API and optional base URL.
// Returns nil if no provider is registered.
func GetProvider(api Api, baseURL string) ApiProvider {
	registryMu.RLock()
	factory, ok := registry[api]
	registryMu.RUnlock()
	if !ok {
		return nil
	}
	return factory(baseURL)
}

// HasProvider checks if a provider is registered for the given API.
func HasProvider(api Api) bool {
	registryMu.RLock()
	_, ok := registry[api]
	registryMu.RUnlock()
	return ok
}

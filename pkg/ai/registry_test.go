// ABOUTME: Tests for the provider registry: registration, lookup, and missing provider
// ABOUTME: Validates thread-safe provider factory registration and retrieval

package ai

import (
	"testing"
)

// stubProvider is a minimal ApiProvider for testing.
type stubProvider struct {
	api Api
}

func (s *stubProvider) Api() Api { return s.api }

func (s *stubProvider) Stream(_ *Model, _ *Context, _ *StreamOptions) *EventStream {
	return NewEventStream(1)
}

func TestRegisterAndGetProvider(t *testing.T) {
	// Use a unique API key to avoid polluting global state across tests.
	testAPI := Api("test-registry-api")

	RegisterProvider(testAPI, func(_ string) ApiProvider {
		return &stubProvider{api: testAPI}
	})

	provider := GetProvider(testAPI, "")
	if provider == nil {
		t.Fatal("GetProvider returned nil for registered API")
	}
	if provider.Api() != testAPI {
		t.Errorf("got Api %q, want %q", provider.Api(), testAPI)
	}
}

func TestGetUnregisteredProviderReturnsNil(t *testing.T) {
	provider := GetProvider(Api("nonexistent-api"), "")
	if provider != nil {
		t.Errorf("expected nil for unregistered API, got %v", provider)
	}
}

func TestHasProvider(t *testing.T) {
	testAPI := Api("test-has-provider-api")

	if HasProvider(testAPI) {
		t.Error("HasProvider returned true before registration")
	}

	RegisterProvider(testAPI, func(_ string) ApiProvider {
		return &stubProvider{api: testAPI}
	})

	if !HasProvider(testAPI) {
		t.Error("HasProvider returned false after registration")
	}
}

func TestGetProviderPassesBaseURL(t *testing.T) {
	testAPI := Api("test-baseurl-api")
	var receivedURL string

	RegisterProvider(testAPI, func(baseURL string) ApiProvider {
		receivedURL = baseURL
		return &stubProvider{api: testAPI}
	})

	GetProvider(testAPI, "https://custom.api.com")
	if receivedURL != "https://custom.api.com" {
		t.Errorf("factory received baseURL %q, want %q", receivedURL, "https://custom.api.com")
	}
}

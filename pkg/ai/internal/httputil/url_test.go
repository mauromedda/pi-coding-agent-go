// ABOUTME: Tests for NormalizeBaseURL: strips trailing /v1 to prevent double-path issues
// ABOUTME: Covers vLLM, OpenAI, empty string, and trailing slash variants

package httputil

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"strips trailing /v1", "http://host:8000/v1", "http://host:8000"},
		{"strips trailing /v1/", "http://host:8000/v1/", "http://host:8000"},
		{"no change without /v1", "http://host:8000", "http://host:8000"},
		{"no change for openai", "https://api.openai.com", "https://api.openai.com"},
		{"empty string", "", ""},
		{"strips trailing slash only", "http://host:8000/", "http://host:8000"},
		{"preserves path before /v1", "http://host:8000/api/v1", "http://host:8000/api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeBaseURL(tt.input); got != tt.want {
				t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

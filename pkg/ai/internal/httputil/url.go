// ABOUTME: URL normalization for API base URLs to prevent double-path issues
// ABOUTME: Strips trailing /v1 suffix so providers can append their own versioned paths

package httputil

import (
	"net/url"
	"strings"
)

// NormalizeBaseURL strips a trailing "/v1" (and any trailing slash) from a base URL.
// This prevents double-versioned paths like "/v1/v1/chat/completions" when the
// provider appends its own versioned path (e.g., "/v1/chat/completions").
// Only strips /v1 when it's the sole top-level path (e.g., http://host:8000/v1),
// not when it's nested (e.g., http://host/api/v1).
func NormalizeBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	baseURL = strings.TrimRight(baseURL, "/")

	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	if u.Path == "/v1" {
		u.Path = ""
		return strings.TrimRight(u.String(), "/")
	}

	return baseURL
}

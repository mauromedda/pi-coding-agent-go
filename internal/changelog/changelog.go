// ABOUTME: Embedded changelog for version history display
// ABOUTME: Uses go:embed to include CHANGELOG.md at compile time

package changelog

import _ "embed"

//go:embed CHANGELOG.md
var content string

// Get returns the embedded changelog content.
func Get() string {
	if content == "" {
		return "No changelog available."
	}
	return content
}

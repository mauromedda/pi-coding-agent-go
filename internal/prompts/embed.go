// ABOUTME: Embeds default prompt templates into the binary via go:embed
// ABOUTME: Provides compiled-in prompts as fallback when disk prompts/ dir is absent

package prompts

import "embed"

//go:embed all:templates
var embeddedFS embed.FS

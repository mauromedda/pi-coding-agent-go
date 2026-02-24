// ABOUTME: Package spec parser: npm (@scope/name@version), git URLs, local paths
// ABOUTME: Detects source type from input format and extracts name/path/tag

package pkgmanager

import (
	"path/filepath"
	"strings"
)

// ParseSpec parses a package specification string into a Spec.
// Supported formats:
//   - NPM:   "name", "@scope/name", "name@version", "@scope/name@version"
//   - Git:   "https://github.com/user/repo", "git@host:user/repo.git", "github.com/user/repo"
//   - Local: "./path", "../path", "/absolute/path"
func ParseSpec(raw string) Spec {
	raw = strings.TrimSpace(raw)

	// Local paths: start with . or /
	if strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "/") {
		name := filepath.Base(raw)
		return Spec{
			Raw:    raw,
			Source: SourceLocal,
			Name:   name,
			Path:   raw,
		}
	}

	// Git URLs: contains "github.com", "gitlab.com", "bitbucket.org",
	// starts with "git@", or contains "://"
	if isGitURL(raw) {
		name, tag := parseGitSpec(raw)
		return Spec{
			Raw:    raw,
			Source: SourceGit,
			Name:   name,
			Path:   raw,
			Tag:    tag,
		}
	}

	// NPM: everything else
	name, tag := parseNPMSpec(raw)
	return Spec{
		Raw:    raw,
		Source: SourceNPM,
		Name:   name,
		Tag:    tag,
	}
}

// isGitURL detects if a string looks like a git repository URL.
func isGitURL(s string) bool {
	if strings.Contains(s, "://") {
		return true
	}
	if strings.HasPrefix(s, "git@") {
		return true
	}
	gitHosts := []string{"github.com", "gitlab.com", "bitbucket.org"}
	for _, host := range gitHosts {
		if strings.Contains(s, host) {
			return true
		}
	}
	return false
}

// parseGitSpec extracts a repo name and optional tag from a git URL.
func parseGitSpec(raw string) (name, tag string) {
	// Split on # for tag: "url#branch"
	parts := strings.SplitN(raw, "#", 2)
	url := parts[0]
	if len(parts) > 1 {
		tag = parts[1]
	}

	// Extract repo name from URL
	url = strings.TrimSuffix(url, ".git")

	// Handle "git@host:user/repo" format
	if idx := strings.LastIndex(url, ":"); strings.HasPrefix(url, "git@") && idx > 0 {
		url = url[idx+1:]
	}

	// Handle "https://host/user/repo" or "host/user/repo"
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		name = url[idx+1:]
	} else {
		name = url
	}

	return name, tag
}

// parseNPMSpec splits an npm package name and optional version tag.
// Handles scoped packages: @scope/name@version
func parseNPMSpec(raw string) (name, tag string) {
	if strings.HasPrefix(raw, "@") {
		// Scoped: @scope/name@version
		// Find the second @ (version separator)
		rest := raw[1:]
		if idx := strings.Index(rest, "@"); idx >= 0 {
			name = raw[:idx+1] // @scope/name
			tag = rest[idx+1:] // version
			return name, tag
		}
		return raw, ""
	}

	// Unscoped: name@version
	if before, after, ok := strings.Cut(raw, "@"); ok {
		return before, after
	}

	return raw, ""
}

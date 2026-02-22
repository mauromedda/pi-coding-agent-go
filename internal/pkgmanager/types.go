// ABOUTME: Package management types: source, spec, info, manifest
// ABOUTME: Supports NPM, Git, and Local package sources with version tracking

package pkgmanager

import "time"

// Source identifies how a package was installed.
type Source int

const (
	SourceNPM Source = iota
	SourceGit
	SourceLocal
)

// String returns the human-readable name of the source.
func (s Source) String() string {
	switch s {
	case SourceNPM:
		return "npm"
	case SourceGit:
		return "git"
	case SourceLocal:
		return "local"
	default:
		return "unknown"
	}
}

// Spec represents a parsed package specification from user input.
type Spec struct {
	Raw    string // original input string
	Source Source
	Name   string // package name (npm: @scope/name, git: repo, local: dir name)
	Path   string // full path (git URL or filesystem path)
	Tag    string // version tag (npm: semver, git: branch/tag)
}

// Info represents an installed package.
type Info struct {
	Name      string `json:"name"`
	Source    Source  `json:"source"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
	Local     bool   `json:"local"` // true if project-local, false if global
}

// Manifest tracks all installed packages.
type Manifest struct {
	Packages []Info `json:"packages"`
}

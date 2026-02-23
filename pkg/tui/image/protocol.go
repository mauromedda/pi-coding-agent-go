// ABOUTME: Terminal image protocol detection (Kitty, iTerm2, half-block fallback)
// ABOUTME: Detects Kitty/Ghostty/WezTerm as Kitty-compatible; caches result via sync.Once

package image

import (
	"os"
	"strings"
	"sync"
)

// ImageProtocol identifies the terminal image rendering protocol.
type ImageProtocol int

const (
	ProtoNone   ImageProtocol = iota // No native image support; use half-block fallback
	ProtoKitty                       // Kitty graphics protocol (also Ghostty, WezTerm)
	ProtoITerm2                      // iTerm2 inline images protocol
)

// String returns the protocol name.
func (p ImageProtocol) String() string {
	switch p {
	case ProtoKitty:
		return "kitty"
	case ProtoITerm2:
		return "iterm2"
	default:
		return "none"
	}
}

// Capability describes the terminal's image and color support.
type Capability struct {
	Images    ImageProtocol
	TrueColor bool
}

var (
	detectOnce   sync.Once
	cachedCap    Capability
)

// Detect probes environment variables and returns the terminal's image capability.
// The result is cached after the first call.
func Detect() Capability {
	detectOnce.Do(func() {
		cachedCap = detect()
	})
	return cachedCap
}

// resetDetectCache clears the cached result so the next Detect call re-probes.
// Used only in tests.
func resetDetectCache() {
	detectOnce = sync.Once{}
	cachedCap = Capability{}
}

func detect() Capability {
	// 1. Kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return Capability{Images: ProtoKitty, TrueColor: true}
	}

	term := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	if term == "kitty" {
		return Capability{Images: ProtoKitty, TrueColor: true}
	}

	// 2. Ghostty (Kitty-compatible)
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || term == "ghostty" {
		return Capability{Images: ProtoKitty, TrueColor: true}
	}

	// 3. WezTerm (Kitty-compatible)
	if os.Getenv("WEZTERM_PANE") != "" || term == "wezterm" {
		return Capability{Images: ProtoKitty, TrueColor: true}
	}

	// 4. iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" || term == "iterm.app" {
		return Capability{Images: ProtoITerm2, TrueColor: true}
	}

	// 5. VSCode / Alacritty: true color but no image protocol
	if term == "vscode" || term == "alacritty" {
		return Capability{Images: ProtoNone, TrueColor: true}
	}

	// 6. Default: no image support
	return Capability{Images: ProtoNone, TrueColor: false}
}

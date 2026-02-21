// ABOUTME: Auto-detect IDE from environment variables and process inspection
// ABOUTME: Identifies VS Code, JetBrains, and other terminal emulators

package ide

import "os"

// IDE represents a detected IDE.
type IDE int

const (
	IDENone IDE = iota
	IDEVSCode
	IDEJetBrains
	IDEOther
)

// String returns the IDE name.
func (i IDE) String() string {
	switch i {
	case IDEVSCode:
		return "vscode"
	case IDEJetBrains:
		return "jetbrains"
	case IDEOther:
		return "other"
	default:
		return "none"
	}
}

// Detect checks environment variables to identify the running IDE.
func Detect() IDE {
	// VS Code detection
	if os.Getenv("VSCODE_PID") != "" ||
		os.Getenv("VSCODE_GIT_ASKPASS_MAIN") != "" ||
		os.Getenv("TERM_PROGRAM") == "vscode" {
		return IDEVSCode
	}

	// JetBrains detection
	if os.Getenv("JETBRAINS_IDE_PORT") != "" ||
		os.Getenv("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
		return IDEJetBrains
	}

	// Check TERM_PROGRAM for other known terminals
	term := os.Getenv("TERM_PROGRAM")
	switch term {
	case "iTerm.app", "Apple_Terminal", "WezTerm", "Alacritty", "kitty":
		return IDEOther
	}

	return IDENone
}

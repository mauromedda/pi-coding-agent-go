// ABOUTME: Tests for terminal image protocol detection
// ABOUTME: Verifies Kitty, iTerm2, and None detection from environment variables

package image

import (
	"os"
	"testing"
)

func withCleanEnv(t *testing.T, fn func()) {
	t.Helper()
	vars := []string{
		"KITTY_WINDOW_ID", "TERM_PROGRAM", "GHOSTTY_RESOURCES_DIR",
		"WEZTERM_PANE", "ITERM_SESSION_ID", "KITTY_PID",
	}
	saved := make(map[string]string)
	for _, v := range vars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	t.Cleanup(func() {
		for _, v := range vars {
			if saved[v] != "" {
				os.Setenv(v, saved[v])
			} else {
				os.Unsetenv(v)
			}
		}
		// Reset cached detection so next test gets a fresh probe.
		resetDetectCache()
	})
	fn()
}

func TestDetect_KittyWindowID(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("KITTY_WINDOW_ID", "1")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty, got %v", cap.Images)
		}
	})
}

func TestDetect_TermProgramKitty(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "kitty")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty, got %v", cap.Images)
		}
	})
}

func TestDetect_Ghostty(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("GHOSTTY_RESOURCES_DIR", "/usr/share/ghostty")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty for Ghostty, got %v", cap.Images)
		}
	})
}

func TestDetect_TermProgramGhostty(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "ghostty")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty for ghostty TERM_PROGRAM, got %v", cap.Images)
		}
	})
}

func TestDetect_WezTerm(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("WEZTERM_PANE", "0")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty for WezTerm, got %v", cap.Images)
		}
	})
}

func TestDetect_TermProgramWezTerm(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "wezterm")
		cap := Detect()
		if cap.Images != ProtoKitty {
			t.Errorf("expected ProtoKitty for wezterm TERM_PROGRAM, got %v", cap.Images)
		}
	})
}

func TestDetect_ITerm2SessionID(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("ITERM_SESSION_ID", "w0t0p0:12345")
		cap := Detect()
		if cap.Images != ProtoITerm2 {
			t.Errorf("expected ProtoITerm2, got %v", cap.Images)
		}
	})
}

func TestDetect_TermProgramITerm(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "iTerm.app")
		cap := Detect()
		if cap.Images != ProtoITerm2 {
			t.Errorf("expected ProtoITerm2 for iTerm.app, got %v", cap.Images)
		}
	})
}

func TestDetect_VSCode(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "vscode")
		cap := Detect()
		if cap.Images != ProtoNone {
			t.Errorf("expected ProtoNone for vscode, got %v", cap.Images)
		}
		if !cap.TrueColor {
			t.Error("expected TrueColor for vscode")
		}
	})
}

func TestDetect_Alacritty(t *testing.T) {
	withCleanEnv(t, func() {
		os.Setenv("TERM_PROGRAM", "alacritty")
		cap := Detect()
		if cap.Images != ProtoNone {
			t.Errorf("expected ProtoNone for alacritty, got %v", cap.Images)
		}
		if !cap.TrueColor {
			t.Error("expected TrueColor for alacritty")
		}
	})
}

func TestDetect_DefaultNone(t *testing.T) {
	withCleanEnv(t, func() {
		cap := Detect()
		if cap.Images != ProtoNone {
			t.Errorf("expected ProtoNone as default, got %v", cap.Images)
		}
	})
}

func TestImageProtocol_String(t *testing.T) {
	tests := []struct {
		proto ImageProtocol
		want  string
	}{
		{ProtoNone, "none"},
		{ProtoKitty, "kitty"},
		{ProtoITerm2, "iterm2"},
		{ImageProtocol(99), "none"},
	}
	for _, tt := range tests {
		if got := tt.proto.String(); got != tt.want {
			t.Errorf("ImageProtocol(%d).String() = %q, want %q", tt.proto, got, tt.want)
		}
	}
}

// ABOUTME: Tests for the lipgloss style bridge from theme.Color ANSI codes
// ABOUTME: Verifies extractColor parsing and style builder output

package btea

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

func TestExtractColor_256Color(t *testing.T) {
	tests := []struct {
		name string
		ansi string
		want string
	}{
		{"fg 256 orange", "\x1b[38;5;208m", "208"},
		{"fg 256 dark", "\x1b[38;5;240m", "240"},
		{"bg 256", "\x1b[48;5;236m", "236"},
		{"fg 256 high", "\x1b[38;5;214m", "214"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractColor(tt.ansi)
			if got != tt.want {
				t.Errorf("extractColor(%q) = %q; want %q", tt.ansi, got, tt.want)
			}
		})
	}
}

func TestExtractColor_BasicFg(t *testing.T) {
	tests := []struct {
		name string
		ansi string
		want string
	}{
		{"red", "\x1b[31m", "1"},
		{"green", "\x1b[32m", "2"},
		{"yellow", "\x1b[33m", "3"},
		{"cyan", "\x1b[36m", "6"},
		{"black", "\x1b[30m", "0"},
		{"white bright", "\x1b[97m", "15"},
		{"bright black", "\x1b[90m", "8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractColor(tt.ansi)
			if got != tt.want {
				t.Errorf("extractColor(%q) = %q; want %q", tt.ansi, got, tt.want)
			}
		})
	}
}

func TestExtractColor_BasicBg(t *testing.T) {
	tests := []struct {
		name string
		ansi string
		want string
	}{
		{"bg dark gray", "\x1b[100m", "8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractColor(tt.ansi)
			if got != tt.want {
				t.Errorf("extractColor(%q) = %q; want %q", tt.ansi, got, tt.want)
			}
		})
	}
}

func TestExtractColor_CompoundTakesLast(t *testing.T) {
	// Compound codes like "\x1b[1m\x1b[97m" should extract the color
	// from the last color-bearing sequence.
	got := extractColor("\x1b[1m\x1b[97m")
	if got != "15" {
		t.Errorf("extractColor(bold+white) = %q; want %q", got, "15")
	}
}

func TestExtractColor_NoColor(t *testing.T) {
	tests := []struct {
		name string
		ansi string
	}{
		{"reset", "\x1b[0m"},
		{"bold only", "\x1b[1m"},
		{"dim only", "\x1b[2m"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractColor(tt.ansi)
			if got != "" {
				t.Errorf("extractColor(%q) = %q; want empty", tt.ansi, got)
			}
		})
	}
}

func TestExtractAttrs(t *testing.T) {
	tests := []struct {
		name      string
		ansi      string
		bold      bool
		dim       bool
		italic    bool
		underline bool
		reverse   bool
	}{
		{"bold", "\x1b[1m", true, false, false, false, false},
		{"dim", "\x1b[2m", false, true, false, false, false},
		{"italic", "\x1b[3m", false, false, true, false, false},
		{"underline", "\x1b[4m", false, false, false, true, false},
		{"reverse", "\x1b[7m", false, false, false, false, true},
		{"bold+white", "\x1b[1m\x1b[97m", true, false, false, false, false},
		{"bold+underline", "\x1b[1m\x1b[4m", true, false, false, true, false},
		{"none", "\x1b[38;5;208m", false, false, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := extractAttrs(tt.ansi)
			if a.bold != tt.bold {
				t.Errorf("bold = %v; want %v", a.bold, tt.bold)
			}
			if a.dim != tt.dim {
				t.Errorf("dim = %v; want %v", a.dim, tt.dim)
			}
			if a.italic != tt.italic {
				t.Errorf("italic = %v; want %v", a.italic, tt.italic)
			}
			if a.underline != tt.underline {
				t.Errorf("underline = %v; want %v", a.underline, tt.underline)
			}
			if a.reverse != tt.reverse {
				t.Errorf("reverse = %v; want %v", a.reverse, tt.reverse)
			}
		})
	}
}

func TestIsBackground(t *testing.T) {
	tests := []struct {
		name string
		ansi string
		want bool
	}{
		{"fg 256", "\x1b[38;5;208m", false},
		{"bg 256", "\x1b[48;5;236m", true},
		{"bg basic", "\x1b[100m", true},
		{"fg basic", "\x1b[31m", false},
		{"bold only", "\x1b[1m", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBackground(tt.ansi)
			if got != tt.want {
				t.Errorf("isBackground(%q) = %v; want %v", tt.ansi, got, tt.want)
			}
		})
	}
}

func TestColorToStyle_ProducesNonEmpty(t *testing.T) {
	// Verify that colorToStyle produces styles that render non-empty.
	tests := []struct {
		name string
		ansi string
	}{
		{"fg 256", "\x1b[38;5;208m"},
		{"fg basic", "\x1b[36m"},
		{"bg 256", "\x1b[48;5;236m"},
		{"bold", "\x1b[1m"},
		{"reverse", "\x1b[7m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := colorToStyle(tt.ansi)
			rendered := s.Render("x")
			if rendered == "" {
				t.Errorf("colorToStyle(%q).Render('x') is empty", tt.ansi)
			}
		})
	}
}

func TestStyles_ProducesOutput(t *testing.T) {
	// Verify Styles() returns styles that render visible text.
	s := Styles()
	rendered := s.Accent.Render("test")
	if rendered == "" {
		t.Error("Styles().Accent.Render('test') is empty")
	}
	rendered = s.UserBg.Render("test")
	if rendered == "" {
		t.Error("Styles().UserBg.Render('test') is empty")
	}
}

func TestStylesCacheHit(t *testing.T) {
	// Two consecutive calls should return identical styles without recomputation.
	s1 := Styles()
	s2 := Styles()

	// Compare by rendering: identical theme → identical output
	if s1.Accent.Render("x") != s2.Accent.Render("x") {
		t.Error("consecutive Styles() calls should return equivalent styles")
	}
	if s1.Primary.Render("x") != s2.Primary.Render("x") {
		t.Error("consecutive Styles() calls should return equivalent Primary")
	}
}

// --- WS4: Pre-computed style fields ---

func TestThemeStyles_HasAssistantBorder(t *testing.T) {
	s := Styles()
	rendered := s.AssistantBorder.Render("│")
	if rendered == "" {
		t.Error("AssistantBorder.Render should produce non-empty output")
	}
}

func TestThemeStyles_HasAssistantError(t *testing.T) {
	s := Styles()
	rendered := s.AssistantError.Render("error text")
	if rendered == "" {
		t.Error("AssistantError.Render should produce non-empty output")
	}
}

func TestThemeStyles_HasOverlayBorder(t *testing.T) {
	s := Styles()
	rendered := s.OverlayBorder.Render("╭─╮")
	if rendered == "" {
		t.Error("OverlayBorder.Render should produce non-empty output")
	}
}

func TestThemeStyles_HasOverlayTitle(t *testing.T) {
	s := Styles()
	rendered := s.OverlayTitle.Render("Title")
	if rendered == "" {
		t.Error("OverlayTitle.Render should produce non-empty output")
	}
}

func BenchmarkAssistantMsgView(b *testing.B) {
	m := NewAssistantMsgModel()
	m.width = 120
	m.Update(AgentTextMsg{Text: "This is a moderately long response that exercises the wrapping logic and style rendering path in the assistant message view."})
	m.Update(AgentToolStartMsg{ToolID: "t1", ToolName: "Read", Args: map[string]any{"path": "/tmp/test.go"}})
	m.Update(AgentToolEndMsg{ToolID: "t1", Text: "file contents here"})

	b.ResetTimer()
	for range b.N {
		_ = m.View()
	}
}

func TestStylesCacheInvalidatedOnThemeChange(t *testing.T) {
	// Verify cache invalidates when theme pointer changes.
	_ = Styles() // prime cache

	original := theme.Current()
	defer theme.Set(original)

	// Create a distinct theme pointer by setting a different builtin
	mono := theme.Builtin("monochrome")
	if mono == nil {
		t.Skip("monochrome theme not available")
	}
	theme.Set(mono)

	// After theme change, Styles() should rebuild (cache miss)
	s := Styles()

	// Verify it built from the monochrome palette by checking that
	// the returned styles are consistent with the current theme
	expected := buildStyles(theme.Current())
	if s.Primary.Render("x") != expected.Primary.Render("x") {
		t.Error("Styles() after theme change should match buildStyles(Current())")
	}
}

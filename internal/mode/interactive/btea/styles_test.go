// ABOUTME: Tests for the lipgloss style bridge from theme.Color ANSI codes
// ABOUTME: Verifies extractColor parsing and style builder output

package btea

import "testing"

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

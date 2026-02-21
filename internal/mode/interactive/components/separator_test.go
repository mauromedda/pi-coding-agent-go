// ABOUTME: Tests for Separator component: width correctness, ANSI output, edge cases
// ABOUTME: Table-driven tests verify rendering at various widths including zero and negative

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		width int
		check func(t *testing.T, lines []string)
	}{
		{
			name:  "renders_full_width",
			width: 40,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 1 {
					t.Fatalf("expected 1 line, got %d", len(lines))
				}
				visible := width.StripANSI(lines[0])
				if len([]rune(visible)) != 40 {
					t.Errorf("expected 40 visible chars, got %d: %q", len([]rune(visible)), visible)
				}
				expected := strings.Repeat("─", 40)
				if visible != expected {
					t.Errorf("visible content = %q, want %q", visible, expected)
				}
			},
		},
		{
			name:  "renders_narrow",
			width: 5,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 1 {
					t.Fatalf("expected 1 line, got %d", len(lines))
				}
				visible := width.StripANSI(lines[0])
				if width.VisibleWidth(lines[0]) != 5 {
					t.Errorf("expected visible width 5, got %d", width.VisibleWidth(lines[0]))
				}
				expected := strings.Repeat("─", 5)
				if visible != expected {
					t.Errorf("visible content = %q, want %q", visible, expected)
				}
			},
		},
		{
			name:  "zero_width_safe",
			width: 0,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				// Should not panic; may output empty line or no lines
				if len(lines) > 1 {
					t.Errorf("expected at most 1 line for zero width, got %d", len(lines))
				}
			},
		},
		{
			name:  "negative_width_safe",
			width: -1,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				// Should not panic
				if len(lines) > 1 {
					t.Errorf("expected at most 1 line for negative width, got %d", len(lines))
				}
			},
		},
		{
			name:  "dim_ansi_present",
			width: 10,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 1 {
					t.Fatalf("expected 1 line, got %d", len(lines))
				}
				line := lines[0]
				if !strings.Contains(line, "\x1b[2m") {
					t.Error("output should contain dim ANSI code \\x1b[2m")
				}
				if !strings.Contains(line, "\x1b[0m") {
					t.Error("output should contain reset ANSI code \\x1b[0m")
				}
			},
		},
		{
			name:  "single_line",
			width: 80,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 1 {
					t.Errorf("expected exactly 1 line, got %d", len(lines))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sep := NewSeparator()
			buf := tui.AcquireBuffer()
			defer tui.ReleaseBuffer(buf)

			sep.Render(buf, tt.width)
			tt.check(t, buf.Lines)
		})
	}
}

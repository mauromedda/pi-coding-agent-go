// ABOUTME: Tests for Footer component: two-line layout, right-alignment, backward compat
// ABOUTME: Table-driven tests verify rendering, padding, ANSI codes, and edge cases

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestFooter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(f *Footer)
		width int
		check func(t *testing.T, lines []string)
	}{
		{
			name: "two_line_output",
			setup: func(f *Footer) {
				f.SetLine1("line one")
				f.SetLine2("left", "right")
			},
			width: 40,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
			},
		},
		{
			name: "line1_content",
			setup: func(f *Footer) {
				f.SetLine1("~/projects (main)")
			},
			width: 40,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 1 {
					t.Fatal("expected at least 1 line")
				}
				stripped := width.StripANSI(lines[0])
				if !strings.Contains(stripped, "~/projects (main)") {
					t.Errorf("line1 should contain %q, got %q", "~/projects (main)", stripped)
				}
			},
		},
		{
			name: "line2_right_alignment",
			setup: func(f *Footer) {
				f.SetLine2("\u2191" + "12k " + "\u2193" + "8k", "gpt-4o")
			},
			width: 40,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
				stripped := width.StripANSI(lines[1])
				if !strings.HasSuffix(stripped, "gpt-4o") {
					t.Errorf("line2 should end with %q, got %q", "gpt-4o", stripped)
				}
			},
		},
		{
			name: "line2_padding",
			setup: func(f *Footer) {
				f.SetLine2("left", "right")
			},
			width: 20,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
				stripped := width.StripANSI(lines[1])
				visW := width.VisibleWidth(lines[1])
				if visW != 20 {
					t.Errorf("line2 visible width = %d, want 20; stripped = %q", visW, stripped)
				}
				// "left" (4) + spaces (11) + "right" (5) = 20
				if !strings.HasPrefix(stripped, "left") {
					t.Errorf("line2 should start with %q, got %q", "left", stripped)
				}
				if !strings.HasSuffix(stripped, "right") {
					t.Errorf("line2 should end with %q, got %q", "right", stripped)
				}
			},
		},
		{
			name: "narrow_width_graceful",
			setup: func(f *Footer) {
				f.SetLine2("long-left-text", "long-right-text")
			},
			width: 10,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				// Must not panic; at least 1 space between left and right
				if len(lines) < 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
				stripped := width.StripANSI(lines[1])
				if !strings.Contains(stripped, " ") {
					t.Error("line2 should have at least 1 space between left and right")
				}
			},
		},
		{
			name: "backward_compat_set_content",
			setup: func(f *Footer) {
				f.SetContent("old style")
			},
			width: 40,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) != 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
				stripped := width.StripANSI(lines[0])
				if !strings.Contains(stripped, "old style") {
					t.Errorf("line1 should contain %q, got %q", "old style", stripped)
				}
				// line2 should be empty/blank
				stripped2 := width.StripANSI(lines[1])
				trimmed := strings.TrimSpace(stripped2)
				if trimmed != "" {
					t.Errorf("line2 should be empty when only SetContent used, got %q", stripped2)
				}
			},
		},
		{
			name: "dim_ansi_present",
			setup: func(f *Footer) {
				f.SetLine1("test")
				f.SetLine2("a", "b")
			},
			width: 20,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("expected 2 lines, got %d", len(lines))
				}
				for i, line := range lines {
					if !strings.Contains(line, "\x1b[2m") {
						t.Errorf("line %d should contain dim ANSI code \\x1b[2m", i)
					}
					if !strings.Contains(line, "\x1b[0m") {
						t.Errorf("line %d should contain reset ANSI code \\x1b[0m", i)
					}
				}
			},
		},
		{
			name: "content_getter",
			setup: func(f *Footer) {
				f.SetContent("test")
			},
			width: 20,
			check: func(t *testing.T, lines []string) {
				t.Helper()
				// This is checked outside of render; we need access to the footer.
				// The table-driven structure calls check on lines, so we verify
				// Content() works by checking it was set via SetContent → line1.
				stripped := width.StripANSI(lines[0])
				if !strings.Contains(stripped, "test") {
					t.Errorf("line1 should contain %q after SetContent", "test")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := NewFooter()
			tt.setup(f)

			buf := tui.AcquireBuffer()
			defer tui.ReleaseBuffer(buf)

			f.Render(buf, tt.width)
			tt.check(t, buf.Lines)
		})
	}

	// Separate test for Content() getter since it's not render-dependent.
	t.Run("content_getter_returns_value", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetContent("test")
		if got := f.Content(); got != "test" {
			t.Errorf("Content() = %q, want %q", got, "test")
		}
	})

	t.Run("mode_label_shown_in_line2", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetModeLabel("[PLAN] Shift+Tab -> Edit")

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		stripped := width.StripANSI(buf.Lines[1])
		if !strings.Contains(stripped, "[PLAN]") {
			t.Errorf("line2 should contain mode label, got %q", stripped)
		}
	})

	t.Run("context_pct_displayed_in_line2", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetContextPct(42)

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		stripped := width.StripANSI(buf.Lines[1])
		if !strings.Contains(stripped, "ctx 42%") {
			t.Errorf("line2 should contain 'ctx 42%%', got %q", stripped)
		}
	})

	t.Run("context_pct_zero_not_shown", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetContextPct(0)

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		stripped := width.StripANSI(buf.Lines[1])
		if strings.Contains(stripped, "ctx") {
			t.Errorf("line2 should not contain 'ctx' when pct is 0, got %q", stripped)
		}
	})

	t.Run("context_pct_color_dim_below_60", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetContextPct(30)

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		line2 := buf.Lines[1]
		// Should contain dim ANSI for ctx (no yellow or red)
		if strings.Contains(line2, "\x1b[33mctx") {
			t.Error("ctx below 60% should NOT use yellow")
		}
		if strings.Contains(line2, "\x1b[31mctx") {
			t.Error("ctx below 60% should NOT use red")
		}
	})

	t.Run("context_pct_color_yellow_60_to_79", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetContextPct(65)

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		line2 := buf.Lines[1]
		if !strings.Contains(line2, "\x1b[33mctx 65%") {
			t.Errorf("ctx 60-79%% should use yellow (\\x1b[33m), got %q", line2)
		}
	})

	t.Run("context_pct_color_red_80_plus", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetContextPct(85)

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		line2 := buf.Lines[1]
		if !strings.Contains(line2, "\x1b[31mctx 85%") {
			t.Errorf("ctx >= 80%% should use red (\\x1b[31m), got %q", line2)
		}
	})

	t.Run("mode_label_empty_no_extra_space", func(t *testing.T) {
		t.Parallel()
		f := NewFooter()
		f.SetLine2("stats", "model")
		f.SetModeLabel("")

		buf := tui.AcquireBuffer()
		defer tui.ReleaseBuffer(buf)
		f.Render(buf, 80)

		if len(buf.Lines) < 2 {
			t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
		}
		stripped := width.StripANSI(buf.Lines[1])
		if strings.Contains(stripped, "[]") {
			t.Errorf("empty mode label should not add brackets, got %q", stripped)
		}
	})
}

// ABOUTME: Tests for the searchable prompt history component
// ABOUTME: Covers history navigation, reverse search, prefix matching, persistence

package component

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func TestHistory_NewHistory(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	if h == nil {
		t.Fatal("NewHistory returned nil")
	}
	if h.Len() != 0 {
		t.Errorf("expected 0 entries, got %d", h.Len())
	}
}

func TestHistory_AddEntry(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("first command")
	h.Add("second command")

	if h.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", h.Len())
	}
}

func TestHistory_AddDuplicate(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("same")
	h.Add("same")

	// Duplicates should be deduplicated (keep the latest)
	if h.Len() != 1 {
		t.Errorf("expected 1 entry after dedup, got %d", h.Len())
	}
}

func TestHistory_NavigateUp(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("first")
	h.Add("second")
	h.Add("third")

	h.SetFocused(true)
	h.HandleInput("\x1b[A") // up -> "third" (most recent)

	if h.Current() != "third" {
		t.Errorf("expected 'third', got %q", h.Current())
	}

	h.HandleInput("\x1b[A") // up -> "second"

	if h.Current() != "second" {
		t.Errorf("expected 'second', got %q", h.Current())
	}
}

func TestHistory_NavigateDown(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("first")
	h.Add("second")

	h.SetFocused(true)
	h.HandleInput("\x1b[A") // up -> "second"
	h.HandleInput("\x1b[A") // up -> "first"
	h.HandleInput("\x1b[B") // down -> "second"

	if h.Current() != "second" {
		t.Errorf("expected 'second', got %q", h.Current())
	}
}

func TestHistory_NavigateDownPastEnd(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("first")

	h.SetFocused(true)
	h.HandleInput("\x1b[A") // up -> "first"
	h.HandleInput("\x1b[B") // down -> empty (back to input)

	if h.Current() != "" {
		t.Errorf("expected empty after navigating past end, got %q", h.Current())
	}
}

func TestHistory_NavigateUpPastStart(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("only")

	h.SetFocused(true)
	h.HandleInput("\x1b[A") // up -> "only"
	h.HandleInput("\x1b[A") // up -> still "only"

	if h.Current() != "only" {
		t.Errorf("expected 'only', got %q", h.Current())
	}
}

func TestHistory_ReverseSearch(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("echo hello")
	h.Add("ls -la")
	h.Add("echo world")

	h.SetFocused(true)
	h.StartSearch()

	// Type search query
	h.HandleInput("e")
	h.HandleInput("c")
	h.HandleInput("h")

	match := h.Current()
	if !strings.Contains(match, "echo") {
		t.Errorf("expected match containing 'echo', got %q", match)
	}
}

func TestHistory_ReverseSearchNoMatch(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("echo hello")

	h.SetFocused(true)
	h.StartSearch()
	h.HandleInput("z")
	h.HandleInput("z")
	h.HandleInput("z")

	match := h.Current()
	if match != "" {
		t.Errorf("expected empty for no match, got %q", match)
	}
}

func TestHistory_CancelSearch(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("test")

	h.SetFocused(true)
	h.StartSearch()
	h.HandleInput("t")

	// Escape cancels search
	h.HandleInput("\x1b")

	if h.IsSearching() {
		t.Error("expected search to be cancelled after Escape")
	}
}

func TestHistory_Focus(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	if h.IsFocused() {
		t.Error("expected not focused initially")
	}
	h.SetFocused(true)
	if !h.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}
}

func TestHistory_RenderEmpty(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.SetFocused(true)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	h.Render(buf, 40)
	// Should not crash
}

func TestHistory_RenderSearch(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("echo hello")
	h.SetFocused(true)
	h.StartSearch()
	h.HandleInput("e")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	h.Render(buf, 40)

	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line during search")
	}
	joined := strings.Join(buf.Lines, "\n")
	if !strings.Contains(joined, "search") && !strings.Contains(joined, "reverse") {
		// At minimum, it should show something
		if !strings.Contains(joined, "echo") && !strings.Contains(joined, "e") {
			t.Error("expected search indicator or match in output")
		}
	}
}

func TestHistory_Invalidate(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Invalidate()
	// Should not crash
}

func TestHistory_SaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h := NewHistory()
	h.Add("cmd1")
	h.Add("cmd2")
	h.Add("cmd3")

	if err := h.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	h2 := NewHistory()
	if err := h2.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if h2.Len() != 3 {
		t.Errorf("expected 3 entries after load, got %d", h2.Len())
	}

	h2.SetFocused(true)
	h2.HandleInput("\x1b[A") // up -> most recent
	if h2.Current() != "cmd3" {
		t.Errorf("expected 'cmd3' as most recent, got %q", h2.Current())
	}
}

func TestHistory_LoadNonexistent(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	err := h.LoadFromFile("/nonexistent/path/history")

	// Loading from nonexistent file should not be an error (fresh start)
	if err != nil {
		t.Errorf("expected nil error for nonexistent file, got %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("expected 0 entries, got %d", h.Len())
	}
}

func TestHistory_SaveCreatesDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "history")

	h := NewHistory()
	h.Add("test")

	if err := h.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestHistory_PrefixMatch(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("git status")
	h.Add("git commit -m 'test'")
	h.Add("ls -la")
	h.Add("git push")

	h.SetFocused(true)
	h.StartSearch()
	h.HandleInput("g")
	h.HandleInput("i")
	h.HandleInput("t")

	match := h.Current()
	if !strings.HasPrefix(match, "git") {
		t.Errorf("expected prefix match starting with 'git', got %q", match)
	}
}

func TestHistory_ResetNavigation(t *testing.T) {
	t.Parallel()

	h := NewHistory()
	h.Add("first")
	h.Add("second")

	h.SetFocused(true)
	h.HandleInput("\x1b[A") // up -> "second"
	h.Reset()

	// After reset, navigation position should be back to "no selection"
	if h.Current() != "" {
		t.Errorf("expected empty after reset, got %q", h.Current())
	}
}

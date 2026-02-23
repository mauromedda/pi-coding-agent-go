// ABOUTME: Tests for FileMentionModel Bubble Tea leaf component
// ABOUTME: Verifies SetItems, fuzzy filtering, navigation, Reset, View rendering

package btea

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: FileMentionModel must satisfy tea.Model.
var _ tea.Model = FileMentionModel{}

func testFileInfos() []FileInfo {
	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	return []FileInfo{
		{Path: "/proj/main.go", RelPath: "main.go", Name: "main.go", Dir: ".", Size: 1024, ModTime: now, IsDir: false},
		{Path: "/proj/pkg/utils.go", RelPath: "pkg/utils.go", Name: "utils.go", Dir: "pkg", Size: 512, ModTime: now, IsDir: false},
		{Path: "/proj/internal", RelPath: "internal", Name: "internal", Dir: ".", Size: 0, ModTime: now, IsDir: true},
		{Path: "/proj/README.md", RelPath: "README.md", Name: "README.md", Dir: ".", Size: 2048, ModTime: now, IsDir: false},
	}
}

func TestFileMentionModel_Init(t *testing.T) {
	m := NewFileMentionModel("/proj")
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestFileMentionModel_SetItemsPopulatesVisible(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	if m.Count() != 4 {
		t.Errorf("Count() = %d; want 4", m.Count())
	}
	vis := m.VisibleItems()
	if len(vis) != 4 {
		t.Fatalf("VisibleItems() len = %d; want 4", len(vis))
	}
}

func TestFileMentionModel_SetFilterFuzzyFilters(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m = m.SetFilter("main")
	vis := m.VisibleItems()
	// "main" should match "main.go" via fuzzy on RelPath
	found := false
	for _, v := range vis {
		if v.RelPath == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SetFilter('main') did not match 'main.go'; got %v", vis)
	}
	// Should not match all 4 items
	if len(vis) >= 4 {
		t.Errorf("SetFilter('main') matched all items; expected fewer")
	}
}

func TestFileMentionModel_Navigation(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())

	if m.SelectedItem().RelPath != "main.go" {
		t.Fatalf("initial SelectedItem().RelPath = %q; want 'main.go'", m.SelectedItem().RelPath)
	}

	// Down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(FileMentionModel)
	if m.SelectedItem().RelPath != "pkg/utils.go" {
		t.Errorf("after down: SelectedItem().RelPath = %q; want 'pkg/utils.go'", m.SelectedItem().RelPath)
	}

	// Up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(FileMentionModel)
	if m.SelectedItem().RelPath != "main.go" {
		t.Errorf("after up: SelectedItem().RelPath = %q; want 'main.go'", m.SelectedItem().RelPath)
	}

	// Up at top: stays
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(FileMentionModel)
	if m.SelectedItem().RelPath != "main.go" {
		t.Errorf("after up at top: SelectedItem().RelPath = %q; want 'main.go'", m.SelectedItem().RelPath)
	}
}

func TestFileMentionModel_SelectedRelPath(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	if got := m.SelectedRelPath(); got != "main.go" {
		t.Errorf("SelectedRelPath() = %q; want 'main.go'", got)
	}
}

func TestFileMentionModel_SelectedRelPathEmpty(t *testing.T) {
	m := NewFileMentionModel("/proj")
	if got := m.SelectedRelPath(); got != "" {
		t.Errorf("SelectedRelPath() on empty = %q; want empty", got)
	}
}

func TestFileMentionModel_Reset(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m = m.SetFilter("main")
	if m.Count() >= 4 {
		t.Fatal("SetFilter should reduce visible count")
	}

	m = m.Reset()
	if m.Count() != 4 {
		t.Errorf("after Reset: Count() = %d; want 4", m.Count())
	}
	if m.filter != "" {
		t.Errorf("after Reset: filter = %q; want empty", m.filter)
	}
}

func TestFileMentionModel_ViewOutput(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m.width = 120
	view := m.View()

	if !strings.Contains(view, "main.go") {
		t.Errorf("View() missing 'main.go'")
	}
	if !strings.Contains(view, "pkg/utils.go") {
		t.Errorf("View() missing 'pkg/utils.go'")
	}
}

func TestFileMentionModel_ViewDirectorySuffix(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m.width = 120
	view := m.View()

	// The directory entry "internal" should appear with a trailing "/"
	if !strings.Contains(view, "internal/") {
		t.Errorf("View() missing directory trailing '/' for 'internal'")
	}
}

func TestFileMentionModel_ViewEmptyList(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "No files found") {
		t.Errorf("View() on empty list should show 'No files found'; got %q", view)
	}
}

func TestFileMentionModel_WindowSizeMsg(t *testing.T) {
	m := NewFileMentionModel("/proj")
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	w := updated.(FileMentionModel)
	if w.width != 100 {
		t.Errorf("width = %d; want 100", w.width)
	}
}

func TestFileMentionModel_SetMaxHeight(t *testing.T) {
	files := make([]FileInfo, 20)
	now := time.Now()
	for i := range files {
		files[i] = FileInfo{RelPath: "file.go", ModTime: now}
	}
	m := NewFileMentionModel("/proj")
	m = m.SetItems(files)
	m = m.SetMaxHeight(5)
	m.width = 80
	view := m.View()
	lines := strings.Split(view, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// maxHeight=5 file items + 1 header line = 6 lines max
	if len(lines) > 6 {
		t.Errorf("View() with maxHeight=5 rendered %d lines; want <= 6 (5 items + header)", len(lines))
	}
}

func TestFileMentionModel_CountEmpty(t *testing.T) {
	m := NewFileMentionModel("/proj")
	if m.Count() != 0 {
		t.Errorf("Count() on empty = %d; want 0", m.Count())
	}
}

func TestFileMentionModel_EnterSelectsItem(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated.(FileMentionModel)

	if cmd == nil {
		t.Fatal("Enter key should return a command")
	}
	msg := cmd()
	sel, ok := msg.(FileMentionSelectMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want FileMentionSelectMsg", msg)
	}
	if sel.RelPath != "main.go" {
		t.Errorf("selected = %q; want 'main.go'", sel.RelPath)
	}
}

func TestFileMentionModel_TabSelectsItem(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("Tab key should return a command")
	}
	msg := cmd()
	sel, ok := msg.(FileMentionSelectMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want FileMentionSelectMsg", msg)
	}
	if sel.RelPath != "main.go" {
		t.Errorf("selected = %q; want 'main.go'", sel.RelPath)
	}
}

func TestFileMentionModel_EscDismisses(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc key should return a command")
	}
	msg := cmd()
	if _, ok := msg.(FileMentionDismissMsg); !ok {
		t.Fatalf("cmd() returned %T; want FileMentionDismissMsg", msg)
	}
}

func TestFileMentionModel_RunesUpdateFilter(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())

	// Type "main"
	for _, r := range "main" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(FileMentionModel)
	}

	if m.filter != "main" {
		t.Errorf("filter = %q; want 'main'", m.filter)
	}
	// Should have filtered to match main.go
	found := false
	for _, v := range m.VisibleItems() {
		if v.RelPath == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("typing 'main' should show main.go in visible items")
	}
}

func TestFileMentionModel_BackspaceDeletesFilter(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m = m.SetFilter("main")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(FileMentionModel)

	if m.filter != "mai" {
		t.Errorf("filter after backspace = %q; want 'mai'", m.filter)
	}
}

func TestFileMentionModel_EnterOnEmptyListNoop(t *testing.T) {
	m := NewFileMentionModel("/proj")
	// No items loaded

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter on empty list should return nil cmd")
	}
}

func TestFileMentionModel_ViewShowsHeader(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Files") {
		t.Error("View() should contain header with 'Files'")
	}
}

func TestFileMentionModel_ViewShowsLoadingState(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m.loading = true
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "Scanning") {
		t.Error("View() with loading=true should show scanning message")
	}
}

func TestFileMentionModel_ViewShowsNoMatches(t *testing.T) {
	m := NewFileMentionModel("/proj")
	m = m.SetItems(testFileInfos())
	m = m.SetFilter("zzzznonexistent")
	m.width = 80

	view := m.View()
	if !strings.Contains(view, "No matching") {
		t.Error("View() with no matches should show 'No matching' message")
	}
}

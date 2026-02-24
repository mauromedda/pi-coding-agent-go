// ABOUTME: Tests for auto-dismissing dropdown overlays when trigger character is deleted
// ABOUTME: Covers file mention (@) and command palette (/) dismissal on backspace

package btea

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModel_FileMention_DismissedWhenAtDeleted(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 40

	// Type "@" to open file mention overlay
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	m = result.(AppModel)

	if m.overlay == nil {
		t.Fatal("overlay = nil; want FileMentionModel after typing @")
	}
	if _, ok := m.overlay.(FileMentionModel); !ok {
		t.Fatalf("overlay = %T; want FileMentionModel", m.overlay)
	}

	// Editor should have "@"
	if got := m.editor.Text(); got != "@" {
		t.Fatalf("editor = %q; want %q", got, "@")
	}

	// Backspace to delete the "@"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(AppModel)

	// Overlay should be dismissed
	if m.overlay != nil {
		t.Errorf("overlay = %T; want nil after deleting @", m.overlay)
	}
}

func TestAppModel_FileMention_StaysWhenFilterDeleted(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 40

	// Type "@" then "m" to filter
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	m = result.(AppModel)

	// Populate with items
	items := []FileInfo{
		{RelPath: "main.go", Name: "main.go", ModTime: time.Now()},
	}
	result, _ = m.Update(FileScanResultMsg{Items: items})
	m = result.(AppModel)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "@m" {
		t.Fatalf("editor = %q; want %q", got, "@m")
	}

	// Backspace deletes "m" but "@" remains -> overlay should stay
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(AppModel)

	if m.overlay == nil {
		t.Error("overlay should remain when @ is still present")
	}
}

func TestAppModel_CmdPalette_DismissedWhenSlashDeleted(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 40

	// Type "/" to open command palette
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = result.(AppModel)

	if m.overlay == nil {
		t.Fatal("overlay = nil; want CmdPaletteModel after typing /")
	}
	if _, ok := m.overlay.(CmdPaletteModel); !ok {
		t.Fatalf("overlay = %T; want CmdPaletteModel", m.overlay)
	}

	// Backspace to delete the "/"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(AppModel)

	// Overlay should be dismissed
	if m.overlay != nil {
		t.Errorf("overlay = %T; want nil after deleting /", m.overlay)
	}
}

func TestAppModel_FileMention_DismissedWithPrefixText(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 40

	// Type "check " first, then "@"
	for _, r := range "check " {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = result.(AppModel)
	}
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}})
	m = result.(AppModel)

	if m.overlay == nil {
		t.Fatal("overlay = nil; want FileMentionModel")
	}

	// Backspace to delete the "@"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(AppModel)

	// Overlay should be dismissed; editor should still have "check "
	if m.overlay != nil {
		t.Errorf("overlay = %T; want nil after deleting @", m.overlay)
	}
	if got := m.editor.Text(); got != "check " {
		t.Errorf("editor = %q; want %q", got, "check ")
	}
}

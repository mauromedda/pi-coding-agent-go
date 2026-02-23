// ABOUTME: FileMentionModel is a Bubble Tea leaf for file path autocomplete
// ABOUTME: Port of pkg/tui/component/filemention.go; fuzzy filter on RelPath, no filesystem I/O

package btea

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// FileInfo holds file metadata for display.
type FileInfo struct {
	Path    string
	RelPath string
	Name    string
	Dir     string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// FileMentionSelectMsg is returned when the user selects a file.
type FileMentionSelectMsg struct{ RelPath string }

// FileMentionDismissMsg is returned when the user dismisses the file mention.
type FileMentionDismissMsg struct{}

// FileMentionModel is a fuzzy file selector for @file mentions.
// Implements tea.Model with value semantics. No filesystem I/O; items
// are provided externally via SetItems.
type FileMentionModel struct {
	items       []FileInfo
	visible     []FileInfo
	selected    int
	scrollOff   int
	maxHeight   int
	filter      string
	projectRoot string
	width       int
	loading     bool
}

// NewFileMentionModel creates a new file mention model for the given project root.
func NewFileMentionModel(projectRoot string) FileMentionModel {
	return FileMentionModel{
		projectRoot: projectRoot,
		maxHeight:   15,
	}
}

// Init returns nil; no commands needed at startup.
func (m FileMentionModel) Init() tea.Cmd {
	return nil
}

// Update handles key and window-size messages.
func (m FileMentionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			if len(msg.Runes) > 0 {
				m.filter += string(msg.Runes)
				m.selected = 0
				m.scrollOff = 0
				m.applyFilter()
			}
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.selected = 0
				m.scrollOff = 0
				m.applyFilter()
			}
		case tea.KeyUp:
			m.moveUp()
		case tea.KeyDown:
			m.moveDown()
		case tea.KeyEnter, tea.KeyTab:
			rel := m.SelectedRelPath()
			if rel != "" {
				return m, func() tea.Msg { return FileMentionSelectMsg{RelPath: rel} }
			}
		case tea.KeyEsc:
			return m, func() tea.Msg { return FileMentionDismissMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the file list with header, loading state, and color coding.
func (m FileMentionModel) View() string {
	s := Styles()
	var b strings.Builder

	// Header line
	header := "  Files"
	if m.filter != "" {
		header += fmt.Sprintf(" matching %q", m.filter)
	}
	b.WriteString(s.Dim.Render(header))

	// Loading state
	if m.loading && len(m.items) == 0 {
		b.WriteByte('\n')
		b.WriteString(s.Dim.Render("  Scanning files..."))
		return b.String()
	}

	// No matches state
	if len(m.visible) == 0 {
		b.WriteByte('\n')
		if m.filter != "" {
			b.WriteString(s.Dim.Render("  No matching files"))
		} else {
			b.WriteString(s.Dim.Render("  No files found"))
		}
		return b.String()
	}

	end := min(m.scrollOff+m.maxHeight, len(m.visible))
	for i := m.scrollOff; i < end; i++ {
		item := m.visible[i]
		selected := i == m.selected
		line := formatFileItem(s, item, m.width, selected)
		b.WriteByte('\n')
		b.WriteString(line)
	}

	return b.String()
}

// SetItems replaces the file list and resets selection. Returns a new model.
func (m FileMentionModel) SetItems(items []FileInfo) FileMentionModel {
	m.items = items
	m.selected = 0
	m.scrollOff = 0
	m.applyFilter()
	return m
}

// SetFilter sets the fuzzy filter string and refilters. Returns a new model.
func (m FileMentionModel) SetFilter(f string) FileMentionModel {
	m.filter = f
	m.selected = 0
	m.scrollOff = 0
	m.applyFilter()
	return m
}

// SetMaxHeight limits the number of visible rows. Returns a new model.
func (m FileMentionModel) SetMaxHeight(h int) FileMentionModel {
	m.maxHeight = h
	return m
}

// SelectedItem returns the currently selected file info.
// Returns a zero-value FileInfo if the list is empty.
func (m FileMentionModel) SelectedItem() FileInfo {
	if len(m.visible) == 0 {
		return FileInfo{}
	}
	return m.visible[m.selected]
}

// SelectedRelPath returns the relative path of the selected file.
func (m FileMentionModel) SelectedRelPath() string {
	if len(m.visible) == 0 {
		return ""
	}
	return m.visible[m.selected].RelPath
}

// VisibleItems returns the currently filtered/visible items.
func (m FileMentionModel) VisibleItems() []FileInfo {
	return m.visible
}

// Reset clears the filter and selection. Returns a new model.
func (m FileMentionModel) Reset() FileMentionModel {
	m.filter = ""
	m.selected = 0
	m.scrollOff = 0
	m.applyFilter()
	return m
}

// Count returns the number of visible items.
func (m FileMentionModel) Count() int {
	return len(m.visible)
}

func (m *FileMentionModel) moveUp() {
	if m.selected > 0 {
		m.selected--
		m.adjustScroll()
	}
}

func (m *FileMentionModel) moveDown() {
	if m.selected < len(m.visible)-1 {
		m.selected++
		m.adjustScroll()
	}
}

func (m *FileMentionModel) adjustScroll() {
	if m.selected < m.scrollOff {
		m.scrollOff = m.selected
	}
	if m.selected >= m.scrollOff+m.maxHeight {
		m.scrollOff = m.selected - m.maxHeight + 1
	}
}

func (m *FileMentionModel) applyFilter() {
	if m.filter == "" {
		m.visible = make([]FileInfo, len(m.items))
		copy(m.visible, m.items)
		return
	}

	paths := make([]string, len(m.items))
	for i, item := range m.items {
		paths[i] = item.RelPath
	}
	matches := fuzzy.Find(m.filter, paths)
	m.visible = make([]FileInfo, len(matches))
	for i, match := range matches {
		m.visible[i] = m.items[match.Index]
	}
}

func formatFileItem(s ThemeStyles, item FileInfo, w int, selected bool) string {
	var line string

	if item.IsDir {
		line = fmt.Sprintf("  %s/", s.Info.Render(item.RelPath))
	} else {
		line = fmt.Sprintf("  %s", item.RelPath)
	}

	// Size formatting
	sizeStr := fmt.Sprintf("%d bytes", item.Size)
	if item.Size >= 1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(item.Size)/1024)
	}

	modTime := item.ModTime.Format("Jan 02 15:04")
	line += "  " + s.Secondary.Render(fmt.Sprintf("(%s, %s)", sizeStr, modTime))

	if w > 0 {
		line = width.TruncateToWidth(line, w)
	}

	if selected {
		line = s.Bold.Render(s.Selection.Render(line))
	}
	return line
}

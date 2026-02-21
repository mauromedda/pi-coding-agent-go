// ABOUTME: Fuzzy file selector component for Claude Code style @file mentions
// ABOUTME: Supports keyboard navigation, fuzzy search, and file selection
// ABOUTME: Used when user types @ in editor to autocomplete file paths

package component

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/fuzzy"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// FileInfo holds file metadata for display
type FileInfo struct {
	Path    string
	RelPath string // Relative to project root
	Name    string
	Dir     string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// FileMentionSelector is a fuzzy file selector for @file mentions
type FileMentionSelector struct {
	items       []FileInfo
	visible     []FileInfo
	selected    int
	scrollOff   int
	maxHeight   int
	filter      string
	dirty       bool
	projectRoot string
	baseDir     string
	mu          sync.Mutex
}

// NewFileMentionSelector creates a new file selector
func NewFileMentionSelector(projectRoot, baseDir string) *FileMentionSelector {
	fm := &FileMentionSelector{
		maxHeight:   15,
		projectRoot: projectRoot,
		baseDir:     baseDir,
		dirty:       true,
	}
	return fm
}

// ScanProject scans the project directory for files to suggest
func (fm *FileMentionSelector) ScanProject() error {
	var files []FileInfo

	err := filepath.Walk(fm.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories
		if strings.Contains(path, "/.") && !strings.HasSuffix(path, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip .git directory
		if strings.Contains(path, "/.git/") {
			return nil
		}

		// Skip common binary/compiled files for speed
		ext := filepath.Ext(path)
		if isBinaryExt(ext) {
			return nil
		}

		relPath, err := filepath.Rel(fm.projectRoot, path)
		if err != nil {
			return nil
		}

		files = append(files, FileInfo{
			Path:    path,
			RelPath: relPath,
			Name:    info.Name(),
			Dir:     filepath.Dir(relPath),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		})

		return nil
	})

	if err != nil {
		return err
	}

	fm.mu.Lock()
	fm.items = files
	fm.applyFilterLocked()
	fm.dirty = true
	fm.mu.Unlock()
	return nil
}

// isBinaryExt checks if a file extension is typically binary
func isBinaryExt(ext string) bool {
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
		".ico": true, ".svg": true, ".pdf": true, ".doc": true, ".docx": true,
		".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".bin": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".pkg": true, ".deb": true, ".rpm": true,
	}
	return binaryExts[ext]
}

// SetItems replaces the file list and refilters.
func (fm *FileMentionSelector) SetItems(items []FileInfo) {
	fm.mu.Lock()
	fm.items = items
	fm.selected = 0
	fm.scrollOff = 0
	fm.applyFilterLocked()
	fm.dirty = true
	fm.mu.Unlock()
}

// SetFilter sets the fuzzy filter string and refilters
func (fm *FileMentionSelector) SetFilter(f string) {
	fm.mu.Lock()
	fm.filter = f
	fm.selected = 0
	fm.scrollOff = 0
	fm.applyFilter()
	fm.dirty = true
	fm.mu.Unlock()
}

// SetMaxHeight sets the maximum visible items
func (fm *FileMentionSelector) SetMaxHeight(h int) {
	fm.mu.Lock()
	fm.maxHeight = h
	fm.dirty = true
	fm.mu.Unlock()
}

// SelectedItem returns the currently selected file info
func (fm *FileMentionSelector) SelectedItem() FileInfo {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if len(fm.visible) == 0 {
		return FileInfo{}
	}
	return fm.visible[fm.selected]
}

// SelectedPath returns the full path of the selected file
func (fm *FileMentionSelector) SelectedPath() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if len(fm.visible) == 0 {
		return ""
	}
	return fm.visible[fm.selected].Path
}

// SelectedRelPath returns the relative path of the selected file
func (fm *FileMentionSelector) SelectedRelPath() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if len(fm.visible) == 0 {
		return ""
	}
	return fm.visible[fm.selected].RelPath
}

// Invalidate marks the component for re-render
func (fm *FileMentionSelector) Invalidate() {
	fm.mu.Lock()
	fm.dirty = true
	fm.mu.Unlock()
}

// HandleInput processes keyboard input for navigation
func (fm *FileMentionSelector) HandleInput(data string) {
	fm.mu.Lock()
	k := key.ParseKey(data)
	switch k.Type {
	case key.KeyUp:
		fm.moveUpLocked()
	case key.KeyDown:
		fm.moveDownLocked()
	case key.KeyEnter:
		// Accept selection - handled by caller
	case key.KeyEscape:
		fm.filter = ""
		fm.selected = 0
		fm.scrollOff = 0
		fm.applyFilter()
		fm.dirty = true
	}
	fm.mu.Unlock()
}

func (fm *FileMentionSelector) moveUpLocked() {
	if fm.selected > 0 {
		fm.selected--
		fm.adjustScrollLocked()
		fm.dirty = true
	}
}

func (fm *FileMentionSelector) moveUp() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.moveUpLocked()
}

func (fm *FileMentionSelector) moveDownLocked() {
	if fm.selected < len(fm.visible)-1 {
		fm.selected++
		fm.adjustScrollLocked()
		fm.dirty = true
	}
}

func (fm *FileMentionSelector) moveDown() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.moveDownLocked()
}

func (fm *FileMentionSelector) adjustScrollLocked() {
	if fm.selected < fm.scrollOff {
		fm.scrollOff = fm.selected
	}
	if fm.selected >= fm.scrollOff+fm.maxHeight {
		fm.scrollOff = fm.selected - fm.maxHeight + 1
	}
}

func (fm *FileMentionSelector) adjustScroll() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.adjustScrollLocked()
}

func (fm *FileMentionSelector) applyFilter() {
	fm.applyFilterLocked()
	fm.dirty = true
}

func (fm *FileMentionSelector) applyFilterLocked() {
	if fm.filter == "" {
		fm.visible = make([]FileInfo, len(fm.items))
		copy(fm.visible, fm.items)
		return
	}

	// Fuzzy match on relative paths
	paths := make([]string, len(fm.items))
	for i, item := range fm.items {
		paths[i] = item.RelPath
	}

	matches := fuzzy.Find(fm.filter, paths)
	fm.visible = make([]FileInfo, len(matches))
	for i, m := range matches {
		fm.visible[i] = fm.items[m.Index]
	}
}

// Render writes the file list into the buffer
func (fm *FileMentionSelector) Render(out *tui.RenderBuffer, w int) {
	if len(fm.visible) == 0 {
		out.WriteLine("\x1b[2mNo files found\x1b[0m")
		return
	}

	end := min(fm.scrollOff+fm.maxHeight, len(fm.visible))

	for i := fm.scrollOff; i < end; i++ {
		item := fm.visible[i]
		line := fm.formatItem(item, w, i == fm.selected)
		out.WriteLine(line)
	}
}

func (fm *FileMentionSelector) formatItem(item FileInfo, w int, selected bool) string {
	// Format: relative path with color coding
	var line string

	// Color coding: directories are blue, others default
	if item.IsDir {
		line = fmt.Sprintf("  \x1b[34m%s/\x1b[0m", item.RelPath)
	} else {
		line = fmt.Sprintf("  %s", item.RelPath)
	}

	// Add size and modification time in parentheses
	sizeStr := fmt.Sprintf("%d bytes", item.Size)
	if item.Size >= 1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(item.Size)/1024)
	}
	modTime := item.ModTime.Format("Jan 02 15:04")

	line += fmt.Sprintf("  \x1b[90m(%s, %s)\x1b[0m", sizeStr, modTime)

	line = width.TruncateToWidth(line, w)

	if selected {
		line = "\x1b[1m\x1b[7m" + line + "\x1b[0m"
	}
	return line
}

// SelectionAccepted returns the relative path of the selected file.
func (fm *FileMentionSelector) SelectionAccepted() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if len(fm.visible) == 0 {
		return ""
	}
	item := fm.visible[fm.selected]
	if item.Path == "" {
		return ""
	}
	return item.RelPath
}

// Count returns the number of visible items
func (fm *FileMentionSelector) Count() int {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return len(fm.visible)
}

// Reset clears the selection and filter
func (fm *FileMentionSelector) Reset() {
	fm.mu.Lock()
	fm.filter = ""
	fm.selected = 0
	fm.scrollOff = 0
	fm.applyFilterLocked()
	fm.dirty = true
	fm.mu.Unlock()
}

// Filter returns the current filter string
func (fm *FileMentionSelector) Filter() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.filter
}

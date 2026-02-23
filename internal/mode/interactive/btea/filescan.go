// ABOUTME: Async file scanning for @file mention autocomplete
// ABOUTME: Uses git ls-files for speed; falls back to os.ReadDir for non-git dirs

package btea

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// FileScanResultMsg carries the scanned file list back to the Update loop.
type FileScanResultMsg struct {
	Items []FileInfo
}

// scanProjectFilesCmd returns a tea.Cmd that scans the project directory
// for files. Uses `git ls-files` when inside a git repo (fast, respects
// .gitignore), falls back to a shallow directory walk otherwise.
func scanProjectFilesCmd(root string) tea.Cmd {
	return func() tea.Msg {
		items := scanGitFiles(root)
		if items == nil {
			items = scanDirFiles(root)
		}
		return FileScanResultMsg{Items: items}
	}
}

// scanGitFiles runs `git ls-files` and returns FileInfo entries.
// Returns nil if git is unavailable or root is not a git repo.
func scanGitFiles(root string) []FileInfo {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return nil
	}

	items := make([]FileInfo, 0, len(lines))
	for _, rel := range lines {
		if rel == "" {
			continue
		}
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			// File listed by git but stat failed; include with zero metadata
			items = append(items, FileInfo{
				Path:    abs,
				RelPath: rel,
				Name:    filepath.Base(rel),
				Dir:     filepath.Dir(rel),
			})
			continue
		}
		items = append(items, FileInfo{
			Path:    abs,
			RelPath: rel,
			Name:    filepath.Base(rel),
			Dir:     filepath.Dir(rel),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		})
	}
	return items
}

// scanDirFiles performs a shallow walk (max 2 levels) as a fallback.
func scanDirFiles(root string) []FileInfo {
	var items []FileInfo
	maxDepth := 2

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		// Skip hidden directories and common noise
		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}

		info, _ := d.Info()
		var size int64
		var modTime time.Time
		if info != nil {
			size = info.Size()
			modTime = info.ModTime()
		}

		items = append(items, FileInfo{
			Path:    path,
			RelPath: rel,
			Name:    name,
			Dir:     filepath.Dir(rel),
			Size:    size,
			ModTime: modTime,
			IsDir:   d.IsDir(),
		})
		return nil
	})
	return items
}

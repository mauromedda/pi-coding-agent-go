// ABOUTME: Tests for async file scanning commands (git ls-files and fallback)
// ABOUTME: Verifies scanProjectFilesCmd returns FileScanResultMsg with files

package btea

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanProjectFilesCmd_ReturnsFileScanResultMsg(t *testing.T) {
	// Use the actual repo root (we know we're in a git repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up to find the repo root (go test runs from package dir)
	root := cwd
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Skip("not inside a git repository")
		}
		root = parent
	}

	cmd := scanProjectFilesCmd(root)
	msg := cmd()

	result, ok := msg.(FileScanResultMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want FileScanResultMsg", msg)
	}
	if len(result.Items) == 0 {
		t.Error("FileScanResultMsg.Items is empty; expected files from git repo")
	}

	// Verify items have RelPath set
	for _, item := range result.Items[:min(5, len(result.Items))] {
		if item.RelPath == "" {
			t.Error("item has empty RelPath")
		}
		if item.Path == "" {
			t.Error("item has empty Path")
		}
	}
}

func TestScanDirFiles_SkipsGitAndVendor(t *testing.T) {
	tmp := t.TempDir()
	// Create some files
	os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.WriteFile(filepath.Join(tmp, ".git", "config"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(tmp, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tmp, "node_modules", "pkg.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmp, ".hidden"), []byte("secret"), 0644)

	items := scanDirFiles(tmp)

	for _, item := range items {
		if strings.HasPrefix(item.RelPath, ".git") || strings.HasPrefix(item.RelPath, "node_modules") {
			t.Errorf("scanDirFiles should skip %q", item.RelPath)
		}
	}

	// Dotfiles outside .git should be included
	foundHidden := false
	foundMain := false
	for _, item := range items {
		if item.RelPath == ".hidden" {
			foundHidden = true
		}
		if item.RelPath == "main.go" {
			foundMain = true
		}
	}
	if !foundHidden {
		t.Error("scanDirFiles should include .hidden (dotfiles are allowed)")
	}
	if !foundMain {
		t.Error("scanDirFiles should include main.go")
	}
}

func TestScanDirFiles_IncludesDotfileDirs(t *testing.T) {
	tmp := t.TempDir()
	// .claude/ is a user config dir that should be scannable
	os.MkdirAll(filepath.Join(tmp, ".claude"), 0755)
	os.WriteFile(filepath.Join(tmp, ".claude", "LICENSE"), []byte("MIT"), 0644)
	os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main"), 0644)

	items := scanDirFiles(tmp)

	found := false
	for _, item := range items {
		if item.RelPath == filepath.Join(".claude", "LICENSE") {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(items))
		for i, item := range items {
			names[i] = item.RelPath
		}
		t.Errorf("scanDirFiles should include .claude/LICENSE; got: %v", names)
	}
}

func TestScanDirFiles_StillSkipsGitAndNodeModules(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "objects"), 0755)
	os.MkdirAll(filepath.Join(tmp, "node_modules", "pkg"), 0755)
	os.WriteFile(filepath.Join(tmp, ".git", "config"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmp, "node_modules", "pkg", "index.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main"), 0644)

	items := scanDirFiles(tmp)

	for _, item := range items {
		if strings.HasPrefix(item.RelPath, ".git") || strings.HasPrefix(item.RelPath, "node_modules") {
			t.Errorf("scanDirFiles should skip %q", item.RelPath)
		}
	}
}

func TestScanGitFiles_NonGitDir_ReturnsNil(t *testing.T) {
	tmp := t.TempDir()
	result := scanGitFiles(tmp)
	if result != nil {
		t.Errorf("scanGitFiles in non-git dir should return nil; got %d items", len(result))
	}
}

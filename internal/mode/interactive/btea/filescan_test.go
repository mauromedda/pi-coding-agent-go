// ABOUTME: Tests for async file scanning commands (git ls-files and fallback)
// ABOUTME: Verifies scanProjectFilesCmd returns FileScanResultMsg with files

package btea

import (
	"os"
	"path/filepath"
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

func TestScanDirFiles_SkipsHiddenAndVendor(t *testing.T) {
	tmp := t.TempDir()
	// Create some files
	os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tmp, "node_modules", "pkg.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmp, ".hidden"), []byte(""), 0644)

	items := scanDirFiles(tmp)

	for _, item := range items {
		if item.Name == ".git" || item.Name == "node_modules" || item.Name == ".hidden" {
			t.Errorf("scanDirFiles should skip %q", item.Name)
		}
	}

	found := false
	for _, item := range items {
		if item.RelPath == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("scanDirFiles should include main.go")
	}
}

func TestScanGitFiles_NonGitDir_ReturnsNil(t *testing.T) {
	tmp := t.TempDir()
	result := scanGitFiles(tmp)
	if result != nil {
		t.Errorf("scanGitFiles in non-git dir should return nil; got %d items", len(result))
	}
}

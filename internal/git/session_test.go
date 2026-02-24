// ABOUTME: Tests for session worktree lifecycle: setup, merge, discard, keep
// ABOUTME: Uses temporary git repos; exercises real git commands for integration

package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupSessionWorktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("SetupSessionWorktree: %v", err)
	}
	if sw == nil {
		t.Fatal("SetupSessionWorktree returned nil")
	}

	// Verify the worktree was created
	if _, err := os.Stat(sw.Info.Path); os.IsNotExist(err) {
		t.Errorf("worktree path %q does not exist", sw.Info.Path)
	}

	// Verify branch name starts with "pi-go/session-"
	if !strings.HasPrefix(sw.Info.Branch, "pi-go/session-") {
		t.Errorf("branch = %q; want prefix 'pi-go/session-'", sw.Info.Branch)
	}

	// Verify original branch was captured
	if sw.OrigBranch == "" {
		t.Error("OrigBranch is empty")
	}

	// Cleanup
	_ = sw.Discard()
}

func TestSetupSessionWorktree_NonGitDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sw, err := SetupSessionWorktree(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sw != nil {
		t.Error("expected nil SessionWorktree for non-git dir")
	}
}

func TestSetupSessionWorktree_SkipsInsideWorktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("first setup: %v", err)
	}

	// Try to set up inside the worktree: should return nil (no nesting)
	sw2, err := SetupSessionWorktree(sw.Info.Path)
	if err != nil {
		t.Fatalf("nested setup: %v", err)
	}
	if sw2 != nil {
		t.Error("expected nil SessionWorktree inside existing worktree")
	}

	_ = sw.Discard()
}

func TestSessionWorktree_Merge(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Create a file in the worktree
	testFile := filepath.Join(sw.Info.Path, "session-test.txt")
	if err := os.WriteFile(testFile, []byte("session work"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, sw.Info.Path, "add", "session-test.txt")
	runGit(t, sw.Info.Path, "commit", "-m", "session work")

	// Merge back
	if err := sw.Merge(); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// Verify file exists in original repo
	mergedFile := filepath.Join(repo, "session-test.txt")
	if _, err := os.Stat(mergedFile); os.IsNotExist(err) {
		t.Error("merged file does not exist in original repo")
	}
}

func TestSessionWorktree_Discard(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	wtPath := sw.Info.Path
	branch := sw.Info.Branch

	if err := sw.Discard(); err != nil {
		t.Fatalf("Discard: %v", err)
	}

	// Verify worktree directory is gone
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree path should not exist after Discard")
	}

	// Verify branch is deleted
	out := runGit(t, repo, "branch", "--list", branch)
	if strings.TrimSpace(out) != "" {
		t.Errorf("branch %q still exists after Discard", branch)
	}
}

func TestSessionWorktree_Keep(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	wtPath := sw.Info.Path
	if err := sw.Keep(); err != nil {
		t.Fatalf("Keep: %v", err)
	}

	// Worktree should still exist
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree should still exist after Keep")
	}

	// Cleanup for test
	_ = Remove(wtPath)
}

func TestIsPiGoWorktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sw, err := SetupSessionWorktree(repo)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer func() { _ = sw.Discard() }()

	if !IsPiGoWorktree(sw.Info.Path) {
		t.Error("expected IsPiGoWorktree=true for session worktree")
	}

	if IsPiGoWorktree(repo) {
		t.Error("expected IsPiGoWorktree=false for main repo")
	}

	if IsPiGoWorktree(t.TempDir()) {
		t.Error("expected IsPiGoWorktree=false for non-git dir")
	}
}

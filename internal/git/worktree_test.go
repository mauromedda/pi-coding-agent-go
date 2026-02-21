// ABOUTME: Tests for git worktree operations: create, remove, list, detect, root
// ABOUTME: Uses temporary git repos for isolation; exercises real git commands

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repo with one empty commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")
	return dir
}

// runGit runs a git command in the given directory and returns trimmed stdout.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestCreate(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	info, err := Create(repo, "test-feature")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify worktree directory was created.
	expectedPath := filepath.Join(repo, ".pi-go", "worktrees", "test-feature")
	if info.Path != expectedPath {
		t.Errorf("Path = %q, want %q", info.Path, expectedPath)
	}

	// Verify directory exists on disk.
	if _, err := os.Stat(info.Path); os.IsNotExist(err) {
		t.Errorf("worktree directory does not exist: %s", info.Path)
	}

	// Verify branch was created.
	if info.Branch != "pi-go/test-feature" {
		t.Errorf("Branch = %q, want %q", info.Branch, "pi-go/test-feature")
	}

	// Verify HEAD is a full 40-char hex hash.
	if info.Head == "" {
		t.Error("Head should not be empty")
	}
	if !isFullHexHash(info.Head) {
		t.Errorf("Head = %q, want 40-char hex hash", info.Head)
	}
}

// isFullHexHash reports whether s is a 40-character lowercase hex string.
var hexHashRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

func isFullHexHash(s string) bool {
	return hexHashRe.MatchString(s)
}

func TestCreate_InvalidName(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	cases := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"slash", "feat/bad", "invalid worktree name"},
		{"dotdot", "feat..bad", "invalid worktree name"},
		{"backslash", `feat\bad`, "invalid worktree name"},
		{"semicolon", "feat;bad", "invalid worktree name"},
		{"ampersand", "feat&bad", "invalid worktree name"},
		{"pipe", "feat|bad", "invalid worktree name"},
		{"empty", "", "invalid worktree name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Create(repo, tc.input)
			if err == nil {
				t.Fatal("expected error for invalid name")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	info, err := Create(repo, "to-remove")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify directory exists before removal.
	if _, err := os.Stat(info.Path); os.IsNotExist(err) {
		t.Fatal("worktree directory should exist before Remove")
	}

	if err := Remove(info.Path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify directory is gone.
	if _, err := os.Stat(info.Path); !os.IsNotExist(err) {
		t.Error("worktree directory should not exist after Remove")
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	// Create two worktrees.
	_, err := Create(repo, "wt-alpha")
	if err != nil {
		t.Fatalf("Create wt-alpha: %v", err)
	}
	_, err = Create(repo, "wt-beta")
	if err != nil {
		t.Fatalf("Create wt-beta: %v", err)
	}

	list, err := List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Should have 3 entries: main + 2 worktrees.
	if len(list) != 3 {
		t.Fatalf("len(List) = %d, want 3", len(list))
	}

	// Verify main worktree is first and marked as Main.
	if !list[0].Main {
		t.Error("first entry should have Main=true")
	}

	// Verify worktree branches exist in the list and Head is valid.
	branches := make(map[string]bool)
	for _, w := range list {
		branches[w.Branch] = true
		if !isFullHexHash(w.Head) {
			t.Errorf("worktree %q: Head = %q, want 40-char hex hash", w.Path, w.Head)
		}
	}
	if !branches["pi-go/wt-alpha"] {
		t.Error("expected branch pi-go/wt-alpha in list")
	}
	if !branches["pi-go/wt-beta"] {
		t.Error("expected branch pi-go/wt-beta in list")
	}
}

func TestIsWorktree_MainRepo(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	// A main git repo IS a working tree.
	if !IsWorktree(repo) {
		t.Error("expected IsWorktree=true for main repo")
	}
}

func TestIsWorktree_Worktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	info, err := Create(repo, "wt-check")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !IsWorktree(info.Path) {
		t.Error("expected IsWorktree=true for created worktree")
	}
}

func TestIsWorktree_NotGit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if IsWorktree(dir) {
		t.Error("expected IsWorktree=false for non-git directory")
	}
}

func TestRepoRoot(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	// Main repo root.
	root, err := RepoRoot(repo)
	if err != nil {
		t.Fatalf("RepoRoot (main): %v", err)
	}
	if root != repo {
		// Resolve symlinks for macOS /private/var vs /var.
		resolvedRepo, _ := filepath.EvalSymlinks(repo)
		resolvedRoot, _ := filepath.EvalSymlinks(root)
		if resolvedRoot != resolvedRepo {
			t.Errorf("RepoRoot = %q, want %q", root, repo)
		}
	}

	// Worktree should also resolve to its own path (worktrees have their own toplevel).
	info, err := Create(repo, "wt-root")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	wtRoot, err := RepoRoot(info.Path)
	if err != nil {
		t.Fatalf("RepoRoot (worktree): %v", err)
	}
	resolvedWtRoot, _ := filepath.EvalSymlinks(wtRoot)
	resolvedInfoPath, _ := filepath.EvalSymlinks(info.Path)
	if resolvedWtRoot != resolvedInfoPath {
		t.Errorf("RepoRoot (worktree) = %q, want %q", wtRoot, info.Path)
	}
}

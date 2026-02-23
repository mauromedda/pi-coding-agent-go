// ABOUTME: Tests for checkpoint stack including SaveNamed with metadata fields
// ABOUTME: Tests requiring git operations need an initialized git repo in temp dir

package ide

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temp dir with an initialized git repo and one commit.
// Returns the path. If git is not available, the test is skipped.
func initGitRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}

	// Create initial file and commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}

	return dir
}

func TestCheckpoint_SaveNamed(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Modify a tracked file so stash has something to capture
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewCheckpointStack(dir)
	if err := s.SaveNamed("pre-refactor", "before big cleanup", "edit", "main.go", "execute"); err != nil {
		t.Fatalf("SaveNamed: %v", err)
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("List returned %d; want 1", len(list))
	}

	cp := list[0]
	if cp.Name != "pre-refactor" {
		t.Errorf("Name = %q; want %q", cp.Name, "pre-refactor")
	}
	if cp.Description != "before big cleanup" {
		t.Errorf("Description = %q; want %q", cp.Description, "before big cleanup")
	}
	if cp.Mode != "execute" {
		t.Errorf("Mode = %q; want %q", cp.Mode, "execute")
	}
	if cp.ToolName != "edit" {
		t.Errorf("ToolName = %q; want %q", cp.ToolName, "edit")
	}
}

func TestCheckpoint_SaveNamed_EmptyName(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	// Modify a tracked file so stash has something to capture
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("stuff"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewCheckpointStack(dir)
	if err := s.SaveNamed("", "", "bash", "echo hi", ""); err != nil {
		t.Fatalf("SaveNamed with empty name: %v", err)
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("List returned %d; want 1", len(list))
	}

	cp := list[0]
	if cp.Name != "" {
		t.Errorf("Name = %q; want empty", cp.Name)
	}
	if cp.ToolName != "bash" {
		t.Errorf("ToolName = %q; want %q", cp.ToolName, "bash")
	}
}

func TestCheckpointStack_List_IncludesNamedFields(t *testing.T) {
	t.Parallel()

	dir := initGitRepo(t)

	s := NewCheckpointStack(dir)

	// Save a regular checkpoint: modify tracked file
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("change-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("write", "a.txt"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Save a named checkpoint: modify tracked file again
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("change-b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveNamed("named-cp", "description here", "edit", "b.txt", "plan"); err != nil {
		t.Fatalf("SaveNamed: %v", err)
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List returned %d; want 2", len(list))
	}

	// Newest first: index 0 is the named one
	if list[0].Name != "named-cp" {
		t.Errorf("list[0].Name = %q; want %q", list[0].Name, "named-cp")
	}
	if list[0].Mode != "plan" {
		t.Errorf("list[0].Mode = %q; want %q", list[0].Mode, "plan")
	}

	// Older one has empty named fields
	if list[1].Name != "" {
		t.Errorf("list[1].Name = %q; want empty", list[1].Name)
	}
}

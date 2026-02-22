// ABOUTME: Tests for all installer implementations and CLI dispatch
// ABOUTME: Uses tempdirs, local bare repos, and symlink verification

package pkgmanager

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Local installer tests
// ---------------------------------------------------------------------------

func TestLocalInstaller_Install(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	destDir := t.TempDir()

	spec := Spec{
		Source: SourceLocal,
		Name:   "my-plugin",
		Path:   srcDir,
	}

	inst := &LocalInstaller{}
	info, err := inst.Install(context.Background(), spec, destDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if info.Name != "my-plugin" {
		t.Errorf("Name = %q; want %q", info.Name, "my-plugin")
	}
	if info.Source != SourceLocal {
		t.Errorf("Source = %v; want SourceLocal", info.Source)
	}
	if info.Version != "local" {
		t.Errorf("Version = %q; want %q", info.Version, "local")
	}
	if !info.Local {
		t.Error("expected Local = true")
	}

	// Verify symlink exists and points to source.
	link := filepath.Join(destDir, "my-plugin")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}

	absSrc, _ := filepath.Abs(srcDir)
	if target != absSrc {
		t.Errorf("symlink target = %q; want %q", target, absSrc)
	}
}

func TestLocalInstaller_InstallReplacesExisting(t *testing.T) {
	t.Parallel()

	srcDir1 := t.TempDir()
	srcDir2 := t.TempDir()
	destDir := t.TempDir()

	inst := &LocalInstaller{}
	spec := Spec{Source: SourceLocal, Name: "pkg", Path: srcDir1}

	if _, err := inst.Install(context.Background(), spec, destDir); err != nil {
		t.Fatalf("first install: %v", err)
	}

	spec.Path = srcDir2
	if _, err := inst.Install(context.Background(), spec, destDir); err != nil {
		t.Fatalf("second install: %v", err)
	}

	link := filepath.Join(destDir, "pkg")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}

	absSrc2, _ := filepath.Abs(srcDir2)
	if target != absSrc2 {
		t.Errorf("symlink target = %q; want %q", target, absSrc2)
	}
}

func TestLocalInstaller_InstallBadSource(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()
	spec := Spec{Source: SourceLocal, Name: "bad", Path: "/nonexistent/path/xyz"}

	inst := &LocalInstaller{}
	_, err := inst.Install(context.Background(), spec, destDir)
	if err == nil {
		t.Fatal("expected error for nonexistent source path")
	}
}

func TestLocalInstaller_Remove(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create a symlink manually.
	link := filepath.Join(destDir, "pkg")
	if err := os.Symlink(srcDir, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	inst := &LocalInstaller{}
	spec := Spec{Source: SourceLocal, Name: "pkg"}

	if err := inst.Remove(spec, destDir); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("expected symlink to be removed")
	}
}

func TestLocalInstaller_RemoveNotSymlink(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()
	// Create a regular directory instead of a symlink.
	if err := os.Mkdir(filepath.Join(destDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	inst := &LocalInstaller{}
	spec := Spec{Source: SourceLocal, Name: "pkg"}

	err := inst.Remove(spec, destDir)
	if err == nil {
		t.Fatal("expected error when removing non-symlink")
	}
	if !strings.Contains(err.Error(), "not a symlink") {
		t.Errorf("error = %q; want to contain 'not a symlink'", err.Error())
	}
}

func TestLocalInstaller_Update(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create a symlink.
	link := filepath.Join(destDir, "pkg")
	if err := os.Symlink(srcDir, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	inst := &LocalInstaller{}
	spec := Spec{Source: SourceLocal, Name: "pkg"}

	info, err := inst.Update(context.Background(), spec, destDir)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if info.Name != "pkg" {
		t.Errorf("Name = %q; want %q", info.Name, "pkg")
	}
	if info.Path != srcDir {
		t.Errorf("Path = %q; want %q", info.Path, srcDir)
	}
}

func TestLocalInstaller_List(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()
	srcDir := t.TempDir()

	// Create a symlink and a regular directory.
	if err := os.Symlink(srcDir, filepath.Join(destDir, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(destDir, "regular"), 0o755); err != nil {
		t.Fatal(err)
	}

	inst := &LocalInstaller{}
	infos, err := inst.List(destDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("expected 1 symlink entry, got %d", len(infos))
	}
	if infos[0].Name != "linked" {
		t.Errorf("Name = %q; want %q", infos[0].Name, "linked")
	}
	if infos[0].Source != SourceLocal {
		t.Errorf("Source = %v; want SourceLocal", infos[0].Source)
	}
}

func TestLocalInstaller_ListEmpty(t *testing.T) {
	t.Parallel()

	inst := &LocalInstaller{}
	infos, err := inst.List(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 entries, got %d", len(infos))
	}
}

// ---------------------------------------------------------------------------
// Git installer tests
// ---------------------------------------------------------------------------

// setupBareRepo creates a local bare git repo with one commit for testing.
func setupBareRepo(t *testing.T) string {
	t.Helper()

	// Check git is available.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	bare := filepath.Join(t.TempDir(), "test-repo.git")

	// Create a temporary working repo, commit, then clone --bare.
	work := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}

	cmds := [][]string{
		{"git", "init", work},
		{"git", "-C", work, "config", "user.email", "test@test.com"},
		{"git", "-C", work, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v: %s: %v", args, out, err)
		}
	}

	// Create a file and commit.
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmds = [][]string{
		{"git", "-C", work, "add", "."},
		{"git", "-C", work, "commit", "-m", "initial"},
		{"git", "clone", "--bare", work, bare},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v: %s: %v", args, out, err)
		}
	}

	return bare
}

func TestGitInstaller_Install(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git test in short mode")
	}

	bare := setupBareRepo(t)
	destDir := t.TempDir()

	spec := Spec{
		Source: SourceGit,
		Name:   "test-repo",
		Path:   bare,
	}

	inst := &GitInstaller{}
	info, err := inst.Install(context.Background(), spec, destDir)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if info.Name != "test-repo" {
		t.Errorf("Name = %q; want %q", info.Name, "test-repo")
	}
	if info.Source != SourceGit {
		t.Errorf("Source = %v; want SourceGit", info.Source)
	}
	if info.Version == "" || info.Version == "unknown" {
		t.Errorf("Version = %q; want a commit hash", info.Version)
	}

	// Verify the clone dir exists with .git.
	cloneDir := filepath.Join(destDir, "test-repo")
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); err != nil {
		t.Errorf("expected .git directory in clone: %v", err)
	}
}

func TestGitInstaller_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git test in short mode")
	}

	bare := setupBareRepo(t)
	destDir := t.TempDir()

	spec := Spec{Source: SourceGit, Name: "test-repo", Path: bare}
	inst := &GitInstaller{}

	// Install first.
	if _, err := inst.Install(context.Background(), spec, destDir); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Update.
	info, err := inst.Update(context.Background(), spec, destDir)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if info.Version == "" || info.Version == "unknown" {
		t.Errorf("Version = %q; want a commit hash", info.Version)
	}
}

func TestGitInstaller_Remove(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()
	repoDir := filepath.Join(destDir, "my-repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	inst := &GitInstaller{}
	spec := Spec{Source: SourceGit, Name: "my-repo"}

	if err := inst.Remove(spec, destDir); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Error("expected repo directory to be removed")
	}
}

func TestGitInstaller_List(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()

	// Create dirs with and without .git.
	if err := os.MkdirAll(filepath.Join(destDir, "repo-a", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(destDir, "repo-b", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(destDir, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	inst := &GitInstaller{}
	infos, err := inst.List(destDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(infos) != 2 {
		t.Fatalf("expected 2 git repos, got %d", len(infos))
	}

	names := map[string]bool{}
	for _, info := range infos {
		names[info.Name] = true
		if info.Source != SourceGit {
			t.Errorf("Source = %v; want SourceGit", info.Source)
		}
	}
	if !names["repo-a"] || !names["repo-b"] {
		t.Errorf("expected repo-a and repo-b, got %v", names)
	}
}

func TestGitInstaller_ListEmpty(t *testing.T) {
	t.Parallel()

	inst := &GitInstaller{}
	infos, err := inst.List(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 entries, got %d", len(infos))
	}
}

// ---------------------------------------------------------------------------
// NPM installer tests
// ---------------------------------------------------------------------------

func TestNPMInstaller_List(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()

	// Create a package.json with dependencies.
	pkgJSON := `{
		"dependencies": {
			"lodash": "^4.17.21",
			"express": "^4.18.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(destDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := &NPMInstaller{}
	infos, err := inst.List(destDir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(infos) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(infos))
	}

	names := map[string]string{}
	for _, info := range infos {
		names[info.Name] = info.Version
		if info.Source != SourceNPM {
			t.Errorf("Source = %v; want SourceNPM", info.Source)
		}
	}
	if v, ok := names["lodash"]; !ok || v != "^4.17.21" {
		t.Errorf("lodash version = %q; want %q", v, "^4.17.21")
	}
}

func TestNPMInstaller_ListEmpty(t *testing.T) {
	t.Parallel()

	inst := &NPMInstaller{}
	infos, err := inst.List(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 entries, got %d", len(infos))
	}
}

func TestNPMInstaller_ListInvalidJSON(t *testing.T) {
	t.Parallel()

	destDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(destDir, "package.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := &NPMInstaller{}
	_, err := inst.List(destDir)
	if err == nil {
		t.Fatal("expected error for invalid package.json")
	}
}

// ---------------------------------------------------------------------------
// CLI dispatch tests
// ---------------------------------------------------------------------------

func TestRunCLI_NoArgs(t *testing.T) {
	t.Parallel()
	err := RunCLI(nil, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q; want to contain 'usage'", err.Error())
	}
}

func TestRunCLI_UnknownSubcommand(t *testing.T) {
	t.Parallel()
	err := RunCLI([]string{"frobnicate"}, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("error = %q; want to contain 'unknown subcommand'", err.Error())
	}
}

func TestRunCLI_InstallNoSpec(t *testing.T) {
	t.Parallel()
	err := RunCLI([]string{"install"}, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for install with no spec")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error = %q; want to contain 'requires'", err.Error())
	}
}

func TestRunCLI_RemoveNoSpec(t *testing.T) {
	t.Parallel()
	err := RunCLI([]string{"remove"}, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for remove with no spec")
	}
}

func TestRunCLI_UpdateNoSpec(t *testing.T) {
	t.Parallel()
	err := RunCLI([]string{"update"}, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for update with no spec")
	}
}

func TestRunCLI_ListEmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := RunCLI([]string{"list"}, dir, dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestRunCLI_InstallLocal(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	globalDir := t.TempDir()
	localDir := t.TempDir()

	// Install a local package with -l flag.
	err := RunCLI([]string{"install", "-l", srcDir}, globalDir, localDir)
	if err != nil {
		t.Fatalf("RunCLI install: %v", err)
	}

	// Verify manifest in localDir was updated.
	m, err := LoadManifest(localDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	name := filepath.Base(srcDir)
	found := m.Find(name, true)
	if found == nil {
		t.Fatalf("expected to find %q in manifest", name)
	}
	if found.Source != SourceLocal {
		t.Errorf("Source = %v; want SourceLocal", found.Source)
	}
}

func TestExtractFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		flag     string
		wantBool bool
		wantArgs []string
	}{
		{"present", []string{"-l", "foo"}, "-l", true, []string{"foo"}},
		{"absent", []string{"foo", "bar"}, "-l", false, []string{"foo", "bar"}},
		{"empty", nil, "-l", false, nil},
		{"multiple", []string{"-l", "a", "-l", "b"}, "-l", true, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, args := extractFlag(tt.args, tt.flag)
			if got != tt.wantBool {
				t.Errorf("found = %v; want %v", got, tt.wantBool)
			}
			if len(args) != len(tt.wantArgs) {
				t.Errorf("args = %v; want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestFormatSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  Source
		err   bool
	}{
		{"npm", SourceNPM, false},
		{"git", SourceGit, false},
		{"local", SourceLocal, false},
		{"NPM", SourceNPM, false},
		{"unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := FormatSource(tt.input)
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("FormatSource(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

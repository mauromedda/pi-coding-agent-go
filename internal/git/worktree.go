// ABOUTME: Git worktree operations: create, remove, list, detect, and root
// ABOUTME: Wraps git CLI commands with exec.CommandContext and 30s timeout

package git

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const gitTimeout = 30 * time.Second

// validName matches names containing only alphanumerics, hyphens, underscores, and dots
// (no path separators, no shell-special chars, no consecutive dots).
// DEPRECATED: Use isValidWorktreeName for enhanced security validation
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// WorktreeInfo holds metadata about a git worktree.
type WorktreeInfo struct {
	Path   string // absolute path to worktree
	Branch string // branch name
	Head   string // HEAD commit hash (full 40-char hex)
	Bare   bool   // true if bare
	Main   bool   // true if main working tree
}

// Create creates a new worktree at .pi-go/worktrees/<name> with branch pi-go/<name>.
// The branch is created based on HEAD. Returns info about the created worktree.
// repoDir must be the repository root (use RepoRoot to resolve).
func Create(repoDir, name string) (info WorktreeInfo, err error) {
	if err := validateName(name); err != nil {
		return WorktreeInfo{}, err
	}

	wtPath := filepath.Join(repoDir, ".pi-go", "worktrees", name)
	branch := "pi-go/" + name

	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := gitCmd(ctx, repoDir, "worktree", "add", "-b", branch, wtPath)
	if err != nil {
		return WorktreeInfo{}, fmt.Errorf("git worktree create: %w: %s", err, out)
	}

	// Clean up worktree on disk if subsequent steps fail.
	defer func() {
		if err != nil {
			_ = Remove(wtPath)
		}
	}()

	// Read full HEAD hash of the new worktree.
	head, err := gitCmd(ctx, wtPath, "rev-parse", "HEAD")
	if err != nil {
		return WorktreeInfo{}, fmt.Errorf("git worktree create: read HEAD: %w", err)
	}

	return WorktreeInfo{
		Path:   wtPath,
		Branch: branch,
		Head:   strings.TrimSpace(head),
	}, nil
}

// Remove removes a worktree at the given path using git worktree remove --force.
// The command runs from the parent directory to avoid locking the worktree CWD.
func Remove(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// Run from parent directory to avoid holding a lock on the worktree itself.
	parentDir := filepath.Dir(path)
	out, err := gitCmd(ctx, parentDir, "worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, out)
	}
	return nil
}

// List returns all worktrees for the repo at repoDir by parsing
// git worktree list --porcelain output.
func List(repoDir string) ([]WorktreeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := gitCmd(ctx, repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w: %s", err, out)
	}

	return parsePorcelain(out)
}

// IsWorktree reports whether dir is inside a git working tree.
func IsWorktree(dir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := gitCmd(ctx, dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

// RepoRoot returns the repository root for the given directory
// via git rev-parse --show-toplevel.
func RepoRoot(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	out, err := gitCmd(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("git repo root: %w: %s", err, out)
	}
	return strings.TrimSpace(out), nil
}

// validateName checks that a worktree name is safe for use as a directory
// and branch component. Uses enhanced security validation.
func validateName(name string) error {
	if !isValidWorktreeName(name) {
		return fmt.Errorf("invalid worktree name %q: must contain only alphanumerics, hyphens, underscores, and dots (max 64 chars, no consecutive dots)", name)
	}
	return nil
}

// gitCmd runs a git command with the given context and working directory.
// Returns combined stdout as a string.
func gitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	// Validate and sanitize git arguments to prevent command injection
	sanitizedArgs, err := sanitizeGitArgs(args)
	if err != nil {
		return "", fmt.Errorf("git command validation failed: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", sanitizedArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// parsePorcelain parses the output of git worktree list --porcelain into
// a slice of WorktreeInfo. The format is:
//
//	worktree /path/to/main
//	HEAD abc1234
//	branch refs/heads/main
//
//	worktree /path/to/feature
//	HEAD def5678
//	branch refs/heads/feature
func parsePorcelain(output string) ([]WorktreeInfo, error) {
	var worktrees []WorktreeInfo
	var current *WorktreeInfo
	isFirst := true

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
				Main: isFirst,
			}
			isFirst = false

		case strings.HasPrefix(line, "HEAD "):
			if current != nil {
				current.Head = strings.TrimPrefix(line, "HEAD ")
			}

		case strings.HasPrefix(line, "branch "):
			if current != nil {
				ref := strings.TrimPrefix(line, "branch ")
				// Strip refs/heads/ prefix to get branch name.
				current.Branch = strings.TrimPrefix(ref, "refs/heads/")
			}

		case line == "bare":
			if current != nil {
				current.Bare = true
			}

		case line == "":
			// Empty line separates entries; continue.
		}
	}

	// Append last entry.
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, scanner.Err()
}

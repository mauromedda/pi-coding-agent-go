// ABOUTME: Session worktree lifecycle: auto-create per session, merge/discard/keep on exit
// ABOUTME: Wraps worktree.go primitives with session naming and branch management

package git

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SessionWorktree holds state for a per-session worktree.
type SessionWorktree struct {
	Info       WorktreeInfo
	OrigBranch string // branch that was active before the worktree
	RepoRoot   string // root of the original repository
}

// SetupSessionWorktree creates a fresh worktree for the current session.
// Returns nil (no error) if the directory is not a git repo or is already
// inside a pi-go worktree (no nesting).
func SetupSessionWorktree(cwd string) (*SessionWorktree, error) {
	// Don't nest inside an existing pi-go worktree.
	if IsPiGoWorktree(cwd) {
		return nil, nil
	}

	repoRoot, err := RepoRoot(cwd)
	if err != nil {
		// Not a git repo: skip silently.
		return nil, nil
	}

	// Get current branch before creating worktree.
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	branchOut, err := gitCmd(ctx, repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("session worktree: get branch: %w", err)
	}
	origBranch := strings.TrimSpace(branchOut)

	// Generate session name: session-YYYYMMDD-HHmmss
	name := "session-" + time.Now().Format("20060102-150405")

	info, err := Create(repoRoot, name)
	if err != nil {
		return nil, fmt.Errorf("session worktree: create: %w", err)
	}

	return &SessionWorktree{
		Info:       info,
		OrigBranch: origBranch,
		RepoRoot:   repoRoot,
	}, nil
}

// IsPiGoWorktree reports whether dir is inside a pi-go-managed worktree
// (path contains ".pi-go/worktrees/").
func IsPiGoWorktree(dir string) bool {
	return strings.Contains(dir, ".pi-go/worktrees/") || strings.Contains(dir, ".pi-go\\worktrees\\")
}

// Merge checks out the original branch in the main repo, merges the
// worktree branch, removes the worktree, and deletes the branch.
func (sw *SessionWorktree) Merge() error {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// Checkout original branch in main repo.
	if _, err := gitCmd(ctx, sw.RepoRoot, "checkout", sw.OrigBranch); err != nil {
		return fmt.Errorf("session worktree merge: checkout %s: %w", sw.OrigBranch, err)
	}

	// Merge the worktree branch.
	if _, err := gitCmd(ctx, sw.RepoRoot, "merge", sw.Info.Branch, "--no-edit"); err != nil {
		return fmt.Errorf("session worktree merge: merge %s: %w", sw.Info.Branch, err)
	}

	// Remove worktree.
	if err := Remove(sw.Info.Path); err != nil {
		return fmt.Errorf("session worktree merge: remove worktree: %w", err)
	}

	// Delete branch.
	if _, err := gitCmd(ctx, sw.RepoRoot, "branch", "-d", sw.Info.Branch); err != nil {
		// Non-fatal: branch might already be gone.
		_ = err
	}

	return nil
}

// Discard removes the worktree and force-deletes the branch.
func (sw *SessionWorktree) Discard() error {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// Checkout original branch first (in case we're "on" the worktree branch).
	_, _ = gitCmd(ctx, sw.RepoRoot, "checkout", sw.OrigBranch)

	if err := Remove(sw.Info.Path); err != nil {
		return fmt.Errorf("session worktree discard: remove: %w", err)
	}

	if _, err := gitCmd(ctx, sw.RepoRoot, "branch", "-D", sw.Info.Branch); err != nil {
		// Non-fatal: branch might already be gone.
		_ = err
	}

	return nil
}

// Keep is a no-op: the worktree and branch are left in place for the user.
func (sw *SessionWorktree) Keep() error {
	return nil
}

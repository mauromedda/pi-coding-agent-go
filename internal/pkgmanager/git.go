// ABOUTME: Git installer clones/pulls repositories for package management
// ABOUTME: Supports branch/tag selection via --branch flag and scans for .git dirs

package pkgmanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitInstaller implements Installer using git clone/pull.
type GitInstaller struct{}

// Install clones a git repository into <destDir>/<name>.
// If spec.Tag is set, it clones the specific branch/tag.
func (g *GitInstaller) Install(ctx context.Context, spec Spec, destDir string) (*Info, error) {
	target := filepath.Join(destDir, spec.Name)

	args := []string{"clone"}
	if spec.Tag != "" {
		args = append(args, "--branch", spec.Tag)
	}

	// Strip the #tag fragment from the URL if present.
	url := spec.Path
	if idx := strings.Index(url, "#"); idx >= 0 {
		url = url[:idx]
	}

	args = append(args, url, target)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone %s: %w", spec.Name, err)
	}

	version := spec.Tag
	if version == "" {
		version = g.headRef(ctx, target)
	}

	return &Info{
		Name:        spec.Name,
		Source:      SourceGit,
		Path:        target,
		Version:     version,
		InstalledAt: time.Now(),
	}, nil
}

// Remove deletes the cloned repository directory.
func (g *GitInstaller) Remove(spec Spec, destDir string) error {
	target := filepath.Join(destDir, spec.Name)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("removing %s: %w", spec.Name, err)
	}
	return nil
}

// Update runs git pull in the existing repository directory.
func (g *GitInstaller) Update(ctx context.Context, spec Spec, destDir string) (*Info, error) {
	target := filepath.Join(destDir, spec.Name)

	cmd := exec.CommandContext(ctx, "git", "-C", target, "pull")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git pull %s: %w", spec.Name, err)
	}

	version := g.headRef(ctx, target)

	return &Info{
		Name:        spec.Name,
		Source:      SourceGit,
		Path:        target,
		Version:     version,
		InstalledAt: time.Now(),
	}, nil
}

// List scans destDir for subdirectories containing a .git folder.
func (g *GitInstaller) List(destDir string) ([]Info, error) {
	entries, err := os.ReadDir(destDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", destDir, err)
	}

	var infos []Info
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gitDir := filepath.Join(destDir, entry.Name(), ".git")
		if _, err := os.Stat(gitDir); err != nil {
			continue
		}
		infos = append(infos, Info{
			Name:   entry.Name(),
			Source: SourceGit,
			Path:   filepath.Join(destDir, entry.Name()),
		})
	}
	return infos, nil
}

// headRef returns the short HEAD commit hash for a repository.
func (g *GitInstaller) headRef(ctx context.Context, repoDir string) string {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

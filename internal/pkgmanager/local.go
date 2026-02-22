// ABOUTME: Local installer creates symlinks for locally-developed packages
// ABOUTME: Install symlinks source path into destDir; update is a no-op

package pkgmanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalInstaller implements Installer by symlinking local directories.
type LocalInstaller struct{}

// Install creates a symlink from <destDir>/<name> pointing to spec.Path.
func (l *LocalInstaller) Install(_ context.Context, spec Spec, destDir string) (*Info, error) {
	absPath, err := filepath.Abs(spec.Path)
	if err != nil {
		return nil, fmt.Errorf("resolving path %s: %w", spec.Path, err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("source path %s: %w", absPath, err)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating dest dir: %w", err)
	}

	link := filepath.Join(destDir, spec.Name)
	// Remove existing symlink if present.
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return nil, fmt.Errorf("removing existing link %s: %w", link, err)
		}
	}

	if err := os.Symlink(absPath, link); err != nil {
		return nil, fmt.Errorf("creating symlink %s -> %s: %w", link, absPath, err)
	}

	return &Info{
		Name:        spec.Name,
		Source:      SourceLocal,
		Path:        absPath,
		Version:     "local",
		InstalledAt: time.Now(),
		Local:       true,
	}, nil
}

// Remove removes the symlink from <destDir>/<name>.
func (l *LocalInstaller) Remove(spec Spec, destDir string) error {
	link := filepath.Join(destDir, spec.Name)

	fi, err := os.Lstat(link)
	if err != nil {
		return fmt.Errorf("stat %s: %w", link, err)
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink", link)
	}

	if err := os.Remove(link); err != nil {
		return fmt.Errorf("removing symlink %s: %w", link, err)
	}
	return nil
}

// Update is a no-op for local packages since they point to the source directory.
func (l *LocalInstaller) Update(_ context.Context, spec Spec, destDir string) (*Info, error) {
	link := filepath.Join(destDir, spec.Name)

	target, err := os.Readlink(link)
	if err != nil {
		return nil, fmt.Errorf("reading symlink %s: %w", link, err)
	}

	return &Info{
		Name:        spec.Name,
		Source:      SourceLocal,
		Path:        target,
		Version:     "local",
		InstalledAt: time.Now(),
		Local:       true,
	}, nil
}

// List scans destDir for symlinks and returns their info.
func (l *LocalInstaller) List(destDir string) ([]Info, error) {
	entries, err := os.ReadDir(destDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", destDir, err)
	}

	var infos []Info
	for _, entry := range entries {
		full := filepath.Join(destDir, entry.Name())
		fi, err := os.Lstat(full)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(full)
		if err != nil {
			continue
		}
		infos = append(infos, Info{
			Name:   entry.Name(),
			Source: SourceLocal,
			Path:   target,
			Local:  true,
		})
	}
	return infos, nil
}

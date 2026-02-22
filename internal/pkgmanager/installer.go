// ABOUTME: Installer interface for package install/remove/update/list operations
// ABOUTME: Composite installer dispatches to source-specific implementations

package pkgmanager

import "context"

// Installer defines operations for a package source.
type Installer interface {
	Install(ctx context.Context, spec Spec, destDir string) (*Info, error)
	Remove(spec Spec, destDir string) error
	Update(ctx context.Context, spec Spec, destDir string) (*Info, error)
	List(destDir string) ([]Info, error)
}

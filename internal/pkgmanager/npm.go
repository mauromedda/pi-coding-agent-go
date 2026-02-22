// ABOUTME: NPM installer wraps npm CLI for install/remove/update/list operations
// ABOUTME: Uses os/exec to run npm commands with --prefix for destination directory

package pkgmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// NPMInstaller implements Installer using the npm CLI.
type NPMInstaller struct{}

// Install runs npm install --prefix <destDir> <name>@<tag>.
func (n *NPMInstaller) Install(ctx context.Context, spec Spec, destDir string) (*Info, error) {
	pkg := spec.Name
	if spec.Tag != "" {
		pkg = spec.Name + "@" + spec.Tag
	}

	cmd := exec.CommandContext(ctx, "npm", "install", "--prefix", destDir, pkg)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("npm install %s: %w", pkg, err)
	}

	version, err := n.readInstalledVersion(spec.Name, destDir)
	if err != nil {
		version = spec.Tag
	}

	return &Info{
		Name:        spec.Name,
		Source:      SourceNPM,
		Path:        filepath.Join(destDir, "node_modules", spec.Name),
		Version:     version,
		InstalledAt: time.Now(),
	}, nil
}

// Remove runs npm uninstall --prefix <destDir> <name>.
func (n *NPMInstaller) Remove(spec Spec, destDir string) error {
	cmd := exec.Command("npm", "uninstall", "--prefix", destDir, spec.Name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm uninstall %s: %w", spec.Name, err)
	}
	return nil
}

// Update runs npm update --prefix <destDir> <name>.
func (n *NPMInstaller) Update(ctx context.Context, spec Spec, destDir string) (*Info, error) {
	cmd := exec.CommandContext(ctx, "npm", "update", "--prefix", destDir, spec.Name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("npm update %s: %w", spec.Name, err)
	}

	version, err := n.readInstalledVersion(spec.Name, destDir)
	if err != nil {
		version = spec.Tag
	}

	return &Info{
		Name:        spec.Name,
		Source:      SourceNPM,
		Path:        filepath.Join(destDir, "node_modules", spec.Name),
		Version:     version,
		InstalledAt: time.Now(),
	}, nil
}

// List reads package.json dependencies from destDir and returns installed packages.
func (n *NPMInstaller) List(destDir string) ([]Info, error) {
	pkgPath := filepath.Join(destDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %w", err)
	}

	var pkgJSON struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}

	infos := make([]Info, 0, len(pkgJSON.Dependencies))
	for name, version := range pkgJSON.Dependencies {
		infos = append(infos, Info{
			Name:    name,
			Source:  SourceNPM,
			Path:    filepath.Join(destDir, "node_modules", name),
			Version: version,
		})
	}
	return infos, nil
}

// readInstalledVersion reads the installed version from node_modules/<name>/package.json.
func (n *NPMInstaller) readInstalledVersion(name, destDir string) (string, error) {
	pkgPath := filepath.Join(destDir, "node_modules", name, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", fmt.Errorf("reading installed package.json: %w", err)
	}

	var pkgJSON struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return "", fmt.Errorf("parsing installed package.json: %w", err)
	}
	return pkgJSON.Version, nil
}

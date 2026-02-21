// ABOUTME: Standard filesystem paths for pi-go configuration and data
// ABOUTME: Resolves ~/.pi-go/ for global and .pi-go/ for project-local paths

package config

import (
	"os"
	"path/filepath"
)

const (
	globalDirName  = ".pi-go"
	projectDirName = ".pi-go"
)

// GlobalDir returns the user-global config directory (~/.pi-go/).
func GlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", globalDirName)
	}
	return filepath.Join(home, globalDirName)
}

// ProjectDir returns the project-local config directory (.pi-go/ in cwd).
func ProjectDir(projectRoot string) string {
	return filepath.Join(projectRoot, projectDirName)
}

// SessionsDir returns the sessions storage directory.
func SessionsDir() string {
	return filepath.Join(GlobalDir(), "sessions")
}

// AuthFile returns the path to the auth credentials file.
func AuthFile() string {
	return filepath.Join(GlobalDir(), "auth.json")
}

// GlobalConfigFile returns the path to the global config file.
func GlobalConfigFile() string {
	return filepath.Join(GlobalDir(), "config.json")
}

// ProjectConfigFile returns the path to the project-local config file.
func ProjectConfigFile(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "config.json")
}

// SkillsDirs returns the skill directories in resolution order
// (project-local first, then global, then Claude Code compat).
func SkillsDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "skills"),
		filepath.Join(GlobalDir(), "skills"),
	}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".claude", "skills"))
	}
	return dirs
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

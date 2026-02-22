// ABOUTME: Standard filesystem paths for pi-go configuration and data
// ABOUTME: Resolves ~/.pi-go/ for global and .pi-go/ for project-local paths

package config

import (
	"os"
	"path/filepath"
	"runtime"
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
		filepath.Join(home, ".claude", "skills"),
	}
	return dirs
}

// UserSettingsFile returns the path to the user settings file.
func UserSettingsFile() string {
	return filepath.Join(GlobalDir(), "settings.json")
}

// ProjectSettingsFile returns the path to the project settings file.
func ProjectSettingsFile(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "settings.json")
}

// ClaudeSettingsFile returns the path to Claude Code settings file.
func ClaudeSettingsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
}

// ProjectClaudeSettingsFile returns the path to project-local Claude settings file.
func ProjectClaudeSettingsFile(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "settings.json")
}

// LocalSettingsFile returns the path to the local (gitignored) settings file.
func LocalSettingsFile(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "settings.local.json")
}

// ManagedSettingsFile returns the platform-dependent managed settings path.
func ManagedSettingsFile() string {
	switch runtime.GOOS {
	case "linux":
		return "/etc/pi-go/settings.json"
	default:
		// macOS / fallback
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, "Library", "Application Support", "pi-go", "settings.json")
		}
		return filepath.Join("/etc", "pi-go", "settings.json")
	}
}

// RulesDirs returns the rules directories for a project.
func RulesDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "rules"),
		filepath.Join(home, ".claude", "rules"),
	}
	return dirs
}

// AgentsDirs returns the agents directories for a project in resolution order.
func AgentsDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "agents"),
		filepath.Join(home, ".pi-go", "agents"),
		filepath.Join(home, ".claude", "agents"),
		filepath.Join(projectRoot, ".claude", "agents"),
	}
	return dirs
}

// PromptsDirs returns the prompts directories for a project in resolution order.
func PromptsDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "prompts"),
		filepath.Join(home, ".pi-go", "prompts"),
		filepath.Join(home, ".claude", "prompts"),
		filepath.Join(projectRoot, ".claude", "prompts"),
	}
	return dirs
}

// ThemesDirs returns the themes directories for a project in resolution order.
func ThemesDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "themes"),
		filepath.Join(home, ".pi-go", "themes"),
		filepath.Join(home, ".claude", "themes"),
		filepath.Join(projectRoot, ".claude", "themes"),
	}
	return dirs
}

// ExtensionsDirs returns the extensions directories for a project in resolution order.
func ExtensionsDirs(projectRoot string) []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(ProjectDir(projectRoot), "extensions"),
		filepath.Join(home, ".pi-go", "extensions"),
		filepath.Join(home, ".claude", "extensions"),
		filepath.Join(projectRoot, ".claude", "extensions"),
	}
	return dirs
}

// PackagesDir returns the global packages directory (~/.pi-go/packages/).
func PackagesDir() string {
	return filepath.Join(GlobalDir(), "packages")
}

// PackagesDirLocal returns the project-local packages directory.
func PackagesDirLocal(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "packages")
}

// AgentsDir returns the agents directory for a project (legacy).
func AgentsDir(projectRoot string) string {
	return filepath.Join(ProjectDir(projectRoot), "agents")
}

// EnsureDir creates a directory and all parents if they don't exist.
// Uses 0o700 for directories containing sensitive data (auth, sessions).
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o700)
}

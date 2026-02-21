// ABOUTME: MCP server configuration loading from settings and .mcp.json files
// ABOUTME: Merges server definitions from user, project, and Claude compat sources

package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerConfig describes how to connect to an MCP server.
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Type    string            `json:"type,omitempty"` // "stdio" (default) or "http"
	URL     string            `json:"url,omitempty"`  // For HTTP transport
}

// MCPConfig is the top-level structure of an .mcp.json file.
type MCPConfig struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// SettingsWithMCP extends settings with MCP server definitions.
type SettingsWithMCP struct {
	MCPServers map[string]ServerConfig `json:"mcpServers,omitempty"`
}

// LoadConfig loads MCP server configurations from all sources.
// Sources checked in order (later sources override):
//  1. ~/.pi-go/settings.json → mcpServers
//  2. <project>/.mcp.json
//  3. <project>/.pi-go/settings.local.json → mcpServers
//  4. ~/.claude/settings.json → mcpServers (compat)
//  5. <project>/.claude/settings.local.json → mcpServers (compat)
func LoadConfig(projectDir, homeDir string) map[string]ServerConfig {
	merged := make(map[string]ServerConfig)

	sources := []string{
		filepath.Join(homeDir, ".pi-go", "settings.json"),
		filepath.Join(homeDir, ".claude", "settings.json"),
	}
	for _, path := range sources {
		if servers := loadServersFromSettings(path); servers != nil {
			for k, v := range servers {
				merged[k] = v
			}
		}
	}

	// .mcp.json in project root
	if servers := loadMCPJSON(filepath.Join(projectDir, ".mcp.json")); servers != nil {
		for k, v := range servers {
			merged[k] = v
		}
	}

	// Project-local settings (override)
	localSources := []string{
		filepath.Join(projectDir, ".pi-go", "settings.local.json"),
		filepath.Join(projectDir, ".claude", "settings.local.json"),
	}
	for _, path := range localSources {
		if servers := loadServersFromSettings(path); servers != nil {
			for k, v := range servers {
				merged[k] = v
			}
		}
	}

	return merged
}

func loadServersFromSettings(path string) map[string]ServerConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var s SettingsWithMCP
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	return s.MCPServers
}

func loadMCPJSON(path string) map[string]ServerConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg.MCPServers
}

// ServerConfigEnv returns the environment variables for a server config as a slice.
func ServerConfigEnv(cfg ServerConfig) []string {
	if len(cfg.Env) == 0 {
		return nil
	}
	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// ABOUTME: Environment variable expansion in config string fields
// ABOUTME: Replaces ${VAR} patterns with os.Getenv values; unset vars become empty

package config

import (
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// ResolveEnvVars expands ${VAR} patterns in string fields of Settings.
func ResolveEnvVars(s *Settings) {
	s.Model = expandEnv(s.Model)
	s.BaseURL = expandEnv(s.BaseURL)
	s.DefaultMode = expandEnv(s.DefaultMode)

	if s.StatusLine != nil {
		s.StatusLine.Command = expandEnv(s.StatusLine.Command)
	}

	for k, v := range s.Env {
		s.Env[k] = expandEnv(v)
	}

	for event, defs := range s.Hooks {
		for i := range defs {
			defs[i].Command = expandEnv(defs[i].Command)
			defs[i].Matcher = expandEnv(defs[i].Matcher)
		}
		s.Hooks[event] = defs
	}
}

// expandEnv replaces ${VAR} with os.Getenv(VAR). Unset vars become "".
func expandEnv(s string) string {
	if s == "" {
		return s
	}
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}

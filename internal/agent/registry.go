// ABOUTME: Thread-safe registry for agent definitions with builtin + custom loading
// ABOUTME: Provides Get/List/Register operations; loads from project and home directories

package agent

import (
	"sort"
	"sync"
)

// Registry holds agent definitions loaded from builtins and custom sources.
// Safe for concurrent access.
type Registry struct {
	mu   sync.RWMutex
	defs map[string]Definition
}

// NewRegistry creates a registry pre-loaded with builtin definitions
// and any custom definitions found in projectDir/homeDir agent directories.
// Custom definitions override builtins with the same name.
func NewRegistry(projectDir, homeDir string) *Registry {
	defs, _ := LoadDefinitions(projectDir, homeDir)
	if defs == nil {
		defs = make(map[string]Definition)
	}
	return &Registry{defs: defs}
}

// Get returns the definition with the given name.
// The second return value indicates whether the name was found.
func (r *Registry) Get(name string) (Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.defs[name]
	return def, ok
}

// List returns all registered definitions, sorted by name for deterministic output.
func (r *Registry) List() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Definition, 0, len(r.defs))
	for _, def := range r.defs {
		result = append(result, def)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// Register adds or overrides a definition in the registry.
func (r *Registry) Register(def Definition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defs[def.Name] = def
}

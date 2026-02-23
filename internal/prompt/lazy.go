// ABOUTME: Lazy skill loading: defers file I/O until first access via sync.Once
// ABOUTME: SkillRegistry enumerates skills by name; BuildSkillRefs controls eager vs lazy assembly

package prompt

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

// LazySkill wraps a skill definition with deferred content loading.
type LazySkill struct {
	Name        string
	Description string
	Path        string
	once        sync.Once
	content     string
	err         error
}

// Load reads the skill file on first call and caches the result.
// Subsequent calls return the cached content.
func (ls *LazySkill) Load() (string, error) {
	ls.once.Do(func() {
		skill, err := parseSkillFile(ls.Path)
		if err != nil {
			ls.err = fmt.Errorf("loading skill %s: %w", ls.Name, err)
			return
		}
		ls.content = skill.Content
	})
	return ls.content, ls.err
}

// SkillRegistry holds lazy skill references indexed by name.
type SkillRegistry struct {
	skills map[string]*LazySkill
	names  []string
}

// NewSkillRegistry scans skill directories and builds a registry.
// Skill files are enumerated but not read; content is loaded lazily.
func NewSkillRegistry(dirs []string) *SkillRegistry {
	byName := make(map[string]*LazySkill)

	// Load in reverse order so higher-priority dirs override
	for i := len(dirs) - 1; i >= 0; i-- {
		skills := scanSkillDir(dirs[i])
		for _, s := range skills {
			byName[s.Name] = s
		}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	return &SkillRegistry{skills: byName, names: names}
}

// Names returns sorted skill names without loading content.
func (r *SkillRegistry) Names() []string {
	return r.names
}

// Get returns a lazy skill by name, or nil if not found.
func (r *SkillRegistry) Get(name string) *LazySkill {
	return r.skills[name]
}

// scanSkillDir enumerates skill files in a directory without reading content.
func scanSkillDir(dir string) []*LazySkill {
	skills, err := loadSkillsFromDir(dir)
	if err != nil {
		return nil
	}

	lazy := make([]*LazySkill, 0, len(skills))
	for _, s := range skills {
		lazy = append(lazy, &LazySkill{
			Name:        s.Name,
			Description: s.Description,
			Path:        s.SourcePath,
		})
	}
	return lazy
}

// BuildSkillRefs converts registry skills into SkillRef slices for system prompt assembly.
// When preload is true (local model), all skill content is loaded eagerly.
// When preload is false (remote model), only name+description summaries are included.
func BuildSkillRefs(reg *SkillRegistry, preload bool) []SkillRef {
	refs := make([]SkillRef, 0, len(reg.names))

	for _, name := range reg.names {
		ls := reg.skills[name]
		if preload {
			content, err := ls.Load()
			if err != nil {
				continue
			}
			refs = append(refs, SkillRef{Name: name, Content: content})
		} else {
			// Lazy: include only description as a summary
			refs = append(refs, SkillRef{
				Name:    name,
				Content: fmt.Sprintf("[Available on demand] %s", strings.TrimSpace(ls.Description)),
			})
		}
	}

	return refs
}

// NewSkillRegistryForProject creates a SkillRegistry using standard skill resolution paths.
func NewSkillRegistryForProject(projectDir string) *SkillRegistry {
	return NewSkillRegistry(config.SkillsDirs(projectDir))
}

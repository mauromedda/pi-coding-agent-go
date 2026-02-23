// ABOUTME: Tests for lazy skill loading: content deferred until first access
// ABOUTME: Verifies sync.Once caching, SkillRegistry enumeration, and adaptive assembly

package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLazySkill_LoadDefersRead(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "test.md")
	os.WriteFile(skillPath, []byte("---\nname: test\ndescription: A test\n---\n\n# Test content"), 0o644)

	ls := &LazySkill{Path: skillPath, Name: "test", Description: "A test"}

	// Content should be empty before Load
	if ls.content != "" {
		t.Error("expected empty content before Load()")
	}

	content, err := ls.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !strings.Contains(content, "# Test content") {
		t.Errorf("Load() content = %q, want to contain '# Test content'", content)
	}
}

func TestLazySkill_LoadCaches(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "cached.md")
	os.WriteFile(skillPath, []byte("---\nname: cached\ndescription: Cached\n---\n\nOriginal"), 0o644)

	ls := &LazySkill{Path: skillPath, Name: "cached", Description: "Cached"}

	// First load
	content1, err := ls.Load()
	if err != nil {
		t.Fatalf("first Load() error: %v", err)
	}

	// Overwrite the file
	os.WriteFile(skillPath, []byte("---\nname: cached\ndescription: Cached\n---\n\nModified"), 0o644)

	// Second load should return cached content
	content2, err := ls.Load()
	if err != nil {
		t.Fatalf("second Load() error: %v", err)
	}

	if content1 != content2 {
		t.Errorf("expected cached content; got different:\n  first:  %q\n  second: %q", content1, content2)
	}
}

func TestLazySkill_LoadError(t *testing.T) {
	ls := &LazySkill{Path: "/nonexistent/skill.md", Name: "bad", Description: "Bad"}

	_, err := ls.Load()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSkillRegistry_Enumerate(t *testing.T) {
	dir := t.TempDir()

	// Create two skill files
	for _, name := range []string{"alpha", "beta"} {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0o755)
		content := "---\nname: " + name + "\ndescription: Skill " + name + "\n---\n\n# " + name + " content"
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644)
	}

	reg := NewSkillRegistry([]string{dir})

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("Names() returned %d skills, want 2", len(names))
	}

	// Names should be available without loading content
	for _, name := range names {
		ls := reg.Get(name)
		if ls == nil {
			t.Fatalf("Get(%q) returned nil", name)
		}
		if ls.content != "" {
			t.Errorf("skill %q content should be empty before Load()", name)
		}
	}
}

func TestSkillRegistry_LoadOnDemand(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "myskill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: myskill\ndescription: My skill\n---\n\n# Loaded content"), 0o644)

	reg := NewSkillRegistry([]string{dir})

	ls := reg.Get("myskill")
	if ls == nil {
		t.Fatal("Get returned nil")
	}

	content, err := ls.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !strings.Contains(content, "# Loaded content") {
		t.Errorf("content = %q, want '# Loaded content'", content)
	}
}

func TestBuildSkillRefs_Eager(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"s1", "s2"} {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0o755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
			[]byte("---\nname: "+name+"\ndescription: Desc "+name+"\n---\n\n# Content "+name), 0o644)
	}

	reg := NewSkillRegistry([]string{dir})
	refs := BuildSkillRefs(reg, true)

	if len(refs) != 2 {
		t.Fatalf("BuildSkillRefs(eager) returned %d refs, want 2", len(refs))
	}
	for _, ref := range refs {
		if !strings.Contains(ref.Content, "# Content") {
			t.Errorf("eager ref %q should have full content, got %q", ref.Name, ref.Content)
		}
	}
}

func TestBuildSkillRefs_Lazy(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"s1", "s2"} {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0o755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
			[]byte("---\nname: "+name+"\ndescription: Desc "+name+"\n---\n\n# Content "+name), 0o644)
	}

	reg := NewSkillRegistry([]string{dir})
	refs := BuildSkillRefs(reg, false)

	if len(refs) != 2 {
		t.Fatalf("BuildSkillRefs(lazy) returned %d refs, want 2", len(refs))
	}
	for _, ref := range refs {
		if strings.Contains(ref.Content, "# Content") {
			t.Errorf("lazy ref %q should have summary only, got %q", ref.Name, ref.Content)
		}
		if !strings.Contains(ref.Content, "Desc") {
			t.Errorf("lazy ref %q should contain description, got %q", ref.Name, ref.Content)
		}
	}
}

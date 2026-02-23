// ABOUTME: Profile loading and validation for personality configurations
// ABOUTME: Parses YAML profiles defining behavioral traits and check levels

package personality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile defines a personality configuration with behavioral traits and check levels.
type Profile struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Traits      TraitSet          `yaml:"traits"`
	Checks      map[string]string `yaml:"checks"` // check name -> level (minimal/standard/strict/paranoid)
}

// TraitSet defines behavioral traits for the personality.
type TraitSet struct {
	Verbosity     string `yaml:"verbosity"`      // "terse", "balanced", "verbose"
	RiskTolerance string `yaml:"risk_tolerance"`  // "cautious", "balanced", "aggressive"
	Explanation   string `yaml:"explanation"`     // "minimal", "standard", "detailed"
	AutoPlan      string `yaml:"auto_plan"`       // "never", "suggest", "auto"
}

var (
	validVerbosity     = map[string]bool{"terse": true, "balanced": true, "verbose": true}
	validRiskTolerance = map[string]bool{"cautious": true, "balanced": true, "aggressive": true}
	validExplanation   = map[string]bool{"minimal": true, "standard": true, "detailed": true}
	validAutoPlan      = map[string]bool{"never": true, "suggest": true, "auto": true}
	validCheckLevel    = map[string]bool{"minimal": true, "standard": true, "strict": true, "paranoid": true}
)

// DefaultTraitSet returns balanced defaults.
func DefaultTraitSet() TraitSet {
	return TraitSet{
		Verbosity:     "balanced",
		RiskTolerance: "balanced",
		Explanation:   "standard",
		AutoPlan:      "suggest",
	}
}

// LoadProfile reads a single YAML profile from a file.
func LoadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", path, err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validate profile %s: %w", path, err)
	}

	return &p, nil
}

// LoadProfiles reads all YAML profiles from a directory.
func LoadProfiles(dir string) (map[string]*Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read profiles directory %s: %w", dir, err)
	}

	profiles := make(map[string]*Profile)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		p, err := LoadProfile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		profiles[p.Name] = p
	}

	return profiles, nil
}

// Validate checks that a profile's fields are valid.
func (p *Profile) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if !validVerbosity[p.Traits.Verbosity] {
		return fmt.Errorf("invalid verbosity %q: must be terse, balanced, or verbose", p.Traits.Verbosity)
	}
	if !validRiskTolerance[p.Traits.RiskTolerance] {
		return fmt.Errorf("invalid risk_tolerance %q: must be cautious, balanced, or aggressive", p.Traits.RiskTolerance)
	}
	if !validExplanation[p.Traits.Explanation] {
		return fmt.Errorf("invalid explanation %q: must be minimal, standard, or detailed", p.Traits.Explanation)
	}
	if !validAutoPlan[p.Traits.AutoPlan] {
		return fmt.Errorf("invalid auto_plan %q: must be never, suggest, or auto", p.Traits.AutoPlan)
	}
	for checkName, level := range p.Checks {
		if !validCheckLevel[level] {
			return fmt.Errorf("invalid check level %q for %q: must be minimal, standard, strict, or paranoid", level, checkName)
		}
	}
	return nil
}

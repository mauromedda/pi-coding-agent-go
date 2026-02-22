// ABOUTME: Tests for package spec parsing
// ABOUTME: Validates NPM, Git, and Local source detection and name/tag extraction

package pkgmanager

import "testing"

func TestParseSpec_NPM(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw      string
		wantName string
		wantTag  string
	}{
		{"lodash", "lodash", ""},
		{"lodash@4.17.21", "lodash", "4.17.21"},
		{"@scope/name", "@scope/name", ""},
		{"@scope/name@1.2.3", "@scope/name", "1.2.3"},
		{"express@latest", "express", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			spec := ParseSpec(tt.raw)
			if spec.Source != SourceNPM {
				t.Errorf("Source = %v; want SourceNPM", spec.Source)
			}
			if spec.Name != tt.wantName {
				t.Errorf("Name = %q; want %q", spec.Name, tt.wantName)
			}
			if spec.Tag != tt.wantTag {
				t.Errorf("Tag = %q; want %q", spec.Tag, tt.wantTag)
			}
		})
	}
}

func TestParseSpec_Git(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw      string
		wantName string
		wantTag  string
	}{
		{"https://github.com/user/repo", "repo", ""},
		{"https://github.com/user/repo.git", "repo", ""},
		{"https://github.com/user/repo#main", "repo", "main"},
		{"git@github.com:user/myrepo.git", "myrepo", ""},
		{"git@github.com:user/myrepo.git#v1.0", "myrepo", "v1.0"},
		{"github.com/user/tool", "tool", ""},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			spec := ParseSpec(tt.raw)
			if spec.Source != SourceGit {
				t.Errorf("Source = %v; want SourceGit", spec.Source)
			}
			if spec.Name != tt.wantName {
				t.Errorf("Name = %q; want %q", spec.Name, tt.wantName)
			}
			if spec.Tag != tt.wantTag {
				t.Errorf("Tag = %q; want %q", spec.Tag, tt.wantTag)
			}
		})
	}
}

func TestParseSpec_Local(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw      string
		wantName string
		wantPath string
	}{
		{"./my-plugin", "my-plugin", "./my-plugin"},
		{"../shared/pkg", "pkg", "../shared/pkg"},
		{"/home/user/plugins/tool", "tool", "/home/user/plugins/tool"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			spec := ParseSpec(tt.raw)
			if spec.Source != SourceLocal {
				t.Errorf("Source = %v; want SourceLocal", spec.Source)
			}
			if spec.Name != tt.wantName {
				t.Errorf("Name = %q; want %q", spec.Name, tt.wantName)
			}
			if spec.Path != tt.wantPath {
				t.Errorf("Path = %q; want %q", spec.Path, tt.wantPath)
			}
		})
	}
}

func TestParseSpec_RawPreserved(t *testing.T) {
	t.Parallel()
	raw := "  lodash@4.17.21  "
	spec := ParseSpec(raw)
	if spec.Raw != "lodash@4.17.21" {
		t.Errorf("Raw = %q; want trimmed %q", spec.Raw, "lodash@4.17.21")
	}
}

func TestSource_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		source Source
		want   string
	}{
		{SourceNPM, "npm"},
		{SourceGit, "git"},
		{SourceLocal, "local"},
		{Source(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.source.String(); got != tt.want {
				t.Errorf("String() = %q; want %q", got, tt.want)
			}
		})
	}
}

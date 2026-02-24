// ABOUTME: Tests for built-in themes: default, dark, light, monochrome
// ABOUTME: Verifies each theme has a non-empty name and fully populated palette

package theme

import (
	"reflect"
	"testing"
)

func TestBuiltinThemes_AllExist(t *testing.T) {
	t.Parallel()
	names := []string{"default", "dark", "light", "monochrome"}
	for _, name := range names {
		th := Builtin(name)
		if th == nil {
			t.Errorf("Builtin(%q) returned nil", name)
			continue
		}
		if th.Name != name {
			t.Errorf("Builtin(%q).Name = %q", name, th.Name)
		}
	}
}

func TestBuiltinThemes_UnknownReturnsNil(t *testing.T) {
	t.Parallel()
	if th := Builtin("nonexistent"); th != nil {
		t.Errorf("Builtin(nonexistent) should return nil, got %v", th)
	}
}

func TestBuiltinThemes_AllPalettesPopulated(t *testing.T) {
	t.Parallel()
	for _, name := range BuiltinNames() {
		th := Builtin(name)
		if th == nil {
			t.Fatalf("Builtin(%q) returned nil", name)
		}
		v := reflect.ValueOf(th.Palette)
		for i := range v.NumField() {
			f := v.Field(i)
			if f.Type() != reflect.TypeFor[Color]() {
				continue
			}
			c := f.Interface().(Color)
			if c.Code() == "" {
				t.Errorf("Builtin(%q).Palette.%s has empty code", name, v.Type().Field(i).Name)
			}
		}
	}
}

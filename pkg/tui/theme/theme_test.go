// ABOUTME: Tests for theme types: Color.Apply, Palette completeness, bold/dim modifiers
// ABOUTME: Verifies ANSI wrapping, empty color passthrough, and all palette fields populated

package theme

import (
	"reflect"
	"strings"
	"testing"
)

func TestColor_Apply_WrapsText(t *testing.T) {
	t.Parallel()
	c := Color{code: "\x1b[32m"}
	got := c.Apply("hello")
	want := "\x1b[32mhello\x1b[0m"
	if got != want {
		t.Errorf("Apply() = %q; want %q", got, want)
	}
}

func TestColor_Apply_EmptyCode_PassesThrough(t *testing.T) {
	t.Parallel()
	c := Color{}
	got := c.Apply("hello")
	if got != "hello" {
		t.Errorf("Apply() = %q; want %q", got, "hello")
	}
}

func TestColor_Apply_EmptyText(t *testing.T) {
	t.Parallel()
	c := Color{code: "\x1b[31m"}
	got := c.Apply("")
	want := "\x1b[31m\x1b[0m"
	if got != want {
		t.Errorf("Apply() = %q; want %q", got, want)
	}
}

func TestColor_Code(t *testing.T) {
	t.Parallel()
	c := Color{code: "\x1b[36m"}
	if c.Code() != "\x1b[36m" {
		t.Errorf("Code() = %q; want %q", c.Code(), "\x1b[36m")
	}
}

func TestColor_Bold(t *testing.T) {
	t.Parallel()
	c := NewColor("\x1b[32m")
	bold := c.Bold()
	got := bold.Apply("ok")
	if !strings.Contains(got, "\x1b[1m") {
		t.Errorf("Bold().Apply() should contain bold code, got %q", got)
	}
	if !strings.Contains(got, "\x1b[32m") {
		t.Errorf("Bold().Apply() should contain original color, got %q", got)
	}
}

func TestColor_Dim(t *testing.T) {
	t.Parallel()
	c := NewColor("\x1b[32m")
	dim := c.Dim()
	got := dim.Apply("ok")
	if !strings.Contains(got, "\x1b[2m") {
		t.Errorf("Dim().Apply() should contain dim code, got %q", got)
	}
}

func TestNewColor(t *testing.T) {
	t.Parallel()
	c := NewColor("\x1b[33m")
	if c.Code() != "\x1b[33m" {
		t.Errorf("NewColor() code = %q; want %q", c.Code(), "\x1b[33m")
	}
}

func TestDefaultPalette_AllFieldsSet(t *testing.T) {
	t.Parallel()
	p := DefaultPalette()
	v := reflect.ValueOf(p)
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Type() != reflect.TypeFor[Color]() {
			continue
		}
		c := f.Interface().(Color)
		if c.Code() == "" {
			t.Errorf("DefaultPalette().%s has empty code", v.Type().Field(i).Name)
		}
	}
}

// ABOUTME: JSON theme file loading with validation and default fallback
// ABOUTME: Unset palette fields inherit from DefaultPalette to ensure completeness

package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// jsonPalette is the JSON-friendly representation of a Palette.
// Fields use snake_case to match the JSON file format.
type jsonPalette struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Muted     string `json:"muted"`
	Accent    string `json:"accent"`

	Success string `json:"success"`
	Warning string `json:"warning"`
	Error   string `json:"error"`
	Info    string `json:"info"`

	Border    string `json:"border"`
	Selection string `json:"selection"`
	Prompt    string `json:"prompt"`

	ToolRead  string `json:"tool_read"`
	ToolBash  string `json:"tool_bash"`
	ToolWrite string `json:"tool_write"`
	ToolOther string `json:"tool_other"`

	FooterPath   string `json:"footer_path"`
	FooterBranch string `json:"footer_branch"`
	FooterModel  string `json:"footer_model"`
	FooterCost   string `json:"footer_cost"`
	FooterPerm   string `json:"footer_perm"`

	UserBg string `json:"user_bg"`

	Bold      string `json:"bold"`
	Dim       string `json:"dim"`
	Italic    string `json:"italic"`
	Underline string `json:"underline"`
}

type jsonTheme struct {
	Name    string      `json:"name"`
	Palette jsonPalette `json:"palette"`
}

// LoadFile reads a JSON theme file and returns a Theme.
// Missing palette fields fall back to DefaultPalette values.
func LoadFile(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading theme file: %w", err)
	}

	var jt jsonTheme
	if err := json.Unmarshal(data, &jt); err != nil {
		return nil, fmt.Errorf("parsing theme file: %w", err)
	}

	base := DefaultPalette()
	p := convertPalette(jt.Palette, base)

	return &Theme{
		Name:    jt.Name,
		Palette: p,
	}, nil
}

// convertPalette maps jsonPalette fields onto a Palette, using base for empty fields.
func convertPalette(jp jsonPalette, base Palette) Palette {
	p := base // start with defaults

	// Map JSON fields to Palette fields by reflection on matching names.
	// This avoids a long manual mapping.
	jpv := reflect.ValueOf(jp)
	pv := reflect.ValueOf(&p).Elem()
	jpt := jpv.Type()

	for i := range jpt.NumField() {
		jsonVal := jpv.Field(i).String()
		if jsonVal == "" {
			continue
		}
		fieldName := jpt.Field(i).Name
		pf := pv.FieldByName(fieldName)
		if pf.IsValid() && pf.CanSet() {
			pf.Set(reflect.ValueOf(NewColor(jsonVal)))
		}
	}

	return p
}

// ABOUTME: Template variable resolution for prompt fragments using text/template
// ABOUTME: Replaces {{.VAR}} placeholders with runtime values

package prompts

import (
	"bytes"
	"fmt"
	"text/template"
)

// RenderVariables replaces template variables in the content string.
// Variables are passed as a map; undefined variables produce empty strings.
func RenderVariables(content string, vars map[string]string) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

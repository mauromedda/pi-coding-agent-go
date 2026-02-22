// ABOUTME: HTML exporter for chat sessions using Go html/template
// ABOUTME: Renders messages as styled HTML with role indicators and collapsible tool calls

package export

import (
	"html/template"
	"io"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// ExportHTML renders a slice of messages as a styled HTML document to w.
// The output uses a dark theme with role-specific color indicators:
// User (blue), Assistant (green), Tool (gray).
// Tool results are rendered inside collapsible <details> elements.
func ExportHTML(messages []ai.Message, w io.Writer) error {
	return htmlTmpl.Execute(w, messages)
}

// roleClass maps a message role to a CSS class name.
func roleClass(role ai.Role) string {
	switch role {
	case ai.RoleUser:
		return "user"
	case ai.RoleAssistant:
		return "assistant"
	default:
		return "system"
	}
}

// inputString returns the tool input as a formatted string.
func inputString(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	return string(input)
}

// isToolResult checks if content is a tool result type.
func isToolResult(ct ai.ContentType) bool {
	return ct == ai.ContentToolResult
}

// isToolUse checks if content is a tool use type.
func isToolUse(ct ai.ContentType) bool {
	return ct == ai.ContentToolUse
}

// isText checks if content is a text type.
func isText(ct ai.ContentType) bool {
	return ct == ai.ContentText
}

// escapeNewlines converts newlines to <br> for HTML rendering.
func escapeNewlines(s string) template.HTML {
	escaped := template.HTMLEscapeString(s)
	return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>\n"))
}

var funcMap = template.FuncMap{
	"roleClass":   roleClass,
	"inputString": inputString,
	"isToolResult":   isToolResult,
	"isToolUse":      isToolUse,
	"isText":         isText,
	"escapeNewlines": escapeNewlines,
}

var htmlTmpl = template.Must(template.New("session").Funcs(funcMap).Parse(htmlTemplate))

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Chat Session Export</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #1e1e2e;
    color: #cdd6f4;
    font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
    font-size: 14px;
    line-height: 1.6;
    padding: 24px;
    max-width: 900px;
    margin: 0 auto;
  }
  .message {
    margin-bottom: 16px;
    padding: 12px 16px;
    border-radius: 8px;
    border-left: 4px solid;
  }
  .message.user {
    border-left-color: #89b4fa;
    background: #1e1e2e;
  }
  .message.assistant {
    border-left-color: #a6e3a1;
    background: #1e1e2e;
  }
  .message.system {
    border-left-color: #9399b2;
    background: #1e1e2e;
  }
  .role-badge {
    display: inline-block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 2px 8px;
    border-radius: 4px;
    margin-bottom: 8px;
  }
  .user .role-badge { background: #89b4fa22; color: #89b4fa; }
  .assistant .role-badge { background: #a6e3a122; color: #a6e3a1; }
  .system .role-badge { background: #9399b222; color: #9399b2; }
  .content-block { margin-top: 8px; }
  .tool-use {
    background: #313244;
    padding: 8px 12px;
    border-radius: 6px;
    margin-top: 8px;
  }
  .tool-name {
    color: #cba6f7;
    font-weight: 600;
  }
  .tool-input {
    color: #a6adc8;
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-all;
    margin-top: 4px;
  }
  details {
    background: #313244;
    padding: 8px 12px;
    border-radius: 6px;
    margin-top: 8px;
  }
  details summary {
    cursor: pointer;
    color: #9399b2;
    font-size: 12px;
    font-weight: 600;
  }
  details .result-content {
    margin-top: 8px;
    color: #a6adc8;
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .error-result summary { color: #f38ba8; }
  .error-result .result-content { color: #f38ba8; }
</style>
</head>
<body>
{{- range . }}
<div class="message {{ roleClass .Role }}">
  <span class="role-badge">{{ .Role }}</span>
  {{- range .Content }}
    {{- if isText .Type }}
  <div class="content-block">{{ escapeNewlines .Text }}</div>
    {{- else if isToolUse .Type }}
  <div class="tool-use">
    <span class="tool-name">{{ .Name }}</span>
    <div class="tool-input">{{ inputString .Input }}</div>
  </div>
    {{- else if isToolResult .Type }}
  <details{{ if .IsError }} class="error-result"{{ end }}>
    <summary>Tool Result ({{ .ID }}){{ if .IsError }} — error{{ end }}</summary>
    <div class="result-content">{{ escapeNewlines .ResultText }}</div>
  </details>
    {{- end }}
  {{- end }}
</div>
{{- end }}
</body>
</html>
`

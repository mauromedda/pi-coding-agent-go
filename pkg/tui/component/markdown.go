// ABOUTME: Terminal markdown renderer that converts markdown to ANSI-styled text
// ABOUTME: Supports bold, italic, code, headers, lists, links, and fenced code blocks

package component

import (
	"regexp"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

const (
	ansiBold      = "\x1b[1m"
	ansiItalic    = "\x1b[3m"
	ansiDim       = "\x1b[2m"
	ansiUnderline = "\x1b[4m"
	ansiCyan      = "\x1b[36m"
	ansiReset     = "\x1b[0m"
)

var (
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic     = regexp.MustCompile(`\*(.+?)\*`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
)

// Markdown renders markdown-formatted text with ANSI styling.
type Markdown struct {
	content string
	dirty   bool
	cached  []string
}

// NewMarkdown creates a Markdown component with the given content.
func NewMarkdown(content string) *Markdown {
	return &Markdown{content: content, dirty: true}
}

// SetContent updates the markdown content.
func (md *Markdown) SetContent(content string) {
	md.content = content
	md.dirty = true
}

// Invalidate marks the component for re-render.
func (md *Markdown) Invalidate() {
	md.dirty = true
}

// Render writes the styled markdown lines into the buffer.
func (md *Markdown) Render(out *tui.RenderBuffer, _ int) {
	if md.dirty {
		md.cached = md.renderLines()
		md.dirty = false
	}
	out.WriteLines(md.cached)
}

func (md *Markdown) renderLines() []string {
	if md.content == "" {
		return []string{""}
	}

	raw := strings.Split(md.content, "\n")
	var result []string
	inCodeBlock := false
	var codeLang string

	for i := 0; i < len(raw); i++ {
		line := raw[i]

		// Fenced code block toggle
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeLang = strings.TrimPrefix(line, "```")
				codeLang = strings.TrimSpace(codeLang)
				if codeLang != "" {
					result = append(result, ansiDim+"    ["+codeLang+"]"+ansiReset)
				}
				continue
			}
			inCodeBlock = false
			continue
		}

		if inCodeBlock {
			result = append(result, "    "+ansiDim+line+ansiReset)
			continue
		}

		// Headers
		if h, level := parseHeader(line); level > 0 {
			styled := ansiBold + ansiCyan + h + ansiReset
			result = append(result, styled)
			continue
		}

		// Unordered list (- or *)
		if trimmed := strings.TrimSpace(line); (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			content := trimmed[2:]
			styled := "  \u2022 " + md.renderInline(content)
			result = append(result, styled)
			continue
		}

		// Ordered list (1. 2. etc)
		if isOrderedListItem(line) {
			idx := strings.Index(line, ". ")
			if idx > 0 {
				num := strings.TrimSpace(line[:idx])
				content := line[idx+2:]
				styled := "  " + num + ". " + md.renderInline(content)
				result = append(result, styled)
				continue
			}
		}

		// Empty line (paragraph break)
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
			continue
		}

		// Regular text with inline formatting
		result = append(result, md.renderInline(line))
	}

	return result
}

func (md *Markdown) renderInline(s string) string {
	// Process bold first (** before *)
	s = reBold.ReplaceAllString(s, ansiBold+"$1"+ansiReset)

	// Process italic (* but not **)
	s = reItalic.ReplaceAllStringFunc(s, func(match string) string {
		// Skip if it looks like already-processed bold remnants
		if strings.Contains(match, "**") {
			return match
		}
		inner := match[1 : len(match)-1]
		return ansiItalic + inner + ansiReset
	})

	// Process inline code
	s = reInlineCode.ReplaceAllString(s, ansiDim+ansiCyan+"$1"+ansiReset)

	// Process links: [text](url) -> underlined text
	s = reLink.ReplaceAllString(s, ansiUnderline+"$1"+ansiReset)

	return s
}

func parseHeader(line string) (string, int) {
	if strings.HasPrefix(line, "### ") {
		return line[4:], 3
	}
	if strings.HasPrefix(line, "## ") {
		return line[3:], 2
	}
	if strings.HasPrefix(line, "# ") {
		return line[2:], 1
	}
	return "", 0
}

func isOrderedListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	for i, c := range trimmed {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 && i+1 < len(trimmed) && trimmed[i+1] == ' ' {
			return true
		}
		break
	}
	return false
}

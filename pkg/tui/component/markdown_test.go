// ABOUTME: Tests for the terminal markdown renderer component
// ABOUTME: Covers bold, italic, code, headers, lists, links, code blocks

package component

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func renderMarkdown(md string, w int) []string {
	comp := NewMarkdown(md)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	comp.Render(buf, w)
	return append([]string{}, buf.Lines...)
}

func TestMarkdown_NewMarkdown(t *testing.T) {
	t.Parallel()

	md := NewMarkdown("Hello")
	if md == nil {
		t.Fatal("NewMarkdown returned nil")
	}
}

func TestMarkdown_PlainText(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("Hello world", 80)
	if len(lines) == 0 {
		t.Fatal("expected at least 1 line")
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "Hello world") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Hello world' in output, got %v", lines)
	}
}

func TestMarkdown_Bold(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("This is **bold** text", 80)
	joined := strings.Join(lines, "\n")

	// Should contain ANSI bold (\x1b[1m) around "bold"
	if !strings.Contains(joined, "\x1b[1m") {
		t.Error("expected ANSI bold sequence in output")
	}
	if !strings.Contains(joined, "bold") {
		t.Error("expected 'bold' text in output")
	}
}

func TestMarkdown_Italic(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("This is *italic* text", 80)
	joined := strings.Join(lines, "\n")

	// Should contain ANSI italic (\x1b[3m)
	if !strings.Contains(joined, "\x1b[3m") {
		t.Error("expected ANSI italic sequence in output")
	}
	if !strings.Contains(joined, "italic") {
		t.Error("expected 'italic' text in output")
	}
}

func TestMarkdown_InlineCode(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("Use `fmt.Println` here", 80)
	joined := strings.Join(lines, "\n")

	// Should contain ANSI dim or color for code
	if !strings.Contains(joined, "\x1b[") {
		t.Error("expected ANSI formatting for inline code")
	}
	if !strings.Contains(joined, "fmt.Println") {
		t.Error("expected 'fmt.Println' in output")
	}
}

func TestMarkdown_CodeBlock(t *testing.T) {
	t.Parallel()

	md := "```go\nfunc main() {\n}\n```"
	lines := renderMarkdown(md, 80)

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines for code block, got %d", len(lines))
	}

	// Code block lines should be indented
	foundIndented := false
	for _, l := range lines {
		stripped := width.StripANSI(l)
		if strings.Contains(stripped, "func main()") {
			if strings.HasPrefix(stripped, "  ") || strings.HasPrefix(stripped, "    ") {
				foundIndented = true
			}
		}
	}
	if !foundIndented {
		t.Error("expected code block content to be indented")
	}
}

func TestMarkdown_Header1(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("# Title", 80)
	if len(lines) == 0 {
		t.Fatal("expected at least 1 line")
	}
	joined := strings.Join(lines, "\n")

	// Headers should be bold
	if !strings.Contains(joined, "\x1b[1m") {
		t.Error("expected bold formatting for h1")
	}
	if !strings.Contains(joined, "Title") {
		t.Error("expected 'Title' in output")
	}
}

func TestMarkdown_Header2(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("## Subtitle", 80)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "\x1b[1m") {
		t.Error("expected bold formatting for h2")
	}
	if !strings.Contains(joined, "Subtitle") {
		t.Error("expected 'Subtitle' in output")
	}
}

func TestMarkdown_Header3(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("### Section", 80)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "\x1b[1m") {
		t.Error("expected bold formatting for h3")
	}
	if !strings.Contains(joined, "Section") {
		t.Error("expected 'Section' in output")
	}
}

func TestMarkdown_UnorderedList(t *testing.T) {
	t.Parallel()

	md := "- Item one\n- Item two\n- Item three"
	lines := renderMarkdown(md, 80)

	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for _, l := range lines {
		stripped := width.StripANSI(l)
		if !strings.Contains(stripped, "\u2022") && !strings.Contains(stripped, "-") && !strings.Contains(stripped, "*") {
			// Accept bullet point or dash
			if !strings.HasPrefix(strings.TrimSpace(stripped), "Item") {
				continue
			}
		}
	}
}

func TestMarkdown_OrderedList(t *testing.T) {
	t.Parallel()

	md := "1. First\n2. Second\n3. Third"
	lines := renderMarkdown(md, 80)

	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// Verify items are present
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "First") || !strings.Contains(joined, "Second") || !strings.Contains(joined, "Third") {
		t.Error("expected all list items in output")
	}
}

func TestMarkdown_StarList(t *testing.T) {
	t.Parallel()

	md := "* Alpha\n* Beta"
	lines := renderMarkdown(md, 80)

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Alpha") || !strings.Contains(joined, "Beta") {
		t.Error("expected list items in output")
	}
}

func TestMarkdown_Link(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("Click [here](https://example.com)", 80)
	joined := strings.Join(lines, "\n")

	// Link text should be underlined (\x1b[4m)
	if !strings.Contains(joined, "\x1b[4m") {
		t.Error("expected underline formatting for link text")
	}
	if !strings.Contains(joined, "here") {
		t.Error("expected 'here' link text in output")
	}
}

func TestMarkdown_MultiParagraph(t *testing.T) {
	t.Parallel()

	md := "First paragraph\n\nSecond paragraph"
	lines := renderMarkdown(md, 80)

	// Should have blank line between paragraphs
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %v", len(lines), lines)
	}
}

func TestMarkdown_EmptyInput(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("", 80)
	if len(lines) > 1 {
		t.Errorf("expected 0 or 1 lines for empty input, got %d", len(lines))
	}
}

func TestMarkdown_Invalidate(t *testing.T) {
	t.Parallel()

	md := NewMarkdown("test")
	md.Invalidate()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	md.Render(buf, 40)
	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line after invalidate")
	}
}

func TestMarkdown_SetContent(t *testing.T) {
	t.Parallel()

	md := NewMarkdown("old")
	md.SetContent("new content")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	md.Render(buf, 40)

	joined := strings.Join(buf.Lines, "\n")
	if !strings.Contains(joined, "new content") {
		t.Errorf("expected 'new content' in output, got %q", joined)
	}
}

func TestMarkdown_CodeBlockPreservesContent(t *testing.T) {
	t.Parallel()

	md := "```\nline1\nline2\n```"
	lines := renderMarkdown(md, 80)

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "line1") || !strings.Contains(joined, "line2") {
		t.Error("expected code block lines preserved")
	}
}

func TestMarkdown_BoldAndItalicMixed(t *testing.T) {
	t.Parallel()

	lines := renderMarkdown("**bold** and *italic*", 80)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "bold") || !strings.Contains(joined, "italic") {
		t.Error("expected both bold and italic text in output")
	}
}

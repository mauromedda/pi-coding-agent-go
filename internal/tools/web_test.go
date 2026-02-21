// ABOUTME: Tests for WebFetch HTML-to-markdown, WebSearch result formatting, and cache
// ABOUTME: Uses in-process test HTTP server to avoid external dependencies

package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

func TestHtmlToMarkdown_Basic(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>Hello <strong>world</strong></p></body></html>`
	md := htmlToMarkdown(html)

	if !strings.Contains(md, "# Title") {
		t.Errorf("expected heading, got %q", md)
	}
	if !strings.Contains(md, "**world**") {
		t.Errorf("expected bold, got %q", md)
	}
}

func TestHtmlToMarkdown_SkipsScriptAndStyle(t *testing.T) {
	html := `<html><body><script>alert('x')</script><style>.x{}</style><p>Content</p></body></html>`
	md := htmlToMarkdown(html)

	if strings.Contains(md, "alert") {
		t.Error("script content should be stripped")
	}
	if strings.Contains(md, ".x{") {
		t.Error("style content should be stripped")
	}
	if !strings.Contains(md, "Content") {
		t.Error("body content should be preserved")
	}
}

func TestHtmlToMarkdown_Links(t *testing.T) {
	html := `<html><body><a href="https://example.com">Click here</a></body></html>`
	md := htmlToMarkdown(html)

	if !strings.Contains(md, "[Click here](https://example.com)") {
		t.Errorf("expected markdown link, got %q", md)
	}
}

func TestHtmlToMarkdown_Lists(t *testing.T) {
	html := `<html><body><ul><li>Item 1</li><li>Item 2</li></ul></body></html>`
	md := htmlToMarkdown(html)

	if !strings.Contains(md, "- Item 1") {
		t.Errorf("expected list item, got %q", md)
	}
}

func TestHtmlToMarkdown_Code(t *testing.T) {
	html := `<html><body><p>Use <code>go test</code> to run tests.</p></body></html>`
	md := htmlToMarkdown(html)

	if !strings.Contains(md, "`go test`") {
		t.Errorf("expected inline code, got %q", md)
	}
}

func TestHtmlToMarkdown_PreBlock(t *testing.T) {
	html := `<html><body><pre>func main() {
	fmt.Println("hello")
}</pre></body></html>`
	md := htmlToMarkdown(html)

	if !strings.Contains(md, "```") {
		t.Errorf("expected code block, got %q", md)
	}
}

func TestWebFetch_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Test Page</h1><p>Hello from test server</p></body></html>`))
	}))
	defer srv.Close()

	tool := NewWebFetchTool()
	result, err := tool.Execute(context.Background(), "test", map[string]any{"url": srv.URL}, nil)
	if err != nil {
		t.Fatalf("WebFetch: %v", err)
	}
	if result.IsError {
		t.Fatalf("WebFetch error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Test Page") {
		t.Errorf("expected page content, got %q", result.Content)
	}
}

func TestWebFetch_Cache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`<html><body><p>Cached content</p></body></html>`))
	}))
	defer srv.Close()

	tool := NewWebFetchTool()
	// First call
	_, _ = tool.Execute(context.Background(), "t1", map[string]any{"url": srv.URL}, nil)
	// Second call (should hit cache)
	_, _ = tool.Execute(context.Background(), "t2", map[string]any{"url": srv.URL}, nil)

	if calls != 1 {
		t.Errorf("expected 1 server call (cached), got %d", calls)
	}
}

func TestWebFetch_MissingURL(t *testing.T) {
	tool := NewWebFetchTool()
	result, _ := tool.Execute(context.Background(), "t", map[string]any{}, nil)
	if !result.IsError {
		t.Error("expected error for missing URL")
	}
}

func TestWebSearch_FormatResults(t *testing.T) {
	result := braveSearchResult{}
	result.Web.Results = []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}{
		{Title: "Go", URL: "https://go.dev", Description: "The Go programming language"},
		{Title: "Go Wiki", URL: "https://go.dev/wiki", Description: "Go wiki"},
	}

	formatted := formatSearchResults("golang", result)
	if !strings.Contains(formatted, "golang") {
		t.Error("expected query in output")
	}
	if !strings.Contains(formatted, "[Go](https://go.dev)") {
		t.Error("expected markdown link in output")
	}
	if !strings.Contains(formatted, "2.") {
		t.Error("expected numbered results")
	}
}

func TestWebSearch_MissingAPIKey(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")

	tool := NewWebSearchTool()
	result, _ := tool.Execute(context.Background(), "t", map[string]any{"query": "test"}, nil)
	if !result.IsError {
		t.Error("expected error for missing API key")
	}
}

func TestWebCache_Basic(t *testing.T) {
	c := newWebCache()

	c.Set("key1", "value1")
	v, ok := c.Get("key1")
	if !ok || v != "value1" {
		t.Errorf("expected value1, got %q (ok=%v)", v, ok)
	}
}

func TestWebCache_Miss(t *testing.T) {
	c := newWebCache()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestWebCache_Eviction(t *testing.T) {
	c := newWebCache()

	// Fill to capacity
	for i := 0; i < cacheMaxEntries+5; i++ {
		c.Set(string(rune('a'+i%26))+string(rune('0'+i/26)), "v")
	}

	c.mu.Lock()
	count := len(c.entries)
	c.mu.Unlock()

	if count > cacheMaxEntries {
		t.Errorf("cache should not exceed %d entries, has %d", cacheMaxEntries, count)
	}
}

func TestExtractText(t *testing.T) {
	// Dummy test for extractText helper
	noop := func(agent.ToolUpdate) {}
	_ = noop
	_ = time.Second
}

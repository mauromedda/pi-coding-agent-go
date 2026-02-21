// ABOUTME: WebFetch tool: fetches URLs, extracts readable content, converts to markdown
// ABOUTME: Uses golang.org/x/net/html for parsing; 15-minute LRU cache

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"golang.org/x/net/html"
)

var fetchCache = newWebCache()

// NewWebFetchTool creates a tool that fetches and extracts content from URLs.
func NewWebFetchTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "webfetch",
		Label:       "Fetch Web Page",
		Description: "Fetch a URL and extract its readable content as markdown.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["url"],
			"properties": {
				"url":    {"type": "string", "description": "URL to fetch"},
				"prompt": {"type": "string", "description": "What to extract from the page (optional)"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeWebFetch,
	}
}

func executeWebFetch(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	url, err := requireStringParam(params, "url")
	if err != nil {
		return errResult(err), nil
	}

	// Upgrade HTTP to HTTPS (skip localhost for testing)
	if strings.HasPrefix(url, "http://") && !strings.Contains(url, "localhost") && !strings.Contains(url, "127.0.0.1") {
		url = "https://" + url[7:]
	}

	// Check cache
	if cached, ok := fetchCache.Get(url); ok {
		return agent.ToolResult{Content: cached}, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errResult(fmt.Errorf("creating request: %w", err)), nil
	}
	req.Header.Set("User-Agent", "pi-go/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return errResult(fmt.Errorf("fetching %s: %w", url, err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errResult(fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return errResult(fmt.Errorf("reading response: %w", err)), nil
	}

	content := htmlToMarkdown(string(body))
	content = truncateOutput(content, maxReadOutput)

	fetchCache.Set(url, content)
	return agent.ToolResult{Content: content}, nil
}

// htmlToMarkdown converts HTML to readable markdown.
func htmlToMarkdown(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return raw // Return raw if parse fails
	}

	var b strings.Builder
	extractReadable(doc, &b, false)
	return strings.TrimSpace(b.String())
}

// extractReadable walks the HTML tree and extracts text content.
func extractReadable(n *html.Node, b *strings.Builder, inPre bool) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "nav", "footer", "header", "iframe", "noscript":
			return // Skip non-content elements
		case "h1":
			b.WriteString("\n# ")
		case "h2":
			b.WriteString("\n## ")
		case "h3":
			b.WriteString("\n### ")
		case "h4", "h5", "h6":
			b.WriteString("\n#### ")
		case "p", "div", "section", "article":
			b.WriteString("\n\n")
		case "br":
			b.WriteString("\n")
		case "li":
			b.WriteString("\n- ")
		case "pre":
			b.WriteString("\n```\n")
			inPre = true
		case "code":
			if !inPre {
				b.WriteString("`")
			}
		case "a":
			// Extract link text and href
			href := getAttr(n, "href")
			if href != "" {
				text := extractText(n)
				if text != "" {
					b.WriteString(fmt.Sprintf("[%s](%s)", text, href))
					return // Don't recurse into children
				}
			}
		case "strong", "b":
			b.WriteString("**")
		case "em", "i":
			b.WriteString("*")
		}
	}

	if n.Type == html.TextNode {
		text := n.Data
		if !inPre {
			text = strings.Join(strings.Fields(text), " ")
		}
		if text != "" && text != " " {
			b.WriteString(text)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractReadable(c, b, inPre)
	}

	// Closing tags
	if n.Type == html.ElementNode {
		switch n.Data {
		case "pre":
			b.WriteString("\n```\n")
		case "code":
			if !inPre {
				b.WriteString("`")
			}
		case "strong", "b":
			b.WriteString("**")
		case "em", "i":
			b.WriteString("*")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			b.WriteString("\n")
		}
	}
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

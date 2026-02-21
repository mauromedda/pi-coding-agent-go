// ABOUTME: WebSearch tool: searches via Brave Search API and returns markdown results
// ABOUTME: Reads API key from config env or BRAVE_SEARCH_API_KEY environment variable

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const braveSearchURL = "https://api.search.brave.com/res/v1/web/search"

// NewWebSearchTool creates a tool that searches the web via Brave Search.
func NewWebSearchTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "websearch",
		Label:       "Web Search",
		Description: "Search the web using Brave Search API. Returns top results as markdown.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {"type": "string", "description": "Search query"},
				"count": {"type": "integer", "description": "Number of results (default 10, max 20)"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeWebSearch,
	}
}

func executeWebSearch(ctx context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	query, err := requireStringParam(params, "query")
	if err != nil {
		return errResult(err), nil
	}

	count := intParam(params, "count", 10)
	if count > 20 {
		count = 20
	}

	apiKey := os.Getenv("BRAVE_SEARCH_API_KEY")
	if apiKey == "" {
		return errResult(fmt.Errorf("BRAVE_SEARCH_API_KEY not set")), nil
	}

	u, _ := url.Parse(braveSearchURL)
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", count))
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return errResult(fmt.Errorf("creating search request: %w", err)), nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return errResult(fmt.Errorf("search request failed: %w", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return errResult(fmt.Errorf("search API returned %d: %s", resp.StatusCode, string(body))), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return errResult(fmt.Errorf("reading search response: %w", err)), nil
	}

	var result braveSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return errResult(fmt.Errorf("parsing search response: %w", err)), nil
	}

	return agent.ToolResult{Content: formatSearchResults(query, result)}, nil
}

type braveSearchResult struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

func formatSearchResults(query string, result braveSearchResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Search results for: %s\n\n", query))

	for i, r := range result.Web.Results {
		b.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, r.Title, r.URL))
		if r.Description != "" {
			b.WriteString(fmt.Sprintf("   %s\n", r.Description))
		}
		b.WriteString("\n")
	}

	if len(result.Web.Results) == 0 {
		b.WriteString("No results found.\n")
	}

	return b.String()
}

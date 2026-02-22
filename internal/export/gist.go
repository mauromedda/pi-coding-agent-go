// ABOUTME: GitHub Gist creation for sharing sessions
// ABOUTME: Posts session content as a gist using GitHub API with GITHUB_TOKEN

package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const githubGistAPI = "https://api.github.com/gists"

// gistFile represents a single file in a gist request.
type gistFile struct {
	Content string `json:"content"`
}

// gistRequest is the JSON payload sent to the GitHub Gist API.
type gistRequest struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]gistFile `json:"files"`
}

// gistResponse holds the relevant fields from the GitHub API response.
type gistResponse struct {
	HTMLURL string `json:"html_url"`
}

// CreateGist creates a GitHub Gist with the given content and returns its URL.
// It reads the GitHub personal access token from the GITHUB_TOKEN environment variable.
// Returns an error if GITHUB_TOKEN is not set or the API call fails.
func CreateGist(content string, description string, public bool) (string, error) {
	return createGistWithURL(githubGistAPI, content, description, public)
}

// createGistWithURL is the internal implementation that accepts a configurable API URL,
// enabling testability via httptest.NewServer.
func createGistWithURL(apiURL string, content string, description string, public bool) (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	reqBody := gistRequest{
		Description: description,
		Public:      public,
		Files: map[string]gistFile{
			"session.md": {Content: content},
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshalling gist request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("posting gist: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var gistResp gistResponse
	if err := json.Unmarshal(body, &gistResp); err != nil {
		return "", fmt.Errorf("decoding gist response: %w", err)
	}

	return gistResp.HTMLURL, nil
}

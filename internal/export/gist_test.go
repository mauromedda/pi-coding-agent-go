// ABOUTME: Tests for GitHub Gist creation
// ABOUTME: Uses httptest.NewServer to mock GitHub API

package export

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateGist_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token-123" {
			t.Errorf("expected Bearer token, got %q", auth)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}

		var req gistRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshalling body: %v", err)
		}

		if req.Description != "test session" {
			t.Errorf("expected description 'test session', got %q", req.Description)
		}
		if !req.Public {
			t.Error("expected public gist")
		}

		file, ok := req.Files["session.md"]
		if !ok {
			t.Fatal("expected session.md file in gist")
		}
		if file.Content != "# Session\nHello" {
			t.Errorf("unexpected content: %q", file.Content)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(gistResponse{
			HTMLURL: "https://gist.github.com/abc123",
		})
	}))
	defer srv.Close()

	t.Setenv("GITHUB_TOKEN", "test-token-123")

	url, err := createGistWithURL(srv.URL, "# Session\nHello", "test session", true)
	if err != nil {
		t.Fatalf("CreateGist: %v", err)
	}

	if url != "https://gist.github.com/abc123" {
		t.Errorf("expected gist URL, got %q", url)
	}
}

func TestCreateGist_PrivateGist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req gistRequest
		json.Unmarshal(body, &req)

		if req.Public {
			t.Error("expected private gist (public=false)")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(gistResponse{
			HTMLURL: "https://gist.github.com/private456",
		})
	}))
	defer srv.Close()

	t.Setenv("GITHUB_TOKEN", "token")

	url, err := createGistWithURL(srv.URL, "content", "desc", false)
	if err != nil {
		t.Fatalf("CreateGist: %v", err)
	}

	if url != "https://gist.github.com/private456" {
		t.Errorf("expected private gist URL, got %q", url)
	}
}

func TestCreateGist_MissingToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")

	_, err := CreateGist("content", "desc", false)
	if err == nil {
		t.Fatal("expected error for missing GITHUB_TOKEN")
	}

	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("expected error mentioning GITHUB_TOKEN, got %q", err.Error())
	}
}

func TestCreateGist_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	t.Setenv("GITHUB_TOKEN", "bad-token")

	_, err := createGistWithURL(srv.URL, "content", "desc", false)
	if err == nil {
		t.Fatal("expected error for API failure")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status code in error, got %q", err.Error())
	}
}

func TestCreateGist_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	t.Setenv("GITHUB_TOKEN", "token")

	_, err := createGistWithURL(srv.URL, "content", "desc", true)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got %q", err.Error())
	}
}

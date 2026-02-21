// ABOUTME: Tests for session forking and PR linking
// ABOUTME: Covers fork, PR link CRUD, and error cases with temp directories

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestSession(t *testing.T, dir, sessionID string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, sessionID+".jsonl")
	records := []string{
		`{"v":1,"type":"session_start","ts":"2025-01-01T00:00:00Z","data":{"id":"` + sessionID + `","model":"test","cwd":"/tmp"}}`,
		`{"v":1,"type":"user","ts":"2025-01-01T00:01:00Z","data":{"content":"hello"}}`,
		`{"v":1,"type":"assistant","ts":"2025-01-01T00:02:00Z","data":{"content":"hi","model":"test","usage":{"input":10,"output":5},"stop_reason":"end_turn"}}`,
	}
	var data []byte
	for _, r := range records {
		data = append(data, []byte(r+"\n")...)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFork(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	sessionID := "original-session"
	writeTestSession(t, dir, sessionID)

	result, err := Fork(dir, sessionID)
	if err != nil {
		t.Fatalf("Fork() error: %v", err)
	}

	if result.OriginalID != sessionID {
		t.Errorf("OriginalID = %q; want %q", result.OriginalID, sessionID)
	}
	if result.NewID == "" {
		t.Error("NewID is empty")
	}
	if result.NewID == sessionID {
		t.Error("NewID should differ from OriginalID")
	}
	// NewID should be 32 hex chars (16 bytes)
	if len(result.NewID) != 32 {
		t.Errorf("NewID length = %d; want 32", len(result.NewID))
	}

	// Verify original still exists
	origPath := filepath.Join(dir, sessionID+".jsonl")
	if _, err := os.Stat(origPath); err != nil {
		t.Errorf("original session file missing: %v", err)
	}

	// Verify new session file exists with same content
	newPath := filepath.Join(dir, result.NewID+".jsonl")
	origData, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("reading original: %v", err)
	}
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("reading fork: %v", err)
	}
	if string(origData) != string(newData) {
		t.Error("forked session content differs from original")
	}
}

func TestFork_NonExistent(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := Fork(dir, "does-not-exist")
	if err == nil {
		t.Error("Fork() of non-existent session should return error")
	}
}

func TestLinkPR(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	err := LinkPR(dir, 42, "session-abc")
	if err != nil {
		t.Fatalf("LinkPR() error: %v", err)
	}

	// Verify the file was created
	data, err := os.ReadFile(filepath.Join(dir, "pr_links.json"))
	if err != nil {
		t.Fatalf("reading pr_links.json: %v", err)
	}

	var links map[string]string
	if err := json.Unmarshal(data, &links); err != nil {
		t.Fatalf("parsing pr_links.json: %v", err)
	}

	if links["42"] != "session-abc" {
		t.Errorf("PR 42 -> %q; want %q", links["42"], "session-abc")
	}
}

func TestGetPRSession(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := LinkPR(dir, 42, "session-abc"); err != nil {
		t.Fatalf("LinkPR() error: %v", err)
	}

	sessionID, err := GetPRSession(dir, 42)
	if err != nil {
		t.Fatalf("GetPRSession() error: %v", err)
	}
	if sessionID != "session-abc" {
		t.Errorf("GetPRSession(42) = %q; want %q", sessionID, "session-abc")
	}
}

func TestGetPRSession_NotFound(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Create an empty pr_links.json
	if err := os.WriteFile(filepath.Join(dir, "pr_links.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := GetPRSession(dir, 999)
	if err == nil {
		t.Error("GetPRSession() for non-linked PR should return error")
	}
}

func TestListPRLinks(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Link 3 PRs
	prs := map[int]string{
		10: "session-aaa",
		20: "session-bbb",
		30: "session-ccc",
	}
	for pr, sid := range prs {
		if err := LinkPR(dir, pr, sid); err != nil {
			t.Fatalf("LinkPR(%d) error: %v", pr, err)
		}
	}

	links, err := ListPRLinks(dir)
	if err != nil {
		t.Fatalf("ListPRLinks() error: %v", err)
	}

	if len(links) != 3 {
		t.Fatalf("ListPRLinks() returned %d links; want 3", len(links))
	}

	for pr, expectedSID := range prs {
		got, ok := links[pr]
		if !ok {
			t.Errorf("PR %d not found in links", pr)
			continue
		}
		if got != expectedSID {
			t.Errorf("PR %d -> %q; want %q", pr, got, expectedSID)
		}
	}
}

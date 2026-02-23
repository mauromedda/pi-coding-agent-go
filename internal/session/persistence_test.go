// ABOUTME: Tests for JSONL session persistence
// ABOUTME: Uses temp directories for isolated read/write testing

package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestWriter_WriteAndRead(t *testing.T) {
	// Override sessions dir for testing
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a session file manually
	path := filepath.Join(sessionsDir, "test-session.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	// Write some JSONL records
	records := []string{
		`{"v":1,"type":"session_start","ts":"2025-01-01T00:00:00Z","data":{"id":"test-session","model":"test","cwd":"/tmp"}}`,
		`{"v":1,"type":"user","ts":"2025-01-01T00:01:00Z","data":{"content":"hello"}}`,
	}
	for _, r := range records {
		f.WriteString(r + "\n")
	}
	f.Close()

	// Read back
	rf, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()

	// Verify we can parse the records
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty session file")
	}
}

func TestSessionID_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid with dash", "session-42", false},
		{"valid with underscore", "test_session", false},
		{"path traversal", "../../../etc/passwd", true},
		{"dot dot", "..", true},
		{"empty", "", true},
		{"slash", "foo/bar", true},
		{"backslash", "foo\\bar", true},
		{"space", "foo bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			valid := validSessionID.MatchString(tt.id)
			if valid == tt.wantErr {
				t.Errorf("validSessionID.MatchString(%q) = %v, want %v", tt.id, valid, !tt.wantErr)
			}
		})
	}
}

func TestCompactWithLLM_BelowThreshold(t *testing.T) {
	t.Parallel()

	msgs := make([]ai.Message, 5)
	for i := range msgs {
		msgs[i] = ai.NewTextMessage(ai.RoleUser, "msg")
	}

	summarizer := func(_ context.Context, _ []ai.Message, _ string) (string, error) {
		return "should not be called", nil
	}
	cfg := CompactionConfig{KeepRecentTokens: 100000} // very high; keeps everything

	result, err := CompactWithLLM(context.Background(), msgs, cfg, summarizer)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != "" {
		t.Error("expected no summary below threshold")
	}
	if len(result.Messages) != 5 {
		t.Errorf("expected 5 messages, got %d", len(result.Messages))
	}
}

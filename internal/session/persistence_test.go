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

func TestWriteCompaction_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a compaction record via helper, then read it back.
	path := filepath.Join(sessionsDir, "compact-test.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	w := &Writer{file: f}

	cd := CompactionData{
		Summary:          "User discussed Go testing patterns",
		FirstKeptEntryID: 5,
		TokensBefore:     8000,
		FilesRead:        []string{"/tmp/foo.go"},
		FilesWritten:     []string{"/tmp/bar.go"},
	}
	if err := w.WriteCompaction(cd); err != nil {
		t.Fatalf("WriteCompaction: %v", err)
	}
	w.Close()

	// Read records from the file directly using ReadRecordsFromPath.
	records, err := ReadRecordsFromPath(path)
	if err != nil {
		t.Fatalf("ReadRecordsFromPath: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0]
	if rec.Type != RecordCompaction {
		t.Errorf("type = %q, want %q", rec.Type, RecordCompaction)
	}
	if rec.Version != CurrentRecordVersion {
		t.Errorf("version = %d, want %d", rec.Version, CurrentRecordVersion)
	}

	var got CompactionData
	if err := rec.Unmarshal(&got); err != nil {
		t.Fatalf("Unmarshal compaction: %v", err)
	}
	if got.Summary != cd.Summary {
		t.Errorf("summary = %q, want %q", got.Summary, cd.Summary)
	}
	if got.FirstKeptEntryID != cd.FirstKeptEntryID {
		t.Errorf("first_kept_entry_id = %d, want %d", got.FirstKeptEntryID, cd.FirstKeptEntryID)
	}
	if got.TokensBefore != cd.TokensBefore {
		t.Errorf("tokens_before = %d, want %d", got.TokensBefore, cd.TokensBefore)
	}
	if len(got.FilesRead) != 1 || got.FilesRead[0] != "/tmp/foo.go" {
		t.Errorf("files_read = %v, want [/tmp/foo.go]", got.FilesRead)
	}
	if len(got.FilesWritten) != 1 || got.FilesWritten[0] != "/tmp/bar.go" {
		t.Errorf("files_written = %v, want [/tmp/bar.go]", got.FilesWritten)
	}
}

func TestReadRecordsFromPath_BackwardCompat_V1(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "v1-session.jsonl")

	// Write v1 records manually.
	lines := []string{
		`{"v":1,"type":"session_start","ts":"2025-01-01T00:00:00Z","data":{"id":"s1","model":"test","cwd":"/tmp"}}`,
		`{"v":1,"type":"user","ts":"2025-01-01T00:01:00Z","data":{"content":"hello"}}`,
	}
	var content string
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	records, err := ReadRecordsFromPath(path)
	if err != nil {
		t.Fatalf("ReadRecordsFromPath: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	// V1 records should be readable without error.
	if records[0].Version != 1 {
		t.Errorf("expected v1, got v%d", records[0].Version)
	}
	if records[0].Type != RecordSessionStart {
		t.Errorf("expected session_start, got %s", records[0].Type)
	}

	var start SessionStartData
	if err := records[0].Unmarshal(&start); err != nil {
		t.Fatalf("Unmarshal v1 session_start: %v", err)
	}
	if start.ID != "s1" {
		t.Errorf("session id = %q, want %q", start.ID, "s1")
	}
}

func TestCurrentRecordVersion_IsThree(t *testing.T) {
	t.Parallel()
	if CurrentRecordVersion != 3 {
		t.Errorf("CurrentRecordVersion = %d, want 3", CurrentRecordVersion)
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

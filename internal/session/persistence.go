// ABOUTME: JSONL session persistence with append-only writes
// ABOUTME: Reads line-by-line with bufio.Scanner; crash-safe via O_APPEND

package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

// RecordType identifies the type of JSONL record.
type RecordType string

const (
	RecordSessionStart RecordType = "session_start"
	RecordUser         RecordType = "user"
	RecordAssistant    RecordType = "assistant"
	RecordToolCall     RecordType = "tool_call"
	RecordToolResult   RecordType = "tool_result"
	RecordCheckpoint   RecordType = "checkpoint"
	RecordCompaction   RecordType = "compaction"
	RecordBranch       RecordType = "branch"
	RecordSessionEnd   RecordType = "session_end"
)

// Record is the envelope for all JSONL entries.
type Record struct {
	Version int             `json:"v"`
	Type    RecordType      `json:"type"`
	TS      string          `json:"ts"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// SessionStartData holds session_start metadata.
type SessionStartData struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	CWD   string `json:"cwd"`
}

// UserData holds user message data.
type UserData struct {
	Content  string        `json:"content"`
	Mentions []MentionData `json:"mentions,omitempty"`
}

// MentionData holds @file mention info.
type MentionData struct {
	Path  string `json:"path"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

// AssistantData holds assistant response data.
type AssistantData struct {
	Content    string   `json:"content"`
	Model      string   `json:"model"`
	Usage      UsageData `json:"usage"`
	StopReason string   `json:"stop_reason"`
}

// UsageData holds token usage.
type UsageData struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// Writer appends records to a session JSONL file.
type Writer struct {
	file *os.File
}

// NewWriter creates a Writer for the given session ID.
func NewWriter(sessionID string) (*Writer, error) {
	dir := config.SessionsDir()
	if err := config.EnsureDir(dir); err != nil {
		return nil, fmt.Errorf("creating sessions dir: %w", err)
	}

	path := filepath.Join(dir, sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}

	return &Writer{file: f}, nil
}

// WriteRecord appends a record to the session file.
func (w *Writer) WriteRecord(recType RecordType, data any) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling record data: %w", err)
	}

	rec := Record{
		Version: 1,
		Type:    recType,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Data:    dataBytes,
	}

	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}

	line = append(line, '\n')
	if _, err := w.file.Write(line); err != nil {
		return fmt.Errorf("writing record: %w", err)
	}
	return nil
}

// Close closes the session file.
func (w *Writer) Close() error {
	return w.file.Close()
}

// ReadRecords reads all records from a session file.
func ReadRecords(sessionID string) ([]Record, error) {
	path := filepath.Join(config.SessionsDir(), sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session %s: %w", sessionID, err)
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue // Skip malformed lines
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("scanning session %s: %w", sessionID, err)
	}
	return records, nil
}

// ListSessions scans the sessions directory and returns session start records.
func ListSessions() ([]SessionStartData, error) {
	dir := config.SessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions dir: %w", err)
	}

	var sessions []SessionStartData
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := readFirstLine(path)
		if err != nil {
			continue
		}
		sessions = append(sessions, data)
	}
	return sessions, nil
}

func readFirstLine(path string) (SessionStartData, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionStartData{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return SessionStartData{}, fmt.Errorf("empty session file")
	}

	var rec Record
	if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
		return SessionStartData{}, fmt.Errorf("parsing first record: %w", err)
	}

	var start SessionStartData
	if err := json.Unmarshal(rec.Data, &start); err != nil {
		return SessionStartData{}, fmt.Errorf("parsing session start: %w", err)
	}
	return start, nil
}

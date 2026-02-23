// ABOUTME: JSONL session persistence with append-only writes
// ABOUTME: Reads line-by-line with bufio.Scanner; crash-safe via O_APPEND

package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

const (
	scannerInitialBuf = 64 * 1024     // 64KB initial buffer (was 1MB)
	scannerMaxBuf     = 10 * 1024 * 1024 // 10MB max line
)

// scannerBufPool reuses scanner buffers across ReadRecords calls.
var scannerBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, scannerInitialBuf)
	},
}

// validSessionID validates that a session ID contains only safe characters
// to prevent path traversal attacks.
var validSessionID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

// Unmarshal unmarshals the record data into v.
func (r *Record) Unmarshal(v any) error {
	if r.Data == nil {
		return nil
	}
	return json.Unmarshal(r.Data, v)
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
	Content    string    `json:"content"`
	Model      string    `json:"model"`
	Usage      UsageData `json:"usage"`
	StopReason string    `json:"stop_reason"`
}

// UsageData holds token usage.
type UsageData struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// CompactionData holds compaction record metadata.
type CompactionData struct {
	Summary          string   `json:"summary"`
	FirstKeptEntryID int      `json:"first_kept_entry_id"`
	TokensBefore     int      `json:"tokens_before"`
	FilesRead        []string `json:"files_read,omitempty"`
	FilesWritten     []string `json:"files_written,omitempty"`
}

// CurrentRecordVersion is the version stamped on new records.
// V1: original format. V3: adds compaction and branch records.
// Reading is backward-compatible with all prior versions.
const CurrentRecordVersion = 3

// Writer appends records to a session JSONL file.
type Writer struct {
	file *os.File
}

// NewWriter creates a Writer for the given session ID.
func NewWriter(sessionID string) (*Writer, error) {
	if !validSessionID.MatchString(sessionID) {
		return nil, fmt.Errorf("invalid session ID %q: must match [a-zA-Z0-9_-]+", sessionID)
	}
	dir := config.SessionsDir()
	if err := config.EnsureDir(dir); err != nil {
		return nil, fmt.Errorf("creating sessions dir: %w", err)
	}

	path := filepath.Join(dir, sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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
		Version: CurrentRecordVersion,
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

// WriteCompaction writes a compaction record to the session file.
func (w *Writer) WriteCompaction(data CompactionData) error {
	return w.WriteRecord(RecordCompaction, data)
}

// Close closes the session file.
func (w *Writer) Close() error {
	return w.file.Close()
}

// ReadRecords reads all records from a session file.
func ReadRecords(sessionID string) ([]Record, error) {
	if !validSessionID.MatchString(sessionID) {
		return nil, fmt.Errorf("invalid session ID %q: must match [a-zA-Z0-9_-]+", sessionID)
	}

	path := filepath.Join(config.SessionsDir(), sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session %s: %w", sessionID, err)
	}
	defer f.Close()

	var records []Record
	buf := scannerBufPool.Get().([]byte)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf[:0], scannerMaxBuf)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			log.Printf("warning: session %s line %d: malformed JSONL: %v", sessionID, lineNum, err)
			continue
		}
		records = append(records, rec)
	}

	scannerBufPool.Put(buf)

	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("scanning session %s: %w", sessionID, err)
	}
	return records, nil
}

// ReadRecordsFromPath reads all records from a JSONL file at the given path.
// It accepts records of any version for backward compatibility.
func ReadRecordsFromPath(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer f.Close()

	var records []Record
	buf := scannerBufPool.Get().([]byte)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf[:0], scannerMaxBuf)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			log.Printf("warning: %s line %d: malformed JSONL: %v", path, lineNum, err)
			continue
		}
		records = append(records, rec)
	}

	scannerBufPool.Put(buf)

	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("scanning %s: %w", path, err)
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

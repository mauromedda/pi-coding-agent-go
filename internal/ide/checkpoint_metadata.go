// ABOUTME: JSON persistence for checkpoint metadata enabling cross-session history
// ABOUTME: Stores checkpoint records in .pi-go/checkpoints/ directory

package ide

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MetadataStore persists checkpoint metadata to disk as JSON files.
type MetadataStore struct {
	dir string
}

// CheckpointRecord is a serializable checkpoint entry.
type CheckpointRecord struct {
	ID          string    `json:"id"`
	Ref         string    `json:"ref"`
	Timestamp   time.Time `json:"timestamp"`
	ToolName    string    `json:"toolName"`
	ToolArgs    string    `json:"toolArgs"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Mode        string    `json:"mode,omitempty"`
	SessionID   string    `json:"sessionId"`
}

// NewMetadataStore creates a store that persists to the given directory.
// Creates the directory if it doesn't exist.
func NewMetadataStore(dir string) (*MetadataStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating metadata dir %s: %w", dir, err)
	}
	return &MetadataStore{dir: dir}, nil
}

// Save persists a checkpoint record to disk.
func (m *MetadataStore) Save(record CheckpointRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling record %s: %w", record.ID, err)
	}

	filename := fmt.Sprintf("%s_%s.json",
		record.Timestamp.UTC().Format("20060102T150405Z"),
		record.ID,
	)
	path := filepath.Join(m.dir, filename)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing record %s: %w", record.ID, err)
	}
	return nil
}

// List returns all checkpoint records, newest first.
func (m *MetadataStore) List() ([]CheckpointRecord, error) {
	return m.listFiltered(func(_ CheckpointRecord) bool { return true })
}

// ListBySession returns checkpoints for a specific session.
func (m *MetadataStore) ListBySession(sessionID string) ([]CheckpointRecord, error) {
	return m.listFiltered(func(r CheckpointRecord) bool {
		return r.SessionID == sessionID
	})
}

// Delete removes a checkpoint record by ID.
func (m *MetadataStore) Delete(id string) error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return fmt.Errorf("reading metadata dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		// Filename format: {timestamp}_{id}.json
		name := strings.TrimSuffix(e.Name(), ".json")
		parts := strings.SplitN(name, "_", 2)
		if len(parts) == 2 && parts[1] == id {
			return os.Remove(filepath.Join(m.dir, e.Name()))
		}
	}

	return fmt.Errorf("checkpoint record %q not found", id)
}

// GenerateID returns a random UUID v4 string.
func GenerateID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating UUID: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func (m *MetadataStore) listFiltered(match func(CheckpointRecord) bool) ([]CheckpointRecord, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("reading metadata dir: %w", err)
	}

	var records []CheckpointRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(m.dir, e.Name()))
		if err != nil {
			continue // skip unreadable files
		}

		var r CheckpointRecord
		if err := json.Unmarshal(data, &r); err != nil {
			continue // skip corrupt files
		}

		if match(r) {
			records = append(records, r)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	return records, nil
}

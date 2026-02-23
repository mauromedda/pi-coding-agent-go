// ABOUTME: Table-driven tests for the SSE reader parsing logic
// ABOUTME: Covers single events, multi-line data, multiple events, empty streams, comments

package sse

import (
	"io"
	"strings"
	"testing"
)

func TestReaderNext_LargeLine(t *testing.T) {
	t.Parallel()

	// Verify that the scanner handles lines up to 1MB.
	bigData := strings.Repeat("x", 512*1024)
	input := "data: " + bigData + "\n\n"
	reader := NewReader(strings.NewReader(input))
	ev, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Data != bigData {
		t.Errorf("data length = %d, want %d", len(ev.Data), len(bigData))
	}
}

func BenchmarkReaderNext_MultiLineData(b *testing.B) {
	// Build a multi-line event: 50 data lines.
	var sb strings.Builder
	for range 50 {
		sb.WriteString("data: some payload data for benchmarking\n")
	}
	sb.WriteString("\n")
	payload := sb.String()

	for b.Loop() {
		reader := NewReader(strings.NewReader(payload))
		_, _ = reader.Next()
	}
}

func TestSSEReader_PoolRoundTrip(t *testing.T) {
	t.Parallel()

	// Get a reader, use it, close it (returns buffer to pool), get another.
	// This should not panic or produce incorrect results.
	for range 10 {
		r := NewReader(strings.NewReader("data: hello\n\n"))
		ev, err := r.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Data != "hello" {
			t.Errorf("data = %q; want %q", ev.Data, "hello")
		}
		r.Close()
	}
}

func TestSSEReader_DoubleClose(t *testing.T) {
	t.Parallel()

	// Double close should not panic.
	r := NewReader(strings.NewReader("data: test\n\n"))
	r.Close()
	r.Close() // should be a no-op
}

func TestReaderNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantEvents []*Event
		wantErr    error
	}{
		{
			name:  "single event with all fields",
			input: "event: message\ndata: hello world\nid: 1\n\n",
			wantEvents: []*Event{
				{Type: "message", Data: "hello world", ID: "1"},
			},
		},
		{
			name:  "event with only data field",
			input: "data: just data\n\n",
			wantEvents: []*Event{
				{Type: "", Data: "just data", ID: ""},
			},
		},
		{
			name:  "multi-line data",
			input: "data: line one\ndata: line two\ndata: line three\n\n",
			wantEvents: []*Event{
				{Type: "", Data: "line one\nline two\nline three", ID: ""},
			},
		},
		{
			name:  "multiple events",
			input: "event: first\ndata: one\n\nevent: second\ndata: two\n\n",
			wantEvents: []*Event{
				{Type: "first", Data: "one", ID: ""},
				{Type: "second", Data: "two", ID: ""},
			},
		},
		{
			name:       "empty stream",
			input:      "",
			wantEvents: nil,
		},
		{
			name:  "comments are skipped",
			input: ": this is a comment\ndata: visible\n\n",
			wantEvents: []*Event{
				{Type: "", Data: "visible", ID: ""},
			},
		},
		{
			name:       "only comments and blank lines",
			input:      ": comment\n\n: another\n\n",
			wantEvents: nil,
		},
		{
			name:  "fields without space after colon",
			input: "event:nospace\ndata:value\nid:42\n\n",
			wantEvents: []*Event{
				{Type: "nospace", Data: "value", ID: "42"},
			},
		},
		{
			name:  "event type with data containing colon",
			input: "data: key: value\n\n",
			wantEvents: []*Event{
				{Type: "", Data: "key: value", ID: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := NewReader(strings.NewReader(tt.input))
			var got []*Event

			for {
				ev, err := reader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				got = append(got, ev)
			}

			if len(got) != len(tt.wantEvents) {
				t.Fatalf("got %d events, want %d", len(got), len(tt.wantEvents))
			}

			for i, want := range tt.wantEvents {
				if got[i].Type != want.Type {
					t.Errorf("event[%d].Type = %q, want %q", i, got[i].Type, want.Type)
				}
				if got[i].Data != want.Data {
					t.Errorf("event[%d].Data = %q, want %q", i, got[i].Data, want.Data)
				}
				if got[i].ID != want.ID {
					t.Errorf("event[%d].ID = %q, want %q", i, got[i].ID, want.ID)
				}
			}
		})
	}
}

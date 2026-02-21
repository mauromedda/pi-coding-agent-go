// ABOUTME: Table-driven tests for the SSE reader parsing logic
// ABOUTME: Covers single events, multi-line data, multiple events, empty streams, comments

package sse

import (
	"io"
	"strings"
	"testing"
)

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
		tt := tt
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

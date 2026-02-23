// ABOUTME: Tests for TTFB network probe: latency measurement and classification
// ABOUTME: Uses httptest mock servers with configurable delays

package perf

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeTTFB_LocalLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {}\n\n")
	}))
	defer srv.Close()

	result := ProbeTTFB(context.Background(), srv.URL, "test-key")

	if result.Latency != LatencyLocal {
		t.Errorf("expected LatencyLocal, got %v (TTFB=%v)", result.Latency, result.TTFB)
	}
	if result.TTFB >= 50*time.Millisecond {
		t.Errorf("expected TTFB < 50ms for local server, got %v", result.TTFB)
	}
}

func TestProbeTTFB_FastLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {}\n\n")
	}))
	defer srv.Close()

	result := ProbeTTFB(context.Background(), srv.URL, "test-key")

	if result.Latency != LatencyFast {
		t.Errorf("expected LatencyFast, got %v (TTFB=%v)", result.Latency, result.TTFB)
	}
}

func TestProbeTTFB_SlowLatency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(600 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {}\n\n")
	}))
	defer srv.Close()

	result := ProbeTTFB(context.Background(), srv.URL, "test-key")

	if result.Latency != LatencySlow {
		t.Errorf("expected LatencySlow, got %v (TTFB=%v)", result.Latency, result.TTFB)
	}
}

func TestProbeTTFB_ErrorDefaultsToSlow(t *testing.T) {
	result := ProbeTTFB(context.Background(), "http://127.0.0.1:1", "test-key")

	if result.Latency != LatencySlow {
		t.Errorf("expected LatencySlow on error, got %v", result.Latency)
	}
}

func TestProbeTTFB_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Even a 500 response has a TTFB; we measure first byte, not success.
	result := ProbeTTFB(context.Background(), srv.URL, "test-key")

	// Should still classify based on actual TTFB (which is fast for a local server).
	if result.TTFB == 0 {
		t.Error("expected non-zero TTFB even on server error")
	}
}

func TestProbeTTFB_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := ProbeTTFB(ctx, "http://127.0.0.1:1", "test-key")

	if result.Latency != LatencySlow {
		t.Errorf("expected LatencySlow on cancelled context, got %v", result.Latency)
	}
}

func TestProbeTTFB_BaseURLWithV1Suffix(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {}\n\n")
	}))
	defer srv.Close()

	// baseURL already includes /v1; should NOT produce /v1/v1/chat/completions
	ProbeTTFB(context.Background(), srv.URL+"/v1", "test-key")

	if gotPath != "/v1/chat/completions" {
		t.Errorf("expected path /v1/chat/completions, got %q", gotPath)
	}
}

func TestProbeTTFB_SendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {}\n\n")
	}))
	defer srv.Close()

	ProbeTTFB(context.Background(), srv.URL, "sk-test-123")

	if gotAuth != "Bearer sk-test-123" {
		t.Errorf("expected Authorization 'Bearer sk-test-123', got %q", gotAuth)
	}
}

func TestLatencyClass_String(t *testing.T) {
	tests := []struct {
		class LatencyClass
		want  string
	}{
		{LatencyLocal, "local"},
		{LatencyFast, "fast"},
		{LatencySlow, "slow"},
		{LatencyClass(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.class.String(); got != tt.want {
				t.Errorf("LatencyClass(%d).String() = %q, want %q", tt.class, got, tt.want)
			}
		})
	}
}

func TestClassifyLatency(t *testing.T) {
	tests := []struct {
		name string
		ttfb time.Duration
		want LatencyClass
	}{
		{"zero", 0, LatencyLocal},
		{"10ms", 10 * time.Millisecond, LatencyLocal},
		{"49ms", 49 * time.Millisecond, LatencyLocal},
		{"50ms", 50 * time.Millisecond, LatencyFast},
		{"200ms", 200 * time.Millisecond, LatencyFast},
		{"499ms", 499 * time.Millisecond, LatencyFast},
		{"500ms", 500 * time.Millisecond, LatencySlow},
		{"2s", 2 * time.Second, LatencySlow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyLatency(tt.ttfb); got != tt.want {
				t.Errorf("classifyLatency(%v) = %v, want %v", tt.ttfb, got, tt.want)
			}
		})
	}
}

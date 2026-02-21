// ABOUTME: Tests for the shared HTTP client: basic requests, retry on 429, SSE streaming
// ABOUTME: Uses httptest.NewServer for deterministic, isolated test scenarios

package httputil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientDoBasicRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("got method %s, want POST", r.Method)
		}
		if r.Header.Get("X-Custom") != "test-value" {
			t.Errorf("got header %q, want %q", r.Header.Get("X-Custom"), "test-value")
		}
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, map[string]string{"X-Custom": "test-value"})

	resp, err := client.Do(context.Background(), http.MethodPost, "/test", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	got, _ := io.ReadAll(resp.Body)
	if string(got) != "hello" {
		t.Errorf("got body %q, want %q", string(got), "hello")
	}
}

func TestClientDoRetryOn429(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, nil)

	resp, err := client.Do(context.Background(), http.MethodGet, "/retry", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("got %d attempts, want 3", got)
	}
}

func TestClientDoRetryOn5xx(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, nil)

	resp, err := client.Do(context.Background(), http.MethodGet, "/retry-5xx", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestClientDoExhaustsRetries(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, nil)

	resp, err := client.Do(context.Background(), http.MethodGet, "/always-429", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
}

func TestClientStreamSSE(t *testing.T) {
	t.Parallel()

	ssePayload := "event: message\ndata: hello\n\nevent: done\ndata: world\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ssePayload))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, nil)

	reader, resp, err := client.StreamSSE(context.Background(), http.MethodGet, "/events", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	ev1, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error on first event: %v", err)
	}
	if ev1.Type != "message" || ev1.Data != "hello" {
		t.Errorf("event1 = %+v, want type=message data=hello", ev1)
	}

	ev2, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error on second event: %v", err)
	}
	if ev2.Type != "done" || ev2.Data != "world" {
		t.Errorf("event2 = %+v, want type=done data=world", ev2)
	}

	_, err = reader.Next()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestClientDoRetryWithBody(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	wantBody := `{"prompt":"hello"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)

		body, _ := io.ReadAll(r.Body)
		if string(body) != wantBody {
			t.Errorf("attempt %d: got body %q, want %q", n, string(body), wantBody)
		}

		if n <= 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, nil)

	// bytes.NewReader implements io.Seeker, enabling body rewind on retry.
	resp, err := client.Do(context.Background(), http.MethodPost, "/retry-body", bytes.NewReader([]byte(wantBody)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := attempts.Load(); got != 2 {
		t.Errorf("got %d attempts, want 2", got)
	}
}

func TestNewClientHasTimeout(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com", nil)

	if client.httpClient.Timeout == 0 {
		t.Error("httpClient.Timeout is zero; want a non-zero timeout")
	}
}

func TestNewClientHasTransportTimeouts(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example.com", nil)

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("httpClient.Transport is not *http.Transport")
	}

	if transport.TLSHandshakeTimeout == 0 {
		t.Error("TLSHandshakeTimeout is zero; want a non-zero timeout")
	}
	if transport.ResponseHeaderTimeout == 0 {
		t.Error("ResponseHeaderTimeout is zero; want a non-zero timeout")
	}
}

func TestClientDoRespectsContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := NewClient(srv.URL, nil)
	_, err := client.Do(ctx, http.MethodGet, "/cancelled", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

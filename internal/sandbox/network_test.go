// ABOUTME: Tests for NetworkFilter HTTP CONNECT proxy domain filtering
// ABOUTME: Covers allowed/blocked domain routing, ProxyEnv, and graceful shutdown

package sandbox

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNetworkFilter_AllowedDomain(t *testing.T) {
	t.Parallel()

	// Start a local HTTP server to act as the "allowed" upstream.
	upstreamAddr := startTestUpstream(t, "hello from allowed")

	_, port, _ := net.SplitHostPort(upstreamAddr)

	nf := NewNetworkFilter([]string{"127.0.0.1"})
	proxyAddr, err := nf.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = nf.Stop(ctx)
	})

	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/ok", port))
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "hello from allowed") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestNetworkFilter_BlockedDomain(t *testing.T) {
	t.Parallel()

	nf := NewNetworkFilter([]string{"example.com"})
	proxyAddr, err := nf.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = nf.Stop(ctx)
	})

	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	// "blocked.evil.com" is not in the allowlist; proxy must reject it.
	resp, err := client.Get("http://blocked.evil.com/secret")
	if err != nil {
		// A connection-level rejection is acceptable.
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for blocked domain, got %d", resp.StatusCode)
	}
}

func TestNetworkFilter_BlockedDomain_CONNECT(t *testing.T) {
	t.Parallel()

	nf := NewNetworkFilter([]string{"example.com"})
	proxyAddr, err := nf.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = nf.Stop(ctx)
	})

	// Issue a raw CONNECT request for a blocked domain.
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	_, _ = fmt.Fprintf(conn, "CONNECT blocked.evil.com:443 HTTP/1.1\r\nHost: blocked.evil.com:443\r\n\r\n")

	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	response := string(buf[:n])

	if !strings.Contains(response, "403") {
		t.Errorf("expected 403 in CONNECT response for blocked domain, got: %s", response)
	}
}

func TestNetworkFilter_StopCleanup(t *testing.T) {
	t.Parallel()

	nf := NewNetworkFilter([]string{"example.com"})
	proxyAddr, err := nf.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Verify the proxy is listening.
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("proxy should be reachable before Stop: %v", err)
	}
	conn.Close()

	// Stop the proxy.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := nf.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Verify the port is no longer accepting connections.
	_, err = net.DialTimeout("tcp", proxyAddr, 500*time.Millisecond)
	if err == nil {
		t.Error("proxy should not accept connections after Stop")
	}
}

func TestNetworkFilter_ProxyEnv(t *testing.T) {
	t.Parallel()

	nf := NewNetworkFilter([]string{"example.com"})
	proxyAddr, err := nf.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = nf.Stop(ctx)
	})

	envVars := nf.ProxyEnv()
	if len(envVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(envVars))
	}

	expected := "http://" + proxyAddr
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			t.Errorf("malformed env var: %q", env)
			continue
		}
		key, val := parts[0], parts[1]
		if key != "HTTP_PROXY" && key != "HTTPS_PROXY" {
			t.Errorf("unexpected env key: %q", key)
		}
		if val != expected {
			t.Errorf("expected value %q, got %q", expected, val)
		}
	}
}

func TestNetworkFilter_ConnectSemaphore(t *testing.T) {
	t.Parallel()

	// Verify that NewNetworkFilter initializes the connSem field.
	nf := NewNetworkFilter([]string{"example.com"})
	if nf.connSem == nil {
		t.Fatal("connSem is nil; expected initialized semaphore channel")
	}
	if cap(nf.connSem) != maxConcurrentConns {
		t.Errorf("connSem capacity = %d; want %d", cap(nf.connSem), maxConcurrentConns)
	}
}

// startTestUpstream creates a local HTTP server returning the given body.
// Returns the listener address (host:port).
func startTestUpstream(t *testing.T, body string) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	return ln.Addr().String()
}
